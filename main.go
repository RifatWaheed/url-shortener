package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Printf("unable to find env variables : %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required (set it in .env or the environment)")
	}

	// BASE_URL is the public URL users click, which is separate from the address
	// the app listens on (behind nginx, Docker, Fly or a load balancer they differ),
	// so it can't be derived from ADDR.
	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		log.Fatal("BASE_URL is required (the public URL short links start with, e.g. https://sho.rt)")
	}

	addr := getEnv("ADDR", "127.0.0.1:8080")

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("unable to reach database : %v", err)
	}

	log.Println("connected to database")

	// =============================================================================//

	srv := &Server{db: pool, baseURL: baseURL}
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.routes()))

}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
