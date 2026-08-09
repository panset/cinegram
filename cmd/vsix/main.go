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

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := write(out, files, manifest, target, types); err != nil {
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

// manifest is the handful of package.json fields the vsixmanifest repeats.
type manifest struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Publisher   string `json:"publisher"`
	Engines     struct {
		VSCode string `json:"vscode"`
	} `json:"engines"`
}

func readManifest(p string) (*manifest, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading the extension manifest: %w", err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	for field, value := range map[string]string{"name": m.Name, "version": m.Version, "publisher": m.Publisher} {
		if value == "" {
			return nil, fmt.Errorf("%s is missing %q, which the package identity needs", p, field)
		}
	}
	return &m, nil
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

func write(out string, files []entry, m *manifest, target string, types []byte) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	z := zip.NewWriter(f)

	if err := add(z, "[Content_Types].xml", types, 0o644); err != nil {
		return err
	}
	if err := add(z, "extension.vsixmanifest", vsixManifest(m, target), 0o644); err != nil {
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

func vsixManifest(m *manifest, target string) []byte {
	platform := ""
	if target != "" {
		platform = fmt.Sprintf(" TargetPlatform=%q", target)
	}
	display := m.DisplayName
	if display == "" {
		display = m.Name
	}

	return []byte(xml.Header + fmt.Sprintf(`<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011" xmlns:d="http://schemas.microsoft.com/developer/vsx-schema-design/2011">
  <Metadata>
    <Identity Language="en-US" Id="%s" Version="%s" Publisher="%s"%s/>
    <DisplayName>%s</DisplayName>
    <Description xml:space="preserve">%s</Description>
    <Tags></Tags>
    <GalleryFlags>Public</GalleryFlags>
    <Properties>
      <Property Id="Microsoft.VisualStudio.Code.Engine" Value="%s"/>
      <Property Id="Microsoft.VisualStudio.Code.ExtensionDependencies" Value=""/>
      <Property Id="Microsoft.VisualStudio.Code.ExtensionPack" Value=""/>
      <Property Id="Microsoft.VisualStudio.Code.ExtensionKind" Value="workspace"/>
    </Properties>
  </Metadata>
  <Installation>
    <InstallationTarget Id="Microsoft.VisualStudio.Code"/>
  </Installation>
  <Dependencies/>
  <Assets>
    <Asset Type="Microsoft.VisualStudio.Code.Manifest" Path="extension/package.json" Addressable="true"/>
  </Assets>
</PackageManifest>
`, esc(m.Name), esc(m.Version), esc(m.Publisher), platform, esc(display), esc(m.Description), esc(m.Engines.VSCode)))
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
