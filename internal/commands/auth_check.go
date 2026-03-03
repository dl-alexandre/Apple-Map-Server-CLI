package commands

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/config"
)

const authCheckUsage = `Usage:
  ams auth check [--token <token>]

Check the status and expiry of your Apple Maps API token.

The command decodes the JWT payload to show:
- Token expiry date and time
- Time remaining until expiry
- Issuer and other claims
- Whether the token is still valid

Examples:
  ams auth check                           # Check token from env/config
  ams auth check --token <jwt-token>       # Check a specific token
`

func NewAuthCheckCommand() Command {
	return Command{
		Name:      "auth check",
		UsageLine: "auth check [--token <token>]",
		Summary:   "Check token status and expiry",
		Usage:     authCheckUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			var token string

			// Parse flags
			for i := 0; i < len(args); i++ {
				if args[i] == "--token" && i+1 < len(args) {
					token = args[i+1]
					i++
				}
			}

			// If no token provided, load from environment or config
			if token == "" {
				// Try environment first
				token = os.Getenv("AMS_MAPS_TOKEN")

				// Try config file
				if token == "" {
					cfg, err := config.Load()
					if err == nil && cfg.MapsToken != "" {
						token = cfg.MapsToken
					}
				}
			}

			if token == "" {
				fmt.Fprintln(stderr, "error: No token found.")
				fmt.Fprintln(stderr, "Set AMS_MAPS_TOKEN environment variable or use 'ams config set maps_token <token>'")
				fmt.Fprintln(stderr, "Or pass token directly with --token flag")
				return ExitUsageError
			}

			// Parse the token
			parser := auth.NewJWTParser()
			claims, err := parser.Parse(token)
			if err != nil {
				fmt.Fprintf(stderr, "error: Failed to parse token: %v\n", err)
				return ExitRuntimeError
			}

			// Display token information
			fmt.Fprintln(stdout, "Token Information:")
			fmt.Fprintln(stdout)

			if claims.Issuer != "" {
				fmt.Fprintf(stdout, "  Issuer (Team ID): %s\n", claims.Issuer)
			}

			if claims.Subject != "" {
				fmt.Fprintf(stdout, "  Subject:          %s\n", claims.Subject)
			}

			if claims.ID != "" {
				fmt.Fprintf(stdout, "  Token ID (jti):   %s\n", claims.ID)
			}

			if claims.Origin != "" {
				fmt.Fprintf(stdout, "  Origin:           %s\n", claims.Origin)
			}

			// Check expiry
			if claims.ExpiresAt > 0 {
				expiry := time.Unix(claims.ExpiresAt, 0)
				fmt.Fprintf(stdout, "  Expires At:       %s\n", expiry.Format(time.RFC1123))

				duration := time.Until(expiry)
				if duration > 0 {
					fmt.Fprintf(stdout, "  Time Remaining:   %s\n", auth.FormatDuration(duration))

					// Warning levels
					if duration < 24*time.Hour {
						fmt.Fprintln(stdout)
						fmt.Fprintln(stdout, "⚠️  WARNING: Token expires in less than 24 hours!")
						fmt.Fprintln(stdout, "    Generate a new token at: https://developer.apple.com/maps/server-api/")
					} else if duration < 48*time.Hour {
						fmt.Fprintln(stdout)
						fmt.Fprintln(stdout, "⚠️  NOTICE: Token expires in less than 48 hours")
					}
				} else {
					fmt.Fprintln(stdout)
					fmt.Fprintln(stdout, "❌ EXPIRED: Token has already expired!")
					fmt.Fprintln(stdout, "   Expired: "+auth.FormatDuration(-duration)+" ago")
					fmt.Fprintln(stdout, "   Generate a new token at: https://developer.apple.com/maps/server-api/")
					return ExitRuntimeError
				}
			}

			if claims.IssuedAt > 0 {
				issued := time.Unix(claims.IssuedAt, 0)
				fmt.Fprintf(stdout, "  Issued At:        %s\n", issued.Format(time.RFC1123))
			}

			if claims.NotBefore > 0 {
				notBefore := time.Unix(claims.NotBefore, 0)
				fmt.Fprintf(stdout, "  Not Before:       %s\n", notBefore.Format(time.RFC1123))
			}

			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, "✅ Token is valid")

			return ExitSuccess
		},
	}
}
