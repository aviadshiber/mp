// Package output provides formatters for CLI output: JSON, table, CSV, and JSONL.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/itchyny/gojq"
)

// PrintJSON pretty-prints v as indented JSON to w.
func PrintJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// FilterFields takes a slice of maps and returns a new slice containing only
// the specified fields from each map.
func FilterFields(data []map[string]any, fields []string) []map[string]any {
	if len(fields) == 0 {
		return data
	}
	result := make([]map[string]any, 0, len(data))
	for _, item := range data {
		filtered := make(map[string]any, len(fields))
		for _, f := range fields {
			if val, ok := item[f]; ok {
				filtered[f] = val
			}
		}
		result = append(result, filtered)
	}
	return result
}

// FilterFieldsSingle filters a single map to only the specified fields.
func FilterFieldsSingle(data map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return data
	}
	filtered := make(map[string]any, len(fields))
	for _, f := range fields {
		if val, ok := data[f]; ok {
			filtered[f] = val
		}
	}
	return filtered
}

// ApplyJQ runs a jq expression against the input data and writes results to w.
func ApplyJQ(w io.Writer, data any, expr string) error {
	query, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("parsing jq expression: %w", err)
	}

	iter := query.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return fmt.Errorf("jq evaluation: %w", err)
		}
		if err := PrintJSON(w, v); err != nil {
			return fmt.Errorf("writing jq result: %w", err)
		}
	}
	return nil
}

// ApplyTemplate renders data through a Go text/template and writes to w.
func ApplyTemplate(w io.Writer, data any, tmpl string) error {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	_, err = buf.WriteTo(w)
	return err
}
