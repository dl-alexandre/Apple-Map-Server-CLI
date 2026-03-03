package commands

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

const snapshotUsage = `Usage:
  ams snapshot <center> [--zoom N] [--size WxH] [--format png|jpg] [--output <path>]

Generate a static map image (snapshot) for a location.

The snapshot API requires URL signing with your private key for authentication.

Examples:
  ams snapshot "37.7749,-122.4194"
  ams snapshot "San Francisco, CA" --zoom 14 --size 600x400
  ams snapshot "1 Infinite Loop, Cupertino" --zoom 16 --output map.png
  ams snapshot "London, UK" --size 800x600 --format jpg --output london.jpg
`

// SnapshotConfig holds configuration for snapshot generation
type SnapshotConfig struct {
	Center string
	Zoom   int
	Size   string
	Format string
	Output string
}

var snapshotRequest = doSnapshotRequest

func NewSnapshotCommand() Command {
	return Command{
		Name:      "snapshot",
		UsageLine: "snapshot <center> [--zoom N] [--size WxH] [--format png|jpg] [--output <path>]",
		Summary:   "Generate a static map image",
		Usage:     snapshotUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			// Validate center argument
			if len(args) == 0 {
				fmt.Fprintln(stderr, "snapshot requires a center location")
				fmt.Fprint(stderr, snapshotUsage)
				return ExitUsageError
			}

			center := args[0]

			// Parse flags
			var zoom int = 12
			size := "600x400"
			format := "png"
			output := ""

			// Simple flag parsing
			for i := 1; i < len(args); i++ {
				switch args[i] {
				case "--zoom":
					if i+1 < len(args) {
						z, err := strconv.Atoi(args[i+1])
						if err == nil && z >= 1 && z <= 20 {
							zoom = z
						}
						i++
					}
				case "--size":
					if i+1 < len(args) {
						size = args[i+1]
						i++
					}
				case "--format":
					if i+1 < len(args) {
						f := strings.ToLower(args[i+1])
						if f == "png" || f == "jpg" || f == "jpeg" {
							format = f
						}
						i++
					}
				case "--output":
					if i+1 < len(args) {
						output = args[i+1]
						i++
					}
				}
			}

			// Default output filename
			if output == "" {
				output = fmt.Sprintf("snapshot_%d.%s", time.Now().Unix(), format)
			}

			// Load Maps Token (for consistency check)
			_, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, snapshotUsage)
				return ExitUsageError
			}

			// Get Team ID and Key ID from environment
			teamID := os.Getenv("AMS_TEAM_ID")
			keyID := os.Getenv("AMS_KEY_ID")

			if teamID == "" || keyID == "" {
				fmt.Fprintln(stderr, "error: AMS_TEAM_ID and AMS_KEY_ID environment variables required")
				fmt.Fprintln(stderr, "These identify your Apple Developer team and Maps key")
				fmt.Fprint(stderr, snapshotUsage)
				return ExitUsageError
			}

			// Print token expiry warning
			fmt.Fprint(stderr, TokenExpiryWarning)

			// Create snapshot signer
			privateKey := os.Getenv("AMS_PRIVATE_KEY")
			if privateKey == "" {
				// Try to load from file
				keyPath := os.Getenv("AMS_PRIVATE_KEY_PATH")
				if keyPath != "" {
					data, err := os.ReadFile(keyPath)
					if err != nil {
						fmt.Fprintf(stderr, "failed to read private key: %v\n", err)
						return ExitUsageError
					}
					privateKey = string(data)
				}
			}

			if privateKey == "" {
				fmt.Fprintln(stderr, "error: AMS_PRIVATE_KEY or AMS_PRIVATE_KEY_PATH environment variable required")
				fmt.Fprintln(stderr, "The snapshot API requires your private key for URL signing")
				fmt.Fprint(stderr, snapshotUsage)
				return ExitUsageError
			}

			signer, err := auth.NewSnapshotSigner(teamID, keyID, privateKey)
			if err != nil {
				fmt.Fprintf(stderr, "failed to create snapshot signer: %v\n", err)
				return ExitUsageError
			}

			// Build URL parameters
			params := map[string]string{
				"teamId": teamID,
				"keyId":  keyID,
				"t":      "standard",
			}

			if format == "jpg" || format == "jpeg" {
				params["format"] = "jpg"
			}

			// Build the base URL
			client, err := httpclient.New()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitUsageError
			}

			baseURL := client.BaseURL
			urlPath := buildSnapshotPath(baseURL, center, zoom, size, params)

			// Sign the URL
			signature, err := signer.SignURL(urlPath)
			if err != nil {
				fmt.Fprintf(stderr, "failed to sign URL: %v\n", err)
				return ExitRuntimeError
			}

			// Append signature to URL
			fullURL := fmt.Sprintf("%s&signature=%s", urlPath, signature)

			// Download the image
			if err := downloadSnapshot(fullURL, output, stderr); err != nil {
				fmt.Fprintf(stderr, "failed to download snapshot: %v\n", err)
				return ExitRuntimeError
			}

			fmt.Fprintf(stdout, "Snapshot saved to: %s\n", output)
			return ExitSuccess
		},
	}
}

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

func downloadSnapshot(url, outputPath string, stderr io.Writer) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Write image to file
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func doSnapshotRequest(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
