package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	switch ext {
	case ".csv", ".parquet", ".pq":
	default:
		log.Fatal("Unsupported file type. Supported: .csv, .parquet, .pq")
	}

	// If not already inside the spawned Ghostty window, open one.
	if os.Getenv("HMM_INNER") != "1" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			log.Fatal("Failed to resolve path", "error", err)
		}

		// Copy the file to a stable temp location so the caller can
		// clean up the original immediately (e.g. Python temp files).
		tmpFile, err := os.CreateTemp("", "hmm-*"+ext)
		if err != nil {
			log.Fatal("Failed to create temp file", "error", err)
		}
		tmpPath := tmpFile.Name()

		src, err := os.Open(absPath)
		if err != nil {
			os.Remove(tmpPath)
			log.Fatal("Failed to open source file", "error", err)
		}
		if _, err := io.Copy(tmpFile, src); err != nil {
			src.Close()
			tmpFile.Close()
			os.Remove(tmpPath)
			log.Fatal("Failed to copy file", "error", err)
		}
		src.Close()
		tmpFile.Close()

		self, err := os.Executable()
		if err != nil {
			os.Remove(tmpPath)
			log.Fatal("Failed to find own executable", "error", err)
		}

		// The inner process will delete the temp copy when done.
		shellCmd := fmt.Sprintf("HMM_INNER=1 HMM_CLEANUP=%s exec %s %s",
			shellQuote(tmpPath), shellQuote(self), shellQuote(tmpPath))
		cmd := exec.Command("ghostty", "-e", "/bin/sh", "-c", shellCmd)
		if err := cmd.Start(); err != nil {
			os.Remove(tmpPath)
			log.Fatal("Failed to open Ghostty window", "error", err)
		}
		return
	}

	// Running inside the Ghostty window — show the TUI.
	// Clean up the temp copy when done.
	if cleanup := os.Getenv("HMM_CLEANUP"); cleanup != "" {
		defer os.Remove(cleanup)
	}
	var data *reader.TableData
	var err error

	switch ext {
	case ".csv":
		data, err = reader.ReadCSV(path)
	case ".parquet", ".pq":
		data, err = reader.ReadParquet(path)
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

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
