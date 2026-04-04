# hmm

<p>
  <img src="assets/logo.svg" width="540" alt="hmm — Let me look at that data...">
</p>

A terminal UI for viewing CSV, Parquet, and Excel files.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Prerequisites

- [Ghostty](https://ghostty.org) terminal emulator

## Install

### Homebrew

```bash
brew install suntianxun/tap/hmm
```

### pip / uv

```bash
uv pip install hmm-viewer              # also installs the hmm CLI
uv pip install 'hmm-viewer[pandas]'    # with pandas support
uv pip install 'hmm-viewer[polars]'    # with polars support
```

The Go binary is automatically downloaded from GitHub releases on first use — no Go toolchain needed.

### Go

```bash
go install github.com/suntianxun/hmm@latest
```

## Usage

```bash
hmm data.csv
hmm data.parquet
hmm data.xlsx
```

Each invocation opens the data in a **new [Ghostty](https://ghostty.org) terminal window**. The original terminal is immediately available — useful when calling `hmm` from a debugger or script.

Excel files with multiple sheets are displayed with a **tab bar** — press `Tab` / `Shift+Tab` to switch between sheets.

### Keybindings

| Key | Action |
|---|---|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `h` / `←` | Move left |
| `l` / `→` | Move right |
| `g` / `Home` | Go to first row |
| `G` / `End` | Go to last row |
| `PgUp` / `PgDn` | Scroll by page |
| `f` | Open filter for current column |
| `F` | Clear filter on current column |
| `s` | Open sort overlay |
| `S` | Clear sort |
| `y` | Copy (then `y`=cell, `d`=distinct values, `r`=row) |
| `e` | Export filtered data to file |
| `Tab` | Next sheet (Excel) |
| `Shift+Tab` | Previous sheet (Excel) |
| `q` / `Esc` | Quit |

### Sorting

Press `s` to open the sort overlay. Use fuzzy search to find columns, then select them in priority order with `Space` or `Enter`. Each selected column gets a numbered badge showing its sort priority. Press `a` to sort ascending or `d` to sort descending. Press `S` to clear the sort.

### Exporting

Press `e` to open the export overlay. Type a file path ending in `.csv` or `.parquet` and press `Enter`. The currently filtered/sorted rows will be exported. An error is shown if the file extension is not supported. Press `Esc` to cancel.

### Subcommands

```bash
hmm readme    # Show the Python helper README
```

## Python API

The `hmm-viewer` package also provides a Python API for viewing pandas/polars DataFrames directly:

```python
from hmm_viewer import hmm

hmm(df)  # works with pandas and polars DataFrames

# Dict of DataFrames → each key becomes a sheet tab
hmm(pd.read_excel("file.xlsx", sheet_name=None))
```

Useful from a debugger (ipdb, pdb) or REPL — the Ghostty window opens independently, so your debug session continues unblocked. See [python/README.md](python/README.md) for details.

## Supported formats

- **CSV** (`.csv`)
- **Parquet** (`.parquet`, `.pq`)
- **Excel** (`.xlsx`, `.xls`) — each sheet is displayed as a tab
