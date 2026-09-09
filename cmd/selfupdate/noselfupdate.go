//go:build noselfupdate

package selfupdate

import (
	"zclone/lib/buildinfo"
)

func init() {
	buildinfo.Tags = append(buildinfo.Tags, "noselfupdate")
}
