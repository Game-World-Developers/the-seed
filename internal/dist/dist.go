// Package dist builds a distributable asset bundle: a manifest describing
// every declared Asset (from internal/ir, so it only ever sees resolved,
// validated data) plus a zip archive containing the referenced files.
//
// This is the part of Phase 8's "platform-aware asset cooking, packaging,
// compression, and manifests" achievable without a target-specific
// toolchain: it is genuinely platform-*aware* (the manifest records which
// platform label it was built for, and docs/platforms.md documents how a
// future per-platform cooking step — texture compression, format
// conversion — would plug in here) but does not yet transform assets
// per-target, since no target-specific asset transform exists yet to
// perform. Compression here means "the bundle is a zip archive," not
// "each asset was recompressed for its target" — see the package doc
// comment on Bundle for the honest boundary.
package dist

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"Game-Developers-World/seed/internal/ir"

	"gopkg.in/yaml.v3"
)

// ManifestVersion follows the same bump rule as internal/ir.Version and
// internal/trace.Version: patch for docs-only changes, minor for
// additive fields, major for breaking changes.
const ManifestVersion = "0.1.0"

// AppMetadata is a project's distributable identity: what a packaged
// bundle calls itself, which version it is, and the reverse-DNS-style
// identifier platform stores/installers expect. Icons and code-signing
// are deliberately not part of this type — see LoadAppMetadata's doc
// comment for why they're an explicit, undone gap rather than stubbed
// fields nothing populates.
type AppMetadata struct {
	Name       string `json:"name" yaml:"name"`
	Version    string `json:"version" yaml:"version"`
	Identifier string `json:"identifier" yaml:"identifier"`
}

// LoadAppMetadata reads <root>/app.yaml if present, filling in defaults
// for any field it doesn't set (or if the file doesn't exist at all — a
// fresh project has no reason to require one up front).
//
// Icons and signing are not handled anywhere in this package: an icon
// needs actual image content this tool has no way to generate, and
// signing needs a platform-specific credential (an Apple/Windows
// certificate) that must never be embedded in a Seed project — exactly
// what this roadmap item warns against. Both are real, undone gaps;
// AppMetadata intentionally has no IconPath/SigningIdentity field to
// avoid implying either is wired up.
func LoadAppMetadata(root string) (AppMetadata, error) {
	meta := AppMetadata{
		Name:       filepath.Base(mustAbs(root)),
		Version:    "0.1.0",
		Identifier: "",
	}

	data, err := os.ReadFile(filepath.Join(root, "app.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			if meta.Identifier == "" {
				meta.Identifier = "com.example." + sanitizeIdentifier(meta.Name)
			}
			return meta, nil
		}
		return AppMetadata{}, fmt.Errorf("reading app.yaml: %w", err)
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return AppMetadata{}, fmt.Errorf("parsing app.yaml: %w", err)
	}
	if meta.Version == "" {
		meta.Version = "0.1.0"
	}
	if meta.Identifier == "" {
		meta.Identifier = "com.example." + sanitizeIdentifier(meta.Name)
	}
	return meta, nil
}

func sanitizeIdentifier(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, r)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// AssetEntry is one packaged asset's manifest record.
type AssetEntry struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Manifest describes one packaged bundle.
type Manifest struct {
	ManifestVersion string       `json:"manifest_version"`
	App             AppMetadata  `json:"app"`
	Platform        string       `json:"platform"`
	Assets          []AssetEntry `json:"assets"`
}

// BuildManifest reads every Asset in model (already resolved by
// ir.Build — never raw YAML) and stats+hashes the file at its Path,
// relative to root. It returns an error naming the first missing/
// unreadable asset file rather than silently omitting it — a manifest
// that doesn't match what actually shipped is worse than no manifest.
func BuildManifest(model *ir.IR, app AppMetadata, root, platform string) (*Manifest, error) {
	m := &Manifest{
		ManifestVersion: ManifestVersion,
		App:             app,
		Platform:        platform,
	}
	for _, a := range model.Assets {
		full := filepath.Join(root, a.Path)
		f, err := os.Open(full)
		if err != nil {
			return nil, fmt.Errorf("asset %s (%s): %w", a.Qualified(), a.Path, err)
		}
		h := sha256.New()
		size, err := io.Copy(h, f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("hashing asset %s (%s): %w", a.Qualified(), a.Path, err)
		}
		m.Assets = append(m.Assets, AssetEntry{
			Name:   a.Qualified(),
			Kind:   a.Kind,
			Path:   a.Path,
			Size:   size,
			SHA256: hex.EncodeToString(h.Sum(nil)),
		})
	}
	return m, nil
}

// WriteBundle writes a zip archive at outPath containing manifest.json at
// the archive root plus every asset file the manifest describes, read
// from root+entry.Path. Files are stored, not additionally transformed —
// see the package doc comment for why per-asset cooking isn't done here.
func WriteBundle(manifest *Manifest, root, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating bundle: %w", err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	mw, err := zw.Create("manifest.json")
	if err != nil {
		return fmt.Errorf("adding manifest.json: %w", err)
	}
	if _, err := mw.Write(manifestJSON); err != nil {
		return fmt.Errorf("writing manifest.json: %w", err)
	}

	for _, a := range manifest.Assets {
		if err := addFileToZip(zw, filepath.Join(root, a.Path), a.Path); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zw *zip.Writer, srcPath, archivePath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", srcPath, err)
	}
	defer f.Close()

	w, err := zw.Create(filepath.ToSlash(archivePath))
	if err != nil {
		return fmt.Errorf("adding %s to bundle: %w", archivePath, err)
	}
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("writing %s to bundle: %w", archivePath, err)
	}
	return nil
}
