// Package rcflags implements command line flags to set up the remote control
package rcflags

import (
	"github.com/spf13/pflag"
	"zclone/fs/config/flags"
	"zclone/fs/rc"
)

// FlagPrefix is the prefix used to uniquely identify command line flags.
const FlagPrefix = "rc-"

// AddFlags adds the remote control flags to the flagSet
func AddFlags(flagSet *pflag.FlagSet) {
	flags.AddFlagsFromOptions(flagSet, "", rc.OptionsInfo)
}
