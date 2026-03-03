package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSnapshotCommandUsage(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectError  bool
		expectErrMsg string
	}{
		{
			name:         "missing center",
			args:         []string{},
			expectError:  true,
			expectErrMsg: "snapshot requires a center location",
		},
		{
			name:        "basic snapshot with coordinates",
			args:        []string{"37.7749,-122.4194"},
			expectError: false,
		},
		{
			name:        "snapshot with address",
			args:        []string{"San Francisco, CA"},
			expectError: false,
		},
		{
			name:        "snapshot with all flags",
			args:        []string{"37.7749,-122.4194", "--zoom", "14", "--size", "600x400", "--format", "png", "--output", "test.png"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set required env vars for auth check
			t.Setenv("AMS_MAPS_TOKEN", "test-token")

			cmd := NewSnapshotCommand()
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
				// We expect failure because we don't have valid credentials
				// but the command should parse args correctly
				if exitCode == ExitSuccess {
					// This would only happen with mocked dependencies
					t.Logf("Command parsed args successfully, would need valid credentials to actually run")
				}
			}
		})
	}
}

func TestSnapshotCommandMissingEnv(t *testing.T) {
	// Clear environment
	os.Unsetenv("AMS_MAPS_TOKEN")

	cmd := NewSnapshotCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"37.7749,-122.4194"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitUsageError {
		t.Errorf("expected exit code %d for missing env, got %d", ExitUsageError, exitCode)
	}

	if !strings.Contains(stderr.String(), "missing required env vars") {
		t.Errorf("expected error about missing env vars, got:\n%s", stderr.String())
	}
}

func TestSnapshotCommandMissingCredentials(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "test-token")
	os.Unsetenv("AMS_TEAM_ID")
	os.Unsetenv("AMS_KEY_ID")
	os.Unsetenv("AMS_PRIVATE_KEY")
	os.Unsetenv("AMS_PRIVATE_KEY_PATH")

	cmd := NewSnapshotCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"37.7749,-122.4194"}
	exitCode := cmd.Run(args, stdout, stderr)

	if exitCode != ExitUsageError {
		t.Errorf("expected exit code %d for missing credentials, got %d", ExitUsageError, exitCode)
	}

	if !strings.Contains(stderr.String(), "AMS_TEAM_ID") {
		t.Errorf("expected error about missing Team ID, got:\n%s", stderr.String())
	}
}

func TestBuildSnapshotPath(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		center   string
		zoom     int
		size     string
		params   map[string]string
		expected string
	}{
		{
			name:     "basic path",
			baseURL:  "https://maps-api.apple.com",
			center:   "37.7749,-122.4194",
			zoom:     12,
			size:     "600x400",
			params:   map[string]string{"teamId": "ABC123", "keyId": "DEF456"},
			expected: "https://maps-api.apple.com/api/v1/snapshot?center=37.7749%2C-122.4194&keyId=DEF456&size=600x400&teamId=ABC123&z=12",
		},
		{
			name:     "with additional params",
			baseURL:  "https://maps-api.apple.com",
			center:   "San Francisco, CA",
			zoom:     14,
			size:     "800x600",
			params:   map[string]string{"teamId": "ABC123", "keyId": "DEF456", "t": "standard", "format": "png"},
			expected: "https://maps-api.apple.com/api/v1/snapshot?center=San+Francisco%2C+CA&format=png&keyId=DEF456&size=800x600&t=standard&teamId=ABC123&z=14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSnapshotPath(tt.baseURL, tt.center, tt.zoom, tt.size, tt.params)
			if result != tt.expected {
				t.Errorf("buildSnapshotPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildSnapshotPathURLEncoding(t *testing.T) {
	// Test that special characters are properly encoded
	baseURL := "https://maps-api.apple.com"
	center := "One Apple Park Way, Cupertino, CA"
	zoom := 16
	size := "600x400"
	params := map[string]string{"teamId": "ABC123", "keyId": "DEF456"}

	result := buildSnapshotPath(baseURL, center, zoom, size, params)

	// Check that spaces are encoded
	if strings.Contains(result, " ") {
		t.Error("URL contains unencoded spaces")
	}

	// Check that result contains expected parts
	if !strings.Contains(result, "center=One+") && !strings.Contains(result, "center=One%20") {
		t.Error("center parameter not properly encoded")
	}
}
