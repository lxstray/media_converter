package converter

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/disintegration/imaging"
)

func Sc2m4a(w *http.ResponseWriter, r *http.Request, url string) {
	info := GetSCInfo(url) //TODO: запустить в горутину если получиться получить video id другим способом
	tempAudioPath, tempCoverPath := GenerateTempFilesNames()

	getSCCover(info.ThumbnailURL, tempCoverPath)

	audioChan := make(chan io.ReadCloser, 10000000)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		audioCmd := exec.Command("yt-dlp", "--audio-quality", "0", "-f", "mp3", "--no-playlist", url, "-o", "-") //TODO: чекунть обязатльено ли тут юзать mp3 для качества
		audioPipe, err := audioCmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		audioChan <- audioPipe
		if err := audioCmd.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ffmpegCmd := exec.Command("ffmpeg", "-i", "pipe:0", "-i", tempCoverPath, "-map", "0", "-map", "1", "-c", "copy", "-c:a", "aac", "-metadata", "artist="+info.Uploader, "-metadata", "title="+info.Title, "-disposition:v:0", "attached_pic", tempAudioPath)
		ffmpegCmd.Stdin = <-audioChan
		if err := ffmpegCmd.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()

	fileName := info.Uploader + "_-_" + info.Title + ".m4a"
	resp := *w

	resp.Header().Set("Content-Type", "audio/m4a")
	resp.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	http.ServeFile(resp, r, tempAudioPath)

	defer os.Remove(tempAudioPath)
	defer os.Remove(tempCoverPath)
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

func GetSCInfo(url string) SCInfo {
	infoJSONCmd := exec.Command("yt-dlp", "--no-playlist", "-j", url)

	infoJSON, err := infoJSONCmd.Output()
	if err != nil {
		log.Println("error infoJSONCmd:", err)
	}

	var info SoundCloudData
	err = json.Unmarshal(infoJSON, &info)
	if err != nil {
		log.Println("error parsing JSON:", err)
	}

	var thumbnailURL string
	for _, thumbnail := range info.Thumbnails {
		if thumbnail.ID == "original" {
			thumbnailURL = thumbnail.URL
			break
		}
	}

	infoWithThumb := SCInfo{
		Title:        info.Title,
		Uploader:     info.Uploader,
		ThumbnailURL: thumbnailURL,
	}

	return infoWithThumb
}

func getSCCover(thumbURL string, tempCoverPath string) {
	resp, err := http.Get(thumbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	img, err := imaging.Decode(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = imaging.Save(img, tempCoverPath)
	if err != nil {
		log.Fatal(err)
	}
}
