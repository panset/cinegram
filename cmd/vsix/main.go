// Command vsix packages the VS Code extension.
//
// A .vsix is a ZIP with three things in it: the extension under `extension/`,
// an `extension.vsixmanifest` describing it, and a `[Content_Types].xml` naming
// the MIME type of every file extension present. That is the whole format, and
// archive/zip is in the standard library — so packaging costs one Go file
// rather than a Node toolchain, an npm lockfile and `vsce`, none of which this
// repository would otherwise need.
//
// Platform-specific packages exist because the extension ships the compiler it
// shells out to. Each package carries exactly one binary, and VS Code installs
// the one matching the machine:
//
//	bazel build //cmd/cinegram:cinegram_darwin-arm64
//	bazel run  //cmd/vsix -- \
//	  --target darwin-arm64 \
//	  --binary "$(bazel cquery --output=files //cmd/cinegram:cinegram_darwin-arm64)" \
//	  --out dist/cinegram-darwin-arm64.vsix
package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// targets are the platform names VS Code uses in a package's TargetPlatform,
// mapped to where the extension expects to find its binary. binary.js reads the
// same layout from the other side.
var targets = map[string]bool{
	"darwin-arm64": true,
	"darwin-x64":   true,
	"linux-arm64":  true,
	"linux-x64":    true,
	"win32-x64":    true,
}

// contentTypes maps a file extension to the MIME type the manifest must declare.
// A .vsix with an undeclared extension in it is rejected on install, so an
// unknown one is an error rather than a guess.
var contentTypes = map[string]string{
	".json": "application/json",
	".js":   "application/javascript",
	".css":  "text/css",
	".md":   "text/markdown",
	".png":  "image/png",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".txt":  "text/plain",
	"":      "application/octet-stream", // the binary, which has no extension
	".exe":  "application/octet-stream",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "vsix:", err)
		os.Exit(1)
	}
}

func run() error {
	var target, binary, out, dir string
	flag.StringVar(&target, "target", "", "platform this package is for, e.g. darwin-arm64 (omit for a package with no binary)")
	flag.StringVar(&binary, "binary", "", "path to the cinegram binary to bundle")
	flag.StringVar(&out, "out", "", "where to write the .vsix")
	flag.StringVar(&dir, "extension-dir", "", "the extension source (default: editors/vscode in the workspace)")
	flag.Parse()

	if out == "" {
		return fmt.Errorf("--out is required")
	}
	if target != "" && !targets[target] {
		return fmt.Errorf("unknown --target %q: use one of %s", target, strings.Join(sortedKeys(targets), ", "))
	}
	if (target == "") != (binary == "") {
		return fmt.Errorf("--target and --binary go together: a platform package is one with that platform's binary in it")
	}

	root := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if dir == "" {
		if root == "" {
			return fmt.Errorf("--extension-dir is required outside `bazel run`")
		}
		dir = filepath.Join(root, "editors", "vscode")
	}
	out = resolve(out, root)
	binary = resolve(binary, root)

	manifest, err := readManifest(filepath.Join(dir, "package.json"))
	if err != nil {
		return err
	}

	files, err := collect(dir)
	if err != nil {
		return err
	}
	if binary != "" {
		files = append(files, entry{
			archive: path.Join("extension", "bin", target, binaryName(target)),
			source:  binary,
			mode:    0o755,
		})
	}

	// Built before anything is created, so a file type nobody declared is an
	// error rather than a half-written archive left on disk.
	types, err := contentTypesXML(files)
	if err != nil {
		return err
	}

	present := map[string]bool{}
	for _, e := range files {
		present[e.archive] = true
	}
	warnListing(manifest, present)

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := write(out, files, manifest, target, types, present); err != nil {
		return err
	}

	info, err := os.Stat(out)
	if err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d files, %.1f MB)\n", out, len(files)+2, float64(info.Size())/(1<<20))
	return nil
}

