// Package cmd defines the CLI commands for the mp tool.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/aviadshiber/mp/internal/iostreams"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// versionInfo is set by main via SetVersionInfo.
	versionInfo struct {
		version string
		commit  string
		date    string
	}

	// Global flag values bound to viper.
	cfgProjectID string
	cfgRegion    string
	cfgQuiet     bool
	cfgJSON      string
	cfgJQ        string
	cfgTemplate  string

	io *iostreams.IOStreams
)

// SetVersionInfo stores build metadata for the version command.
func SetVersionInfo(version, commit, date string) {
	versionInfo.version = version
	versionInfo.commit = commit
	versionInfo.date = date
}

var rootCmd = &cobra.Command{
	Use:   "mp",
	Short: "Mixpanel CLI - query, export, and manage Mixpanel data",
	Long: `mp is a command-line tool for interacting with the Mixpanel API.

It supports querying analytics, exporting raw events, managing user profiles,
and inspecting project metadata. Output can be formatted as JSON, tables, CSV,
or filtered with jq expressions and Go templates.

Configuration is stored in ~/.config/mp/config.yaml and can be overridden
with flags or environment variables (MP_PROJECT_ID, MP_REGION, MP_TOKEN).`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		io = iostreams.New()
		io.SetQuiet(viper.GetBool("quiet"))

		// Validate region if provided.
		region := viper.GetString("region")
		if region != "" {
			region = strings.ToLower(region)
			if region != "us" && region != "eu" && region != "in" {
				return fmt.Errorf("invalid region %q; must be one of: us, eu, in", region)
			}
		}
		return nil
	},
}

func init() {
	// Load config file into global viper.
	home, _ := os.UserHomeDir()
	if home != "" {
		viper.SetConfigFile(home + "/.config/mp/config.yaml")
		viper.SetConfigType("yaml")
		_ = viper.ReadInConfig() // Ignore error if file doesn't exist yet.
	}

	// Bind env vars before flag parsing.
	viper.SetEnvPrefix("MP")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Persistent flags available to all subcommands.
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&cfgProjectID, "project-id", "p", "", "Mixpanel project ID (env: MP_PROJECT_ID)")
	pf.StringVarP(&cfgRegion, "region", "r", "", "API region: us, eu, in (env: MP_REGION)")
	pf.BoolVarP(&cfgQuiet, "quiet", "q", false, "Suppress non-essential output (env: MP_QUIET)")
	pf.StringVar(&cfgJSON, "json", "", "Output JSON; optionally comma-separated field list")
	pf.StringVar(&cfgJQ, "jq", "", "Filter JSON output with a jq expression (requires --json)")
	pf.StringVar(&cfgTemplate, "template", "", "Format output with a Go template (requires --json)")

	// Allow --json to be used without a value (e.g., "mp version --json").
	pf.Lookup("json").NoOptDefVal = " "

	// Bind flags to viper keys so env vars and config file values also work.
	_ = viper.BindPFlag("project_id", pf.Lookup("project-id"))
	_ = viper.BindPFlag("region", pf.Lookup("region"))
	_ = viper.BindPFlag("quiet", pf.Lookup("quiet"))

	// Register subcommands.
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newConfigCmd())
}

// Execute runs the root command. Called from main.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		// Print error in red to stderr.
		s := iostreams.New()
		fmt.Fprintln(s.ErrOut, s.Failure("Error: "+err.Error()))
		return err
	}
	return nil
}

// getIO returns the current IOStreams instance, initializing if needed.
func getIO() *iostreams.IOStreams {
	if io == nil {
		io = iostreams.New()
	}
	return io
}

// isDebug reports whether debug mode is enabled via MP_DEBUG env var.
func isDebug() bool {
	return os.Getenv("MP_DEBUG") == "1"
}

// jsonOutputRequested reports whether the --json flag was explicitly set.
func jsonOutputRequested(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("json")
}
