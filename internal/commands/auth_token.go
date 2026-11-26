package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

// #nosec G101 - This is documentation text, not hardcoded credentials
const authTokenUsage = `Usage:
  ams auth token [--raw|--json]

Exchange an Apple Maps Server API access token using a Maps token.

WARNING: Apple Maps Server API tokens expire every 7 days and must be
manually regenerated at https://developer.apple.com/maps/server-api/
`

var accessTokenProvider = auth.GetAccessToken
var nowFunc = time.Now

func NewAuthTokenCommand() Command {
	return Command{
		Name:      "auth token",
		UsageLine: "auth token [--raw|--json]",
		Summary:   "Exchange Maps token for access token",
		Usage:     authTokenUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			var options struct {
				Raw  bool `help:"Output token only"`
				JSON bool `help:"Output token as JSON"`
			}
			if err := parseOptions(args, &options); err != nil {
				if errors.Is(err, errHelpRequested) {
					_, _ = fmt.Fprint(stdout, authTokenUsage)
					return ExitSuccess
				}
				_, _ = fmt.Fprintln(stderr, err)
				_, _ = fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			if options.Raw && options.JSON {
				_, _ = fmt.Fprintln(stderr, "raw and json cannot be used together")
				_, _ = fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			client, err := httpclient.New()
			if err != nil {
				_, _ = fmt.Fprintln(stderr, err)
				_, _ = fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				_, _ = fmt.Fprintln(stderr, err)
				_, _ = fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			now := nowFunc().UTC()

			// Print token expiry warning before fetching
			fmt.Fprint(stderr, TokenExpiryWarning)

			token, source, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					_, _ = fmt.Fprintln(stderr, err)
					_, _ = fmt.Fprint(stderr, authTokenUsage)
					return ExitUsageError
				}
				_, _ = fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}

			if options.Raw {
				_, _ = fmt.Fprintln(stdout, token.Value)
				return ExitSuccess
			}

			if options.JSON {
				payload := map[string]any{
					"access_token":       token.Value,
					"maps_token_present": strings.TrimSpace(cfg.MapsToken) != "",
					"source":             string(source),
				}
				payload["expires_in"] = token.ExpiresIn
				if !token.ExpiresAt.IsZero() {
					payload["expires_at"] = token.ExpiresAt.UTC().Format(time.RFC3339)
				} else {
					payload["expires_at"] = ""
				}
				data, err := json.Marshal(payload)
				if err != nil {
					_, _ = fmt.Fprintln(stderr, err)
					return ExitRuntimeError
				}
				_, _ = fmt.Fprintln(stdout, string(data))
				return ExitSuccess
			}

			_, _ = fmt.Fprintf(stdout, "maps_token_present %t\n", strings.TrimSpace(cfg.MapsToken) != "")
			if !token.ExpiresAt.IsZero() {
				_, _ = fmt.Fprintf(stdout, "access_token_expires_at %s\n", token.ExpiresAt.UTC().Format(time.RFC3339))
			} else if token.ExpiresIn > 0 {
				_, _ = fmt.Fprintf(stdout, "access_token_expires_in %ds\n", token.ExpiresIn)
			}
			_, _ = fmt.Fprintf(stdout, "source %s\n", source)
			return ExitSuccess
		},
	}
}
