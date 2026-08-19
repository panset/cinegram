# Releasing cinegram

Everything ships from one tag. Pushing `v<MAJOR.MINOR.PATCH>` runs
`.github/workflows/release.yml`, which qualifies the commit, builds every
deliverable once, fans out to one job per distribution channel, and then
installs from the published release to prove the whole path. Nothing is ever
built or published by hand.

## Cutting a release

1. Bump the version in its five homes — they must agree, and
   `//editors/vscode:assets_test` fails when they do not:
   - `cmd/cinegram/version.go` (the constant)
   - `editors/vscode/package.json` (`"version"`)
   - `editors/vscode/CHANGELOG.md` (add the new `## [x.y.z] — date` entry;
     this text becomes the Marketplace changelog)
   - `packaging/npm/package.json` (`"version"`)
   - `packaging/pypi/pyproject.toml` (`version = "x.y.z"`)

   The last two are not bookkeeping: each shim downloads the release named by
   its own manifest, so a stale one makes `npx cinegram@x.y.z` fetch something
   else entirely.
2. Land that commit on `main` through a PR like any other (CI runs
   `bazel test //...`).
3. Tag and push:

   ```sh
   git tag v0.2.0 && git push origin v0.2.0
   ```

That is the whole procedure. If the tag does not match the source version,
the workflow's first step refuses it — delete the tag, fix, re-tag.

## What the pipeline does

```
qualify   bazel test //... + version gate; cross-builds the 5 CLI binaries
          and the 5 platform .vsix packages; uploads both as artifacts
smoke     runs the actual shipped binaries on macOS, Linux and Windows
          runners: version, lint, compile, preview, and a real GIF record
          through headless Chrome
channel-github-binaries    gh release create with binaries + SHA256SUMS + .vsix
channel-vscode-marketplace PUTs each .vsix to the gallery REST API
channel-npm                publishes packaging/npm — the `npx cinegram` launcher
channel-pypi               publishes packaging/pypi — the `uvx cinegram` launcher
verify    downloads releases/latest/download/cinegram-$TARGET exactly as the
          skill does, checks it announces the tag's version, runs
          `cinegram upgrade --check`, then runs `npx cinegram@$V version` and
          `uvx cinegram==$V version` so each shim's own download path is
          exercised end to end
```

Channel jobs are independent: if one fails (an expired PAT, a Marketplace
hiccup), fix the cause and re-run just that job from the Actions UI — the
qualified artifacts are still attached to the run.

## The contracts

Break one of these and installs break somewhere you cannot see:

- **Asset names are `cinegram-<os>-<arch>`** (`darwin-arm64`, `darwin-x64`,
  `linux-x64`, `linux-arm64`, `win32-x64.exe`), following VS Code's platform
  vocabulary. `skills/cinegram/SKILL.md` downloads them by name, and so do
  `cinegram upgrade` (`cmd/cinegram/upgrade.go`) and both package-manager
  launchers (`packaging/npm/bin/cinegram.js`,
  `packaging/pypi/src/cinegram/__init__.py`) — four copies of one table, kept
  honest by the release jobs' gate below.
- **`releases/latest/download/<asset>` must keep working** — it is the skill's
  install URL and how `upgrade` discovers the newest version (via the
  `/releases/latest` redirect, deliberately no GitHub API).
- **`SHA256SUMS` ships with every release** — `upgrade` refuses a binary that
  does not match it, and so do both package-manager shims. The `channel-npm`
  and `channel-pypi` jobs gate on it from the other side: before either
  publishes, every asset name their platform tables can produce must appear in
  the release's `SHA256SUMS`, so a platform can never be added to a launcher
  without being added to the build matrix.
- **One version everywhere.** The tag, the CLI constant, `package.json`, the
  changelog top entry and the two packaging manifests all agree; the version
  gate plus `//editors/vscode:assets_test` enforce it. A published binary must
  never claim a version other than the tag it was built from, or `upgrade`
  loops — and a shim manifest must never name a version whose release does not
  exist, or every `npx cinegram@that` 404s.
- **A channel consumes qualified artifacts and never rebuilds.** If it built
  its own copy, the smoke tests would have qualified something other than
  what shipped.

## Adding a distribution channel

A playground, a package manager, a container image — the pattern is the same:

1. Add one job to `release.yml` with `needs: smoke`, downloading the `dist`
   and/or `vsix` artifacts. Do not rebuild anything in it.

   The exception is a channel that ships a *launcher* rather than the bytes:
   `channel-npm` and `channel-pypi` publish packages that resolve
   `releases/download/v$V/<asset>` the first time a user runs them, so they
   also need `channel-github-binaries` — published a moment before the release
   exists, they would ship a package whose every invocation 404s. Such a
   channel still consumes the `dist` artifact, to gate on the asset names it
   will ask for at run time.
