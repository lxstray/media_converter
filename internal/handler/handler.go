package handler

import (
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

	if strings.Contains(url, "youtu") {
		converter.Yt2m4a(&w, r, url)
	}
	if strings.Contains(url, "soundcloud") {
		converter.Sc2m4a(&w, r, url)
	}
}
