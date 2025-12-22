package tag

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name    string
		id      ID
		tagName string
		wantErr bool
		errMsg  string
		wantTag Tag
	}{
		{
			name:    "valid tag",
			id:      validID,
			tagName: "golang",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "golang",
			},
		},
		{
			name:    "normalizes tag name",
			id:      validID,
			tagName: "  Golang  ",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "golang",
			},
		},
		{
			name:    "uppercase tag name",
			id:      validID,
			tagName: "PROGRAMMING",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "programming",
			},
		},
		{
			name:    "mixed case with spaces",
			id:      validID,
			tagName: "  Web Development  ",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "web development",
			},
		},
		{
			name:    "empty name",
			id:      validID,
			tagName: "",
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "whitespace-only name",
			id:      validID,
			tagName: "   ",
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "tab and newline characters",
			id:      validID,
			tagName: "\t\n  \t",
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "single character tag",
			id:      validID,
			tagName: "a",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "a",
			},
		},
		{
			name:    "numeric tag name",
			id:      validID,
			tagName: "123",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "123",
			},
		},
		{
			name:    "special characters",
			id:      validID,
			tagName: "C++",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "c++",
			},
		},
		{
			name:    "tag with hyphens",
			id:      validID,
			tagName: "front-end",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "front-end",
			},
		},
		{
			name:    "tag with underscores",
			id:      validID,
			tagName: "web_development",
			wantErr: false,
			wantTag: Tag{
				ID:   validID,
				Name: "web_development",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.id, tt.tagName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Equal(t, Tag{}, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTag, got)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already normalized",
			input: "golang",
			want:  "golang",
		},
		{
			name:  "uppercase",
			input: "GOLANG",
			want:  "golang",
		},
		{
			name:  "mixed case",
			input: "GoLang",
			want:  "golang",
		},
		{
			name:  "leading spaces",
			input: "  golang",
			want:  "golang",
		},
		{
			name:  "trailing spaces",
			input: "golang  ",
			want:  "golang",
		},
		{
			name:  "leading and trailing spaces",
			input: "  golang  ",
			want:  "golang",
		},
		{
			name:  "mixed case with spaces",
			input: "  GoLang  ",
			want:  "golang",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  "",
		},
		{
			name:  "tabs and spaces",
			input: "\t golang \t",
			want:  "golang",
		},
		{
			name:  "newlines and spaces",
			input: "\n golang \n",
			want:  "golang",
		},
		{
			name:  "multiple words",
			input: "Web Development",
			want:  "web development",
		},
		{
			name:  "multiple words with extra spaces",
			input: "  Web   Development  ",
			want:  "web   development",
		},
		{
			name:  "special characters preserved",
			input: "C++",
			want:  "c++",
		},
		{
			name:  "numbers",
			input: "123",
			want:  "123",
		},
		{
			name:  "alphanumeric",
			input: "Go1.21",
			want:  "go1.21",
		},
		{
			name:  "hyphenated",
			input: "Front-End",
			want:  "front-end",
		},
		{
			name:  "underscored",
			input: "WEB_DEVELOPMENT",
			want:  "web_development",
		},
		{
			name:  "unicode characters",
			input: "日本語",
			want:  "日本語",
		},
		{
			name:  "mixed unicode and ascii",
			input: "  Golang 日本語  ",
			want:  "golang 日本語",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
