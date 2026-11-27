package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/auth"
)

var timeNow = func() time.Time {
	return time.Now().UTC()
}

// #nosec G101 - This is documentation text, not hardcoded credentials
const authJwtUsage = `Usage:
  ams auth jwt --team-id <ID> --private-key <path> [--key-id <ID>] [--origin <domain>]
  ams auth jwt --env

Generate a JWT token using ES256 signing for Apple Maps Server API.

This is the modern authentication method that generates JWTs from your
private key (.p8 file) instead of manually creating them in the Apple
Developer Portal.

Required environment variables (or flags):
  AMS_TEAM_ID          Your Apple Developer Team ID (10 characters)
  AMS_PRIVATE_KEY_PATH Path to your .p8 private key file

Optional:
  AMS_KEY_ID           Key ID from Apple Developer Portal
  AMS_ORIGIN           Origin domain for the token (e.g., example.com)

Examples:
  # Using environment variables
  export AMS_TEAM_ID="ABCD123456"
  export AMS_PRIVATE_KEY_PATH="/path/to/AuthKey.p8"
  ams auth jwt --env

  # Using flags
  ams auth jwt --team-id "ABCD123456" --private-key "./AuthKey.p8"
  
  # With optional key ID and origin
  ams auth jwt --team-id "ABCD123456" --private-key "./AuthKey.p8" \
    --key-id "DEF123GHIJ" --origin "myapp.com"

  # Output to file
  ams auth jwt --env > jwt.txt
`

func NewAuthJWTCommand() Command {
	return Command{
		Name:      "auth jwt",
		UsageLine: "auth jwt [--team-id <ID> --private-key <path>] [--key-id <ID>] [--origin <domain>] [--env]",
		Summary:   "Generate ES256 JWT from private key",
		Usage:     authJwtUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			var options struct {
				TeamID     string `name:"team-id"`
				PrivateKey string `name:"private-key"`
				KeyID      string `name:"key-id"`
				Origin     string
				Env        bool
			}
			if err := parseOptions(args, &options); err != nil {
				if errors.Is(err, errHelpRequested) {
					_, _ = fmt.Fprint(stdout, authJwtUsage)
					return ExitSuccess
				}
				_, _ = fmt.Fprintln(stderr, err)
				_, _ = fmt.Fprint(stderr, authJwtUsage)
				return ExitUsageError
			}

			// Validate inputs
			if !options.Env {
				if options.TeamID == "" || options.PrivateKey == "" {
					_, _ = fmt.Fprintln(stderr, "error: --team-id and --private-key are required (or use --env)")
					_, _ = fmt.Fprint(stderr, authJwtUsage)
					return ExitUsageError
				}
			} else {
				// Load from environment
				options.TeamID = os.Getenv("AMS_TEAM_ID")
				options.PrivateKey = os.Getenv("AMS_PRIVATE_KEY_PATH")
				if options.KeyID == "" {
					options.KeyID = os.Getenv("AMS_KEY_ID")
				}
				if options.Origin == "" {
					options.Origin = os.Getenv("AMS_ORIGIN")
				}

				if options.TeamID == "" || options.PrivateKey == "" {
					_, _ = fmt.Fprintln(stderr, "error: AMS_TEAM_ID and AMS_PRIVATE_KEY_PATH environment variables must be set")
					_, _ = fmt.Fprint(stderr, authJwtUsage)
					return ExitUsageError
				}
			}

			// Validate team ID format (should be 10 characters)
			if len(options.TeamID) != 10 {
				_, _ = fmt.Fprintf(stderr, "warning: Team ID should be 10 characters (got %d)\n", len(options.TeamID))
			}

			// Create JWT signer
			signer, err := auth.NewES256Signer(options.PrivateKey, options.TeamID)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
				return ExitRuntimeError
			}

			if options.KeyID != "" {
				signer.SetKeyID(options.KeyID)
			}
			if options.Origin != "" {
				signer.SetOrigin(options.Origin)
			}

			// Generate JWT
			jwt, err := signer.GenerateJWT(timeNow())
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "error generating JWT: %v\n", err)
				return ExitRuntimeError
			}

			// Output the JWT
			_, _ = fmt.Fprintln(stdout, jwt)

			// Print info to stderr
			_, _ = fmt.Fprintln(stderr, "\nJWT generated successfully!")
			_, _ = fmt.Fprintln(stderr, "This JWT is valid for 7 days (Apple's maximum).")
			_, _ = fmt.Fprintln(stderr, "\nTo use this JWT:")
			_, _ = fmt.Fprintln(stderr, "  export AMS_MAPS_TOKEN=$(ams auth jwt --env)")
			_, _ = fmt.Fprintln(stderr, "  ams auth token  # Exchange for access token")

			return ExitSuccess
		},
	}
}
