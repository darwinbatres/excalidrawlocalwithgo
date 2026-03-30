package buildinfo

// Build-time variables set via ldflags.
var (
	Version   = "dev"
	CommitSHA = "unknown"
	BuildTime = "unknown"
)
