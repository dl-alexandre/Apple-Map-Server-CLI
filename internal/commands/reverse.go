package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
	"github.com/olekukonko/tablewriter"
)

const reverseUsage = `Usage:
  ams reverse <lat>,<lon> [--limit N] [--json]

Reverse geocode coordinates.
`

var reverseRequest = doReverseRequest

func NewReverseCommand() Command {
	return Command{
		Name:      "reverse",
		UsageLine: "reverse <lat>,<lon> [--limit N] [--json]",
		Summary:   "Reverse geocode coordinates",
		Usage:     reverseUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			fs := flag.NewFlagSet("reverse", flag.ContinueOnError)
			jsonOut := fs.Bool("json", false, "Output raw JSON response")
			limit := fs.Int("limit", 5, "Maximum number of results to display")
			fs.SetOutput(io.Discard)
			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, reverseUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, reverseUsage)
				return ExitUsageError
			}

			if *limit < 1 {
				fmt.Fprintln(stderr, "limit must be at least 1")
				fmt.Fprint(stderr, reverseUsage)
				return ExitUsageError
			}

			if fs.NArg() != 1 {
				fmt.Fprintln(stderr, "reverse requires coordinates")
				fmt.Fprint(stderr, reverseUsage)
				return ExitUsageError
			}

			lat, lon, err := parseCoordinates(fs.Arg(0))
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, reverseUsage)
				return ExitUsageError
			}

			client, err := newHTTPClient()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, reverseUsage)
				return ExitUsageError
			}

			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, reverseUsage)
				return ExitUsageError
			}

			// Print token expiry warning
			fmt.Fprint(stderr, TokenExpiryWarning)

			now := nowFunc().UTC()
			token, _, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					fmt.Fprintln(stderr, err)
					fmt.Fprint(stderr, reverseUsage)
					return ExitUsageError
				}
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}

			status, body, err := reverseRequest(client, token.Value, lat, lon)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}
			if status < 200 || status > 299 {
				fmt.Fprintf(stderr, "reverse failed with status %d\n", status)
				if len(body) > 0 {
					fmt.Fprintln(stderr, string(body))
				}
				return ExitRuntimeError
			}

			if *jsonOut {
				pretty, ok := formatReverseJSON(body)
				if ok {
					fmt.Fprintln(stdout, pretty)
					return ExitSuccess
				}
				fmt.Fprintln(stdout, string(body))
				return ExitSuccess
			}

			ok := writeReverseTable(stdout, body, *limit)
			if !ok {
				fmt.Fprintln(stdout, string(body))
				return ExitSuccess
			}
			return ExitSuccess
		},
	}
}

func doReverseRequest(client *httpclient.Client, token string, lat, lon float64) (int, []byte, error) {
	params := url.Values{}
	params.Set("loc", fmt.Sprintf("%.6f,%.6f", lat, lon))
	if params.Get("loc") == "" {
		return 0, nil, errors.New("coordinates are empty")
	}
	req, err := client.NewRequest(http.MethodGet, "/v1/reverseGeocode", params, nil)
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

func parseCoordinates(input string) (float64, float64, error) {
	parts := strings.Split(input, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid coordinates: %s", input)
	}
	latStr := strings.TrimSpace(parts[0])
	lonStr := strings.TrimSpace(parts[1])
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude: %s", latStr)
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude: %s", lonStr)
	}
	if lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("latitude out of range: %v", lat)
	}
	if lon < -180 || lon > 180 {
		return 0, 0, fmt.Errorf("longitude out of range: %v", lon)
	}
	return lat, lon, nil
}

func writeReverseTable(w io.Writer, body []byte, limit int) bool {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}

	resultsAny, ok := payload["results"]
	if !ok {
		return false
	}

	results, ok := resultsAny.([]any)
	if !ok {
		return false
	}

	if len(results) == 0 {
		fmt.Fprintln(w, "no results")
		return true
	}

	table := tablewriter.NewWriter(w)
	table.Header("Address", "Latitude", "Longitude")

	rows := 0
	maxRows := limit
	if maxRows <= 0 {
		maxRows = 1
	}

	for _, entry := range results {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		address := pickString(item, "formattedAddress", "displayAddress", "name")
		lat, lon, hasCoord := coordinateValues(item)
		if address == "" && !hasCoord {
			continue
		}
		latStr := "-"
		lonStr := "-"
		if hasCoord {
			latStr = fmt.Sprintf("%.6f", lat)
			lonStr = fmt.Sprintf("%.6f", lon)
		}
		if address == "" {
			address = "-"
		}
		if err := table.Append([]string{address, latStr, lonStr}); err != nil {
			return false
		}
		rows++
		if rows >= maxRows {
			break
		}
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

func formatReverseJSON(body []byte) (string, bool) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		return "", false
	}
	return buf.String(), true
}
