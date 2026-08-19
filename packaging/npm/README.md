# cinegram

Animated, narrated architecture diagrams from a Mermaid-compatible DSL. A
`.dgm` file is a Mermaid diagram plus `scenario`, `view` and `interact` blocks;
`cinegram` compiles it into a self-contained HTML page that plays.

```sh
npx cinegram preview diagram.dgm -o out.html
npx cinegram lint diagram.dgm --strict
npx cinegram record diagram.dgm -o out.gif
```

## What this package is

A launcher, not a copy of the compiler. cinegram itself is a single static Go
binary published on
[GitHub releases](https://github.com/panset/cinegram/releases); this package
downloads the one that matches your platform the first time you run it, checks
it against the release's `SHA256SUMS`, and executes it. There is no install
script, so `npm install --ignore-scripts` changes nothing: a failure can only
happen at the moment you run the command, in front of you.

The version is pinned to this package's own version, so `npx cinegram@0.3.0`
runs exactly the v0.3.0 binary. Because the cache entry is version-pinned,
`cinegram upgrade` refuses to run under the launcher — install a newer version
instead (`npx cinegram@latest …`).

Supported platforms: `darwin-arm64`, `darwin-x64`, `linux-x64`, `linux-arm64`,
`win32-x64`. Anything else gets a clear error naming the platform.

## Cache

Binaries land in one directory per version:

| | |
| --- | --- |
| macOS, Linux | `~/.cinegram/bin/v<version>/cinegram-<os>-<arch>` |
| Windows | `%LOCALAPPDATA%\cinegram\bin\v<version>\cinegram-win32-x64.exe` |

Delete the directory to force a fresh download.

## Environment

| Variable | Effect |
| --- | --- |
| `CINEGRAM_BIN` | Run this binary and skip everything above — no download, no cache. |
| `CINEGRAM_CACHE_DIR` | Use this directory instead of the default cache root. |
| `CINEGRAM_DOWNLOAD_BASE` | Fetch from a mirror instead of GitHub releases; the launcher appends `/v<version>/<asset>`. For testing and internal mirrors — the checksum check still applies. |

Some subcommands need more than the binary: PNG frames and `record` need a
Chrome/Chromium (`$CINEGRAM_CHROME`), and `--format mp4|webm` needs ffmpeg
(`$CINEGRAM_FFMPEG`). GIF recording needs nothing else.

## More

- [Documentation](https://panset.github.io/cinegram/)
- [Source](https://github.com/panset/cinegram)
- Also available as `uvx cinegram`, as a VS Code extension, and as a plain
  binary download.
