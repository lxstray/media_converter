package main

//TODO: поразмышлять насчет htmx

import (
	"fmt"
	"media-converter/m/v2/internal/handler"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler.Home)
	http.HandleFunc("POST /convertToAudio", handler.ConvertToAudio)
	http.HandleFunc("POST /downloadFromPlaylist", handler.DownloadFromPlaylist)
	fmt.Println("server starting on port: 8080")
	http.ListenAndServe(":8080", nil)
}
