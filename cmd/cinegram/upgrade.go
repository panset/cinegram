// The upgrade command: replace this binary with the latest GitHub release.
//
// The release workflow (see RELEASING.md) publishes one statically linked
// binary per platform under a fixed name, cinegram-<os>-<arch>, together with
// a SHA256SUMS file. This file consumes exactly that contract and nothing
// more: the newest version is learned from where /releases/latest redirects,
// so no GitHub API, no JSON, and no credentials are involved.
//
// Everything network-shaped is behind the upgrader struct so tests can point
// it at an httptest server; the pure helpers (asset naming, version
// comparison, checksum parsing) are tested directly.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// releaseBase is where releases live. The path segments appended to it —
// /releases/latest, /releases/download/<tag>/<asset> — are GitHub's, and the
// asset names are the release workflow's; both are contracts documented in
// RELEASING.md.
const releaseBase = "https://github.com/panset/cinegram"

func cmdUpgrade(args []string, stdout, stderr io.Writer) error {
	var check bool
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.BoolVar(&check, "check", false, "report whether a newer release exists; exit 1 when one is")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("upgrade takes no arguments")
	}

	// A Bazel build output is not upgradable in any useful sense: the next
	// `bazel build` would silently put the old version back.
	if os.Getenv("BUILD_WORKSPACE_DIRECTORY") != "" {
		return fmt.Errorf("this binary is a Bazel build output; rebuild it with `bazel build //cmd/cinegram` instead of upgrading in place")
	}

	// The npm and PyPI shims (packaging/) cache one binary per released
	// version and set this when they spawn it. Overwriting that file would
	// make `npx cinegram@0.3.0` run something other than 0.3.0 ever after, so
	// the newer version has to be asked for the same way the old one was.
	if manager := os.Getenv("CINEGRAM_MANAGED_BY"); manager != "" {
		how := map[string]string{"npm": "`npx cinegram@latest …`", "pypi": "`uvx cinegram@latest …`"}[manager]
		if how == "" {
			how = "the same package manager"
		}
		return fmt.Errorf("this binary was fetched by the %s package into a version-pinned cache; upgrading in place would overwrite that pin — ask %s for the newer version instead", manager, how)
	}

	u := &upgrader{
		base:    releaseBase,
		client:  &http.Client{Timeout: 5 * time.Minute},
		current: version,
		exePath: os.Executable,
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
	}
	return u.run(check, stdout, stderr)
}

// upgrader carries every seam the command needs faked in tests.
type upgrader struct {
	base    string
	client  *http.Client
	current string
	exePath func() (string, error)
	goos    string
	goarch  string
}

func (u *upgrader) run(check bool, stdout, stderr io.Writer) error {
	latest, err := u.latestVersion()
	if err != nil {
		return err
	}

	newer, err := versionLess(u.current, latest)
	if err != nil {
		return err
	}
	if !newer {
		fmt.Fprintf(stdout, "cinegram %s is up to date (latest release is %s)\n", u.current, latest)
		return nil
	}
	if check {
		fmt.Fprintf(stdout, "cinegram %s → %s available; run `cinegram upgrade`\n", u.current, latest)
		return fmt.Errorf("a newer release is available")
	}

	asset, err := releaseAsset(u.goos, u.goarch)
	if err != nil {
		return err
	}

	fmt.Fprintf(stderr, "downloading %s %s…\n", asset, "v"+latest)
	binary, err := u.download(latest, asset)
	if err != nil {
		return err
	}

	path, err := replaceExecutable(u.exePath, binary)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "upgraded %s: %s → %s\n", path, u.current, latest)
	return nil
}

// latestVersion asks where /releases/latest lands. GitHub answers the request
// with a redirect to /releases/tag/<tag>, so the final URL alone names the
// newest release — no API call, no rate-limited JSON.
func (u *upgrader) latestVersion() (string, error) {
	resp, err := u.client.Get(u.base + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checking %s/releases/latest: HTTP %d (no releases yet?)", u.base, resp.StatusCode)
	}
	return versionFromReleaseURL(resp.Request.URL.Path)
}

