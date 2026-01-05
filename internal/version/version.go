package version

// Version is the current version of MateCommit
// This version should be updated in each release
const Version = "1.7.0"

// GitCommit is the git commit hash (injected at build time)
var GitCommit = "dev"

// BuildDate is the build date (injected at build time)
var BuildDate = "unknown"

// FullVersion returns the version with the v prefix
func FullVersion() string {
	return "v" + Version
}

// Info returns complete version information
func Info() string {
	return "v" + Version + " (commit: " + GitCommit + ", built: " + BuildDate + ")"
}
