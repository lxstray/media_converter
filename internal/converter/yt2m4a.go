package converter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

//TODO: попробовать прочитать результат ffmpeg в слайс []byte

func Yt2m4a(w http.ResponseWriter, r *http.Request, info VideoInfo) {
	tempAudioPath, tempCoverPath := GenerateTempFilesNames()
	defer os.Remove(tempAudioPath)
	defer os.Remove(tempCoverPath)

	getCover(info.VideoID, tempCoverPath)

	audioChan := make(chan io.ReadCloser)
	errChan := make(chan error, 2)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		audioCmd := exec.Command("yt-dlp", "-x", "--audio-quality", "0", "-f", "m4a", "--no-playlist", info.URL, "-o", "-")
		audioPipe, err := audioCmd.StdoutPipe()
		if err != nil {
			log.Println("yt-dlp pipe:", err)
			errChan <- err
		}
		audioChan <- audioPipe
		if err := audioCmd.Run(); err != nil {
			log.Println("yt-dlp cmd:", err)
			errChan <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ffmpegCmd := exec.Command("ffmpeg", "-i", "pipe:0", "-i", tempCoverPath, "-map", "0", "-map", "1", "-c", "copy", "-metadata", "artist="+info.Uploader, "-metadata", "title="+info.Title, "-disposition:v:0", "attached_pic", tempAudioPath)
		ffmpegCmd.Stdin = <-audioChan
		if err := ffmpegCmd.Run(); err != nil {
			log.Println("ffmpeg cmd:", err)
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

type VideoInfo struct {
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
	VideoID  string `json:"id"`
	URL      string `json:"url"`
}

func GetYoutubeInfo(url string) (VideoInfo, error) {
	infoJSONCmd := exec.Command("yt-dlp", "--no-playlist", "-j", url)

	var info VideoInfo

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

// TODO: fatal
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

func GenerateTempFilesNames() (string, string) {
	tempAudio := "tmp/audio/" + uuid.New().String() + ".m4a"
	tempCover := "tmp/cover/" + uuid.New().String() + ".png"

	return tempAudio, tempCover
}

type PlaylistInfo struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
	VideoID  string `json:"id"`
}

func GetPlaylistInfo(w http.ResponseWriter, r *http.Request, url string) {
	playlistCmd := exec.Command("yt-dlp", "--dump-json", "--flat-playlist", url)
	output, err := playlistCmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error in playlistCmd: %v", err), http.StatusInternalServerError)
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var infos []PlaylistInfo

	for scanner.Scan() {
		line := scanner.Text()
		var info PlaylistInfo
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			log.Printf("Pasing error JSON: %v", err)
			continue
		}
		infos = append(infos, info)
	}

	if err := scanner.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Scanner error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(infos); err != nil {
		http.Error(w, fmt.Sprintf("Encoder error: %v", err), http.StatusInternalServerError)
		return
	}
}
