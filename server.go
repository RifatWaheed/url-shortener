package main

import (
	"net/http"
	"time"
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
	mux.HandleFunc("GET /slow", (func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte("done\n"))
	}))

	return mux
}
