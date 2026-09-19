package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxCodeAttempts = 5

var ErrCodeExhausted = errors.New("could not generate a unique short code")

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
