package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newProfilesCmd())
}

func newProfilesCmd() *cobra.Command {
	profilesCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Query and manage user profiles",
		Long:  "Query, inspect, and manage Mixpanel user profiles (Engage API).",
	}

	profilesCmd.AddCommand(newProfilesQueryCmd())
	profilesCmd.AddCommand(newProfilesGroupsCmd())
	return profilesCmd
}

func newProfilesQueryCmd() *cobra.Command {
	var (
		where       string
		distinctID  string
		distinctIDs string
		properties  string
		cohortID    int
		limit       int
		pageSize    int
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query user profiles with auto-pagination",
		Long: `Query user profiles from the Mixpanel Engage API. Automatically paginates
through all matching results unless a --limit is specified.

Results are returned as a table by default showing distinct_id and selected
properties. Use --json for the full API response.`,
		Example: `  # Find a user by email
  mp profiles query --where 'user["$email"]=="alice@example.com"'

  # Look up a specific user
  mp profiles query --distinct-id user123

  # Query specific properties
  mp profiles query --properties '$email,$name,$last_seen' --limit 100

  # Query users in a cohort
  mp profiles query --cohort-id 67890 --limit 50

  # Multiple distinct IDs
  mp profiles query --distinct-ids "user1,user2,user3"

  # JSON output
  mp profiles query --where 'user["$city"]=="San Francisco"' --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesQuery(cmd, where, distinctID, distinctIDs, properties, cohortID, limit, pageSize)
		},
	}

	cmd.Flags().StringVar(&where, "where", "", "Filter expression (e.g., user[\"$email\"]==\"alice@example.com\")")
	cmd.Flags().StringVar(&distinctID, "distinct-id", "", "Single distinct ID to look up")
	cmd.Flags().StringVar(&distinctIDs, "distinct-ids", "", "Comma-separated list of distinct IDs")
	cmd.Flags().StringVar(&properties, "properties", "", "Comma-separated output property names (e.g., $email,$name)")
	cmd.Flags().IntVar(&cohortID, "cohort-id", 0, "Filter by cohort ID")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum total profiles to fetch (0 = all)")
	cmd.Flags().IntVar(&pageSize, "page-size", 1000, "Profiles per page (max 1000)")

	return cmd
}

// engageResponse represents one page of the Engage API response.
type engageResponse struct {
	Page      int              `json:"page"`
	PageSize  int              `json:"page_size"`
	SessionID string           `json:"session_id"`
	Status    string           `json:"status"`
	Total     int              `json:"total"`
	Results   []map[string]any `json:"results"`
}

func runProfilesQuery(cmd *cobra.Command, where, distinctID, distinctIDs, properties string, cohortID, limit, pageSize int) error {
	if pageSize < 1 || pageSize > 1000 {
		return fmt.Errorf("`--page-size` must be between 1 and 1000")
	}

	c, err := newClient()
	if err != nil {
		return err
	}

	// Build base form parameters.
	baseParams := url.Values{}
	if err := addProjectID(baseParams); err != nil {
		return err
	}

	if where != "" {
		baseParams.Set("where", where)
	}
	if distinctID != "" {
		baseParams.Set("distinct_id", distinctID)
	}
	if distinctIDs != "" {
		ids := splitCSV(distinctIDs)
		baseParams.Set("distinct_ids", toJSONArray(ids))
	}
	if properties != "" {
		props := splitCSV(properties)
		baseParams.Set("output_properties", toJSONArray(props))
	}
	if cohortID > 0 {
		cohortJSON, _ := json.Marshal(map[string]int{"id": cohortID})
		baseParams.Set("filter_by_cohort", string(cohortJSON))
	}
	baseParams.Set("page_size", strconv.Itoa(pageSize))

	// Auto-paginate.
	var allResults []map[string]any
	var sessionID string
	page := 0
	totalFromAPI := -1

	for {
		params := url.Values{}
		for k, v := range baseParams {
			params[k] = v
		}
		params.Set("page", strconv.Itoa(page))
		if sessionID != "" {
			params.Set("session_id", sessionID)
		}

		resp, err := c.Post(client.APIFamilyQuery, "/engage", params)
		if err != nil {
			return fmt.Errorf("querying profiles (page %d): %w", page, err)
		}

		body, err := readResponseBody(resp.Body, resp.StatusCode)
		if err != nil {
			return err
		}

		var pageResp engageResponse
		if err := json.Unmarshal(body, &pageResp); err != nil {
			return fmt.Errorf("parsing profiles response: %w", err)
		}

		if pageResp.Status != "ok" && pageResp.Status != "" {
			return fmt.Errorf("engage API returned status %q", pageResp.Status)
		}

		allResults = append(allResults, pageResp.Results...)
		sessionID = pageResp.SessionID
		if totalFromAPI < 0 {
			totalFromAPI = pageResp.Total
		}

		// Check if we have enough results or reached the end.
		if limit > 0 && len(allResults) >= limit {
			allResults = allResults[:limit]
			break
		}
		if len(allResults) >= totalFromAPI {
			break
		}
		if len(pageResp.Results) < pageSize {
			break
		}

		page++
	}

	// Build a combined response for JSON output.
	combined := map[string]any{
		"total":   totalFromAPI,
		"count":   len(allResults),
		"results": allResults,
	}

	handled, err := handleJSONOutput(cmd, combined)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	// Default: render table.
	return renderProfilesTable(allResults, properties)
}

// renderProfilesTable renders profile results as a table with distinct_id
// and selected property columns.
func renderProfilesTable(results []map[string]any, propertiesFlag string) error {
	s := getIO()

	if len(results) == 0 {
		s.Printf("No profiles found.\n")
		return nil
	}

	// Determine which property columns to show.
	requestedProps := splitCSV(propertiesFlag)

	// If no properties were explicitly requested, discover from the first few results.
	if len(requestedProps) == 0 {
		propSet := make(map[string]bool)
		scanCount := len(results)
		if scanCount > 10 {
			scanCount = 10
		}
		for i := 0; i < scanCount; i++ {
			if props, ok := results[i]["$properties"].(map[string]any); ok {
				for k := range props {
					propSet[k] = true
				}
			}
		}
		for k := range propSet {
			requestedProps = append(requestedProps, k)
		}
		sort.Strings(requestedProps)

		// Cap auto-discovered columns at a reasonable number.
		if len(requestedProps) > 10 {
			requestedProps = requestedProps[:10]
		}
	}

	headers := make([]string, 0, 1+len(requestedProps))
	headers = append(headers, "DISTINCT_ID")
	headers = append(headers, requestedProps...)

	rows := make([][]string, 0, len(results))
	for _, r := range results {
		did, _ := r["$distinct_id"].(string)
		row := make([]string, 0, 1+len(requestedProps))
		row = append(row, did)

		props, _ := r["$properties"].(map[string]any)
		for _, p := range requestedProps {
			val := ""
			if props != nil {
				if v, ok := props[p]; ok && v != nil {
					val = fmt.Sprintf("%v", v)
				}
			}
			row = append(row, val)
		}
		rows = append(rows, row)
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	s.Printf("\n%s %d profiles\n", s.Muted("Showing"), len(results))
	return nil
}
