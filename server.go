package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	db      *pgxpool.Pool
	baseURL string
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /shorten", s.shortenUrlHandler)
	mux.HandleFunc("GET /{code}", s.redirectHandler)

	return mux
}
