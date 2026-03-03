# Apple Map Server CLI (ams)

`ams` is a CLI tool for interacting with Apple Maps Server APIs.

**⚠️ IMPORTANT: Apple Maps Server API tokens expire every 7 days.** When your token expires, you must manually generate a new one at https://developer.apple.com/maps/server-api/

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
- `search [--near "lat,lng"] [--region "n,e,s,w"] [--near-address <addr>] [--limit N] [--category <cat>] [--json] <query>` Search for places and POIs
- `search autocomplete [--near "lat,lng"] [--limit N] [--json] <query>` Get autocomplete suggestions
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

## Environment Variables

- `AMS_MAPS_TOKEN` (**required**) - Maps Token from Apple Developer portal
  - **⚠️ EXPIRES EVERY 7 DAYS** - must be manually regenerated
- `AMS_BASE_URL` (optional, override API base URL)
- `AMS_DEBUG=1` (optional, emit token exchange debug logs to stderr)

## Token Authentication

Apple Maps Server API requires a Maps Token from the Apple Developer portal. This token **expires every 7 days** and there is no auto-renewal API.

When your token expires:
1. Go to https://developer.apple.com/maps/server-api/
2. Generate a new Maps Token
3. Update your environment: `export AMS_MAPS_TOKEN=<your-new-token>`

The CLI will print a warning before each API call reminding you of this limitation.

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
