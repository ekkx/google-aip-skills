// Command build generates the google-aip skill from the upstream
// aip-dev/google.aip.dev repository.
//
// Usage:
//
//	go run ./cmd/build [-source <path>] [-out <skill-dir>]
//
// If -source is omitted, the upstream repo is shallow-cloned into a temp dir.
// The output skill directory defaults to .claude/skills/google-aip.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"gopkg.in/yaml.v3"
)

const upstreamRepo = "https://github.com/aip-dev/google.aip.dev.git"

type categoryYAML struct {
	Code    string `yaml:"code"`
	Title   string `yaml:"title"`
	Default bool   `yaml:"default"`
}

type scopeYAML struct {
	Title      string         `yaml:"title"`
	Order      int            `yaml:"order"`
	Categories []categoryYAML `yaml:"categories"`
}

type category struct {
	Code    string
	Title   string
	Order   int
	Default bool
}

type scope struct {
	Code       string
	Title      string
	Order      int
	Categories map[string]category // code -> category
	catOrder   []string            // category codes in declaration order
}

type aipFrontmatter struct {
	ID        int    `yaml:"id"`
	State     string `yaml:"state"`
	Created   string `yaml:"created"`
	Placement struct {
		Category string `yaml:"category"`
		Order    int    `yaml:"order"`
	} `yaml:"placement"`
}

type aip struct {
	Scope    string
	Category string
	ID       int
	Title    string
	State    string
	SrcPath  string
}

func (a aip) filename() string {
	return fmt.Sprintf("%04d.md", a.ID)
}

func main() {
	source := flag.String("source", "", "Path to an already-cloned google.aip.dev repo. If empty, a shallow clone is made.")
	out := flag.String("out", filepath.Join("skills", "google-aip"), "Output skill directory")
	repoRoot := flag.String("repo-root", ".", "Repo root for non-Claude-Code agent entry points (AGENTS.md, .cursor/rules/). Use the same value as -out's parent for normal builds.")
	emitCodex := flag.Bool("emit-codex", true, "Generate AGENTS.md at repo root for Codex / generic agents.")
	emitCursor := flag.Bool("emit-cursor", true, "Generate .cursor/rules/google-aip.mdc for Cursor.")
	flag.Parse()

	log.SetFlags(0)

	srcDir, sha, cleanup, err := resolveSource(*source)
	if err != nil {
		log.Fatalf("source: %v", err)
	}
	defer cleanup()

	scopes, err := loadScopes(srcDir)
	if err != nil {
		log.Fatalf("loadScopes: %v", err)
	}
	if len(scopes) == 0 {
		log.Fatalf("no scopes found under %s/aip/*/scope.yaml", srcDir)
	}

	aips, err := collectAIPs(srcDir, scopes)
	if err != nil {
		log.Fatalf("collectAIPs: %v", err)
	}
	if len(aips) == 0 {
		log.Fatalf("no approved AIPs found")
	}

	if err := cleanSkillDir(*out); err != nil {
		log.Fatalf("clean: %v", err)
	}
	if err := writeAIPFiles(aips, *out); err != nil {
		log.Fatalf("writeAIPFiles: %v", err)
	}

	byScope := groupByScope(aips)
	for code, scope := range scopes {
		items := byScope[code]
		if len(items) == 0 {
			continue
		}
		if err := writeScopeIndex(scope, items, *out); err != nil {
			log.Fatalf("writeScopeIndex(%s): %v", code, err)
		}
	}

	if err := writeSkillMD(scopes, byScope, *out); err != nil {
		log.Fatalf("writeSkillMD: %v", err)
	}
	if err := writeSourceMD(*out, sha, len(aips)); err != nil {
		log.Fatalf("writeSourceMD: %v", err)
	}

	if *emitCodex {
		if err := writeAgentsMD(scopes, byScope, *repoRoot, *out); err != nil {
			log.Fatalf("writeAgentsMD: %v", err)
		}
	}
	if *emitCursor {
		if err := writeCursorRule(scopes, byScope, *repoRoot, *out); err != nil {
			log.Fatalf("writeCursorRule: %v", err)
		}
	}

	fmt.Fprintf(os.Stderr, "built %d approved AIPs across %d scopes -> %s\n",
		len(aips), len(byScope), *out)
}

