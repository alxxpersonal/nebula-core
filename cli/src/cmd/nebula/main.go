package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/cmd"
	"github.com/gravitrone/nebula-core/cli/internal/config"
	"github.com/gravitrone/nebula-core/cli/internal/ui"
)

func main() {
	root := &cobra.Command{
		Use:   "nebula",
		Short: "Nebula - agent context layer",
		Long:  "Nebula CLI: manage entities, approve agent actions, add knowledge, and monitor jobs.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTUI()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(cmd.LoginCmd())
	root.AddCommand(cmd.AgentCmd())
	root.AddCommand(cmd.KeysCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Force truecolor so hex colors render correctly
	// Must be set before any lipgloss style initialization
	os.Setenv("COLORTERM", "truecolor")
}

func runTUI() error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("not logged in. run 'nebula login' first.")
		return err
	}

	client := api.NewDefaultClient(cfg.APIKey)
	app := ui.NewApp(client, cfg)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui error: %w", err)
	}
	return nil
}
