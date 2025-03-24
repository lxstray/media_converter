package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
)

func Tiktok2mp4(w http.ResponseWriter, r *http.Request, tiktok_url string) {
	videoInfo, err := GetTiktokInfo(tiktok_url)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println("tiktok info:", err)
		return
	}

	videoCmd := exec.Command("yt-dlp", "-f", "best", tiktok_url, "-o", "-")

	videoPipe, err := videoCmd.StdoutPipe()
	if err != nil {
		http.Error(w, "Failed to create yt-dlp pipe", http.StatusInternalServerError)
		log.Println("yt-dlp pipe error:", err)
		return
	}

	if err := videoCmd.Start(); err != nil {
		http.Error(w, "Failed to start yt-dlp", http.StatusInternalServerError)
		log.Println("yt-dlp start error:", err)
		return
	}

	fileName := "tiktok_" + videoInfo.VideoID + ".mp4"
	encodedFileName := url.PathEscape(fileName)

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, encodedFileName))

	if _, err := io.Copy(w, videoPipe); err != nil {
		log.Println("Streaming error:", err)
	}

	if err := videoCmd.Wait(); err != nil {
		log.Println("yt-dlp execution error:", err)
	}
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
