// Package reveal provides the reveal command.
package reveal

import (
	"fmt"

	"github.com/spf13/cobra"
	"zclone/cmd"
	"zclone/fs/config/obscure"
)

func init() {
	cmd.Root.AddCommand(commandDefinition)
}

var commandDefinition = &cobra.Command{
	Use:   "reveal password",
	Short: `Reveal obscured password from zclone.conf`,
	Annotations: map[string]string{
		"versionIntroduced": "v1.43",
	},
	Run: func(command *cobra.Command, args []string) {
		cmd.CheckArgs(1, 1, command, args)
		cmd.Run(false, false, command, func() error {
			revealed, err := obscure.Reveal(args[0])
			if err != nil {
				return err
			}
			fmt.Println(revealed)
			return nil
		})
	},
	Hidden: true,
}
