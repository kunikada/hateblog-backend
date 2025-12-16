package hostname

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	tests := map[string]string{
		"Example.com":    "example.com",
		"https://a.io/":  "a.io",
		"sub.domain.com": "sub.domain.com",
	}
	for input, expected := range tests {
		got, err := Normalize(input)
		require.NoError(t, err, input)
		require.Equal(t, expected, got)
	}
}

func TestNormalizeInvalid(t *testing.T) {
	for _, input := range []string{"", "://bad", "foo/bar", "spa ce", "あ.example.com"} {
		_, err := Normalize(input)
		require.Error(t, err, input)
	}
}
