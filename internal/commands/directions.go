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
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
	"github.com/olekukonko/tablewriter"
)

const directionsUsage = `Usage:
  ams directions <origin> <destination> [--mode <transport>] [--eta] [--json]

Get directions between two locations.

Transport modes: car, walk, transit, bike (default: car)
Origin/destination can be:
  - Coordinates: "37.7857,-122.4011"
  - Address: "1 Infinite Loop, Cupertino, CA"

Examples:
  ams directions "37.7857,-122.4011" "San Francisco City Hall, CA"
  ams directions "1 Infinite Loop, Cupertino, CA" "Palo Alto, CA" --mode walk
  ams directions "SF" "LA" --eta
`

var directionsRequest = doDirectionsRequest

func NewDirectionsCommand() Command {
	return Command{
		Name:      "directions",
		UsageLine: "directions <origin> <destination> [--mode <transport>] [--eta] [--json]",
		Summary:   "Get directions between locations",
		Usage:     directionsUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			fs := flag.NewFlagSet("directions", flag.ContinueOnError)
			jsonOut := fs.Bool("json", false, "Output raw JSON response")
			mode := fs.String("mode", "car", "Transport mode: car, walk, transit, bike")
			etaOnly := fs.Bool("eta", false, "Show only ETA and distance summary")
			fs.SetOutput(io.Discard)
			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, directionsUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, directionsUsage)
				return ExitUsageError
			}

			if fs.NArg() != 2 {
				fmt.Fprintln(stderr, "directions requires origin and destination")
				fmt.Fprint(stderr, directionsUsage)
				return ExitUsageError
			}

			transportMode := normalizeTransportMode(*mode)
			if transportMode == "" {
				fmt.Fprintf(stderr, "invalid transport mode: %s\n", *mode)
				fmt.Fprint(stderr, directionsUsage)
				return ExitUsageError
			}

			origin := fs.Arg(0)
			destination := fs.Arg(1)

			client, err := newHTTPClient()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, directionsUsage)
				return ExitUsageError
			}

			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, directionsUsage)
				return ExitUsageError
			}

			// Print token expiry warning
			fmt.Fprint(stderr, TokenExpiryWarning)

			now := nowFunc().UTC()
			token, _, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					fmt.Fprintln(stderr, err)
					fmt.Fprint(stderr, directionsUsage)
					return ExitUsageError
				}
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}

			status, body, err := directionsRequest(client, token.Value, origin, destination, transportMode)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}
			if status < 200 || status > 299 {
				fmt.Fprintf(stderr, "directions failed with status %d\n", status)
				if len(body) > 0 {
					fmt.Fprintln(stderr, string(body))
				}
				return ExitRuntimeError
			}

			if *jsonOut {
				pretty, ok := formatDirectionsJSON(body)
				if ok {
					fmt.Fprintln(stdout, pretty)
					return ExitSuccess
				}
				fmt.Fprintln(stdout, string(body))
				return ExitSuccess
			}

			if *etaOnly {
				ok := writeETASummary(stdout, body)
				if !ok {
					fmt.Fprintln(stdout, string(body))
				}
				return ExitSuccess
			}

			ok := writeDirectionsTable(stdout, body)
			if !ok {
				fmt.Fprintln(stdout, string(body))
			}
			return ExitSuccess
		},
	}
}

func normalizeTransportMode(mode string) string {
	switch strings.ToLower(mode) {
	case "car", "automobile", "auto", "driving":
		return "car"
	case "walk", "walking", "foot":
		return "walk"
	case "transit", "public", "bus", "train", "public_transport":
		return "transit"
	case "bike", "bicycle", "cycling":
		return "bike"
	default:
		return ""
	}
}