func resolveSource(source string) (dir, sha string, cleanup func(), err error) {
	cleanup = func() {}
	if source != "" {
		abs, err := filepath.Abs(source)
		if err != nil {
			return "", "", cleanup, err
		}
		if _, err := os.Stat(filepath.Join(abs, "aip")); err != nil {
			return "", "", cleanup, fmt.Errorf("%s does not look like a google.aip.dev checkout", abs)
		}
		sha, err = gitHeadSHA(abs)
		if err != nil {
			return "", "", cleanup, err
		}
		return abs, sha, cleanup, nil
	}

	tmp, err := os.MkdirTemp("", "google-aip-")
	if err != nil {
		return "", "", cleanup, err
	}
	cleanup = func() { _ = os.RemoveAll(tmp) }

	repoDir := filepath.Join(tmp, "repo")
	fmt.Fprintf(os.Stderr, "cloning %s ...\n", upstreamRepo)
	cmd := exec.Command("git", "clone", "--depth", "1", upstreamRepo, repoDir)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", "", func() {}, fmt.Errorf("git clone: %v: %s", err, outBytes)
	}
	sha, err = gitHeadSHA(repoDir)
	if err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	return repoDir, sha, cleanup, nil
}

func gitHeadSHA(repo string) (string, error) {
	cmd := exec.Command("git", "-C", repo, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func loadScopes(srcDir string) (map[string]*scope, error) {
	scopes := map[string]*scope{}
	aipRoot := filepath.Join(srcDir, "aip")
	entries, err := os.ReadDir(aipRoot)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		scopeYAMLPath := filepath.Join(aipRoot, e.Name(), "scope.yaml")
		data, err := os.ReadFile(scopeYAMLPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		var raw scopeYAML
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("%s: %w", scopeYAMLPath, err)
		}
		s := &scope{
			Code:       e.Name(),
			Title:      orDefault(raw.Title, humanize(e.Name())),
			Order:      raw.Order,
			Categories: map[string]category{},
		}
		for i, c := range raw.Categories {
			title := c.Title
			if title == "" {
				title = humanize(c.Code)
			}
			s.Categories[c.Code] = category{
				Code:    c.Code,
				Title:   title,
				Order:   i,
				Default: c.Default,
			}
			s.catOrder = append(s.catOrder, c.Code)
		}
		// Guarantee a fallback "misc" category exists so AIPs without a
		// declared placement.category always land somewhere.
		if _, has := s.Categories["misc"]; !has {
			s.Categories["misc"] = category{
				Code:  "misc",
				Title: "Miscellaneous",
				Order: len(s.catOrder),
			}
			s.catOrder = append(s.catOrder, "misc")
		}
		scopes[s.Code] = s
	}
	return scopes, nil
}

func collectAIPs(srcDir string, scopes map[string]*scope) ([]aip, error) {
	var all []aip
	for code, s := range scopes {
		scopeDir := filepath.Join(srcDir, "aip", code)
		entries, err := os.ReadDir(scopeDir)
		if err != nil {
			return nil, err
		}
		defaultCat := defaultCategory(s)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			fullPath := filepath.Join(scopeDir, e.Name())
			fm, body, ok, err := parseFrontmatter(fullPath)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", fullPath, err)
			}
			if !ok {
				continue
			}
			if fm.ID == 0 {
				continue
			}
			if fm.State != "approved" {
				continue
			}
			cat := fm.Placement.Category
			if _, found := s.Categories[cat]; !found {
				cat = defaultCat
			}
			all = append(all, aip{
				Scope:    code,
				Category: cat,
				ID:       fm.ID,
				Title:    extractTitle(body, fmt.Sprintf("AIP-%d", fm.ID)),
				State:    fm.State,
				SrcPath:  fullPath,
			})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Scope != all[j].Scope {
			return all[i].Scope < all[j].Scope
		}
		return all[i].ID < all[j].ID
	})
	return all, nil
}

func defaultCategory(s *scope) string {
	for _, c := range s.Categories {
		if c.Default {
			return c.Code
		}
	}
	// loadScopes guarantees "misc" exists.
	return "misc"
}

func parseFrontmatter(path string) (fm aipFrontmatter, body string, ok bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fm, "", false, err
	}
	text := string(data)
	if !strings.HasPrefix(text, "---") {
		return fm, "", false, nil
	}
	rest := text[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return fm, "", false, nil
	}
	header := rest[:end]
	body = rest[end+len("\n---"):]
	body = strings.TrimLeft(body, "\n")
	if err := yaml.Unmarshal([]byte(header), &fm); err != nil {
		return fm, "", false, err
	}
	return fm, body, true, nil
}

func extractTitle(body, fallback string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return fallback
}

func cleanSkillDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