// versionFromReleaseURL pulls "0.2.0" out of ".../releases/tag/v0.2.0".
func versionFromReleaseURL(path string) (string, error) {
	_, tag, ok := strings.Cut(path, "/releases/tag/")
	if !ok || !strings.HasPrefix(tag, "v") {
		return "", fmt.Errorf("the latest-release URL %q does not name a v* tag", path)
	}
	return strings.TrimPrefix(tag, "v"), nil
}

// download fetches the named asset from the versioned URL — not from
// /latest/, which could move mid-flight — and verifies it against the
// release's SHA256SUMS before anything touches disk.
func (u *upgrader) download(version, asset string) ([]byte, error) {
	dir := u.base + "/releases/download/v" + version + "/"

	sums, err := u.fetch(dir + "SHA256SUMS")
	if err != nil {
		return nil, err
	}
	want, err := checksumFor(sums, asset)
	if err != nil {
		return nil, err
	}

	binary, err := u.fetch(dir + asset)
	if err != nil {
		return nil, err
	}
	got := sha256.Sum256(binary)
	if hex.EncodeToString(got[:]) != want {
		return nil, fmt.Errorf("%s does not match the release's SHA256SUMS; not installing it", asset)
	}
	return binary, nil
}

func (u *upgrader) fetch(url string) ([]byte, error) {
	resp, err := u.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// releaseAsset maps this build's platform onto the release asset names, which
// follow VS Code's platform vocabulary (win32, x64) because the extension
// packages came first.
func releaseAsset(goos, goarch string) (string, error) {
	oses := map[string]string{"darwin": "darwin", "linux": "linux", "windows": "win32"}
	arches := map[string]string{"amd64": "x64", "arm64": "arm64"}
	o, a := oses[goos], arches[goarch]
	if o == "" || a == "" {
		return "", fmt.Errorf("no release is published for %s/%s; build from source with `bazel build //cmd/cinegram`", goos, goarch)
	}
	name := "cinegram-" + o + "-" + a
	if goos == "windows" {
		name += ".exe"
	}
	return name, nil
}

// checksumFor finds one file's hash in sha256sum output ("<hex>  <name>").
func checksumFor(sums []byte, asset string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == asset {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("SHA256SUMS has no entry for %s", asset)
}

// versionLess reports whether b is a newer release than a. Both must be
// dotted integers; anything else is an error rather than a guess, because the
// answer decides whether to overwrite a working binary.
func versionLess(a, b string) (bool, error) {
	av, err := parseVersion(a)
	if err != nil {
		return false, err
	}
	bv, err := parseVersion(b)
	if err != nil {
		return false, err
	}
	for i := range av {
		if av[i] != bv[i] {
			return av[i] < bv[i], nil
		}
	}
	return false, nil
}

func parseVersion(s string) ([3]int, error) {
	var v [3]int
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, fmt.Errorf("version %q is not MAJOR.MINOR.PATCH", s)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, fmt.Errorf("version %q is not MAJOR.MINOR.PATCH", s)
		}
		v[i] = n
	}
	return v, nil
}

// replaceExecutable swaps the running binary for the downloaded bytes.
//
// The new file is written beside the old one — same directory, so the final
// rename never crosses a filesystem — and moved into place atomically. On
// Windows a running executable cannot be overwritten, but it can be renamed,
// so the old binary steps aside as cinegram.exe.old first; the leftover is
// deleted by the next upgrade.
func replaceExecutable(exePath func() (string, error), binary []byte) (string, error) {
	exe, err := exePath()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	tmp, err := os.CreateTemp(filepath.Dir(exe), ".cinegram-upgrade-*")
	if err != nil {
		return "", permissionHint(err, exe)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(binary); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		old := exe + ".old"
		os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return "", permissionHint(err, exe)
		}
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		return "", permissionHint(err, exe)
	}
	return exe, nil
}

// permissionHint turns a bare EACCES into instructions, because "permission
// denied" alone reads as a bug when the fix is sudo or a different install
// location.
func permissionHint(err error, exe string) error {
	if os.IsPermission(err) {
		return fmt.Errorf("%w\ncannot write %s: re-run with elevated permissions, or install to a user-writable location such as ~/.cinegram/cinegram", err, filepath.Dir(exe))
	}
	return err
}
