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

- **Coordinate Parsing Helpers**: Robust validation for geographic inputs
  - `parseCoordinate("lat,lng")` with latitude/longitude bounds checking
  - `parseBoundingBox("n,e,s,w")` with geometry validation
  - 33 comprehensive test cases covering edge cases

### Engineering

- Added 10 dedicated test cases for the search command
- Haversine distance calculation for proximity display in table output
- Token expiry warning integration consistent with other commands
- Updated golden test files to reflect new command in help output
- Total test coverage: 40+ test cases across coordinate parsing and search functionality

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

