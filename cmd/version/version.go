// Package version provides the version command.
package version

import (
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/spf13/cobra"
	"zclone/cmd"
	"zclone/fs"
	"zclone/fs/config/flags"
	"zclone/fs/fshttp"
)

var (
	check = false
	deps  = false
)

func init() {
	cmd.Root.AddCommand(commandDefinition)
	cmdFlags := commandDefinition.Flags()
	flags.BoolVarP(cmdFlags, &check, "check", "", false, "Report whether version checking is configured", "")
	flags.BoolVarP(cmdFlags, &deps, "deps", "", false, "Show the Go dependencies", "")
}

var commandDefinition = &cobra.Command{
	Use:   "version",
	Short: `Show the version number.`,
	Long: `Show the zclone version number, the go version, the build target
OS and architecture, the runtime OS and kernel version and bitness,
build tags and the type of executable (static or dynamic).

For example:

` + "```console" + `
$ zclone version
zclone v1.55.0
- os/version: ubuntu 18.04 (64 bit)
- os/kernel: 4.15.0-136-generic (x86_64)
- os/type: linux
- os/arch: amd64
- go/version: go1.16
- go/linking: static
- go/tags: none
` + "```" + `

Note: before zclone version 1.55 the os/type and os/arch lines were merged,
      and the "go/version" line was tagged as "go version".

This local build does not have a configured release service, so --check
does not make a network request.

If you supply the --deps flag then zclone will print a list of all the
packages it depends on and their versions along with some other
information about the build.`,
	Annotations: map[string]string{
		"versionIntroduced": "v1.33",
	},
	RunE: func(command *cobra.Command, args []string) error {
		ctx := context.Background()
		cmd.CheckArgs(0, 0, command, args)
		if deps {
			return printDependencies()
		}
		if check {
			return CheckVersion(ctx)
		} else {
			cmd.ShowVersion()
		}
		return nil
	},
}

// strip a leading v off the string
func stripV(s string) string {
	if len(s) > 0 && s[0] == 'v' {
		return s[1:]
	}
	return s
}

// GetVersion gets the version available for download
func GetVersion(ctx context.Context, url string) (v *semver.Version, vs string, date time.Time, err error) {
	resp, err := fshttp.NewClient(ctx).Get(url)
	if err != nil {
		return v, vs, date, err
	}
	defer fs.CheckClose(resp.Body, &err)
	if resp.StatusCode != http.StatusOK {
		return v, vs, date, errors.New(resp.Status)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return v, vs, date, err
	}
	vs = strings.TrimSpace(string(bodyBytes))
	vs = strings.TrimPrefix(vs, "zclone ")
	vs = strings.TrimRight(vs, "β")
	date, err = http.ParseTime(resp.Header.Get("Last-Modified"))
	if err != nil {
		return v, vs, date, err
	}
	v, err = semver.NewVersion(stripV(vs))
	return v, vs, date, err
}

// CheckVersion reports that this local build has no configured release service.
func CheckVersion(context.Context) error {
	return errors.New("version checking is disabled because this Zclone build has no configured release service")
}

// Print info about a build module
func printModule(module *debug.Module) {
	if module.Replace != nil {
		fmt.Printf("- %s %s (replaced by %s %s)\n",
			module.Path, module.Version, module.Replace.Path, module.Replace.Version)
	} else {
		fmt.Printf("- %s %s\n", module.Path, module.Version)
	}
}

// printDependencies shows the packages we use in a format like go.mod
func printDependencies() error {
	info, err := buildinfo.ReadFile(os.Args[0])
	if err != nil {
		return fmt.Errorf("error reading build info: %w", err)
	}
	fmt.Println("Go Version:")
	fmt.Printf("- %s\n", info.GoVersion)
	fmt.Println("Main package:")
	printModule(&info.Main)
	fmt.Println("Binary path:")
	fmt.Printf("- %s\n", info.Path)
	fmt.Println("Settings:")
	for _, setting := range info.Settings {
		fmt.Printf("- %s: %s\n", setting.Key, setting.Value)
	}
	fmt.Println("Dependencies:")
	for _, dep := range info.Deps {
		printModule(dep)
	}
	return nil
}
