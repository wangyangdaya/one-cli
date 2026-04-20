package output

import (
	"strings"
)

func Table(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			if l := len(row[i]); l > widths[i] {
				widths[i] = l
			}
		}
	}

	if len(headers) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(renderTableRow(headers, widths))
	b.WriteByte('\n')
	b.WriteString(renderTableSeparator(widths))

	for _, row := range rows {
		b.WriteByte('\n')
		cells := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		b.WriteString(renderTableRow(cells, widths))
	}

	return b.String()
}

func renderTableRow(cells []string, widths []int) string {
	parts := make([]string, len(widths))
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts[i] = padRight(cell, width)
	}

	return strings.Join(parts, " | ")
}

func renderTableSeparator(widths []int) string {
	parts := make([]string, len(widths))
	for i, width := range widths {
		if width < 3 {
			width = 3
		}
		parts[i] = strings.Repeat("-", width)
	}

	return strings.Join(parts, " | ")
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}

	return value + strings.Repeat(" ", width-len(value))
}
