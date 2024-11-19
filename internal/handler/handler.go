package handler

import (
	"net/http"
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
		Message: "goodbye world",
	}

	tmpl.Execute(w, data)
}

func ConvertYoutube2audio(w http.ResponseWriter, r *http.Request) {
	url := r.PostFormValue("url")
	if url == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}
	converter.Yt2m4a(url)
}
