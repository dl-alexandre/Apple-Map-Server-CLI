package commands

import (
	"fmt"
	"io"
	"sort"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/config"
)

const configUsage = `Usage:
  ams config get <key>          Get a configuration value
  ams config set <key> <value>  Set a configuration value
  ams config list               List all configuration values
  ams config path               Show the config file path

Configuration keys:
  maps_token       - Apple Maps API token (⚠️ expires every 7 days)
  team_id          - Apple Developer Team ID
  key_id           - Maps Key ID
  private_key      - Private key content (or use private_key_path)
  private_key_path - Path to private key file
  default_zoom     - Default zoom level for snapshots (1-20)
  default_size     - Default snapshot dimensions (e.g., "800x600")
  default_format   - Default snapshot format (png, jpg)
  default_map_type - Default map style (standard, hybrid, satellite)
  default_limit    - Default search result limit
  default_category - Default POI category filter
  base_url         - API base URL (override for testing)

Examples:
  ams config set team_id ABC123XYZ
  ams config set default_zoom 15
  ams config get team_id
  ams config list

Note: Secrets (maps_token, private_key) are stored in ~/.config/ams/config.yaml
with 0600 permissions. Never commit this file to version control.
`

func NewConfigCommand() Command {
	return Command{
		Name:      "config",
		UsageLine: "config <get|set|list|path> [args...]",
		Summary:   "Manage CLI configuration",
		Usage:     configUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			if len(args) == 0 {
				fmt.Fprintln(stderr, "config requires a subcommand: get, set, list, or path")
				fmt.Fprint(stderr, configUsage)
				return ExitUsageError
			}

			subcommand := args[0]
			switch subcommand {
			case "get":
				return runConfigGet(args[1:], stdout, stderr)
			case "set":
				return runConfigSet(args[1:], stdout, stderr)
			case "list":
				return runConfigList(stdout, stderr)
			case "path":
				return runConfigPath(stdout, stderr)
			default:
				fmt.Fprintf(stderr, "unknown config subcommand: %s\n", subcommand)
				fmt.Fprint(stderr, configUsage)
				return ExitUsageError
			}
		},
	}
}

func runConfigGet(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "config get requires a key name")
		fmt.Fprint(stderr, configUsage)
		return ExitUsageError
	}

	key := args[0]

	// Load current config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "error loading config: %v\n", err)
		return ExitRuntimeError
	}

	value, err := cfg.Get(key)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitUsageError
	}

	// Mask sensitive values
	if key == "maps_token" && value != "" {
		value = maskValue(value)
	}
	if key == "private_key" && value != "" {
		value = maskValue(value)
	}

	fmt.Fprintln(stdout, value)
	return ExitSuccess
}

func runConfigSet(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "config set requires a key and value")
		fmt.Fprint(stderr, configUsage)
		return ExitUsageError
	}

	key := args[0]
	value := args[1]

	// Load current config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "error loading config: %v\n", err)
		return ExitRuntimeError
	}

	// Set the value
	if err := cfg.Set(key, value); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitUsageError
	}

	// Save the config
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(stderr, "error saving config: %v\n", err)
		return ExitRuntimeError
	}

	fmt.Fprintf(stdout, "✓ Set %s = %s\n", key, maskIfSensitive(key, value))
	return ExitSuccess
}

func runConfigList(stdout, stderr io.Writer) int {
	// Load current config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "error loading config: %v\n", err)
		return ExitRuntimeError
	}

	values := cfg.List()

	// Sort keys for consistent output
	var keys []string
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Print header
	fmt.Fprintln(stdout, "Configuration:")
	fmt.Fprintln(stdout)

	// Print values
	for _, k := range keys {
		v := values[k]
		if v == "" {
			v = "(not set)"
		} else {
			v = maskIfSensitive(k, v)
		}
		fmt.Fprintf(stdout, "  %s = %s\n", k, v)
	}

	// Print config file path
	path, _ := config.ConfigFilePath()
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Config file: %s\n", path)

	return ExitSuccess
}

func runConfigPath(stdout, stderr io.Writer) int {
	path, err := config.ConfigFilePath()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitRuntimeError
	}

	fmt.Fprintln(stdout, path)
	return ExitSuccess
}

func maskValue(value string) string {
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func maskIfSensitive(key, value string) string {
	sensitiveKeys := map[string]bool{
		"maps_token":  true,
		"private_key": true,
	}
	if sensitiveKeys[key] && value != "" {
		return maskValue(value)
	}
	return value
}
