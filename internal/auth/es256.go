package auth

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ES256Signer handles ES256 (ECDSA P-256 with SHA-256) JWT signing for Apple Maps Server API
type ES256Signer struct {
	privateKey *ecdsa.PrivateKey
	teamID     string
	keyID      string
	origin     string
}

// NewES256Signer creates a new JWT signer from a private key file
// The private key should be in PKCS#8 format (.p8 file from Apple Developer Portal)
func NewES256Signer(privateKeyPath, teamID string) (*ES256Signer, error) {
	if strings.TrimSpace(teamID) == "" {
		return nil, fmt.Errorf("team ID is required")
	}

	if strings.TrimSpace(privateKeyPath) == "" {
		return nil, fmt.Errorf("private key path is required")
	}

	// #nosec G304 - Private key path is intentionally configurable via env var/CLI flag
	// This is legitimate for CLI tools where users specify their own key file location
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key file: %w", err)
	}

	privateKey, err := parsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	return &ES256Signer{
		privateKey: privateKey,
		teamID:     teamID,
	}, nil
}

// NewES256SignerFromReader creates a new JWT signer from a private key reader
func NewES256SignerFromReader(r io.Reader, teamID string) (*ES256Signer, error) {
	if strings.TrimSpace(teamID) == "" {
		return nil, fmt.Errorf("team ID is required")
	}

	keyData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %w", err)
	}

	privateKey, err := parsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	return &ES256Signer{
		privateKey: privateKey,
		teamID:     teamID,
	}, nil
}

// SetKeyID sets the Key ID (optional, extracted from JWT if available)
func (s *ES256Signer) SetKeyID(keyID string) {
	s.keyID = keyID
}

// SetOrigin sets the origin domain for the JWT (optional)
func (s *ES256Signer) SetOrigin(origin string) {
	s.origin = origin
}

// GenerateJWT generates a new JWT token for Apple Maps Server API
// The token is valid for 7 days (Apple's maximum)
func (s *ES256Signer) GenerateJWT(now time.Time) (string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	// Apple Maps Server API uses 7-day maximum expiration
	exp := now.Add(7 * 24 * time.Hour)

	// Build JWT header
	header := map[string]string{
		"alg": "ES256",
		"typ": "JWT",
	}
	if s.keyID != "" {
		header["kid"] = s.keyID
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("encoding header: %w", err)
	}

	// Build JWT claims
	claims := map[string]any{
		"iss": s.teamID,
		"iat": now.Unix(),
		"exp": exp.Unix(),
	}

	if s.origin != "" {
		claims["origin"] = s.origin
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encoding claims: %w", err)
	}

	// Encode header and payload
	headerB64 := base64URLEncode(headerJSON)
	claimsB64 := base64URLEncode(claimsJSON)

	// Create signing input
	signingInput := headerB64 + "." + claimsB64

	// Sign with ES256
	signature, err := s.signES256([]byte(signingInput))
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}

	// Build final JWT
	jwt := signingInput + "." + base64URLEncode(signature)

	return jwt, nil
}

// signES256 signs data using ECDSA with P-256 curve and SHA-256
func (s *ES256Signer) signES256(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)

	r, sInt, err := ecdsa.Sign(rand.Reader, s.privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("ECDSA sign failed: %w", err)
	}

	// Convert signature to raw format (r || s)
	// ES256 signature is r || s, each 32 bytes (P-256 curve)
	curveByteSize := (s.privateKey.Curve.Params().BitSize + 7) / 8

	// Ensure r and s are properly padded to curveByteSize
	rBytes := r.Bytes()
	sBytes := sInt.Bytes()

	// Pad with leading zeros if necessary
	if len(rBytes) < curveByteSize {
		rBytes = append(make([]byte, curveByteSize-len(rBytes)), rBytes...)
	}
	if len(sBytes) < curveByteSize {
		sBytes = append(make([]byte, curveByteSize-len(sBytes)), sBytes...)
	}

	// Concatenate r || s (raw format for JWT)
	signature := make([]byte, curveByteSize*2)
	copy(signature[:curveByteSize], rBytes)
	copy(signature[curveByteSize:], sBytes)

	return signature, nil
}

// parsePrivateKey parses a private key from PEM or PKCS#8 format
func parsePrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	// Try PEM format first
	block, _ := pem.Decode(data)
	if block != nil {
		// Try PKCS#8
		if block.Type == "PRIVATE KEY" {
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parsing PKCS#8: %w", err)
			}
			ecKey, ok := key.(*ecdsa.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("key is not ECDSA (required for ES256)")
			}
			return ecKey, nil
		}

		// Try EC private key format
		if block.Type == "EC PRIVATE KEY" {
			return x509.ParseECPrivateKey(block.Bytes)
		}
	}

	// Try raw PKCS#8 (binary)
	key, err := x509.ParsePKCS8PrivateKey(data)
	if err == nil {
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not ECDSA (required for ES256)")
		}
		return ecKey, nil
	}

	// Try raw EC private key
	return x509.ParseECPrivateKey(data)
}
