package models

// BuildProgressType represents the type of build progress event
type BuildProgressType string

const (
	// Build events
	BuildProgressStart    BuildProgressType = "build_start"
	BuildProgressPlatform BuildProgressType = "build_platform"
	BuildProgressComplete BuildProgressType = "build_complete"

	// Upload events
	UploadProgressStart    BuildProgressType = "upload_start"
	UploadProgressAsset    BuildProgressType = "upload_asset"
	UploadProgressComplete BuildProgressType = "upload_complete"

	// Error event
	BuildProgressError BuildProgressType = "error"
)

// BuildProgress represents a progress update during binary building and uploading
type BuildProgress struct {
	Type     BuildProgressType
	Platform string // e.g., "linux/amd64", "windows/arm64"
	Asset    string // e.g., "matecommit_version_linux_x86_64.tar.gz"
	Current  int    // Current item being processed
	Total    int    // Total items to process
	Error    error  // Error if Type is BuildProgressError
}
