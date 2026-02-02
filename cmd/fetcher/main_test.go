package main

import (
	"testing"
)

func TestNullableText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  any
	}{
		{
			name:  "normal text",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "spaces",
			input: "   ",
			want:  nil,
		},
		{
			name:  "tabs and newlines",
			input: "\t\n",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nullableText(tt.input)
			if got != tt.want {
				t.Errorf("nullableText(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid ascii",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "valid utf8",
			input: "こんにちは",
			want:  "こんにちは",
		},
		{
			name:  "invalid utf8",
			input: "hello\xffworld",
			want:  "helloworld",
		},
		{
			name:  "mixed valid and invalid",
			input: string([]byte{0xe3, 0x81, 0x82, 0xff, 0xe3, 0x81, 0x84}), // "あ" + invalid + "い"
			want:  "あい",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeUTF8(tt.input); got != tt.want {
				t.Errorf("sanitizeUTF8(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