func writeAIPFiles(aips []aip, skillRoot string) error {
	for _, a := range aips {
		destDir := filepath.Join(skillRoot, "references", a.Scope, a.Category)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
		dest := filepath.Join(destDir, a.filename())
		if err := copyFile(a.SrcPath, dest); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func groupByScope(aips []aip) map[string][]aip {
	out := map[string][]aip{}
	for _, a := range aips {
		out[a.Scope] = append(out[a.Scope], a)
	}
	return out
}

func writeScopeIndex(s *scope, aips []aip, skillRoot string) error {
	byCat := map[string][]aip{}
	for _, a := range aips {
		byCat[a.Category] = append(byCat[a.Category], a)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s (`%s`)\n\n", s.Title, s.Code)
	fmt.Fprintf(&b,
		"AIPs in the `%s` scope, grouped by category. Read the specific AIP file from "+
			"`references/%s/<category>/NNNN.md` when you need its full content.\n\n",
		s.Code, s.Code)

	for _, code := range s.catOrder {
		cat := s.Categories[code]
		items := byCat[code]
		if len(items) == 0 {
			continue
		}
		sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
		fmt.Fprintf(&b, "## %s (`%s`)\n\n", cat.Title, cat.Code)
		for _, a := range items {
			fmt.Fprintf(&b, "- **AIP-%d** — %s (`references/%s/%s/%s`)\n",
				a.ID, a.Title, a.Scope, a.Category, a.filename())
		}
		b.WriteString("\n")
	}

	out := filepath.Join(skillRoot, "references", s.Code, "INDEX.md")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, []byte(b.String()), 0o644)
}

func writeSkillMD(scopes map[string]*scope, byScope map[string][]aip, skillRoot string) error {
	desc := "Authoritative reference for Google AIP (API Improvement Proposals) — " +
		"the design guidelines maintained at https://aip.dev for resource-oriented " +
		"API design, naming, errors, pagination, long-running operations, versioning, " +
		"and related conventions. Use this skill whenever the user is designing, " +
		"reviewing, or implementing an API and any of these terms or concepts come up: " +
		"AIP, aip.dev, resource names, standard methods (Get/List/Create/Update/Delete), " +
		"custom methods, LRO, pagination tokens, field masks, error codes, API versioning, " +
		"or 'how does Google design X'. Also trigger when a specific AIP number is mentioned " +
		"(e.g. 'AIP-121', 'AIP-158'). Prefer this skill over generic API advice — the " +
		"content here is the actual upstream specification."

	ordered := orderedScopesWithAIPs(scopes, byScope)

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: google-aip\n")
	fmt.Fprintf(&b, "description: %s\n", yamlSingleQuoted(desc))
	b.WriteString("---\n\n")
	b.WriteString("# Google AIP (API Improvement Proposals)\n\n")
	b.WriteString("This skill bundles the full, current text of every approved Google AIP, " +
		"sourced verbatim from the upstream [aip-dev/google.aip.dev](https://github.com/aip-dev/google.aip.dev) " +
		"repository. The bundled snapshot is refreshed automatically; see `SOURCE.md` " +
		"for the exact upstream commit this build was generated from.\n\n")
	b.WriteString("## How to use this skill\n\n")
	b.WriteString("AIPs are organized into **scopes** (general guidance vs. Google-Cloud-specific, etc.). " +
		"Each scope contains **categories** (e.g. resource design, errors), and each category " +
		"contains numbered AIP documents.\n\n")
	b.WriteString("1. **If the user mentions a specific AIP number** (e.g. `AIP-121`), open the matching " +
		"file directly. AIP numbers are unique across scopes — search " +
		"`references/*/*/NNNN.md` (zero-padded to 4 digits).\n")
	b.WriteString("2. **Otherwise, pick the relevant scope** from the table below, read its `INDEX.md` " +
		"to find the right category and AIP number, then read the individual AIP file.\n")
	b.WriteString("3. **Cite AIPs by number** in your response (e.g. \"per AIP-131…\") so the user " +
		"can verify against aip.dev.\n\n")
	b.WriteString("## Scopes\n\n")
	writeScopeTable(&b, ordered, byScope, "references")
	b.WriteString("\n## Notes on usage\n\n")
	b.WriteString("- The AIP markdown files preserve the upstream YAML frontmatter " +
		"(`id`, `state`, `created`, `placement`, etc.) — useful for cross-referencing.\n")
	b.WriteString("- Only AIPs with `state: approved` are included. Drafts and reviewing AIPs " +
		"are intentionally excluded to avoid recommending unsettled guidance.\n")
	b.WriteString("- When an AIP references another (e.g. AIP-131 mentions AIP-121), follow " +
		"the link by reading the referenced file in the same `references/` tree.\n")

	return os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte(b.String()), 0o644)
}

// orderedScopesWithAIPs returns scopes ordered by their declared order,
// excluding any scope that has zero approved AIPs.
func orderedScopesWithAIPs(scopes map[string]*scope, byScope map[string][]aip) []*scope {
	ordered := make([]*scope, 0, len(scopes))
	for _, s := range scopes {
		if len(byScope[s.Code]) == 0 {
			continue
		}
		ordered = append(ordered, s)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Order < ordered[j].Order })
	return ordered
}

