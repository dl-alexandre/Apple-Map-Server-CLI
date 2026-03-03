package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
	"github.com/olekukonko/tablewriter"
)

const searchUsage = `Usage:
  ams search [--near "lat,lng"] [--region "n,e,s,w"] [--near-address <addr>] [--limit N] [--category <cat>] [--json] <query>

Search for places and points of interest.

Note: Flags must come before the query (positional arguments).

Examples:
  ams search --near "37.7749,-122.4194" "coffee shops"
  ams search --near-address "San Francisco, CA" --limit 20 restaurants
  ams search --region "37.8,-122.4,37.7,-122.5" --category fuel "gas stations"
`

// SearchResponse represents the top-level response from Apple Maps /v1/search
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// SearchResult represents an individual POI or address found in the search
type SearchResult struct {
	Name                  string     `json:"name,omitempty"`
	FormattedAddressLines []string   `json:"formattedAddressLines,omitempty"`
	Coordinate            Coordinate `json:"coordinate"`
	PoiCategory           string     `json:"poiCategory,omitempty"`
}

// Coordinate represents a geographic coordinate
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

var searchRequest = doSearchRequest

func NewSearchCommand() Command {
	return Command{
		Name:      "search",
		UsageLine: "search [--near \"lat,lng\"] [--region \"n,e,s,w\"] [--near-address <addr>] [--limit N] [--category <cat>] [--json] <query>",
		Summary:   "Search for places and points of interest",
		Usage:     searchUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			fs := flag.NewFlagSet("search", flag.ContinueOnError)
			near := fs.String("near", "", "Center point for search as 'lat,lng'")
			region := fs.String("region", "", "Bounding box as 'north,east,south,west'")
			nearAddress := fs.String("near-address", "", "Address to center the search around (will be geocoded)")
			limit := fs.Int("limit", 10, "Maximum number of results to return")
			category := fs.String("category", "", "Filter by POI category (e.g., restaurant, cafe)")
			jsonOut := fs.Bool("json", false, "Output raw JSON response")
			fs.SetOutput(io.Discard)
			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, searchUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, searchUsage)
				return ExitUsageError
			}

			// Validate query is present
			if fs.NArg() == 0 {
				fmt.Fprintln(stderr, "search requires a query")
				fmt.Fprint(stderr, searchUsage)
				return ExitUsageError
			}
			query := strings.Join(fs.Args(), " ")

			// Validate limit
			if *limit < 1 {
				fmt.Fprintln(stderr, "limit must be at least 1")
				fmt.Fprint(stderr, searchUsage)
				return ExitUsageError
			}

			// Validate mutually exclusive geographic flags
			geoFlagsUsed := 0
			if *near != "" {
				geoFlagsUsed++
			}
			if *region != "" {
				geoFlagsUsed++
			}
			if *nearAddress != "" {
				geoFlagsUsed++
			}

			if geoFlagsUsed > 1 {
				fmt.Fprintln(stderr, "error: cannot combine --near, --region, and --near-address; please choose one")
				fmt.Fprint(stderr, searchUsage)
				return ExitUsageError
			}

			// Validate coordinate formats early (before network calls) and store for later use
			var validatedLat, validatedLng float64
			var validatedNorth, validatedEast, validatedSouth, validatedWest float64
			hasValidatedCenter := false
			hasValidatedBbox := false

			if *near != "" {
				lat, lng, err := parseCoordinate(*near)
				if err != nil {
					fmt.Fprintf(stderr, "error parsing --near: %v\n", err)
					fmt.Fprint(stderr, searchUsage)
					return ExitUsageError
				}
				validatedLat, validatedLng = lat, lng
				hasValidatedCenter = true
			}

			if *region != "" {
				n, e, s, w, err := parseBoundingBox(*region)
				if err != nil {
					fmt.Fprintf(stderr, "error parsing --region: %v\n", err)
					fmt.Fprint(stderr, searchUsage)
					return ExitUsageError
				}
				validatedNorth, validatedEast, validatedSouth, validatedWest = n, e, s, w
				hasValidatedBbox = true
			}

			client, err := newHTTPClient()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, searchUsage)
				return ExitUsageError
			}

			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, searchUsage)
				return ExitUsageError
			}

			// Print token expiry warning
			fmt.Fprint(stderr, TokenExpiryWarning)

			now := nowFunc().UTC()
			token, _, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					fmt.Fprintln(stderr, err)
					fmt.Fprint(stderr, searchUsage)
					return ExitUsageError
				}
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}

			// Prepare search parameters using validated values
			searchLat := validatedLat
			searchLng := validatedLng
			hasCenter := hasValidatedCenter
			bboxNorth, bboxEast, bboxSouth, bboxWest := validatedNorth, validatedEast, validatedSouth, validatedWest
			hasBbox := hasValidatedBbox

			if *nearAddress != "" {
				// Geocode the address first
				status, body, err := geocodeRequest(client, token.Value, *nearAddress)
				if err != nil {
					fmt.Fprintf(stderr, "error geocoding address: %v\n", err)
					return ExitRuntimeError
				}
				if status < 200 || status > 299 {
					fmt.Fprintf(stderr, "geocoding failed with status %d\n", status)
					if len(body) > 0 {
						fmt.Fprintln(stderr, string(body))
					}
					return ExitRuntimeError
				}

				// Extract coordinates from geocode response
				var geoResp map[string]any
				if err := json.Unmarshal(body, &geoResp); err != nil {
					fmt.Fprintf(stderr, "error parsing geocode response: %v\n", err)
					return ExitRuntimeError
				}

				resultsAny, ok := geoResp["results"]
				if !ok {
					fmt.Fprintln(stderr, "geocoding returned no results")
					return ExitRuntimeError
				}

				results, ok := resultsAny.([]any)
				if !ok || len(results) == 0 {
					fmt.Fprintln(stderr, "geocoding returned no results")
					return ExitRuntimeError
				}

				firstResult, ok := results[0].(map[string]any)
				if !ok {
					fmt.Fprintln(stderr, "error parsing geocode result")
					return ExitRuntimeError
				}

				lat, lng, hasCoord := coordinateValues(firstResult)
				if !hasCoord {
					fmt.Fprintln(stderr, "geocoding result missing coordinates")
					return ExitRuntimeError
				}

				searchLat = lat
				searchLng = lng
				hasCenter = true
			}

			// Build and execute search request
			status, body, err := searchRequest(client, token.Value, query, *limit, *category, searchLat, searchLng, hasCenter, bboxNorth, bboxEast, bboxSouth, bboxWest, hasBbox)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}
			if status < 200 || status > 299 {
				fmt.Fprintf(stderr, "search failed with status %d\n", status)
				if len(body) > 0 {
					fmt.Fprintln(stderr, string(body))
				}
				return ExitRuntimeError
			}

			// Output results
			if *jsonOut {
				pretty, ok := formatJSON(body)
				if ok {
					fmt.Fprintln(stdout, pretty)
					return ExitSuccess
				}
				fmt.Fprintln(stdout, string(body))
				return ExitSuccess
			}

			ok := writeSearchTable(stdout, body, *limit, hasCenter, searchLat, searchLng)
			if !ok {
				fmt.Fprintln(stdout, string(body))
				return ExitSuccess
			}
			return ExitSuccess
		},
	}
}

