package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

const maxCodeAttempts = 5

var ErrCodeExhausted = errors.New("could not generate a unique short code")
var ErrLinkNotFound = errors.New("link not found")
var ErrLinkExpired = errors.New("link expired")

func (pgStore *PostgresStore) CreateLink(ctx context.Context, longURL string) (string, error) {
	for range maxCodeAttempts {
		shortCode := GenerateShortCode()

		_, err := pgStore.db.Exec(ctx,
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

type Link struct {
	LongURL   string
	ExpiresAt *time.Time
}

func (pgStore *PostgresStore) GetLink(ctx context.Context, shortCode string) (Link, error) {

	var link Link

	err := pgStore.db.QueryRow(ctx,
		`SELECT long_url, expires_at FROM links WHERE short_code = $1 `, shortCode).Scan(&link.LongURL, &link.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Link{}, ErrLinkNotFound
	}

	if err != nil {
		return Link{}, err
	}

	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return Link{}, ErrLinkExpired
	}

	return link, nil
}
