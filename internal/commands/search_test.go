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

func TestSearchCommandUsage(t *testing.T) {
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
			expectErrMsg: "search requires a query",
		},
		{
			name:         "invalid near format",
			args:         []string{"--near", "invalid", "coffee"},
			expectError:  true,
			expectErrMsg: "error parsing --near",
		},
		{
			name:         "multiple geo flags",
			args:         []string{"--near", "37.77,-122.41", "--region", "37.8,-122.4,37.7,-122.5", "coffee"},
			expectError:  true,
			expectErrMsg: "cannot combine --near, --region, and --near-address",
		},
		{
			name:        "valid search with query",
			args:        []string{"coffee shops"},
			expectError: false,
		},
		{
			name:        "search with near coordinates",
			args:        []string{"--near", "37.7749,-122.4194", "coffee"},
			expectError: false,
		},
		{
			name:         "invalid region format",
			args:         []string{"--region", "invalid", "coffee"},
			expectError:  true,
			expectErrMsg: "error parsing --region",
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
			cmd := NewSearchCommand()
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// Mock the functions that would normally require API calls
			originalSearchRequest := searchRequest
			searchRequest = func(client *httpclient.Client, token string, query string, limit int, category string, centerLat, centerLng float64, hasCenter bool, bboxNorth, bboxEast, bboxSouth, bboxWest float64, hasBbox bool) (int, []byte, error) {
				return 200, []byte(`{"results":[]}`), nil
			}
			defer func() { searchRequest = originalSearchRequest }()

			originalAccessTokenProvider := accessTokenProvider
			accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
				return auth.Token{Value: "mock-token"}, auth.AccessTokenFetched, nil
			}
			defer func() { accessTokenProvider = originalAccessTokenProvider }()

			originalNowFunc := nowFunc
			nowFunc = func() time.Time {
				return time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
			}
			defer func() { nowFunc = originalNowFunc }()

			originalNewHTTPClient := newHTTPClient
			newHTTPClient = func() (*httpclient.Client, error) {
				return httpclient.New()
			}
			defer func() { newHTTPClient = originalNewHTTPClient }()

			t.Setenv("AMS_MAPS_TOKEN", "test-token")

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

func TestSearchCommandWithNearAddress(t *testing.T) {
	// Save and restore original functions
	originalSearchRequest := searchRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient
	originalGeocodeRequest := geocodeRequest

	defer func() {
		searchRequest = originalSearchRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
		geocodeRequest = originalGeocodeRequest
	}()

	// Track what was passed to search
	var capturedHasCenter bool
	var capturedLat, capturedLng float64

	searchRequest = func(client *httpclient.Client, token string, query string, limit int, category string, centerLat, centerLng float64, hasCenter bool, bboxNorth, bboxEast, bboxSouth, bboxWest float64, hasBbox bool) (int, []byte, error) {
		capturedHasCenter = hasCenter
		capturedLat = centerLat
		capturedLng = centerLng
		return 200, mustMarshalJSON(SearchResponse{Results: []SearchResult{{Name: "Philz Coffee", Coordinate: Coordinate{Latitude: 37.775, Longitude: -122.418}}}}), nil
	}

	geocodeRequest = func(client *httpclient.Client, token, query string) (int, []byte, error) {
		resp := map[string]any{
			"results": []any{
				map[string]any{
					"coordinate": map[string]any{
						"latitude":  37.7749,
						"longitude": -122.4194,
					},
				},
			},
		}
		body, _ := json.Marshal(resp)
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

	cmd := NewSearchCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"--near-address", "San Francisco, CA", "coffee"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
	}

	if !capturedHasCenter {
		t.Error("expected hasCenter to be true when using --near-address")
	}

	if capturedLat != 37.7749 || capturedLng != -122.4194 {
		t.Errorf("expected coordinates (37.7749, -122.4194), got (%.4f, %.4f)", capturedLat, capturedLng)
	}

	if !strings.Contains(stdout.String(), "Philz Coffee") {
		t.Errorf("expected output to contain 'Philz Coffee', got:\n%s", stdout.String())
	}
}

