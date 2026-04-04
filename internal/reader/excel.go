package reader

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func ReadExcel(path string) ([]SheetData, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open excel: %w", err)
	}
	defer f.Close()

	sheetNames := f.GetSheetList()
	if len(sheetNames) == 0 {
		return nil, fmt.Errorf("excel file has no sheets")
	}

	sheets := make([]SheetData, 0, len(sheetNames))
	for _, name := range sheetNames {
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, fmt.Errorf("read sheet %q: %w", name, err)
		}
		if len(rows) == 0 {
			sheets = append(sheets, SheetData{
				Name: name,
				Data: &TableData{},
			})
			continue
		}

		columns := rows[0]
		dataRows := make([][]string, 0, len(rows)-1)
		numCols := len(columns)
		for _, row := range rows[1:] {
			// Pad short rows to match column count.
			padded := make([]string, numCols)
			copy(padded, row)
			dataRows = append(dataRows, padded)
		}

		sheets = append(sheets, SheetData{
			Name: name,
			Data: &TableData{
				Columns: columns,
				Rows:    dataRows,
			},
		})
	}

	return sheets, nil
}
