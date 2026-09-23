package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxCodeAttempts = 5

var ErrCodeExhausted = errors.New("could not generate a unique short code")
var ErrLinkNotFound = errors.New("link not found")
var ErrLinkExpired = errors.New("link expired")

func CreateLink(ctx context.Context, db *pgxpool.Pool, longURL string) (string, error) {
	for range maxCodeAttempts {
		shortCode, err := GenerateShortCode()
		if err != nil {
			return "", err
		}

		_, err = db.Exec(ctx,
			`INSERT INTO links (short_code, long_url) VALUES ($1, $2)`,
			shortCode, longURL)
		if err == nil {
			return shortCode, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return "", err // real db failure
	}

	return "", ErrCodeExhausted
}

func GetLongUrl(ctx context.Context, db *pgxpool.Pool, shortCode string) (string, error) {
	var longURL string
	var expires_at *time.Time

	err := db.QueryRow(ctx,
		`SELECT long_url, expires_at FROM links WHERE short_code = $1 `, shortCode).Scan(&longURL, &expires_at)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrLinkNotFound
	}
	if err != nil {
		return "", err
	}

	if expires_at != nil && time.Now().After(*expires_at) {
		return "", ErrLinkExpired
	}

	return longURL, nil
}
