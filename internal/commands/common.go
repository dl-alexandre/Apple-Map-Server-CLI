package commands

import "github.com/dl-alexandre/Apple-Map-Server-CLI/internal/version"

func userAgent() string {
	v := version.Version
	if v == "" {
		v = "dev"
	}
	return "ams/" + v
}
