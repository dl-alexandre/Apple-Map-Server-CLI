// Package mapsrv provides a high-level client for Apple Maps Server APIs.
// It abstracts the HTTP transport, authentication, and common operations
// to provide a clean interface for CLI commands.
package mapsrv

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
)

// Client provides a unified interface to Apple Maps Server APIs
type Client struct {
	HTTPClient   *http.Client
	BaseURL      string
	Token        string
	SnapshotAuth *SnapshotAuth
}

// SnapshotAuth holds credentials for snapshot API URL signing
type SnapshotAuth struct {
	TeamID     string
	KeyID      string
	PrivateKey string
	Signer     *auth.SnapshotSigner
}

// SnapshotParams defines parameters for generating a map snapshot
type SnapshotParams struct {
	Center  string
	Zoom    int
	Size    string
	Format  string
	MapType string // "standard", "hybrid", "satellite"
}

// NewClient creates a new mapsrv client with the given base URL and auth token
func NewClient(baseURL, token string) *Client {
	return &Client{
		HTTPClient: &http.Client{},
		BaseURL:    baseURL,
		Token:      token,
	}
}

// WithSnapshotAuth configures the client with snapshot API credentials
func (c *Client) WithSnapshotAuth(teamID, keyID, privateKey string) (*Client, error) {
	signer, err := auth.NewSnapshotSigner(teamID, keyID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot signer: %w", err)
	}

	c.SnapshotAuth = &SnapshotAuth{
		TeamID:     teamID,
		KeyID:      keyID,
		PrivateKey: privateKey,
		Signer:     signer,
	}

	return c, nil
}

// DownloadSnapshot generates and downloads a map snapshot image.
// Returns the raw image bytes or an error.
func (c *Client) DownloadSnapshot(params SnapshotParams) ([]byte, error) {
	if c.SnapshotAuth == nil {
		return nil, fmt.Errorf("snapshot authentication not configured")
	}

	// Build URL parameters
	queryParams := map[string]string{
		"teamId": c.SnapshotAuth.TeamID,
		"keyId":  c.SnapshotAuth.KeyID,
		"t":      params.MapType,
	}

	if params.MapType == "" {
		queryParams["t"] = "standard"
	}

	if params.Format == "jpg" || params.Format == "jpeg" {
		queryParams["format"] = "jpg"
	}

	// Build the URL path
	urlPath := buildSnapshotPath(c.BaseURL, params.Center, params.Zoom, params.Size, queryParams)

	// Sign the URL
	signature, err := c.SnapshotAuth.Signer.SignURL(urlPath)
	if err != nil {
		return nil, fmt.Errorf("signing URL: %w", err)
	}

	// Construct full URL with signature
	fullURL := fmt.Sprintf("%s&signature=%s", urlPath, signature)

	// Download the image
	return c.downloadImage(fullURL)
}

// buildSnapshotPath constructs the snapshot API URL path
func buildSnapshotPath(baseURL, center string, zoom int, size string, params map[string]string) string {
	query := url.Values{}
	query.Set("center", center)
	query.Set("z", strconv.Itoa(zoom))
	query.Set("size", size)

	for key, value := range params {
		query.Set(key, value)
	}

	return fmt.Sprintf("%s/api/v1/snapshot?%s", baseURL, query.Encode())
}

// downloadImage performs the HTTP GET and returns the response body
func (c *Client) downloadImage(url string) ([]byte, error) {
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return data, nil
}

// SaveSnapshot downloads a snapshot and saves it to the specified path
func (c *Client) SaveSnapshot(params SnapshotParams, outputPath string) error {
	data, err := c.DownloadSnapshot(params)
	if err != nil {
		return err
	}

	return writeFile(outputPath, data)
}

// writeFile writes data to the specified path, creating directories if needed
func writeFile(path string, data []byte) error {
	// Create output directory if needed
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}

	return os.WriteFile(path, data, 0600) // #nosec G306 - user output file with restricted permissions
}
