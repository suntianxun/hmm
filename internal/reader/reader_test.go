package reader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCSV(t *testing.T) {
	dir := t.TempDir()

	t.Run("basic", func(t *testing.T) {
		path := filepath.Join(dir, "basic.csv")
		os.WriteFile(path, []byte("name,age\nalice,30\nbob,25\n"), 0644)

		data, err := ReadCSV(path)
		if err != nil {
			t.Fatalf("ReadCSV: %v", err)
		}
		if len(data.Columns) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(data.Columns))
		}
		if data.Columns[0] != "name" || data.Columns[1] != "age" {
			t.Fatalf("unexpected columns: %v", data.Columns)
		}
		if len(data.Rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(data.Rows))
		}
		if data.Rows[0][0] != "alice" || data.Rows[0][1] != "30" {
			t.Fatalf("unexpected row 0: %v", data.Rows[0])
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(dir, "empty.csv")
		os.WriteFile(path, []byte(""), 0644)

		data, err := ReadCSV(path)
		if err != nil {
			t.Fatalf("ReadCSV: %v", err)
		}
		if len(data.Columns) != 0 {
			t.Fatalf("expected 0 columns, got %d", len(data.Columns))
		}
	})

	t.Run("header only", func(t *testing.T) {
		path := filepath.Join(dir, "header.csv")
		os.WriteFile(path, []byte("a,b,c\n"), 0644)

		data, err := ReadCSV(path)
		if err != nil {
			t.Fatalf("ReadCSV: %v", err)
		}
		if len(data.Columns) != 3 {
			t.Fatalf("expected 3 columns, got %d", len(data.Columns))
		}
		if len(data.Rows) != 0 {
			t.Fatalf("expected 0 rows, got %d", len(data.Rows))
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := ReadCSV(filepath.Join(dir, "nope.csv"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("ragged rows", func(t *testing.T) {
		path := filepath.Join(dir, "ragged.csv")
		os.WriteFile(path, []byte("a,b\n1\n1,2,3\n"), 0644)

		data, err := ReadCSV(path)
		if err != nil {
			t.Fatalf("ReadCSV: %v", err)
		}
		if len(data.Rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(data.Rows))
		}
	})
}

func TestWriteCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")

	data := &TableData{
		Columns: []string{"x", "y"},
		Rows:    [][]string{{"1", "2"}, {"3", "4"}},
	}

	if err := WriteCSV(path, data); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	got, err := ReadCSV(path)
	if err != nil {
		t.Fatalf("ReadCSV roundtrip: %v", err)
	}
	if len(got.Columns) != 2 || got.Columns[0] != "x" {
		t.Fatalf("unexpected columns: %v", got.Columns)
	}
	if len(got.Rows) != 2 || got.Rows[1][1] != "4" {
		t.Fatalf("unexpected rows: %v", got.Rows)
	}
}

func TestParquetRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.parquet")

	data := &TableData{
		Columns: []string{"name", "value"},
		Rows:    [][]string{{"alice", "100"}, {"bob", "200"}},
	}

	if err := WriteParquet(path, data); err != nil {
		t.Fatalf("WriteParquet: %v", err)
	}

	got, err := ReadParquet(path)
	if err != nil {
		t.Fatalf("ReadParquet: %v", err)
	}
	if len(got.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(got.Columns))
	}
	if len(got.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got.Rows))
	}
	if got.Rows[0][0] != "alice" || got.Rows[0][1] != "100" {
		t.Fatalf("unexpected row 0: %v", got.Rows[0])
	}
}

func TestWriteParquetNoColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.parquet")

	err := WriteParquet(path, &TableData{})
	if err == nil {
		t.Fatal("expected error for empty columns")
	}
}

func TestReadParquetMissingFile(t *testing.T) {
	_, err := ReadParquet("/nonexistent/file.parquet")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValueToString(t *testing.T) {
	// Test the null case via the public API — write a parquet with data and read back
	// The function is tested indirectly through roundtrip tests above.
	// Here we just ensure it doesn't panic on various types.
	dir := t.TempDir()
	path := filepath.Join(dir, "types.parquet")

	data := &TableData{
		Columns: []string{"str", "num"},
		Rows:    [][]string{{"hello", "42"}, {"", "0"}},
	}

	if err := WriteParquet(path, data); err != nil {
		t.Fatalf("WriteParquet: %v", err)
	}

	got, err := ReadParquet(path)
	if err != nil {
		t.Fatalf("ReadParquet: %v", err)
	}
	if got.Rows[0][0] != "hello" {
		t.Fatalf("expected 'hello', got %q", got.Rows[0][0])
	}
}
