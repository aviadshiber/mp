package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSchemasCmd())
}

func newSchemasCmd() *cobra.Command {
	schemasCmd := &cobra.Command{
		Use:   "schemas",
		Short: "Manage event and profile schemas",
		Long:  "List and inspect event and profile schemas in your Mixpanel project.",
	}

	schemasCmd.AddCommand(newSchemasListCmd())
	schemasCmd.AddCommand(newSchemasGetCmd())
	return schemasCmd
}

func newSchemasListCmd() *cobra.Command {
	var (
		entityType string
		name       string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schemas",
		Long: `List schemas in the project, optionally filtered by entity type and name.
Without flags, lists all schemas. Use --entity-type to filter by event or profile.`,
		Example: `  # List all schemas
  mp schemas list

  # List only event schemas
  mp schemas list --entity-type event

  # List schemas matching a specific name
  mp schemas list --entity-type event --name "Signup"

  # JSON output
  mp schemas list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchemasList(cmd, entityType, name)
		},
	}

	cmd.Flags().StringVar(&entityType, "entity-type", "", "Entity type: event, profile")
	cmd.Flags().StringVar(&name, "name", "", "Schema name (requires --entity-type)")

	return cmd
}

func runSchemasList(cmd *cobra.Command, entityType, name string) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	pid, err := requireProjectID()
	if err != nil {
		return err
	}

	// Build path based on flags.
	path := fmt.Sprintf("/projects/%s/schemas", pid)
	if entityType != "" {
		path += "/" + entityType
		if name != "" {
			path += "/" + name
		}
	}

	resp, err := c.Get(client.APIFamilyApp, path, nil)
	if err != nil {
		return fmt.Errorf("listing schemas: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing schemas response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderSchemasList(result, false)
}

// renderSchemasList renders schemas as a summary table.
func renderSchemasList(result map[string]any, detailed bool) error {
	s := getIO()

	resultsRaw, ok := result["results"].([]any)
	if !ok || len(resultsRaw) == 0 {
		s.Printf("No schemas found.\n")
		return nil
	}

	if detailed && len(resultsRaw) == 1 {
		return renderSchemaDetailed(resultsRaw[0])
	}

	headers := []string{"ENTITY TYPE", "NAME", "DESCRIPTION", "PROPERTIES"}
	rows := make([][]string, 0, len(resultsRaw))

	for _, r := range resultsRaw {
		schema, ok := r.(map[string]any)
		if !ok {
			continue
		}

		entityType, _ := schema["entityType"].(string)
		name, _ := schema["name"].(string)
		desc, _ := schema["description"].(string)

		propCount := 0
		if schemaJSON, ok := schema["schemaJson"].(map[string]any); ok {
			if props, ok := schemaJSON["properties"].(map[string]any); ok {
				propCount = len(props)
			}
		}

		rows = append(rows, []string{entityType, name, desc, fmt.Sprintf("%d", propCount)})
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}

// renderSchemaDetailed renders a single schema with full property details.
func renderSchemaDetailed(schemaRaw any) error {
	s := getIO()

	schema, ok := schemaRaw.(map[string]any)
	if !ok {
		return output.PrintJSON(s.Out, schemaRaw)
	}

	entityType, _ := schema["entityType"].(string)
	name, _ := schema["name"].(string)
	desc, _ := schema["description"].(string)

	s.Printf("Entity Type: %s\n", entityType)
	s.Printf("Name:        %s\n", name)
	if desc != "" {
		s.Printf("Description: %s\n", desc)
	}
	s.Printf("\n")

	schemaJSON, ok := schema["schemaJson"].(map[string]any)
	if !ok {
		return nil
	}

	props, ok := schemaJSON["properties"].(map[string]any)
	if !ok || len(props) == 0 {
		s.Printf("No properties defined.\n")
		return nil
	}

	// Sort property names.
	propNames := make([]string, 0, len(props))
	for k := range props {
		propNames = append(propNames, k)
	}
	sort.Strings(propNames)

	headers := []string{"PROPERTY", "TYPE", "DESCRIPTION"}
	rows := make([][]string, 0, len(propNames))

	for _, pName := range propNames {
		propDef, ok := props[pName].(map[string]any)
		if !ok {
			rows = append(rows, []string{pName, "", ""})
			continue
		}

		propType, _ := propDef["type"].(string)
		propDesc, _ := propDef["description"].(string)
		rows = append(rows, []string{pName, propType, propDesc})
	}

	output.PrintTable(s.Out, headers, rows, s.IsTerminal())
	return nil
}

func newSchemasGetCmd() *cobra.Command {
	var (
		entityType string
		name       string
	)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get detailed schema for an event or profile",
		Long: `Get the full schema for a specific event or profile type, including all
property definitions.`,
		Example: `  # Get schema for the Signup event
  mp schemas get --entity-type event --name "Signup"

  # Get schema for user profiles
  mp schemas get --entity-type profile --name "default"

  # JSON output
  mp schemas get --entity-type event --name "Signup" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchemasGet(cmd, entityType, name)
		},
	}

	cmd.Flags().StringVar(&entityType, "entity-type", "", "Entity type: event, profile (required)")
	cmd.Flags().StringVar(&name, "name", "", "Schema name (required)")

	_ = cmd.MarkFlagRequired("entity-type")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runSchemasGet(cmd *cobra.Command, entityType, name string) error {
	c, err := newClient()
	if err != nil {
		return err
	}

	pid, err := requireProjectID()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/projects/%s/schemas/%s/%s", pid, entityType, name)
	resp, err := c.Get(client.APIFamilyApp, path, nil)
	if err != nil {
		return fmt.Errorf("getting schema: %w", err)
	}

	body, err := readResponseBody(resp.Body, resp.StatusCode)
	if err != nil {
		return err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing schema response: %w", err)
	}

	handled, err := handleJSONOutput(cmd, result)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return renderSchemasList(result, true)
}
