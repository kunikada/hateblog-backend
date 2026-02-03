package main

import (
	"testing"
	"time"
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

func TestResolveCreatedAt(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, jst)

	tests := []struct {
		name     string
		now      time.Time
		postedAt time.Time
		want     time.Time
	}{
		{
			name:     "24時間未満は現在時刻",
			now:      now,
			postedAt: now.Add(-23*time.Hour - 59*time.Minute - 59*time.Second),
			want:     now,
		},
		{
			name:     "ちょうど24時間前はposted_at",
			now:      now,
			postedAt: now.Add(-24 * time.Hour),
			want:     now.Add(-24 * time.Hour),
		},
		{
			name:     "24時間超はposted_at",
			now:      now,
			postedAt: now.Add(-24*time.Hour - 1*time.Second),
			want:     now.Add(-24*time.Hour - 1*time.Second),
		},
		{
			name:     "タイムゾーン差分があっても24時間判定は絶対時刻基準",
			now:      now,
			postedAt: time.Date(2026, 2, 2, 3, 0, 1, 0, time.UTC), // JSTで 2026-02-02 12:00:01
			want:     now,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCreatedAt(tt.now, tt.postedAt)
			if !got.Equal(tt.want) {
				t.Fatalf("resolveCreatedAt() = %s, want %s", got, tt.want)
			}
		})
	}
}
