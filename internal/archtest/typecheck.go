package archtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// TypeCheckedModule is every package this module owns, type-checked once
// with go/types, so that an instrument asking "what is this expression's
// real type" gets an answer resolved by the compiler's own rules rather than
// by matching how the source happens to be spelled. LoadCommandLiterals and
// LoadComposition both need this, over overlapping files (LoadComposition
// wants only pkg/game, which LoadCommandLiterals also visits as part of the
// whole module), so loadTypeCheckedModule caches the result per module root
// and both loaders share the one pass within a process.
//
// Building it runs `go list -export -json -deps ./...` once to learn every
// package's Dir, GoFiles, TestGoFiles, XTestGoFiles and, for every
// dependency including the standard library, the path to its compiled
// export data - then type-checks each of this module's own packages with
// go/types, resolving imports from that export data through
// go/importer's "gc" compiler mode. No dependency is checked from its own
// source a second time: an import of one of this module's own packages is
// resolved the same way an import of a third-party one is, which is what
// real compilation does for a production import. An external test that uses
// a name an internal _test.go file exports fails to check, loudly; the tree
// has no such file.
type TypeCheckedModule struct {
	Fset     *token.FileSet
	Packages map[string]*CheckedPackage
}

// CheckedPackage is one of this module's own packages. Files holds GoFiles
// followed by TestGoFiles - the internal test augmentation, sharing one
// package clause - and Files[:NumProd] are the production files. XTestFiles
// is the external test package (package foo_test) the same directory may
// also hold; it is always entirely test files, checked separately because it
// is a different package.
type CheckedPackage struct {
	ImportPath string
	Files      []*ast.File
	RelPaths   []string
	NumProd    int
	Info       *types.Info

	XTestFiles    []*ast.File
	XTestRelPaths []string
	XTestInfo     *types.Info
}

var (
	moduleCacheMu sync.Mutex
	moduleCache   = map[string]*TypeCheckedModule{}
)

// loadTypeCheckedModule returns the module rooted at root, type-checking it
// on first call and reusing the result for the rest of the process.
func loadTypeCheckedModule(root string) (*TypeCheckedModule, error) {
	moduleCacheMu.Lock()
	defer moduleCacheMu.Unlock()
	if m, ok := moduleCache[root]; ok {
		return m, nil
	}
	m, err := buildTypeCheckedModule(root)
	if err != nil {
		return nil, err
	}
	moduleCache[root] = m
	return m, nil
}

// goListPkg is the subset of `go list -json` fields this loader reads.
type goListPkg struct {
	ImportPath   string
	Dir          string
	Standard     bool
	GoFiles      []string
	TestGoFiles  []string
	XTestGoFiles []string
	Imports      []string
	Export       string
	Error        *struct{ Err string }
}

