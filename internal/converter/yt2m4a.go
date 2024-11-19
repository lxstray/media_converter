package converter

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

//TODO: попробовать прочитать результат ffmpeg в слайс []byte
//TODO: подумать насчет log и fmt

func Yt2m4a(url string) {
	info := getInfo(url) //TODO: запустить в горутину если получиться получить video id другим способом
	fmt.Println("title: ", info.Title, ", author: ", info.Uploader, ", id", info.VideoID)

	tempAudioPath, tempCoverPath := generateTempFilesNames()
	fmt.Println(tempAudioPath, ", ", tempCoverPath)

	getCover(info.VideoID, tempCoverPath) //TODO: go и waitgroup

	audioChan := make(chan io.ReadCloser)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		audioCmd := exec.Command("yt-dlp", "-x", "-f", "m4a", "--no-playlist", url, "-o", "-")
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
		ffmpegCmd := exec.Command("ffmpeg", "-i", "pipe:0", "-i", tempCoverPath, "-map", "0", "-map", "1", "-c", "copy", "-metadata", "artist="+info.Uploader, "-metadata", "title="+info.Title, "-disposition:v:0", "attached_pic", tempAudioPath)
		ffmpegCmd.Stdin = <-audioChan
		if err := ffmpegCmd.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()
	fmt.Println("done -_-")
}

type VideoInfo struct {
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
	VideoID  string `json:"id"`
}

func getInfo(url string) VideoInfo {
	infoJSONCmd := exec.Command("yt-dlp", "--no-playlist", "-j", url)

	infoJSON, err := infoJSONCmd.Output()
	if err != nil {
		log.Println("error infoJSONCmd:", err)
	}

	var info VideoInfo
	err = json.Unmarshal(infoJSON, &info)
	if err != nil {
		log.Println("error parsing JSON:", err)
	}

	return info
}

func getCover(videoId string, tempCoverPath string) {
	coverUrl := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoId)

	resp, err := http.Get(coverUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var croppedImg *image.NRGBA

	if resp.StatusCode == 404 {
		coverUrl := fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", videoId)
		resp, err := http.Get(coverUrl)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		img, err := imaging.Decode(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		croppedImg = imaging.Crop(img, image.Rect(105, 45, 105+270, 45+270))
	} else {
		img, err := imaging.Decode(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		croppedImg = imaging.CropCenter(img, min(img.Bounds().Dx(), img.Bounds().Dy()), min(img.Bounds().Dx(), img.Bounds().Dy()))
	}

	finalImg := imaging.Resize(croppedImg, 1600, 1600, imaging.Lanczos)
	err = imaging.Save(finalImg, tempCoverPath)
	if err != nil {
		log.Fatal(err)
	}
}

func generateTempFilesNames() (string, string) {
	tempAudio := "tmp/audio/" + uuid.New().String() + ".m4a"
	tempCover := "tmp/cover/" + uuid.New().String() + ".png"

	return tempAudio, tempCover
}
