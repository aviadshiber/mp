package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newActivityCmd())
}

func newActivityCmd() *cobra.Command {
	var (
		distinctIDs string
		from        string
		to          string
	)

	cmd := &cobra.Command{
		Use:   "activity",
		Short: "Query user activity stream",
		Long: `Query the activity stream for specific users. Shows recent events performed
by one or more users identified by their distinct IDs.`,
		Example: `  # Activity for a single user
  mp activity --distinct-ids "user123" --from 2024-01-01 --to 2024-01-31

  # Activity for multiple users
  mp activity --distinct-ids "user1,user2,user3" --from 2024-01-01 --to 2024-01-31

  # JSON output
  mp activity --distinct-ids "user123" --from 2024-01-01 --to 2024-01-31 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runActivity(cmd, distinctIDs, from, to)
		},
	}

	cmd.Flags().StringVar(&distinctIDs, "distinct-ids", "", "Comma-separated distinct IDs (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date yyyy-mm-dd (required)")

	_ = cmd.MarkFlagRequired("distinct-ids")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runActivity(cmd *cobra.Command, distinctIDs, from, to string) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	ids := splitCSV(distinctIDs)
	if len(ids) == 0 {
		return fmt.Errorf("`--distinct-ids` must specify at least one ID")
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}
	params.Set("distinct_ids", toJSONArray(ids))
	params.Set("from_date", from)
	params.Set("to_date", to)

	resp, err := c.Get(client.APIFamilyQuery, "/stream/query", params)
	if err != nil {
		return fmt.Errorf("querying activity stream: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing activity response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderActivityTable(result)
}

// renderActivityTable renders the activity stream as a table.
// Response shape: {"results": {"events": [{"event": "Page View", "properties": {"time": 1704067200, ...}}]}}
func renderActivityTable(result map[string]any) error {
	s := getIO()

	results, ok := result["results"].(map[string]any)
	if !ok {
		return output.PrintJSON(s.Out, result)
	}

	eventsRaw, ok := results["events"].([]any)
	if !ok || len(eventsRaw) == 0 {
		s.Printf("No activity found.\n")
		return nil
	}

	// Discover key properties from the first few events for column display.
	keyProps := discoverKeyProperties(eventsRaw)

	headers := make([]string, 0, 2+len(keyProps))
	headers = append(headers, "TIME", "EVENT")
	headers = append(headers, keyProps...)

	rows := make([][]string, 0, len(eventsRaw))
	for _, evRaw := range eventsRaw {
		ev, ok := evRaw.(map[string]any)
		if !ok {
			continue
		}

		eventName, _ := ev["event"].(string)
		props, _ := ev["properties"].(map[string]any)

		// Format time.
		timeStr := ""
		if t, ok := props["time"].(float64); ok {
			timeStr = time.Unix(int64(t), 0).UTC().Format("2006-01-02 15:04:05")
		} else if ts, ok := props["time"].(string); ok {
			timeStr = ts
		}

		row := make([]string, 0, 2+len(keyProps))
		row = append(row, timeStr, eventName)
		for _, p := range keyProps {
			val := ""
			if v, ok := props[p]; ok && v != nil {
				val = fmt.Sprintf("%v", v)
			}
			row = append(row, val)
		}
		rows = append(rows, row)
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	s.Printf("\n%s %d events\n", s.Muted("Showing"), len(rows))
	return nil
}

// discoverKeyProperties examines the first few events and returns the most
// common non-internal property names (excluding time, distinct_id, etc.).
func discoverKeyProperties(events []any) []string {
	// Properties to exclude from auto-discovery (internal/common).
	exclude := map[string]bool{
		"time": true, "distinct_id": true, "$distinct_id": true,
		"$import": true, "$insert_id": true, "mp_processing_time_ms": true,
	}

	propCount := make(map[string]int)
	scanCount := len(events)
	if scanCount > 20 {
		scanCount = 20
	}

	for i := 0; i < scanCount; i++ {
		ev, ok := events[i].(map[string]any)
		if !ok {
			continue
		}
		props, ok := ev["properties"].(map[string]any)
		if !ok {
			continue
		}
		for k := range props {
			if !exclude[k] {
				propCount[k]++
			}
		}
	}

	// Sort by frequency, pick top 5.
	type propFreq struct {
		name  string
		count int
	}
	freqs := make([]propFreq, 0, len(propCount))
	for k, v := range propCount {
		freqs = append(freqs, propFreq{k, v})
	}
	sort.Slice(freqs, func(i, j int) bool {
		if freqs[i].count != freqs[j].count {
			return freqs[i].count > freqs[j].count
		}
		return freqs[i].name < freqs[j].name
	})

	maxProps := 5
	if len(freqs) < maxProps {
		maxProps = len(freqs)
	}

	result := make([]string, 0, maxProps)
	for i := 0; i < maxProps; i++ {
		result = append(result, freqs[i].name)
	}
	return result
}
