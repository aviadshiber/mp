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
	queryCmd.AddCommand(newQueryInsightsCmd())
}

func newQueryInsightsCmd() *cobra.Command {
	var bookmarkID int

	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Query a saved Insights report",
		Long: `Query a saved Insights report by its bookmark ID. Returns the computed series
data for the report.`,
		Example: `  # Query a saved insight
  mp query insights --bookmark-id 12345

  # JSON output
  mp query insights --bookmark-id 12345 --json

  # Filter with jq
  mp query insights --bookmark-id 12345 --json --jq '.series'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryInsights(cmd, bookmarkID)
		},
	}

	cmd.Flags().IntVar(&bookmarkID, "bookmark-id", 0, "Saved report bookmark ID (required)")
	_ = cmd.MarkFlagRequired("bookmark-id")

	return cmd
}

func runQueryInsights(cmd *cobra.Command, bookmarkID int) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}
	params.Set("bookmark_id", fmt.Sprintf("%d", bookmarkID))

	resp, err := c.Get(client.APIFamilyQuery, "/insights", params)
	if err != nil {
		return fmt.Errorf("querying insights: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing insights response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderInsightsTable(result)
}

// renderInsightsTable renders insights data as a table.
// Response shape: {"series": {eventName: {date: count}}, "headers": [...dates], ...}
func renderInsightsTable(result map[string]any) error {
	s := getIO()

	series, ok := result["series"].(map[string]any)
	if !ok {
		return output.PrintJSON(s.Out, result)
	}

	// Get dates from headers if available, otherwise from the series data.
	var dates []string
	if headersRaw, ok := result["headers"].([]any); ok {
		for _, h := range headersRaw {
			dates = append(dates, fmt.Sprintf("%v", h))
		}
	}

	// Get sorted event names.
	eventNames := make([]string, 0, len(series))
	for name := range series {
		eventNames = append(eventNames, name)
	}
	sort.Strings(eventNames)

	// If no headers, discover dates from the first event's data.
	if len(dates) == 0 && len(eventNames) > 0 {
		if evData, ok := series[eventNames[0]].(map[string]any); ok {
			for d := range evData {
				dates = append(dates, d)
			}
			sort.Strings(dates)
		}
	}

	if len(dates) == 0 || len(eventNames) == 0 {
		s.Printf("No insights data returned.\n")
		return nil
	}

	// Build headers: DATE + one column per event.
	headers := make([]string, 0, 1+len(eventNames))
	headers = append(headers, "DATE")
	headers = append(headers, eventNames...)

	rows := make([][]string, 0, len(dates))
	for _, date := range dates {
		row := make([]string, 0, 1+len(eventNames))
		row = append(row, date)
		for _, name := range eventNames {
			val := "0"
			if evData, ok := series[name].(map[string]any); ok {
				if v, exists := evData[date]; exists {
					val = fmt.Sprintf("%v", v)
				}
			}
			row = append(row, val)
		}
		rows = append(rows, row)
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}
