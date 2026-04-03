package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/suntianxun/hmm/internal/reader"
)

type exportModel struct {
	input   textinput.Model
	err     string // error message to display
	data    *reader.TableData
	rows    [][]string // filtered rows to export
	done    bool
	success string
}

func newExportModel(data *reader.TableData, rows [][]string) exportModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/output.csv or .parquet"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	return exportModel{
		input: ti,
		data:  data,
		rows:  rows,
	}
}

func (e *exportModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			path := strings.TrimSpace(e.input.Value())
			if path == "" {
				e.err = "Path cannot be empty"
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".csv" && ext != ".parquet" {
				e.err = "File must end in .csv or .parquet"
				return nil
			}
			e.err = ""

			exportData := &reader.TableData{
				Columns: e.data.Columns,
				Rows:    e.rows,
			}

			var exportErr error
			if ext == ".csv" {
				exportErr = reader.WriteCSV(path, exportData)
			} else {
				exportErr = reader.WriteParquet(path, exportData)
			}

			if exportErr != nil {
				e.err = fmt.Sprintf("Export failed: %v", exportErr)
				return nil
			}

			e.done = true
			e.success = fmt.Sprintf("Exported %d rows to %s", len(e.rows), filepath.Base(path))
			return nil
		}
	}

	var cmd tea.Cmd
	e.input, cmd = e.input.Update(msg)
	// Clear error when user types
	e.err = ""
	return cmd
}

func (e *exportModel) view(maxWidth int) string {
	var b strings.Builder

	title := filterTitleStyle.Render("Export data")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(e.input.View())
	b.WriteString("\n")

	if e.err != "" {
		b.WriteString("\n")
		b.WriteString(exportErrorStyle.Render(e.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(filterHintStyle.Render(
		fmt.Sprintf("Rows: %d | Supported: .csv, .parquet", len(e.rows))))
	b.WriteString("\n")
	b.WriteString(filterHintStyle.Render(
		"Enter: export | Esc: cancel"))

	return filterBorderStyle.MaxWidth(maxWidth).Render(b.String())
}
