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
	"os"
	"strings"
	"sync"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

const geocodeUsage = `Usage:
  ams geocode [--json] [--limit N] [--file <path>] [--concurrency N] <address>

Geocode an address.
`

var geocodeRequest = doGeocodeRequest
var progressEnabled = defaultProgressEnabled

func NewGeocodeCommand() Command {
	return Command{
		Name:      "geocode",
		UsageLine: "geocode [--json] [--limit N] [--file <path>] [--concurrency N] <address>",
		Summary:   "Geocode an address",
		Usage:     geocodeUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			fs := flag.NewFlagSet("geocode", flag.ContinueOnError)
			jsonOut := fs.Bool("json", false, "Output raw JSON response")
			limit := fs.Int("limit", 5, "Maximum number of results to display")
			filePath := fs.String("file", "", "Path to file with one query per line")
			concurrency := fs.Int("concurrency", 4, "Number of concurrent requests")
			fs.SetOutput(io.Discard)
			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, geocodeUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, geocodeUsage)
				return ExitUsageError
			}

			if *limit < 1 {
				fmt.Fprintln(stderr, "limit must be at least 1")
				fmt.Fprint(stderr, geocodeUsage)
				return ExitUsageError
			}

			if *concurrency < 1 {
				fmt.Fprintln(stderr, "concurrency must be at least 1")
				fmt.Fprint(stderr, geocodeUsage)
				return ExitUsageError
			}

			if *filePath != "" && fs.NArg() > 0 {
				fmt.Fprintln(stderr, "geocode does not accept address arguments when --file is set")
				fmt.Fprint(stderr, geocodeUsage)
				return ExitUsageError
			}

			if *filePath == "" && fs.NArg() == 0 {
				fmt.Fprintln(stderr, "geocode requires an address")
				fmt.Fprint(stderr, geocodeUsage)
				return ExitUsageError
			}

			client, err := newHTTPClient()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, geocodeUsage)
				return ExitUsageError
			}

			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, geocodeUsage)
				return ExitUsageError
			}

			// Print token expiry warning
			fmt.Fprint(stderr, TokenExpiryWarning)

			now := nowFunc().UTC()
			token, _, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					fmt.Fprintln(stderr, err)
					fmt.Fprint(stderr, geocodeUsage)
					return ExitUsageError
				}
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}

			if *filePath != "" {
				return runGeocodeBatch(*filePath, *limit, *concurrency, *jsonOut, token.Value, client, stdout, stderr)
			}

			query := strings.Join(fs.Args(), " ")
			status, body, err := geocodeRequest(client, token.Value, query)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}
			if status < 200 || status > 299 {
				fmt.Fprintf(stderr, "geocode failed with status %d\n", status)
				if len(body) > 0 {
					fmt.Fprintln(stderr, string(body))
				}
				return ExitRuntimeError
			}

			if *jsonOut {
				pretty, ok := formatJSON(body)
				if ok {
					fmt.Fprintln(stdout, pretty)
					return ExitSuccess
				}
				fmt.Fprintln(stdout, string(body))
				return ExitSuccess
			}

			ok := writeGeocodeTable(stdout, body, *limit)
			if !ok {
				fmt.Fprintln(stdout, string(body))
				return ExitSuccess
			}
			return ExitSuccess
		},
	}
}

func runGeocodeBatch(path string, limit int, concurrency int, jsonOut bool, token string, client *httpclient.Client, stdout, stderr io.Writer) int {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprint(stderr, geocodeUsage)
		return ExitUsageError
	}
	defer file.Close()

	queries, err := readQueries(file)
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprint(stderr, geocodeUsage)
		return ExitUsageError
	}

	if len(queries) == 0 {
		return ExitSuccess
	}

	var bar *progressbar.ProgressBar
	if progressEnabled(jsonOut, stdout, stderr) {
		bar = progressbar.NewOptions(len(queries),
			progressbar.OptionSetWriter(stderr),
			progressbar.OptionSetDescription("geocode"),
			progressbar.OptionShowCount(),
			progressbar.OptionClearOnFinish(),
		)
	}

	jobs := make(chan geocodeJob)
	results := make(chan geocodeResult)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				status, body, reqErr := geocodeRequest(client, token, job.query)
				results <- geocodeResult{idx: job.idx, query: job.query, status: status, body: body, err: reqErr}
			}
		}()
	}

	go func() {
		for idx, query := range queries {
			jobs <- geocodeJob{idx: idx, query: query}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	collected := make([]geocodeResult, len(queries))
	for res := range results {
		collected[res.idx] = res
		if bar != nil {
			_ = bar.Add(1)
		}
	}

	if bar != nil {
		_ = bar.Finish()
	}

	hasFailure := false
	for _, res := range collected {
		if jsonOut {
			if res.err != nil {
				hasFailure = true
				_ = writeJSONLine(stdout, res.query, nil, res.err.Error(), 0)
				continue
			}
			if res.status < 200 || res.status > 299 {
				hasFailure = true
				_ = writeJSONLine(stdout, res.query, nil, fmt.Sprintf("status %d", res.status), res.status)
				continue
			}
			_ = writeJSONLine(stdout, res.query, res.body, "", res.status)
			continue
		}

		fmt.Fprintf(stdout, "input: %s\n", res.query)
		if res.err != nil {
			hasFailure = true
			fmt.Fprintf(stdout, "error: %s\n", res.err)
			continue
		}
		if res.status < 200 || res.status > 299 {
			hasFailure = true
			fmt.Fprintf(stdout, "error: status %d\n", res.status)
			continue
		}
		if !writeGeocodeTable(stdout, res.body, limit) {
			fmt.Fprintln(stdout, string(res.body))
		}
	}

	if hasFailure {
		return ExitRuntimeError
	}
	return ExitSuccess
}

func doGeocodeRequest(client *httpclient.Client, token, query string) (int, []byte, error) {
	params := url.Values{}
	params.Set("q", query)
	if params.Get("q") == "" {
		return 0, nil, errors.New("query is empty")
	}
	req, err := client.NewRequest(http.MethodGet, "/v1/geocode", params, nil)
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

type geocodeJob struct {
	idx   int
	query string
}

type geocodeResult struct {
	idx    int
	query  string
	status int
	body   []byte
	err    error
}

func writeJSONLine(w io.Writer, input string, result []byte, errText string, status int) error {
	payload := map[string]any{
		"input": input,
	}
	if errText != "" {
		payload["error"] = errText
		if status != 0 {
			payload["status"] = status
		}
	} else if len(result) > 0 {
		payload["result"] = json.RawMessage(result)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func defaultProgressEnabled(jsonOut bool, stdout, stderr io.Writer) bool {
	if jsonOut && !isTerminalWriter(stdout) {
		return false
	}
	return isTerminalWriter(stderr)
}

func isTerminalWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func formatJSON(body []byte) (string, bool) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		return "", false
	}
	return buf.String(), true
}

func writeGeocodeTable(w io.Writer, body []byte, limit int) bool {
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

func pickString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if str, ok := value.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

func coordinateValues(payload map[string]any) (float64, float64, bool) {
	coordAny, ok := payload["coordinate"]
	if !ok {
		return 0, 0, false
	}

	coordMap, ok := coordAny.(map[string]any)
	if !ok {
		return 0, 0, false
	}

	lat, okLat := coordMap["latitude"].(float64)
	lon, okLon := coordMap["longitude"].(float64)
	if !okLat || !okLon {
		return 0, 0, false
	}

	return lat, lon, true
}
