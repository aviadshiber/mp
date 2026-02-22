package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCohortsCmd())
}

func newCohortsCmd() *cobra.Command {
	cohortsCmd := &cobra.Command{
		Use:   "cohorts",
		Short: "Manage cohorts",
		Long:  "List and inspect Mixpanel cohorts.",
	}

	cohortsCmd.AddCommand(newCohortsListCmd())
	return cohortsCmd
}

func newCohortsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all cohorts in the project",
		Long:  "List all cohorts in the current project with their IDs, names, counts, and descriptions.",
		Example: `  # List all cohorts
  mp cohorts list

  # JSON output
  mp cohorts list --json

  # Filter with jq
  mp cohorts list --json --jq '.[].name'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCohortsList(cmd)
		},
	}
	return cmd
}

func runCohortsList(cmd *cobra.Command) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}

	resp, err := c.Post(client.APIFamilyQuery, "/cohorts/list", params)
	if err != nil {
		return fmt.Errorf("listing cohorts: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var cohorts []map[string]any
	if err := json.Unmarshal(body, &cohorts); err != nil {
		return fmt.Errorf("parsing cohorts response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, cohorts)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderCohortsList(cohorts)
}

func renderCohortsList(cohorts []map[string]any) error {
	s := getIO()

	if len(cohorts) == 0 {
		s.Printf("No cohorts found.\n")
		return nil
	}

	// Sort by ID for consistent output.
	sort.Slice(cohorts, func(i, j int) bool {
		idI, _ := cohorts[i]["id"].(float64)
		idJ, _ := cohorts[j]["id"].(float64)
		return idI < idJ
	})

	headers := []string{"ID", "NAME", "COUNT", "CREATED", "DESCRIPTION"}
	rows := make([][]string, 0, len(cohorts))

	for _, c := range cohorts {
		id := fmt.Sprintf("%.0f", c["id"])
		name, _ := c["name"].(string)
		count := fmt.Sprintf("%v", c["count"])
		created, _ := c["created"].(string)
		desc, _ := c["description"].(string)

		rows = append(rows, []string{id, name, count, created, desc})
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}
