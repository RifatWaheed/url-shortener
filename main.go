package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	if err := run(); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}

}

func run() error {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Warn("unable to find env variables", "err", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required (set it in .env or the environment)")
	}

	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		return errors.New("BASE_URL is required (the public URL short links start with, e.g. https://sho.rt)")
	}

	addr := getEnv("ADDR", "127.0.0.1:8080")

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("create connection pool: %w", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	slog.Info("connected to database")

	// =============================================================================//

	srv := &Server{
		db:      pool,
		baseURL: baseURL,
	}

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           logRequests(recoverPanic(srv.routes())),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("listening on", "addr", addr)

	return httpSrv.ListenAndServe()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
