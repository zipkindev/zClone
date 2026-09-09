// Package filterflags implements command line flags to set up a filter
package filterflags

import (
	"github.com/spf13/pflag"
	"zclone/fs/config/flags"
	"zclone/fs/filter"
)

// AddFlags adds the non filing system specific flags to the command
func AddFlags(flagSet *pflag.FlagSet) {
	flags.AddFlagsFromOptions(flagSet, "", filter.OptionsInfo)
}
