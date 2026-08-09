// Finding the cinegram binary.
//
// The extension knows nothing about diagrams: it shells out to the compiler and
// renders whatever comes back. That keeps the two in step for free — a new
// diagram type or timing rule reaches the preview with no change here — but it
// does mean the binary has to be found before anything can be drawn.
//
// The order below runs from most deliberate to most incidental, so an explicit
// answer always wins over a guess.

'use strict';

const fs = require('fs');
const path = require('path');
const vscode = require('vscode');

/** Cached result of the last search, cleared when the setting changes. */
let cached = null;

function reset() {
  cached = null;
}

/**
 * Returns { path, source } for the binary to run, or null if none was found.
 * `source` names which rule matched, so the error message can say what was
 * tried rather than only that it failed.
 */
function resolve() {
  if (cached !== null) return cached;
  cached = search();
  return cached;
}

function search() {
  const configured = vscode.workspace.getConfiguration('cinegram').get('path');
  if (configured) {
    // An explicit path that does not exist is a mistake worth reporting rather
    // than silently falling through to a different binary than the one asked
    // for — which would be far more confusing than an error.
    return { path: configured, source: 'the cinegram.path setting', verified: isExecutable(configured) };
  }

  const bundled = bundledPath();
  if (bundled && isExecutable(bundled)) {
    return { path: bundled, source: 'the copy bundled with the extension', verified: true };
  }

  for (const folder of vscode.workspace.workspaceFolders || []) {
    const built = path.join(folder.uri.fsPath, 'bazel-bin', 'cmd', 'cinegram', 'cinegram_', exe('cinegram'));
    if (isExecutable(built)) {
      return { path: built, source: 'the Bazel build in this workspace', verified: true };
    }
  }

  // Last resort, and unverifiable without running it: let the spawn decide.
  return { path: exe('cinegram'), source: 'your PATH', verified: false };
}

/**
 * Where a per-platform binary would live once the extension ships one. The
 * layout matches what VS Code's own platform-specific packages use: one
 * directory per target, only the matching one present in a given .vsix.
 */
function bundledPath() {
  const target = platformTarget();
  if (!target) return null;
  return path.join(__dirname, '..', 'bin', target, exe('cinegram'));
}

function platformTarget() {
  const os = { darwin: 'darwin', linux: 'linux', win32: 'win32' }[process.platform];
  const arch = { arm64: 'arm64', x64: 'x64' }[process.arch];
  return os && arch ? `${os}-${arch}` : null;
}

function exe(name) {
  return process.platform === 'win32' ? `${name}.exe` : name;
}

function isExecutable(p) {
  try {
    return fs.statSync(p).isFile();
  } catch (e) {
    return false;
  }
}

/**
 * A message for the reader when the binary could not be run, naming what was
 * tried. Shown in place of the diagram, so it has to be useful without a log.
 */
function describeFailure(found, err) {
  const detail = err && err.code === 'ENOENT'
    ? 'it is not there'
    : (err && err.message) || 'it could not be run';

  return `Could not run the cinegram binary (${found ? found.source : 'no candidate found'}): ${detail}. ` +
    'Build it with `bazel build //cmd/cinegram`, or set `cinegram.path` to a binary.';
}

module.exports = { resolve, reset, describeFailure, platformTarget };
