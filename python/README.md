# hmm-viewer

Python package for [hmm](https://github.com/suntianxun/hmm) — a terminal UI for viewing CSV, Parquet, and Excel files.

The Go binary is automatically downloaded from GitHub releases on first use. No need to install Go.

## Installation

```bash
uv pip install hmm-viewer              # base install
uv pip install 'hmm-viewer[pandas]'    # with pandas support
uv pip install 'hmm-viewer[polars]'    # with polars support
```

## Usage

### CLI

After installing, the `hmm` command is available:

```bash
hmm data.csv
hmm data.parquet
hmm data.xlsx
```

### Python API

```python
from hmm_viewer import hmm

# Works with pandas DataFrames
import pandas as pd
df = pd.read_csv("data.csv")
hmm(df)

# Works with polars DataFrames
import polars as pl
df = pl.read_parquet("data.parquet")
hmm(df)

# Dict of DataFrames — each key becomes a sheet tab
sheets = pd.read_excel("data.xlsx", sheet_name=None)
hmm(sheets)
```

This is especially useful from a debugger (ipdb, pdb, ipython) to quickly
inspect a DataFrame in the terminal.

## How it works

- **CLI**: The `hmm` command is a thin wrapper that delegates to the Go binary.
- **Python API**: `hmm(df)` writes the DataFrame to a temporary Parquet file (or Excel file for a dict of DataFrames), launches the `hmm` binary, and cleans up the temp file when done. For dicts the TUI spinner shows immediately while the Excel file is being written.
- **Auto-download**: If the Go binary isn't found on `$PATH` or `~/go/bin`, it is automatically downloaded from GitHub releases and cached at `~/.cache/hmm/`.
