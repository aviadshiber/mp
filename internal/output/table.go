package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintTable writes tabular data. When isTTY is true it renders aligned columns
// with a header. When false it outputs tab-separated values for piping.
func PrintTable(w io.Writer, headers []string, rows [][]string, isTTY bool) {
	if !isTTY {
		printTSV(w, headers, rows)
		return
	}

	table := tablewriter.NewTable(w,
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenColumns: tw.Off,
					BetweenRows:    tw.Off,
				},
			},
		})),
	)

	table.Header(toAny(headers)...)
	for _, row := range rows {
		table.Append(toAny(row)...)
	}
	table.Render()
}

// printTSV writes headers and rows as tab-separated values.
func printTSV(w io.Writer, headers []string, rows [][]string) {
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
}

// toAny converts a string slice to an any slice for the tablewriter API.
func toAny(ss []string) []any {
	result := make([]any, len(ss))
	for i, s := range ss {
		result[i] = s
	}
	return result
}
