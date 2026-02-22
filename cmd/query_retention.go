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
	queryCmd.AddCommand(newQueryRetentionCmd())
}

func newQueryRetentionCmd() *cobra.Command {
	var (
		from          string
		to            string
		retentionType string
		bornEvent     string
		event         string
		bornWhere     string
		where         string
		interval      int
		intervalCount int
		unit          string
		on            string
		limit         int
	)

	cmd := &cobra.Command{
		Use:   "retention",
		Short: "Query user retention data",
		Long: `Query user retention data from the Mixpanel analytics API. Shows how many users
return to perform an action after their initial visit or signup.`,
		Example: `  # Basic retention for January 2024
  mp query retention --from 2024-01-01 --to 2024-01-31

  # Birth retention based on Signup event
  mp query retention --from 2024-01-01 --to 2024-01-31 \
    --retention-type birth --born-event "Signup" --event "Login"

  # Weekly retention with property breakdown
  mp query retention --from 2024-01-01 --to 2024-03-31 \
    --unit week --on 'properties["country"]'

  # Custom interval settings
  mp query retention --from 2024-01-01 --to 2024-01-31 \
    --interval 7 --interval-count 10

  # JSON output
  mp query retention --from 2024-01-01 --to 2024-01-31 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryRetention(cmd, from, to, retentionType, bornEvent, event,
				bornWhere, where, interval, intervalCount, unit, on, limit)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&retentionType, "retention-type", "", "Retention type: birth, compounded")
	cmd.Flags().StringVar(&bornEvent, "born-event", "", "Birth event name (for birth retention)")
	cmd.Flags().StringVar(&event, "event", "", "Return event name")
	cmd.Flags().StringVar(&bornWhere, "born-where", "", "Filter expression for birth event")
	cmd.Flags().StringVar(&where, "where", "", "Filter expression for return event")
	cmd.Flags().IntVar(&interval, "interval", 0, "Interval length in units")
	cmd.Flags().IntVar(&intervalCount, "interval-count", 0, "Number of intervals to show")
	cmd.Flags().StringVar(&unit, "unit", "", "Time unit: day, week, month")
	cmd.Flags().StringVar(&on, "on", "", "Property expression for breakdown")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of breakdown values")

	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runQueryRetention(cmd *cobra.Command, from, to, retentionType, bornEvent, event,
	bornWhere, where string, interval, intervalCount int, unit, on string, limit int) error {
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

	if retentionType != "" {
		params.Set("retention_type", retentionType)
	}
	if bornEvent != "" {
		params.Set("born_event", bornEvent)
	}
	if event != "" {
		params.Set("event", event)
	}
	if bornWhere != "" {
		params.Set("born_where", bornWhere)
	}
	if where != "" {
		params.Set("where", where)
	}
	if interval > 0 {
		params.Set("interval", fmt.Sprintf("%d", interval))
	}
	if intervalCount > 0 {
		params.Set("interval_count", fmt.Sprintf("%d", intervalCount))
	}
	if unit != "" {
		params.Set("unit", unit)
	}
	if on != "" {
		params.Set("on", on)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	resp, err := c.Get(client.APIFamilyQuery, "/retention", params)
	if err != nil {
		return fmt.Errorf("querying retention: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing retention response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderRetentionTable(result)
}

// renderRetentionTable renders retention data as a table.
// Response shape: {"2024-01-01": {"counts": [100, 50, 30], "first": 100}, ...}
func renderRetentionTable(result map[string]any) error {
	s := getIO()

	if len(result) == 0 {
		s.Printf("No retention data returned.\n")
		return nil
	}

	// Collect and sort dates.
	dates := make([]string, 0, len(result))
	maxCols := 0
	for date, v := range result {
		if _, ok := v.(map[string]any); !ok {
			continue
		}
		dates = append(dates, date)
		if entry, ok := v.(map[string]any); ok {
			if counts, ok := entry["counts"].([]any); ok && len(counts) > maxCols {
				maxCols = len(counts)
			}
		}
	}
	sort.Strings(dates)

	if len(dates) == 0 {
		s.Printf("No retention data returned.\n")
		return nil
	}

	// Build headers: DATE | FIRST | DAY 0 | DAY 1 | ...
	headers := make([]string, 0, 2+maxCols)
	headers = append(headers, "DATE", "FIRST")
	for i := 0; i < maxCols; i++ {
		headers = append(headers, fmt.Sprintf("DAY %d", i))
	}

	rows := make([][]string, 0, len(dates))
	for _, date := range dates {
		entry, ok := result[date].(map[string]any)
		if !ok {
			continue
		}

		first, _ := entry["first"].(float64)
		counts, _ := entry["counts"].([]any)

		row := make([]string, 0, 2+maxCols)
		row = append(row, date, fmt.Sprintf("%.0f", first))

		for i := 0; i < maxCols; i++ {
			val := ""
			if i < len(counts) {
				val = fmt.Sprintf("%v", counts[i])
			}
			row = append(row, val)
		}
		rows = append(rows, row)
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}
