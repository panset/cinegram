# Releasing cinegram

Everything ships from one tag. Pushing `v<MAJOR.MINOR.PATCH>` runs
`.github/workflows/release.yml`, which qualifies the commit, builds every
deliverable once, fans out to one job per distribution channel, and then
installs from the published release to prove the whole path. Nothing is ever
built or published by hand.

## Cutting a release

1. Bump the version in its three homes — they must agree, and
   `//editors/vscode:assets_test` fails when they do not:
   - `cmd/cinegram/version.go` (the constant)
   - `editors/vscode/package.json` (`"version"`)
   - `editors/vscode/CHANGELOG.md` (add the new `## [x.y.z] — date` entry;
     this text becomes the Marketplace changelog)
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
verify    downloads releases/latest/download/cinegram-$TARGET exactly as the
          skill does, checks it announces the tag's version, runs
          `cinegram upgrade --check`
```

Channel jobs are independent: if one fails (an expired PAT, a Marketplace
hiccup), fix the cause and re-run just that job from the Actions UI — the
qualified artifacts are still attached to the run.

## The contracts

Break one of these and installs break somewhere you cannot see:

- **Asset names are `cinegram-<os>-<arch>`** (`darwin-arm64`, `darwin-x64`,
  `linux-x64`, `linux-arm64`, `win32-x64.exe`), following VS Code's platform
  vocabulary. `skills/cinegram/SKILL.md` downloads them by name, and so does
  `cinegram upgrade` (`cmd/cinegram/upgrade.go`).
- **`releases/latest/download/<asset>` must keep working** — it is the skill's
  install URL and how `upgrade` discovers the newest version (via the
  `/releases/latest` redirect, deliberately no GitHub API).
- **`SHA256SUMS` ships with every release** — `upgrade` refuses a binary that
  does not match it.
- **One version everywhere.** The tag, the CLI constant, `package.json` and
  the changelog top entry all agree; the version gate plus
  `//editors/vscode:assets_test` enforce it. A published binary must never
  claim a version other than the tag it was built from, or `upgrade` loops.
- **A channel consumes qualified artifacts and never rebuilds.** If it built
  its own copy, the smoke tests would have qualified something other than
  what shipped.

## Adding a distribution channel

A playground, a package manager, a container image — the pattern is the same:

1. Add one job to `release.yml` with `needs: smoke`, downloading the `dist`
   and/or `vsix` artifacts. Do not rebuild anything in it.
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

The GitHub release itself uses the workflow's own `GITHUB_TOKEN`; no other
credentials exist.

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
