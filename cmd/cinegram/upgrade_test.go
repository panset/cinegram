package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseAsset(t *testing.T) {
	cases := []struct {
		goos, goarch string
		want         string
		wantErr      bool
	}{
		{"darwin", "arm64", "cinegram-darwin-arm64", false},
		{"darwin", "amd64", "cinegram-darwin-x64", false},
		{"linux", "amd64", "cinegram-linux-x64", false},
		{"linux", "arm64", "cinegram-linux-arm64", false},
		{"windows", "amd64", "cinegram-win32-x64.exe", false},
		{"plan9", "amd64", "", true},
		{"linux", "riscv64", "", true},
	}
	for _, c := range cases {
		got, err := releaseAsset(c.goos, c.goarch)
		if c.wantErr != (err != nil) {
			t.Errorf("releaseAsset(%s, %s): err = %v, wantErr %v", c.goos, c.goarch, err, c.wantErr)
			continue
		}
		if got != c.want {
			t.Errorf("releaseAsset(%s, %s) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestVersionLess(t *testing.T) {
	cases := []struct {
		a, b    string
		want    bool
		wantErr bool
	}{
		{"0.1.0", "0.2.0", true, false},
		{"0.1.0", "0.1.0", false, false},
		{"0.2.0", "0.1.9", false, false},
		{"0.9.9", "1.0.0", true, false},
		{"0.1.9", "0.1.10", true, false},
		{"0.1.0", "0.2", false, true},
		{"0.1.0", "0.2.0-rc1", false, true},
		{"dev", "0.2.0", false, true},
	}
	for _, c := range cases {
		got, err := versionLess(c.a, c.b)
		if c.wantErr != (err != nil) {
			t.Errorf("versionLess(%q, %q): err = %v, wantErr %v", c.a, c.b, err, c.wantErr)
			continue
		}
		if got != c.want {
			t.Errorf("versionLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestVersionFromReleaseURL(t *testing.T) {
	if v, err := versionFromReleaseURL("/panset/cinegram/releases/tag/v0.3.1"); err != nil || v != "0.3.1" {
		t.Errorf("got %q, %v", v, err)
	}
	if _, err := versionFromReleaseURL("/panset/cinegram/releases"); err == nil {
		t.Error("a URL with no tag should be an error")
	}
	if _, err := versionFromReleaseURL("/panset/cinegram/releases/tag/nightly"); err == nil {
		t.Error("a tag without the v prefix should be an error")
	}
}

func TestChecksumFor(t *testing.T) {
	sums := []byte("aaaa  cinegram-linux-x64\nbbbb  cinegram-darwin-arm64\n")
	if got, err := checksumFor(sums, "cinegram-darwin-arm64"); err != nil || got != "bbbb" {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := checksumFor(sums, "cinegram-win32-x64.exe"); err == nil {
		t.Error("a missing asset should be an error")
	}
}

// releaseServer fakes just enough of GitHub: the /releases/latest redirect,
// the SHA256SUMS file, and one binary asset.
func releaseServer(t *testing.T, tag string, assets map[string][]byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/releases/tag/"+tag, http.StatusFound)
	})
	mux.HandleFunc("/releases/tag/"+tag, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "release page")
	})
	var sums strings.Builder
	for name, body := range assets {
		h := sha256.Sum256(body)
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(h[:]), name)
		body := body
		mux.HandleFunc("/releases/download/"+tag+"/"+name, func(w http.ResponseWriter, r *http.Request) {
			w.Write(body)
		})
	}
	mux.HandleFunc("/releases/download/"+tag+"/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, sums.String())
	})
	return httptest.NewServer(mux)
}

func testUpgrader(t *testing.T, srv *httptest.Server, current, exe string) *upgrader {
	t.Helper()
	return &upgrader{
		base:    srv.URL,
		client:  srv.Client(),
		current: current,
		exePath: func() (string, error) { return exe, nil },
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
	}
}

func TestUpgradeReplacesTheBinary(t *testing.T) {
	asset, err := releaseAsset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("no release asset for this host: %v", err)
	}
	newBinary := []byte("the new binary")
	srv := releaseServer(t, "v0.2.0", map[string][]byte{asset: newBinary})
	defer srv.Close()

	exe := filepath.Join(t.TempDir(), "cinegram")
	if err := os.WriteFile(exe, []byte("the old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr strings.Builder
	u := testUpgrader(t, srv, "0.1.0", exe)
	if err := u.run(false, &stdout, &stderr); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBinary) {
		t.Errorf("binary content = %q, want %q", got, newBinary)
	}
	if !strings.Contains(stdout.String(), "0.1.0 → 0.2.0") {
		t.Errorf("stdout should name both versions, got %q", stdout.String())
	}
}

func TestUpgradeRefusesACorruptDownload(t *testing.T) {
	asset, err := releaseAsset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("no release asset for this host: %v", err)
	}
	exe := filepath.Join(t.TempDir(), "cinegram")
	if err := os.WriteFile(exe, []byte("the old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// A server whose SHA256SUMS disagrees with the asset it serves.
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/releases/tag/v0.2.0", http.StatusFound)
	})
	mux.HandleFunc("/releases/tag/v0.2.0", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "release page")
	})
	mux.HandleFunc("/releases/download/v0.2.0/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%064d  %s\n", 0, asset)
	})
	mux.HandleFunc("/releases/download/v0.2.0/"+asset, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("the new binary"))
	})
	bad := httptest.NewServer(mux)
	defer bad.Close()
	u := testUpgrader(t, bad, "0.1.0", exe)

	var stdout, stderr strings.Builder
	if err := u.run(false, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "SHA256SUMS") {
		t.Fatalf("a checksum mismatch must refuse to install; got err = %v", err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "the old binary" {
		t.Errorf("the binary must be untouched after a refused upgrade, got %q", got)
	}
}

func TestUpgradeCheck(t *testing.T) {
	asset, err := releaseAsset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("no release asset for this host: %v", err)
	}
	srv := releaseServer(t, "v0.2.0", map[string][]byte{asset: []byte("bin")})
	defer srv.Close()

	exe := filepath.Join(t.TempDir(), "cinegram")

	// Stale: reports the newer version and fails, without touching anything.
	var stdout, stderr strings.Builder
	u := testUpgrader(t, srv, "0.1.0", exe)
	if err := u.run(true, &stdout, &stderr); err == nil {
		t.Error("--check must exit non-zero when a newer release exists")
	}
	if !strings.Contains(stdout.String(), "0.2.0") {
		t.Errorf("--check should name the newer version, got %q", stdout.String())
	}

	// Current: succeeds quietly.
	stdout.Reset()
	u = testUpgrader(t, srv, "0.2.0", exe)
	if err := u.run(true, &stdout, &stderr); err != nil {
		t.Errorf("--check on an up-to-date binary: %v", err)
	}
	if !strings.Contains(stdout.String(), "up to date") {
		t.Errorf("got %q", stdout.String())
	}
}