// entry is one file on its way into the package.
type entry struct {
	archive string // path inside the zip
	source  string // path on disk
	mode    fs.FileMode
}

// manifest is the package.json fields the vsixmanifest repeats.
//
// The first block is the package's identity, which install needs. The rest is
// the Marketplace listing — the icon, the categories, the links down the side
// of the page. None of it changes what the extension does, and all of it is
// the difference between a listing and a listing nobody installs from.
type manifest struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Publisher   string `json:"publisher"`
	Engines     struct {
		VSCode string `json:"vscode"`
	} `json:"engines"`

	Icon          string   `json:"icon"`
	Preview       bool     `json:"preview"`
	Pricing       string   `json:"pricing"`
	Categories    []string `json:"categories"`
	Keywords      []string `json:"keywords"`
	ExtensionKind []string `json:"extensionKind"`
	Homepage      string   `json:"homepage"`
	GalleryBanner struct {
		Color string `json:"color"`
		Theme string `json:"theme"`
	} `json:"galleryBanner"`
	Repository repository `json:"repository"`
	Bugs       struct {
		URL string `json:"url"`
	} `json:"bugs"`
	QnA         json.RawMessage `json:"qna"`
	Contributes struct {
		Languages []struct {
			Extensions []string `json:"extensions"`
		} `json:"languages"`
	} `json:"contributes"`
}

// repository is `{"url": …}` in this package.json, but npm also allows the bare
// string form, and a manifest that fails to parse over that would be a puzzling
// way to find out.
type repository struct {
	URL string
}

func (r *repository) UnmarshalJSON(data []byte) error {
	var s string
	if json.Unmarshal(data, &s) == nil {
		r.URL = s
		return nil
	}
	var obj struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	r.URL = obj.URL
	return nil
}

// nameRE is what the Marketplace accepts as an extension name. Catching it here
// costs nothing; finding out at publish time costs a version number, because a
// version once published cannot be reused.
var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*$`)

func readManifest(p string) (*manifest, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading the extension manifest: %w", err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	for _, field := range []struct{ name, value string }{
		{"name", m.Name},
		{"version", m.Version},
		{"publisher", m.Publisher},
	} {
		if field.value == "" {
			return nil, fmt.Errorf("%s is missing %q, which the package identity needs", p, field.name)
		}
	}
	if !nameRE.MatchString(m.Name) {
		return nil, fmt.Errorf("%s has name %q; the Marketplace allows only lowercase letters, digits and dashes", p, m.Name)
	}
	if !nameRE.MatchString(m.Publisher) {
		return nil, fmt.Errorf("%s has publisher %q; the Marketplace allows only lowercase letters, digits and dashes", p, m.Publisher)
	}
	return &m, nil
}

// listingAssets are the four files the Marketplace renders a listing from. Each
// is optional in the format and each is a visible hole in the page when it is
// missing, so an absent one is a warning rather than an error — packaging a
// build to try locally should not require a changelog.
var listingAssets = []struct{ assetType, file, why string }{
	{"Microsoft.VisualStudio.Services.Content.Details", "README.md", "the Marketplace page will have no body"},
	{"Microsoft.VisualStudio.Services.Content.Changelog", "CHANGELOG.md", "the Changelog tab will be empty"},
	{"Microsoft.VisualStudio.Services.Content.License", "LICENSE.txt", "the License tab will be empty"},
}

// qnaLink reads the `qna` field, which npm allows to be a URL, the string
// "marketplace", or false. Only the first two produce anything here.
func (m *manifest) qnaLink() (url string, disabled bool) {
	if len(m.QnA) == 0 {
		return "", false
	}
	var b bool
	if json.Unmarshal(m.QnA, &b) == nil {
		return "", !b
	}
	var s string
	if json.Unmarshal(m.QnA, &s) == nil && s != "marketplace" {
		return s, false
	}
	return "", false
}

// sourceURL normalises whatever `repository.url` holds into the https URL the
// Marketplace links to: npm accepts `git+https://…`, `git@host:owner/repo.git`
// and a bare path, and the gallery renders the link verbatim.
func sourceURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "git+")
	if rest, ok := strings.CutPrefix(s, "git@"); ok {
		s = "https://" + strings.Replace(rest, ":", "/", 1)
	}
	s = strings.TrimSuffix(s, ".git")
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return ""
	}
	return s
}

