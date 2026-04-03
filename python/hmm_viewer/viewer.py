import os
import platform
import shutil
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

    Usage from ipdb/pdb:
        from hmm_viewer import hmm
        hmm(df)
    """
    binary = _find_or_install_binary()

    tmp = tempfile.NamedTemporaryFile(suffix=".parquet", delete=False)
    try:
        _write_parquet(df, tmp.name)
        subprocess.run([binary, tmp.name])
    finally:
        os.unlink(tmp.name)


def _find_or_install_binary():
    """Find the hmm binary, downloading it from GitHub releases if needed."""
    # 1. Check PATH
    found = shutil.which("hmm")
    if found:
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

    tmp = tempfile.NamedTemporaryFile(suffix=f".{ext}", delete=False)
    try:
        urllib.request.urlretrieve(url, tmp.name)

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
    except urllib.error.HTTPError as e:
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
