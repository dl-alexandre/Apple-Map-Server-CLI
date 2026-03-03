# Apple Map Server CLI (ams)

`ams` is a CLI tool for interacting with Apple Maps Server APIs.

**⚠️ IMPORTANT: Apple Maps Server API tokens expire every 7 days.** When your token expires, you must manually generate a new one at https://developer.apple.com/maps/server-api/

## 🚀 Quick Start

The easiest way to get started? Use the **unified** command—search for any place and instantly generate a map image:

```bash
# Install
$ go install github.com/dl-alexandre/Apple-Map-Server-CLI/cmd/ams@latest

# Set your token (get one from Apple Developer Portal)
$ export AMS_MAPS_TOKEN="your-token-here"

# Search for a place and generate a snapshot in one command!
$ ams unified "Golden Gate Bridge"
Found: Golden Gate Bridge (37.819900, -122.478300)
✓ Snapshot saved to: Golden_Gate_Bridge_1234567890.png

# Search near a location with custom zoom
$ ams unified "coffee shops" --near "37.7749,-122.4194" --zoom 16

# Or use traditional commands for more control
$ ams search --near "37.7749,-122.4194" "restaurants"
$ ams directions "San Francisco" "Palo Alto" --mode car
$ ams geocode "1 Infinite Loop, Cupertino, CA"
```

## Install

```bash
go install github.com/dl-alexandre/Apple-Map-Server-CLI/cmd/ams@latest
```

## Install (Homebrew)

```bash
brew tap dl-alexandre/tap
brew install ams
```

## Usage

```bash
ams help
ams version
ams ping
ams ping --request-id
ams auth token
ams geocode "1 Infinite Loop, Cupertino, CA"
amr geocode --json "1 Infinite Loop, Cupertino, CA"
ams geocode --limit 3 "1 Infinite Loop, Cupertino, CA"
ams geocode --file queries.txt --limit 3
ams geocode --file queries.txt --json
ams geocode --file queries.txt --concurrency 8
ams reverse 37.3317,-122.0301
ams reverse 37.3317,-122.0301 --limit 3
ams reverse 37.3317,-122.0301 --json
ams directions "37.7857,-122.4011" "San Francisco City Hall, CA"
ams directions "SF" "LA" --mode car --eta
ams directions "1 Infinite Loop, Cupertino" "Palo Alto, CA" --mode walk
ams search --near "37.7749,-122.4194" "coffee shops"
ams search --near-address "San Francisco, CA" --limit 20 restaurants
ams search --region "37.8,-122.4,37.7,-122.5" --category fuel "gas stations"
```

## Commands

- `help [command]` Show help for a command
- `auth token [--raw|--json]` Exchange JWT for an access token
- `geocode [--json] [--limit N] [--file <path>] [--concurrency N] <address>` Geocode an address
- `reverse <lat>,<lon> [--limit N] [--json]` Reverse geocode coordinates
- `directions <origin> <destination> [--mode <transport>] [--eta] [--json]` Get directions between locations
- `search [--near "lat,lng"] [--region "n,e,s,w"] [--near-address <addr>] [--no-cache] [--limit N] [--category <cat>] [--json] <query>` Search for places and POIs
- `search autocomplete [--near "lat,lng"] [--limit N] [--json] <query>` Get autocomplete suggestions
- `cache <stats|clear>` Manage geocode cache
- `snapshot <center> [--zoom N] [--size WxH] [--format png|jpg] [--output <path>]` Generate static map image
- `unified <query> [--near "lat,lng"] [--zoom N] [--output <path>]` Search and snapshot in one command
- `version` Show version info
- `ping [--request-id]` Ping the Apple Map Server

### Directions

Transport modes: `car` (default), `walk`, `transit`, `bike`

Get turn-by-turn directions:
```bash
ams directions "37.7857,-122.4011" "San Francisco City Hall, CA"
```

Get only ETA and distance:
```bash
ams directions "SF" "LA" --eta
```

Use different transport mode:
```bash
ams directions "1 Infinite Loop, Cupertino" "Palo Alto, CA" --mode bike
```

### Search

Search for places and points of interest. Geographic bounds are mutually exclusive—use only one of `--near`, `--region`, or `--near-address`.

**Search near coordinates:**
```bash
ams search --near "37.7749,-122.4194" "coffee shops"
```

**Search near an address (geocoded automatically):**
```bash
ams search --near-address "San Francisco, CA" "restaurants"
```

**Search within a bounding box:**
```bash
ams search --region "37.8,-122.4,37.7,-122.5" "gas stations"
```

**Filter by category and limit results:**
```bash
ams search --near "37.7749,-122.4194" --category cafe --limit 20 "coffee"
```

**Get JSON output:**
```bash
ams search --near "37.7749,-122.4194" --json "pizza"
```

**Note:** Flags must come before the query (positional arguments).

### Caching

When using `--near-address`, geocoded coordinates are automatically cached to reduce API calls and improve performance. The cache is stored in your system's cache directory (e.g., `~/.cache/ams/geocode_cache.json` on Linux).

**Cached address search:**
```bash
# First call geocodes and caches the result
ams search --near-address "San Francisco, CA" "restaurants"

# Subsequent calls use cached coordinates (instant!)
ams search --near-address "San Francisco, CA" "coffee"
```

**Bypass cache:**
```bash
ams search --near-address "123 Main St" --no-cache "pizza"
```

**Cache details:**
- TTL: 30 days (addresses rarely change coordinates)
- Location: OS-specific cache directory
- Format: JSON with timestamps