// skip lists what never belongs in a package.
//
// `bin` is excluded because its contents are chosen per target rather than
// copied from a working tree that may hold several. The Bazel and Go files are
// excluded because the extension is shipped, not built: the only Go here is the
// asset-drift test and the tool that fixes it, both of which belong to this
// repository rather than to anyone installing the result.
func skip(name string) bool {
	switch name {
	case "node_modules", ".git", ".DS_Store", "bin", "sync", "BUILD.bazel":
		return true
	}
	return strings.HasPrefix(name, ".vscode") || strings.HasSuffix(name, ".go")
}

func collect(dir string) ([]entry, error) {
	var out []entry
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, entry{
			archive: path.Join("extension", filepath.ToSlash(rel)),
			source:  p,
			mode:    info.Mode().Perm(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Deterministic order, so the same inputs produce the same archive.
	sort.Slice(out, func(i, j int) bool { return out[i].archive < out[j].archive })
	return out, nil
}

// warnListing reports what the Marketplace page will be missing.
//
// None of this stops a package installing, which is exactly why it is worth
// saying out loud: a .vsix with no icon and no README installs perfectly and
// lists as a blank grey square with no description under it.
func warnListing(m *manifest, present map[string]bool) {
	var problems []string
	for _, a := range listingAssets {
		if !present[path.Join("extension", a.file)] {
			problems = append(problems, fmt.Sprintf("no %s in the extension: %s", a.file, a.why))
		}
	}
	switch {
	case m.Icon == "":
		problems = append(problems, `no "icon" in package.json: the listing will show a blank tile`)
	case !present[path.Join("extension", m.Icon)]:
		problems = append(problems, fmt.Sprintf(
			"%s is named by \"icon\" in package.json but is not in the package: the listing will show a blank tile", m.Icon))
	}
	if len(m.Categories) == 0 {
		problems = append(problems, `no "categories" in package.json: the listing will not be filed under anything`)
	}
	if sourceURL(m.Repository.URL) == "" {
		problems = append(problems, `no usable "repository" in package.json: the listing will have no Repository link`)
	}
	for _, s := range problems {
		fmt.Fprintf(os.Stderr, "vsix: warning: %s\n", s)
	}
}

func write(out string, files []entry, m *manifest, target string, types []byte, present map[string]bool) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	z := zip.NewWriter(f)

	if err := add(z, "[Content_Types].xml", types, 0o644); err != nil {
		return err
	}
	if err := add(z, "extension.vsixmanifest", vsixManifest(m, target, present), 0o644); err != nil {
		return err
	}

	for _, e := range files {
		data, err := os.ReadFile(e.source)
		if err != nil {
			return err
		}
		if err := add(z, e.archive, data, e.mode); err != nil {
			return err
		}
	}
	return z.Close()
}

func add(z *zip.Writer, name string, data []byte, mode fs.FileMode) error {
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	h.SetMode(mode)
	w, err := z.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, strings.NewReader(string(data)))
	return err
}

