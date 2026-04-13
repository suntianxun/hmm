package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	// Parse --wait flag: hmm [--wait] <path>
	args := os.Args[1:]
	waitMode := false
	if len(args) >= 1 && args[0] == "--wait" {
		waitMode = true
		args = args[1:]
	}

	if len(args) != 1 {
		log.Fatal("Usage: hmm [--wait] <file.csv|file.parquet|file.xlsx> | hmm readme | hmm --version")
	}

	path := args[0]

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
	case ".csv", ".parquet", ".pq", ".xlsx", ".xls":
	default:
		log.Fatal("Unsupported file type. Supported: .csv, .parquet, .pq, .xlsx, .xls")
	}

	// If not already inside the spawned Ghostty window, open one.
	if os.Getenv("HMM_INNER") != "1" {
		self, err := os.Executable()
		if err != nil {
			log.Fatal("Failed to find own executable", "error", err)
		}

		if waitMode {
			// File doesn't exist yet — pass path directly to inner
			// process which will poll for it. No file copy needed.
			absPath, err := filepath.Abs(path)
			if err != nil {
				log.Fatal("Failed to resolve path", "error", err)
			}
			shellCmd := fmt.Sprintf("HMM_INNER=1 HMM_CLEANUP=%s exec %s --wait %s",
				shellQuote(absPath), shellQuote(self), shellQuote(absPath))
			if err := launchGhostty(shellCmd); err != nil {
				log.Fatal("Failed to open Ghostty window", "error", err)
			}
			return
		}

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

		// The inner process will delete the temp copy when done.
		shellCmd := fmt.Sprintf("HMM_INNER=1 HMM_CLEANUP=%s exec %s %s",
			shellQuote(tmpPath), shellQuote(self), shellQuote(tmpPath))
		if err := launchGhostty(shellCmd); err != nil {
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

	filename := filepath.Base(path)

	var loadFn tui.LoadFunc
	if waitMode {
		loadFn = makeWaitLoadFunc(path, ext)
	} else {
		loadFn = makeLoadFunc(path, ext)
	}
	m := tui.NewLoading(filename, loadFn)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal("TUI error", "error", err)
	}
}

func makeLoadFunc(path, ext string) tui.LoadFunc {
	return func() ([]reader.SheetData, error) {
		switch ext {
		case ".xlsx", ".xls":
			return reader.ReadExcel(path)
		case ".csv":
			data, err := reader.ReadCSV(path)
			if err != nil {
				return nil, err
			}
			return []reader.SheetData{{Name: "Sheet1", Data: data}}, nil
		case ".parquet", ".pq":
			data, err := reader.ReadParquet(path)
			if err != nil {
				return nil, err
			}
			return []reader.SheetData{{Name: "Sheet1", Data: data}}, nil
		default:
			return nil, fmt.Errorf("unsupported extension: %s", ext)
		}
	}
}

// makeWaitLoadFunc returns a LoadFunc that polls until the file appears,
// then reads it. Used with --wait so the TUI can show a spinner while
// an external process (e.g. Python) writes the file.
func makeWaitLoadFunc(path, ext string) tui.LoadFunc {
	return func() ([]reader.SheetData, error) {
		// Poll until the file exists.
		for {
			if _, err := os.Stat(path); err == nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		return makeLoadFunc(path, ext)()
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// launchGhostty opens a new Ghostty terminal window running shellCmd.
// On macOS it tries the +new-window IPC action first to reuse the running
// Ghostty instance (avoids a second dock icon), falling back to launching
// a new process.
func launchGhostty(shellCmd string) error {
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("ghostty", "+new-window",
			"--command=/bin/sh -c "+shellQuote(shellCmd))
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	cmd := exec.Command("ghostty", "-e", "/bin/sh", "-c", shellCmd)
	return cmd.Start()
}
