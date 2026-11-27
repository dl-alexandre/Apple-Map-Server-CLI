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
	cacheDir := filepath.Join(filepath.Clean(root), "ams")
	dir, err := os.OpenRoot(cacheDir) // #nosec G703 -- user-configured cache directory for fixed update_check.json
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer func() { _ = dir.Close() }()
	if err := dir.Remove("update_check.json"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
