package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunTUIMissingConfigReturnsError(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	err := runTUI()
	assert.Error(t, err)
}

func TestMainHelpFlagDoesNotExit(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"nebula", "--help"}
	defer func() { os.Args = oldArgs }()

	// main() should return normally for help (no os.Exit).
	main()
}
