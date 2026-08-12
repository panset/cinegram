---
name: publish-site
description: >
  Regenerate and publish the Cinegram GitHub Pages demo site (docs/). Use
  when anything under examples/ changes, when pkg/emit/html or its runtime
  assets change, when //site:site_test fails, or when the user asks to update
  or publish the demo site.
---

# Publishing the demo site

`docs/` is the site GitHub Pages serves, straight from `main` — there is no
CI build after the commit, so committing `docs/` **is** publishing. The
generator lives in `site/`; it renders one self-contained page per standalone
example plus an index.

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

## What the generator decides for you

- Every `examples/*.dgm` gets a page **except** one that another example
  pulls in via `view … from` — readers reach it by drill-down. (Two examples
  referencing each other still publish: the first alphabetically.)
- The index blurb is the example's leading `%%` comment block, up to a
  `%% ---` separator. Write that block for a site visitor.
- Two examples must not share a basename; `sync` fails naming the collision.
- The sweep touches only `docs/demos/`; a hand-placed top-level file in
  `docs/` (a `CNAME`, say) survives.

## If the site is not live at all

One-time repository setting, not a file: Settings → Pages → Deploy from a
branch → `main`, `/docs`. The site then serves at
<https://panset.github.io/cinegram/>.
