package httpclient

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://maps-api.apple.com"
const baseURLEnvVar = "AMS_BASE_URL"

type Client struct {
	BaseURL    string
	HTTP       *http.Client
	MaxRetries int
	RetryDelay time.Duration
}

func New() (*Client, error) {
	baseURL, err := resolveBaseURL()
	if err != nil {
		return nil, err
	}

	return &Client{
		BaseURL:    baseURL,
		HTTP:       &http.Client{Timeout: 15 * time.Second},
		MaxRetries: 2,
		RetryDelay: time.Second,
	}, nil
}

func resolveBaseURL() (string, error) {
	raw := strings.TrimSpace(os.Getenv(baseURLEnvVar))
	if raw == "" {
		return DefaultBaseURL, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid %s: %w", baseURLEnvVar, err)
	}

	if !parsed.IsAbs() || parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid %s: must be absolute https url", baseURLEnvVar)
	}

	normalized := strings.TrimRight(parsed.String(), "/")
	if normalized == "" {
		return "", fmt.Errorf("invalid %s: empty url", baseURLEnvVar)
	}

	return normalized, nil
}

func (c *Client) NewRequest(method, path string, query url.Values, body io.Reader) (*http.Request, error) {
	if c.BaseURL == "" {
		return nil, errors.New("base URL is empty")
	}

	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	cleanPath := strings.TrimPrefix(path, "/")
	basePath := strings.TrimSuffix(base.Path, "/")
	if strings.HasSuffix(basePath, "/v1") {
		cleanPath = strings.TrimPrefix(cleanPath, "v1/")
		if cleanPath == "v1" {
			cleanPath = ""
		}
	}
	rel, err := url.Parse(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("parse path: %w", err)
	}

	full := base.ResolveReference(rel)
	if query != nil {
		full.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(method, full.String(), body)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	return req, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	retries := c.MaxRetries
	if retries < 0 {
		retries = 0
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		resp, err := c.HTTP.Do(req)
		if err == nil && !shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("request failed with status %d", resp.StatusCode)
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		if attempt == retries {
			break
		}

		delay := c.retryDelay(resp)
		time.Sleep(delay)
	}

	return nil, lastErr
}

func shouldRetry(status int) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	return status >= 500 && status <= 599
}

func (c *Client) retryDelay(resp *http.Response) time.Duration {
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	if c.RetryDelay <= 0 {
		return time.Second
	}
	return c.RetryDelay
}

func RequestIDs(headers http.Header) []string {
	keys := []string{"X-Request-Id", "X-Request-ID", "X-Apple-Request-UUID", "X-Correlation-Id"}
	var values []string
	for _, key := range keys {
		if value := headers.Get(key); value != "" {
			values = append(values, value)
		}
	}
	return values
}
