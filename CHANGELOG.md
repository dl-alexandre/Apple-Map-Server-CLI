# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased] - Search API Release

### Added

- **Search Command (`ams search`)**: Full support for the Apple Maps Server `/v1/search` endpoint
  - Search for places and points of interest with queries
  - `--near "lat,lng"`: Search centered around specific coordinates
  - `--region "n,e,s,w"`: Search within a bounding box
  - `--near-address "addr"`: Geocode an address first, then search around it
  - `--limit N`: Limit results (default: 10)
  - `--category "cat"`: Filter by POI category (e.g., restaurant, cafe)
  - `--json`: Machine-readable JSON output
  - Table output with distance calculation when using `--near`
  - Mutual exclusivity enforced for geographic bounds

- **Autocomplete Command (`ams search autocomplete`)**: Full support for `/v1/searchAutocomplete`
  - Get predictive search suggestions for partial queries
  - `--near "lat,lng"`: Location bias for more relevant suggestions
  - `--limit N`: Limit suggestions (default: 10)
  - `--json`: Machine-readable JSON output with `displayLines` and `completionUrl`
  - Table output showing suggestion text and completion URLs
  - Routed as subcommand from `ams search` with clean flag separation

- **Cache Package**: Geocode result caching for `--near-address`
  - Automatic caching to `os.UserCacheDir()/ams/geocode_cache.json`
  - 30-day TTL for cached coordinates
  - `--no-cache` flag to bypass cache when needed
  - Atomic writes to prevent cache corruption
  - Stats and management methods for cache inspection
  - 15 comprehensive cache test cases

- **Cache Management Command**: Administrative interface for geocode cache
  - `ams cache stats`: Display cache location, size, entry count, and TTL
  - `ams cache clear`: Remove all cached entries with confirmation
  - Cross-platform cache directory detection
  - Human-readable file size formatting (B, KB, MB)
  - Cache statistics: total, active, and expired entries

- **Unified Command**: Search + Snapshot in one command
  - `ams unified <query>` searches and generates snapshot
  - `--near "lat,lng"`: Search near specific coordinates
  - `--zoom N`: Control snapshot zoom level
  - `--output <path>`: Custom output filename
  - Automatic filename from search result (sanitized)
  - Graceful degradation without snapshot credentials
  - 6 comprehensive unified command tests

- **Shell Completion**: Tab completion for bash and zsh
  - Auto-complete commands and subcommands
  - Flag completion with descriptions
  - Dynamic command suggestions
  - Scripts in `scripts/completion.bash` and `scripts/completion.zsh`
  - Easy installation instructions

- **Snapshot Command**: Static map image generation (Web Snapshots API)
  - `ams snapshot <center>` generates map images
  - `--zoom N`: Zoom level 1-20 (default: 12)
  - `--size WxH`: Image dimensions (default: 600x400)
  - `--format png|jpg`: Output format (default: png)
  - `--output <path>`: Save to specific file
  - ECDSA signature generation for URL signing
  - Support for coordinates or address as center point
  - 6 comprehensive snapshot command tests

- **Snapshot Authentication**: URL signing with private key
  - Requires `AMS_TEAM_ID`, `AMS_KEY_ID`, and `AMS_PRIVATE_KEY`
  - ECDSA signature using ES256 algorithm
  - SHA-256 hash of teamId + URL path
  - ASN.1 DER encoding with base64 URL-safe encoding

- **Coordinate Parsing Helpers**: Robust validation for geographic inputs
  - `parseCoordinate("lat,lng")` with latitude/longitude bounds checking
  - `parseBoundingBox("n,e,s,w")` with geometry validation
  - 33 comprehensive test cases covering edge cases

### Engineering

- Added 10 dedicated test cases for the autocomplete command
- Added 15 cache package test cases for geocode caching functionality
- Added cache management command tests
- Added 6 snapshot command tests
- Added 6 unified command tests
- Added ECDSA signature generation tests for URL signing
- Added shell completion scripts (bash and zsh)
- Subcommand routing from `ams search` to `ams search autocomplete` with clean separation
- Total test coverage: 80+ test cases across all features

### Usage Notes

Due to standard Go `flag` package parsing behavior, flags must come before positional arguments:
```bash
# Correct
ams search --near "37.7749,-122.4194" "coffee shops"

# Incorrect - flags after positional args will be treated as part of the query
ams search "coffee shops" --near "37.7749,-122.4194"
```

## Previous Releases

### Directions API & Token Lifecycle Improvements

**New Features**
- **Directions Command (`ams directions`)**: Full support for `/v1/directions` endpoint
- **Transport Modes**: `--mode` flag with support for `car`, `walk`, `transit`, `bike`
- **ETA Summary (`--eta`)**: Quick distance and time summary
- **JSON Output (`--json`)**: Machine-readable output for directions

**UX Improvements**
- **Proactive Token Expiry Warnings**: Automatic warning to stderr before authenticated API calls
- Help menus updated to reflect directions capabilities

**Engineering**
- 10 new dedicated test cases for directions command
- Updated golden test files for stderr warning outputs

