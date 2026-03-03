package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultZoom != 14 {
		t.Errorf("DefaultZoom = %d, want 14", cfg.DefaultZoom)
	}
	if cfg.DefaultSize != "800x600" {
		t.Errorf("DefaultSize = %q, want 800x600", cfg.DefaultSize)
	}
	if cfg.DefaultFormat != "png" {
		t.Errorf("DefaultFormat = %q, want png", cfg.DefaultFormat)
	}
	if cfg.DefaultMapType != "standard" {
		t.Errorf("DefaultMapType = %q, want standard", cfg.DefaultMapType)
	}
	if cfg.DefaultLimit != 10 {
		t.Errorf("DefaultLimit = %d, want 10", cfg.DefaultLimit)
	}
	if cfg.BaseURL != "https://maps-api.apple.com" {
		t.Errorf("BaseURL = %q, want https://maps-api.apple.com", cfg.BaseURL)
	}
}

func TestConfigSetAndGet(t *testing.T) {
	cfg := DefaultConfig()

	// Test setting and getting various types
	tests := []struct {
		key   string
		value string
	}{
		{"team_id", "ABC123"},
		{"key_id", "DEF456"},
		{"default_zoom", "18"},
		{"default_size", "1200x800"},
		{"default_format", "jpg"},
		{"default_category", "restaurant"},
		{"base_url", "https://example.com"},
	}

	for _, tt := range tests {
		if err := cfg.Set(tt.key, tt.value); err != nil {
			t.Errorf("Set(%q, %q) error = %v", tt.key, tt.value, err)
			continue
		}

		got, err := cfg.Get(tt.key)
		if err != nil {
			t.Errorf("Get(%q) error = %v", tt.key, err)
			continue
		}

		if got != tt.value {
			t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.value)
		}
	}
}

func TestConfigSetUnknownKey(t *testing.T) {
	cfg := DefaultConfig()

	err := cfg.Set("unknown_key", "value")
	if err == nil {
		t.Error("Set() with unknown key should return error")
	}
}

func TestConfigGetUnknownKey(t *testing.T) {
	cfg := DefaultConfig()

	_, err := cfg.Get("unknown_key")
	if err == nil {
		t.Error("Get() with unknown key should return error")
	}
}

func TestConfigSetInvalidZoom(t *testing.T) {
	cfg := DefaultConfig()

	err := cfg.Set("default_zoom", "not_a_number")
	if err == nil {
		t.Error("Set() with invalid zoom should return error")
	}
}

func TestConfigSetInvalidLimit(t *testing.T) {
	cfg := DefaultConfig()

	err := cfg.Set("default_limit", "not_a_number")
	if err == nil {
		t.Error("Set() with invalid limit should return error")
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	cfg := DefaultConfig()

	// Set environment variables
	os.Setenv("AMS_TEAM_ID", "ENV_TEAM_ID")
	os.Setenv("AMS_KEY_ID", "ENV_KEY_ID")
	os.Setenv("AMS_MAPS_TOKEN", "ENV_TOKEN")
	os.Setenv("AMS_BASE_URL", "https://env.example.com")

	// Reset after test
	defer func() {
		os.Unsetenv("AMS_TEAM_ID")
		os.Unsetenv("AMS_KEY_ID")
		os.Unsetenv("AMS_MAPS_TOKEN")
		os.Unsetenv("AMS_BASE_URL")
	}()

	cfg.loadFromEnv()

	if cfg.TeamID != "ENV_TEAM_ID" {
		t.Errorf("TeamID = %q, want ENV_TEAM_ID", cfg.TeamID)
	}
	if cfg.KeyID != "ENV_KEY_ID" {
		t.Errorf("KeyID = %q, want ENV_KEY_ID", cfg.KeyID)
	}
	if cfg.MapsToken != "ENV_TOKEN" {
		t.Errorf("MapsToken = %q, want ENV_TOKEN", cfg.MapsToken)
	}
	if cfg.BaseURL != "https://env.example.com" {
		t.Errorf("BaseURL = %q, want https://env.example.com", cfg.BaseURL)
	}
}

func TestConfigFilePath(t *testing.T) {
	path, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath() error = %v", err)
	}

	if path == "" {
		t.Error("ConfigFilePath() returned empty string")
	}

	// Should contain "ams" and "config.yaml"
	if !contains(path, "ams") {
		t.Error("ConfigFilePath() should contain 'ams'")
	}
	if !contains(path, "config.yaml") {
		t.Error("ConfigFilePath() should contain 'config.yaml'")
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}

	if dir == "" {
		t.Error("ConfigDir() returned empty string")
	}

	if !contains(dir, "ams") {
		t.Error("ConfigDir() should contain 'ams'")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()

	// Override config path for this test
	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if origConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", origConfigDir)
		}
	}()

	// Create and save config
	cfg := DefaultConfig()
	cfg.TeamID = "TEST_TEAM"
	cfg.KeyID = "TEST_KEY"
	cfg.DefaultZoom = 20

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	path, _ := ConfigFilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load and verify
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.TeamID != "TEST_TEAM" {
		t.Errorf("Loaded TeamID = %q, want TEST_TEAM", loaded.TeamID)
	}
	if loaded.KeyID != "TEST_KEY" {
		t.Errorf("Loaded KeyID = %q, want TEST_KEY", loaded.KeyID)
	}
	if loaded.DefaultZoom != 20 {
		t.Errorf("Loaded DefaultZoom = %d, want 20", loaded.DefaultZoom)
	}
}

func TestConfigList(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TeamID = "TEST_TEAM"
	cfg.DefaultZoom = 15

	list := cfg.List()

	// Check that all keys are present
	requiredKeys := []string{
		"maps_token", "team_id", "key_id", "private_key", "private_key_path",
		"default_zoom", "default_size", "default_format", "default_map_type",
		"default_limit", "default_category", "base_url",
	}

	for _, key := range requiredKeys {
		if _, ok := list[key]; !ok {
			t.Errorf("List() missing key: %s", key)
		}
	}

	// Check values
	if list["team_id"] != "TEST_TEAM" {
		t.Errorf("List()[team_id] = %q, want TEST_TEAM", list["team_id"])
	}
	if list["default_zoom"] != "15" {
		t.Errorf("List()[default_zoom] = %q, want 15", list["default_zoom"])
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
