// Exporting a .dgm to a GIF, mp4 or webm.
//
// This is the one place in the extension that does *not* go through
// src/compile.js, and the asymmetry is deliberate. Compiling is `execFileSync`
// because markdown-it's renderer cannot await anything, and a compile is
// milliseconds. Recording is neither: `cinegram record` spawns one headless
// Chrome per frame, four at a time, so a ten-second scenario at 12fps is 121
// browsers and can run for minutes. Blocking the extension host for that would
// freeze the editor, and finishing with no sign of progress would look like a
// hang. So export gets `child_process.spawn`, a cancellable notification, and
// the `--progress` protocol the CLI grew for it.
//
// Everything else is the usual rule: the extension holds no diagram knowledge.
// The view and scenario pickers are filled from a compile envelope, the output
// format is whatever the save dialog's extension implies (`record` infers it
// from the filename), and the CLI's own failure messages are surfaced verbatim
// — they already name CINEGRAM_CHROME and CINEGRAM_FFMPEG, which is more than a
// rewritten message would.

'use strict';

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');
const vscode = require('vscode');

const binary = require('./binary');
const compile = require('./compile');

/** The `Cinegram` output channel: every line the recorder writes lands here. */
let channel = null;

/**
 * Exports in flight, keyed by output path. A double-click on the command must
 * not put two encoders on one file, and the file is the only thing worth
 * guarding — two exports of the same source to different names are fine.
 */
const running = new Map();

/** Capture fills the bar to here; encoding holds it there until the file lands. */
const CAPTURE_SHARE = 95;

function register(context) {
  channel = vscode.window.createOutputChannel('Cinegram');
  context.subscriptions.push(channel);

  context.subscriptions.push(
    vscode.commands.registerCommand('cinegram.export', (uri) => exportAnimation(uri))
  );
}

// --- the command --------------------------------------------------------------

async function exportAnimation(uri) {
  const target = uri || activeUri();
  if (!target) {
    vscode.window.showInformationMessage('Open a .dgm file to export it.');
    return;
  }
  // `record` reads from disk with os.ReadFile, so anything that is not a real
  // file — an untitled buffer, a git diff, a remote scheme — has nothing to
  // record from.
  if (target.scheme !== 'file') {
    vscode.window.showInformationMessage('Save this diagram to a file before exporting it.');
    return;
  }
  if (!(await ensureSaved(target))) return;

  // Compile first. It costs milliseconds and it is what stops a two-minute
  // record from failing on a typo — and it is also where the view and scenario
  // lists come from, so it is not a check that could be skipped for speed.
  const result = compile.file(target.fsPath);
  const errors = (result.diagnostics || []).filter((d) => d.severity === 'error');
  if (errors.length) {
    logDiagnostics(target.fsPath, errors);
    show(await vscode.window.showErrorMessage(describeDiagnostic(errors[0]), 'Show Output'));
    return;
  }
  if (!result.timeline) {
    vscode.window.showErrorMessage(path.basename(target.fsPath) + ' produced no diagram to record.');
    return;
  }

  const view = await pickView(result.timeline);
  if (!view) return;
  const scenario = await pickScenario(view);
  if (!scenario) return;

  const out = await vscode.window.showSaveDialog({
    // The extension chosen here *is* the format: `record` infers the container
    // from the output name, so offering these filters removes a prompt rather
    // than adding one.
    defaultUri: vscode.Uri.file(swapExtension(target.fsPath, '.gif')),
    filters: { GIF: ['gif'], MP4: ['mp4'], WebM: ['webm'] },
    saveLabel: 'Export'
  });
  if (!out) return;

  if (running.has(out.fsPath)) {
    vscode.window.showInformationMessage('Already exporting to ' + path.basename(out.fsPath) + '.');
    return;
  }
  running.set(out.fsPath, true);
  try {
    await runExport(target, out, view, scenario);
  } finally {
    running.delete(out.fsPath);
  }
}

/**
 * What the command means by "this diagram" when nothing was passed to it.
 *
 * The active *text* editor is the usual answer, but a .dgm opened through
 * *Open With… → Cinegram Animation* is not one — it is a webview tab — so the
 * command palette would otherwise be useless from the very editor this
 * extension just added. The tab's input carries the uri either way.
 */
function activeUri() {
  const editor = vscode.window.activeTextEditor;
  if (editor) return editor.document.uri;

  const groups = vscode.window.tabGroups;
  const tab = groups && groups.activeTabGroup && groups.activeTabGroup.activeTab;
  const input = tab && tab.input;
  return input && input.uri ? input.uri : undefined;
}

/**
 * Recording reads the file on disk, so a dirty buffer would silently record the
 * previous version — the kind of wrong answer that is only noticed after the
 * GIF is in a pull request. Modal, because the alternative is a toast nobody
 * reads while they wait for a minute of recording that was never going to be
 * right.
 */
