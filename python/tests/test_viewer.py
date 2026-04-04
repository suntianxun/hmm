import os
import tempfile
from unittest import mock

import pytest

from hmm_viewer.viewer import (
    _cached_binary_path,
    _find_or_install_binary,
    _write_parquet,
    hmm,
)


class TestCachedBinaryPath:
    def test_returns_path_with_version(self):
        path = _cached_binary_path()
        assert "hmm" in path
        assert ".cache" in path

    def test_no_exe_on_macos(self):
        path = _cached_binary_path()
        assert not path.endswith(".exe")


class TestFindOrInstallBinary:
    def test_finds_on_path(self, tmp_path):
        fake_bin = tmp_path / "hmm"
        fake_bin.write_text("#!/bin/sh\n")
        fake_bin.chmod(0o755)

        with mock.patch("shutil.which", return_value=str(fake_bin)):
            result = _find_or_install_binary()
        assert result == str(fake_bin)

    def test_finds_go_bin(self, tmp_path):
        fake_bin = tmp_path / "hmm"
        fake_bin.write_text("#!/bin/sh\n")

        with mock.patch("shutil.which", return_value=None):
            with mock.patch("os.path.expanduser", return_value=str(fake_bin)):
                with mock.patch("os.path.isfile", return_value=True):
                    result = _find_or_install_binary()
        assert result == str(fake_bin)


class TestWriteParquet:
    def test_pandas_dataframe(self):
        pd = pytest.importorskip("pandas")
        df = pd.DataFrame({"a": [1, 2], "b": [3, 4]})

        with tempfile.NamedTemporaryFile(suffix=".parquet", delete=False) as f:
            path = f.name
        try:
            _write_parquet(df, path)
            assert os.path.getsize(path) > 0
        finally:
            os.unlink(path)

    def test_unsupported_type(self):
        with tempfile.NamedTemporaryFile(suffix=".parquet", delete=False) as f:
            path = f.name
        try:
            with pytest.raises(TypeError, match="Unsupported type"):
                _write_parquet({"not": "a dataframe"}, path)
        finally:
            os.unlink(path)


class TestHmm:
    def test_hmm_calls_binary(self):
        """Verify hmm() calls the binary with the temp file path."""
        pd = pytest.importorskip("pandas")
        df = pd.DataFrame({"x": [1]})

        with mock.patch("hmm_viewer.viewer._find_or_install_binary", return_value="/usr/bin/true"):
            with mock.patch("hmm_viewer.viewer._write_parquet"):
                with mock.patch("subprocess.run") as mock_run:
                    with mock.patch("os.unlink"):
                        hmm(df)

        mock_run.assert_called_once()
        args = mock_run.call_args[0][0]
        assert args[0] == "/usr/bin/true"
        assert args[1].endswith(".parquet")

    def test_hmm_cleans_up_temp_file(self):
        """Verify temp file is cleaned up even if subprocess fails."""
        pd = pytest.importorskip("pandas")
        df = pd.DataFrame({"x": [1]})

        with mock.patch("hmm_viewer.viewer._find_or_install_binary", return_value="/usr/bin/false"):
            with mock.patch("hmm_viewer.viewer._write_parquet"):
                with mock.patch("subprocess.run", side_effect=OSError("boom")):
                    with mock.patch("os.unlink") as mock_unlink:
                        with pytest.raises(OSError):
                            hmm(df)

        mock_unlink.assert_called_once()
