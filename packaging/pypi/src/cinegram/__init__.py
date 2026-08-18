"""``uvx cinegram`` — fetch the released binary for this platform, then get out
of the way.

cinegram is a Go program shipped as one static binary per platform. This
package carries no binary of its own and runs no install hook: the download
happens here, at the first use, so a failure lands in front of the person who
asked for it rather than in the middle of an unattended install.

The version is the distribution's own — ``importlib.metadata.version`` — so
``uvx cinegram==0.3.0`` fetches exactly the v0.3.0 binary and nothing else ever
decides which release to use. That is also why ``cinegram upgrade`` refuses to
run under this launcher (see ``cmd/cinegram/upgrade.go``): the cache entry is
version-pinned, and overwriting it in place would make the pin lie.

The asset names are the release contract, shared with ``cinegram upgrade`` and
with the VS Code extension: ``cinegram-<os>-<arch>``, in VS Code's platform
vocabulary. See RELEASING.md.
"""

from __future__ import annotations

import hashlib
import os
import re
import sys
import urllib.request
from pathlib import Path

__all__ = ["main"]

# Where releases live. CINEGRAM_DOWNLOAD_BASE points the launcher at a mirror —
# or, in this repository's own tests, at a local server — and is documented in
# README.md. The checksum check below is not relaxed for it: a mirror has to
# serve the same bytes as the release, or nothing is installed.
DEFAULT_BASE = "https://github.com/panset/cinegram/releases/download"

# The platform table, in the release's vocabulary — VS Code's, by way of
# editors/vscode/src/binary.js. TARGETS is the part that matters: not every
# os/arch pair has a build (there is no Windows-on-ARM one), and a launcher
# that cheerfully computed one would 404 on the first run. The release workflow
# gates on this exact list appearing in the release's SHA256SUMS, and
# packaging/npm/bin/cinegram.js carries the same three constants.
OSES = {"darwin": "darwin", "linux": "linux", "win32": "win32"}
ARCHES = {
    "x86_64": "x64",
    "amd64": "x64",
    "arm64": "arm64",
    "aarch64": "arm64",
}
TARGETS = ("darwin-arm64", "darwin-x64", "linux-arm64", "linux-x64", "win32-x64")


class LauncherError(Exception):
    """A failure worth printing without a traceback."""


def package_version() -> str:
    """The version of this distribution, which is the release to fetch.

    Installed, that is what the metadata says. Run straight out of the source
    tree — ``python src/cinegram/__main__.py`` while working on the launcher —
    there is no metadata, so the manifest beside it answers instead.
    """
    try:
        from importlib.metadata import PackageNotFoundError, version

        return version("cinegram")
    except Exception:  # PackageNotFoundError, or no metadata at all
        pyproject = Path(__file__).resolve().parents[2] / "pyproject.toml"
        try:
            text = pyproject.read_text(encoding="utf-8")
        except OSError as err:
            raise LauncherError(
                "cannot tell which cinegram version to fetch: the package is not "
                f"installed and {pyproject} is unreadable ({err})"
            ) from err
        match = re.search(r'(?m)^version = "(\d+\.\d+\.\d+)"$', text)
        if not match:
            raise LauncherError(f"{pyproject} does not name a version")
        return match.group(1)


def platform_key() -> "tuple[str, str]":
    """This interpreter's platform and machine, as the tables above spell them."""
    import platform

    return sys.platform, platform.machine().lower()


def asset_name(plat: str, machine: str) -> str:
    o = OSES.get("win32" if plat.startswith("win") else plat)
    a = ARCHES.get(machine)
    target = f"{o}-{a}"
    if target not in TARGETS:
        raise LauncherError(
            f"no cinegram release is published for {plat}-{machine}; "
            f"supported platforms are {', '.join(TARGETS)}. "
            "Build from source (https://github.com/panset/cinegram), or point "
            "$CINEGRAM_BIN at a binary you built."
        )
    return f"cinegram-{target}" + (".exe" if o == "win32" else "")