var scopeBlurbs = map[string]string{
	"general":          "Cross-cutting API design principles applicable to any API.",
	"cloud":            "Conventions specific to Google Cloud APIs.",
	"auth":             "Authentication and authorization patterns.",
	"client-libraries": "Guidance for generated client libraries (idiomatic surface, packaging).",
	"aog":              "Actions on Google (conversational / assistant APIs).",
	"apps":             "Google Workspace / Apps APIs.",
	"firebase":         "Firebase platform APIs.",
}

// writeScopeTable writes the per-scope table used by every agent entry point.
// referencePrefix is what each row's index link is relative to: e.g. "references"
// when the consumer reads from skills/google-aip/SKILL.md, or
// "skills/google-aip/references" when the consumer reads from the repo root.
func writeScopeTable(b *strings.Builder, ordered []*scope, byScope map[string][]aip, referencePrefix string) {
	b.WriteString("| Scope | What it covers | AIP count | Index |\n")
	b.WriteString("|---|---|---:|---|\n")
	for _, s := range ordered {
		blurb, ok := scopeBlurbs[s.Code]
		if !ok {
			blurb = s.Title
		}
		fmt.Fprintf(b, "| `%s` | %s | %d | [`%s/%s/INDEX.md`](%s/%s/INDEX.md) |\n",
			s.Code, blurb, len(byScope[s.Code]),
			referencePrefix, s.Code,
			referencePrefix, s.Code)
	}
}

// writeAgentsMD generates an AGENTS.md at the repo root for Codex and other
// agents that pick up AGENTS.md hierarchically. Links point at the same
// `skills/google-aip/references/` tree that the Claude Code skill uses.
func writeAgentsMD(scopes map[string]*scope, byScope map[string][]aip, repoRoot, skillRoot string) error {
	// referencePrefix is relative to AGENTS.md's location (= repoRoot).
	referencePrefix, err := filepath.Rel(repoRoot, filepath.Join(skillRoot, "references"))
	if err != nil {
		return err
	}
	referencePrefix = filepath.ToSlash(referencePrefix)

	ordered := orderedScopesWithAIPs(scopes, byScope)

	var b strings.Builder
	b.WriteString("# Google AIP (API Improvement Proposals)\n\n")
	b.WriteString("This file gives any AGENTS.md-aware agent (OpenAI Codex CLI, etc.) " +
		"the same Google AIP reference that the bundled Claude Code skill provides. " +
		"It bundles the full, current text of every approved AIP, sourced verbatim " +
		"from [aip-dev/google.aip.dev](https://github.com/aip-dev/google.aip.dev) " +
		"and refreshed daily by CI. See `skills/google-aip/SOURCE.md` for the exact " +
		"upstream commit this build was generated from.\n\n")
	b.WriteString("## When to consult this reference\n\n")
	b.WriteString("Use these documents whenever the user is designing, reviewing, or " +
		"implementing an API and mentions any of: AIP, aip.dev, resource-oriented " +
		"design, resource names, standard methods (Get/List/Create/Update/Delete), " +
		"custom methods, long-running operations (LRO), pagination tokens, field " +
		"masks, error codes, API versioning, or a specific `AIP-NNN` number. " +
		"Prefer this material over generic API advice — it is the actual upstream " +
		"specification.\n\n")
	b.WriteString("## How to navigate\n\n")
	b.WriteString("AIPs are organized into **scopes** (general guidance vs. Google-Cloud-specific, " +
		"etc.). Each scope contains **categories** (e.g. resource design, errors), " +
		"and each category contains numbered AIP documents.\n\n")
	fmt.Fprintf(&b,
		"1. **If the user names a specific AIP number** (e.g. `AIP-121`), open the matching "+
			"file directly under `%s/<scope>/<category>/<NNNN>.md` (zero-padded to 4 digits).\n",
		referencePrefix)
	b.WriteString("2. **Otherwise, pick the relevant scope** from the table below, read its `INDEX.md` " +
		"to find the right category and AIP number, then read the individual AIP file.\n")
	b.WriteString("3. **Cite AIPs by number** in your response (e.g. \"per AIP-131…\") so the user " +
		"can verify against aip.dev.\n\n")
	b.WriteString("## Scopes\n\n")
	writeScopeTable(&b, ordered, byScope, referencePrefix)
	b.WriteString("\n## Notes\n\n")
	b.WriteString("- The AIP markdown files preserve the upstream YAML frontmatter " +
		"(`id`, `state`, `created`, `placement`, …) — useful for cross-referencing.\n")
	b.WriteString("- Only AIPs with `state: approved` are included. Drafts and reviewing " +
		"AIPs are intentionally excluded so unsettled guidance is never recommended.\n")
	b.WriteString("- When an AIP references another (e.g. AIP-131 mentions AIP-121), follow " +
		"the link by reading the referenced file in the same reference tree.\n")

	out := filepath.Join(repoRoot, "AGENTS.md")
	return os.WriteFile(out, []byte(b.String()), 0o644)
}

