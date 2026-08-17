---
name: build
description: >
  Build, test, format and exercise the cinegram repository, and the hermetic
  invariants every change must preserve. Use before any change to Go,
  JavaScript or BUILD files, when a bazel or gazelle command is needed, when
  regenerating golden fixtures, and before adding a dependency or a new asset.
---

# Building cinegram

Bazel with bzlmod is the only build system. Go is not installed locally —
`rules_go` fetches a hermetic SDK and Gazelle runs through Bazel, so a bare
`go build` or `go test` has nothing to run.

## Hermetic

The repository takes nothing from outside itself. Four rules are the same rule:

1. **No third-party Go dependency.** Standard library only — a hand-rolled
   lexer and recursive-descent parser need nothing more, and it keeps the build
   hermetic with no `go_deps` lockfile to churn.
2. **Runtime assets live only in `pkg/emit/html/assets/`.** `go:embed` cannot
   reach outside its own package. That directory is the single canonical copy
   of `runtime.js`, `runtime.css` and the vendored `mermaid.min.js` (2.7 MB,
   committed because the build needs it). Gazelle generates `embedsrcs` from
   the `//go:embed` directives.
3. **`runtime.js` is a classic script, never an ES module.** Module scripts are
   blocked on `file://` by CORS and are awkward in VS Code webviews.
4. **The preview page carries no external URL at all.** It has to work from the
   filesystem and under a webview CSP; `pkg/emit/html/html_test.go` enforces it.

## Build and test

```sh
bazel build //...
bazel test  //...
bazel run   //:gazelle       # after adding, moving or renaming a package
```

One test: `bazel test //pkg/parser:parser_test --test_filter=TestSplitLinks`,
with `--test_output=all` to see it.

Gazelle must leave `web/playground/BUILD.bazel` and
`web/playground/wasm/BUILD.bazel` untouched — each opens with
`# gazelle:ignore` and is hand-written for reasons the file itself states.

## Regenerate golden fixtures

`-update` writes through `BUILD_WORKSPACE_DIRECTORY`, which only `bazel run`
sets; under `bazel test` the sandbox makes the write fail.

```sh
bazel run //pkg/parser:parser_test   -- -update
bazel run //pkg/compile:compile_test -- -update
```

Done when `bazel test //...` is green and `git diff` shows only the fixtures
you meant to change.

## Before finishing

`gofmt` exists only inside the hermetic SDK, and Bazel does not check it:

```sh
"$(find "$(bazel info output_base)/external" -name gofmt -type f | head -1)" -w ./cmd ./pkg ./site
```

Done when `bazel test //...` is green and gofmt has run.

## Exercise the CLI

Relative paths work — the binary resolves them against `BUILD_WORKING_DIRECTORY`.

```sh
bazel run //cmd/cinegram -- preview examples/01-basics/01-k8s-request.dgm -o /tmp/k8s.html
bazel run //cmd/cinegram -- site examples -o /tmp/site   # folder tree → browsable site
bazel run //cmd/cinegram -- compile examples/01-basics/01-k8s-request.dgm
bazel run //cmd/cinegram -- mermaid examples/01-basics/01-k8s-request.dgm
bazel run //cmd/cinegram -- lint    examples/01-basics/01-k8s-request.dgm
bazel run //cmd/cinegram -- record  examples/02-storytelling/01-payment-checkout.dgm -o /tmp/out.gif --fps 10
```

`record` shells out to the same headless Chrome `frame` uses, once per frame in
a small worker pool, then encodes. GIF goes through `pkg/gifenc` — stdlib only,
so it works with nothing installed; `--format mp4|webm` needs ffmpeg on `PATH`
or `$CINEGRAM_FFMPEG`.
