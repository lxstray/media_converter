package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"sync"
)

func Tiktok2mp4(w http.ResponseWriter, r *http.Request, tiktok_url string) {
	infoChan := make(chan TikTokInfo)
	infoDone := make(chan bool)
	errChan := make(chan error, 2)
	videoPipeChan := make(chan io.ReadCloser)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(infoChan)
		
		videoInfo, err := GetTiktokInfo(tiktok_url)
		if err != nil {
			errChan <- err
			return
		}
		
		infoChan <- videoInfo
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		
		videoCmd := exec.Command("yt-dlp", "-f", "best", tiktok_url, "-o", "-")
		
		videoPipe, err := videoCmd.StdoutPipe()
		if err != nil {
			errChan <- err
			return
		}
		
		if err := videoCmd.Start(); err != nil {
			errChan <- err
			return
		}
		
		videoPipeChan <- videoPipe
		
		if err := videoCmd.Wait(); err != nil {
			log.Println("yt-dlp execution error:", err)
		}
	}()

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Transfer-Encoding", "chunked")
	
	go func() {
		select {
		case videoInfo, ok := <-infoChan:
			if ok {
				fileName := "tiktok_" + videoInfo.VideoID + ".mp4"
				encodedFileName := url.PathEscape(fileName)
				w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, encodedFileName))
			} else {
				w.Header().Set("Content-Disposition", `attachment; filename="tiktok_video.mp4"`)
			}
		case <-r.Context().Done():
			return
		}
		close(infoDone)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("TikTok processing error:", err)
			return
		}
	case videoPipe := <-videoPipeChan:
		if _, err := io.Copy(w, videoPipe); err != nil {
			log.Println("Streaming error:", err)
		}
	case <-r.Context().Done():
		log.Println("Request canceled by client")
		return
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(videoPipeChan)
	}()
}

type TikTokInfo struct {
	VideoID string `json:"id"`
}

func GetTiktokInfo(url string) (TikTokInfo, error) {
	infoJSONCmd := exec.Command("yt-dlp", "--no-playlist", "-j", url)

	var info TikTokInfo

	infoJSON, err := infoJSONCmd.Output()
	if err != nil {
		log.Println("error infoJSONCmd:", err)
		return info, err
	}

	err = json.Unmarshal(infoJSON, &info)
	if err != nil {
		log.Println("error parsing JSON:", err)
		return info, err
	}

	return info, nil
}