// writeCursorRule generates .cursor/rules/google-aip.mdc for Cursor's
// description-triggered rules system.
func writeCursorRule(scopes map[string]*scope, byScope map[string][]aip, repoRoot, skillRoot string) error {
	rulePath := filepath.Join(repoRoot, ".cursor", "rules", "google-aip.mdc")
	if err := os.MkdirAll(filepath.Dir(rulePath), 0o755); err != nil {
		return err
	}
	// referencePrefix is relative to the rule file's location.
	referencePrefix, err := filepath.Rel(filepath.Dir(rulePath), filepath.Join(skillRoot, "references"))
	if err != nil {
		return err
	}
	referencePrefix = filepath.ToSlash(referencePrefix)

	ordered := orderedScopesWithAIPs(scopes, byScope)

	desc := "Authoritative reference for Google AIP (API Improvement Proposals). " +
		"Apply when the user is designing or reviewing an API and mentions AIPs, " +
		"resource-oriented design, standard methods, pagination, errors, " +
		"long-running operations, API versioning, or a specific AIP-NNN number."

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "description: %s\n", yamlSingleQuoted(desc))
	b.WriteString("alwaysApply: false\n")
	b.WriteString("---\n\n")
	b.WriteString("# Google AIP (API Improvement Proposals)\n\n")
	b.WriteString("Bundled, offline reference for every approved Google AIP, sourced " +
		"verbatim from [aip-dev/google.aip.dev](https://github.com/aip-dev/google.aip.dev) " +
		"and refreshed daily by CI.\n\n")
	b.WriteString("## How to navigate\n\n")
	fmt.Fprintf(&b,
		"1. **If a specific AIP number is mentioned** (e.g. `AIP-121`), open the matching "+
			"file directly under `%s/<scope>/<category>/<NNNN>.md`.\n",
		referencePrefix)
	b.WriteString("2. **Otherwise, choose the right scope** from the table below, read its " +
		"`INDEX.md`, then read the individual AIP file.\n")
	b.WriteString("3. **Cite AIPs by number** in responses (e.g. \"per AIP-131…\") so the user " +
		"can verify against aip.dev.\n\n")
	b.WriteString("## Scopes\n\n")
	writeScopeTable(&b, ordered, byScope, referencePrefix)
	b.WriteString("\nOnly AIPs with `state: approved` are included. Each file preserves the " +
		"upstream YAML frontmatter (`id`, `state`, `created`, `placement`, …).\n")

	return os.WriteFile(rulePath, []byte(b.String()), 0o644)
}

func writeSourceMD(skillRoot, sha string, total int) error {
	// Intentionally omit a generated-at timestamp: the build must be idempotent
	// for a given upstream commit so daily CI doesn't churn commits when nothing
	// has actually changed. The git commit timestamp on this file is the source
	// of truth for when it was last refreshed.
	content := "# Source\n\n" +
		"- Upstream repository: <https://github.com/aip-dev/google.aip.dev>\n" +
		fmt.Sprintf("- Commit SHA: `%s`\n", sha) +
		fmt.Sprintf("- Approved AIPs imported: %d\n", total)
	return os.WriteFile(filepath.Join(skillRoot, "SOURCE.md"), []byte(content), 0o644)
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// yamlSingleQuoted wraps s as a YAML single-quoted scalar. This is the safest
// way to embed an arbitrary one-line string in frontmatter: single-quoted
// scalars allow any character (including ": ", "#", etc.) except a literal
// single quote, which YAML escapes by doubling.
func yamlSingleQuoted(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func humanize(code string) string {
	parts := strings.Split(code, "-")
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
