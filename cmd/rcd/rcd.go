// Package rcd provides the rcd command.
package rcd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"zclone/cmd"
	"zclone/fs"
	"zclone/fs/rc"
	"zclone/fs/rc/rcflags"
	"zclone/fs/rc/rcserver"
	libhttp "zclone/lib/http"
	"zclone/lib/systemd"
)

func init() {
	cmd.Root.AddCommand(commandDefinition)
}

var commandDefinition = &cobra.Command{
	Use:   "rcd <path to files to serve>*",
	Short: `Run zclone listening to remote control commands only.`,
	Long: `This runs zclone so that it only listens to remote control commands.

This is useful if you are controlling zclone via the rc API.

If you pass in a path to a directory, zclone will serve that directory
for GET requests on the URL passed in.  It will also open the URL in
the browser when zclone is run. If authentication is configured, the URL
does not contain credentials.

See the [rc documentation](/rc/) for more info on the rc flags.

` + strings.TrimSpace(libhttp.Help(rcflags.FlagPrefix)+libhttp.TemplateHelp(rcflags.FlagPrefix)+libhttp.AuthHelp(rcflags.FlagPrefix)),
	Annotations: map[string]string{
		"versionIntroduced": "v1.45",
		"groups":            "RC",
	},
	Run: func(command *cobra.Command, args []string) {
		cmd.CheckArgs(0, 1, command, args)
		if rc.Opt.Enabled {
			fs.Fatalf(nil, "Don't supply --rc flag when using rcd")
		}

		// Start the rc
		rc.Opt.Enabled = true
		if len(args) > 0 {
			rc.Opt.Files = args[0]
		}

		s, err := rcserver.Start(context.Background(), &rc.Opt)
		if err != nil {
			fs.Fatalf(nil, "Failed to start remote control: %v", err)
		}
		if s == nil {
			fs.Fatal(nil, "rc server not configured")
		}

		// Notify stopping on exit
		defer systemd.Notify()()

		s.Wait()
	},
}
