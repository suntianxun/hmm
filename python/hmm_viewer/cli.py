"""CLI wrapper: runs the hmm Go binary, downloading it if needed."""

import subprocess
import sys

from hmm_viewer.viewer import _find_or_install_binary


def main():
    binary = _find_or_install_binary()
    result = subprocess.run([binary] + sys.argv[1:])
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
