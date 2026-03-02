package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyTextToClipboardCommandFailureBranch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script clipboard shim is unix-only")
	}

	tmp := t.TempDir()
	bin := filepath.Join(tmp, "pbcopy")
	script := "#!/bin/sh\nexit 1\n"
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o755))
	t.Setenv("PATH", tmp)

	err := copyTextToClipboard("hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clipboard copy failed")
}

