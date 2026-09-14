package rendering

type FormatFunc func(output []map[string]any) error

func Format(outputFormat string) FormatFunc {
	switch outputFormat {
	case "table":
		return FormatAsTable
	case "JSON":
		return FormatAsJSON
	default:
		return FormatAsTable
	}
}
