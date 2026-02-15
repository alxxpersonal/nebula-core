package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
)

// RunInteractiveLogin prompts for username, calls login API, and persists config.
func RunInteractiveLogin(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	fmt.Fprint(out, "username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	if username == "" {
		return fmt.Errorf("username is required")
	}

	client := api.NewDefaultClient("")
	resp, err := client.Login(username)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	cfg := &config.Config{
		APIKey:           resp.APIKey,
		UserEntityID:     resp.EntityID,
		Username:         resp.Username,
		Theme:            "dark",
		VimKeys:          true,
		QuickstartPending: true,
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(out, "logged in as %s\n", resp.Username)
	fmt.Fprintf(out, "config saved to %s\n", config.Path())
	return nil
}

// LoginCmd returns the `nebula login` command.
func LoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a Nebula server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return RunInteractiveLogin(os.Stdin, os.Stdout)
		},
	}
}
