package reader

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/parquet-go/parquet-go"
)

func WriteCSV(path string, data *TableData) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write(data.Columns); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, row := range data.Rows {
		if err := w.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	w.Flush()
	return w.Error()
}

func WriteParquet(path string, data *TableData) error {
	if len(data.Columns) == 0 {
		return fmt.Errorf("no columns to write")
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create parquet: %w", err)
	}
	defer f.Close()

	// Build schema: all columns as string since TableData stores strings
	group := make(parquet.Group)
	for _, col := range data.Columns {
		group[col] = parquet.String()
	}
	schema := parquet.NewSchema("table", group)

	w := parquet.NewWriter(f, schema)
	defer w.Close()

	for _, row := range data.Rows {
		prow := make(parquet.Row, len(data.Columns))
		for j := range data.Columns {
			val := ""
			if j < len(row) {
				val = row[j]
			}
			prow[j] = parquet.ValueOf(val).Level(0, 0, j)
		}
		if _, err := w.WriteRows([]parquet.Row{prow}); err != nil {
			return fmt.Errorf("write parquet row: %w", err)
		}
	}

	return nil
}
