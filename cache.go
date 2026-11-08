package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedStore struct {
	LinkStore
	redisClient *redis.Client
	ttl         time.Duration
}

func (c *CachedStore) GetLink(ctx context.Context, shortCode string) (Link, error) {
	key := "link:" + shortCode
	val, err := c.redisClient.Get(ctx, key).Result()

	// cache miss
	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.Info("Cache miss ", "err", err)
		} else {
			slog.Info("Redis error ", "err", err)
		}

		fetchedLink, getLinkErr := c.LinkStore.GetLink(ctx, shortCode)
		if getLinkErr != nil {
			return Link{}, getLinkErr
		}

		data, jsonErr := json.Marshal(fetchedLink)
		duration := time.Until(*fetchedLink.ExpiresAt)

		if jsonErr != nil {
			fmt.Errorf("JSON marshall error : %w", jsonErr)
		}

		setCommandStatus, setErr := c.redisClient.Set(ctx, key, data, duration).Result()
		if setErr != nil {
			slog.Info("SET error", "err", setErr)
		}
		slog.Info("SET command Status", "status", setCommandStatus)
	}
	// if above section is not executed then its a cache hit
	json.Unmarshal([]byte(val))
}
