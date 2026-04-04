package reader

// TableData holds the parsed contents of a data file.
type TableData struct {
	Columns []string
	Rows    [][]string
}

// SheetData pairs a sheet name with its table data.
type SheetData struct {
	Name string
	Data *TableData
}
