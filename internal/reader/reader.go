package reader

// TableData holds the parsed contents of a data file.
type TableData struct {
	Columns []string
	Rows    [][]string
}
