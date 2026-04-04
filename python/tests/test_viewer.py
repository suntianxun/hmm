import os
import tempfile
from unittest import mock

import pytest

from hmm_viewer.viewer import (
    _cached_binary_path,
    _find_or_install_binary,
    _to_pandas,
    _write_excel,
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


class TestToPandas:
    def test_pandas_passthrough(self):
        pd = pytest.importorskip("pandas")
        df = pd.DataFrame({"a": [1]})
        result = _to_pandas(df)
        assert result is df

    def test_unsupported_type(self):
        with pytest.raises(TypeError, match="Unsupported type"):
            _to_pandas({"not": "a dataframe"})


class TestWriteExcel:
    def test_pandas_dict(self):
        pd = pytest.importorskip("pandas")
        pytest.importorskip("openpyxl")
        sheets = {
            "Sheet1": pd.DataFrame({"a": [1, 2], "b": [3, 4]}),
            "Sheet2": pd.DataFrame({"x": [10, 20]}),
        }

        with tempfile.NamedTemporaryFile(suffix=".xlsx", delete=False) as f:
            path = f.name
        try:
            _write_excel(sheets, path)
            assert os.path.getsize(path) > 0
            # Verify sheet names round-trip
            result = pd.read_excel(path, sheet_name=None)
            assert set(result.keys()) == {"Sheet1", "Sheet2"}
        finally:
            os.unlink(path)

    def test_data_round_trips(self):
        """Verify cell values survive the write/read cycle."""
        pd = pytest.importorskip("pandas")
        pytest.importorskip("openpyxl")
        original = pd.DataFrame({"name": ["Alice", "Bob"], "score": [95, 88]})

        with tempfile.NamedTemporaryFile(suffix=".xlsx", delete=False) as f:
            path = f.name
        try:
            _write_excel({"Data": original}, path)
            loaded = pd.read_excel(path, sheet_name="Data")
            pd.testing.assert_frame_equal(loaded, original)
        finally:
            os.unlink(path)

    def test_empty_dict_raises(self):
        with tempfile.NamedTemporaryFile(suffix=".xlsx", delete=False) as f:
            path = f.name
        try:
            with pytest.raises(ValueError, match="Empty dict"):
                _write_excel({}, path)
        finally:
            os.unlink(path)


class TestHmmDict:
    def test_hmm_dict_calls_binary_with_wait(self):
        """Verify hmm() launches binary with --wait, writes xlsx, then waits."""
        pd = pytest.importorskip("pandas")
        sheets = {"A": pd.DataFrame({"x": [1]})}

        mock_proc = mock.MagicMock()
        mock_proc.wait.return_value = 0

        def fake_write(data, path):
            # Simulate _write_excel by creating the staging file
            with open(path, "w") as f:
                f.write("fake")

        with mock.patch("hmm_viewer.viewer._find_or_install_binary", return_value="/usr/bin/true"):
            with mock.patch("hmm_viewer.viewer._write_excel", side_effect=fake_write):
                with mock.patch("subprocess.Popen", return_value=mock_proc) as mock_popen:
                    hmm(sheets)

        mock_popen.assert_called_once()
        args = mock_popen.call_args[0][0]
        assert args[0] == "/usr/bin/true"
        assert args[1] == "--wait"
        assert args[2].endswith(".xlsx")
        mock_proc.wait.assert_called_once()

    def test_hmm_dict_cleans_up_on_write_error(self):
        """Verify staging and target are cleaned up if _write_excel fails."""
        pd = pytest.importorskip("pandas")
        sheets = {"A": pd.DataFrame({"x": [1]})}

        mock_proc = mock.MagicMock()

        with mock.patch("hmm_viewer.viewer._find_or_install_binary", return_value="/usr/bin/true"):
            with mock.patch("hmm_viewer.viewer._write_excel", side_effect=RuntimeError("write failed")):
                with mock.patch("subprocess.Popen", return_value=mock_proc):
                    with pytest.raises(RuntimeError, match="write failed"):
                        hmm(sheets)

        mock_proc.terminate.assert_called_once()


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
