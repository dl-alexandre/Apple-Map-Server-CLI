package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/httpclient"
)

const pingUsage = `Usage:
  ams ping [--request-id]

Ping the Apple Map Server.
`

var newHTTPClient = httpclient.New

func NewPingCommand() Command {
	return Command{
		Name:      "ping",
		UsageLine: "ping [--request-id]",
		Summary:   "Ping the Apple Map Server",
		Usage:     pingUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			fs := flag.NewFlagSet("ping", flag.ContinueOnError)
			requestID := fs.Bool("request-id", false, "Include request ID headers")
			fs.SetOutput(io.Discard)
			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, pingUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, pingUsage)
				return ExitUsageError
			}

			if fs.NArg() != 0 {
				fmt.Fprintln(stderr, "ping accepts no arguments")
				fmt.Fprint(stderr, pingUsage)
				return ExitUsageError
			}

			client, err := newHTTPClient()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, pingUsage)
				return ExitUsageError
			}

			cfg, err := auth.LoadConfigFromEnv()
			if err != nil {
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, pingUsage)
				return ExitUsageError
			}
			fmt.Fprintln(stdout, "auth ok")

			now := time.Now().UTC()
			token, _, err := accessTokenProvider(cfg, client, now)
			if err != nil {
				if auth.IsMissingEnv(err) {
					fmt.Fprintln(stderr, err)
					fmt.Fprint(stderr, pingUsage)
					return ExitUsageError
				}
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}
			fmt.Fprintln(stdout, "token ok")

			params := url.Values{}
			params.Set("q", "Cupertino")
			params.Set("limit", "1")
			req, err := client.NewRequest(http.MethodGet, "/v1/geocode", params, nil)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}
			req.Header.Set("Authorization", "Bearer "+token.Value)
			req.Header.Set("User-Agent", userAgent())

			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return ExitRuntimeError
			}
			defer resp.Body.Close()

			fmt.Fprintf(stdout, "status %d\n", resp.StatusCode)
			if *requestID {
				for _, requestID := range httpclient.RequestIDs(resp.Header) {
					fmt.Fprintf(stdout, "request_id %s\n", requestID)
				}
			}

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return ExitRuntimeError
			}

			return ExitSuccess
		},
	}
}
