package version

import (
	"fmt"
	"runtime"
)

var (
	version   = "0.0.0-dev"            // version number of the software
	buildDate = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

type Info struct {
	BuildDate string `json:"buildDate"`
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// String returns info as a human-friendly version string.
func (info Info) String() string {
	return info.Version
}

func Get() Info {
	return Info{
		BuildDate: buildDate,
		Version:   version,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
