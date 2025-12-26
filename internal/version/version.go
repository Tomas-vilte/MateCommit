package version

// Version is the current version of MateCommit
// This version should be updated in each release
const Version = "1.5.0"

// FullVersion returns the version with the v prefix
func FullVersion() string {
	return "v" + Version
}
