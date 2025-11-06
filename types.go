package main

// Build info set via -ldflags at build time (optional).
var (
	buildVersion = "dev"
	buildCommit  = ""
	buildTime    = ""
)

// Info holds build metadata for the binary.
type Info struct {
	Version string
	Commit  string
	Built   string
}

// BuildInfo returns the build metadata.
func BuildInfo() Info {
	return Info{
		Version: buildVersion,
		Commit:  buildCommit,
		Built:   buildTime,
	}
}
