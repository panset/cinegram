package main

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A manifest with every listing field filled in, so that a test naming one
// field is testing that field rather than the absence of the others.
func fullManifest() *manifest {
	m := &manifest{
		Name:          "cinegram",
		DisplayName:   "Cinegram",
		Description:   "Animated architecture diagrams.",
		Version:       "0.1.0",
		Publisher:     "cinegram",
		Icon:          "icon.png",
		Preview:       true,
		Pricing:       "Free",
		Categories:    []string{"Programming Languages", "Visualization"},
		Keywords:      []string{"diagram", "mermaid"},
		ExtensionKind: []string{"workspace"},
		Homepage:      "https://example.test/readme",
	}
	m.Engines.VSCode = "^1.90.0"
	m.Repository.URL = "https://github.com/panset/cinegram.git"
	m.Bugs.URL = "https://github.com/panset/cinegram/issues"
	m.GalleryBanner.Color = "#12161F"
	m.GalleryBanner.Theme = "dark"
	m.Contributes.Languages = []struct {
		Extensions []string `json:"extensions"`
	}{{Extensions: []string{".dgm"}}}
	return m
}

func everything() map[string]bool {
	return map[string]bool{
		"extension/package.json": true,
		"extension/README.md":    true,
		"extension/CHANGELOG.md": true,
		"extension/LICENSE.txt":  true,
		"extension/icon.png":     true,
	}
}

// TestManifestIsWellFormed is the floor: the gallery rejects a package whose
// manifest does not parse, and a hand-built XML string is exactly the kind of
// thing that stops parsing when someone adds a field.
func TestManifestIsWellFormed(t *testing.T) {
	got := vsixManifest(fullManifest(), "darwin-arm64", everything())
	var doc struct{}
	if err := xml.Unmarshal(got, &doc); err != nil {
		t.Fatalf("manifest is not valid XML: %v\n%s", err, got)
	}
}

