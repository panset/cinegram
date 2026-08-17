package main

// version is the single source of truth for what this build calls itself.
//
// Three copies of a version exist in the repository — this constant, the
// extension's package.json, and the top entry of the extension CHANGELOG —
// and //editors/vscode:assets_test fails when they disagree. The release
// workflow additionally refuses a tag that does not match this constant, so
// a published binary can never claim a version other than the tag it was
// built from. See RELEASING.md.
const version = "0.3.0"
