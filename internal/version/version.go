// Package version provides version information for the binary.
package version

// These variables are set via ldflags during build
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// Info returns formatted version information
func Info() string {
	return "myrient-dl " + Version + " (commit: " + GitCommit + ", built: " + BuildTime + ")"
}
