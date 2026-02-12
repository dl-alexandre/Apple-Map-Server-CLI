# Apple Map Server CLI (ams)

`ams` is a minimal CLI foundation for interacting with Apple Map Server APIs.

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
ams geocode --json "1 Infinite Loop, Cupertino, CA"
ams geocode --limit 3 "1 Infinite Loop, Cupertino, CA"
ams geocode --file queries.txt --limit 3
ams geocode --file queries.txt --json
ams geocode --file queries.txt --concurrency 8
ams reverse 37.3317,-122.0301
ams reverse 37.3317,-122.0301 --limit 3
ams reverse 37.3317,-122.0301 --json
```

## Commands

- `help [command]` Show help for a command
- `auth token [--raw|--json]` Exchange JWT for an access token
- `geocode [--json] [--limit N] [--file <path>] [--concurrency N] <address>` Geocode an address
- `reverse <lat>,<lon> [--limit N] [--json]` Reverse geocode coordinates
- `version` Show version info
- `ping [--request-id]` Ping the Apple Map Server

## Environment Variables

- `AMS_MAPS_TOKEN` (required, Maps Token from Apple Developer portal; expires after 7 days for Server API with Restriction=None)
- `AMS_BASE_URL` (optional, override API base URL)
- `AMS_DEBUG=1` (optional, emit token exchange debug logs to stderr)

## Auth Token

```bash
ams auth token
ams auth token --raw
ams auth token --json
```

## Token Rotation

When your `AMS_MAPS_TOKEN` expires (7 days for Server API with Restriction=None), generate a new Maps Token in the Apple Developer portal and replace the environment variable. The CLI will automatically refresh short-lived access tokens, but it cannot recover from an expired Maps Token and will return an auth error until you rotate it.

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
