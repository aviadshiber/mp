package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/spf13/cobra"
)

func init() {
	queryCmd.AddCommand(newQueryPropertiesCmd())
}

func newQueryPropertiesCmd() *cobra.Command {
	var (
		event     string
		from      string
		to        string
		on        string
		where     string
		queryType string
		unit      string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "properties",
		Short: "Query event properties over time",
		Long: `Query event property data from the Mixpanel analytics API. Returns property
values broken down by time, similar to segmentation but scoped to a single
event's properties.`,
		Example: `  # Daily breakdown of Signup by country
  mp query properties --event "Signup" --from 2024-01-01 --to 2024-01-31 \
    --on 'properties["country"]'

  # Unique purchase amounts per week
  mp query properties --event "Purchase" --from 2024-01-01 --to 2024-03-31 \
    --on 'properties["amount"]' --type unique --unit week

  # Filter by property value
  mp query properties --event "Page View" --from 2024-01-01 --to 2024-01-31 \
    --on 'properties["page"]' --where 'properties["country"]=="US"' --limit 50

  # JSON output with jq
  mp query properties --event "Signup" --from 2024-01-01 --to 2024-01-31 \
    --on 'properties["country"]' --json --jq '.data.values'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryProperties(cmd, event, from, to, on, where, queryType, unit, limit)
		},
	}

	cmd.Flags().StringVar(&event, "event", "", "Event name (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date yyyy-mm-dd (required)")
	cmd.Flags().StringVar(&on, "on", "", "Property expression for breakdown (e.g., properties[\"country\"])")
	cmd.Flags().StringVar(&where, "where", "", "Filter expression")
	cmd.Flags().StringVar(&queryType, "type", "", "Aggregation type: general, unique, average")
	cmd.Flags().StringVar(&unit, "unit", "", "Time unit: minute, hour, day, week, month")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of property values (max 10000)")

	_ = cmd.MarkFlagRequired("event")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runQueryProperties(cmd *cobra.Command, event, from, to, on, where, queryType, unit string, limit int) error {
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
	if where != "" {
		params.Set("where", where)
	}
	if queryType != "" {
		params.Set("type", queryType)
	}
	if unit != "" {
		params.Set("unit", unit)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	resp, err := c.Get(client.APIFamilyQuery, "/events/properties", params)
	if err != nil {
		return fmt.Errorf("querying event properties: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing properties response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	// Reuse the segmentation table renderer since the response shape is identical.
	return renderSegmentationTable(result)
}
