package main

import (
	"context"
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

type LinkStore interface {
	CreateLink(ctx context.Context, longURL string) (string, error)
	GetLongUrl(ctx context.Context, shortCode string) (string, error)
}

func (srv *Server) shortenUrlHandler(w http.ResponseWriter, r *http.Request) {
	req, err := DecodeJSONBody[ShortenRequest](w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := ValidateUrl(req); err != nil {
		writeError(w, err)
		return
	}

	shortCode, err := srv.db.CreateLink(r.Context(), req.Url)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := ShortenResponse{ShortCode: shortCode, ShortURL: srv.baseURL + "/" + shortCode}
	writeJSON(w, http.StatusCreated, resp)
}

func (srv *Server) redirectHandler(w http.ResponseWriter, r *http.Request) {

	shortCode := r.PathValue("code")
	if err := ValidateShortCode(shortCode); err != nil {
		writeError(w, err)
		return
	}

	longURL, err := srv.db.GetLongUrl(r.Context(), shortCode)
	if err != nil {
		writeError(w, err)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	healthResponse := HealthResponseObj{Message: "server is up"}
	writeJSON(w, http.StatusOK, healthResponse)
}
