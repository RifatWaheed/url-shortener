package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type HealthResponseObj struct {
	Message string `json:"message"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	healthResponse := HealthResponseObj{Message: "server is up"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		log.Printf("health: encode failed: %v", err)
	}
}

func helloApiHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}