async function ensureSaved(uri) {
  const doc = vscode.workspace.textDocuments.find((d) => d.uri.toString() === uri.toString());
  if (!doc || !doc.isDirty) return true;

  const choice = await vscode.window.showWarningMessage(
    path.basename(uri.fsPath) + ' has unsaved changes, and recording reads the file from disk.',
    { modal: true },
    'Save and Export'
  );
  if (choice !== 'Save and Export') return false;
  return doc.save();
}

/** The view to record: no prompt when a document holds only one diagram. */
async function pickView(timeline) {
  const views = timeline.views || [];
  if (!views.length) return null;
  if (views.length === 1) return views[0];

  const items = views.map((v) => ({
    label: v.title || v.id,
    description: v.id === timeline.root ? 'opens here' : '',
    detail: plural((v.scenarios || []).length, 'scenario'),
    view: v
  }));
  const pick = await vscode.window.showQuickPick(items, { placeHolder: 'Which diagram?' });
  return pick ? pick.view : null;
}

/** Scenarios are per view, which is why the view is chosen first. */
async function pickScenario(view) {
  const scenarios = view.scenarios || [];
  if (!scenarios.length) {
    vscode.window.showErrorMessage(
      (view.title || view.id) + ' has no scenarios, so there is no animation to record.'
    );
    return null;
  }
  if (scenarios.length === 1) return scenarios[0];

  const items = scenarios.map((s) => ({
    label: s.name || s.id,
    detail: formatDuration(s.duration),
    scenario: s
  }));
  const pick = await vscode.window.showQuickPick(items, { placeHolder: 'Which scenario?' });
  return pick ? pick.scenario : null;
}

// --- running the recorder -------------------------------------------------------

async function runExport(source, out, view, scenario) {
  const found = binary.resolve();
  const settings = vscode.workspace.getConfiguration('cinegram');

  const args = [
    'record', source.fsPath,
    '-o', out.fsPath,
    '--progress',
    '--fps', String(settings.get('record.fps') || 12),
    '--width', String(settings.get('record.width') || 1280),
    '--height', String(settings.get('record.height') || 720),
    '--view', view.id,
    '--scenario', scenario.id
  ];

  channel.appendLine('$ ' + found.path + ' ' + args.join(' '));

  const outcome = await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Exporting ' + path.basename(out.fsPath),
      cancellable: true
    },
    (progress, token) => spawnRecorder(found, args, progress, token)
  );

  if (outcome.cancelled) {
    channel.appendLine('cancelled');
    return;
  }
  if (outcome.error || outcome.code !== 0) {
    const message = outcome.error
      ? binary.describeFailure(found, outcome.error)
      : outcome.summary || 'Recording failed. See the Cinegram output for what the recorder said.';
    show(await vscode.window.showErrorMessage(message, 'Show Output'));
    return;
  }

  await announce(source, out, scenario);
}

/**
 * Spawns `record` and translates its stderr into the notification.
 *
 * Never rejects: a failure is a value, because the caller has to tell "the user
 * pressed Cancel" apart from "ffmpeg is missing" and both are ordinary
 * outcomes.
 */
function spawnRecorder(found, args, progress, token) {
  return new Promise((resolve) => {
    let child;
    try {
      child = spawn(found.path, args, {
        env: recorderEnv(),
        // A process group of its own, so cancelling can take the browsers with
        // it. Without this, killing the recorder orphans up to four headless
        // Chromes that go on rendering frames nobody will ever see.
        detached: process.platform !== 'win32',
        windowsHide: true
      });
    } catch (err) {
      resolve({ error: err });
      return;
    }

    let cancelled = false;
    const lines = [];
    let reported = 0;
    let pending = '';

    const onLine = (line) => {
      channel.appendLine(line);
      const tick = /^cinegram-progress capture (\d+) (\d+)$/.exec(line);
      if (tick) {
        const done = Number(tick[1]);
        const total = Number(tick[2]);
        // VS Code's `increment` is relative, so what is reported is the change
        // since last time, not the percentage reached.
        const at = total > 0 ? (done / total) * CAPTURE_SHARE : 0;
        progress.report({ increment: at - reported, message: 'Frame ' + done + ' of ' + total });
        reported = at;
        return;
      }
      if (line === 'cinegram-progress encode') {
        progress.report({ message: 'Encoding…' });
        return;
      }
      lines.push(line);
    };

    child.stderr.setEncoding('utf8');
    child.stderr.on('data', (chunk) => {
      pending += chunk;
      let at;
      while ((at = pending.indexOf('\n')) >= 0) {
        onLine(pending.slice(0, at).replace(/\r$/, ''));
        pending = pending.slice(at + 1);
      }
    });
    // stdout is not used by `record`, but an unread pipe fills and stalls the
    // child, so it is drained rather than left alone.
    child.stdout.resume();

    child.on('error', (err) => resolve({ error: err }));
    child.on('close', (code) => {
      if (pending) onLine(pending);
      resolve({ code: code, cancelled: cancelled, summary: summarise(lines) });
    });

    token.onCancellationRequested(() => {
      cancelled = true;
      killTree(child);
    });
  });
}

