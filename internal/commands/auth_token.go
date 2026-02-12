package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

const authTokenUsage = `Usage:
  ams auth token [--raw|--json]

Exchange an Apple Maps Server API access token using a Maps token.
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
			fs := flag.NewFlagSet("auth token", flag.ContinueOnError)
			rawOut := fs.Bool("raw", false, "Output token only")
			jsonOut := fs.Bool("json", false, "Output token as JSON")
			fs.SetOutput(io.Discard)
			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, authTokenUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			if *rawOut && *jsonOut {
				fmt.Fprintln(stderr, "raw and json cannot be used together")
				fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			if fs.NArg() != 0 {
				fmt.Fprintln(stderr, "auth token accepts no arguments")
				fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			client, err := httpclient.New()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, authTokenUsage)
				return ExitUsageError
			}

			now := nowFunc().UTC()
			token, source, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					fmt.Fprintln(stderr, err)
					fmt.Fprint(stderr, authTokenUsage)
					return ExitUsageError
				}
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}

			if *rawOut {
				fmt.Fprintln(stdout, token.Value)
				return ExitSuccess
			}

			if *jsonOut {
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
					fmt.Fprintln(stderr, err)
					return ExitRuntimeError
				}
				fmt.Fprintln(stdout, string(data))
				return ExitSuccess
			}

			fmt.Fprintf(stdout, "maps_token_present %t\n", strings.TrimSpace(cfg.MapsToken) != "")
			if !token.ExpiresAt.IsZero() {
				fmt.Fprintf(stdout, "access_token_expires_at %s\n", token.ExpiresAt.UTC().Format(time.RFC3339))
			} else if token.ExpiresIn > 0 {
				fmt.Fprintf(stdout, "access_token_expires_in %ds\n", token.ExpiresIn)
			}
			fmt.Fprintf(stdout, "source %s\n", source)
			return ExitSuccess
		},
	}
}
