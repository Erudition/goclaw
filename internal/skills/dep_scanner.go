package skills

import (
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SkillManifest holds dependency info for a skill.
// Populated by ScanSkillDeps via static analysis of scripts/ directory,
// optionally overridden/filtered by SKILL.md frontmatter (deps: / exclude_deps:).
type SkillManifest struct {
	Requires       []string `json:"requires,omitempty"`        // system binaries (python3, pandoc, ffmpeg)
	RequiresPython []string `json:"requires_python,omitempty"` // raw Python import names (e.g. "openpyxl", "cv2")
	RequiresNode   []string `json:"requires_node,omitempty"`   // npm package names (e.g. "docx", "pptxgenjs")
	ScriptsDir     string   `json:"-"`                         // absolute path to scripts/ dir, used for PYTHONPATH
	// Manifest-origin fields — populated when SKILL.md declares deps:/exclude_deps:.
	Explicit     []string `json:"explicit,omitempty"`      // raw dep strings from SKILL.md deps: (e.g. "pip:psycopg2-binary")
	ExcludeDeps  []string `json:"exclude_deps,omitempty"`  // filter list from SKILL.md exclude_deps:
	FromManifest bool     `json:"from_manifest,omitempty"` // true when Explicit was the authoritative source
}

// IsEmpty returns true if the manifest has no dependencies.
func (m *SkillManifest) IsEmpty() bool {
	return len(m.Requires) == 0 && len(m.RequiresPython) == 0 && len(m.RequiresNode) == 0
}

// ScanSkillDeps auto-detects dependencies by statically analyzing the scripts/ directory,
// then applies any SKILL.md frontmatter overrides (deps: / exclude_deps:).
func ScanSkillDeps(skillDir string) *SkillManifest {
	scan := scanScriptsDir(filepath.Join(skillDir, "scripts"))
	deps, excludeDeps := parseSkillManifestFile(skillMdPath(skillDir))
	if len(deps) == 0 && len(excludeDeps) == 0 {
		return scan
	}
	merged := applyManifestOverride(scan, deps, excludeDeps)
	if merged.FromManifest {
		slog.Debug("dep_scanner: manifest override applied",
			"dir", skillDir,
			"explicit_count", len(deps),
			"scan_py", len(scan.RequiresPython),
			"scan_node", len(scan.RequiresNode))
	} else if len(excludeDeps) > 0 {
		slog.Debug("dep_scanner: manifest exclude applied",
			"dir", skillDir, "exclude_count", len(excludeDeps))
	}
	return merged
}

// scanScriptsDir statically analyzes script files to detect dependencies.
// Local module directories (subdirs of scriptsDir) are excluded from pyImports;
// stdlib/pip resolution is handled at check time via PYTHONPATH.
func scanScriptsDir(scriptsDir string) *SkillManifest {
	m := &SkillManifest{ScriptsDir: scriptsDir}

	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return m
	}

	pyImports := make(map[string]bool)
	nodeImports := make(map[string]bool)
	binaries := make(map[string]bool)
	// Track directory names and .py filenames — these are local modules and must never be reported as missing.
	localModules := make(map[string]bool)
	// The scripts directory itself can be referenced as a module (e.g. "from scripts import utils").
	localModules[filepath.Base(scriptsDir)] = true

	// Walk the scripts directory once to find all local modules (dirs and .py files)
	_ = filepath.WalkDir(scriptsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != scriptsDir {
				localModules[d.Name()] = true
			}
		} else if strings.HasSuffix(d.Name(), ".py") {
			localModules[strings.TrimSuffix(d.Name(), ".py")] = true
		}
		return nil
	})

	// Walk again to scan imports in all files
	_ = filepath.WalkDir(scriptsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		scanFile(path, pyImports, nodeImports, binaries)
		return nil
	})

	for b := range binaries {
		m.Requires = append(m.Requires, b)
	}
	// Store raw import names — skip local modules and Python stdlib.
	// Stdlib is also resolved at check time via actual import, but filtering here
	// prevents false positives when the checker fails (timeout, env issue, crash).
	for pkg := range pyImports {
		if !localModules[pkg] && !pythonStdlib[pkg] {
			m.RequiresPython = append(m.RequiresPython, pkg)
		}
	}
	for pkg := range nodeImports {
		m.RequiresNode = append(m.RequiresNode, pkg)
	}

	// Auto-detect runtime from file extensions
	if len(pyImports) > 0 && !binaries["python3"] {
		m.Requires = append(m.Requires, "python3")
	}
	if len(nodeImports) > 0 && !binaries["node"] {
		m.Requires = append(m.Requires, "node")
	}

	if !m.IsEmpty() {
		slog.Debug("dep_scanner: scanned", "dir", scriptsDir,
			"bins", len(m.Requires), "py", len(m.RequiresPython), "node", len(m.RequiresNode))
	}

	return m
}

