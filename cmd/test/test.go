// Package test provides the test command.
package test

import (
	"github.com/spf13/cobra"
	"zclone/cmd"
)

func init() {
	cmd.Root.AddCommand(Command)
}

// Command definition for cobra
var Command = &cobra.Command{
	Use:   "test <subcommand>",
	Short: `Run a test command`,
	Long: `Zclone test is used to run test commands.

Select which test command you want with the subcommand, eg

` + "```console" + `
zclone test memory remote:
` + "```" + `

Each subcommand has its own options which you can see in their help.

**NB** Be careful running these commands, they may do strange things
so reading their documentation first is recommended.`,
	Annotations: map[string]string{
		"versionIntroduced": "v1.55",
	},
}
