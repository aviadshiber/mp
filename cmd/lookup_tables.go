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
	rootCmd.AddCommand(newLookupTablesCmd())
}

func newLookupTablesCmd() *cobra.Command {
	lookupTablesCmd := &cobra.Command{
		Use:     "lookup-tables",
		Aliases: []string{"lt"},
		Short:   "Manage lookup tables",
		Long:    "List and inspect lookup tables in your Mixpanel project.",
	}

	lookupTablesCmd.AddCommand(newLookupTablesListCmd())
	return lookupTablesCmd
}

func newLookupTablesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all lookup tables",
		Long:  "List all lookup tables in the project with their metadata.",
		Example: `  # List all lookup tables
  mp lookup-tables list

  # JSON output
  mp lookup-tables list --json

  # Filter with jq
  mp lookup-tables list --json --jq '.[].name'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLookupTablesList(cmd)
		},
	}
	return cmd
}

func runLookupTablesList(cmd *cobra.Command) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	params := url.Values{}
	if err := addProjectID(params); err != nil {
		return err
	}

	resp, err := c.Get(client.APIFamilyIngestion, "/lookup-tables", params)
	if err != nil {
		return fmt.Errorf("listing lookup tables: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing lookup tables response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderLookupTables(result)
}

func renderLookupTables(result any) error {
	s := getIO()

	// The response may be an array or an object with a results field.
	var tables []map[string]any

	switch v := result.(type) {
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				tables = append(tables, m)
			}
		}
	case map[string]any:
		// Check for "results" key.
		if res, ok := v["results"].([]any); ok {
			for _, item := range res {
				if m, ok := item.(map[string]any); ok {
					tables = append(tables, m)
				}
			}
		} else {
			// Treat each key-value as a table entry.
			for k, val := range v {
				if m, ok := val.(map[string]any); ok {
					m["_key"] = k
					tables = append(tables, m)
				}
			}
		}
	}

	if len(tables) == 0 {
		s.Printf("No lookup tables found.\n")
		return nil
	}

	// Sort by name for consistent output.
	sort.Slice(tables, func(i, j int) bool {
		nameI := tableName(tables[i])
		nameJ := tableName(tables[j])
		return nameI < nameJ
	})

	headers := []string{"NAME", "ID", "ROWS", "COLUMNS"}
	rows := make([][]string, 0, len(tables))

	for _, t := range tables {
		name := tableName(t)
		id := ""
		if v, ok := t["id"].(string); ok {
			id = v
		} else if v, ok := t["id"].(float64); ok {
			id = fmt.Sprintf("%.0f", v)
		}

		rowCount := ""
		if v, ok := t["rowCount"].(float64); ok {
			rowCount = fmt.Sprintf("%.0f", v)
		} else if v, ok := t["row_count"].(float64); ok {
			rowCount = fmt.Sprintf("%.0f", v)
		}

		colCount := ""
		if v, ok := t["columnCount"].(float64); ok {
			colCount = fmt.Sprintf("%.0f", v)
		} else if cols, ok := t["columns"].([]any); ok {
			colCount = fmt.Sprintf("%d", len(cols))
		}

		rows = append(rows, []string{name, id, rowCount, colCount})
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}

func tableName(t map[string]any) string {
	if name, ok := t["name"].(string); ok {
		return name
	}
	if key, ok := t["_key"].(string); ok {
		return key
	}
	return ""
}
