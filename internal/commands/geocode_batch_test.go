package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

func TestReadQueries(t *testing.T) {
	input := strings.NewReader("\n# comment\nfirst\n  second  \n#another\n\nthird\n")
	queries, err := readQueries(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 3 {
		t.Fatalf("expected 3 queries, got %d", len(queries))
	}
	if queries[0] != "first" || queries[1] != "second" || queries[2] != "third" {
		t.Fatalf("unexpected queries: %#v", queries)
	}
}

func TestGeocodeBatchJSONProgressToStderr(t *testing.T) {
	path := writeTempQueries(t, "one\n")
	setupGeocodeBatchStubs(t)
	progressEnabled = func(jsonOut bool, stdout, stderr io.Writer) bool {
		return true
	}
	t.Cleanup(func() {
		progressEnabled = defaultProgressEnabled
	})

	cmd := NewGeocodeCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--file", path, "--json"}, stdout, stderr)

	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}
	if stdout.Len() == 0 {
		t.Fatalf("expected JSONL output")
	}
	if strings.Contains(stdout.String(), "geocode") {
		t.Fatalf("did not expect progress output in stdout: %q", stdout.String())
	}
}

func TestGeocodeBatchConcurrencyInvalid(t *testing.T) {
	path := writeTempQueries(t, "one\n")
	setupGeocodeBatchStubs(t)

	cmd := NewGeocodeCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--file", path, "--concurrency", "0"}, stdout, stderr)

	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}
	if !strings.Contains(stderr.String(), "concurrency must be at least 1") {
		t.Fatalf("expected concurrency error, got %q", stderr.String())
	}
}

func TestGeocodeBatchOrderingWithConcurrency(t *testing.T) {
	path := writeTempQueries(t, "one\ntwo\nthree\n")
	setupGeocodeBatchStubs(t)

	geocodeRequest = func(client *httpclient.Client, token, query string) (int, []byte, error) {
		switch query {
		case "one":
			time.Sleep(30 * time.Millisecond)
		case "two":
			time.Sleep(10 * time.Millisecond)
		case "three":
			// no delay
		}
		return 200, []byte(`{"results":[]}`), nil
	}
	t.Cleanup(func() {
		geocodeRequest = doGeocodeRequest
	})

	cmd := NewGeocodeCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--file", path, "--json", "--concurrency", "2"}, stdout, stderr)

	if code != ExitSuccess {
		t.Fatalf("expected exit %d, got %d", ExitSuccess, code)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 json lines, got %d", len(lines))
	}

	for i, line := range lines {
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("invalid json line: %v", err)
		}
		input, ok := payload["input"].(string)
		if !ok {
			t.Fatalf("missing input field")
		}
		if input != []string{"one", "two", "three"}[i] {
			t.Fatalf("expected input %q at %d, got %q", []string{"one", "two", "three"}[i], i, input)
		}
	}
}

func TestGeocodeBatchMixedFailureExitCode(t *testing.T) {
	path := writeTempQueries(t, "one\ntwo\n")
	setupGeocodeBatchStubs(t)

	var call int32
	geocodeRequest = func(client *httpclient.Client, token, query string) (int, []byte, error) {
		current := atomic.AddInt32(&call, 1)
		if current == 1 {
			return 200, []byte(`{"results":[]}`), nil
		}
		return 500, []byte("oops"), nil
	}
	t.Cleanup(func() {
		geocodeRequest = doGeocodeRequest
	})

	cmd := NewGeocodeCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--file", path}, stdout, stderr)

	if code != ExitRuntimeError {
		t.Fatalf("expected exit %d, got %d", ExitRuntimeError, code)
	}
	if strings.Count(stdout.String(), "input:") != 2 {
		t.Fatalf("expected output for all inputs, got %q", stdout.String())
	}
}

func TestGeocodeBatchMissingFile(t *testing.T) {
	setupGeocodeBatchStubs(t)
	cmd := NewGeocodeCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"--file", filepath.Join(t.TempDir(), "missing.txt")}, stdout, stderr)

	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}
}

func setupGeocodeBatchStubs(t *testing.T) {
	t.Helper()
	t.Setenv("AMS_MAPS_TOKEN", "maps-token")

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "token", ExpiresIn: 3600, ExpiresAt: now.Add(time.Hour)}, auth.AccessTokenFetched, nil
	}
	newHTTPClient = func() (*httpclient.Client, error) {
		return &httpclient.Client{BaseURL: "https://example.com"}, nil
	}
	geocodeRequest = func(client *httpclient.Client, token, query string) (int, []byte, error) {
		return 200, []byte(`{"results":[]}`), nil
	}
	progressEnabled = func(jsonOut bool, stdout, stderr io.Writer) bool {
		return false
	}

	t.Cleanup(func() {
		accessTokenProvider = auth.GetAccessToken
		newHTTPClient = httpclient.New
		geocodeRequest = doGeocodeRequest
		progressEnabled = defaultProgressEnabled
	})
}

func writeTempQueries(t *testing.T, content string) string {
	file := filepath.Join(t.TempDir(), "queries.txt")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return file
}
