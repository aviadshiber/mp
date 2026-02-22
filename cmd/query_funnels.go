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
	queryCmd.AddCommand(newQueryFunnelsCmd())
}

func newQueryFunnelsCmd() *cobra.Command {
	funnelsCmd := &cobra.Command{
		Use:   "funnels",
		Short: "Query funnel conversion data",
		Long: `Query funnel conversion data from the Mixpanel analytics API. Shows step-by-step
conversion rates for a configured funnel.

Use "mp query funnels list" to see available funnels and their IDs.`,
	}

	funnelsCmd.AddCommand(newFunnelsQueryCmd())
	funnelsCmd.AddCommand(newFunnelsListCmd())

	return funnelsCmd
}

func newFunnelsQueryCmd() *cobra.Command {
	var (
		funnelID   int
		from       string
		to         string
		length     int
		lengthUnit string
		unit       string
		on         string
		where      string
		limit      int
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query a specific funnel by ID",
		Long: `Query a specific funnel by its ID. Returns step-by-step conversion data
broken down by date.`,
		Example: `  # Query funnel for January 2024
  mp query funnels query --funnel-id 7509 --from 2024-01-01 --to 2024-01-31

  # With conversion window and time unit
  mp query funnels query --funnel-id 7509 --from 2024-01-01 --to 2024-01-31 \
    --length 14 --length-unit day --unit week

  # Breakdown by property
  mp query funnels query --funnel-id 7509 --from 2024-01-01 --to 2024-01-31 \
    --on 'properties["country"]'

  # JSON output
  mp query funnels query --funnel-id 7509 --from 2024-01-01 --to 2024-01-31 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFunnelsQuery(cmd, funnelID, from, to, length, lengthUnit, unit, on, where, limit)
		},
	}

	cmd.Flags().IntVar(&funnelID, "funnel-id", 0, "Funnel ID (required; use 'funnels list' to find IDs)")
	cmd.Flags().StringVar(&from, "from", "", "Start date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date yyyy-mm-dd (required)")
	cmd.Flags().IntVar(&length, "length", 0, "Conversion window length")
	cmd.Flags().StringVar(&lengthUnit, "length-unit", "", "Conversion window unit: second, minute, hour, day")
	cmd.Flags().StringVar(&unit, "unit", "", "Time unit: day, week, month")
	cmd.Flags().StringVar(&on, "on", "", "Property expression for breakdown")
	cmd.Flags().StringVar(&where, "where", "", "Filter expression")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of breakdown values (max 10000, default 255)")

	_ = cmd.MarkFlagRequired("funnel-id")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runFunnelsQuery(cmd *cobra.Command, funnelID int, from, to string, length int, lengthUnit, unit, on, where string, limit int) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}
	params.Set("funnel_id", fmt.Sprintf("%d", funnelID))
	params.Set("from_date", from)
	params.Set("to_date", to)

	if length > 0 {
		params.Set("length", fmt.Sprintf("%d", length))
	}
	if lengthUnit != "" {
		params.Set("length_unit", lengthUnit)
	}
	if unit != "" {
		params.Set("unit", unit)
	}
	if on != "" {
		params.Set("on", on)
	}
	if where != "" {
		params.Set("where", where)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	resp, err := c.Get(client.APIFamilyQuery, "/funnels", params)
	if err != nil {
		return fmt.Errorf("querying funnels: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing funnels response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderFunnelTable(result)
}

// renderFunnelTable renders funnel step data as a table showing step name,
// count, overall conversion %, and step conversion %.
func renderFunnelTable(result map[string]any) error {
	s := getIO()

	// The response has {"data": {date: {"steps": [...]}}, "meta": {"dates": [...]}}
	data, ok := result["data"].(map[string]any)
	if !ok {
		return output.PrintJSON(s.Out, result)
	}

	// Find the latest date's steps to show the overall funnel.
	meta, _ := result["meta"].(map[string]any)
	dates, _ := meta["dates"].([]any)

	if len(dates) == 0 {
		// Try using data keys directly.
		for k := range data {
			dates = append(dates, k)
		}
	}
	if len(dates) == 0 {
		s.Printf("No data returned.\n")
		return nil
	}

	// Use the last date.
	latestDate := fmt.Sprintf("%v", dates[len(dates)-1])
	dateData, ok := data[latestDate].(map[string]any)
	if !ok {
		// Try to find any date with data.
		for _, d := range dates {
			ds := fmt.Sprintf("%v", d)
			if dd, ok2 := data[ds].(map[string]any); ok2 {
				dateData = dd
				latestDate = ds
				break
			}
		}
		if dateData == nil {
			s.Printf("No funnel data found.\n")
			return nil
		}
	}

	steps, ok := dateData["steps"].([]any)
	if !ok || len(steps) == 0 {
		s.Printf("No funnel steps found.\n")
		return nil
	}

	headers := []string{"STEP", "EVENT", "COUNT", "OVERALL %", "STEP %"}
	rows := make([][]string, 0, len(steps))

	for i, stepRaw := range steps {
		step, ok := stepRaw.(map[string]any)
		if !ok {
			continue
		}

		eventName, _ := step["event"].(string)
		count, _ := step["count"].(float64)
		overallPct, _ := step["overall_conv_ratio"].(float64)
		stepPct, _ := step["step_conv_ratio"].(float64)

		row := []string{
			fmt.Sprintf("%d", i+1),
			eventName,
			fmt.Sprintf("%.0f", count),
			fmt.Sprintf("%.1f%%", overallPct*100),
			fmt.Sprintf("%.1f%%", stepPct*100),
		}
		rows = append(rows, row)
	}

	s.Printf("Funnel data for %s:\n\n", latestDate)
	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}

func newFunnelsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all saved funnels",
		Long:  "List all saved funnels in the project with their IDs and names.",
		Example: `  # List all funnels
  mp query funnels list

  # JSON output
  mp query funnels list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFunnelsList(cmd)
		},
	}
	return cmd
}

func runFunnelsList(cmd *cobra.Command) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}

	resp, err := c.Get(client.APIFamilyQuery, "/funnels/list", params)
	if err != nil {
		return fmt.Errorf("listing funnels: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var funnels []map[string]any
	if err := json.Unmarshal(body, &funnels); err != nil {
		return fmt.Errorf("parsing funnels list response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, funnels)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderFunnelsList(funnels)
}

func renderFunnelsList(funnels []map[string]any) error {
	s := getIO()

	if len(funnels) == 0 {
		s.Printf("No funnels found.\n")
		return nil
	}

	// Sort by funnel_id for consistent output.
	sort.Slice(funnels, func(i, j int) bool {
		idI, _ := funnels[i]["funnel_id"].(float64)
		idJ, _ := funnels[j]["funnel_id"].(float64)
		return idI < idJ
	})

	headers := []string{"ID", "NAME"}
	rows := make([][]string, 0, len(funnels))

	for _, f := range funnels {
		id := fmt.Sprintf("%.0f", f["funnel_id"])
		name, _ := f["name"].(string)
		rows = append(rows, []string{id, name})
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}