// runGoList runs `go list -export -json` with the given patterns and decodes
// the concatenated JSON objects it prints, one per matched package.
func runGoList(root string, patterns ...string) ([]goListPkg, error) {
	args := append([]string{"list", "-export", "-json"}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, errOut.String())
	}
	dec := json.NewDecoder(&out)
	var pkgs []goListPkg
	for {
		var p goListPkg
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("go list %s output: %w", strings.Join(patterns, " "), err)
		}
		if p.Error != nil {
			return nil, fmt.Errorf("go list %s: %s", p.ImportPath, p.Error.Err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// exportLookup resolves an import path to its compiled export data,
// preloaded in bulk from one `go list -export -deps ./...` and filled in on
// demand for the rare path that call misses - a package only a test file
// imports, which the plain build graph does not reach.
type exportLookup struct {
	root  string
	mu    sync.Mutex
	files map[string]string
}

func (l *exportLookup) open(path string) (io.ReadCloser, error) {
	l.mu.Lock()
	f, ok := l.files[path]
	l.mu.Unlock()
	if !ok {
		pkgs, err := runGoList(l.root, path)
		if err != nil {
			return nil, err
		}
		if len(pkgs) == 0 || pkgs[0].Export == "" {
			return nil, fmt.Errorf("no export data for %s", path)
		}
		f = pkgs[0].Export
		l.mu.Lock()
		l.files[path] = f
		l.mu.Unlock()
	}
	return os.Open(f)
}

// moduleImporter special-cases "unsafe", which go list never reports export
// data for, then delegates to the gc-export-data importer.
type moduleImporter struct {
	delegate types.Importer
}

func (m moduleImporter) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	return m.delegate.Import(path)
}

func isOwnPackage(importPath string) bool {
	return importPath == ModulePath || strings.HasPrefix(importPath, ModulePath+"/")
}

func buildTypeCheckedModule(root string) (*TypeCheckedModule, error) {
	bulk, err := runGoList(root, "-deps", "./...")
	if err != nil {
		return nil, err
	}
	lookup := &exportLookup{root: root, files: map[string]string{}}
	for _, p := range bulk {
		if p.Export != "" {
			lookup.files[p.ImportPath] = p.Export
		}
	}
	fset := token.NewFileSet()
	imp := moduleImporter{delegate: importer.ForCompiler(fset, "gc", lookup.open)}

	m := &TypeCheckedModule{Fset: fset, Packages: map[string]*CheckedPackage{}}
	for _, p := range bulk {
		if p.Standard || !isOwnPackage(p.ImportPath) {
			continue
		}
		cp, err := checkOnePackage(fset, root, imp, p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.ImportPath, err)
		}
		m.Packages[p.ImportPath] = cp
	}
	return m, nil
}

func checkOnePackage(fset *token.FileSet, root string, imp types.Importer, p goListPkg) (*CheckedPackage, error) {
	cp := &CheckedPackage{ImportPath: p.ImportPath}

	prodFiles, prodRel, err := parseModuleFiles(fset, root, p.Dir, p.GoFiles)
	if err != nil {
		return nil, err
	}
	testFiles, testRel, err := parseModuleFiles(fset, root, p.Dir, p.TestGoFiles)
	if err != nil {
		return nil, err
	}
	cp.NumProd = len(prodFiles)
	cp.Files = append(append([]*ast.File{}, prodFiles...), testFiles...)
	cp.RelPaths = append(append([]string{}, prodRel...), testRel...)

	var checkErrs []error
	collect := func(e error) { checkErrs = append(checkErrs, e) }

	if len(cp.Files) > 0 {
		cp.Info = &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
		if cp.ImportPath == "againrom/pkg/game" || cp.ImportPath == savPackagePath || slices.Contains(p.Imports, savPackagePath) {
			cp.Info.Defs = map[*ast.Ident]types.Object{}
			cp.Info.Uses = map[*ast.Ident]types.Object{}
		}
		conf := &types.Config{Importer: imp, Error: collect}
		conf.Check(p.ImportPath, fset, cp.Files, cp.Info)
	}

	xFiles, xRel, err := parseModuleFiles(fset, root, p.Dir, p.XTestGoFiles)
	if err != nil {
		return nil, err
	}
	if len(xFiles) > 0 {
		cp.XTestFiles = xFiles
		cp.XTestRelPaths = xRel
		cp.XTestInfo = &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
		conf := &types.Config{Importer: imp, Error: collect}
		conf.Check(p.ImportPath+" [external test]", fset, xFiles, cp.XTestInfo)
	}

	if len(checkErrs) > 0 {
		msgs := make([]string, len(checkErrs))
		for i, e := range checkErrs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("type-check errors:\n%s", strings.Join(msgs, "\n"))
	}
	return cp, nil
}

// parseModuleFiles parses each named file in dir and reports it under its
// path relative to the module root, in slash form, matching every other
// report in this package.
func parseModuleFiles(fset *token.FileSet, root, dir string, names []string) ([]*ast.File, []string, error) {
	if len(names) == 0 {
		return nil, nil, nil
	}
	files := make([]*ast.File, 0, len(names))
	rels := make([]string, 0, len(names))
	for _, name := range names {
		abs := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, abs, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, nil, err
		}
		rel, relErr := filepath.Rel(root, abs)
		if relErr != nil {
			rel = abs
		}
		files = append(files, f)
		rels = append(rels, filepath.ToSlash(rel))
	}
	return files, rels, nil
}
