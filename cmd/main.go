package main

//TODO: поразмышлять насчет htmx

import (
	"fmt"
	"net/http"

	"media-converter/m/v2/internal/handler"
)

func main() {
	http.HandleFunc("/", handler.Home)
	http.HandleFunc("POST /convertYoutube2audio", handler.ConvertYoutube2audio)
	fmt.Println("server starting on port: 8080")
	http.ListenAndServe(":8080", nil)
}
