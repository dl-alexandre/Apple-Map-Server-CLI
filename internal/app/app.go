package app

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/kong"
	appcli "github.com/dl-alexandre/Apple-Map-Server-CLI/internal/cli"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/commands"
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/version"
)

// CLI defines the complete ams command tree. Kong owns command selection,
// positional arguments, flag parsing, and generated help for the CLI.
type CLI struct {
	Auth        AuthCLI        `cmd:"" help:"Manage Apple Maps authentication"`
	Geocode     GeocodeCLI     `cmd:"" help:"Geocode an address"`
	Reverse     ReverseCLI     `cmd:"" help:"Reverse geocode coordinates"`
	Directions  DirectionsCLI  `cmd:"" help:"Get directions between locations"`
	Search      SearchCLI      `cmd:"" help:"Search for places"`
	Cache       CacheCLI       `cmd:"" help:"Manage the geocode cache"`
	Snapshot    SnapshotCLI    `cmd:"" help:"Generate a static map image"`
	Unified     UnifiedCLI     `cmd:"" help:"Search and generate a snapshot"`
	Config      ConfigCLI      `cmd:"" help:"Manage CLI configuration"`
	Ping        PingCLI        `cmd:"" help:"Check API connectivity"`
	CheckUpdate CheckUpdateCLI `cmd:"check-update" help:"Check for available updates"`
	Version     VersionCLI     `cmd:"" help:"Show version information"`
	Help        HelpCLI        `cmd:"" help:"Show help for a command"`
}

type AuthCLI struct {
	Token AuthTokenCLI `cmd:"" help:"Exchange a Maps token for an access token"`
	Check AuthCheckCLI `cmd:"" help:"Check token status and expiry"`
	JWT   AuthJWTCLI   `cmd:"" help:"Generate an ES256 JWT from a private key"`
}

type AuthTokenCLI struct {
	Raw  bool `help:"Output token only"`
	JSON bool `help:"Output token as JSON"`
}

type AuthCheckCLI struct {
	Token string `help:"Check this token instead of environment or config"`
}

type AuthJWTCLI struct {
	TeamID     string `name:"team-id" help:"Apple Developer Team ID"`
	PrivateKey string `name:"private-key" help:"Path to .p8 private key file"`
	KeyID      string `name:"key-id" help:"Key ID from Apple Developer Portal"`
	Origin     string `help:"Origin domain"`
	Env        bool   `help:"Use AMS_TEAM_ID and AMS_PRIVATE_KEY_PATH"`
}

type GeocodeCLI struct {
	JSON        bool     `help:"Output raw JSON response"`
	Limit       int      `default:"5" help:"Maximum number of results to display"`
	File        string   `help:"Path to file with one query per line"`
	Concurrency int      `default:"4" help:"Number of concurrent requests"`
	Query       []string `arg:"" optional:"" name:"address" help:"Address to geocode"`
}

type ReverseCLI struct {
	JSON  bool   `help:"Output raw JSON response"`
	Limit int    `default:"5" help:"Maximum number of results to display"`
	Point string `arg:"" name:"coordinates" help:"Coordinates in lat,lng form"`
}

type DirectionsCLI struct {
	JSON        bool   `help:"Output raw JSON response"`
	Mode        string `default:"car" help:"Transport mode"`
	ETA         bool   `help:"Show only ETA and distance summary"`
	Origin      string `arg:"" name:"origin" help:"Starting location"`
	Destination string `arg:"" name:"destination" help:"Destination location"`
}

type SearchCLI struct {
	Near        string   `help:"Center point for search as lat,lng"`
	Region      string   `help:"Bounding box as north,east,south,west"`
	NearAddress string   `name:"near-address" help:"Address to center the search around"`
	Limit       int      `default:"10" help:"Maximum number of results to return"`
	Category    string   `help:"Filter by POI category"`
	JSON        bool     `help:"Output raw JSON response"`
	NoCache     bool     `name:"no-cache" help:"Bypass cache for geocoding"`
	Query       []string `arg:"" optional:"" name:"query" help:"Search query"`
}

func (SearchCLI) Help() string {
	return "To request suggestions, use: ams search autocomplete <query>."
}

