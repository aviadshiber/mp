package output

import (
	"encoding/json"
	"io"
)

// PrintJSONL writes each element of data as a single-line JSON object (JSON Lines format).
// This is suitable for streaming large result sets where each record is independent.
func PrintJSONL(w io.Writer, data []map[string]any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, item := range data {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}

// JSONLWriter streams individual JSON objects one per line.
type JSONLWriter struct {
	enc *json.Encoder
}

// NewJSONLWriter creates a JSONLWriter that writes to w.
func NewJSONLWriter(w io.Writer) *JSONLWriter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &JSONLWriter{enc: enc}
}

// Write encodes a single value as one JSON line.
func (jw *JSONLWriter) Write(v any) error {
	return jw.enc.Encode(v)
}
