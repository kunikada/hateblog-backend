package tag

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNormalizeName(t *testing.T) {
	require.Equal(t, "go", NormalizeName(" Go "))
	require.Equal(t, "web-dev", NormalizeName("Web-Dev"))
}

func TestNew(t *testing.T) {
	tag, err := New(uuid.New(), "  Go ")
	require.NoError(t, err)
	require.Equal(t, "go", tag.Name)
}

func TestNewInvalid(t *testing.T) {
	_, err := New(uuid.New(), "  ")
	require.ErrorIs(t, err, ErrInvalidTag)
}
