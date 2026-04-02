# hmm-viewer

Python helper to view DataFrames in the [hmm](https://github.com/suntianxun/hmm) TUI viewer.

## Installation

```bash
uv pip install -e ./python                 # base install
uv pip install -e './python[pandas]'       # with pandas support
uv pip install -e './python[polars]'       # with polars support
```

## Usage

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
```

This is especially useful from a debugger (ipdb, pdb, ipython) to quickly
inspect a DataFrame in the terminal.

## How it works

`hmm()` writes the DataFrame to a temporary Parquet file, launches the `hmm`
TUI binary to display it, and cleans up the temp file when done.

## Requirements

The `hmm` Go binary must be on your `$PATH` or installed at `~/go/bin/hmm`.
Install it from the repo root:

```bash
go install .
```
