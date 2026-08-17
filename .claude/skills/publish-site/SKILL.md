---
name: publish-site
description: >
  Regenerate and publish the Cinegram GitHub Pages demo site (docs/). Use
  when anything under examples/ changes, when pkg/emit/html or its runtime
  assets change, when //site:site_test fails, or when the user asks to update
  or publish the demo site.
---

# Publishing the demo site

**`docs/` is generated — regenerate it, never edit it.** Nothing rebuilds it on
the way to production, so an edit by hand ships as-is and is lost at the next
`sync`.

`docs/` is the site GitHub Pages serves. The `pages` workflow
(`.github/workflows/pages.yml`) runs on every push to `main`: it gates on
`bazel test //...`, uploads the committed `docs/` **verbatim** (it never
rebuilds it), and assembles the playground beside it at `/playground/` — so
committing `docs/` to `main` is still what publishes the demo pages. The
generator is `pkg/sitegen` — the same one `cinegram site` exposes — wrapped
by `site/` with the repo's presentation (title, hero, playground and GitHub
links) and rooted at `docs/demos/`. A page's URL is its source's path under
`examples/`, so **moving an example moves its published URL** — the sweep
deletes the page at the old path and nothing redirects. Weigh that before
reorganising the tree. Pages share one runtime copy from `demos/assets/` instead of
inlining mermaid per page; the generated `demos/index.html` is the landing
page, and the top-level `docs/index.html` is only a redirect into it.

## Update

1. Regenerate:

   ```sh
   bazel run //site:sync
   ```

   It prints each `updated`/`removed` file, or `already in sync`. Anything on
   stderr is a compile warning that will ship to the live site — fix the
   example rather than publishing a degraded demo.

2. Verify: `bazel test //site:site_test` passes.

3. Commit the `docs/` changes **in the same commit** as the example or
   renderer change that caused them, so no commit on `main` serves a stale
   site.

Done when the test passes and `git status` shows nothing unstaged under
`docs/`.

## The tour

`examples/` is a tour, not a bag of files: `01-basics/` → `04-diagram-types/`,
and `docs/demos/` mirrors it folder for folder. `pod-a.dgm` stays at the root
because two demos in different folders drill into it (`../pod-a.dgm`).

## What the generator decides for you

- Every `examples/**/*.dgm` gets a page **except** one that another example
  pulls in via `view … from` — readers reach it by drill-down. (Two examples
  referencing each other still publish: the first alphabetically.)
- The index blurb is the example's leading `%%` comment block, up to a
  `%% ---` separator. Write that block for a site visitor.
- Subfolders of `examples/` become site folders with their own index; an
  optional numeric filename prefix (`01-name.dgm`) orders entries and is
  stripped from links, titles and page names.
- Two examples must not publish as the same page (same folder, same
  prefix-stripped name); `sync` fails naming the collision.
- Every demo page carries an *Edit in playground* button whose link encodes
  the whole document — the same format as the playground's Share button,
  minted at build time.
- The sweep touches only `docs/demos/` (recursively — it nests now); a
  hand-placed top-level file in `docs/` (a `CNAME`, say) survives.

## If the site is not live at all

One-time repository setting, not a file: Settings → Pages → Source =
"GitHub Actions" (or `gh api -X PUT repos/panset/cinegram/pages -f
build_type=workflow`), then run the `pages` workflow
(`gh workflow run pages.yml`). The site then serves at
<https://panset.github.io/cinegram/>. If a deploy looks wrong, check the
latest `pages` run before touching settings — a stale `docs/` fails the
qualify step and blocks the deploy by design.
