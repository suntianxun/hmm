package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/log"
	"github.com/suntianxun/hmm/internal/reader"
	"github.com/suntianxun/hmm/internal/tui"
)

// version is set via ldflags at build time.
var version = "dev"

//go:embed python/README.md
var pythonReadme string

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: hmm <file.csv|file.parquet> | hmm readme | hmm --version")
	}

	path := os.Args[1]

	if path == "--version" || path == "-v" {
		fmt.Println("hmm " + version)
		return
	}

	if path == "readme" {
		out, err := glamour.Render(pythonReadme, "dark")
		if err != nil {
			log.Fatal("Failed to render readme", "error", err)
		}
		fmt.Print(out)
		return
	}
	ext := strings.ToLower(filepath.Ext(path))

	var data *reader.TableData
	var err error

	switch ext {
	case ".csv":
		data, err = reader.ReadCSV(path)
	case ".parquet", ".pq":
		data, err = reader.ReadParquet(path)
	default:
		log.Fatal("Unsupported file type. Supported: .csv, .parquet, .pq")
	}

	if err != nil {
		log.Fatal("Failed to read file", "error", err)
	}

	if len(data.Columns) == 0 {
		log.Fatal("File has no columns")
	}

	filename := filepath.Base(path)
	m := tui.New(data, filename)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal("TUI error", "error", err)
	}
}