type AutocompleteCLI struct {
	Near  string   `help:"Center point for location bias as lat,lng"`
	Limit int      `default:"10" help:"Maximum number of suggestions to return"`
	JSON  bool     `help:"Output raw JSON response"`
	Query []string `arg:"" optional:"" name:"query" help:"Autocomplete query"`
}

type CacheCLI struct {
	Stats CacheActionCLI `cmd:"" help:"Show cache statistics"`
	Clear CacheActionCLI `cmd:"" help:"Clear cached entries"`
}

type CacheActionCLI struct{}

type SnapshotCLI struct {
	Zoom   int    `default:"12" help:"Map zoom level"`
	Size   string `default:"600x400" help:"Image dimensions"`
	Format string `default:"png" help:"Image format"`
	Output string `help:"Output image path"`
	Center string `arg:"" name:"center" help:"Location or coordinates"`
}

type UnifiedCLI struct {
	Near   string   `help:"Search center as coordinates"`
	Zoom   int      `default:"14" help:"Snapshot zoom level"`
	Output string   `help:"Output image path"`
	Query  []string `arg:"" optional:"" name:"query" help:"Place to search for"`
}

type ConfigCLI struct {
	Get  ConfigKeyCLI    `cmd:"" help:"Get a configuration value"`
	Set  ConfigSetCLI    `cmd:"" help:"Set a configuration value"`
	List ConfigActionCLI `cmd:"" help:"List configuration values"`
	Path ConfigActionCLI `cmd:"" help:"Show the config file path"`
}

type ConfigKeyCLI struct {
	Key string `arg:"" name:"key"`
}
type ConfigSetCLI struct {
	Key   string `arg:"" name:"key"`
	Value string `arg:"" name:"value"`
}
type ConfigActionCLI struct{}

type PingCLI struct {
	RequestID bool `name:"request-id" help:"Include request ID headers"`
}
type CheckUpdateCLI struct {
	Force bool `help:"Force check, bypassing the cache"`
	Clear bool `help:"Clear the update cache"`
}
type VersionCLI struct{}
type HelpCLI struct {
	Target []string `arg:"" optional:"" name:"command" help:"Command to show help for"`
}

// Run parses and executes one command. Exit status 2 is reserved for Kong
// parsing errors and command usage errors; command implementations report
// their established exit codes through the legacy command adapter.
func Run(args []string, stdout, stderr io.Writer) int {
	appcli.SetBuildInfo(version.Version, version.Commit, version.Date)
	commands.AutoUpdateCheck()

	cli := &CLI{}
	parser, err := newParser(cli, stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return commands.ExitUsageError
	}
	if len(args) == 0 {
		_, _ = parser.Parse([]string{"--help"})
		return commands.ExitSuccess
	}
	ctx, err := parser.Parse(normalizeLongFlags(normalizeSearchAutocompleteArgs(args)))
	if err != nil {
		if containsHelpFlag(args) {
			return commands.ExitSuccess
		}
		if len(args) > 0 && !knownTopLevelCommand(args[0]) {
			fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
			printRootUsage(stderr)
			return commands.ExitUsageError
		}
		fmt.Fprintln(stderr, err)
		printParseUsage(err, cli, stderr)
		return commands.ExitUsageError
	}
	if containsHelpFlag(args) {
		return commands.ExitSuccess
	}
	if len(args) == 1 && args[0] == "help" {
		_, _ = parser.Parse([]string{"--help"})
		return commands.ExitSuccess
	}
	return runCommand(ctx, cli, args, stdout, stderr)
}

func newParser(cli *CLI, stdout, stderr io.Writer) (*kong.Kong, error) {
	return kong.New(cli,
		kong.Name("ams"),
		kong.Description("Apple Maps Server API command line interface"),
		kong.Writers(stdout, stderr),
		kong.UsageOnError(),
		kong.Exit(func(int) {}),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
	)
}

func runCommand(ctx *kong.Context, cli *CLI, original []string, stdout, stderr io.Writer) int {
	// Kong handles command and option syntax. Existing business commands are
	// kept behind a small adapter while they are migrated independently.
	if len(original) == 0 || ctx.Command() == "" {
		_ = ctx.PrintUsage(false)
		return commands.ExitSuccess
	}
	name, args := legacyInvocation(commandName(ctx.Command()), cli)
	if name == "" {
		fmt.Fprintf(stderr, "unsupported command: %s\n", ctx.Command())
		return commands.ExitUsageError
	}
	if name == "search autocomplete" && containsAnyFlag(original, "region", "near-address", "category", "no-cache") {
		fmt.Fprintln(stderr, "unknown autocomplete flag")
		fmt.Fprint(stderr, commands.NewAutocompleteCommand().Usage)
		return commands.ExitUsageError
	}
	return commandByName(name).Run(args, stdout, stderr)
}

