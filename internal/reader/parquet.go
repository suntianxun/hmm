package reader

import (
	"fmt"
	"os"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/format"
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

	// Extract column names and logical types from schema
	schema := pf.Schema()
	fields := schema.Fields()
	columns := make([]string, len(fields))
	logicalTypes := make([]*format.LogicalType, len(fields))
	for i, field := range fields {
		columns[i] = field.Name()
		logicalTypes[i] = field.Type().LogicalType()
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
						row[j] = valueToString(v, logicalTypes[j])
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

func valueToString(v parquet.Value, lt *format.LogicalType) string {
	if v.IsNull() {
		return ""
	}
	switch v.Kind() {
	case parquet.Boolean:
		return fmt.Sprintf("%v", v.Boolean())
	case parquet.Int32:
		if lt != nil && lt.Date != nil {
			return time.Unix(int64(v.Int32())*86400, 0).UTC().Format("2006-01-02")
		}
		return fmt.Sprintf("%d", v.Int32())
	case parquet.Int64:
		if lt != nil && lt.Timestamp != nil {
			ts := v.Int64()
			var t time.Time
			switch {
			case lt.Timestamp.Unit.Millis != nil:
				t = time.UnixMilli(ts).UTC()
			case lt.Timestamp.Unit.Nanos != nil:
				t = time.Unix(ts/1e9, ts%1e9).UTC()
			default: // micros
				t = time.UnixMicro(ts).UTC()
			}
			if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
				return t.Format("2006-01-02")
			}
			return t.Format("2006-01-02 15:04:05")
		}
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
