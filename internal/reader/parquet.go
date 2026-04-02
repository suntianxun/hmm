package reader

import (
	"fmt"
	"os"

	"github.com/parquet-go/parquet-go"
)

func ReadParquet(path string) (*TableData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open parquet: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat parquet: %w", err)
	}

	pf, err := parquet.OpenFile(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("parse parquet: %w", err)
	}

	// Extract column names from schema
	schema := pf.Schema()
	fields := schema.Fields()
	columns := make([]string, len(fields))
	for i, field := range fields {
		columns[i] = field.Name()
	}

	// Read all rows
	var rows [][]string
	buf := make([]parquet.Row, 256)

	for _, rg := range pf.RowGroups() {
		reader := rg.Rows()
		for {
			n, err := reader.ReadRows(buf)
			for i := 0; i < n; i++ {
				row := make([]string, len(columns))
				for j, v := range buf[i] {
					if j < len(columns) {
						row[j] = valueToString(v)
					}
				}
				rows = append(rows, row)
			}
			if err != nil {
				break
			}
		}
		reader.Close()
	}

	return &TableData{
		Columns: columns,
		Rows:    rows,
	}, nil
}

func valueToString(v parquet.Value) string {
	if v.IsNull() {
		return ""
	}
	switch v.Kind() {
	case parquet.Boolean:
		return fmt.Sprintf("%v", v.Boolean())
	case parquet.Int32:
		return fmt.Sprintf("%d", v.Int32())
	case parquet.Int64:
		return fmt.Sprintf("%d", v.Int64())
	case parquet.Float:
		return fmt.Sprintf("%g", v.Float())
	case parquet.Double:
		return fmt.Sprintf("%g", v.Double())
	case parquet.ByteArray, parquet.FixedLenByteArray:
		return string(v.ByteArray())
	default:
		return v.String()
	}
}
