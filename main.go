package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/hello", helloApiHandler)
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))

}
