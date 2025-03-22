package converter

import (
	"io"
	"log"
	"net/http"
	"os/exec"
)

func Tiktok2mp4(w http.ResponseWriter, r *http.Request, url string) {
	videoCmd := exec.Command("yt-dlp", "-f", "'best'", url, "-o", "-")

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

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Disposition", `attachment; filename="audio.mp4"`)

	if _, err := io.Copy(w, videoPipe); err != nil {
		log.Println("Streaming error:", err)
	}

	if err := videoCmd.Wait(); err != nil {
		log.Println("yt-dlp execution error:", err)
	}
}
