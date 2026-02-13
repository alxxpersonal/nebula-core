package cmd

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginCmdRejectsEmptyUsername(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Provide an empty username line.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	_, _ = io.WriteString(w, "\n")
	_ = w.Close()
	os.Stdin = r

	cmd := LoginCmd()
	cmd.SetArgs([]string{})
	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestAgentCmdUnknownSubcommandDeterministicError(t *testing.T) {
	cmd := AgentCmd()
	cmd.SetArgs([]string{"nope"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestAgentCmdHelpWorks(t *testing.T) {
	cmd := AgentCmd()
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestKeysCmdNotLoggedInErrors(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	cmd := KeysCmd()
	cmd.SetArgs([]string{"list"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}
