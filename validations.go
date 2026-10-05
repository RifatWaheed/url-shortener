package main

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrUrlEmpty             = errors.New("url is required")
	ErrUrlLengthCapExceeded = errors.New("url length is too large")
	ErrUrlParsingFailed     = errors.New("url is invalid")
	ErrUrlSchemeNotValid    = errors.New("url must use http or https")
	ErrUrlHostMissing       = errors.New("url must include a host")

	ErrShortCodeEmptyString      = errors.New("short code can't be empty string")
	ErrShortCodeLengthExceeded   = errors.New("short code length exceeded")
	ErrShortCodeInvalidCharacter = errors.New("short code has invalid character")

	ErrBodyContentType     = errors.New("Content-Type must be application/json")
	ErrBodyTooLarge        = errors.New("request body must not exceed 1MB")
	ErrBodyInvalidJSON     = errors.New("invalid JSON body")
	ErrBodyMultipleObjects = errors.New("body must contain a single JSON object")
)

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

func ValidateShortCode(shortCode string) error {
	if len(shortCode) == 0 {
		return ErrShortCodeEmptyString
	}
	if len(shortCode) > codeLen {
		return ErrShortCodeLengthExceeded
	}
	for i := 0; i < len(shortCode); i++ {
		if !strings.ContainsRune(alphabet, rune(shortCode[i])) {
			return ErrShortCodeInvalidCharacter
		}
	}

	return nil
}

func DecodeJSONBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var body T
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return body, ErrBodyContentType
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) //1MB cap so decoder doesn't start decoding any huge body

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return body, ErrBodyTooLarge
		}

		return body, ErrBodyInvalidJSON
	}

	if decoder.More() {
		return body, ErrBodyMultipleObjects
	}

	return body, nil
}
