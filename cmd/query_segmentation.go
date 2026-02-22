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
	queryCmd.AddCommand(newQuerySegmentationCmd())
}

func newQuerySegmentationCmd() *cobra.Command {
	var (
		event     string
		from      string
		to        string
		on        string
		unit      string
		where     string
		queryType string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "segmentation",
		Short: "Query event segmentation data",
		Long: `Query event segmentation from the Mixpanel analytics API. Returns event counts
broken down by time and optionally by a property (using --on).

This is the most commonly used analytics query in Mixpanel.`,
		Example: `  # Daily signups for January 2024
  mp query segmentation --event "Signup" --from 2024-01-01 --to 2024-01-31

  # Signups broken down by country
  mp query segmentation --event "Signup" --from 2024-01-01 --to 2024-01-31 \
    --on 'properties["country"]'

  # Unique logins per week
  mp query segmentation --event "Login" --from 2024-01-01 --to 2024-01-31 \
    --unit week --type unique

  # Filter by property and limit results
  mp query segmentation --event "Purchase" --from 2024-01-01 --to 2024-01-31 \
    --where 'properties["amount"] > 100' --limit 50

  # JSON output with jq
  mp query segmentation --event "Signup" --from 2024-01-01 --to 2024-01-31 --json \
    --jq '.data.values'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuerySegmentation(cmd, event, from, to, on, unit, where, queryType, limit)
		},
	}

	cmd.Flags().StringVar(&event, "event", "", "Event name to segment (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&on, "on", "", "Property expression for breakdown (e.g., properties[\"country\"])")
	cmd.Flags().StringVar(&unit, "unit", "", "Time unit: minute, hour, day, week, month")
	cmd.Flags().StringVar(&where, "where", "", "Filter expression")
	cmd.Flags().StringVar(&queryType, "type", "", "Aggregation type: general, unique, average")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of breakdown values (max 10000)")

	_ = cmd.MarkFlagRequired("event")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runQuerySegmentation(cmd *cobra.Command, event, from, to, on, unit, where, queryType string, limit int) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}
	params.Set("event", event)
	params.Set("from_date", from)
	params.Set("to_date", to)

	if on != "" {
		params.Set("on", on)
	}
	if unit != "" {
		params.Set("unit", unit)
	}
	if where != "" {
		params.Set("where", where)
	}
	if queryType != "" {
		params.Set("type", queryType)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	resp, err := c.Get(client.APIFamilyQuery, "/segmentation", params)
	if err != nil {
		return fmt.Errorf("querying segmentation: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing segmentation response: %w", err)
	}

	// Handle --json output (with optional jq/template).
	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	// Default: render as table.
	return renderSegmentationTable(result)
}

// renderSegmentationTable renders segmentation data as a human-readable table.
// The response shape is:
//
//	{"data": {"series": [...dates], "values": {segmentName: {date: count}}}}
func renderSegmentationTable(result map[string]any) error {
	s := getIO()

	data, ok := result["data"].(map[string]any)
	if !ok {
		return output.PrintJSON(s.Out, result)
	}

	seriesRaw, _ := data["series"].([]any)
	valuesRaw, _ := data["values"].(map[string]any)

	if len(seriesRaw) == 0 || len(valuesRaw) == 0 {
		s.Printf("No data returned.\n")
		return nil
	}

	// Build date list from series.
	dates := make([]string, 0, len(seriesRaw))
	for _, d := range seriesRaw {
		dates = append(dates, fmt.Sprintf("%v", d))
	}

	// Get sorted segment names.
	segments := make([]string, 0, len(valuesRaw))
	for seg := range valuesRaw {
		segments = append(segments, seg)
	}
	sort.Strings(segments)

	// If there is only one segment (no breakdown), show a simple Date | Count table.
	if len(segments) == 1 {
		headers := []string{"DATE", "COUNT"}
		segData, _ := valuesRaw[segments[0]].(map[string]any)
		rows := make([][]string, 0, len(dates))
		for _, date := range dates {
			count := "0"
			if v, exists := segData[date]; exists {
				count = fmt.Sprintf("%v", v)
			}
			rows = append(rows, []string{date, count})
		}
		output.PrintTable(s.Out, headers, rows, s.IsTerminal())
		return nil
	}

	// Multiple segments: show Segment | date1 | date2 | ...
	headers := make([]string, 0, 1+len(dates))
	headers = append(headers, "SEGMENT")
	headers = append(headers, dates...)

	rows := make([][]string, 0, len(segments))
	for _, seg := range segments {
		segData, _ := valuesRaw[seg].(map[string]any)
		row := make([]string, 0, 1+len(dates))
		row = append(row, seg)
		for _, date := range dates {
			val := "0"
			if v, exists := segData[date]; exists {
				val = fmt.Sprintf("%v", v)
			}
			row = append(row, val)
		}
		rows = append(rows, row)
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}
