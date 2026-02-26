package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWaitForAPIHealthReturnsTrueWhenHealthy handles waitForAPIHealth success on a healthy local endpoint.
func TestWaitForAPIHealthReturnsTrueWhenHealthy(t *testing.T) {
	previousProbe := waitForAPIHealthProbe
	t.Cleanup(func() {
		waitForAPIHealthProbe = previousProbe
	})
	attempts := 0
	waitForAPIHealthProbe = func() (string, error) {
		attempts++
		if attempts < 2 {
			return "", assert.AnError
		}
		return "ok", nil
	}

	assert.True(t, waitForAPIHealth(2*time.Second))
	assert.GreaterOrEqual(t, attempts, 2)
}

// TestWaitForAPIHealthReturnsFalseOnTimeout handles waitForAPIHealth timeout behavior when no API is reachable.
func TestWaitForAPIHealthReturnsFalseOnTimeout(t *testing.T) {
	previousProbe := waitForAPIHealthProbe
	t.Cleanup(func() {
		waitForAPIHealthProbe = previousProbe
	})
	attempts := 0
	waitForAPIHealthProbe = func() (string, error) {
		attempts++
		return "", assert.AnError
	}

	start := time.Now()
	assert.False(t, waitForAPIHealth(300*time.Millisecond))
	assert.GreaterOrEqual(t, time.Since(start), 250*time.Millisecond)
	assert.Greater(t, attempts, 0)
}

// TestResolveServerDirRejectsInvalidEnv handles invalid explicit server-dir overrides.
func TestResolveServerDirRejectsInvalidEnv(t *testing.T) {
	t.Setenv("NEBULA_SERVER_DIR", t.TempDir())

	_, err := resolveServerDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NEBULA_SERVER_DIR does not point")
}

// TestResolveServerDirFindsServerUnderWorkingDir handles cwd-based server discovery.
func TestResolveServerDirFindsServerUnderWorkingDir(t *testing.T) {
	root := t.TempDir()
	serverDir := filepath.Join(root, "server")
	require.NoError(t, os.MkdirAll(filepath.Join(serverDir, "src", "nebula_api"), 0o755))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(serverDir, "src", "nebula_api", "app.py"), []byte("app = None\n"), 0o644),
	)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(root))
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	t.Setenv("NEBULA_SERVER_DIR", "")
	got, err := resolveServerDir()
	require.NoError(t, err)
	expected, err := filepath.Abs(serverDir)
	require.NoError(t, err)
	assert.Equal(t, normalizePathPrefix(expected), normalizePathPrefix(got))
}

// TestRunStartCmdRejectsInvalidServerEnv handles invalid server path failures before process launch.
func TestRunStartCmdRejectsInvalidServerEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NEBULA_SERVER_DIR", t.TempDir())

	var out bytes.Buffer
	err := runStartCmd(&out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NEBULA_SERVER_DIR does not point")
}

// TestRunStartCmdReturnsHelpfulErrorWhenUvicornMissing handles missing uvicorn setup errors.
func TestRunStartCmdReturnsHelpfulErrorWhenUvicornMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	serverDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(serverDir, "src", "nebula_api"), 0o755))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(serverDir, "src", "nebula_api", "app.py"), []byte("app = None\n"), 0o644),
	)
	t.Setenv("NEBULA_SERVER_DIR", serverDir)

	var out bytes.Buffer
	err := runStartCmd(&out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uvicorn not found")
	_, statErr := os.Stat(apiLockPath())
	assert.True(t, os.IsNotExist(statErr))
}

// TestRunStopCmdReturnsErrorOnCorruptLock handles invalid lock-file parse failures.
func TestRunStopCmdReturnsErrorOnCorruptLock(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(runtimeDir(), 0o700))
	require.NoError(t, os.WriteFile(apiLockPath(), []byte("{broken"), 0o600))

	var out bytes.Buffer
	err := runStopCmd(&out)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "read api lock"))
}

// normalizePathPrefix handles macOS /private path aliases for robust path equality assertions.
func normalizePathPrefix(path string) string {
	path = filepath.Clean(path)
	return strings.TrimPrefix(path, "/private")
}
