package commands

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

func TestPingDefaultOutput(t *testing.T) {
	setupPingEnv(t)

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "token", ExpiresIn: 3600, ExpiresAt: now.Add(time.Hour)}, auth.AccessTokenFetched, nil
	}
	newHTTPClient = func() (*httpclient.Client, error) {
		return &httpclient.Client{
			BaseURL: "https://example.com",
			HTTP: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			})},
		}, nil
	}
	t.Cleanup(resetPingStubs)

	cmd := NewPingCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run(nil, stdout, stderr)

	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "auth ok") || !strings.Contains(output, "token ok") || !strings.Contains(output, "status 200") {
		t.Fatalf("unexpected output: %q", output)
	}
	if strings.Contains(output, "request_id") {
		t.Fatalf("did not expect request_id in output: %q", output)
	}
}

func TestPingRequestIDOutput(t *testing.T) {
	setupPingEnv(t)

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "token", ExpiresIn: 3600, ExpiresAt: now.Add(time.Hour)}, auth.AccessTokenFetched, nil
	}
	newHTTPClient = func() (*httpclient.Client, error) {
		return &httpclient.Client{
			BaseURL: "https://example.com",
			HTTP: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				headers := http.Header{}
				headers.Set("X-Request-Id", "abc-123")
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     headers,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			})},
		}, nil
	}
	t.Cleanup(resetPingStubs)

	cmd := NewPingCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--request-id"}, stdout, stderr)

	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "request_id abc-123") {
		t.Fatalf("expected request id output, got %q", output)
	}
}

func TestPingRequestIDMissingHeader(t *testing.T) {
	setupPingEnv(t)

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "token", ExpiresIn: 3600, ExpiresAt: now.Add(time.Hour)}, auth.AccessTokenFetched, nil
	}
	newHTTPClient = func() (*httpclient.Client, error) {
		return &httpclient.Client{
			BaseURL: "https://example.com",
			HTTP: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			})},
		}, nil
	}
	t.Cleanup(resetPingStubs)

	cmd := NewPingCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--request-id"}, stdout, stderr)

	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	output := stdout.String()
	if strings.Contains(output, "request_id") {
		t.Fatalf("did not expect request_id output, got %q", output)
	}
}

func setupPingEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AMS_MAPS_TOKEN", "maps-token")
}

func resetPingStubs() {
	accessTokenProvider = auth.GetAccessToken
	newHTTPClient = httpclient.New
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
