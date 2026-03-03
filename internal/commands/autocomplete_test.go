package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

func TestAutocompleteCommandUsage(t *testing.T) {
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
			expectErrMsg: "autocomplete requires a query",
		},
		{
			name:         "invalid near format",
			args:         []string{"--near", "invalid", "starbu"},
			expectError:  true,
			expectErrMsg: "error parsing --near",
		},
		{
			name:        "valid autocomplete with query",
			args:        []string{"starbu"},
			expectError: false,
		},
		{
			name:        "autocomplete with near coordinates",
			args:        []string{"--near", "37.7749,-122.4194", "coffee"},
			expectError: false,
		},
		{
			name:         "limit too low",
			args:         []string{"--limit", "0", "coffee"},
			expectError:  true,
			expectErrMsg: "limit must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original functions
			originalAutocompleteRequest := autocompleteRequest
			originalAccessTokenProvider := accessTokenProvider
			originalNowFunc := nowFunc
			originalNewHTTPClient := newHTTPClient

			defer func() {
				autocompleteRequest = originalAutocompleteRequest
				accessTokenProvider = originalAccessTokenProvider
				nowFunc = originalNowFunc
				newHTTPClient = originalNewHTTPClient
			}()

			// Mock autocomplete request
			autocompleteRequest = func(client *httpclient.Client, token string, query string, limit int, centerLat, centerLng float64, hasCenter bool) (int, []byte, error) {
				return 200, []byte(`{"results":[]}`), nil
			}

			// Mock access token provider
			accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
				return auth.Token{Value: "mock-token"}, auth.AccessTokenFetched, nil
			}

			// Mock time
			nowFunc = func() time.Time {
				return time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
			}

			// Mock HTTP client
			newHTTPClient = func() (*httpclient.Client, error) {
				return httpclient.New()
			}

			t.Setenv("AMS_MAPS_TOKEN", "test-token")

			cmd := NewAutocompleteCommand()
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
				if exitCode != ExitSuccess {
					t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
				}
			}
		})
	}
}

func TestAutocompleteCommandBasic(t *testing.T) {
	// Save and restore original functions
	originalAutocompleteRequest := autocompleteRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		autocompleteRequest = originalAutocompleteRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	autocompleteRequest = func(client *httpclient.Client, token string, query string, limit int, centerLat, centerLng float64, hasCenter bool) (int, []byte, error) {
		response := AutocompleteResponse{
			Results: []AutocompleteResult{
				{
					DisplayLines:  []string{"Starbucks", "123 Main St, San Francisco, CA"},
					CompletionURL: "/v1/search/completion/abc123",
				},
				{
					DisplayLines:  []string{"Starbucks Reserve", "456 Market St, San Francisco, CA"},
					CompletionURL: "/v1/search/completion/def456",
				},
			},
		}
		return 200, mustMarshalJSON(response), nil
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

	cmd := NewAutocompleteCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"starbu"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Starbucks") {
		t.Errorf("expected output to contain 'Starbucks', got:\n%s", output)
	}
	if !strings.Contains(output, "123 Main St") {
		t.Errorf("expected output to contain '123 Main St', got:\n%s", output)
	}
}

func TestAutocompleteCommandJSONOutput(t *testing.T) {
	// Save and restore original functions
	originalAutocompleteRequest := autocompleteRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		autocompleteRequest = originalAutocompleteRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	response := AutocompleteResponse{
		Results: []AutocompleteResult{
			{
				DisplayLines:  []string{"Starbucks", "123 Main St"},
				CompletionURL: "/v1/search/completion/abc123",
			},
		},
	}

	autocompleteRequest = func(client *httpclient.Client, token string, query string, limit int, centerLat, centerLng float64, hasCenter bool) (int, []byte, error) {
		return 200, mustMarshalJSON(response), nil
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

	cmd := NewAutocompleteCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"--json", "starbu"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
	}

	// Verify JSON output is valid and contains expected data
	var parsed AutocompleteResponse
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Errorf("JSON output is not valid: %v\nOutput: %s", err, stdout.String())
	}

	if len(parsed.Results) != 1 || !strings.Contains(parsed.Results[0].DisplayLines[0], "Starbucks") {
		t.Errorf("JSON output does not contain expected data: %s", stdout.String())
	}

	if parsed.Results[0].CompletionURL != "/v1/search/completion/abc123" {
		t.Errorf("expected CompletionURL '/v1/search/completion/abc123', got '%s'", parsed.Results[0].CompletionURL)
	}
}

