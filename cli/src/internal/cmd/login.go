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

			fmt.Print("server url (e.g. http://localhost:8000): ")
			serverURL, _ := reader.ReadString('\n')
			serverURL = strings.TrimSpace(serverURL)

			fmt.Print("username: ")
			username, _ := reader.ReadString('\n')
			username = strings.TrimSpace(username)

			if serverURL == "" || username == "" {
				return fmt.Errorf("server url and username are required")
			}

			client := api.NewClient(serverURL, "")
			resp, err := client.Login(username)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			cfg := &config.Config{
				ServerURL:    serverURL,
				APIKey:       resp.APIKey,
				UserEntityID: resp.EntityID,
				Username:     resp.Username,
				Theme:        "dark",
				VimKeys:      true,
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
