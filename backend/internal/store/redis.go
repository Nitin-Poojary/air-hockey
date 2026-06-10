package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	playerExpiry = 30 * 24 * time.Hour
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(redisURL string) (*RedisStore, error) {
	var opts *redis.Options
	var err error

	switch {
	case redisURL == "":
		opts = &redis.Options{Addr: "localhost:6379"}
	case strings.HasPrefix(redisURL, "redis://") || strings.HasPrefix(redisURL, "rediss://"):
		opts, err = redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_URL: %w", err)
		}
	default:
		// host:port (e.g. localhost:6379) — do not ParseURL; bare "redis:6379" would become localhost.
		opts = &redis.Options{Addr: redisURL}
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", opts.Addr, err)
	}

	return &RedisStore{client: client}, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) GetPlayerIDByName(ctx context.Context, normalized string) (string, error) {
	nameKey := fmt.Sprintf("player:name:%s", normalized)
	
	playerID, err := s.client.Get(ctx, nameKey).Result()
	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		return "", err
	}

	s.client.Expire(ctx, nameKey, playerExpiry)
	idKey := fmt.Sprintf("player:id:%s", playerID)
	s.client.Expire(ctx, idKey, playerExpiry)
	s.client.HSet(ctx, idKey, "lastSeen", time.Now().Format(time.RFC3339))

	return playerID, nil
}

func (s *RedisStore) GetPlayerNameByID(ctx context.Context, playerID string) (string, error) {
	idKey := fmt.Sprintf("player:id:%s", playerID)
	name, err := s.client.HGet(ctx, idKey, "name").Result()
	if err == redis.Nil {
		return "", nil
	}
	return name, err
}

func (s *RedisStore) RegisterPlayerName(ctx context.Context, normalized string, playerID string, originalName string) (bool, error) {
	nameKey := fmt.Sprintf("player:name:%s", normalized)
	set, err := s.client.SetNX(ctx, nameKey, playerID, playerExpiry).Result()
	if err != nil {
		return false, err
	}

	if !set {
		return false, nil
	}

	idKey := fmt.Sprintf("player:id:%s", playerID)
	_, err = s.client.HSet(ctx, idKey, map[string]interface{}{
		"name":     originalName,
		"lastSeen": time.Now().Format(time.RFC3339),
	}).Result()
	if err != nil {
		fmt.Printf("Warning: failed to write player:id hash for %s: %v\n", playerID, err)
	}
	s.client.Expire(ctx, idKey, playerExpiry)

	return true, nil
}
