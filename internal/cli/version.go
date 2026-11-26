package cli

import cliver "github.com/dl-alexandre/cli-tools/version"

// Build-time variables re-export cli-tools/version values for package users.
var (
	Version    = cliver.Version
	BinaryName = "ams"
	GitHubRepo = "dl-alexandre/Apple-Map-Server-CLI"
	GitCommit  = cliver.GitCommit
	BuildTime  = cliver.BuildTime
)

// SetBuildInfo synchronizes the shared version package with this binary's
// existing linker-injected metadata.
func SetBuildInfo(version, commit, date string) {
	Version, GitCommit, BuildTime = version, commit, date
	cliver.Version = version
	cliver.BinaryName = BinaryName
	cliver.GitCommit = commit
	cliver.BuildTime = date
}