// TestManifestCarriesTheListing pins the elements the Marketplace page is built
// from. Every one of them is optional as far as installing goes, which is why
// losing one would otherwise be invisible until the listing was live.
func TestManifestCarriesTheListing(t *testing.T) {
	got := string(vsixManifest(fullManifest(), "darwin-arm64", everything()))

	for _, want := range []string{
		`TargetPlatform="darwin-arm64"`,
		`<Asset Type="Microsoft.VisualStudio.Services.Content.Details" Path="extension/README.md"`,
		`<Asset Type="Microsoft.VisualStudio.Services.Content.Changelog" Path="extension/CHANGELOG.md"`,
		`<Asset Type="Microsoft.VisualStudio.Services.Content.License" Path="extension/LICENSE.txt"`,
		`<Asset Type="Microsoft.VisualStudio.Services.Icons.Default" Path="extension/icon.png"`,
		`<Icon>extension/icon.png</Icon>`,
		`<License>extension/LICENSE.txt</License>`,
		`<Categories>Programming Languages,Visualization</Categories>`,
		`<GalleryFlags>Public Preview</GalleryFlags>`,
		`Id="Microsoft.VisualStudio.Services.Links.Source" Value="https://github.com/panset/cinegram"`,
		`Id="Microsoft.VisualStudio.Services.Links.Support" Value="https://github.com/panset/cinegram/issues"`,
		`Id="Microsoft.VisualStudio.Services.Branding.Color" Value="#12161F"`,
		`Id="Microsoft.VisualStudio.Services.Content.Pricing" Value="Free"`,
		`Id="Microsoft.VisualStudio.Code.ExtensionKind" Value="workspace"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("manifest is missing %s", want)
		}
	}
}

// TestManifestOmitsAbsentAssets is the one that matters most here, because the
// failure it prevents is not a bad listing but a package that will not install:
// an Asset naming a file the ZIP does not contain is rejected outright.
func TestManifestOmitsAbsentAssets(t *testing.T) {
	only := map[string]bool{"extension/package.json": true}
	got := string(vsixManifest(fullManifest(), "", only))

	for _, unwanted := range []string{"README.md", "CHANGELOG.md", "LICENSE.txt", "icon.png"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("manifest names %s, which is not in the package", unwanted)
		}
	}
	if strings.Contains(got, "TargetPlatform") {
		t.Error("a package with no binary should carry no TargetPlatform")
	}
}

func TestTags(t *testing.T) {
	got := strings.Join(tags(fullManifest()), ",")
	// The file extension is how VS Code offers the extension to someone who has
	// just opened a .dgm, which for an unknown language is the only way in.
	for _, want := range []string{"diagram", "mermaid", "Programming Languages", "__ext_dgm"} {
		if !strings.Contains(got, want) {
			t.Errorf("tags %q is missing %q", got, want)
		}
	}
}

func TestSourceURL(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"https://github.com/panset/cinegram.git", "https://github.com/panset/cinegram"},
		{"git+https://github.com/panset/cinegram.git", "https://github.com/panset/cinegram"},
		{"git@github.com:panset/cinegram.git", "https://github.com/panset/cinegram"},
		{"https://github.com/panset/cinegram", "https://github.com/panset/cinegram"},
		{"panset/cinegram", ""},
		{"", ""},
	} {
		if got := sourceURL(c.in); got != c.want {
			t.Errorf("sourceURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestReadManifestRejectsBadIdentity guards the two fields that cannot be fixed
// after the fact: a version published under the wrong identity is spent.
func TestReadManifestRejectsBadIdentity(t *testing.T) {
	for name, body := range map[string]string{
		"no publisher":        `{"name":"cinegram","version":"0.1.0"}`,
		"uppercase name":      `{"name":"Cinegram","version":"0.1.0","publisher":"cinegram"}`,
		"uppercase publisher": `{"name":"cinegram","version":"0.1.0","publisher":"Cinegram"}`,
		"name with a space":   `{"name":"cine gram","version":"0.1.0","publisher":"cinegram"}`,
	} {
		p := filepath.Join(t.TempDir(), "package.json")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := readManifest(p); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestReadManifestAcceptsAStringRepository(t *testing.T) {
	p := filepath.Join(t.TempDir(), "package.json")
	body := `{"name":"cinegram","version":"0.1.0","publisher":"cinegram",
	          "repository":"https://github.com/panset/cinegram.git"}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := readManifest(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := sourceURL(m.Repository.URL); got != "https://github.com/panset/cinegram" {
		t.Errorf("repository = %q", got)
	}
}

func TestQnA(t *testing.T) {
	for _, c := range []struct {
		raw      string
		url      string
		disabled bool
	}{
		{`"https://example.test/q"`, "https://example.test/q", false},
		{`"marketplace"`, "", false},
		{`false`, "", true},
		{``, "", false},
	} {
		m := &manifest{}
		if c.raw != "" {
			m.QnA = []byte(c.raw)
		}
		url, disabled := m.qnaLink()
		if url != c.url || disabled != c.disabled {
			t.Errorf("qna %s = (%q, %v), want (%q, %v)", c.raw, url, disabled, c.url, c.disabled)
		}
	}
}

// TestContentTypesRejectsAnUndeclaredExtension pins the reason that map exists:
// a .vsix holding a file type the manifest does not declare fails on install,
// so guessing a type would trade a build error for a shipped broken package.
func TestContentTypesRejectsAnUndeclaredExtension(t *testing.T) {
	_, err := contentTypesXML([]entry{{archive: "extension/thing.woff2"}})
	if err == nil {
		t.Fatal("expected an error for an undeclared extension")
	}
	if !strings.Contains(err.Error(), "woff2") {
		t.Errorf("error should name the extension: %v", err)
	}
}

// TestPackageContents walks a whole package: what gets in, what stays out, and
// that everything the manifest points at is really there.
func TestPackageContents(t *testing.T) {
	dir := t.TempDir()
	put := func(name, body string) {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put("package.json", `{"name":"cinegram","displayName":"Cinegram","version":"0.1.0",
	  "publisher":"cinegram","icon":"icon.png","categories":["Visualization"],
	  "repository":{"url":"https://github.com/panset/cinegram.git"},
	  "engines":{"vscode":"^1.90.0"}}`)
	put("README.md", "# Cinegram")
	put("CHANGELOG.md", "# Changelog")
	put("LICENSE.txt", "MIT")
	put("icon.png", "\x89PNG")
	put("src/extension.js", "'use strict';")
	// None of these belong in a package: the repository builds the extension,
	// the installer only runs it.
	put("BUILD.bazel", "# bazel")
	put("assets_test.go", "package assets")
	put("sync/sync.go", "package main")

	m, err := readManifest(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	types, err := contentTypesXML(files)
	if err != nil {
		t.Fatal(err)
	}
	present := map[string]bool{}
	for _, e := range files {
		present[e.archive] = true
	}

	out := filepath.Join(t.TempDir(), "test.vsix")
	if err := write(out, files, m, "linux-x64", types, present); err != nil {
		t.Fatal(err)
	}

	z, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()

	got := map[string]bool{}
	for _, f := range z.File {
		got[f.Name] = true
	}

	for _, want := range []string{
		"[Content_Types].xml", "extension.vsixmanifest",
		"extension/package.json", "extension/README.md", "extension/CHANGELOG.md",
		"extension/LICENSE.txt", "extension/icon.png", "extension/src/extension.js",
	} {
		if !got[want] {
			t.Errorf("package is missing %s", want)
		}
	}
	for _, unwanted := range []string{
		"extension/BUILD.bazel", "extension/assets_test.go", "extension/sync/sync.go",
	} {
		if got[unwanted] {
			t.Errorf("package should not contain %s", unwanted)
		}
	}

	// Every Asset the manifest points at has to exist, or install fails.
	manifestXML := read(t, &z.Reader, "extension.vsixmanifest")
	var pkg struct {
		Assets struct {
			Asset []struct {
				Path string `xml:"Path,attr"`
			} `xml:"Asset"`
		} `xml:"Assets"`
	}
	if err := xml.Unmarshal([]byte(manifestXML), &pkg); err != nil {
		t.Fatalf("manifest does not parse: %v", err)
	}
	if len(pkg.Assets.Asset) < 5 {
		t.Errorf("expected the manifest to declare the listing assets, got %d", len(pkg.Assets.Asset))
	}
	for _, a := range pkg.Assets.Asset {
		if !got[a.Path] {
			t.Errorf("manifest points at %s, which is not in the package", a.Path)
		}
	}
}

func read(t *testing.T, z *zip.Reader, name string) string {
	t.Helper()
	f, err := z.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
