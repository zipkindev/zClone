// cmdtest_test creates a testable interface to zclone main
//
// The interface is used to perform end-to-end test of
// commands, flags, environment variables etc.

package cmdtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"zclone/fs"
)

// TestMain is initially called by go test to initiate the testing.
// TestMain is also called during the tests to start zclone main in a fresh context (using exec.Command).
// The context is determined by setting/finding the environment variable ZCLONE_TEST_MAIN
func TestMain(m *testing.M) {
	_, found := os.LookupEnv(zcloneTestMain)
	if !found {
		// started by Go test => execute tests
		err := os.Setenv(zcloneTestMain, "true")
		if err != nil {
			fs.Fatalf(nil, "Unable to set %s: %s", zcloneTestMain, err.Error())
		}
		os.Exit(m.Run())
	} else {
		// started by func zcloneExecMain => call zclone main in cmdtest.go
		err := os.Unsetenv(zcloneTestMain)
		if err != nil {
			fs.Fatalf(nil, "Unable to unset %s: %s", zcloneTestMain, err.Error())
		}
		main()
	}
}

const zcloneTestMain = "ZCLONE_TEST_MAIN"

// zcloneExecMain calls zclone with the given environment and arguments.
// The environment variables are in a single string separated by ;
// The terminal output is returned as a string.
func zcloneExecMain(env string, args ...string) (string, error) {
	_, found := os.LookupEnv(zcloneTestMain)
	if !found {
		fs.Fatalf(nil, "Unexpected execution path: %s is missing.", zcloneTestMain)
	}
	// make a call to self to execute zclone main in a predefined environment (enters TestMain above)
	command := exec.Command(os.Args[0], args...)
	command.Env = getEnvInitial()
	if env != "" {
		command.Env = append(command.Env, strings.Split(env, ";")...)
	}
	out, err := command.CombinedOutput()
	return string(out), err
}

// zcloneEnv calls zclone with the given environment and arguments.
// The environment variables are in a single string separated by ;
// The test config file is automatically configured in ZCLONE_CONFIG.
// The terminal output is returned as a string.
func zcloneEnv(env string, args ...string) (string, error) {
	envConfig := env
	if testConfig != "" {
		if envConfig != "" {
			envConfig += ";"
		}
		envConfig += "ZCLONE_CONFIG=" + testConfig
	}
	return zcloneExecMain(envConfig, args...)
}

// zclone calls zclone with the given arguments, E.g. "version","--help".
// The test config file is automatically configured in ZCLONE_CONFIG.
// The terminal output is returned as a string.
func zclone(args ...string) (string, error) {
	return zcloneEnv("", args...)
}

// getEnvInitial returns the os environment variables cleaned for ZCLONE_ vars (except ZCLONE_TEST_MAIN).
func getEnvInitial() []string {
	if envInitial == nil {
		// Set initial environment variables
		osEnv := os.Environ()
		for i := range osEnv {
			if !strings.HasPrefix(osEnv[i], "ZCLONE_") || strings.HasPrefix(osEnv[i], zcloneTestMain) {
				envInitial = append(envInitial, osEnv[i])
			}
		}
	}
	return envInitial
}

var envInitial []string

// createTestEnvironment creates a temporary testFolder and
// sets testConfig to testFolder/zclone.config.
func createTestEnvironment(t *testing.T) {
	//Set temporary folder for config and test data
	tempFolder := t.TempDir()
	testFolder = filepath.ToSlash(tempFolder)

	// Set path to temporary config file
	testConfig = testFolder + "/zclone.config"
}

var testFolder string
var testConfig string

// createTestFile creates the file testFolder/name
func createTestFile(name string, t *testing.T) string {
	err := os.WriteFile(testFolder+"/"+name, []byte("content_of_"+name), 0666)
	require.NoError(t, err)
	return testFolder + "/" + name
}

// createTestFolder creates the folder testFolder/name
func createTestFolder(name string, t *testing.T) string {
	err := os.Mkdir(testFolder+"/"+name, 0777)
	require.NoError(t, err)
	return testFolder + "/" + name
}

// createSimpleTestData creates simple test data in testFolder/subFolder
func createSimpleTestData(t *testing.T) string {
	createTestFolder("testdata", t)
	createTestFile("testdata/file1.txt", t)
	createTestFile("testdata/file2.txt", t)
	createTestFolder("testdata/folderA", t)
	createTestFile("testdata/folderA/fileA1.txt", t)
	createTestFile("testdata/folderA/fileA2.txt", t)
	createTestFolder("testdata/folderA/folderAA", t)
	createTestFile("testdata/folderA/folderAA/fileAA1.txt", t)
	createTestFile("testdata/folderA/folderAA/fileAA2.txt", t)
	createTestFolder("testdata/folderB", t)
	createTestFile("testdata/folderB/fileB1.txt", t)
	createTestFile("testdata/folderB/fileB2.txt", t)

	t.Cleanup(func() {
		err := os.RemoveAll(testFolder + "/testdata")
		require.NoError(t, err)
	})

	return testFolder + "/testdata"
}

// TestCmdTest demonstrates and verifies the test functions for end-to-end testing of zclone
func TestCmdTest(t *testing.T) {
	createTestEnvironment(t)

	// Test simple call and output from zclone
	out, err := zclone("version")
	t.Log("zclone version\n" + out)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone v")
		assert.Contains(t, out, "version: ")
		assert.NotContains(t, out, "Error:")
		assert.NotContains(t, out, "--help")
		assert.NotContains(t, out, " DEBUG : ")
		assert.Regexp(t, "zclone\\s+v\\d+\\.\\d+", out) // zclone v_.__
	}

	// Test multiple arguments and DEBUG output
	out, err = zclone("version", "-vv")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone v")
		assert.Contains(t, out, " DEBUG : ")
	}

	// Test error and error output
	out, err = zclone("version", "--provoke-an-error")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "exit status 2")
		assert.Contains(t, out, "Error: unknown flag")
	}

	// Test effect of environment variable
	env := "ZCLONE_LOG_LEVEL=DEBUG"
	out, err = zcloneEnv(env, "version")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone v")
		assert.Contains(t, out, " DEBUG : ")
	}

	// Test effect of multiple environment variables, including one with ,
	env = "ZCLONE_LOG_LEVEL=DEBUG;ZCLONE_LOG_FORMAT=date,shortfile;ZCLONE_STATS=173ms"
	out, err = zcloneEnv(env, "version")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone v")
		assert.Contains(t, out, " DEBUG : ")
		assert.Regexp(t, "[^\\s]+\\.go:\\d+:", out) // ___.go:__:
		assert.Contains(t, out, "173ms")
	}

	// Test setup of config file
	out, err = zclone("config", "create", "myLocal", "local")
	if assert.NoError(t, err) {
		assert.Contains(t, out, "[myLocal]")
		assert.Contains(t, out, "type = local")
	}

	// Test creation of simple test data
	createSimpleTestData(t)

	// Test access to config file and simple test data
	out, err = zclone("lsl", "myLocal:"+testFolder)
	t.Log("zclone lsl myLocal:testFolder\n" + out)
	if assert.NoError(t, err) {
		assert.Contains(t, out, "zclone.config")
		assert.Contains(t, out, "testdata/folderA/fileA1.txt")
	}

}