func contentTypesXML(files []entry) ([]byte, error) {
	seen := map[string]bool{}
	for _, e := range files {
		seen[strings.ToLower(path.Ext(e.archive))] = true
	}
	seen[".xml"] = true
	seen[".vsixmanifest"] = true

	var defaults []string
	for _, ext := range sortedKeys(seen) {
		ct, ok := contentTypes[ext]
		switch {
		case ext == ".xml":
			ct = "text/xml"
		case ext == ".vsixmanifest":
			ct = "text/xml"
		case !ok:
			return nil, fmt.Errorf("no content type declared for %q files; add one to contentTypes in cmd/vsix", ext)
		}
		// An extensionless file is declared as the empty Extension, which the
		// format allows and which is how the binary gets through.
		defaults = append(defaults, fmt.Sprintf(`  <Default Extension="%s" ContentType="%s"/>`,
			strings.TrimPrefix(ext, "."), ct))
	}

	return []byte(xml.Header +
		"<Types xmlns=\"http://schemas.openxmlformats.org/package/2006/content-types\">\n" +
		strings.Join(defaults, "\n") +
		"\n</Types>\n"), nil
}

// vsixManifest renders extension.vsixmanifest.
//
// `present` is the set of archive paths actually going into the package, and it
// gates every Asset element: an Asset naming a file the ZIP does not contain is
// rejected on install, which turns a missing changelog into a broken package
// rather than a missing tab.
func vsixManifest(m *manifest, target string, present map[string]bool) []byte {
	platform := ""
	if target != "" {
		platform = fmt.Sprintf(" TargetPlatform=%q", target)
	}
	display := m.DisplayName
	if display == "" {
		display = m.Name
	}
	flags := "Public"
	if m.Preview {
		flags = "Public Preview"
	}

	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString(`<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011" xmlns:d="http://schemas.microsoft.com/developer/vsx-schema-design/2011">` + "\n")
	b.WriteString("  <Metadata>\n")
	fmt.Fprintf(&b, "    <Identity Language=\"en-US\" Id=\"%s\" Version=\"%s\" Publisher=\"%s\"%s/>\n",
		esc(m.Name), esc(m.Version), esc(m.Publisher), platform)
	fmt.Fprintf(&b, "    <DisplayName>%s</DisplayName>\n", esc(display))
	fmt.Fprintf(&b, "    <Description xml:space=\"preserve\">%s</Description>\n", esc(m.Description))
	fmt.Fprintf(&b, "    <Tags>%s</Tags>\n", esc(strings.Join(tags(m), ",")))
	fmt.Fprintf(&b, "    <Categories>%s</Categories>\n", esc(strings.Join(m.Categories, ",")))
	fmt.Fprintf(&b, "    <GalleryFlags>%s</GalleryFlags>\n", flags)
	if m.Icon != "" && present[path.Join("extension", m.Icon)] {
		fmt.Fprintf(&b, "    <Icon>extension/%s</Icon>\n", esc(filepath.ToSlash(m.Icon)))
	}
	if present["extension/LICENSE.txt"] {
		b.WriteString("    <License>extension/LICENSE.txt</License>\n")
	}

	b.WriteString("    <Properties>\n")
	for _, p := range properties(m) {
		fmt.Fprintf(&b, "      <Property Id=\"%s\" Value=\"%s\"/>\n", p[0], esc(p[1]))
	}
	b.WriteString("    </Properties>\n")
	b.WriteString("  </Metadata>\n")
	b.WriteString("  <Installation>\n    <InstallationTarget Id=\"Microsoft.VisualStudio.Code\"/>\n  </Installation>\n")
	b.WriteString("  <Dependencies/>\n")

	b.WriteString("  <Assets>\n")
	b.WriteString("    <Asset Type=\"Microsoft.VisualStudio.Code.Manifest\" Path=\"extension/package.json\" Addressable=\"true\"/>\n")
	for _, a := range listingAssets {
		if present[path.Join("extension", a.file)] {
			fmt.Fprintf(&b, "    <Asset Type=\"%s\" Path=\"extension/%s\" Addressable=\"true\"/>\n", a.assetType, a.file)
		}
	}
	if m.Icon != "" && present[path.Join("extension", m.Icon)] {
		fmt.Fprintf(&b, "    <Asset Type=\"Microsoft.VisualStudio.Services.Icons.Default\" Path=\"extension/%s\" Addressable=\"true\"/>\n",
			esc(filepath.ToSlash(m.Icon)))
	}
	b.WriteString("  </Assets>\n")
	b.WriteString("</PackageManifest>\n")
	return []byte(b.String())
}

