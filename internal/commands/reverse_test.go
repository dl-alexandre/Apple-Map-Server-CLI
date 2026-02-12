package commands

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

func TestReverseParseErrors(t *testing.T) {
	setupReverseStubs(t)

	cmd := NewReverseCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"foo"}, stdout, stderr)
	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}

	stdout.Reset()
	stderr.Reset()
	code = cmd.Run([]string{"91,0"}, stdout, stderr)
	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}

	stdout.Reset()
	stderr.Reset()
	code = cmd.Run([]string{"0,181"}, stdout, stderr)
	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}
}

func TestReverseLimitValidation(t *testing.T) {
	setupReverseStubs(t)

	cmd := NewReverseCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--limit", "0", "0,0"}, stdout, stderr)
	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}
}

func TestReverseJSONPassthrough(t *testing.T) {
	setupReverseStubs(t)

	reverseRequest = func(client *httpclient.Client, token string, lat, lon float64) (int, []byte, error) {
		return 200, []byte(`{"results":[{"name":"One"}]}`), nil
	}
	t.Cleanup(func() {
		reverseRequest = doReverseRequest
	})

	cmd := NewReverseCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--json", "0,0"}, stdout, stderr)
	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}
	expected := "{\n  \"results\": [\n    {\n      \"name\": \"One\"\n    }\n  ]\n}"
	if strings.TrimSpace(stdout.String()) != expected {
		t.Fatalf("expected pretty json, got %q", stdout.String())
	}
}

func TestReverseHumanTruncates(t *testing.T) {
	setupReverseStubs(t)

	reverseRequest = func(client *httpclient.Client, token string, lat, lon float64) (int, []byte, error) {
		return 200, []byte(`{"results":[{"formattedAddress":"First","coordinate":{"latitude":1,"longitude":2}},{"formattedAddress":"Second","coordinate":{"latitude":3,"longitude":4}}]}`), nil
	}
	t.Cleanup(func() {
		reverseRequest = doReverseRequest
	})

	cmd := NewReverseCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--limit", "1", "0,0"}, stdout, stderr)
	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}
	output := stdout.String()
	if !strings.Contains(output, "First") {
		t.Fatalf("expected first result, got %q", output)
	}
	if strings.Contains(output, "Second") {
		t.Fatalf("expected truncated output, got %q", output)
	}
}

func setupReverseStubs(t *testing.T) {
	t.Helper()
	t.Setenv("AMS_MAPS_TOKEN", "maps-token")

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "token", ExpiresIn: 3600, ExpiresAt: now.Add(time.Hour)}, auth.AccessTokenFetched, nil
	}
	newHTTPClient = func() (*httpclient.Client, error) {
		return &httpclient.Client{BaseURL: "https://example.com"}, nil
	}

	t.Cleanup(func() {
		accessTokenProvider = auth.GetAccessToken
		newHTTPClient = httpclient.New
		reverseRequest = doReverseRequest
	})
}
