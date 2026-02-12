package auth

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Config struct {
	MapsToken string
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		MapsToken: os.Getenv("AMS_MAPS_TOKEN"),
	}

	var missing []string
	if strings.TrimSpace(cfg.MapsToken) == "" {
		missing = append(missing, "AMS_MAPS_TOKEN")
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return Config{}, MissingEnvError{Missing: missing}
	}

	return cfg, nil
}

type MissingEnvError struct {
	Missing []string
}

func (err MissingEnvError) Error() string {
	if len(err.Missing) == 0 {
		return "missing required env vars"
	}
	return fmt.Sprintf("missing required env vars: %s", strings.Join(err.Missing, ", "))
}

func IsMissingEnv(err error) bool {
	var missing MissingEnvError
	return errors.As(err, &missing)
}
