package api_key

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"hateblog/internal/domain/api_key"
	"hateblog/internal/domain/repository"
)

// Service handles API key generation and management.
type Service struct {
	repo      repository.APIKeyRepository
	keyPrefix string
}

// NewService creates a new API key service.
func NewService(repo repository.APIKeyRepository, keyPrefix string) *Service {
	if keyPrefix == "" {
		keyPrefix = "hb_live_"
	}
	return &Service{
		repo:      repo,
		keyPrefix: keyPrefix,
	}
}

// GenerateParams contains parameters for generating an API key.
type GenerateParams struct {
	Name        *string
	Description *string
	ExpiresAt   *time.Time
	CreatedIP   *string
	CreatedUA   *string
	CreatedRef  *string
}

// GeneratedAPIKey represents a newly generated API key with its plaintext value.
type GeneratedAPIKey struct {
	ID          uuid.UUID
	Key         string
	Name        *string
	Description *string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

// GenerateAPIKey creates a new API key with a random value and stores its hash.
func (s *Service) GenerateAPIKey(ctx context.Context, params GenerateParams) (*GeneratedAPIKey, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("api key service not initialized")
	}

	// Generate UUID
	id := uuid.New()

	// Generate random 32-character hex string
	randomBytes := make([]byte, 16) // 16 bytes = 32 hex characters
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}
	randomHex := hex.EncodeToString(randomBytes)

	// Create full key
	plaintextKey := s.keyPrefix + randomHex

	// Hash the key with bcrypt (cost factor 12)
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(plaintextKey), 12)
	if err != nil {
		return nil, fmt.Errorf("hash api key: %w", err)
	}

	// Current time
	now := time.Now().UTC()

	// Create domain API key
	apiKey, err := api_key.New(api_key.Params{
		ID:               id,
		KeyHash:          string(hashedKey),
		Name:             params.Name,
		Description:      params.Description,
		CreatedAt:        now,
		ExpiresAt:        params.ExpiresAt,
		CreatedIP:        params.CreatedIP,
		CreatedUserAgent: params.CreatedUA,
		CreatedReferrer:  params.CreatedRef,
	})
	if err != nil {
		return nil, fmt.Errorf("create api key domain object: %w", err)
	}

	// Store in repository
	if err := s.repo.Store(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("store api key: %w", err)
	}

	// Return generated key with plaintext
	return &GeneratedAPIKey{
		ID:          id,
		Key:         plaintextKey,
		Name:        params.Name,
		Description: params.Description,
		CreatedAt:   now,
		ExpiresAt:   params.ExpiresAt,
	}, nil
}
