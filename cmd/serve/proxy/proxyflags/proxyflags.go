// Package proxyflags implements command line flags to set up a proxy
package proxyflags

import (
	"github.com/spf13/pflag"
	"zclone/cmd/serve/proxy"
	"zclone/fs/config/flags"
)

// AddFlags adds the non filing system specific flags to the command
func AddFlags(flagSet *pflag.FlagSet) {
	flags.AddFlagsFromOptions(flagSet, "", proxy.OptionsInfo)
}
