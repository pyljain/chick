package rendering

import (
	"github.com/fatih/color"
	"github.com/rodaine/table"
)

func FormatAsTable(output []map[string]any) error {

	headers := []interface{}{}

	for k := range output[0] {
		headers = append(headers, k)
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New(headers...)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, record := range output {
		row := []any{}
		for _, h := range headers {
			row = append(row, record[h.(string)])
		}
		tbl.AddRow(row...)

	}

	tbl.Print()

	return nil

}
