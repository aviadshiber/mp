package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/spf13/cobra"
)

func newProfilesGroupsCmd() *cobra.Command {
	var (
		groupKey    string
		where       string
		properties  string
		limit       int
		pageSize    int
	)

	cmd := &cobra.Command{
		Use:   "groups",
		Short: "Query group profiles with auto-pagination",
		Long: `Query group profiles from the Mixpanel Engage API. Works like "profiles query"
but targets a specific group analytics key (e.g., companies, accounts).

Automatically paginates through all matching results unless a --limit is specified.`,
		Example: `  # Query all company profiles
  mp profiles groups --group-key companies

  # Filter group profiles
  mp profiles groups --group-key companies \
    --where 'user["plan"]=="enterprise"'

  # Query specific properties
  mp profiles groups --group-key accounts \
    --properties 'name,plan,created' --limit 50

  # JSON output
  mp profiles groups --group-key companies --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesGroups(cmd, groupKey, where, properties, limit, pageSize)
		},
	}

	cmd.Flags().StringVar(&groupKey, "group-key", "", "Group analytics key (e.g., companies, accounts) (required)")
	cmd.Flags().StringVar(&where, "where", "", "Filter expression")
	cmd.Flags().StringVar(&properties, "properties", "", "Comma-separated output property names")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum total profiles to fetch (0 = all)")
	cmd.Flags().IntVar(&pageSize, "page-size", 1000, "Profiles per page (max 1000)")

	_ = cmd.MarkFlagRequired("group-key")

	return cmd
}

func runProfilesGroups(cmd *cobra.Command, groupKey, where, properties string, limit, pageSize int) error {
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

	baseParams.Set("data_group_id", groupKey)

	if where != "" {
		baseParams.Set("where", where)
	}
	if properties != "" {
		props := splitCSV(properties)
		baseParams.Set("output_properties", toJSONArray(props))
	}
	baseParams.Set("page_size", strconv.Itoa(pageSize))

	// Auto-paginate (same logic as profiles query).
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
			return fmt.Errorf("querying group profiles (page %d): %w", page, err)
		}

		body, err := readResponseBody(resp.Body, resp.StatusCode)
		if err != nil {
			return err
		}

		var pageResp engageResponse
		if err := json.Unmarshal(body, &pageResp); err != nil {
			return fmt.Errorf("parsing group profiles response: %w", err)
		}

		if pageResp.Status != "ok" && pageResp.Status != "" {
			return fmt.Errorf("engage API returned status %q", pageResp.Status)
		}

		allResults = append(allResults, pageResp.Results...)
		sessionID = pageResp.SessionID
		if totalFromAPI < 0 {
			totalFromAPI = pageResp.Total
		}

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

	// Reuse the profiles table renderer.
	return renderProfilesTable(allResults, properties)
}
