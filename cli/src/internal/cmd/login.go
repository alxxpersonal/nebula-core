package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
)

// LoginCmd returns the `nebula login` command.
func LoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a Nebula server",
		RunE: func(_ *cobra.Command, _ []string) error {
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("username: ")
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
				APIKey:       resp.APIKey,
				UserEntityID: resp.EntityID,
				Username:     resp.Username,
				Theme:        "dark",
				VimKeys:      true,
				QuickstartPending: true,
			}

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Printf("logged in as %s\n", resp.Username)
			fmt.Printf("config saved to %s\n", config.Path())
			return nil
		},
	}
}
