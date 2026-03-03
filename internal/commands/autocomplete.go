package commands

import (
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

const autocompleteUsage = `Usage:
  ams search autocomplete [--near "lat,lng"] [--limit N] [--json] <query>

Get autocomplete suggestions for search queries.

Note: Flags must come before the query (positional arguments).

Examples:
  ams search autocomplete --near "37.7749,-122.4194" "starbu"
  ams search autocomplete --limit 10 --json "pizza"
`

// AutocompleteResponse represents the top-level response from Apple Maps /v1/searchAutocomplete
type AutocompleteResponse struct {
	Results []AutocompleteResult `json:"results"`
}

// AutocompleteResult represents a single autocomplete suggestion
type AutocompleteResult struct {
	DisplayLines  []string `json:"displayLines"`
	CompletionURL string   `json:"completionUrl,omitempty"`
	// Additional fields may include:
	// - coordinate (for location-biased results)
	// - poiCategory
}

var autocompleteRequest = doAutocompleteRequest

func NewAutocompleteCommand() Command {
	return Command{
		Name:      "search autocomplete",
		UsageLine: "search autocomplete [--near \"lat,lng\"] [--limit N] [--json] <query>",
		Summary:   "Get autocomplete suggestions for search queries",
		Usage:     autocompleteUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			fs := flag.NewFlagSet("autocomplete", flag.ContinueOnError)
			near := fs.String("near", "", "Center point for location bias as 'lat,lng'")
			limit := fs.Int("limit", 10, "Maximum number of suggestions to return")
			jsonOut := fs.Bool("json", false, "Output raw JSON response")
			fs.SetOutput(io.Discard)

			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, autocompleteUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, autocompleteUsage)
				return ExitUsageError
			}

			// Validate query is present
			if fs.NArg() == 0 {
				fmt.Fprintln(stderr, "autocomplete requires a query")
				fmt.Fprint(stderr, autocompleteUsage)
				return ExitUsageError
			}
			query := strings.Join(fs.Args(), " ")

			// Validate limit
			if *limit < 1 {
				fmt.Fprintln(stderr, "limit must be at least 1")
				fmt.Fprint(stderr, autocompleteUsage)
				return ExitUsageError
			}

			// Validate and parse coordinate if provided
			var searchLat, searchLng float64
			hasCenter := false

			if *near != "" {
				lat, lng, err := parseCoordinate(*near)
				if err != nil {
					fmt.Fprintf(stderr, "error parsing --near: %v\n", err)
					fmt.Fprint(stderr, autocompleteUsage)
					return ExitUsageError
				}
				searchLat = lat
				searchLng = lng
				hasCenter = true
			}

			client, err := newHTTPClient()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, autocompleteUsage)
				return ExitUsageError
			}

			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, autocompleteUsage)
				return ExitUsageError
			}

			// Print token expiry warning
			fmt.Fprint(stderr, TokenExpiryWarning)

			now := nowFunc().UTC()
			token, _, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					fmt.Fprintln(stderr, err)
					fmt.Fprint(stderr, autocompleteUsage)
					return ExitUsageError
				}
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}

			// Execute autocomplete request
			status, body, err := autocompleteRequest(client, token.Value, query, *limit, searchLat, searchLng, hasCenter)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}
			if status < 200 || status > 299 {
				fmt.Fprintf(stderr, "autocomplete failed with status %d\n", status)
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

			ok := writeAutocompleteTable(stdout, body, *limit)
			if !ok {
				fmt.Fprintln(stdout, string(body))
				return ExitSuccess
			}
			return ExitSuccess
		},
	}
}

func doAutocompleteRequest(
	client *httpclient.Client,
	token string,
	query string,
	limit int,
	centerLat, centerLng float64,
	hasCenter bool,
) (int, []byte, error) {
	params := url.Values{}
	params.Set("q", query)

	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	if hasCenter {
		params.Set("center", fmt.Sprintf("%.6f,%.6f", centerLat, centerLng))
	}

	req, err := client.NewRequest(http.MethodGet, "/v1/searchAutocomplete", params, nil)
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

func writeAutocompleteTable(w io.Writer, body []byte, limit int) bool {
	var resp AutocompleteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return false
	}

	if len(resp.Results) == 0 {
		fmt.Fprintln(w, "no suggestions")
		return true
	}

	table := tablewriter.NewWriter(w)
	table.Header("Suggestion", "Completion URL")

	rows := 0
	for _, result := range resp.Results {
		if rows >= limit {
			break
		}

		// Combine display lines for the suggestion
		suggestion := "-"
		if len(result.DisplayLines) > 0 {
			suggestion = strings.Join(result.DisplayLines, " | ")
		}

		completionURL := result.CompletionURL
		if completionURL == "" {
			completionURL = "-"
		}

		if err := table.Append([]string{suggestion, completionURL}); err != nil {
			return false
		}
		rows++
	}

	if rows == 0 {
		fmt.Fprintln(w, "no suggestions")
		return true
	}

	if err := table.Render(); err != nil {
		return false
	}
	return true
}
