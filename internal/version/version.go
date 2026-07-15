package version

// Set by goreleaser via ldflags; "dev" for local builds.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
