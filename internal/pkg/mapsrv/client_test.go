package mapsrv

import (
	"os"
	"testing"
)

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
			name:     "basic path with coordinates",
			baseURL:  "https://maps-api.apple.com",
			center:   "37.7749,-122.4194",
			zoom:     12,
			size:     "600x400",
			params:   map[string]string{"teamId": "ABC123", "keyId": "DEF456"},
			expected: "https://maps-api.apple.com/api/v1/snapshot?center=37.7749%2C-122.4194&keyId=DEF456&size=600x400&teamId=ABC123&z=12",
		},
		{
			name:     "path with address",
			baseURL:  "https://maps-api.apple.com",
			center:   "San Francisco, CA",
			zoom:     14,
			size:     "800x600",
			params:   map[string]string{"teamId": "ABC123", "keyId": "DEF456", "t": "standard"},
			expected: "https://maps-api.apple.com/api/v1/snapshot?center=San+Francisco%2C+CA&keyId=DEF456&size=800x600&t=standard&teamId=ABC123&z=14",
		},
		{
			name:     "path with jpeg format",
			baseURL:  "https://maps-api.apple.com",
			center:   "London, UK",
			zoom:     10,
			size:     "1200x800",
			params:   map[string]string{"teamId": "ABC123", "keyId": "DEF456", "format": "jpg"},
			expected: "https://maps-api.apple.com/api/v1/snapshot?center=London%2C+UK&format=jpg&keyId=DEF456&size=1200x800&teamId=ABC123&z=10",
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

func TestSnapshotParamsDefaults(t *testing.T) {
	params := SnapshotParams{
		Center: "37.7749,-122.4194",
		Zoom:   12,
		Size:   "600x400",
		Format: "png",
	}

	if params.MapType != "" {
		t.Error("MapType should default to empty string")
	}
}

func TestClientCreation(t *testing.T) {
	client := NewClient("https://maps-api.apple.com", "test-token")

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.BaseURL != "https://maps-api.apple.com" {
		t.Errorf("BaseURL = %q, want %q", client.BaseURL, "https://maps-api.apple.com")
	}

	if client.Token != "test-token" {
		t.Errorf("Token = %q, want %q", client.Token, "test-token")
	}

	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestClientWithoutSnapshotAuth(t *testing.T) {
	client := NewClient("https://maps-api.apple.com", "test-token")

	params := SnapshotParams{
		Center: "37.7749,-122.4194",
		Zoom:   12,
		Size:   "600x400",
	}

	_, err := client.DownloadSnapshot(params)
	if err == nil {
		t.Error("Expected error when downloading snapshot without auth")
	}

	if err.Error() != "snapshot authentication not configured" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestWriteFile(t *testing.T) {
	// Test writing to temp directory
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test-snapshot.png"
	testData := []byte("test image data")

	err := writeFile(testFile, testData)
	if err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}

	// Verify file was written
	readData, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(readData) != string(testData) {
		t.Error("Written data doesn't match original")
	}
}
