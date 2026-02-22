package cmd

import (
	"encoding/json"
	"fmt"
	iolib "io"
	"net/url"
	"os"
	"strings"

	"github.com/aviadshiber/mp/internal/client"
	"github.com/aviadshiber/mp/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newClient creates an authenticated Mixpanel API client from the current
// configuration state (viper config + env vars + flags).
func newClient() (*client.Client, error) {
	sa := viper.GetString("service_account")
	ss := viper.GetString("service_secret")

	// MP_TOKEN env var overrides config: "user:secret".
	if token := os.Getenv("MP_TOKEN"); token != "" {
		parts := strings.SplitN(token, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("MP_TOKEN must be in the format `user:secret`")
		}
		sa, ss = parts[0], parts[1]
	}

	region := viper.GetString("region")
	if region == "" {
		region = client.RegionUS
	}

	projectID := viper.GetString("project_id")

	return client.New(sa, ss, region, projectID, isDebug())
}

// requireProjectID returns the configured project ID or an error telling the
// user how to set it.
func requireProjectID() (string, error) {
	pid := viper.GetString("project_id")
	if pid == "" {
		return "", fmt.Errorf("project ID is required; set via `--project-id`, `MP_PROJECT_ID` env, or `mp config set project_id <id>`")
	}
	return pid, nil
}

// readResponseBody reads the full body of an HTTP response and closes it.
// It returns an error if the status code indicates a failure.
func readResponseBody(resp iolib.ReadCloser, statusCode int) ([]byte, error) {
	defer resp.Close()
	body, err := iolib.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	if statusCode >= 400 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", statusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// handleJSONOutput processes a parsed JSON value through --jq or --template
// filters, or prints it as pretty JSON. It returns true if JSON output was
// handled (i.e., --json was requested), false otherwise.
func handleJSONOutput(cmd *cobra.Command, data any) (bool, error) {
	if !jsonOutputRequested(cmd) {
		return false, nil
	}

	s := getIO()

	jqExpr, _ := cmd.Flags().GetString("jq")
	tmpl, _ := cmd.Flags().GetString("template")

	switch {
	case jqExpr != "":
		return true, output.ApplyJQ(s.Out, data, jqExpr)
	case tmpl != "":
		return true, output.ApplyTemplate(s.Out, data, tmpl)
	default:
		return true, output.PrintJSON(s.Out, data)
	}
}

// toJSONArray encodes a string slice as a JSON array string,
// e.g., ["Signup","Login"].
func toJSONArray(items []string) string {
	b, _ := json.Marshal(items)
	return string(b)
}

// addProjectID sets the project_id query/form parameter if configured.
func addProjectID(params url.Values) error {
	pid, err := requireProjectID()
	if err != nil {
		return err
	}
	params.Set("project_id", pid)
	return nil
}

// splitCSV splits a comma-separated string into trimmed, non-empty parts.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
