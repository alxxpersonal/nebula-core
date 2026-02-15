package main

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunTUIMissingConfigReturnsError(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	_ = inW.Close()
	_ = outW.Close()
	os.Stdin = inR
	os.Stdout = outW
	defer func() {
		_, _ = io.Copy(io.Discard, outR)
		_ = outR.Close()
		_ = inR.Close()
	}()

	err = runTUI()
	assert.Error(t, err)
}

func TestMainHelpFlagDoesNotExit(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"nebula", "--help"}
	defer func() { os.Args = oldArgs }()

	// main() should return normally for help (no os.Exit).
	main()
}
