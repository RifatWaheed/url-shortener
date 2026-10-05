package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, resp any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf(" encode failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_server_error"
	message := "Something went wrong"
	switch {
	// request body decoding
	case errors.Is(err, ErrBodyContentType):
		status, code, message = http.StatusUnsupportedMediaType, "content_type_invalid", ErrBodyContentType.Error()
	case errors.Is(err, ErrBodyTooLarge):
		status, code, message = http.StatusRequestEntityTooLarge, "body_too_large", ErrBodyTooLarge.Error()
	case errors.Is(err, ErrBodyInvalidJSON):
		status, code, message = http.StatusBadRequest, "json_invalid", ErrBodyInvalidJSON.Error()
	case errors.Is(err, ErrBodyMultipleObjects):
		status, code, message = http.StatusBadRequest, "json_multiple_objects", ErrBodyMultipleObjects.Error()

	// shorten: request validation
	case errors.Is(err, ErrUrlEmpty):
		status, code, message = http.StatusBadRequest, "url_required", ErrUrlEmpty.Error()
	case errors.Is(err, ErrUrlLengthCapExceeded):
		status, code, message = http.StatusBadRequest, "url_too_long", ErrUrlLengthCapExceeded.Error()
	case errors.Is(err, ErrUrlParsingFailed):
		status, code, message = http.StatusBadRequest, "url_invalid", ErrUrlParsingFailed.Error()
	case errors.Is(err, ErrUrlSchemeNotValid):
		status, code, message = http.StatusBadRequest, "url_scheme_invalid", ErrUrlSchemeNotValid.Error()
	case errors.Is(err, ErrUrlHostMissing):
		status, code, message = http.StatusBadRequest, "url_host_missing", ErrUrlHostMissing.Error()

	// shorten: storage
	case errors.Is(err, ErrCodeExhausted):
		status, code, message = http.StatusServiceUnavailable, "short_code_exhausted", "could not shorten url, please try again"

	// redirect: a malformed code can never exist, so it is reported the same as a missing link
	case errors.Is(err, ErrShortCodeEmptyString),
		errors.Is(err, ErrShortCodeLengthExceeded),
		errors.Is(err, ErrShortCodeInvalidCharacter),
		errors.Is(err, ErrLinkNotFound):
		status, code, message = http.StatusNotFound, "link_not_found", ErrLinkNotFound.Error()
	case errors.Is(err, ErrLinkExpired):
		status, code, message = http.StatusGone, "link_expired", ErrLinkExpired.Error()
	}

	if status >= 500 {
		log.Printf("%v", err)
	}

	writeJSON(w, status, errorResponse{Error: errorBody{Code: code, Message: message}})
}
