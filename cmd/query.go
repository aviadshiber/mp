package cmd

import (
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Run analytics queries against Mixpanel",
	Long: `Run analytics queries against the Mixpanel Query API.

Available subcommands let you query event segmentation, aggregate event counts,
user properties, funnels, retention, and more.`,
}

func init() {
	rootCmd.AddCommand(queryCmd)
}
