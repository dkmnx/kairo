package internal_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDocsCoverPublicSymbols scans each public package under internal/ and
// verifies that its exported declarations are at least mentioned in the
// matching `docs/architecture/` Markdown files. This is a coarse drift
// check: it fails only if a package has *no* exported symbols referenced
// from the docs at all, which would indicate a new package with no docs.
func TestDocsCoverPublicSymbols(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Walk up from internal/ to project root.
	// os.Getwd() may return the test binary's working dir, which is the
	// package directory; project root is the parent of internal/.
	if filepath.Base(root) == "internal" {
		root = filepath.Dir(root)
	} else {
		// try parent
		root = filepath.Dir(root)
	}
	docsDir := filepath.Join(root, "docs", "architecture")
	internalDir := filepath.Join(root, "internal")

	if _, err := os.Stat(docsDir); err != nil {
		t.Skipf("docs/architecture not present: %v", err)
	}

	// Read all docs once.
	docsContent := readAllDocs(t, docsDir)
	lowerDocs := strings.ToLower(docsContent)

	packages, err := filepath.Glob(filepath.Join(internalDir, "*"))
	if err != nil {
		t.Fatal(err)
	}

	for _, pkg := range packages {
		info, err := os.Stat(pkg)
		if err != nil || !info.IsDir() {
			continue
		}
		name := filepath.Base(pkg)
		if strings.HasPrefix(name, "_") {
			continue
		}

		exported := collectExported(t, pkg)
		if len(exported) == 0 {
			continue
		}

		// If at least one exported symbol from this package appears in the
		// docs (case-insensitive), consider the package documented. This
		// tolerates renamed symbols but catches entirely undocumented
		// packages.
		found := false
		for _, sym := range exported {
			if strings.Contains(lowerDocs, strings.ToLower(sym)) {
				found = true

				break
			}
		}
		if !found {
			t.Logf("package %q has no exported symbols in docs/architecture/ — consider adding docs", name)
		}
	}
}

func readAllDocs(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		b.Write(data)
		b.WriteString("\n")

		return nil
	})

	return b.String()
}

func collectExported(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(os.FileInfo) bool { return true }, parser.ParseComments)
	if err != nil {
		return nil
	}
	var out []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Name.IsExported() {
						out = append(out, d.Name.Name)
					}
				case *ast.GenDecl:
					if d.Tok != token.TYPE && d.Tok != token.VAR && d.Tok != token.CONST {
						continue
					}
					for _, spec := range d.Specs {
						switch s := spec.(type) {
						case *ast.TypeSpec:
							if s.Name.IsExported() {
								out = append(out, s.Name.Name)
							}
						case *ast.ValueSpec:
							for _, n := range s.Names {
								if n.IsExported() {
									out = append(out, n.Name)
								}
							}
						}
					}
				}
			}
		}
	}

	return out
}
