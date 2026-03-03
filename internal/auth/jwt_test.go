package auth

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestJWTParser_Parse(t *testing.T) {
	// Create a test JWT token
	// Header: {"alg":"ES256","kid":"ABC123","typ":"JWT"}
	// Payload: {"iss":"DEF456GHIJ","iat":1437179036,"exp":1493298100,"origin":"*.example.com"}
	// Note: This is NOT a valid signed token, just for parsing tests
	testToken := "eyJhbGciOiJFUzI1NiIsImtpZCI6IkFCQzEyMyIsInR5cCI6IkpXVCJ9." +
		"eyJpc3MiOiJERUY0NTZHSElKIiwiaWF0IjoxNDM3MTc5MDM2LCJleHAiOjE0OTMyOTgxMDAsIm9yaWdpbiI6IiouZXhhbXBsZS5jb20ifQ." +
		"fake_signature_not_used"

	parser := NewJWTParser()
	claims, err := parser.Parse(testToken)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if claims.Issuer != "DEF456GHIJ" {
		t.Errorf("Issuer = %q, want DEF456GHIJ", claims.Issuer)
	}

	if claims.ExpiresAt != 1493298100 {
		t.Errorf("ExpiresAt = %d, want 1493298100", claims.ExpiresAt)
	}

	if claims.IssuedAt != 1437179036 {
		t.Errorf("IssuedAt = %d, want 1437179036", claims.IssuedAt)
	}

	if claims.Origin != "*.example.com" {
		t.Errorf("Origin = %q, want *.example.com", claims.Origin)
	}
}

func TestJWTParser_ParseInvalid(t *testing.T) {
	parser := NewJWTParser()

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"no dots", "nodots"},
		{"one dot", "only.one"},
		{"too many dots", "too.many.dots.here"},
		{"invalid base64", "invalid.not_base64.here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.token)
			if err == nil {
				t.Errorf("Parse(%q) should return error", tt.token)
			}
		})
	}
}

func TestJWTParser_ParseWithExpiry(t *testing.T) {
	// Token with expiry at 1493298100 (May 27, 2017)
	testToken := "eyJhbGciOiJFUzI1NiIsImtpZCI6IkFCQzEyMyIsInR5cCI6IkpXVCJ9." +
		"eyJpc3MiOiJERUY0NTZHSUoiLCJpYXQiOjE0MzcxNzkwMzYsImV4cCI6MTQ5MzI5ODEwMCwib3JpZ2luIjoiKi5leGFtcGxlLmNvbSJ9." +
		"fake_signature"

	parser := NewJWTParser()
	expiry, err := parser.ParseWithExpiry(testToken)
	if err != nil {
		t.Fatalf("ParseWithExpiry() error = %v", err)
	}

	expected := time.Unix(1493298100, 0)
	if !expiry.Equal(expected) {
		t.Errorf("Expiry = %v, want %v", expiry, expected)
	}
}

func TestJWTParser_ParseWithExpiryNoExp(t *testing.T) {
	// Token without expiry claim
	// Payload: {"iss":"DEF456"}
	testToken := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJpc3MiOiJERUY0NTYifQ." +
		"fake_signature"

	parser := NewJWTParser()
	_, err := parser.ParseWithExpiry(testToken)
	if err == nil {
		t.Error("ParseWithExpiry() should return error for token without exp claim")
	}
}

func TestJWTParser_IsExpired(t *testing.T) {
	// Create a token that expires 1 hour from now
	futureExp := time.Now().Add(1 * time.Hour).Unix()
	payload := fmt.Sprintf(`{"exp":%d}`, futureExp)
	// We need to base64 encode it, but for this test we'll use a proper structure
	testToken := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9." +
		base64URLEncode([]byte(payload)) + "." +
		"fake_signature"

	parser := NewJWTParser()
	isExpired, expiry, err := parser.IsExpired(testToken)
	if err != nil {
		t.Fatalf("IsExpired() error = %v", err)
	}

	if isExpired {
		t.Error("IsExpired() = true for future token")
	}

	if expiry.IsZero() {
		t.Error("Expiry is zero")
	}
}

func TestJWTParser_TimeUntilExpiry(t *testing.T) {
	// Use the test token with known expiry
	testToken := "eyJhbGciOiJFUzI1NiIsImtpZCI6IkFCQzEyMyIsInR5cCI6IkpXVCJ9." +
		"eyJpc3MiOiJERUY0NTZHSUoiLCJpYXQiOjE0MzcxNzkwMzYsImV4cCI6MTQ5MzI5ODEwMCwib3JpZ2luIjoiKi5leGFtcGxlLmNvbSJ9." +
		"fake_signature"

	parser := NewJWTParser()
	duration, expiry, err := parser.TimeUntilExpiry(testToken)
	if err != nil {
		t.Fatalf("TimeUntilExpiry() error = %v", err)
	}

	// Token expired in 2017, so duration should be negative
	if duration > 0 {
		t.Error("Expected negative duration for expired token")
	}

	expectedExpiry := time.Unix(1493298100, 0)
	if !expiry.Equal(expectedExpiry) {
		t.Errorf("Expiry = %v, want %v", expiry, expectedExpiry)
	}
}

func TestBase64URLDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard", "SGVsbG8gV29ybGQ", "Hello World"},
		{"with padding needed", "SGVsbG8", "Hello"},
		{"url safe -", "SGVsbG8td29ybGQ", "Hello-world"},
		{"url safe _", "SGVsbG8_d29ybGQ", "Hello?world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := base64URLDecode(tt.input)
			if err != nil {
				t.Fatalf("base64URLDecode(%q) error = %v", tt.input, err)
			}
			if string(result) != tt.expected {
				t.Errorf("base64URLDecode(%q) = %q, want %q", tt.input, string(result), tt.expected)
			}
		})
	}
}

func TestBase64URLDecodeInvalid(t *testing.T) {
	// Test with truly invalid base64 (characters not in base64 alphabet)
	result, err := base64URLDecode("!!!invalid!!!")
	if err == nil {
		t.Error("base64URLDecode() should return error for invalid input")
	}
	// Note: base64.StdEncoding.DecodeString returns empty slice on error, not nil
	// So we just check that err is not nil
	_ = result // Suppress unused warning
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{7 * 24 * time.Hour, "7 days"},
		{7*24*time.Hour + 5*time.Hour, "7 days, 5 hours"},
		{3 * time.Hour, "3 hours"},
		{3*time.Hour + 30*time.Minute, "3 hours, 30 minutes"},
		{45 * time.Minute, "45 minutes"},
		{30 * time.Second, "less than a minute"},
		{-1 * time.Hour, "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

// Helper for encoding in tests
func base64URLEncode(data []byte) string {
	s := base64.URLEncoding.EncodeToString(data)
	// Remove padding
	s = strings.TrimRight(s, "=")
	return s
}
