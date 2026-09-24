package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type HealthResponseObj struct {
	Message string `json:"message"`
}

type ShortenRequest struct {
	Url string `json:"url"`
}

type ShortenResponse struct {
	ShortCode string `json:"shortCode"`
	ShortURL  string `json:"shortUrl"`
}

func (s *Server) shortenUrlHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeJSONBody[ShortenRequest](w, r)
	if !ok {
		return
	}

	if err := ValidateUrl(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortCode, err := CreateLink(r.Context(), s.db, req.Url)
	if err != nil {
		log.Printf("shorten: %v", err)
		http.Error(w, "could not shorten url", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := ShortenResponse{ShortCode: shortCode, ShortURL: s.baseURL + "/" + shortCode}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("shorten: encode failed: %v", err)
	}
}

func (s *Server) redirectHandler(w http.ResponseWriter, r *http.Request) {

	shortCode := r.PathValue("code")
	if !isValidShortCode(shortCode) {
		http.NotFound(w, r)
		return
	}

	longURL, err := GetLongUrl(r.Context(), s.db, shortCode)

	switch {
	case errors.Is(err, ErrLinkNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, ErrLinkExpired):
		http.Error(w, "Link Expired", http.StatusGone)
		return
	case err != nil:
		log.Printf("redirect : %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	healthResponse := HealthResponseObj{Message: "server is up"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		log.Printf("health: encode failed: %v", err)
	}
}
