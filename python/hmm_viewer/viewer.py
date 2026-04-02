import os
import shutil
import subprocess
import tempfile


def hmm(df):
    """View a pandas or polars DataFrame in the hmm TUI viewer.

    Usage from ipdb/pdb:
        from hmm_viewer import hmm
        hmm(df)
    """
    binary = shutil.which("hmm")
    if binary is None:
        # Check common Go bin locations
        go_bin = os.path.expanduser("~/go/bin/hmm")
        if os.path.isfile(go_bin):
            binary = go_bin
        else:
            raise FileNotFoundError(
                "hmm binary not found. Install it with: "
                "cd <repo> && go install ."
            )

    tmp = tempfile.NamedTemporaryFile(suffix=".parquet", delete=False)
    try:
        _write_parquet(df, tmp.name)
        subprocess.run([binary, tmp.name])
    finally:
        os.unlink(tmp.name)


def _write_parquet(df, path):
    typ = type(df).__module__.split(".")[0]

    if typ == "pandas":
        df.to_parquet(path, index=False)
    elif typ == "polars":
        df.write_parquet(path)
    else:
        raise TypeError(
            f"Unsupported type: {type(df).__module__}.{type(df).__name__}. "
            "Expected a pandas or polars DataFrame."
        )
