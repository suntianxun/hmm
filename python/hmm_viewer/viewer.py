import os
import platform
import shutil
import ssl
import subprocess
import tarfile
import tempfile
import urllib.request
import zipfile

# Keep in sync with Go releases
__version__ = "0.1.3"

_GITHUB_REPO = "suntianxun/hmm"
_CACHE_DIR = os.path.join(os.path.expanduser("~"), ".cache", "hmm")


def hmm(df):
    """View a pandas or polars DataFrame in the hmm TUI viewer.

    Accepts a single DataFrame (pandas or polars) or a dict of DataFrames
    (as returned by ``pandas.read_excel(..., sheet_name=None)`` or
    ``polars.read_excel(..., sheet_name=None)``).  When a dict is passed
    each key becomes a sheet tab in the TUI.

    Usage from ipdb/pdb/ipython:
        from hmm_viewer import hmm
        hmm(df)
        hmm(pd.read_excel("file.xlsx", sheet_name=None))
    """
    binary = _find_or_install_binary()

    if isinstance(df, dict):
        # Reserve a temp path, remove it so --wait knows to poll, then
        # launch hmm immediately so the spinner shows while we write.
        tmp = tempfile.NamedTemporaryFile(suffix=".xlsx", delete=False)
        tmp_path = tmp.name
        tmp.close()
        os.unlink(tmp_path)  # hmm --wait polls for this path to appear

        proc = subprocess.Popen([binary, "--wait", tmp_path])
        try:
            # Write to a staging file, then atomic-rename so hmm never
            # reads a partially-written xlsx.
            staging = tmp_path + ".tmp.xlsx"
            _write_excel(df, staging)
            os.rename(staging, tmp_path)
            proc.wait()
        except BaseException:
            proc.terminate()
            # Clean up staging/target if they exist
            for p in (staging, tmp_path):
                try:
                    os.unlink(p)
                except OSError:
                    pass
            raise
    else:
        tmp = tempfile.NamedTemporaryFile(suffix=".parquet", delete=False)
        try:
            _write_parquet(df, tmp.name)
            subprocess.run([binary, tmp.name])
        finally:
            os.unlink(tmp.name)


def _is_python_script(path):
    """Return True if the file is a Python script (not the Go binary)."""
    try:
        with open(path, "rb") as f:
            header = f.read(64)
            return b"python" in header
    except OSError:
        return False


def _find_or_install_binary():
    """Find the hmm binary, downloading it from GitHub releases if needed."""
    # 1. Check PATH (skip Python CLI wrappers to avoid infinite recursion)
    found = shutil.which("hmm")
    if found and not _is_python_script(found):
        return found

    # 2. Check common Go bin location
    go_bin = os.path.expanduser("~/go/bin/hmm")
    if os.path.isfile(go_bin):
        return go_bin

    # 3. Check our cache
    cached = _cached_binary_path()
    if os.path.isfile(cached) and os.access(cached, os.X_OK):
        return cached

    # 4. Download from GitHub releases
    return _download_binary()


def _cached_binary_path():
    name = "hmm.exe" if platform.system() == "Windows" else "hmm"
    return os.path.join(_CACHE_DIR, __version__, name)


def _load_system_certs(ssl_ctx):
    """Load certificates from macOS Keychains into the SSL context."""
    if platform.system() != "Darwin":
        return
    keychains = [
        "/Library/Keychains/System.keychain",
        "/System/Library/Keychains/SystemRootCertificates.keychain",
    ]
    for keychain in keychains:
        try:
            result = subprocess.run(
                ["security", "find-certificate", "-a", "-p", keychain],
                capture_output=True, text=True, timeout=10,
            )
            if result.returncode == 0 and result.stdout:
                ssl_ctx.load_verify_locations(cadata=result.stdout)
        except Exception:
            pass


def _download_binary():
    system = platform.system().lower()  # linux, darwin, windows
    machine = platform.machine().lower()

    # Map machine to Go arch names
    arch_map = {
        "x86_64": "amd64",
        "amd64": "amd64",
        "aarch64": "arm64",
        "arm64": "arm64",
    }
    arch = arch_map.get(machine)
    if arch is None:
        raise RuntimeError(
            f"Unsupported architecture: {machine}. "
            f"Supported: {', '.join(arch_map.keys())}"
        )

    if system not in ("linux", "darwin", "windows"):
        raise RuntimeError(
            f"Unsupported OS: {system}. Supported: linux, darwin, windows"
        )

    ext = "zip" if system == "windows" else "tar.gz"
    archive_name = f"hmm_{__version__}_{system}_{arch}.{ext}"
    url = (
        f"https://github.com/{_GITHUB_REPO}/releases/download/"
        f"v{__version__}/{archive_name}"
    )

    dest_dir = os.path.join(_CACHE_DIR, __version__)
    os.makedirs(dest_dir, exist_ok=True)

    binary_name = "hmm.exe" if system == "windows" else "hmm"
    dest = os.path.join(dest_dir, binary_name)

    print(f"hmm: downloading binary v{__version__} for {system}/{arch}...")

    ssl_ctx = ssl.create_default_context()
    try:
        import certifi
        ssl_ctx.load_verify_locations(certifi.where())
    except ImportError:
        pass
    # Load system certificates (picks up corporate/internal CAs from macOS Keychain).
    _load_system_certs(ssl_ctx)

    tmp = tempfile.NamedTemporaryFile(suffix=f".{ext}", delete=False)
    try:
        with urllib.request.urlopen(url, context=ssl_ctx) as resp:
            with open(tmp.name, "wb") as f:
                shutil.copyfileobj(resp, f)

        if ext == "zip":
            with zipfile.ZipFile(tmp.name) as zf:
                zf.extract(binary_name, dest_dir)
        else:
            with tarfile.open(tmp.name, "r:gz") as tf:
                member = next(
                    (m for m in tf.getmembers() if m.name.endswith(binary_name)),
                    None,
                )
                if member is None:
                    raise RuntimeError(
                        f"Binary '{binary_name}' not found in archive"
                    )
                member.name = binary_name  # flatten path
                tf.extract(member, dest_dir)
    except (urllib.error.HTTPError, urllib.error.URLError) as e:
        raise RuntimeError(
            f"Failed to download hmm binary from {url}: {e}. "
            f"Check that release v{__version__} exists."
        ) from e
    finally:
        os.unlink(tmp.name)

    os.chmod(dest, 0o755)
    print(f"hmm: installed to {dest}")
    return dest


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


def _to_pandas(df):
    """Convert a DataFrame to pandas, or return it if already pandas."""
    typ = type(df).__module__.split(".")[0]
    if typ == "pandas":
        return df
    if typ == "polars":
        return df.to_pandas()
    raise TypeError(
        f"Unsupported type: {type(df).__module__}.{type(df).__name__}. "
        "Expected a pandas or polars DataFrame."
    )


def _write_excel(sheets, path):
    """Write a dict of DataFrames to an Excel file (one sheet per key)."""
    if not sheets:
        raise ValueError("Empty dict — nothing to display.")

    import pandas as pd

    with pd.ExcelWriter(path, engine="openpyxl") as writer:
        for name, df in sheets.items():
            _to_pandas(df).to_excel(writer, sheet_name=str(name), index=False)