var (
	pyImportRe     = regexp.MustCompile(`^import\s+(\w+)`)
	pyFromRe       = regexp.MustCompile(`^from\s+(\w+)`)
	nodeRequireRe  = regexp.MustCompile(`require\(['"]([\w@][^'"]*)['"]\)`)
	nodeESImportRe = regexp.MustCompile(`from\s+['"]([^'"./][^'"]*?)['"]`)
	shebangRe      = regexp.MustCompile(`^#!\s*/usr/bin/env\s+(\S+)`)
	// Detects JS ES module pattern: `import X from '...'` or `from '...'`.
	// Used to skip false positives when JS imports appear inside Python string literals.
	jsFromStringRe = regexp.MustCompile(`from\s+['"]`)
)

func scanFile(path string, pyImports, nodeImports map[string]bool, binaries map[string]bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)
	ext := filepath.Ext(path)

	// Check shebang
	if strings.HasPrefix(content, "#!") {
		firstLine := strings.SplitN(content, "\n", 2)[0]
		if m := shebangRe.FindStringSubmatch(firstLine); len(m) > 1 {
			binaries[m[1]] = true
		}
	}

	switch ext {
	case ".py":
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if m := pyImportRe.FindStringSubmatch(line); len(m) > 1 {
				// Skip JS ES module imports inside string literals (e.g. `import mermaid from '...'`)
				if !jsFromStringRe.MatchString(line) {
					pyImports[m[1]] = true
				}
			}
			if m := pyFromRe.FindStringSubmatch(line); len(m) > 1 {
				pyImports[m[1]] = true
			}
		}
	case ".js", ".mjs":
		for _, m := range nodeRequireRe.FindAllStringSubmatch(content, -1) {
			if len(m) > 1 {
				nodeImports[normalizeNodePkg(m[1])] = true
			}
		}
		for _, m := range nodeESImportRe.FindAllStringSubmatch(content, -1) {
			if len(m) > 1 {
				nodeImports[normalizeNodePkg(m[1])] = true
			}
		}
	}
}

func normalizeNodePkg(pkg string) string {
	if strings.HasPrefix(pkg, "@") {
		parts := strings.SplitN(pkg, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return pkg
	}
	return strings.SplitN(pkg, "/", 2)[0]
}

// MergeDeps merges two manifests, deduplicating entries.
// Manifest-origin fields (Explicit, ExcludeDeps, FromManifest) are OR-folded /
// unioned so the merged result remains authoritative if either side was.
func MergeDeps(a, b *SkillManifest) *SkillManifest {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return &SkillManifest{
		Requires:       mergeUnique(a.Requires, b.Requires),
		RequiresPython: mergeUnique(a.RequiresPython, b.RequiresPython),
		RequiresNode:   mergeUnique(a.RequiresNode, b.RequiresNode),
		Explicit:       mergeUnique(a.Explicit, b.Explicit),
		ExcludeDeps:    mergeUnique(a.ExcludeDeps, b.ExcludeDeps),
		FromManifest:   a.FromManifest || b.FromManifest,
	}
}

func mergeUnique(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	var result []string
	for _, s := range append(a, b...) {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
