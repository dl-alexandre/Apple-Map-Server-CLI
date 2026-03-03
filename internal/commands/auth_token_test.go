package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

func TestAuthTokenRawOutput(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "maps-token")

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "token-raw", ExpiresIn: 3600, ExpiresAt: now.Add(time.Hour)}, auth.AccessTokenFetched, nil
	}
	t.Cleanup(func() {
		accessTokenProvider = auth.GetAccessToken
	})

	cmd := NewAuthTokenCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--raw"}, stdout, stderr)

	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}
	// stderr should contain the token expiry warning
	if !strings.Contains(stderr.String(), "WARNING: Apple Maps Server API tokens expire every 7 days") {
		t.Fatalf("expected stderr to contain token expiry warning, got %q", stderr.String())
	}
	if stdout.String() != "token-raw\n" {
		t.Fatalf("expected raw token, got %q", stdout.String())
	}
}

func TestAuthTokenJSONOutput(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "maps-token")

	fixedTime := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "token-json", ExpiresIn: 1800, ExpiresAt: fixedTime}, auth.AccessTokenCache, nil
	}
	t.Cleanup(func() {
		accessTokenProvider = auth.GetAccessToken
	})

	cmd := NewAuthTokenCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--json"}, stdout, stderr)

	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}
	// stderr should contain the token expiry warning
	if !strings.Contains(stderr.String(), "WARNING: Apple Maps Server API tokens expire every 7 days") {
		t.Fatalf("expected stderr to contain token expiry warning, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &payload); err != nil {
		t.Fatalf("expected valid json, got error %v", err)
	}

	if payload["access_token"] != "token-json" {
		t.Fatalf("expected access_token field, got %v", payload["access_token"])
	}
	if payload["maps_token_present"] != true {
		t.Fatalf("expected maps_token_present true")
	}
	if payload["source"] != string(auth.AccessTokenCache) {
		t.Fatalf("expected source field, got %v", payload["source"])
	}
	if payload["expires_in"] == nil {
		t.Fatalf("expected expires_in field")
	}
	if payload["expires_at"] == nil {
		t.Fatalf("expected expires_at field")
	}
}
