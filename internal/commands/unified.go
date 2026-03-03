package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

const unifiedUsage = `Usage:
  ams unified <query> [--near "lat,lng"] [--zoom N] [--output <path>]

Search for a place and generate a snapshot in one command.

This combines search and snapshot APIs to quickly visualize search results.
The first search result's coordinates are used for the snapshot.

Examples:
  ams unified "Golden Gate Bridge"
  ams unified "coffee shops" --near "37.7749,-122.4194" --zoom 15
  ams unified "restaurants" --near-address "San Francisco" --zoom 14 --output sf.png
  ams unified "airports" --near "London, UK" --zoom 12
`

var unifiedSearchRequest = doSearchRequest
var unifiedSnapshotRequest = doSnapshotRequest

func NewUnifiedCommand() Command {
	return Command{
		Name:      "unified",
		UsageLine: "unified <query> [--near \"lat,lng\"] [--zoom N] [--output <path>]",
		Summary:   "Search and generate snapshot in one command",
		Usage:     unifiedUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			// Parse arguments
			if len(args) == 0 {
				fmt.Fprintln(stderr, "unified requires a query")
				fmt.Fprint(stderr, unifiedUsage)
				return ExitUsageError
			}

			// Simple flag parsing
			var query string
			var near string
			var zoom int = 14
			var output string = ""

			// Find where flags start
			queryEnd := len(args)
			for i, arg := range args {
				if strings.HasPrefix(arg, "--") {
					queryEnd = i
					break
				}
			}

			query = strings.Join(args[:queryEnd], " ")

			// Parse flags
			for i := queryEnd; i < len(args); i++ {
				switch args[i] {
				case "--near":
					if i+1 < len(args) {
						near = args[i+1]
						i++
					}
				case "--zoom":
					if i+1 < len(args) {
						z, err := strconv.Atoi(args[i+1])
						if err == nil && z >= 1 && z <= 20 {
							zoom = z
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

			if query == "" {
				fmt.Fprintln(stderr, "unified requires a query")
				fmt.Fprint(stderr, unifiedUsage)
				return ExitUsageError
			}

			// Set up HTTP client
			client, err := httpclient.New()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitUsageError
			}

			// Load auth config
			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitUsageError
			}

			// Print token expiry warning
			fmt.Fprint(stderr, TokenExpiryWarning)

			// Get access token
			now := time.Now().UTC()
			token, _, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					fmt.Fprintln(stderr, err)
					return ExitUsageError
				}
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}

			// Step 1: Search for the place
			fmt.Fprintf(stderr, "Searching for: %s\n", query)

			var searchLat, searchLng float64
			hasCenter := false

			// Parse near coordinate if provided
			if near != "" {
				lat, lng, err := parseCoordinate(near)
				if err != nil {
					fmt.Fprintf(stderr, "error parsing --near: %v\n", err)
					return ExitUsageError
				}
				searchLat = lat
				searchLng = lng
				hasCenter = true
			}

			// Execute search
			limit := 1
			status, body, err := unifiedSearchRequest(client, token.Value, query, limit, "", searchLat, searchLng, hasCenter, 0, 0, 0, 0, false)
			if err != nil {
				fmt.Fprintf(stderr, "search error: %v\n", err)
				return ExitRuntimeError
			}
			if status < 200 || status > 299 {
				fmt.Fprintf(stderr, "search failed with status %d\n", status)
				return ExitRuntimeError
			}

			// Parse search results
			var searchResp SearchResponse
			if err := json.Unmarshal(body, &searchResp); err != nil {
				fmt.Fprintf(stderr, "error parsing search response: %v\n", err)
				return ExitRuntimeError
			}

			if len(searchResp.Results) == 0 {
				fmt.Fprintln(stderr, "no search results found")
				return ExitRuntimeError
			}

			// Get coordinates from first result
			result := searchResp.Results[0]
			lat := result.Coordinate.Latitude
			lng := result.Coordinate.Longitude

			placeName := result.Name
			if placeName == "" {
				placeName = query
			}

			fmt.Fprintf(stdout, "Found: %s (%.6f, %.6f)\n", placeName, lat, lng)

			// Step 2: Generate snapshot
			fmt.Fprintf(stderr, "Generating snapshot...\n")

			// Get snapshot credentials
			teamID := os.Getenv("AMS_TEAM_ID")
			keyID := os.Getenv("AMS_KEY_ID")
			privateKey := os.Getenv("AMS_PRIVATE_KEY")

			if teamID == "" || keyID == "" || privateKey == "" {
				fmt.Fprintln(stderr, "warning: snapshot credentials not configured (AMS_TEAM_ID, AMS_KEY_ID, AMS_PRIVATE_KEY)")
				fmt.Fprintln(stderr, "Search completed successfully, but snapshot generation requires separate credentials")
				return ExitSuccess
			}

			signer, err := auth.NewSnapshotSigner(teamID, keyID, privateKey)
			if err != nil {
				fmt.Fprintf(stderr, "failed to create snapshot signer: %v\n", err)
				return ExitUsageError
			}

			// Build snapshot URL
			center := fmt.Sprintf("%.6f,%.6f", lat, lng)
			size := "800x600"

			if output == "" {
				output = fmt.Sprintf("%s_%d.png", sanitizeFilename(placeName), time.Now().Unix())
			}

			params := map[string]string{
				"teamId": teamID,
				"keyId":  keyID,
				"t":      "standard",
			}

			baseURL := client.BaseURL
			urlPath := buildSnapshotPath(baseURL, center, zoom, size, params)

			signature, err := signer.SignURL(urlPath)
			if err != nil {
				fmt.Fprintf(stderr, "failed to sign URL: %v\n", err)
				return ExitRuntimeError
			}

			fullURL := fmt.Sprintf("%s&signature=%s", urlPath, signature)

			// Download snapshot
			if err := downloadSnapshot(fullURL, output, stderr); err != nil {
				fmt.Fprintf(stderr, "failed to download snapshot: %v\n", err)
				return ExitRuntimeError
			}

			fmt.Fprintf(stdout, "✓ Snapshot saved to: %s\n", output)
			return ExitSuccess
		},
	}
}

// sanitizeFilename removes problematic characters from filenames
func sanitizeFilename(name string) string {
	// Replace spaces and special chars with underscore
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

func doUnifiedSnapshotRequest(url string) ([]byte, error) {
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
