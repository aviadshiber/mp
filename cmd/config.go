package cmd

import (
	"fmt"

	"github.com/aviadshiber/mp/internal/config"
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mp configuration",
		Long: `Get, set, and list configuration values stored in ~/.config/mp/config.yaml.

Valid keys: project_id, region, service_account, service_secret`,
	}

	configCmd.AddCommand(newConfigSetCmd())
	configCmd.AddCommand(newConfigGetCmd())
	configCmd.AddCommand(newConfigListCmd())

	return configCmd
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.New()
			if err != nil {
				return err
			}

			key, value := args[0], args[1]
			if err := cfg.Set(key, value); err != nil {
				return err
			}

			s := getIO()
			s.Printf("%s %s=%s\n", s.Success(""),
				s.Bold(key), value)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.New()
			if err != nil {
				return err
			}

			val := cfg.Get(args[0])
			if val == "" {
				return fmt.Errorf("key %q is not set; run: mp config set %s <value>", args[0], args[0])
			}

			s := getIO()
			s.Printf("%s\n", val)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.New()
			if err != nil {
				return err
			}

			entries := cfg.List()
			s := getIO()

			if jsonOutputRequested(cmd) {
				return output.PrintJSON(s.Out, entries)
			}

			if len(entries) == 0 {
				s.Printf("%s\n", s.Muted("No configuration set. Run: mp config set <key> <value>"))
				s.Printf("%s %s\n", s.Muted("Config file:"), cfg.FilePath())
				return nil
			}

			headers := []string{"KEY", "VALUE"}
			rows := make([][]string, len(entries))
			for i, e := range entries {
				rows[i] = []string{e.Key, e.Value}
			}

			output.PrintTable(s.Out, headers, rows, s.IsTerminal())
			s.Printf("\n%s %s\n", s.Muted("Config file:"), cfg.FilePath())
			return nil
		},
	}
}
