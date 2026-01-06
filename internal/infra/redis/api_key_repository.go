package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hateblog/internal/domain/api_key"
	"hateblog/internal/platform/cache"
)

// APIKeyRepository stores API keys in Redis.
type APIKeyRepository struct {
	client *cache.Cache
}

// NewAPIKeyRepository creates a new APIKeyRepository.
func NewAPIKeyRepository(client *cache.Cache) *APIKeyRepository {
	return &APIKeyRepository{
		client: client,
	}
}

// Store saves an API key to Redis.
func (r *APIKeyRepository) Store(ctx context.Context, k *api_key.APIKey) error {
	if k == nil {
		return fmt.Errorf("api key is nil")
	}

	// Serialize to JSON
	data, err := json.Marshal(k)
	if err != nil {
		return fmt.Errorf("marshal api key: %w", err)
	}

	// Build Redis key
	redisKey := fmt.Sprintf("api_key:id:%s", k.ID.String())

	// Calculate TTL if expires_at is set
	var ttl time.Duration
	if k.ExpiresAt != nil {
		ttl = time.Until(*k.ExpiresAt)
		if ttl <= 0 {
			return fmt.Errorf("api key has already expired")
		}
	}

	// Store in Redis
	if err := r.client.Set(ctx, redisKey, data, ttl); err != nil {
		return fmt.Errorf("store api key in redis: %w", err)
	}

	return nil
}

// GetByID retrieves an API key by its ID.
func (r *APIKeyRepository) GetByID(ctx context.Context, id api_key.ID) (*api_key.APIKey, error) {
	// Build Redis key
	redisKey := fmt.Sprintf("api_key:id:%s", id.String())

	// Get from Redis
	data, err := r.client.GetBytes(ctx, redisKey)
	if err != nil {
		return nil, fmt.Errorf("get api key from redis: %w", err)
	}

	// Deserialize
	var k api_key.APIKey
	if err := json.Unmarshal(data, &k); err != nil {
		return nil, fmt.Errorf("unmarshal api key: %w", err)
	}

	return &k, nil
}
