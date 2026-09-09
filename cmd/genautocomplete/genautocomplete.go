// Package genautocomplete provides the completion command.
package genautocomplete

import (
	"github.com/spf13/cobra"
	"zclone/cmd"
)

func init() {
	cmd.Root.AddCommand(completionDefinition)
}

var completionDefinition = &cobra.Command{
	Use:   "completion [shell]",
	Short: `Output completion script for a given shell.`,
	Long: `Generates a shell completion script for zclone.
Run with ` + "`--help`" + ` to list the supported shells.`,
	Annotations: map[string]string{
		"versionIntroduced": "v1.33",
	},
	Aliases: []string{"genautocomplete"},
}
