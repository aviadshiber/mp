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
	rootCmd.AddCommand(newPipelinesCmd())
}

func newPipelinesCmd() *cobra.Command {
	pipelinesCmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Manage data pipelines",
		Long:  "List and inspect Mixpanel data pipeline jobs and their status.",
	}

	pipelinesCmd.AddCommand(newPipelinesListCmd())
	pipelinesCmd.AddCommand(newPipelinesStatusCmd())
	return pipelinesCmd
}

func newPipelinesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipeline jobs",
		Long:  "List all configured data pipeline jobs in the project.",
		Example: `  # List all pipelines
  mp pipelines list

  # JSON output
  mp pipelines list --json

  # Filter with jq
  mp pipelines list --json --jq '.. | .name? // empty'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipelinesList(cmd)
		},
	}
	return cmd
}

func runPipelinesList(cmd *cobra.Command) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}

	resp, err := c.Get(client.APIFamilyExport, "/nessie/pipeline/jobs", params)
	if err != nil {
		return fmt.Errorf("listing pipelines: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing pipelines response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderPipelinesList(result)
}

// renderPipelinesList renders pipeline jobs as a table.
// Response shape: {"projectId": [{"name": "...", "frequency": "...", "sync_enabled": "...", ...}]}
func renderPipelinesList(result any) error {
	s := getIO()

	// Collect all pipeline jobs from all project IDs.
	type pipelineJob struct {
		Name           string
		Frequency      string
		SyncEnabled    string
		LastDispatched string
	}

	var jobs []pipelineJob

	switch v := result.(type) {
	case map[string]any:
		for _, val := range v {
			jobList, ok := val.([]any)
			if !ok {
				continue
			}
			for _, jobRaw := range jobList {
				job, ok := jobRaw.(map[string]any)
				if !ok {
					continue
				}
				pj := pipelineJob{
					Name:           fmt.Sprintf("%v", job["name"]),
					Frequency:      fmt.Sprintf("%v", job["frequency"]),
					SyncEnabled:    fmt.Sprintf("%v", job["sync_enabled"]),
					LastDispatched: fmt.Sprintf("%v", job["last_dispatched"]),
				}
				jobs = append(jobs, pj)
			}
		}
	case []any:
		for _, jobRaw := range v {
			job, ok := jobRaw.(map[string]any)
			if !ok {
				continue
			}
			pj := pipelineJob{
				Name:           fmt.Sprintf("%v", job["name"]),
				Frequency:      fmt.Sprintf("%v", job["frequency"]),
				SyncEnabled:    fmt.Sprintf("%v", job["sync_enabled"]),
				LastDispatched: fmt.Sprintf("%v", job["last_dispatched"]),
			}
			jobs = append(jobs, pj)
		}
	}

	if len(jobs) == 0 {
		s.Printf("No pipeline jobs found.\n")
		return nil
	}

	// Sort by name for consistent output.
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].Name < jobs[j].Name
	})

	headers := []string{"NAME", "FREQUENCY", "SYNC ENABLED", "LAST DISPATCHED"}
	rows := make([][]string, 0, len(jobs))

	for _, job := range jobs {
		rows = append(rows, []string{job.Name, job.Frequency, job.SyncEnabled, job.LastDispatched})
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}

func newPipelinesStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show pipeline status",
		Long:  "Show the current status of data pipeline jobs.",
		Example: `  # Show pipeline status
  mp pipelines status

  # JSON output
  mp pipelines status --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipelinesStatus(cmd)
		},
	}
	return cmd
}

func runPipelinesStatus(cmd *cobra.Command) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}

	resp, err := c.Get(client.APIFamilyExport, "/nessie/pipeline/status", params)
	if err != nil {
		return fmt.Errorf("getting pipeline status: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing pipeline status response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	// Default: print as JSON since pipeline status structure varies.
	return output.PrintJSON(getIO().Out, result)
}
