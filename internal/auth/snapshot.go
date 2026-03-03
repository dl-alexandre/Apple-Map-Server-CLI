package auth

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
)

// SnapshotSigner handles signature generation for Maps Web Snapshots
type SnapshotSigner struct {
	TeamID     string
	KeyID      string
	PrivateKey *ecdsa.PrivateKey
}

// NewSnapshotSigner creates a new snapshot signer from PEM private key
func NewSnapshotSigner(teamID, keyID, privateKeyPEM string) (*SnapshotSigner, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try EC private key format
		key, err = x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not ECDSA")
	}

	return &SnapshotSigner{
		TeamID:     teamID,
		KeyID:      keyID,
		PrivateKey: ecKey,
	}, nil
}

// SignURL generates a signature for a snapshot URL
// The pathAndQuery should be the URL path and query string (e.g., "/api/v1/snapshot?center=...")
func (s *SnapshotSigner) SignURL(pathAndQuery string) (string, error) {
	// Create the signature input: teamId + pathAndQuery
	signatureInput := s.TeamID + pathAndQuery

	// Hash the input
	hash := sha256Hash(signatureInput)

	// Sign with ECDSA
	r, sVal, err := ecdsa.Sign(nil, s.PrivateKey, hash)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	// Convert to ASN.1 DER format
	signature, err := marshalECDSASignature(r, sVal)
	if err != nil {
		return "", fmt.Errorf("failed to marshal signature: %w", err)
	}

	// Base64 URL-safe encoding (no padding)
	encoded := base64.RawURLEncoding.EncodeToString(signature)

	return encoded, nil
}

// sha256Hash computes SHA256 hash
func sha256Hash(input string) []byte {
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

// marshalECDSASignature converts r and s values to ASN.1 DER format
func marshalECDSASignature(r, s *big.Int) ([]byte, error) {
	type ecdsaSignature struct {
		R, S *big.Int
	}
	sig := ecdsaSignature{R: r, S: s}
	return asn1.Marshal(sig)
}

// BuildSnapshotURL builds the base snapshot URL with parameters
func BuildSnapshotURL(baseURL, center string, zoom int, size string, additionalParams map[string]string) string {
	params := make([]string, 0)

	// Required parameters
	params = append(params, fmt.Sprintf("center=%s", center))
	params = append(params, fmt.Sprintf("z=%d", zoom))
	params = append(params, fmt.Sprintf("size=%s", size))

	// Team ID (required for the URL itself)
	if teamID, ok := additionalParams["teamId"]; ok {
		params = append(params, fmt.Sprintf("teamId=%s", teamID))
	}

	// Key ID (required for the URL itself)
	if keyID, ok := additionalParams["keyId"]; ok {
		params = append(params, fmt.Sprintf("keyId=%s", keyID))
	}

	// Optional parameters
	for key, value := range additionalParams {
		if key != "teamId" && key != "keyId" {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
	}

	queryString := strings.Join(params, "&")
	return fmt.Sprintf("%s/api/v1/snapshot?%s", baseURL, queryString)
}
