// environment_test tests the use and precedence of environment variables
//
// The tests rely on functions defined in cmdtest_test.go

package cmdtest

import (
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvironmentVariables demonstrates and verifies the test functions for end-to-end testing of zclone
func TestEnvironmentVariables(t *testing.T) {

	createTestEnvironment(t)

	testdataPath := createSimpleTestData(t)

	// Non backend flags
	// =================

	// First verify default behaviour of the implicit max_depth=-1
	env := ""
	out, err := zcloneEnv(env, "lsl", testFolder)
	//t.Logf("\n" + out)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone.config") // depth 1
		assert.Contains(t, out, "file1.txt")     // depth 2
		assert.Contains(t, out, "fileA1.txt")    // depth 3
		assert.Contains(t, out, "fileAA1.txt")   // depth 4
	}

	// Test of flag.Value
	env = "ZCLONE_MAX_DEPTH=2"
	out, err = zcloneEnv(env, "lsl", testFolder)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "file1.txt")     // depth 2
		assert.NotContains(t, out, "fileA1.txt") // depth 3
	}

	// Test of flag.Changed (tests #5341 Issue1)
	env = "ZCLONE_LOG_LEVEL=DEBUG"
	out, err = zcloneEnv(env, "version", "--quiet")
	if assert.Error(t, err) {
		assert.Contains(t, out, " DEBUG ")
		assert.Contains(t, out, "Can't set -q and --log-level")
		assert.Contains(t, "exit status 1", err.Error())
	}

	// Test of flag.DefValue
	env = "ZCLONE_STATS=173ms"
	out, err = zcloneEnv(env, "help", "flags")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "(default 173ms)")
	}

	// Test of command line flags overriding environment flags
	env = "ZCLONE_MAX_DEPTH=2"
	out, err = zcloneEnv(env, "lsl", testFolder, "--max-depth", "3")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "fileA1.txt")     // depth 3
		assert.NotContains(t, out, "fileAA1.txt") // depth 4
	}

	// Test of debug logging while initialising flags from environment (tests #5341 Enhance1)
	env = "ZCLONE_STATS=173ms"
	out, err = zcloneEnv(env, "version", "-vv")
	if assert.NoError(t, err) {
		assert.Contains(t, out, " DEBUG : ")
		assert.Contains(t, out, "--stats")
		assert.Contains(t, out, "173ms")
		assert.Contains(t, out, "ZCLONE_STATS=")
	}

	// Backend flags and remote name
	// - The listremotes command includes names from environment variables,
	//   the part between "ZCLONE_CONFIG_" and "_TYPE", converted to lowercase.
	// - When using a remote created from env, e.g. with lsd command,
	//   the name is case insensitive in contrast to remotes in config file
	//   (fs.ConfigToEnv converts to uppercase before checking environment).
	// - Previously using a remote created from env, e.g. with lsd command,
	//   would not be possible for remotes with '-' in names, and remote names
	//   with '_' could be referred to with both '-' and '_', because any '-'
	//   were replaced with '_' before lookup.
	// ===================================

	env = "ZCLONE_CONFIG_MY-LOCAL_TYPE=local"
	out, err = zcloneEnv(env, "listremotes")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "my-local:")
	}
	out, err = zcloneEnv(env, "lsl", "my-local:"+testFolder)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone.config")
		assert.Contains(t, out, "file1.txt")
		assert.Contains(t, out, "fileA1.txt")
		assert.Contains(t, out, "fileAA1.txt")
	}
	out, err = zcloneEnv(env, "lsl", "mY-LoCaL:"+testFolder)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone.config")
		assert.Contains(t, out, "file1.txt")
		assert.Contains(t, out, "fileA1.txt")
		assert.Contains(t, out, "fileAA1.txt")
	}
	out, err = zcloneEnv(env, "lsl", "my_local:"+testFolder)
	if assert.Error(t, err) {
		assert.Contains(t, out, "Failed to create file system")
	}

	env = "ZCLONE_CONFIG_MY_LOCAL_TYPE=local"
	out, err = zcloneEnv(env, "listremotes")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "my_local:")
	}
	out, err = zcloneEnv(env, "lsl", "my_local:"+testFolder)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone.config")
		assert.Contains(t, out, "file1.txt")
		assert.Contains(t, out, "fileA1.txt")
		assert.Contains(t, out, "fileAA1.txt")
	}
	out, err = zcloneEnv(env, "lsl", "my-local:"+testFolder)
	if assert.Error(t, err) {
		assert.Contains(t, out, "Failed to create file system")
	}

	// Backend flags and option precedence
	// ===================================

	// Test approach:
	// Verify no symlink warning when skip_links=true one the level with highest precedence
	// and skip_links=false on all levels with lower precedence
	//
	// Reference: https://zclone.org/docs/#precedence
	// Create a symlink in test data
	err = os.Symlink(testdataPath+"/folderA", testdataPath+"/symlinkA")
	if err != nil && runtime.GOOS == "windows" {
		// Windows grants the privilege only to an elevated process or one
		// running with Developer Mode enabled.
		t.Skipf("Skipping as symlinks are unavailable: %v", err)
	}
	require.NoError(t, err)

	// Create a local remote with explicit skip_links=false
	out, err = zclone("config", "create", "myLocal", "local", "skip_links", "false")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "[myLocal]")
		assert.Contains(t, out, "type = local")
		assert.Contains(t, out, "skip_links = false")
	}

	// Verify symlink warning when skip_links=false on all levels
	env = "ZCLONE_SKIP_LINKS=false;ZCLONE_LOCAL_SKIP_LINKS=false;ZCLONE_CONFIG_MYLOCAL_SKIP_LINKS=false"
	out, err = zcloneEnv(env, "lsd", "myLocal,skip_links=false:"+testdataPath, "--skip-links=false")
	//t.Logf("\n" + out)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "NOTICE: symlinkA:")
		assert.Contains(t, out, "folderA")
	}

	// Test precedence of connection strings
	env = "ZCLONE_SKIP_LINKS=false;ZCLONE_LOCAL_SKIP_LINKS=false;ZCLONE_CONFIG_MYLOCAL_SKIP_LINKS=false"
	out, err = zcloneEnv(env, "lsd", "myLocal,skip_links:"+testdataPath, "--skip-links=false")
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Test precedence of command line flags
	env = "ZCLONE_SKIP_LINKS=false;ZCLONE_LOCAL_SKIP_LINKS=false;ZCLONE_CONFIG_MYLOCAL_SKIP_LINKS=false"
	out, err = zcloneEnv(env, "lsd", "myLocal:"+testdataPath, "--skip-links")
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Test precedence of remote specific environment variables (tests #5341 Issue2)
	env = "ZCLONE_SKIP_LINKS=false;ZCLONE_LOCAL_SKIP_LINKS=false;ZCLONE_CONFIG_MYLOCAL_SKIP_LINKS=true"
	out, err = zcloneEnv(env, "lsd", "myLocal:"+testdataPath)
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Test precedence of backend specific environment variables (tests #5341 Issue3)
	env = "ZCLONE_SKIP_LINKS=false;ZCLONE_LOCAL_SKIP_LINKS=true"
	out, err = zcloneEnv(env, "lsd", "myLocal:"+testdataPath)
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Test precedence of backend generic environment variables
	env = "ZCLONE_SKIP_LINKS=true"
	out, err = zcloneEnv(env, "lsd", "myLocal:"+testdataPath)
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Recreate the test remote with explicit skip_links=true
	out, err = zclone("config", "create", "myLocal", "local", "skip_links", "true")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "[myLocal]")
		assert.Contains(t, out, "type = local")
		assert.Contains(t, out, "skip_links = true")
	}

	// Test precedence of config file options
	env = ""
	out, err = zcloneEnv(env, "lsd", "myLocal:"+testdataPath)
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Recreate the test remote with zclone defaults, that is implicit skip_links=false
	out, err = zclone("config", "create", "myLocal", "local")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "[myLocal]")
		assert.Contains(t, out, "type = local")
		assert.NotContains(t, out, "skip_links")
	}

	// Verify the zclone default value (implicit skip_links=false)
	env = ""
	out, err = zcloneEnv(env, "lsd", "myLocal:"+testdataPath)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "NOTICE: symlinkA:")
		assert.Contains(t, out, "folderA")
	}

	// Display of backend defaults (tests #4659)
	//------------------------------------------

	env = "ZCLONE_DRIVE_CHUNK_SIZE=111M"
	out, err = zcloneEnv(env, "help", "flags")
	if assert.NoError(t, err) {
		assert.Regexp(t, "--drive-chunk-size[^\\(]+\\(default 111M\\)", out)
	}

	// Options on referencing remotes (alias, crypt, etc.)
	//----------------------------------------------------

	// Create alias remote on myLocal having implicit skip_links=false
	out, err = zclone("config", "create", "myAlias", "alias", "remote", "myLocal:"+testdataPath)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "[myAlias]")
		assert.Contains(t, out, "type = alias")
		assert.Contains(t, out, "remote = myLocal:")
	}

	// Verify symlink warnings on the alias
	env = ""
	out, err = zcloneEnv(env, "lsd", "myAlias:")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "NOTICE: symlinkA:")
		assert.Contains(t, out, "folderA")
	}

	// Test backend generic flags
	// having effect on the underlying local remote
	env = "ZCLONE_SKIP_LINKS=true"
	out, err = zcloneEnv(env, "lsd", "myAlias:")
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Test backend specific flags
	// having effect on the underlying local remote
	env = "ZCLONE_LOCAL_SKIP_LINKS=true"
	out, err = zcloneEnv(env, "lsd", "myAlias:")
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Test remote specific flags
	// having no effect unless supported by the immediate remote (alias)
	env = "ZCLONE_CONFIG_MYALIAS_SKIP_LINKS=true"
	out, err = zcloneEnv(env, "lsd", "myAlias:")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "NOTICE: symlinkA:")
		assert.Contains(t, out, "folderA")
	}

	env = "ZCLONE_CONFIG_MYALIAS_REMOTE=" + "myLocal:" + testdataPath + "/folderA"
	out, err = zcloneEnv(env, "lsl", "myAlias:")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "fileA1.txt")
		assert.NotContains(t, out, "fileB1.txt")
	}

	// Test command line flags
	// having effect on the underlying local remote
	env = ""
	out, err = zcloneEnv(env, "lsd", "myAlias:", "--skip-links")
	if assert.NoError(t, err) {
		assert.NotContains(t, out, "symlinkA")
		assert.Contains(t, out, "folderA")
	}

	// Test connection specific flags
	// having no effect unless supported by the immediate remote (alias)
	env = ""
	out, err = zcloneEnv(env, "lsd", "myAlias,skip_links:")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "NOTICE: symlinkA:")
		assert.Contains(t, out, "folderA")
	}

	env = ""
	out, err = zcloneEnv(env, "lsl", "myAlias,remote='myLocal:"+testdataPath+"/folderA':", "-vv")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "fileA1.txt")
		assert.NotContains(t, out, "fileB1.txt")
	}

	// Test --use-json-log and -vv combinations
	jsonLogOK := func() {
		t.Helper()
		if assert.NoError(t, err) {
			assert.Contains(t, out, `"level":"debug"`)
			assert.Contains(t, out, `"msg":"Version `)
			assert.Contains(t, out, `"}`)
		}
	}
	env = "ZCLONE_USE_JSON_LOG=1;ZCLONE_LOG_LEVEL=DEBUG"
	out, err = zcloneEnv(env, "version")
	jsonLogOK()
	env = "ZCLONE_USE_JSON_LOG=1"
	out, err = zcloneEnv(env, "version", "-vv")
	jsonLogOK()
	env = "ZCLONE_LOG_LEVEL=DEBUG"
	out, err = zcloneEnv(env, "version", "--use-json-log")
	jsonLogOK()
	env = ""
	out, err = zcloneEnv(env, "version", "-vv", "--use-json-log")
	jsonLogOK()

	// Find all the File filter lines in out and return them
	parseFileFilters := func(out string) (extensions []string) {
		// Match: - (^|/)[^/]*\.jpg$
		find := regexp.MustCompile(`^- \(\^\|\/\)\[\^\/\]\*\\\.(.*?)\$$`)
		for line := range strings.SplitSeq(out, "\n") {
			if m := find.FindStringSubmatch(line); m != nil {
				extensions = append(extensions, m[1])
			}
		}
		return extensions
	}

	// Make sure that multiple valued (stringArray) environment variables are handled properly
	env = ``
	out, err = zcloneEnv(env, "version", "-vv", "--dump", "filters", "--exclude", "*.gif", "--exclude", "*.tif")
	require.NoError(t, err)
	assert.Equal(t, []string{"gif", "tif"}, parseFileFilters(out))

	env = `ZCLONE_EXCLUDE=*.jpg`
	out, err = zcloneEnv(env, "version", "-vv", "--dump", "filters", "--exclude", "*.gif")
	require.NoError(t, err)
	assert.Equal(t, []string{"jpg", "gif"}, parseFileFilters(out))

	env = `ZCLONE_EXCLUDE=*.jpg,*.png`
	out, err = zcloneEnv(env, "version", "-vv", "--dump", "filters", "--exclude", "*.gif", "--exclude", "*.tif")
	require.NoError(t, err)
	assert.Equal(t, []string{"jpg", "png", "gif", "tif"}, parseFileFilters(out))

	env = `ZCLONE_EXCLUDE="*.jpg","*.png"`
	out, err = zcloneEnv(env, "version", "-vv", "--dump", "filters")
	require.NoError(t, err)
	assert.Equal(t, []string{"jpg", "png"}, parseFileFilters(out))

	env = `ZCLONE_EXCLUDE="*.,,,","*.png"`
	out, err = zcloneEnv(env, "version", "-vv", "--dump", "filters")
	require.NoError(t, err)
	assert.Equal(t, []string{",,,", "png"}, parseFileFilters(out))
}
