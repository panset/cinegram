"""``python -m cinegram`` — the same entry point as the ``cinegram`` script.

Running this file directly (``python src/cinegram/__main__.py version``) works
too, which is how the launcher is exercised from a source checkout: the sibling
import below falls back to loading ``__init__.py`` by path when the package is
not on ``sys.path``, and ``package_version()`` then reads the version out of
``pyproject.toml`` instead of installed metadata.
"""

import sys

try:
    from cinegram import main
except ImportError:  # run as a plain file, with no package around it
    import os
    import pathlib

    sys.path.insert(0, os.fspath(pathlib.Path(__file__).resolve().parent.parent))
    from cinegram import main

if __name__ == "__main__":
    sys.exit(main())
