package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newExportCmd())
}

func newExportCmd() *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export raw data from Mixpanel",
		Long:  "Export raw event data from your Mixpanel project.",
	}

	exportCmd.AddCommand(newExportEventsCmd())
	return exportCmd
}

func newExportEventsCmd() *cobra.Command {
	var (
		from  string
		to    string
		event string
		where string
		limit int
	)

	cmd := &cobra.Command{
		Use:   "events",
		Short: "Export raw events as JSONL",
		Long: `Export raw event data from Mixpanel. Returns one JSON object per line (JSONL)
by default, which is ideal for piping to other tools. Use --json to collect all
events into a JSON array instead.`,
		Example: `  # Export all events for January 2024
  mp export events --from 2024-01-01 --to 2024-01-31

  # Export specific events
  mp export events --from 2024-01-01 --to 2024-01-31 --event "Signup,Login"

  # Export with a filter expression
  mp export events --from 2024-01-01 --to 2024-01-31 --where 'properties["country"]=="US"'

  # Export as JSON array with jq filtering
  mp export events --from 2024-01-01 --to 2024-01-31 --json --jq '.[].event'

  # Limit the number of exported events
  mp export events --from 2024-01-01 --to 2024-01-31 --limit 1000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportEvents(cmd, from, to, event, where, limit)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (yyyy-mm-dd, required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (yyyy-mm-dd, required)")
	cmd.Flags().StringVar(&event, "event", "", "Comma-separated event names to filter")
	cmd.Flags().StringVar(&where, "where", "", "Filter expression (e.g., properties[\"country\"]==\"US\")")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of events to export (max 100000)")

	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runExportEvents(cmd *cobra.Command, from, to, event, where string, limit int) error {
	if limit < 0 || limit > 100000 {
		return fmt.Errorf("--limit must be between 0 and 100000")
	}

	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}
	params.Set("from_date", from)
	params.Set("to_date", to)

	if event != "" {
		events := splitCSV(event)
		params.Set("event", toJSONArray(events))
	}
	if where != "" {
		params.Set("where", where)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	resp, err := c.Get(client.APIFamilyExport, "/export", params)
	if err != nil {
		return fmt.Errorf("requesting event export: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := readResponseBody(resp.Body, resp.StatusCode)
		_ = body // error already formatted by readResponseBody
		return fmt.Errorf("API error (HTTP %d)", resp.StatusCode)
	}

	s := getIO()

	// If --json is requested, collect all lines into a JSON array.
	if jsonOutputRequested(cmd) {
		var records []map[string]any
		scanner := bufio.NewScanner(resp.Body)
		// Increase scanner buffer for large lines.
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var record map[string]any
			if err := json.Unmarshal(line, &record); err != nil {
				return fmt.Errorf("parsing JSONL line: %w", err)
			}
			records = append(records, record)
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading response stream: %w", err)
		}

		// Apply jq/template filters if provided.
		var data any = records
		handled, err := handleJSONOutput(cmd, data)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}
		return output.PrintJSON(s.Out, records)
	}

	// Default: stream JSONL directly to stdout.
	jw := output.NewJSONLWriter(s.Out)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			return fmt.Errorf("parsing JSONL line: %w", err)
		}
		if err := jw.Write(record); err != nil {
			return fmt.Errorf("writing JSONL output: %w", err)
		}
	}
	return scanner.Err()
}