func containsAnyFlag(args []string, names ...string) bool {
	for _, arg := range args {
		arg = strings.TrimLeft(arg, "-")
		arg, _, _ = strings.Cut(arg, "=")
		for _, name := range names {
			if arg == name {
				return true
			}
		}
	}
	return false
}

func normalizeLongFlags(args []string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' && isLetter(arg[1]) {
			arg = "-" + arg
		}
		out[i] = arg
	}
	return out
}

// Search autocomplete is historically nested under search, so Kong sees the
// word "autocomplete" as the first query token. Move its option tokens ahead
// of that token to keep the variadic query positional compatible with the
// documented flag-before-query syntax.
func normalizeSearchAutocompleteArgs(args []string) []string {
	if len(args) < 2 || args[0] != "search" || args[1] != "autocomplete" {
		return args
	}
	flags := make([]string, 0, len(args)-2)
	positionals := make([]string, 0, len(args)-2)
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			name := strings.TrimLeft(arg, "-")
			if !strings.Contains(arg, "=") && name != "json" && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	result := []string{"search"}
	result = append(result, flags...)
	result = append(result, "autocomplete")
	return append(result, positionals...)
}

func isLetter(r byte) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

func commandName(command string) string {
	parts := strings.Fields(command)
	for i, part := range parts {
		if strings.HasPrefix(part, "<") || strings.HasPrefix(part, "[") {
			parts = parts[:i]
			break
		}
	}
	return strings.Join(parts, " ")
}

func containsHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-help" || arg == "-h" {
			return true
		}
	}
	return false
}

func knownTopLevelCommand(command string) bool {
	switch command {
	case "auth", "geocode", "reverse", "directions", "search", "cache", "snapshot", "unified", "config", "ping", "check-update", "version", "help":
		return true
	default:
		return false
	}
}

func printParseUsage(err error, cli *CLI, w io.Writer) {
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) && parseErr.Context != nil {
		name, _ := legacyInvocation(commandName(parseErr.Context.Command()), cli)
		if name != "" {
			if usage := commandByName(name).Usage; usage != "" {
				_, _ = fmt.Fprint(w, usage)
				return
			}
		}
	}
	printRootUsage(w)
}

func printRootUsage(w io.Writer) {
	parser, err := newParser(&CLI{}, w, w)
	if err == nil {
		_, _ = parser.Parse([]string{"--help"})
	}
}

