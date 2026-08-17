---
name: publish-site
description: >
  Work on the Cinegram website (www/, zensical.toml, pkg/embedkit) or the
  examples it publishes. Use when anything under examples/ changes, when
  README.md changes, when pkg/emit/html or its runtime assets change, when
  //site:site_test fails, or when the user asks to update, preview or publish
  the site.
---

# The website

<https://panset.github.io/cinegram/> is built by **Zensical** from `www/` and
deployed by `.github/workflows/pages.yml` on every push to `main`. Nothing
about the site is committed as HTML: the workflow renders it.

Three things go into the published `_site/`:

| Part | Where it comes from | In git? |
| --- | --- | --- |
| `www/**` prose | hand-written | yes |
| `www/examples/**`, `www/guide/**`, `www/assets/cinegram/timelines/**` | `bazel run //site:sync` | yes, and checked |
| `www/assets/cinegram/{mermaid.min.js,runtime.*,cinegram-embed.*}` | `cinegram assets` | **no** |
| `/playground/` | `//web/playground:site`, assembled in CI | no |

## Two halves, and why

Bazel cannot run Zensical — it is a Python package, and this repository takes
no dependency it cannot fetch itself (see the `build` skill). So the guarantee
is split: **Bazel owns the input, Zensical owns the render.** `//site:site_test`
regenerates everything a program wrote under `www/` and diffs it against the
committed copy, so Zensical can never be handed a stale page. The workflow
pins its Zensical version for the same reason.

## Change something

1. Regenerate, if you touched `examples/` or `README.md`:

   ```sh
   bazel run //site:sync
   ```

   It prints each `updated`/`removed` file, or `already in sync`. Anything on
   stderr is a compile warning that will ship — fix the example rather than
   publishing a degraded demo.

2. `bazel test //site:site_test` passes.

3. Commit the `www/` changes **in the same commit** as the example or README
   change that caused them.

Done when the test passes and `git status` shows nothing unstaged under `www/`.

## Preview it locally

Zensical is not in the Bazel graph, so it is a one-time install of your own:

```sh
python3 -m venv .venv && .venv/bin/pip install zensical
bazel run //cmd/cinegram -- assets -o "$PWD/www/assets/cinegram"   # once per cinegram change
.venv/bin/zensical serve            # or: .venv/bin/zensical build --strict
```

`--strict` fails on a dead link or anchor, which is what CI runs. The
generated pages are full of cross-references nobody hand-checks.

The kit has to be installed before the first build: `www/assets/cinegram/` is
gitignored apart from `timelines/`, so a fresh checkout has no player in it and
every diagram will fail to load until you run `assets`.

## What the generator decides for you

`site/site.go` and `site/guide.go` are the whole of it; `pkg/sitegen.Discover`
is the shared answer to "what publishes", used by both this site and the
`cinegram site` command.

- Every `examples/**/*.dgm` gets a page **except** one another example pulls in
  via `view … from` — readers reach it by drill-down. (Two examples
  referencing each other still publish: the first alphabetically.)
- The index blurb is the example's leading `%%` comment block, up to a
  `%% ---` separator. Write that block for a site visitor.
- **Generated filenames keep their `01-` prefix.** Zensical builds navigation
  by sorting filenames and there is no `nav` in `zensical.toml`, so the prefix
  is the only thing holding the tour and the guide in reading order. It shows
  in the URL and nowhere else — page titles come from each page's H1.
- `README.md` is cut into the guide by `##` heading, grouped by the table at
  the top of `site/guide.go`. A heading that table does not claim **fails the
  build**, rather than quietly missing from the site. Add it to a page.
- README links are rewritten: a `#anchor` becomes a link to whichever guide
  page now holds that heading, and a repository path goes to GitHub. A link
  the rewriter does not understand is an error.
- `data-height` on each example is a guess from step count (`reservedHeight`),
  measured against the real pages. It only holds space until the player mounts.
- The sweep owns three folders whole and nothing else, so hand-written pages
  are never deleted. `site.Generated` is the list.

**Moving an example moves its published URL** and the sweep deletes the page at
the old path with nothing redirecting. Weigh that before reorganising.

## Embedding is the same path

`www/**` plays its diagrams through `pkg/embedkit` — the loader and stylesheet
that `cinegram assets` installs, documented at `www/embedding.md`. That is
deliberate: the site is the dogfood, so a regression in the embed path takes
these pages down before it reaches anyone else's.

The two files in `pkg/embedkit/assets/` are the canonical copy, under the same
rule as `runtime.js`: `go:embed` cannot reach outside its package, so a second
copy anywhere is a copy that drifts.

## If the site is not live at all

One-time repository setting, not a file: Settings → Pages → Source =
"GitHub Actions" (or `gh api -X PUT repos/panset/cinegram/pages -f
build_type=workflow`), then `gh workflow run pages.yml`. If a deploy looks
wrong, check the latest `pages` run before touching settings — a stale `www/`
fails the qualify step and blocks the deploy by design.
