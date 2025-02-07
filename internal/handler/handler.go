package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"text/template"

	"media-converter/m/v2/internal/converter"
)

func Home(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/template/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Message string
	}{
		Message: "media converter",
	}

	tmpl.Execute(w, data)
}

func ConvertToAudio(w http.ResponseWriter, r *http.Request) {
	url := r.PostFormValue("url")
	if url == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	//TODO: выводить ошилбку что не та ссылка, если это не ютуб или ск
	if strings.Contains(url, "youtu") {
		if strings.Contains(url, "playlist") {
			converter.GetPlaylistInfo(w, r, url)
		} else {
			info := converter.GetYoutubeInfo(url) //TODO: запустить в горутину если получиться получить video id другим способом
			info.URL = url
			converter.Yt2m4a(w, r, info)
		}
	}
	if strings.Contains(url, "soundcloud") {
		converter.Sc2m4a(&w, r, url)
	}
}

func DownloadFromPlaylist(w http.ResponseWriter, r *http.Request) {
	var data converter.VideoInfo
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "JSON decoder error", http.StatusBadRequest)
		return
	}
	converter.Yt2m4a(w, r, data)
}