func doSearchRequest(
	client *httpclient.Client,
	token string,
	query string,
	limit int,
	category string,
	centerLat, centerLng float64,
	hasCenter bool,
	bboxNorth, bboxEast, bboxSouth, bboxWest float64,
	hasBbox bool,
) (int, []byte, error) {
	params := url.Values{}
	params.Set("q", query)

	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	if category != "" {
		params.Set("includePoiCategories", category)
	}

	if hasCenter {
		params.Set("center", fmt.Sprintf("%.6f,%.6f", centerLat, centerLng))
	}

	if hasBbox {
		params.Set("region", fmt.Sprintf("%.6f,%.6f,%.6f,%.6f", bboxNorth, bboxEast, bboxSouth, bboxWest))
	}

	req, err := client.NewRequest(http.MethodGet, "/v1/search", params, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent())

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, body, nil
}

func writeSearchTable(w io.Writer, body []byte, limit int, hasCenter bool, centerLat, centerLng float64) bool {
	var resp SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return false
	}

	if len(resp.Results) == 0 {
		fmt.Fprintln(w, "no results")
		return true
	}

	table := tablewriter.NewWriter(w)

	// Build headers based on whether we have center point for distance calculation
	if hasCenter {
		table.Header("Name", "Category", "Address", "Distance")
	} else {
		table.Header("Name", "Category", "Address", "Coordinates")
	}

	rows := 0
	for _, result := range resp.Results {
		if rows >= limit {
			break
		}

		name := result.Name
		if name == "" {
			name = "-"
		}

		category := result.PoiCategory
		if category == "" {
			category = "-"
		}

		address := "-"
		if len(result.FormattedAddressLines) > 0 {
			address = result.FormattedAddressLines[0]
		}

		var lastCol string
		if hasCenter {
			dist := haversineDistance(centerLat, centerLng, result.Coordinate.Latitude, result.Coordinate.Longitude)
			lastCol = formatDistance(dist)
		} else {
			lastCol = fmt.Sprintf("%.6f,%.6f", result.Coordinate.Latitude, result.Coordinate.Longitude)
		}

		if err := table.Append([]string{name, category, address, lastCol}); err != nil {
			return false
		}
		rows++
	}

	if rows == 0 {
		fmt.Fprintln(w, "no results")
		return true
	}

	if err := table.Render(); err != nil {
		return false
	}
	return true
}

// haversineDistance calculates the distance between two points in meters
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000 // meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
