package fbhttp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	redisNyxUploadPrefix    = "nyx:upload:"
	redisLegacyUploadPrefix = "filebrowser:upload:"
)

// redisUploadCache is an upload cache for multi replica deployments
type redisUploadCache struct {
	client *redis.Client
}

func newRedisUploadCache(redisURL string) (*redisUploadCache, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("redis URL is required")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &redisUploadCache{client: client}, nil
}

func (c *redisUploadCache) nyxKey(filePath string) string {
	return redisNyxUploadPrefix + filePath
}

func (c *redisUploadCache) legacyKey(filePath string) string {
	return redisLegacyUploadPrefix + filePath
}

// Register stores the upload length using the canonical Nyx namespace.
// The scoped removal callback is unused by the redis backend, which does not
// delete partial files on eviction.
func (c *redisUploadCache) Register(filePath string, fileSize int64, _ func() error) {
	err := c.client.Set(context.Background(), c.nyxKey(filePath), fileSize, uploadCacheTTL).Err()
	if err != nil {
		log.Printf("failed to register upload in redis cache: %v", err)
	}
}

// Complete removes the upload from both the Nyx namespace and legacy namespace.
func (c *redisUploadCache) Complete(filePath string) {
	err := c.client.Del(context.Background(), c.nyxKey(filePath), c.legacyKey(filePath)).Err()
	if err != nil {
		log.Printf("failed to complete upload in redis cache: %v", err)
	}
}

// GetLength returns the expected file size, checking the canonical Nyx namespace first
// and falling back to the legacy filebrowser:upload: namespace to preserve in-flight uploads.
func (c *redisUploadCache) GetLength(filePath string) (int64, error) {
	ctx := context.Background()

	// 1. Try canonical Nyx key
	result, err := c.client.Get(ctx, c.nyxKey(filePath)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 2. Safe read-fallback to legacy key without deleting or rewriting
			result, err = c.client.Get(ctx, c.legacyKey(filePath)).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					return 0, fmt.Errorf("no active upload found for the given path")
				}
				return 0, fmt.Errorf("redis error: %w", err)
			}
		} else {
			return 0, fmt.Errorf("redis error: %w", err)
		}
	}

	size, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid upload length in cache: %w", err)
	}

	c.Touch(filePath)

	return size, nil
}

// Touch refreshes the TTL on the active key (Nyx or legacy).
func (c *redisUploadCache) Touch(filePath string) {
	ctx := context.Background()
	ok, err := c.client.Expire(ctx, c.nyxKey(filePath), uploadCacheTTL).Result()
	if err != nil {
		log.Printf("failed to touch upload in redis cache: %v", err)
	}
	if !ok {
		// If Nyx key does not exist, refresh legacy key if active
		if err := c.client.Expire(ctx, c.legacyKey(filePath), uploadCacheTTL).Err(); err != nil {
			log.Printf("failed to touch legacy upload in redis cache: %v", err)
		}
	}
}

func (c *redisUploadCache) Close() {
	c.client.Close()
}
