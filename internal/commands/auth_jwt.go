package commands

import (
	"errors"
	"flag"
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
			fs := flag.NewFlagSet("auth jwt", flag.ContinueOnError)
			teamID := fs.String("team-id", "", "Apple Developer Team ID (10 characters)")
			privateKey := fs.String("private-key", "", "Path to .p8 private key file")
			keyID := fs.String("key-id", "", "Key ID from Apple Developer Portal (optional)")
			origin := fs.String("origin", "", "Origin domain (optional)")
			useEnv := fs.Bool("env", false, "Use environment variables (AMS_TEAM_ID, AMS_PRIVATE_KEY_PATH)")
			fs.SetOutput(io.Discard)

			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					_, _ = fmt.Fprint(stdout, authJwtUsage)
					return ExitSuccess
				}
				_, _ = fmt.Fprintln(stderr, err)
				_, _ = fmt.Fprint(stderr, authJwtUsage)
				return ExitUsageError
			}

			// Validate inputs
			if !*useEnv {
				if *teamID == "" || *privateKey == "" {
					_, _ = fmt.Fprintln(stderr, "error: --team-id and --private-key are required (or use --env)")
					_, _ = fmt.Fprint(stderr, authJwtUsage)
					return ExitUsageError
				}
			} else {
				// Load from environment
				*teamID = os.Getenv("AMS_TEAM_ID")
				*privateKey = os.Getenv("AMS_PRIVATE_KEY_PATH")
				if *keyID == "" {
					*keyID = os.Getenv("AMS_KEY_ID")
				}
				if *origin == "" {
					*origin = os.Getenv("AMS_ORIGIN")
				}

				if *teamID == "" || *privateKey == "" {
					_, _ = fmt.Fprintln(stderr, "error: AMS_TEAM_ID and AMS_PRIVATE_KEY_PATH environment variables must be set")
					_, _ = fmt.Fprint(stderr, authJwtUsage)
					return ExitUsageError
				}
			}

			// Validate team ID format (should be 10 characters)
			if len(*teamID) != 10 {
				_, _ = fmt.Fprintf(stderr, "warning: Team ID should be 10 characters (got %d)\n", len(*teamID))
			}

			// Create JWT signer
			signer, err := auth.NewES256Signer(*privateKey, *teamID)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
				return ExitRuntimeError
			}

			if *keyID != "" {
				signer.SetKeyID(*keyID)
			}
			if *origin != "" {
				signer.SetOrigin(*origin)
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
