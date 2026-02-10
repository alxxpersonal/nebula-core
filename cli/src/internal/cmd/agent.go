package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
)

// AgentCmd returns the `nebula agent` command group.
func AgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
	}
	cmd.AddCommand(agentRegisterCmd())
	cmd.AddCommand(agentListCmd())
	return cmd
}

func agentRegisterCmd() *cobra.Command {
	var desc string
	cmd := &cobra.Command{
		Use:   "register <name>",
		Short: "Register a new agent (creates approval request)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("not logged in: %w", err)
			}
			client := api.NewClient(cfg.ServerURL, cfg.APIKey)

			input := api.RegisterAgentInput{
				Name:            args[0],
				Description:     desc,
				RequestedScopes: []string{"public"},
			}

			resp, err := client.RegisterAgent(input)
			if err != nil {
				return fmt.Errorf("register agent: %w", err)
			}

			fmt.Printf("agent registered: %s\n", resp.AgentID)
			fmt.Printf("status: %s\n", resp.Status)
			fmt.Printf("approval request: %s\n", resp.ApprovalRequestID)
			fmt.Println("approve via 'nebula' inbox or API")
			return nil
		},
	}
	cmd.Flags().StringVarP(&desc, "description", "d", "", "agent description")
	return cmd
}

func agentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("not logged in: %w", err)
			}
			client := api.NewClient(cfg.ServerURL, cfg.APIKey)

			agents, err := client.ListAgents("active")
			if err != nil {
				return fmt.Errorf("list agents: %w", err)
			}

			if len(agents) == 0 {
				fmt.Println("no agents found")
				return nil
			}

			for _, a := range agents {
				trust := "trusted"
				if a.RequiresApproval {
					trust = "untrusted"
				}
				desc := ""
				if a.Description != nil {
					desc = " - " + *a.Description
				}
				fmt.Printf("  %s (%s)%s\n", a.Name, trust, desc)
			}
			return nil
		},
	}
}