func doDirectionsRequest(client *httpclient.Client, token, origin, destination, mode string) (int, []byte, error) {
	params := url.Values{}
	params.Set("origin", origin)
	params.Set("destination", destination)
	if mode != "" && mode != "car" {
		params.Set("mode", mode)
	}

	req, err := client.NewRequest(http.MethodGet, "/v1/directions", params, nil)
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

func formatDirectionsJSON(body []byte) (string, bool) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		return "", false
	}
	return buf.String(), true
}

func writeETASummary(w io.Writer, body []byte) bool {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}

	routesAny, ok := payload["routes"]
	if !ok {
		return false
	}

	routes, ok := routesAny.([]any)
	if !ok || len(routes) == 0 {
		fmt.Fprintln(w, "No routes found")
		return true
	}

	// Get the first (best) route
	route, ok := routes[0].(map[string]any)
	if !ok {
		return false
	}

	distanceMeters, _ := route["distanceMeters"].(float64)
	durationSeconds, _ := route["durationSeconds"].(float64)
	transportType, _ := route["transportType"].(string)
	hasTolls, _ := route["hasTolls"].(bool)

	if distanceMeters == 0 && durationSeconds == 0 {
		fmt.Fprintln(w, "No ETA data available")
		return true
	}

	// Format distance
	distanceStr := formatDistance(distanceMeters)

	// Format duration
	durationStr := formatDuration(durationSeconds)

	fmt.Fprintf(w, "Distance: %s\n", distanceStr)
	fmt.Fprintf(w, "Duration: %s\n", durationStr)
	if transportType != "" {
		fmt.Fprintf(w, "Mode: %s\n", transportType)
	}
	if hasTolls {
		fmt.Fprintln(w, "Note: Route includes tolls")
	}

	return true
}

func formatDistance(meters float64) string {
	if meters < 1000 {
		return fmt.Sprintf("%.0f m", meters)
	}
	km := meters / 1000
	if km < 10 {
		return fmt.Sprintf("%.1f km", km)
	}
	return fmt.Sprintf("%.0f km", km)
}

func formatDuration(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%d hr %d min", hours, minutes)
		}
		return fmt.Sprintf("%d hr", hours)
	}
	return fmt.Sprintf("%d min", minutes)
}

func writeDirectionsTable(w io.Writer, body []byte) bool {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}

	// Print origin/destination if available
	if dest, ok := payload["destination"].(map[string]any); ok {
		if name, ok := dest["name"].(string); ok && name != "" {
			fmt.Fprintf(w, "Destination: %s\n", name)
		}
	}

	routesAny, ok := payload["routes"]
	if !ok {
		return false
	}

	routes, ok := routesAny.([]any)
	if !ok || len(routes) == 0 {
		fmt.Fprintln(w, "No routes found")
		return true
	}

	stepsAny, ok := payload["steps"]
	if !ok {
		return false
	}

	steps, ok := stepsAny.([]any)
	if !ok {
		return false
	}

	// Get the first route for summary
	route, ok := routes[0].(map[string]any)
	if !ok {
		return false
	}

	distanceMeters, _ := route["distanceMeters"].(float64)
	durationSeconds, _ := route["durationSeconds"].(float64)
	transportType, _ := route["transportType"].(string)

	fmt.Fprintf(w, "\nRoute Summary (%s): %s, %s\n\n",
		transportType,
		formatDistance(distanceMeters),
		formatDuration(durationSeconds))

	// Write steps table
	table := tablewriter.NewWriter(w)
	table.Header("Step", "Instruction", "Distance")

	for i, stepAny := range steps {
		step, ok := stepAny.(map[string]any)
		if !ok {
			continue
		}

		instruction, _ := step["instructions"].(string)
		if instruction == "" {
			continue // Skip steps without instructions (usually the starting point)
		}

		stepDistance, _ := step["distanceMeters"].(float64)
		distanceStr := formatDistance(stepDistance)

		stepNum := strconv.Itoa(i)
		if err := table.Append([]string{stepNum, instruction, distanceStr}); err != nil {
			return false
		}
	}

	if err := table.Render(); err != nil {
		return false
	}

	return true
}
