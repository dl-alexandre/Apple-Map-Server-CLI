package commands

import "github.com/dl-alexandre/Apple-Map-Server-CLI/internal/version"

func userAgent() string {
	v := version.Version
	if v == "" {
		v = "dev"
	}
	return "ams/" + v
}

// TokenExpiryWarning is shown when commands require Maps API authentication
const TokenExpiryWarning = `WARNING: Apple Maps Server API tokens expire every 7 days.
When your token expires, you must manually generate a new one at:
  https://developer.apple.com/maps/server-api/
Set the new token: export AMS_MAPS_TOKEN=<your-new-token>

`
