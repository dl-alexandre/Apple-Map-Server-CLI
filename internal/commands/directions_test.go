package commands

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

func TestDirectionsUsageError(t *testing.T) {
	cmd := NewDirectionsCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{}, stdout, stderr)

	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}
	if !strings.Contains(stderr.String(), "requires origin and destination") {
		t.Fatalf("expected origin/destination error, got %q", stderr.String())
	}
}

func TestDirectionsInvalidMode(t *testing.T) {
	cmd := NewDirectionsCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Flags must come before positional args
	code := cmd.Run([]string{"--mode", "invalid", "SF", "LA"}, stdout, stderr)

	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}
	if !strings.Contains(stderr.String(), "invalid transport mode") {
		t.Fatalf("expected invalid mode error, got %q", stderr.String())
	}
}

func TestNormalizeTransportMode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"car", "car"},
		{"CAR", "car"},
		{"automobile", "car"},
		{"driving", "car"},
		{"walk", "walk"},
		{"walking", "walk"},
		{"transit", "transit"},
		{"public", "transit"},
		{"bike", "bike"},
		{"bicycle", "bike"},
		{"cycling", "bike"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		result := normalizeTransportMode(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeTransportMode(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDirectionsTokenMissing(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "")

	cmd := NewDirectionsCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := cmd.Run([]string{"SF", "LA"}, stdout, stderr)

	if code != ExitUsageError {
		t.Fatalf("expected exit %d, got %d", ExitUsageError, code)
	}
	if !strings.Contains(stderr.String(), "missing required env vars") {
		t.Fatalf("expected missing env vars error, got %q", stderr.String())
	}
}

func TestDirectionsShowsTokenWarning(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "test-token")

	directionsRequest = func(client *httpclient.Client, token, origin, destination, mode string) (int, []byte, error) {
		return 200, []byte(`{"routes":[]}`), nil
	}
	t.Cleanup(func() {
		directionsRequest = doDirectionsRequest
	})

	accessTokenProvider = func(cfg auth.Config, client *httpclient.Client, now time.Time) (auth.Token, auth.AccessTokenSource, error) {
		return auth.Token{Value: "access-token"}, auth.AccessTokenFetched, nil
	}
	t.Cleanup(func() {
		accessTokenProvider = auth.GetAccessToken
	})

	cmd := NewDirectionsCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.Run([]string{"SF", "LA"}, stdout, stderr)

	if !strings.Contains(stderr.String(), "WARNING: Apple Maps Server API tokens expire every 7 days") {
		t.Fatalf("expected token expiry warning in stderr, got %q", stderr.String())
	}
}

func TestFormatDistance(t *testing.T) {
	tests := []struct {
		meters   float64
		expected string
	}{
		{500, "500 m"},
		{999, "999 m"},
		{1000, "1.0 km"},
		{1500, "1.5 km"},
		{9500, "9.5 km"},
		{10000, "10 km"},
		{25000, "25 km"},
	}

	for _, tt := range tests {
		result := formatDistance(tt.meters)
		if result != tt.expected {
			t.Errorf("formatDistance(%f) = %q, want %q", tt.meters, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{30, "0 min"},
		{60, "1 min"},
		{300, "5 min"},
		{600, "10 min"},
		{3600, "1 hr"},
		{3660, "1 hr 1 min"},
		{7200, "2 hr"},
		{7260, "2 hr 1 min"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatDuration(%f) = %q, want %q", tt.seconds, result, tt.expected)
		}
	}
}

func TestWriteETASummary(t *testing.T) {
	jsonData := `{
		"routes": [{
			"distanceMeters": 2033,
			"durationSeconds": 506,
			"transportType": "AUTOMOBILE",
			"hasTolls": true
		}]
	}`

	var buf bytes.Buffer
	ok := writeETASummary(&buf, []byte(jsonData))

	if !ok {
		t.Fatal("writeETASummary returned false")
	}

	output := buf.String()
	if !strings.Contains(output, "Distance:") {
		t.Errorf("expected Distance in output, got %q", output)
	}
	if !strings.Contains(output, "Duration:") {
		t.Errorf("expected Duration in output, got %q", output)
	}
	if !strings.Contains(output, "tolls") {
		t.Errorf("expected tolls note in output, got %q", output)
	}
}

func TestWriteETASummaryNoRoutes(t *testing.T) {
	jsonData := `{"routes":[]}`

	var buf bytes.Buffer
	ok := writeETASummary(&buf, []byte(jsonData))

	if !ok {
		t.Fatal("writeETASummary returned false")
	}

	output := buf.String()
	if !strings.Contains(output, "No routes found") {
		t.Errorf("expected 'No routes found' in output, got %q", output)
	}
}
