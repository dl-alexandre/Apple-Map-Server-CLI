package auth

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

func TestGetAccessTokenCaching(t *testing.T) {
	resetAccessTokenCache()

	var calls int32
	client := &httpclient.Client{
		BaseURL: "https://example.com",
		HTTP: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"token-1","expires_in":120}`)),
			}, nil
		})},
	}

	cfg := Config{MapsToken: "maps-token"}
	start := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)

	first, source, err := GetAccessToken(cfg, client, start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != AccessTokenFetched {
		t.Fatalf("expected fetched source, got %s", source)
	}

	second, source, err := GetAccessToken(cfg, client, start.Add(30*time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != AccessTokenCache {
		t.Fatalf("expected cache source, got %s", source)
	}
	if second.Value != first.Value {
		t.Fatalf("expected cached token")
	}

	third, source, err := GetAccessToken(cfg, client, start.Add(70*time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != AccessTokenFetched {
		t.Fatalf("expected refreshed token, got %s", source)
	}
	if third.Value == "" {
		t.Fatalf("expected token value")
	}

	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 token fetch calls, got %d", calls)
	}
}

func TestGetAccessTokenMissingMapsToken(t *testing.T) {
	resetAccessTokenCache()
	client := &httpclient.Client{BaseURL: "https://example.com"}

	_, _, err := GetAccessToken(Config{}, client, time.Now())
	if err == nil {
		t.Fatalf("expected missing env error")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