def cache_root() -> Path:
    """One directory per version, so several pinned versions coexist and none
    of them is ever rewritten."""
    configured = os.environ.get("CINEGRAM_CACHE_DIR")
    if configured:
        return Path(configured)
    if sys.platform.startswith("win"):
        local = os.environ.get("LOCALAPPDATA") or str(Path.home() / "AppData" / "Local")
        return Path(local) / "cinegram" / "bin"
    return Path.home() / ".cinegram" / "bin"


def _get(url: str) -> bytes:
    request = urllib.request.Request(
        url, headers={"User-Agent": f"cinegram-pypi/{package_version()}", "Accept": "*/*"}
    )
    try:
        with urllib.request.urlopen(request) as response:  # follows redirects
            return response.read()
    except Exception as err:
        raise LauncherError(f"downloading {url}: {err}") from err


def checksum_for(sums: bytes, asset: str, version: str) -> str:
    """One entry out of sha256sum output ("<hex>  <name>")."""
    for line in sums.decode("utf-8", "replace").splitlines():
        fields = line.split()
        if len(fields) == 2 and fields[1] == asset:
            return fields[0]
    raise LauncherError(f"the v{version} release's SHA256SUMS has no entry for {asset}")


def download(dest: Path, asset: str, version: str) -> None:
    """Fetch SHA256SUMS first and the asset second, verify the bytes before
    they reach their final name, and move the file into place with a rename so
    a killed download can never leave a half-written binary behind."""
    base = os.environ.get("CINEGRAM_DOWNLOAD_BASE", DEFAULT_BASE).rstrip("/")
    directory = f"{base}/v{version}"

    sys.stderr.write(f"cinegram: downloading {asset} v{version}…\n")
    want = checksum_for(_get(f"{directory}/SHA256SUMS"), asset, version)
    body = _get(f"{directory}/{asset}")
    got = hashlib.sha256(body).hexdigest()
    if got != want:
        raise LauncherError(
            f"{asset} does not match the v{version} release's SHA256SUMS "
            f"(got {got}, want {want}); not installing it"
        )

    dest.parent.mkdir(parents=True, exist_ok=True)
    tmp = dest.with_name(f"{dest.name}.{os.getpid()}.tmp")
    try:
        tmp.write_bytes(body)
        if not sys.platform.startswith("win"):
            tmp.chmod(0o755)
        os.replace(tmp, dest)
    except BaseException:
        tmp.unlink(missing_ok=True)
        raise


def run(binary: str, args: list) -> int:
    """Hand the process over. On POSIX that is literal — execv leaves no
    launcher behind to confuse a Ctrl-C or a `ps`. Windows has no execv worth
    the name, so the child is waited on and its exit code becomes ours."""
    os.environ["CINEGRAM_MANAGED_BY"] = "pypi"
    if sys.platform.startswith("win"):
        import subprocess

        return subprocess.run([binary, *args]).returncode
    try:
        os.execv(binary, [binary, *args])
    except OSError as err:
        raise LauncherError(f"running {binary}: {err}") from err
    return 0  # unreachable: execv does not return


def main(argv=None) -> int:
    args = list(sys.argv[1:] if argv is None else argv)
    try:
        # An explicit binary wins over everything: no download, no cache, no
        # version lookup. This is the seam the repository's own tests and the
        # skill use.
        override = os.environ.get("CINEGRAM_BIN")
        if override:
            return run(override, args)

        version = package_version()
        asset = asset_name(*platform_key())
        binary = cache_root() / f"v{version}" / asset
        if not binary.exists():
            download(binary, asset, version)
        return run(str(binary), args)
    except LauncherError as err:
        sys.stderr.write(f"cinegram: {err}\n")
        return 1
    except KeyboardInterrupt:
        return 130
