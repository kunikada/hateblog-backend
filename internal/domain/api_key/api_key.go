package api_key

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ID is the unique identifier for an API key.
type ID = uuid.UUID

// APIKey represents an API key with its metadata.
type APIKey struct {
	ID                ID
	KeyHash           string
	Name              *string
	Description       *string
	CreatedAt         time.Time
	ExpiresAt         *time.Time
	CreatedIP         *string
	CreatedUserAgent  *string
	CreatedReferrer   *string
}

// Params contains parameters for creating an API key.
type Params struct {
	ID                ID
	KeyHash           string
	Name              *string
	Description       *string
	CreatedAt         time.Time
	ExpiresAt         *time.Time
	CreatedIP         *string
	CreatedUserAgent  *string
	CreatedReferrer   *string
}

var (
	// ErrInvalidAPIKey is returned when API key validation fails.
	ErrInvalidAPIKey = errors.New("invalid API key")
)

// New creates a new APIKey with validation.
func New(params Params) (*APIKey, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}

	return &APIKey{
		ID:                params.ID,
		KeyHash:           params.KeyHash,
		Name:              params.Name,
		Description:       params.Description,
		CreatedAt:         params.CreatedAt,
		ExpiresAt:         params.ExpiresAt,
		CreatedIP:         params.CreatedIP,
		CreatedUserAgent:  params.CreatedUserAgent,
		CreatedReferrer:   params.CreatedReferrer,
	}, nil
}

func validateParams(params Params) error {
	if params.KeyHash == "" {
		return fmt.Errorf("%w: key_hash is required", ErrInvalidAPIKey)
	}

	if params.Name != nil && len(*params.Name) > 100 {
		return fmt.Errorf("%w: name must be at most 100 characters", ErrInvalidAPIKey)
	}

	if params.Description != nil && len(*params.Description) > 500 {
		return fmt.Errorf("%w: description must be at most 500 characters", ErrInvalidAPIKey)
	}

	if params.ExpiresAt != nil && !params.CreatedAt.IsZero() && params.ExpiresAt.Before(params.CreatedAt) {
		return fmt.Errorf("%w: expires_at must be after created_at", ErrInvalidAPIKey)
	}

	return nil
}
