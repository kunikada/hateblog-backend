package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTrustedProxies(t *testing.T) {
	prefixes, err := ParseTrustedProxies([]string{"172.16.0.0/12", " 127.0.0.1 ", "", "::1"})
	require.NoError(t, err)
	require.Len(t, prefixes, 3)
	assert.Equal(t, "172.16.0.0/12", prefixes[0].String())
	assert.Equal(t, "127.0.0.1/32", prefixes[1].String())
	assert.Equal(t, "::1/128", prefixes[2].String())

	_, err = ParseTrustedProxies([]string{"not-an-ip"})
	assert.Error(t, err)
}

func TestRealIP(t *testing.T) {
	trusted, err := ParseTrustedProxies([]string{"172.16.0.0/12", "10.0.0.1"})
	require.NoError(t, err)

	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "untrusted peer keeps remote addr",
			remoteAddr: "203.0.113.5:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.1", "X-Real-IP": "198.51.100.1"},
			want:       "203.0.113.5:1234",
		},
		{
			name:       "trusted peer uses rightmost untrusted hop",
			remoteAddr: "172.18.0.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "1.1.1.1, 198.51.100.7, 10.0.0.1"},
			want:       "198.51.100.7",
		},
		{
			name:       "trusted peer falls back to X-Real-IP",
			remoteAddr: "172.18.0.1:1234",
			headers:    map[string]string{"X-Real-IP": "198.51.100.8"},
			want:       "198.51.100.8",
		},
		{
			name:       "malformed hop keeps remote addr",
			remoteAddr: "172.18.0.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.1, garbage"},
			want:       "172.18.0.1:1234",
		},
		{
			name:       "all hops trusted keeps remote addr",
			remoteAddr: "172.18.0.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			want:       "172.18.0.1:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			handler := RealIP(trusted)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.RemoteAddr
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			handler.ServeHTTP(httptest.NewRecorder(), req)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("no trusted proxies ignores headers", func(t *testing.T) {
		var got string
		handler := RealIP(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r.RemoteAddr
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "172.18.0.1:1234"
		req.Header.Set("X-Forwarded-For", "198.51.100.1")
		handler.ServeHTTP(httptest.NewRecorder(), req)
		assert.Equal(t, "172.18.0.1:1234", got)
	})
}
