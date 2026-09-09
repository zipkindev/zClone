// Package logflags implements command line flags to set up the log
package logflags

import (
	"github.com/spf13/pflag"
	"zclone/fs/config/flags"
	"zclone/fs/log"
)

// AddFlags adds the log flags to the flagSet
func AddFlags(flagSet *pflag.FlagSet) {
	flags.AddFlagsFromOptions(flagSet, "", log.OptionsInfo)
}
