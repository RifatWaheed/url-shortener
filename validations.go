package main

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/url"
)

func ValidateUrl(w http.ResponseWriter, req ShortenRequest) bool {
	parsed, err := url.Parse(req.Url)
	if req.Url == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return false
	}
	if err != nil {
		http.Error(w, "url is not a valid URL", http.StatusBadRequest)
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		http.Error(w, "url must use http or https", http.StatusBadRequest)
		return false
	}
	if parsed.Host == "" {
		http.Error(w, "url must include a host", http.StatusBadRequest)
		return false
	}

	return true
}

func DecodeJSONBody[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return body, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) //1MB cap so decoder doesn't start decoding any huge body

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return body, false
	}

	if decoder.More() {
		http.Error(w, "body must contain a single JSON object", http.StatusBadRequest)
		return body, false
	}

	return body, true
}
