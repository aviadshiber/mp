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
	queryCmd.AddCommand(newQueryFrequencyCmd())
}

func newQueryFrequencyCmd() *cobra.Command {
	var (
		from          string
		to            string
		unit          string
		addictionUnit string
		event         string
		where         string
		on            string
		limit         int
	)

	cmd := &cobra.Command{
		Use:   "frequency",
		Short: "Query event frequency (addiction) data",
		Long: `Query event frequency data from the Mixpanel analytics API. Shows how often
users perform an event within a given time period (also known as "addiction" report).`,
		Example: `  # Daily frequency breakdown for January 2024
  mp query frequency --from 2024-01-01 --to 2024-01-31 \
    --unit day --addiction-unit hour

  # Weekly frequency for a specific event
  mp query frequency --from 2024-01-01 --to 2024-03-31 \
    --unit week --addiction-unit day --event "Login"

  # With property breakdown
  mp query frequency --from 2024-01-01 --to 2024-01-31 \
    --unit day --addiction-unit hour --on 'properties["country"]'

  # JSON output
  mp query frequency --from 2024-01-01 --to 2024-01-31 \
    --unit day --addiction-unit hour --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryFrequency(cmd, from, to, unit, addictionUnit, event, where, on, limit)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&unit, "unit", "", "Time unit: day, week, month (required)")
	cmd.Flags().StringVar(&addictionUnit, "addiction-unit", "", "Frequency unit: hour, day (required)")
	cmd.Flags().StringVar(&event, "event", "", "Event name to analyze")
	cmd.Flags().StringVar(&where, "where", "", "Filter expression")
	cmd.Flags().StringVar(&on, "on", "", "Property expression for breakdown")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of breakdown values")

	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("unit")
	_ = cmd.MarkFlagRequired("addiction-unit")

	return cmd
}

func runQueryFrequency(cmd *cobra.Command, from, to, unit, addictionUnit, event, where, on string, limit int) error {
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
	params.Set("unit", unit)
	params.Set("addiction_unit", addictionUnit)

	if event != "" {
		params.Set("event", event)
	}
	if where != "" {
		params.Set("where", where)
	}
	if on != "" {
		params.Set("on", on)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	resp, err := c.Get(client.APIFamilyQuery, "/retention/addiction", params)
	if err != nil {
		return fmt.Errorf("querying frequency: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing frequency response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderFrequencyTable(result)
}

// renderFrequencyTable renders frequency data as a table.
// Response shape: {"data": {"2024-01-01": [50, 30, 20, 10]}}
func renderFrequencyTable(result map[string]any) error {
	s := getIO()

	data, ok := result["data"].(map[string]any)
	if !ok {
		return output.PrintJSON(s.Out, result)
	}

	if len(data) == 0 {
		s.Printf("No frequency data returned.\n")
		return nil
	}

	// Collect and sort dates, find max frequency buckets.
	dates := make([]string, 0, len(data))
	maxBuckets := 0
	for date, v := range data {
		dates = append(dates, date)
		if buckets, ok := v.([]any); ok && len(buckets) > maxBuckets {
			maxBuckets = len(buckets)
		}
	}
	sort.Strings(dates)

	// Build headers: DATE | FREQ 0 | FREQ 1 | ...
	headers := make([]string, 0, 1+maxBuckets)
	headers = append(headers, "DATE")
	for i := 0; i < maxBuckets; i++ {
		headers = append(headers, fmt.Sprintf("FREQ %d", i))
	}

	rows := make([][]string, 0, len(dates))
	for _, date := range dates {
		buckets, ok := data[date].([]any)
		if !ok {
			continue
		}

		row := make([]string, 0, 1+maxBuckets)
		row = append(row, date)
		for i := 0; i < maxBuckets; i++ {
			val := ""
			if i < len(buckets) {
				val = fmt.Sprintf("%v", buckets[i])
			}
			row = append(row, val)
		}
		rows = append(rows, row)
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}
