//go:build selfupdate

// Package selfupdate provides a disabled compatibility command.
package selfupdate

import (
	"errors"

	"github.com/spf13/cobra"
	"zclone/cmd"
)

var errReleaseServiceDisabled = errors.New("self-update is disabled because this zClone build has no configured release service")

func init() {
	cmd.Root.AddCommand(commandDefinition)
	commandDefinition.Flags().Bool("check", false, "Report that self-update is unavailable")
}

var commandDefinition = &cobra.Command{
	Use:     "selfupdate",
	Aliases: []string{"self-update"},
	Short:   "Report that self-update is unavailable.",
	Long: `This local zClone distribution does not configure a release service.

Build a replacement locally with "make zclone" and install it using your normal
local deployment process.`,
	RunE: func(command *cobra.Command, args []string) error {
		cmd.CheckArgs(0, 0, command, args)
		return errReleaseServiceDisabled
	},
}