func legacyInvocation(command string, c *CLI) (string, []string) {
	flag := func(name, value string) []string { return []string{"--" + name, value} }
	boolFlag := func(name string, on bool) []string {
		if on {
			return []string{"--" + name}
		}
		return nil
	}
	intFlag := func(name string, value int) []string { return []string{"--" + name, fmt.Sprint(value)} }
	switch command {
	case "auth token":
		return "auth token", append(boolFlag("raw", c.Auth.Token.Raw), boolFlag("json", c.Auth.Token.JSON)...)
	case "auth check":
		if c.Auth.Check.Token != "" {
			return "auth check", flag("token", c.Auth.Check.Token)
		}
		return "auth check", nil
	case "auth jwt":
		args := append(flag("team-id", c.Auth.JWT.TeamID), flag("private-key", c.Auth.JWT.PrivateKey)...)
		args = append(args, flag("key-id", c.Auth.JWT.KeyID)...)
		args = append(args, flag("origin", c.Auth.JWT.Origin)...)
		return "auth jwt", append(args, boolFlag("env", c.Auth.JWT.Env)...)
	case "geocode":
		args := append(boolFlag("json", c.Geocode.JSON), intFlag("limit", c.Geocode.Limit)...)
		args = append(args, flag("file", c.Geocode.File)...)
		args = append(args, intFlag("concurrency", c.Geocode.Concurrency)...)
		return "geocode", append(args, c.Geocode.Query...)
	case "reverse":
		args := append(boolFlag("json", c.Reverse.JSON), intFlag("limit", c.Reverse.Limit)...)
		return "reverse", append(args, c.Reverse.Point)
	case "directions":
		args := append(boolFlag("json", c.Directions.JSON), flag("mode", c.Directions.Mode)...)
		args = append(args, boolFlag("eta", c.Directions.ETA)...)
		return "directions", append(args, c.Directions.Origin, c.Directions.Destination)
	case "search":
		if len(c.Search.Query) > 0 && c.Search.Query[0] == "autocomplete" {
			args := append(flag("near", c.Search.Near), intFlag("limit", c.Search.Limit)...)
			args = append(args, boolFlag("json", c.Search.JSON)...)
			return "search autocomplete", append(args, c.Search.Query[1:]...)
		}
		args := append(flag("near", c.Search.Near), flag("region", c.Search.Region)...)
		args = append(args, flag("near-address", c.Search.NearAddress)...)
		args = append(args, intFlag("limit", c.Search.Limit)...)
		args = append(args, flag("category", c.Search.Category)...)
		args = append(args, boolFlag("json", c.Search.JSON)...)
		args = append(args, boolFlag("no-cache", c.Search.NoCache)...)
		return "search", append(args, c.Search.Query...)
	case "cache stats":
		return "cache", []string{"stats"}
	case "cache clear":
		return "cache", []string{"clear"}
	case "snapshot":
		args := append([]string{c.Snapshot.Center}, intFlag("zoom", c.Snapshot.Zoom)...)
		args = append(args, flag("size", c.Snapshot.Size)...)
		args = append(args, flag("format", c.Snapshot.Format)...)
		return "snapshot", append(args, flag("output", c.Snapshot.Output)...)
	case "unified":
		args := append([]string(nil), c.Unified.Query...)
		args = append(args, flag("near", c.Unified.Near)...)
		args = append(args, intFlag("zoom", c.Unified.Zoom)...)
		args = append(args, flag("output", c.Unified.Output)...)
		return "unified", args
	case "config get":
		return "config", []string{"get", c.Config.Get.Key}
	case "config set":
		return "config", []string{"set", c.Config.Set.Key, c.Config.Set.Value}
	case "config list":
		return "config", []string{"list"}
	case "config path":
		return "config", []string{"path"}
	case "ping":
		return "ping", boolFlag("request-id", c.Ping.RequestID)
	case "check-update":
		args := append(boolFlag("force", c.CheckUpdate.Force), boolFlag("clear", c.CheckUpdate.Clear)...)
		return "check-update", args
	case "version":
		return "version", nil
	case "help":
		return "help", c.Help.Target
	default:
		return "", nil
	}
}

func commandByName(name string) commands.Command {
	all := []commands.Command{
		commands.NewAuthTokenCommand(), commands.NewAuthCheckCommand(), commands.NewAuthJWTCommand(),
		commands.NewGeocodeCommand(), commands.NewReverseCommand(), commands.NewDirectionsCommand(),
		commands.NewSearchCommand(), commands.NewAutocompleteCommand(), commands.NewCacheCommand(),
		commands.NewSnapshotCommand(), commands.NewUnifiedCommand(), commands.NewConfigCommand(),
		commands.NewPingCommand(), commands.NewCheckUpdateCommand(), commands.NewVersionCommand(),
	}
	for _, cmd := range all {
		if cmd.Name == name {
			return cmd
		}
	}
	if name == "help" {
		lookup := make(map[string]commands.Command, len(all))
		for _, cmd := range all {
			lookup[cmd.Name] = cmd
		}
		return commands.NewHelpCommand(func(w io.Writer) { _, _ = fmt.Fprint(w, "Use 'ams --help' to list commands.\n") }, func(n string) (commands.Command, bool) { cmd, ok := lookup[n]; return cmd, ok })
	}
	return commands.Command{Run: func([]string, io.Writer, io.Writer) int { return commands.ExitUsageError }}
}

// ParseForTest exposes the same Kong tree used by Run so command/flag parsing
// can be tested without making network calls or executing a command.
func ParseForTest(args []string) (*kong.Context, *CLI, error) {
	cli := &CLI{}
	parser, err := newParser(cli, io.Discard, io.Discard)
	if err != nil {
		return nil, nil, err
	}
	ctx, err := parser.Parse(normalizeLongFlags(normalizeSearchAutocompleteArgs(args)))
	return ctx, cli, err
}
