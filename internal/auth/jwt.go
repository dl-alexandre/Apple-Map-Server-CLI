package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims represents the standard JWT payload structure
type JWTClaims struct {
	// Standard claims
	Issuer    string `json:"iss,omitempty"`
	Subject   string `json:"sub,omitempty"`
	Audience  string `json:"aud,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	NotBefore int64  `json:"nbf,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	ID        string `json:"jti,omitempty"`

	// Apple Maps specific claims
	Origin string `json:"origin,omitempty"`
}

// JWTParser handles JWT token parsing without external dependencies
type JWTParser struct{}

// NewJWTParser creates a new JWT parser
func NewJWTParser() *JWTParser {
	return &JWTParser{}
}

// Parse parses a JWT token and returns the claims
// Note: This only parses the payload, it does NOT verify the signature
func (p *JWTParser) Parse(token string) (*JWTClaims, error) {
	// Split the token into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode the payload (second part)
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	// Parse JSON
	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parsing claims: %w", err)
	}

	return &claims, nil
}

// ParseWithExpiry is a convenience method that parses and returns just the expiry info
func (p *JWTParser) ParseWithExpiry(token string) (time.Time, error) {
	claims, err := p.Parse(token)
	if err != nil {
		return time.Time{}, err
	}

	if claims.ExpiresAt == 0 {
		return time.Time{}, fmt.Errorf("token has no expiration claim")
	}

	return time.Unix(claims.ExpiresAt, 0), nil
}

// IsExpired checks if the token is expired
func (p *JWTParser) IsExpired(token string) (bool, time.Time, error) {
	expiry, err := p.ParseWithExpiry(token)
	if err != nil {
		return false, time.Time{}, err
	}

	return time.Now().After(expiry), expiry, nil
}

// TimeUntilExpiry returns the duration until the token expires
// Returns negative duration if already expired
func (p *JWTParser) TimeUntilExpiry(token string) (time.Duration, time.Time, error) {
	expiry, err := p.ParseWithExpiry(token)
	if err != nil {
		return 0, time.Time{}, err
	}

	return time.Until(expiry), expiry, nil
}

// base64URLDecode decodes base64url encoded data (JWT specific encoding)
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if necessary
	// Base64 URL encoding may omit padding, but Go's decoder requires it
	padding := 4 - (len(s) % 4)
	if padding != 4 {
		s += strings.Repeat("=", padding)
	}

	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	return base64.StdEncoding.DecodeString(s)
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%d days, %d hours", days, hours)
		}
		return fmt.Sprintf("%d days", days)
	}

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
		}
		return fmt.Sprintf("%d hours", hours)
	}

	if minutes > 0 {
		return fmt.Sprintf("%d minutes", minutes)
	}

	return "less than a minute"
}
