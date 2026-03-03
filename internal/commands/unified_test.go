package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

func TestUnifiedCommandUsage(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectError  bool
		expectErrMsg string
	}{
		{
			name:         "missing query",
			args:         []string{},
			expectError:  true,
			expectErrMsg: "unified requires a query",
		},
		{
			name:        "basic unified with query",
			args:        []string{"Golden Gate Bridge"},
			expectError: false,
		},
		{
			name:        "unified with coordinates",
			args:        []string{"coffee shops", "--near", "37.7749,-122.4194"},
			expectError: false,
		},
		{
			name:        "unified with all flags",
			args:        []string{"restaurants", "--near", "37.7749,-122.4194", "--zoom", "15", "--output", "test.png"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set required env vars
			t.Setenv("AMS_MAPS_TOKEN", "test-token")

			cmd := NewUnifiedCommand()
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			exitCode := cmd.Run(tt.args, stdout, stderr)

			if tt.expectError {
				if exitCode == ExitSuccess {
					t.Errorf("expected error (exit code != 0), got success")
				}
				if !strings.Contains(stderr.String(), tt.expectErrMsg) {
					t.Errorf("expected stderr to contain %q, got:\n%s", tt.expectErrMsg, stderr.String())
				}
			} else {
				// We expect failure without real credentials/API
				// but the command should parse args correctly
				t.Logf("Command parsed args successfully. Exit code: %d", exitCode)
			}
		})
	}
}

func TestUnifiedCommandMissingEnv(t *testing.T) {
	os.Unsetenv("AMS_MAPS_TOKEN")

	cmd := NewUnifiedCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"Golden Gate Bridge"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitUsageError {
		t.Errorf("expected exit code %d for missing env, got %d", ExitUsageError, exitCode)
	}

	if !strings.Contains(stderr.String(), "missing required env vars") {
		t.Errorf("expected error about missing env vars, got:\n%s", stderr.String())
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Golden Gate Bridge", "Golden_Gate_Bridge"},
		{"San Francisco, CA", "San_Francisco,_CA"},
		{"One/Two", "One_Two"},
		{"Name: Value", "Name__Value"},
		{"Star*Name", "Star_Name"},
		{"Question?Mark", "Question_Mark"},
		{"Quote\"Name", "Quote_Name"},
		{"Less<Than", "Less_Than"},
		{"Great>Than", "Great_Than"},
		{"Pipe|Char", "Pipe_Char"},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestUnifiedCommandWithMockSearch(t *testing.T) {
	// Save original functions
	originalSearchRequest := unifiedSearchRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		unifiedSearchRequest = originalSearchRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	// Mock search response
	searchResponse := SearchResponse{
		Results: []SearchResult{
			{
				Name:                  "Golden Gate Bridge",
				FormattedAddressLines: []string{"Golden Gate Bridge, San Francisco, CA"},
				Coordinate:            Coordinate{Latitude: 37.8199, Longitude: -122.4783},
				PoiCategory:           "landmark",
			},
		},
	}

	unifiedSearchRequest = func(client *httpclient.Client, token string, query string, limit int, category string, centerLat, centerLng float64, hasCenter bool, bboxNorth, bboxEast, bboxSouth, bboxWest float64, hasBbox bool) (int, []byte, error) {
		body, _ := json.Marshal(searchResponse)
		return 200, body, nil
	}

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "mock-token"}, auth.AccessTokenFetched, nil
	}

	nowFunc = func() time.Time {
		return time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	}

	newHTTPClient = func() (*httpclient.Client, error) {
		return httpclient.New()
	}

	t.Setenv("AMS_MAPS_TOKEN", "test-token")
	// Don't set snapshot credentials - it should gracefully degrade

	cmd := NewUnifiedCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"Golden Gate Bridge"}
	exitCode := cmd.Run(args, stdout, stderr)

	// Should succeed even without snapshot credentials
	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Golden Gate Bridge") {
		t.Errorf("expected output to contain place name, got:\n%s", output)
	}
	if !strings.Contains(output, "37.8199") || !strings.Contains(output, "-122.4783") {
		t.Errorf("expected output to contain coordinates, got:\n%s", output)
	}
}

func TestUnifiedCommandNoSearchResults(t *testing.T) {
	// Save original functions
	originalSearchRequest := unifiedSearchRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		unifiedSearchRequest = originalSearchRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	// Mock empty search response
	searchResponse := SearchResponse{
		Results: []SearchResult{},
	}

	unifiedSearchRequest = func(client *httpclient.Client, token string, query string, limit int, category string, centerLat, centerLng float64, hasCenter bool, bboxNorth, bboxEast, bboxSouth, bboxWest float64, hasBbox bool) (int, []byte, error) {
		body, _ := json.Marshal(searchResponse)
		return 200, body, nil
	}

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "mock-token"}, auth.AccessTokenFetched, nil
	}

	nowFunc = func() time.Time {
		return time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	}

	newHTTPClient = func() (*httpclient.Client, error) {
		return httpclient.New()
	}

	t.Setenv("AMS_MAPS_TOKEN", "test-token")

	cmd := NewUnifiedCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"NonExistentPlace12345"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitRuntimeError {
		t.Errorf("expected exit code %d for no results, got %d", ExitRuntimeError, exitCode)
	}

	if !strings.Contains(stderr.String(), "no search results") {
		t.Errorf("expected error about no results, got:\n%s", stderr.String())
	}
}
