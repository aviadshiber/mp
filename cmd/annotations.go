package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newAnnotationsCmd())
}

func newAnnotationsCmd() *cobra.Command {
	annotationsCmd := &cobra.Command{
		Use:   "annotations",
		Short: "Manage project annotations",
		Long:  "List and inspect annotations (notes) attached to dates in your Mixpanel project.",
	}

	annotationsCmd.AddCommand(newAnnotationsListCmd())
	annotationsCmd.AddCommand(newAnnotationsGetCmd())
	return annotationsCmd
}

func newAnnotationsListCmd() *cobra.Command {
	var (
		from string
		to   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List annotations",
		Long:  "List annotations in the project, optionally filtered by date range.",
		Example: `  # List all annotations
  mp annotations list

  # List annotations for a date range
  mp annotations list --from 2024-01-01 --to 2024-01-31

  # JSON output
  mp annotations list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnnotationsList(cmd, from, to)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date yyyy-mm-dd (optional)")
	cmd.Flags().StringVar(&to, "to", "", "End date yyyy-mm-dd (optional)")

	return cmd
}

func runAnnotationsList(cmd *cobra.Command, from, to string) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	pid, err := requireProjectID()
	if err != nil {
		return err
	}

	params := url.Values{}
	if from != "" {
		params.Set("fromDate", from)
	}
	if to != "" {
		params.Set("toDate", to)
	}

	path := fmt.Sprintf("/projects/%s/annotations", pid)
	resp, err := c.Get(client.APIFamilyApp, path, params)
	if err != nil {
		return fmt.Errorf("listing annotations: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing annotations response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderAnnotationsList(result)
}

func renderAnnotationsList(result map[string]any) error {
	s := getIO()

	resultsRaw, ok := result["results"].([]any)
	if !ok || len(resultsRaw) == 0 {
		s.Printf("No annotations found.\n")
		return nil
	}

	headers := []string{"ID", "DATE", "DESCRIPTION"}
	rows := make([][]string, 0, len(resultsRaw))

	for _, r := range resultsRaw {
		ann, ok := r.(map[string]any)
		if !ok {
			continue
		}
		id := fmt.Sprintf("%.0f", ann["id"])
		date, _ := ann["date"].(string)
		desc, _ := ann["description"].(string)

		rows = append(rows, []string{id, date, desc})
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}

func newAnnotationsGetCmd() *cobra.Command {
	var annotationID int

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a specific annotation by ID",
		Long:  "Get details for a specific annotation by its ID.",
		Example: `  # Get annotation details
  mp annotations get --id 42

  # JSON output
  mp annotations get --id 42 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnnotationsGet(cmd, annotationID)
		},
	}

	cmd.Flags().IntVar(&annotationID, "id", 0, "Annotation ID (required)")
	_ = cmd.MarkFlagRequired("id")

	return cmd
}

func runAnnotationsGet(cmd *cobra.Command, annotationID int) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	pid, err := requireProjectID()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/projects/%s/annotations/%d", pid, annotationID)
	resp, err := c.Get(client.APIFamilyApp, path, nil)
	if err != nil {
		return fmt.Errorf("getting annotation: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing annotation response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	// Render single annotation as a simple table.
	return renderAnnotationsList(result)
}