func TestDoAutocompleteRequest(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		limit        int
		centerLat    float64
		centerLng    float64
		hasCenter    bool
		expectParams map[string]string
	}{
		{
			name:         "basic query only",
			query:        "starbu",
			limit:        10,
			expectParams: map[string]string{"q": "starbu", "limit": "10"},
		},
		{
			name:      "query with center",
			query:     "coffee",
			limit:     20,
			centerLat: 37.7749,
			centerLng: -122.4194,
			hasCenter: true,
			expectParams: map[string]string{
				"q":      "coffee",
				"limit":  "20",
				"center": "37.774900,-122.419400",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client that captures the request
			var capturedReq *http.Request
			mockTransport := &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					capturedReq = req
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"results":[]}`)),
						Header:     make(http.Header),
					}, nil
				},
			}

			client := &http.Client{Transport: mockTransport}
			httpClient := &httpclient.Client{
				BaseURL:    "https://maps-api.apple.com",
				HTTP:       client,
				MaxRetries: 0,
			}

			_, _, err := doAutocompleteRequest(
				httpClient,
				"test-token",
				tt.query,
				tt.limit,
				tt.centerLat,
				tt.centerLng,
				tt.hasCenter,
			)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedReq == nil {
				t.Fatal("expected request to be captured")
			}

			// Parse query parameters
			query := capturedReq.URL.Query()
			for key, expectedValue := range tt.expectParams {
				actualValue := query.Get(key)
				if actualValue != expectedValue {
					t.Errorf("parameter %s: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestWriteAutocompleteTable(t *testing.T) {
	tests := []struct {
		name         string
		response     AutocompleteResponse
		limit        int
		expectOutput []string
		expectOK     bool
	}{
		{
			name: "basic suggestions",
			response: AutocompleteResponse{
				Results: []AutocompleteResult{
					{
						DisplayLines:  []string{"Starbucks", "123 Main St"},
						CompletionURL: "/v1/search/completion/abc123",
					},
					{
						DisplayLines:  []string{"Starbucks Reserve"},
						CompletionURL: "/v1/search/completion/def456",
					},
				},
			},
			limit:        10,
			expectOutput: []string{"Starbucks", "123 Main St", "/v1/search/completion/abc123"},
			expectOK:     true,
		},
		{
			name:         "empty results",
			response:     AutocompleteResponse{Results: []AutocompleteResult{}},
			limit:        10,
			expectOutput: []string{"no suggestions"},
			expectOK:     true,
		},
		{
			name: "single display line",
			response: AutocompleteResponse{
				Results: []AutocompleteResult{
					{
						DisplayLines:  []string{"Coffee Shop"},
						CompletionURL: "/v1/search/completion/xyz789",
					},
				},
			},
			limit:        10,
			expectOutput: []string{"Coffee Shop"},
			expectOK:     true,
		},
		{
			name: "missing completion URL",
			response: AutocompleteResponse{
				Results: []AutocompleteResult{
					{
						DisplayLines: []string{"Cafe"},
					},
				},
			},
			limit:        10,
			expectOutput: []string{"Cafe", "-"},
			expectOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.response)
			var buf bytes.Buffer

			ok := writeAutocompleteTable(&buf, body, tt.limit)

			if ok != tt.expectOK {
				t.Errorf("expected ok=%v, got %v", tt.expectOK, ok)
			}

			output := buf.String()
			for _, expected := range tt.expectOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestAutocompleteCommandWithNear(t *testing.T) {
	// Save and restore original functions
	originalAutocompleteRequest := autocompleteRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		autocompleteRequest = originalAutocompleteRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	var capturedHasCenter bool
	var capturedLat, capturedLng float64

	autocompleteRequest = func(client *httpclient.Client, token string, query string, limit int, centerLat, centerLng float64, hasCenter bool) (int, []byte, error) {
		capturedHasCenter = hasCenter
		capturedLat = centerLat
		capturedLng = centerLng
		return 200, mustMarshalJSON(AutocompleteResponse{Results: []AutocompleteResult{}}), nil
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

	cmd := NewAutocompleteCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"--near", "37.7749,-122.4194", "coffee"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
	}

	if !capturedHasCenter {
		t.Error("expected hasCenter to be true when using --near")
	}

	if capturedLat != 37.7749 || capturedLng != -122.4194 {
		t.Errorf("expected coordinates (37.7749, -122.4194), got (%.4f, %.4f)", capturedLat, capturedLng)
	}
}

func TestAutocompleteCommandAPIError(t *testing.T) {
	// Save and restore original functions
	originalAutocompleteRequest := autocompleteRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		autocompleteRequest = originalAutocompleteRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	autocompleteRequest = func(client *httpclient.Client, token string, query string, limit int, centerLat, centerLng float64, hasCenter bool) (int, []byte, error) {
		return 0, nil, errors.New("network error")
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

	cmd := NewAutocompleteCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"starbu"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitRuntimeError {
		t.Errorf("expected exit code %d for API error, got %d", ExitRuntimeError, exitCode)
	}

	if !strings.Contains(stderr.String(), "network error") {
		t.Errorf("expected stderr to contain 'network error', got:\n%s", stderr.String())
	}
}

func TestSearchSubcommandRouting(t *testing.T) {
	// Test that "ams search autocomplete ..." routes to the autocomplete command
	originalAutocompleteRequest := autocompleteRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		autocompleteRequest = originalAutocompleteRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	var autocompleteCalled bool
	autocompleteRequest = func(client *httpclient.Client, token string, query string, limit int, centerLat, centerLng float64, hasCenter bool) (int, []byte, error) {
		autocompleteCalled = true
		return 200, mustMarshalJSON(AutocompleteResponse{
			Results: []AutocompleteResult{
				{DisplayLines: []string{"Test Result"}},
			},
		}), nil
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

	// Use the main search command but pass "autocomplete" as first arg
	searchCmd := NewSearchCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"autocomplete", "test"}
	exitCode := searchCmd.Run(args, stdout, stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
	}

	if !autocompleteCalled {
		t.Error("expected autocomplete command to be called, but it wasn't")
	}

	if !strings.Contains(stdout.String(), "Test Result") {
		t.Errorf("expected output to contain 'Test Result', got:\n%s", stdout.String())
	}
}
