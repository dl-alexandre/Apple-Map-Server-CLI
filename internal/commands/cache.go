package commands

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/cache"
)

const cacheUsage = `Usage:
  ams cache stats    Show cache statistics
  ams cache clear    Clear all cached entries

Manage the geocode cache for --near-address searches.

The cache is stored in your system's cache directory:
  Linux: ~/.cache/ams/geocode_cache.json
  macOS: ~/Library/Caches/ams/geocode_cache.json
  Windows: %LOCALAPPDATA%\ams\geocode_cache.json

Examples:
  ams cache stats
  ams cache clear
`

func NewCacheCommand() Command {
	return Command{
		Name:      "cache",
		UsageLine: "cache <stats|clear>",
		Summary:   "Manage geocode cache",
		Usage:     cacheUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			// Check for subcommands
			if len(args) == 0 {
				fmt.Fprintln(stderr, "cache requires a subcommand: stats or clear")
				fmt.Fprint(stderr, cacheUsage)
				return ExitUsageError
			}

			subcommand := args[0]
			switch subcommand {
			case "stats":
				return runCacheStats(stdout, stderr)
			case "clear":
				return runCacheClear(stdout, stderr)
			default:
				fmt.Fprintf(stderr, "unknown cache subcommand: %s\n", subcommand)
				fmt.Fprint(stderr, cacheUsage)
				return ExitUsageError
			}
		},
	}
}

func runCacheStats(stdout, stderr io.Writer) int {
	c, err := cache.New()
	if err != nil {
		fmt.Fprintf(stderr, "error initializing cache: %v\n", err)
		return ExitRuntimeError
	}

	// Get cache file info
	path := c.Path()
	info, err := os.Stat(path)
	var fileSize int64 = 0
	if err == nil {
		fileSize = info.Size()
	}

	// Get cache statistics
	total, expired := c.Stats()
	active := total - expired

	// Output
	fmt.Fprintf(stdout, "Cache Location: %s\n", path)
	fmt.Fprintf(stdout, "Cache File Size: %s\n", formatBytes(fileSize))
	fmt.Fprintf(stdout, "Total Entries: %d\n", total)
	fmt.Fprintf(stdout, "Active Entries: %d\n", active)
	fmt.Fprintf(stdout, "Expired Entries: %d\n", expired)
	fmt.Fprintf(stdout, "TTL: %s\n", cache.DefaultTTL)

	if info != nil {
		fmt.Fprintf(stdout, "Last Modified: %s\n", info.ModTime().Format(time.RFC3339))
	}

	return ExitSuccess
}

func runCacheClear(stdout, stderr io.Writer) int {
	c, err := cache.New()
	if err != nil {
		// If we can't initialize cache, it might not exist yet
		fmt.Fprintln(stdout, "Cache is empty (no cache file exists)")
		return ExitSuccess
	}

	// Get stats before clearing
	path := c.Path()
	total, _ := c.Stats()

	if total == 0 {
		fmt.Fprintln(stdout, "Cache is empty (no entries)")
		return ExitSuccess
	}

	// Clear the cache
	c.Clear()

	// Save the empty cache (deletes the file)
	if err := c.Save(); err != nil {
		// If save fails, try to delete the file directly
		if os.Remove(path) == nil {
			fmt.Fprintf(stdout, "Cleared %d cached entries\n", total)
			return ExitSuccess
		}
		fmt.Fprintf(stderr, "error clearing cache: %v\n", err)
		return ExitRuntimeError
	}

	// Verify file was deleted
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(stdout, "Cleared %d cached entries\n", total)
	} else {
		fmt.Fprintf(stdout, "Cleared %d cached entries (cache file truncated)\n", total)
	}

	return ExitSuccess
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)

	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
