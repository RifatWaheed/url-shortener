package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Printf("unable to find env variables : %v", err)
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
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

	addr := getEnv("ADDR", "127.0.0.1:8080")
	baseURL := getEnv("BASE_URL", "http://"+addr)

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
