package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/dl-alexandre/cli-tools/update"
)

// UpdateInfo preserves the package's public result type while using the shared
// update checker.
type UpdateInfo = update.Info

func newUpdateChecker() *update.Checker {
	return update.New(update.Config{
		CurrentVersion: Version,
		BinaryName:     BinaryName,
		GitHubRepo:     GitHubRepo,
		InstallCommand: "brew upgrade ams",
	})
}

// CheckForUpdates performs a manual update check and displays its result.
func CheckForUpdates() (bool, error) {
	info, err := newUpdateChecker().Check(false)
	if err != nil {
		return false, err
	}
	if err := update.DisplayUpdate(info, BinaryName, "table"); err != nil {
		return false, err
	}
	return info.UpdateAvailable, nil
}

// AutoUpdateCheck performs a non-blocking background update check.
func AutoUpdateCheck() { newUpdateChecker().AutoCheck() }

// ClearUpdateCache removes the cache file used by cli-tools/update.
func ClearUpdateCache() error {
	root := os.Getenv("CACHE_DIR")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		root = filepath.Join(home, ".cache")
	}
	path := filepath.Join(root, "ams", "update_check.json")
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
