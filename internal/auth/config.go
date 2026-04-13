package auth

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type Config struct {
	// Legacy token-based auth (JWT from Apple Developer Portal)
	MapsToken string

	// New ES256-based auth (generate JWT from private key)
	TeamID         string
	PrivateKeyPath string
	KeyID          string
	Origin         string
}

// LoadConfigFromEnv loads configuration from environment variables
// Supports both legacy AMS_MAPS_TOKEN and new AMS_TEAM_ID + AMS_PRIVATE_KEY_PATH
func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		MapsToken:      os.Getenv("AMS_MAPS_TOKEN"),
		TeamID:         os.Getenv("AMS_TEAM_ID"),
		PrivateKeyPath: os.Getenv("AMS_PRIVATE_KEY_PATH"),
		KeyID:          os.Getenv("AMS_KEY_ID"),
		Origin:         os.Getenv("AMS_ORIGIN"),
	}

	// Check if we have the legacy token auth
	hasLegacyToken := strings.TrimSpace(cfg.MapsToken) != ""

	// Check if we have the new ES256 auth
	hasTeamID := strings.TrimSpace(cfg.TeamID) != ""
	hasPrivateKey := strings.TrimSpace(cfg.PrivateKeyPath) != ""

	// If we have neither, require at least the legacy token for backwards compatibility
	if !hasLegacyToken && (!hasTeamID || !hasPrivateKey) {
		var missing []string
		if !hasLegacyToken {
			// For new users, suggest the ES256 approach
			if !hasTeamID {
				missing = append(missing, "AMS_TEAM_ID (or AMS_MAPS_TOKEN for legacy)")
			}
			if !hasPrivateKey {
				missing = append(missing, "AMS_PRIVATE_KEY_PATH (or AMS_MAPS_TOKEN for legacy)")
			}
		}
		sort.Strings(missing)
		return Config{}, MissingEnvError{Missing: missing}
	}

	return cfg, nil
}

// IsES256Auth returns true if the config is set up for ES256 JWT generation
func (c Config) IsES256Auth() bool {
	return strings.TrimSpace(c.TeamID) != "" && strings.TrimSpace(c.PrivateKeyPath) != ""
}

// IsLegacyAuth returns true if the config is set up for legacy token auth
func (c Config) IsLegacyAuth() bool {
	return strings.TrimSpace(c.MapsToken) != ""
}

// GetMapsToken returns the Maps token, generating one if using ES256 auth
func (c Config) GetMapsToken() (string, error) {
	// If we have legacy token, use it
	if c.IsLegacyAuth() {
		return c.MapsToken, nil
	}

	// Otherwise, generate from private key
	if !c.IsES256Auth() {
		return "", errors.New("no authentication configured")
	}

	signer, err := NewES256Signer(c.PrivateKeyPath, c.TeamID)
	if err != nil {
		return "", fmt.Errorf("creating JWT signer: %w", err)
	}

	if c.KeyID != "" {
		signer.SetKeyID(c.KeyID)
	}
	if c.Origin != "" {
		signer.SetOrigin(c.Origin)
	}

	jwt, err := signer.GenerateJWT(timeNow())
	if err != nil {
		return "", fmt.Errorf("generating JWT: %w", err)
	}

	return jwt, nil
}

// timeNow is a variable for testing purposes
var timeNow = func() time.Time {
	return time.Now().UTC()
}

// MissingEnvError represents missing required environment variables
type MissingEnvError struct {
	Missing []string
}

func (err MissingEnvError) Error() string {
	if len(err.Missing) == 0 {
		return "missing required env vars"
	}
	return fmt.Sprintf("missing required env vars: %s", strings.Join(err.Missing, ", "))
}

// IsMissingEnv checks if an error is a MissingEnvError
func IsMissingEnv(err error) bool {
	var missing MissingEnvError
	return errors.As(err, &missing)
}
