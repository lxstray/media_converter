package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sync"

	"github.com/disintegration/imaging"
)

//TODO: best practice: добавить возращение error в функцию handler

func Sc2m4a(w http.ResponseWriter, r *http.Request, audioUrl string) {
	info, err := GetSCInfo(audioUrl) //TODO: запустить в горутину если получиться получить video id другим способом
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tempAudioPath, tempCoverPath := GenerateTempFilesNames()
	defer os.Remove(tempAudioPath)
	defer os.Remove(tempCoverPath)

	coverErr := getSCCover(info.ThumbnailURL, tempCoverPath)
	if coverErr != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	audioChan := make(chan io.ReadCloser)
	errChan := make(chan error, 2)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		audioCmd := exec.Command("yt-dlp", "--audio-quality", "0", "-f", "mp3", "--no-playlist", audioUrl, "-o", "-") //TODO: чекунть обязатльено ли тут юзать mp3 для качества
		audioPipe, err := audioCmd.StdoutPipe()
		if err != nil {
			log.Println(err)
			errChan <- err
		}
		audioChan <- audioPipe
		if err := audioCmd.Run(); err != nil {
			log.Println(err)
			errChan <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ffmpegCmd := exec.Command("ffmpeg", "-i", "pipe:0", "-i", tempCoverPath, "-map", "0", "-map", "1", "-c", "copy", "-c:a", "aac", "-metadata", "artist="+info.Uploader, "-metadata", "title="+info.Title, "-disposition:v:0", "attached_pic", tempAudioPath)
		ffmpegCmd.Stdin = <-audioChan
		if err := ffmpegCmd.Run(); err != nil {
			log.Println(err)
			errChan <- err
		}
	}()

	wg.Wait()

	select {
	case <-errChan:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	default:
	}

	fileName := info.Uploader + "_-_" + info.Title + ".m4a"

	encodedFileName := url.PathEscape(fileName)

	w.Header().Set("Content-Type", "audio/m4a")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, encodedFileName))
	http.ServeFile(w, r, tempAudioPath)
}

type SCInfo struct {
	Title        string `json:"title"`
	Uploader     string `json:"uploader"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type SoundCloudData struct {
	Title      string `json:"title"`
	Uploader   string `json:"uploader"`
	Thumbnails []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"thumbnails"`
}

func GetSCInfo(url string) (SCInfo, error) {
	var infoWithThumb SCInfo
	infoJSONCmd := exec.Command("yt-dlp", "--no-playlist", "-j", url)

	infoJSON, err := infoJSONCmd.Output()
	if err != nil {
		log.Println("error infoJSONCmd:", err)
		return infoWithThumb, err
	}

	var info SoundCloudData
	err = json.Unmarshal(infoJSON, &info)
	if err != nil {
		log.Println("error parsing JSON:", err)
		return infoWithThumb, err

	}

	var thumbnailURL string
	for _, thumbnail := range info.Thumbnails {
		if thumbnail.ID == "original" {
			thumbnailURL = thumbnail.URL
			break
		} else if thumbnail.ID == "0" {
			url := thumbnail.URL
			re := regexp.MustCompile(`-(\w+)(\.png)`)
			modifiedURL := re.ReplaceAllString(url, "-original$2")
			thumbnailURL = modifiedURL
			break
		}
	}

	infoWithThumb = SCInfo{
		Title:        info.Title,
		Uploader:     info.Uploader,
		ThumbnailURL: thumbnailURL,
	}

	return infoWithThumb, nil
}

func getSCCover(thumbURL string, tempCoverPath string) error {
	resp, err := http.Get(thumbURL)
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()

	img, err := imaging.Decode(resp.Body)
	if err != nil {
		log.Println(err)
		return err
	}

	err = imaging.Save(img, tempCoverPath)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