2. Give it a post-publish check, in the job or in `verify`, that exercises the
   path a user would actually take to reach the new channel.
3. If the channel changes how users obtain cinegram, update
   `skills/cinegram/SKILL.md` (step 0 is the install procedure agents follow)
   and the install snippet in the release notes above.
4. Document any new secret below.

A channel that serves *content* rather than the binary can instead ride
`main` the way GitHub Pages does. `.github/workflows/pages.yml` runs on every
push to `main`, gates on `bazel test //...`, and assembles the site there:
Zensical renders the committed `www/` (whose generated half `//site:sync`
writes and `//site:site_test` keeps honest), `cinegram assets` installs the
player into it, and `bazel run //web/playground:site` puts the playground at
`/playground/` so its 6.4 MB `.wasm` never enters git. None of it needs a
`release.yml` channel job, because none of it ships the CLI binary — anything
that does must go through a channel job here.

## Secrets

| Secret | Used by | Notes |
| --- | --- | --- |
| `MARKETPLACE_PAT` | `channel-vscode-marketplace` | Azure DevOps PAT: organization **All accessible organizations**, scope **Marketplace → Manage**. Expires (max ~1 year); created 2026-08-12. On a 401 at release time, mint a new one at dev.azure.com → user settings → Personal access tokens, then `gh secret set MARKETPLACE_PAT -R panset/cinegram`. |
| *(none)* | `channel-npm` | npm publishing uses **trusted publishing**, not a token: npmjs.com → package `cinegram` → Settings → Trusted publishing trusts this repository's `release.yml`, and the job's `id-token: write` permission is the whole credential. Tokens are a dead end here — npm is retiring 2FA-bypass tokens (publishing via them ends ~January 2027), and a non-bypass token just fails CI with `EOTP`. |
| *(none)* | `channel-pypi` | PyPI publishing uses **trusted publishing**, not a token: PyPI is told to trust this repository's `release.yml`, and the job's `id-token: write` permission is the whole credential. Nothing to rotate, nothing to leak. |

The GitHub release itself uses the workflow's own `GITHUB_TOKEN`; npm
provenance and PyPI trusted publishing both use the run's OIDC identity. No
other credentials exist.

## Manual follow-ups before the first release carrying the shims

The workflow is complete, but the account-side half of it is not something CI
can do. These happen once, before the tag that first publishes the shims:

1. **Claim the names.** Check `npm view cinegram` and
   https://pypi.org/project/cinegram/ — both must be free, or taken by us. If
   either is taken, fall back to `@cinegram/cli` on npm (a scoped package,
   which needs the `--access public` the job already passes) or `cinegram-cli`
   on PyPI, and change the `name` in that manifest, its README, this file and
   the `verify` job's invocation together.
2. **Publish to npm once by hand, then register the trusted publisher.**
   Unlike PyPI, npm has no pending publisher: the package must exist before
   trust can be configured, and account-level 2FA is mandatory for publishing.
   So, from `packaging/npm`: `npm login`, then
   `npm publish --access public --otp=<code>`. Then at npmjs.com → package
   `cinegram` → Settings → Trusted publishing add GitHub Actions with
   organization `panset`, repository `cinegram`, workflow `release.yml`,
   environment blank, allowed action *npm publish*. (Equivalent CLI, npm ≥
   11.15: `npm trust github cinegram --file release.yml --repo
   panset/cinegram --allow-publish`.) Done 2026-08-19 for `cinegram@0.4.0`.
3. **Register the PyPI trusted publisher** at
   https://pypi.org/manage/account/publishing/ before the project exists —
   PyPI calls this a *pending publisher*. Owner `panset`, repository
   `cinegram`, workflow `release.yml`, environment blank. The first successful
   run creates the project.
4. **Confirm afterwards**: `npx cinegram@<V> version` and
   `uvx cinegram==<V> version` from a machine that has never run them, so the
   download-and-cache path is exercised cold. The `verify` job does this too,
   but on a runner that shares nothing with a user's home directory.

## Rolling back

`releases/latest` points at the newest non-prerelease release. To pull a bad
version out of the install path without deleting history:

```sh
gh release edit v0.2.0 -R panset/cinegram --latest=false --prerelease
```

then tag and release a fixed version. `cinegram upgrade` and the skill both
follow `latest`, so they move as soon as it does. The Marketplace has no
unpublish-a-version; ship a higher version instead.

## Branch protection

CI only gates merges if the branch requires it. One-time setup (or via
Settings → Branches):

```sh
gh api -X PUT repos/panset/cinegram/branches/main/protection \
  --input - <<'EOF'
{
  "required_status_checks": {"strict": false, "contexts": ["test"]},
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null
}
EOF
```