/**
 * Kills the recorder *and* its browsers.
 *
 * Known and accepted: Go's default SIGTERM handling exits without running
 * deferred functions, so a cancelled record leaves one `cinegram-record-*`
 * directory in the system temp dir. That is a few PNGs the OS will clear;
 * orphaned browsers would be four processes at full tilt, which is the part
 * worth spending a process-group kill on.
 */
function killTree(child) {
  if (!child.pid) return;
  try {
    if (process.platform === 'win32') {
      spawn('taskkill', ['/pid', String(child.pid), '/T', '/F'], { windowsHide: true });
    } else {
      process.kill(-child.pid, 'SIGTERM');
    }
  } catch (e) {
    // The group may already be gone, or never have become one. Either way the
    // child itself is still worth a try.
    try {
      child.kill('SIGTERM');
    } catch (e2) {
      /* nothing left to do */
    }
  }
}

/** The two path settings, passed the way the CLI already reads them. */
function recorderEnv() {
  const settings = vscode.workspace.getConfiguration('cinegram');
  const env = Object.assign({}, process.env);
  const chrome = (settings.get('chromePath') || '').trim();
  const ffmpeg = (settings.get('ffmpegPath') || '').trim();
  if (chrome) env.CINEGRAM_CHROME = chrome;
  if (ffmpeg) env.CINEGRAM_FFMPEG = ffmpeg;
  return env;
}

/**
 * The first thing worth reading out of the recorder's stderr.
 *
 * `record`'s own messages are multi-line on purpose — the ffmpeg one names the
 * environment variable *and* suggests recording a GIF instead — so the summary
 * runs to the end of the message rather than stopping at the first line, and
 * the full text is in the output channel regardless.
 */
function summarise(lines) {
  const said = lines
    .map((l) => l.trim())
    .filter((l) => l && !/^recording /.test(l) && !/^wrote /.test(l));
  if (!said.length) return '';
  const text = said.join(' ');
  return text.length > 400 ? text.slice(0, 397) + '…' : text;
}

// --- what happens after ---------------------------------------------------------

async function announce(source, out, scenario) {
  let size = '';
  try {
    size = ' (' + Math.max(1, Math.round(fs.statSync(out.fsPath).size / 1024)) + ' KB)';
  } catch (e) {
    /* the file is there; the recorder just told us so */
  }

  const choice = await vscode.window.showInformationMessage(
    'Wrote ' + path.basename(out.fsPath) + size,
    'Open',
    'Reveal',
    'Copy Markdown'
  );
  if (choice === 'Open') {
    // VS Code's own image viewer animates a GIF and plays a video.
    vscode.commands.executeCommand('vscode.open', out);
  } else if (choice === 'Reveal') {
    vscode.commands.executeCommand('revealFileInOS', out);
  } else if (choice === 'Copy Markdown') {
    const snippet = markdownFor(source, out, scenario);
    await vscode.env.clipboard.writeText(snippet);
    vscode.window.showInformationMessage('Copied ' + snippet);
  }
}

/**
 * `![scenario](demo.gif)`, relative to the .dgm's own directory — which is
 * where a reader who found the diagram will be writing the prose that embeds
 * it. Always posix separators: a Markdown link is a URL, not a path.
 */
function markdownFor(source, out, scenario) {
  const rel = path
    .relative(path.dirname(source.fsPath), out.fsPath)
    .split(path.sep)
    .join('/');
  const href = rel.replace(/ /g, '%20');
  const alt = String(scenario.name || scenario.id || '').replace(/[[\]]/g, '');
  return '![' + alt + '](' + href + ')';
}

// --- odds and ends ----------------------------------------------------------------

function logDiagnostics(file, diagnostics) {
  channel.appendLine(file + ': ' + plural(diagnostics.length, 'error'));
  for (const d of diagnostics) {
    channel.appendLine('  ' + describeDiagnostic(d));
    if (d.hint) channel.appendLine('    ' + d.hint);
  }
}

function describeDiagnostic(d) {
  const where = d.line ? path.basename(d.file || '') + ':' + d.line + ': ' : '';
  return where + d.message;
}

function show(choice) {
  if (choice === 'Show Output') channel.show(true);
}

function swapExtension(fsPath, ext) {
  return path.join(path.dirname(fsPath), path.basename(fsPath, path.extname(fsPath)) + ext);
}

function formatDuration(ms) {
  if (!ms) return '';
  return (ms / 1000).toFixed(1).replace(/\.0$/, '') + 's';
}

function plural(n, word) {
  return n + ' ' + word + (n === 1 ? '' : 's');
}

module.exports = { register };
