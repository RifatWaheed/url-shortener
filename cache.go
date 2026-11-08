package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedStore struct {
	LinkStore
	redisClient     *redis.Client
	ttl             time.Duration
	linkNotFoundTTL time.Duration
}

const linkNotFound = "not-found"

func (c *CachedStore) GetLink(ctx context.Context, shortCode string) (Link, error) {
	key := "link:" + shortCode
	val, err := c.redisClient.Get(ctx, key).Result()

	// cache miss
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.Warn("Redis error ", "err", err)
		}

		fetchedLink, getLinkErr := c.LinkStore.GetLink(ctx, shortCode)
		if getLinkErr != nil {
			// Negative-cache only when a positive TTL is configured: in go-redis a
			// zero TTL means "no expiry", which would pin a 404 on this code forever.
			if errors.Is(getLinkErr, ErrLinkNotFound) && c.linkNotFoundTTL > 0 {
				_, setErr := c.redisClient.Set(ctx, key, linkNotFound, c.linkNotFoundTTL).Result()
				if setErr != nil {
					slog.Warn("SET error", "err", setErr)
				}
			}
			return Link{}, getLinkErr
		}

		data, jsonErr := json.Marshal(fetchedLink)
		if jsonErr != nil {
			return Link{}, fmt.Errorf("JSON marshal error: %w", jsonErr)
		}

		duration := c.ttl
		if fetchedLink.ExpiresAt != nil {
			remaining := time.Until(*fetchedLink.ExpiresAt)
			if remaining <= 0 {
				// Expired between the DB read and now: serve it, but don't cache.
				// A zero/negative TTL in go-redis means "no expiry", which would pin a dead link.
				return fetchedLink, nil
			}
			duration = min(duration, remaining)
		}

		_, setErr := c.redisClient.Set(ctx, key, data, duration).Result()
		if setErr != nil {
			slog.Warn("SET error", "err", setErr)
		}

		return fetchedLink, nil
	}
	// if above section is not executed then its a cache hit
	if val == linkNotFound {
		return Link{}, ErrLinkNotFound
	}

	var link Link
	if err := json.Unmarshal([]byte(val), &link); err != nil {
		return Link{}, fmt.Errorf("JSON unmarshal error: %w", err)
	}
	return link, nil
}
