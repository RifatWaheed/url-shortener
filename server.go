package main

import (
	"net/http"
)

type Server struct {
	db      LinkStore
	baseURL string
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /shorten", s.shortenUrlHandler)
	mux.HandleFunc("GET /{code}", s.redirectHandler)
	mux.HandleFunc("GET /health", healthHandler)

	return mux
}
