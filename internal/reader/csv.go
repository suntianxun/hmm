package reader

import (
	"encoding/csv"
	"fmt"
	"os"
)

func ReadCSV(path string) (*TableData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // allow ragged rows

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	if len(records) == 0 {
		return &TableData{}, nil
	}

	return &TableData{
		Columns: records[0],
		Rows:    records[1:],
	}, nil
}
