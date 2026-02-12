package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

type Token struct {
	Value     string
	ExpiresIn int64
	ExpiresAt time.Time
}

type AccessTokenSource string

const (
	AccessTokenFetched AccessTokenSource = "fetched"
	AccessTokenCache   AccessTokenSource = "cache"
)

const refreshSkew = 60 * time.Second

var accessTokenCache struct {
	sync.Mutex
	token Token
}

func resetAccessTokenCache() {
	accessTokenCache.Lock()
	accessTokenCache.token = Token{}
	accessTokenCache.Unlock()
}

func GetAccessToken(cfg Config, client *httpclient.Client, now time.Time) (Token, AccessTokenSource, error) {
	if strings.TrimSpace(cfg.MapsToken) == "" {
		return Token{}, "", MissingEnvError{Missing: []string{"AMS_MAPS_TOKEN"}}
	}

	if client == nil {
		var err error
		client, err = httpclient.New()
		if err != nil {
			return Token{}, "", err
		}
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	accessTokenCache.Lock()
	if accessTokenCache.token.Value != "" {
		remaining := accessTokenCache.token.ExpiresAt.Sub(now)
		if accessTokenCache.token.ExpiresAt.IsZero() || remaining > refreshSkew {
			token := accessTokenCache.token
			accessTokenCache.Unlock()
			return token, AccessTokenCache, nil
		}
	}
	accessTokenCache.Unlock()

	token, err := exchangeMapsToken(cfg, client, now)
	if err != nil {
		return Token{}, "", err
	}

	accessTokenCache.Lock()
	accessTokenCache.token = token
	accessTokenCache.Unlock()

	return token, AccessTokenFetched, nil
}

func exchangeMapsToken(cfg Config, client *httpclient.Client, now time.Time) (Token, error) {
	req, err := client.NewRequest(http.MethodPost, "/v1/token", nil, nil)
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.MapsToken)

	resp, err := client.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Token{}, fmt.Errorf("read token response: %w", err)
	}
	if debugEnabled() {
		writeTokenDebug(resp.StatusCode, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Token{}, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	token, err := parseTokenResponse(body, now)
	if err != nil {
		return Token{}, err
	}

	if token.Value == "" {
		return Token{}, errors.New("token response missing access token")
	}

	return token, nil
}

func debugEnabled() bool {
	return strings.TrimSpace(os.Getenv("AMS_DEBUG")) == "1"
}

func writeTokenDebug(status int, body []byte) {
	redacted := redactTokenJSON(body)
	fmt.Fprintf(os.Stderr, "debug: token response status=%d bytes=%d\n", status, len(body))
	if redacted != "" {
		fmt.Fprintf(os.Stderr, "debug: token response body=%s\n", redacted)
	}
}

func redactTokenJSON(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	if payload["access_token"] != nil {
		payload["access_token"] = "REDACTED"
	}
	if payload["accessToken"] != nil {
		payload["accessToken"] = "REDACTED"
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}

type tokenResponse struct {
	AccessToken    string `json:"access_token"`
	AccessTokenAlt string `json:"accessToken"`
	ExpiresIn      int64  `json:"expires_in"`
	ExpiresInAlt   int64  `json:"expiresIn"`
	ExpiresAt      string `json:"expires_at"`
	ExpiresAtAlt   string `json:"expiresAt"`
}

func parseTokenResponse(data []byte, now time.Time) (Token, error) {
	var resp tokenResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return Token{}, fmt.Errorf("parse token response: %w", err)
	}

	token := resp.AccessToken
	if token == "" {
		token = resp.AccessTokenAlt
	}

	expiresIn := resp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = resp.ExpiresInAlt
	}

	var expiresAt time.Time
	if resp.ExpiresAt != "" {
		if parsed, err := time.Parse(time.RFC3339, resp.ExpiresAt); err == nil {
			expiresAt = parsed
		}
	}
	if expiresAt.IsZero() && resp.ExpiresAtAlt != "" {
		if parsed, err := time.Parse(time.RFC3339, resp.ExpiresAtAlt); err == nil {
			expiresAt = parsed
		}
	}

	if expiresAt.IsZero() && expiresIn > 0 {
		if now.IsZero() {
			now = time.Now().UTC()
		}
		expiresAt = now.Add(time.Duration(expiresIn) * time.Second)
	}

	return Token{Value: token, ExpiresIn: expiresIn, ExpiresAt: expiresAt}, nil
}
