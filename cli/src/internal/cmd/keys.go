package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
)

// KeysCmd returns the `nebula keys` command group.
func KeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage API keys",
	}
	cmd.AddCommand(keysListCmd())
	cmd.AddCommand(keysCreateCmd())
	cmd.AddCommand(keysRevokeCmd())
	return cmd
}

func keysListCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("not logged in: %w", err)
			}
			client := api.NewDefaultClient(cfg.APIKey)

			var keys []api.APIKey
			if all {
				keys, err = client.ListAllKeys()
			} else {
				keys, err = client.ListKeys()
			}
			if err != nil {
				return fmt.Errorf("list keys: %w", err)
			}

			if len(keys) == 0 {
				fmt.Println("no keys found")
				return nil
			}

			for _, k := range keys {
				owner := k.Name
				if k.OwnerType == "agent" && k.AgentName != nil {
					owner = fmt.Sprintf("agent:%s", *k.AgentName)
				} else if k.EntityName != nil {
					owner = fmt.Sprintf("user:%s", *k.EntityName)
				}
				lastUsed := "never"
				if k.LastUsedAt != nil {
					lastUsed = k.LastUsedAt.Format("2006-01-02 15:04")
				}
				fmt.Printf("  %s  %s  (%s)  last used: %s\n", k.KeyPrefix+"...", k.Name, owner, lastUsed)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&all, "all", "a", false, "show all keys (user + agent)")
	return cmd
}

func keysCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("not logged in: %w", err)
			}
			client := api.NewDefaultClient(cfg.APIKey)

			resp, err := client.CreateKey(args[0])
			if err != nil {
				return fmt.Errorf("create key: %w", err)
			}

			fmt.Printf("key created: %s\n", resp.Name)
			fmt.Printf("api key: %s\n", resp.APIKey)
			fmt.Println("save this key - it won't be shown again")
			return nil
		},
	}
}

func keysRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <key-id>",
		Short: "Revoke an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("not logged in: %w", err)
			}
			client := api.NewDefaultClient(cfg.APIKey)

			if err := client.RevokeKey(args[0]); err != nil {
				return fmt.Errorf("revoke key: %w", err)
			}

			fmt.Println("key revoked")
			return nil
		},
	}
}
