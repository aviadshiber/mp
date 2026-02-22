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
	queryCmd.AddCommand(newQueryEventsCmd())
}

func newQueryEventsCmd() *cobra.Command {
	var (
		event     string
		queryType string
		unit      string
		from      string
		to        string
	)

	cmd := &cobra.Command{
		Use:   "events",
		Short: "Query aggregate event counts over time",
		Long: `Query aggregate event counts from the Mixpanel analytics API. Returns counts
for one or more events broken down by the specified time unit.`,
		Example: `  # Daily signups and logins for January 2024
  mp query events --event "Signup,Login" --type general --unit day \
    --from 2024-01-01 --to 2024-01-31

  # Weekly unique purchases
  mp query events --event "Purchase" --type unique --unit week \
    --from 2024-01-01 --to 2024-03-31

  # Monthly event counts as JSON
  mp query events --event "Signup" --type general --unit month \
    --from 2024-01-01 --to 2024-12-31 --json

  # Filter with jq
  mp query events --event "Signup" --type general --unit day \
    --from 2024-01-01 --to 2024-01-31 --json --jq '.data.values'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryEvents(cmd, event, queryType, unit, from, to)
		},
	}

	cmd.Flags().StringVar(&event, "event", "", "Comma-separated event names (required)")
	cmd.Flags().StringVar(&queryType, "type", "", "Aggregation type: general, unique, average (required)")
	cmd.Flags().StringVar(&unit, "unit", "", "Time unit: minute, hour, day, week, month (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date yyyy-mm-dd (required)")

	_ = cmd.MarkFlagRequired("event")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("unit")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runQueryEvents(cmd *cobra.Command, event, queryType, unit, from, to string) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	events := splitCSV(event)
	if len(events) == 0 {
		return fmt.Errorf("`--event` must specify at least one event name")
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}
	params.Set("event", toJSONArray(events))
	params.Set("type", queryType)
	params.Set("unit", unit)
	params.Set("from_date", from)
	params.Set("to_date", to)

	resp, err := c.Get(client.APIFamilyQuery, "/events", params)
	if err != nil {
		return fmt.Errorf("querying events: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing events response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderEventsTable(result, events)
}

// renderEventsTable renders event query results as a table with one column per event.
// Response shape: {"data": {"series": [...dates], "values": {eventName: {date: count}}}}
func renderEventsTable(result map[string]any, requestedEvents []string) error {
	s := getIO()

	data, ok := result["data"].(map[string]any)
	if !ok {
		return output.PrintJSON(s.Out, result)
	}

	seriesRaw, _ := data["series"].([]any)
	valuesRaw, _ := data["values"].(map[string]any)

	if len(seriesRaw) == 0 {
		s.Printf("No data returned.\n")
		return nil
	}

	dates := make([]string, 0, len(seriesRaw))
	for _, d := range seriesRaw {
		dates = append(dates, fmt.Sprintf("%v", d))
	}

	// Determine event columns: use the order from the response values,
	// sorted for consistency.
	eventNames := make([]string, 0, len(valuesRaw))
	for name := range valuesRaw {
		eventNames = append(eventNames, name)
	}
	sort.Strings(eventNames)

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
			if evData, ok := valuesRaw[name].(map[string]any); ok {
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
