//go:build ignore

package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/stretchr/testify/assert/yaml"
	"zclone/fs"
	"zclone/fstest/runs"
)

var path = flag.String("path", "./docs/content/", "root path")

const (
	configFile        = "fstest/test_all/config.yaml"
	startListIgnores  = "<!--- start list_ignores - DO NOT EDIT THIS SECTION - use make commanddocs --->"
	endListIgnores    = "<!--- end list_ignores - DO NOT EDIT THIS SECTION - use make commanddocs --->"
	startListFailures = "<!--- start list_failures - DO NOT EDIT THIS SECTION - use make commanddocs --->"
	endListFailures   = "<!--- end list_failures - DO NOT EDIT THIS SECTION - use make commanddocs --->"
)

func main() {
	err := replaceBetween(*path, startListIgnores, endListIgnores, getIgnores)
	if err != nil {
		fs.Errorf(*path, "error replacing ignores: %v", err)
	}
	err = replaceBetween(*path, startListFailures, endListFailures, getFailures)
	if err != nil {
		fs.Errorf(*path, "error replacing failures: %v", err)
	}
}

// replaceBetween replaces the text between startSep and endSep with fn()
func replaceBetween(path, startSep, endSep string, fn func() (string, error)) error {
	b, err := os.ReadFile(filepath.Join(path, "bisync.md"))
	if err != nil {
		return err
	}
	doc := string(b)

	before, after, found := strings.Cut(doc, startSep)
	if !found {
		return fmt.Errorf("could not find: %v", startSep)
	}
	_, after, found = strings.Cut(after, endSep)
	if !found {
		return fmt.Errorf("could not find: %v", endSep)
	}

	replaceSection, err := fn()
	if err != nil {
		return err
	}

	newDoc := before + startSep + "\n" + strings.TrimSpace(replaceSection) + "\n" + endSep + after

	err = os.WriteFile(filepath.Join(path, "bisync.md"), []byte(newDoc), 0777)
	if err != nil {
		return err
	}
	return nil
}

// getIgnores updates the list of ignores from config.yaml
func getIgnores() (string, error) {
	config, err := parseConfig()
	if err != nil {
		return "", fmt.Errorf("failed to parse config: %v", err)
	}
	s := ""
	slices.SortFunc(config.Backends, func(a, b runs.Backend) int {
		return cmp.Compare(a.Remote, b.Remote)
	})
	for _, backend := range config.Backends {
		include := false

		if slices.Contains(backend.IgnoreTests, "cmd/bisync") {
			include = true
			s += fmt.Sprintf("- `%s` (`%s`)\n", strings.TrimSuffix(backend.Remote, ":"), backend.Backend)
		}

		for _, ignore := range backend.Ignore {
			if strings.Contains(strings.ToLower(ignore), "bisync") {
				if !include { // don't have header row yet
					s += fmt.Sprintf("- `%s` (`%s`)\n", strings.TrimSuffix(backend.Remote, ":"), backend.Backend)
				}
				include = true
				s += fmt.Sprintf("  - `%s`\n", ignore)
				// TODO: might be neat to add a "reason" param displaying the reason the test is ignored
			}
		}
	}
	return s, nil
}

// getFailures reports that the local distribution does not publish integration test results.
func getFailures() (string, error) {
	return "Integration test reports are not published by this local distribution.", nil
}

// parseConfig reads and parses the config.yaml file
func parseConfig() (*runs.Config, error) {
	d, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	config := &runs.Config{}
	err = yaml.Unmarshal(d, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return config, nil
}
