package cmd

import (
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of mp",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := getIO()

			if jsonOutputRequested(cmd) {
				data := map[string]any{
					"version": versionInfo.version,
					"commit":  versionInfo.commit,
					"date":    versionInfo.date,
				}
				return output.PrintJSON(s.Out, data)
			}

			s.Printf("mp version %s (commit: %s, built: %s)\n",
				versionInfo.version, versionInfo.commit, versionInfo.date)
			return nil
		},
	}
}