// tags are what the Marketplace searches on. The keywords and categories are the
// obvious half; the `__ext_` entries are the other one — they are how VS Code
// offers an extension to someone who has just opened a file it handles, which
// for a language nobody has heard of is the only discovery path there is.
func tags(m *manifest) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" && !seen[strings.ToLower(s)] {
			seen[strings.ToLower(s)] = true
			out = append(out, s)
		}
	}
	for _, k := range m.Keywords {
		add(k)
	}
	for _, c := range m.Categories {
		add(c)
	}
	for _, lang := range m.Contributes.Languages {
		for _, ext := range lang.Extensions {
			add("__ext_" + strings.TrimPrefix(ext, "."))
		}
	}
	return out
}

// properties are the Metadata/Properties entries, in the order vsce writes them.
func properties(m *manifest) [][2]string {
	kind := "workspace"
	if len(m.ExtensionKind) > 0 {
		kind = strings.Join(m.ExtensionKind, ",")
	}
	out := [][2]string{
		{"Microsoft.VisualStudio.Code.Engine", m.Engines.VSCode},
		{"Microsoft.VisualStudio.Code.ExtensionDependencies", ""},
		{"Microsoft.VisualStudio.Code.ExtensionPack", ""},
		{"Microsoft.VisualStudio.Code.ExtensionKind", kind},
		{"Microsoft.VisualStudio.Code.LocalizedLanguages", ""},
	}

	source := sourceURL(m.Repository.URL)
	learn := m.Homepage
	if learn == "" {
		learn = source
	}
	for _, link := range [][2]string{
		{"Microsoft.VisualStudio.Services.Links.Source", source},
		{"Microsoft.VisualStudio.Services.Links.Getstarted", source},
		{"Microsoft.VisualStudio.Services.Links.GitHub", githubURL(source)},
		{"Microsoft.VisualStudio.Services.Links.Support", m.Bugs.URL},
		{"Microsoft.VisualStudio.Services.Links.Learn", learn},
	} {
		if link[1] != "" {
			out = append(out, link)
		}
	}

	if m.GalleryBanner.Color != "" {
		theme := m.GalleryBanner.Theme
		if theme == "" {
			theme = "dark"
		}
		out = append(out,
			[2]string{"Microsoft.VisualStudio.Services.Branding.Color", m.GalleryBanner.Color},
			[2]string{"Microsoft.VisualStudio.Services.Branding.Theme", theme},
		)
	}

	// The README is written for GitHub and rendered by the gallery; without this
	// the gallery falls back to a stricter dialect and the tables come out flat.
	out = append(out, [2]string{"Microsoft.VisualStudio.Services.GitHubFlavoredMarkdown", "true"})

	if m.Pricing != "" {
		out = append(out, [2]string{"Microsoft.VisualStudio.Services.Content.Pricing", m.Pricing})
	}
	if url, disabled := m.qnaLink(); disabled {
		out = append(out, [2]string{"Microsoft.VisualStudio.Code.EnableMarketplaceQnA", "false"})
	} else if url != "" {
		out = append(out, [2]string{"Microsoft.VisualStudio.Services.CustomerQnALink", url})
	}
	return out
}

// githubURL is the Source link again, but only when it really is GitHub — the
// gallery renders this one with a GitHub mark beside it.
func githubURL(source string) string {
	if strings.HasPrefix(source, "https://github.com/") {
		return source
	}
	return ""
}

func esc(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

func binaryName(target string) string {
	if strings.HasPrefix(target, "win32") {
		return "cinegram.exe"
	}
	return "cinegram"
}

// resolve interprets a relative path against the directory the user ran from,
// which under `bazel run` is not the working directory of the process.
func resolve(p, root string) string {
	if p == "" || filepath.IsAbs(p) || root == "" {
		return p
	}
	return filepath.Join(root, p)
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
