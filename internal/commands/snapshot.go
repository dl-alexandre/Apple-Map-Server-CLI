package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/pkg/mapsrv"
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
			zoom := 12
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

			// Load auth config
			_, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitUsageError
			}

			// Print token expiry warning
			fmt.Fprint(stderr, TokenExpiryWarning)

			// Get snapshot credentials
			teamID := os.Getenv("AMS_TEAM_ID")
			keyID := os.Getenv("AMS_KEY_ID")
			privateKey := os.Getenv("AMS_PRIVATE_KEY")

			if privateKey == "" {
				keyPath := os.Getenv("AMS_PRIVATE_KEY_PATH")
				if keyPath != "" {
					// Validate path to prevent directory traversal
					cleanPath := filepath.Clean(keyPath)
					if strings.Contains(cleanPath, "..") {
						fmt.Fprintf(stderr, "error: invalid private key path (path traversal detected)\n")
						return ExitUsageError
					}
					// #nosec G703 - Path is validated above to prevent traversal
					data, err := os.ReadFile(cleanPath)
					if err != nil {
						fmt.Fprintf(stderr, "failed to read private key: %v\n", err)
						return ExitUsageError
					}
					privateKey = string(data)
				}
			}

			if teamID == "" || keyID == "" || privateKey == "" {
				fmt.Fprintln(stderr, "error: AMS_TEAM_ID, AMS_KEY_ID, and AMS_PRIVATE_KEY environment variables required")
				fmt.Fprintln(stderr, "The snapshot API requires your private key for URL signing")
				return ExitUsageError
			}

			// Create HTTP client
			httpClient, err := httpclient.New()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitUsageError
			}

			// Create mapsrv client
			client := mapsrv.NewClient(httpClient.BaseURL, "")
			_, err = client.WithSnapshotAuth(teamID, keyID, privateKey)
			if err != nil {
				fmt.Fprintf(stderr, "failed to configure snapshot auth: %v\n", err)
				return ExitUsageError
			}

			// Build snapshot params
			params := mapsrv.SnapshotParams{
				Center: center,
				Zoom:   zoom,
				Size:   size,
				Format: format,
			}

			// Download and save snapshot
			if err := client.SaveSnapshot(params, output); err != nil {
				fmt.Fprintf(stderr, "failed to generate snapshot: %v\n", err)
				return ExitRuntimeError
			}

			fmt.Fprintf(stdout, "Snapshot saved to: %s\n", output)
			return ExitSuccess
		},
	}
}
