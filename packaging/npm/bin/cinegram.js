#!/usr/bin/env node
// `npx cinegram` — fetch the released binary for this platform, then get out of
// the way.
//
// cinegram is a Go program shipped as one static binary per platform. This
// package carries no binary of its own and runs no install script: an install
// script is skipped by `npm install --ignore-scripts` and fails at a moment
// when nobody is watching, so the download happens here, at the first use, and
// any failure lands in front of the person who asked for it.
//
// The version is the package's own — `require('../package.json').version` — so
// `npx cinegram@0.3.0` fetches exactly the v0.3.0 binaries and nothing else
// ever decides which release to use. That is also why `cinegram upgrade`
// refuses to run under this launcher (see cmd/cinegram/upgrade.go): the cache
// entry is version-pinned, and overwriting it in place would make the pin lie.
//
// The asset names are the release contract, shared with `cinegram upgrade` and
// with the VS Code extension: cinegram-<os>-<arch>, in VS Code's platform
// vocabulary. See RELEASING.md.

'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const crypto = require('crypto');
const { spawnSync } = require('child_process');

const VERSION = require('../package.json').version;

// Where releases live. CINEGRAM_DOWNLOAD_BASE points the launcher at a mirror
// — or, in this repository's own tests, at a local server — and is documented
// in README.md. The checksum check below is not relaxed for it: a mirror has
// to serve the same bytes as the release, or nothing is installed.
const DEFAULT_BASE = 'https://github.com/panset/cinegram/releases/download';

// The platform table. Node's own vocabulary for os and arch happens to be the
// release's — that is where the asset names came from (see
// editors/vscode/src/binary.js) — so this is a translation of nothing much;
// what it really carries is TARGETS, the set of builds that exist. Not every
// pair does: there is no Windows-on-ARM build, and a launcher that cheerfully
// computed one would 404 on the first run. The release workflow gates on this
// exact list appearing in the release's SHA256SUMS.
const OSES = { darwin: 'darwin', linux: 'linux', win32: 'win32' };
const ARCHES = { x64: 'x64', arm64: 'arm64' };
const TARGETS = ['darwin-arm64', 'darwin-x64', 'linux-arm64', 'linux-x64', 'win32-x64'];

function assetName(platform, arch) {
  const target = `${OSES[platform]}-${ARCHES[arch]}`;
  if (!OSES[platform] || !ARCHES[arch] || !TARGETS.includes(target)) {
    throw new Error(
      `no cinegram release is published for ${platform}-${arch}; ` +
        `supported platforms are ${TARGETS.join(', ')}. ` +
        'Build from source (https://github.com/panset/cinegram), or point $CINEGRAM_BIN at a binary you built.'
    );
  }
  return `cinegram-${target}` + (OSES[platform] === 'win32' ? '.exe' : '');
}

// One directory per version, so several pinned versions coexist and none of
// them is ever rewritten.
function cacheRoot() {
  if (process.env.CINEGRAM_CACHE_DIR) return process.env.CINEGRAM_CACHE_DIR;
  if (process.platform === 'win32') {
    const local = process.env.LOCALAPPDATA || path.join(os.homedir(), 'AppData', 'Local');
    return path.join(local, 'cinegram', 'bin');
  }
  return path.join(os.homedir(), '.cinegram', 'bin');
}

function get(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    // http is here only for CINEGRAM_DOWNLOAD_BASE; the default is https.
    const client = url.startsWith('http:') ? require('http') : require('https');
    const req = client.get(
      url,
      { headers: { 'user-agent': `cinegram-npm/${VERSION}`, accept: '*/*' } },
      (res) => {
        const code = res.statusCode;
        if (code >= 300 && code < 400 && res.headers.location) {
          res.resume();
          if (redirects >= 10) return reject(new Error(`too many redirects for ${url}`));
          return resolve(get(new URL(res.headers.location, url).toString(), redirects + 1));
        }
        if (code !== 200) {
          res.resume();
          return reject(new Error(`downloading ${url}: HTTP ${code}`));
        }
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () => resolve(Buffer.concat(chunks)));
        res.on('error', reject);
      }
    );
    req.on('error', (err) => reject(new Error(`downloading ${url}: ${err.message}`)));
  });
}

// checksumFor reads one entry out of sha256sum output ("<hex>  <name>").
function checksumFor(sums, asset) {
  for (const line of sums.toString('utf8').split('\n')) {
    const fields = line.trim().split(/\s+/);
    if (fields.length === 2 && fields[1] === asset) return fields[0];
  }
  throw new Error(`the v${VERSION} release's SHA256SUMS has no entry for ${asset}`);
}

// download fetches SHA256SUMS first and the asset second, verifies the bytes
// before they reach their final name, and moves the file into place with a
// rename so a killed download can never leave a half-written binary behind.
async function download(dest, asset) {
  const base = (process.env.CINEGRAM_DOWNLOAD_BASE || DEFAULT_BASE).replace(/\/+$/, '');
  const dir = `${base}/v${VERSION}`;

  process.stderr.write(`cinegram: downloading ${asset} v${VERSION}…\n`);
  const want = checksumFor(await get(`${dir}/SHA256SUMS`), asset);
  const body = await get(`${dir}/${asset}`);
  const got = crypto.createHash('sha256').update(body).digest('hex');
  if (got !== want) {
    throw new Error(
      `${asset} does not match the v${VERSION} release's SHA256SUMS (got ${got}, want ${want}); not installing it`
    );
  }

  fs.mkdirSync(path.dirname(dest), { recursive: true });
  const tmp = `${dest}.${process.pid}.tmp`;
  try {
    fs.writeFileSync(tmp, body);
    if (process.platform !== 'win32') fs.chmodSync(tmp, 0o755);
    fs.renameSync(tmp, dest);
  } catch (err) {
    try {
      fs.unlinkSync(tmp);
    } catch (_) {
      /* the temp file may never have been created */
    }
    throw err;
  }
}

// run hands the process over: the child owns the terminal, and its exit — code
// or signal — becomes ours, so `npx cinegram lint …` is usable in a script.
function run(bin, args) {
  const res = spawnSync(bin, args, {
    stdio: 'inherit',
    env: Object.assign({}, process.env, { CINEGRAM_MANAGED_BY: 'npm' }),
  });
  if (res.error) {
    if (res.error.code === 'ENOENT') throw new Error(`${bin} is not there`);
    throw new Error(`running ${bin}: ${res.error.message}`);
  }
  if (res.signal) {
    // Die the way the child died, so a Ctrl-C looks like a Ctrl-C to the shell.
    process.kill(process.pid, res.signal);
    return;
  }
  process.exit(res.status === null ? 1 : res.status);
}

async function main(args) {
  // An explicit binary wins over everything: no download, no cache, no version
  // lookup. This is the seam the repository's own tests and the skill use.
  if (process.env.CINEGRAM_BIN) return run(process.env.CINEGRAM_BIN, args);

  const asset = assetName(process.platform, process.arch);
  const bin = path.join(cacheRoot(), `v${VERSION}`, asset);
  if (!fs.existsSync(bin)) await download(bin, asset);
  run(bin, args);
}

main(process.argv.slice(2)).catch((err) => {
  process.stderr.write(`cinegram: ${err.message}\n`);
  process.exit(1);
});
