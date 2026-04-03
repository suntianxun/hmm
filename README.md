# hmm

A terminal UI for viewing CSV and Parquet files.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Install

### Homebrew

```bash
brew install suntianxun/tap/hmm
```

### Go

```bash
go install github.com/suntianxun/hmm@latest
```

## Usage

```bash
hmm data.csv
hmm data.parquet
```

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
| `q` / `Esc` | Quit |

### Sorting

Press `s` to open the sort overlay. Use fuzzy search to find columns, then select them in priority order with `Space` or `Enter`. Each selected column gets a numbered badge showing its sort priority. Press `a` to sort ascending or `d` to sort descending. Press `S` to clear the sort.

### Exporting

Press `e` to open the export overlay. Type a file path ending in `.csv` or `.parquet` and press `Enter`. The currently filtered/sorted rows will be exported. An error is shown if the file extension is not supported. Press `Esc` to cancel.

### Subcommands

```bash
hmm readme    # Show the Python helper README
```

## Python helper

A Python package is included for viewing pandas/polars DataFrames in `hmm` — useful from a debugger or REPL.

```bash
uv pip install -e './python[pandas]'
```

```python
from hmm_viewer import hmm

hmm(df)
```

See `hmm readme` or [python/README.md](python/README.md) for details.

## Supported formats

- **CSV** (`.csv`)
- **Parquet** (`.parquet`, `.pq`)
