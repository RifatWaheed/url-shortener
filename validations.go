package main

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

var ErrUrlEmpty = errors.New("url is required")
var ErrUrlLengthCapExceeded = errors.New("url length is too large")
var ErrUrlParsingFailed = errors.New("url is invalid")
var ErrUrlSchemeNotValid = errors.New("url must use http or https")
var ErrUrlHostMissing = errors.New("url must include a host")

func ValidateUrl(req ShortenRequest) error {
	if req.Url == "" {
		return ErrUrlEmpty
	}

	if len(req.Url) > 2048 {
		return ErrUrlLengthCapExceeded
	}

	parsed, err := url.ParseRequestURI(req.Url)

	if err != nil {
		return ErrUrlParsingFailed
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrUrlSchemeNotValid
	}

	if parsed.Host == "" {
		return ErrUrlHostMissing
	}

	return nil
}

func isValidShortCode(shortCode string) bool {
	if len(shortCode) > codeLen {
		return false
	}
	for i := 0; i < len(shortCode); i++ {
		if !strings.ContainsRune(alphabet, rune(shortCode[i])) {
			return false
		}
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