**Cache management:**
```bash
ams cache stats   # Show cache statistics
ams cache clear   # Clear all cached entries
```

### Autocomplete

Get predictive search suggestions before completing a full search. Great for building search-as-you-type interfaces.

**Basic autocomplete:**
```bash
ams search autocomplete "starbu"
```

**Autocomplete with location bias:**
```bash
ams search autocomplete --near "37.7749,-122.4194" "pizza"
```

**Get more suggestions:**
```bash
ams search autocomplete --limit 20 "coffee"
```

**JSON output:**
```bash
ams search autocomplete --json "taco"
```

The autocomplete response includes:
- `displayLines`: The suggestion text to show to users (usually 1-2 lines)
- `completionUrl`: A URL path to fetch full POI details if the user selects this suggestion

### Snapshot (Static Maps)

Generate static map images using the Apple Maps Web Snapshot API. This requires URL signing with your private key.

**Generate a map image:**
```bash
ams snapshot "37.7749,-122.4194"
ams snapshot "San Francisco, CA" --zoom 14 --size 600x400
ams snapshot "1 Infinite Loop, Cupertino" --zoom 16 --output map.png
```

**Customize the map:**
```bash
# Change zoom level (1-20)
ams snapshot "London, UK" --zoom 10 --size 800x600

# Use JPEG format
ams snapshot "New York, NY" --format jpg --output nyc.jpg

# Specify output file
ams snapshot "Tokyo, Japan" --zoom 12 --output tokyo.png
```

**Snapshot Environment Variables:**
The snapshot API requires additional credentials for URL signing:
- `AMS_TEAM_ID` - Your Apple Developer Team ID (10 characters)
- `AMS_KEY_ID` - Your Maps Key ID (10 characters)  
- `AMS_PRIVATE_KEY` - Your private key content (or use `AMS_PRIVATE_KEY_PATH`)

These are separate from `AMS_MAPS_TOKEN` and are used to cryptographically sign snapshot URLs.

### Unified (Search + Snapshot)

Combine search and snapshot in one powerful command. Search for a place and automatically generate a map image of the first result.

**Quick map generation:**
```bash
ams unified "Golden Gate Bridge"
ams unified "coffee shops" --near "37.7749,-122.4194"
ams unified "restaurants" --near-address "San Francisco" --zoom 14 --output sf.png
```

**How it works:**
1. Searches for the query using the Search API
2. Takes the first result's coordinates
3. Generates a snapshot centered on that location
4. Saves to a file named after the place

**Customize the output:**
```bash
# Change zoom level
ams unified "airports" --near "London, UK" --zoom 12

# Specify output file
ams unified "Statue of Liberty" --output liberty.png

# Search near coordinates
ams unified "pizza" --near "40.7128,-74.0060" --zoom 16
```

The unified command gracefully handles missing snapshot credentials - if you only have `AMS_MAPS_TOKEN`, it will still perform the search and show you the results (just without generating the image).

## Environment Variables

- `AMS_MAPS_TOKEN` (**required**) - Maps Token from Apple Developer portal
  - **⚠️ EXPIRES EVERY 7 DAYS** - must be manually regenerated
- `AMS_TEAM_ID` (required for `snapshot`) - Apple Developer Team ID
- `AMS_KEY_ID` (required for `snapshot`) - Maps Key ID
- `AMS_PRIVATE_KEY` or `AMS_PRIVATE_KEY_PATH` (required for `snapshot`) - Private key for URL signing
- `AMS_BASE_URL` (optional, override API base URL)
- `AMS_DEBUG=1` (optional, emit token exchange debug logs to stderr)

## Token Authentication

Apple Maps Server API requires a Maps Token from the Apple Developer portal. This token **expires every 7 days** and there is no auto-renewal API.

When your token expires:
1. Go to https://developer.apple.com/maps/server-api/
2. Generate a new Maps Token
3. Update your environment: `export AMS_MAPS_TOKEN=<your-new-token>`

The CLI will print a warning before each API call reminding you of this limitation.

## Shell Completion

Tab completion for commands and flags. Scripts are in the `scripts/` directory.

**Bash:**
```bash
# Add to your ~/.bashrc or ~/.bash_profile
source /path/to/Apple-Map-Server-CLI/scripts/completion.bash
```

**Zsh:**
```bash
# Copy to your fpath
mkdir -p ~/.zsh/completions
cp /path/to/Apple-Map-Server-CLI/scripts/completion.zsh ~/.zsh/completions/_ams

# Add to your ~/.zshrc
fpath=(~/.zsh/completions $fpath)
autoload -U compinit && compinit
```

**Now you can use tab completion:**
```bash
ams <TAB>          # Show all commands
ams sea<TAB>       # Auto-complete to "search"
ams search <TAB>   # Show search subcommands and flags
```

## Batch Geocode

When `--file` is used with `--json`, output is JSONL with one object per input.

```bash
ams geocode --file queries.txt --json
```

## Exit Codes

- `0` success
- `1` runtime/API error
- `2` usage error

## Build Metadata

Default values:

- `Version`: `dev`
- `Commit`: `none`
- `Date`: `unknown`

Build example:

```bash
go build -ldflags="-X github.com/dl-alexandre/Apple-Map-Server-CLI/internal/version.Version=v0.1.0 \
-X github.com/dl-alexandre/Apple-Map-Server-CLI/internal/version.Commit=$(git rev-parse --short HEAD) \
-X github.com/dl-alexandre/Apple-Map-Server-CLI/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
./cmd/ams
```
