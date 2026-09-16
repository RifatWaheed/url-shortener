package main

import (
	"encoding/json"
	"log"
	"mime"
	"net/http"
)

type HealthResponseObj struct {
	Message string `json:"message"`
}

type ShortenRequest struct {
	Url string `json:"url"`
}

func shortenUrlHandler(w http.ResponseWriter, r *http.Request) {

}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	healthResponse := HealthResponseObj{Message: "server is up"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		log.Printf("health: encode failed: %v", err)
	}
}

func helloApiHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

func decodeJSONBody[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return body, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) //1MB cap so decoder doesn't start decoding any huge body

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req ShortenRequest
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return body, false
	}

	if decoder.More() {
		http.Error(w, "body must contain a single JSON object", http.StatusBadRequest)
		return body, false
	}

	if req.Url == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return body, false
	}

	return body, true
}