func TestDoSearchRequest(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		limit        int
		category     string
		centerLat    float64
		centerLng    float64
		hasCenter    bool
		bboxNorth    float64
		bboxEast     float64
		bboxSouth    float64
		bboxWest     float64
		hasBbox      bool
		expectParams map[string]string
	}{
		{
			name:         "basic query only",
			query:        "coffee",
			limit:        10,
			expectParams: map[string]string{"q": "coffee", "limit": "10"},
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
		{
			name:      "query with bounding box",
			query:     "restaurants",
			limit:     15,
			category:  "food",
			bboxNorth: 37.8,
			bboxEast:  -122.4,
			bboxSouth: 37.7,
			bboxWest:  -122.5,
			hasBbox:   true,
			expectParams: map[string]string{
				"q":                    "restaurants",
				"limit":                "15",
				"includePoiCategories": "food",
				"region":               "37.800000,-122.400000,37.700000,-122.500000",
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

			_, _, err := doSearchRequest(
				httpClient,
				"test-token",
				tt.query,
				tt.limit,
				tt.category,
				tt.centerLat,
				tt.centerLng,
				tt.hasCenter,
				tt.bboxNorth,
				tt.bboxEast,
				tt.bboxSouth,
				tt.bboxWest,
				tt.hasBbox,
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

func TestWriteSearchTable(t *testing.T) {
	tests := []struct {
		name         string
		response     SearchResponse
		limit        int
		hasCenter    bool
		centerLat    float64
		centerLng    float64
		expectOutput []string
		expectOK     bool
	}{
		{
			name: "basic results without center",
			response: SearchResponse{
				Results: []SearchResult{
					{
						Name:                  "Starbucks",
						FormattedAddressLines: []string{"123 Main St"},
						Coordinate:            Coordinate{Latitude: 37.7749, Longitude: -122.4194},
						PoiCategory:           "cafe",
					},
				},
			},
			limit:        10,
			hasCenter:    false,
			expectOutput: []string{"Starbucks", "cafe", "123 Main St", "37.774900,-122.419400"},
			expectOK:     true,
		},
		{
			name: "results with center shows distance",
			response: SearchResponse{
				Results: []SearchResult{
					{
						Name:                  "Blue Bottle",
						FormattedAddressLines: []string{"456 Market St"},
						Coordinate:            Coordinate{Latitude: 37.775, Longitude: -122.418},
						PoiCategory:           "cafe",
					},
				},
			},
			limit:        10,
			hasCenter:    true,
			centerLat:    37.7749,
			centerLng:    -122.4194,
			expectOutput: []string{"Blue Bottle", "cafe", "456 Market St"},
			expectOK:     true,
		},
		{
			name:         "empty results",
			response:     SearchResponse{Results: []SearchResult{}},
			limit:        10,
			expectOutput: []string{"no results"},
			expectOK:     true,
		},
		{
			name: "result missing name",
			response: SearchResponse{
				Results: []SearchResult{
					{
						FormattedAddressLines: []string{"Unknown Location"},
						Coordinate:            Coordinate{Latitude: 0, Longitude: 0},
					},
				},
			},
			limit:        10,
			expectOutput: []string{"-", "Unknown Location"},
			expectOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.response)
			var buf bytes.Buffer

			ok := writeSearchTable(&buf, body, tt.limit, tt.hasCenter, tt.centerLat, tt.centerLng)

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

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name      string
		lat1      float64
		lng1      float64
		lat2      float64
		lng2      float64
		expected  float64 // approximate in meters
		tolerance float64
	}{
		{
			name:      "same point",
			lat1:      37.7749,
			lng1:      -122.4194,
			lat2:      37.7749,
			lng2:      -122.4194,
			expected:  0,
			tolerance: 1,
		},
		{
			name:      "nearby points in SF",
			lat1:      37.7749,
			lng1:      -122.4194,
			lat2:      37.775,
			lng2:      -122.418,
			expected:  100, // approximately 100 meters
			tolerance: 50,
		},
		{
			name:      "SF to LA",
			lat1:      37.7749,
			lng1:      -122.4194,
			lat2:      34.0522,
			lng2:      -118.2437,
			expected:  559000, // approximately 559 km
			tolerance: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := haversineDistance(tt.lat1, tt.lng1, tt.lat2, tt.lng2)
			diff := distance - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("distance = %.0f, expected approximately %.0f (tolerance: %.0f)", distance, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestSearchCommandJSONOutput(t *testing.T) {
	// Save and restore original functions
	originalSearchRequest := searchRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		searchRequest = originalSearchRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	searchResponse := SearchResponse{
		Results: []SearchResult{
			{
				Name:                  "Starbucks",
				FormattedAddressLines: []string{"123 Main St, San Francisco, CA"},
				Coordinate:            Coordinate{Latitude: 37.7749, Longitude: -122.4194},
				PoiCategory:           "cafe",
			},
		},
	}

	searchRequest = func(client *httpclient.Client, token string, query string, limit int, category string, centerLat, centerLng float64, hasCenter bool, bboxNorth, bboxEast, bboxSouth, bboxWest float64, hasBbox bool) (int, []byte, error) {
		return 200, mustMarshalJSON(searchResponse), nil
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

	cmd := NewSearchCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"--json", "coffee"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
	}

	// Verify JSON output is valid and contains expected data
	var parsed SearchResponse
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Errorf("JSON output is not valid: %v\nOutput: %s", err, stdout.String())
	}

	if len(parsed.Results) != 1 || parsed.Results[0].Name != "Starbucks" {
		t.Errorf("JSON output does not contain expected data: %s", stdout.String())
	}
}

func TestSearchCommandCategoryFilter(t *testing.T) {
	// Save and restore original functions
	originalSearchRequest := searchRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		searchRequest = originalSearchRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	var capturedCategory string
	searchRequest = func(client *httpclient.Client, token string, query string, limit int, category string, centerLat, centerLng float64, hasCenter bool, bboxNorth, bboxEast, bboxSouth, bboxWest float64, hasBbox bool) (int, []byte, error) {
		capturedCategory = category
		return 200, mustMarshalJSON(SearchResponse{Results: []SearchResult{}}), nil
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

	cmd := NewSearchCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"--category", "restaurant", "italian"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d\nstderr: %s", ExitSuccess, exitCode, stderr.String())
	}

	if capturedCategory != "restaurant" {
		t.Errorf("expected category 'restaurant', got '%s'", capturedCategory)
	}
}

func TestSearchCommandAPIError(t *testing.T) {
	// Save and restore original functions
	originalSearchRequest := searchRequest
	originalAccessTokenProvider := accessTokenProvider
	originalNowFunc := nowFunc
	originalNewHTTPClient := newHTTPClient

	defer func() {
		searchRequest = originalSearchRequest
		accessTokenProvider = originalAccessTokenProvider
		nowFunc = originalNowFunc
		newHTTPClient = originalNewHTTPClient
	}()

	searchRequest = func(client *httpclient.Client, token string, query string, limit int, category string, centerLat, centerLng float64, hasCenter bool, bboxNorth, bboxEast, bboxSouth, bboxWest float64, hasBbox bool) (int, []byte, error) {
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

	cmd := NewSearchCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"coffee"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitRuntimeError {
		t.Errorf("expected exit code %d for API error, got %d", ExitRuntimeError, exitCode)
	}

	if !strings.Contains(stderr.String(), "network error") {
		t.Errorf("expected stderr to contain 'network error', got:\n%s", stderr.String())
	}
}

// Helper functions

func mustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

type mockRoundTripper struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}
