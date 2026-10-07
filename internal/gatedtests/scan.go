// Package gatedtests finds top-level tests that reach a Skip or Skipf call
// with a string literal naming an AGAINROM_ variable. It follows bare-name
// calls through functions in the same package's _test.go files, including
// calls inside subtests. Reachability has no helper-depth limit; each
// function is visited once when propagating a known gate to its callers.
//
// The manifest test (selfTestName) compares this source scan with the
// checked-in population. pipeline/check-release-tests.sh independently
// compares that manifest with an asset-free go test -json census before
// running its packages on the supplied lawful roots.
//
// This is a lexical scan, not a type-checked call graph. It does not resolve
// methods, imported helpers, function values, helpers outside _test.go, or
// computed skip messages. It does not prove a call is reachable at runtime.
// A silent return after reading an environment variable is invisible to
// both the source scan and the runtime skip census.
package gatedtests

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// selfTestName is the exact name of this package's own fail-closed
// manifest test (scan_test.go), named here so the package doc comment
// above cannot drift silently away from the test it names — the failure
// this constant exists to prevent already happened once: an earlier
// revision of the doc comment named a test, TestEveryGatedTestIsNamed,
// that does not exist anywhere in this module (round 2 adversarial review,
// finding 3). TestSelfTestNameIsDeclared parses this package's own
// _test.go files with go/ast and requires a function of this name to
// exist, the same check pkg/ui/screenregistry_test.go's
// TestEveryRegisteredSyntheticTestExists performs for the screen
// registry's own Test column.
//
// Unexported (1024 round 3, minor finding: this constant had no consumer
// outside this package — grep -rn SelfTestName --include=*.go returned only
// scan.go and scan_test.go). Nothing outside this package needs this name;
// exporting it claimed a public API this package does not have.
const selfTestName = "TestScanMatchesTheCheckedInPopulationList"

// Test names one gated test function: its package import path, its own
// function name, and the first helper on a route to the skip. Via is empty
// when the test contains the skip itself.
type Test struct {
	Package string
	Func    string
	Via     string
}

// ID is the manifest key: "<package>.<func>".
func (t Test) ID() string { return t.Package + "." + t.Func }

// Scan walks every Go package under moduleRoot (skipping directories whose
// name starts with ".", nested Git worktrees, and the "knowledge" submodule, which this project
// never edits and which carries its own test suite) and returns every
// top-level TestXxx(t *testing.T) function that reaches an AGAINROM_
// t.Skip/t.Skipf call through package-local test helpers, sorted by ID.
// Only files the default build compiles count, as in the release census.
func Scan(moduleRoot string) ([]Test, error) {
	var out []Test
	err := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != moduleRoot {
			if _, statErr := os.Stat(filepath.Join(path, ".git")); statErr == nil {
				return filepath.SkipDir
			} else if !os.IsNotExist(statErr) {
				return statErr
			}
		}
		if path != moduleRoot && (strings.HasPrefix(name, ".") || name == "knowledge" || name == "builds") {
			return filepath.SkipDir
		}
		rel, relErr := filepath.Rel(moduleRoot, path)
		if relErr != nil {
			return relErr
		}
		pkgTests, pkgErr := scanDir(path, filepath.ToSlash(rel))
		if pkgErr != nil {
			return pkgErr
		}
		out = append(out, pkgTests...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out, nil
}

func scanDir(dir, importPath string) ([]Test, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		if !strings.HasSuffix(fi.Name(), "_test.go") {
			return false
		}
		match, err := build.Default.MatchFile(dir, fi.Name())
		return err != nil || match
	}, 0)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, nil
	}

	var out []Test
	for _, pkg := range pkgs {
		// External _test packages cannot call this package's private helpers.
		bodies := map[string]*ast.FuncDecl{}
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || fn.Body == nil {
					continue
				}
				bodies[fn.Name.Name] = fn
			}
		}
		for name, via := range gatedFunctions(bodies) {
			if isTestFunc(bodies[name]) {
				out = append(out, Test{Package: importPath, Func: name, Via: via})
			}
		}
	}
	return out, nil
}

// isTestFunc reports whether fn is a top-level Go test function:
// exported-looking name starting with "Test", one parameter, by convention
// named t of type *testing.T. The parameter type is checked by its
// syntactic shape (*testing.T) rather than by full type-checking, matching
// the lexical-scan style internal/archtest already uses for determinism.
func isTestFunc(fn *ast.FuncDecl) bool {
	if !strings.HasPrefix(fn.Name.Name, "Test") || fn.Name.Name == "Test" {
		return false
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	star, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	return ok && pkgIdent.Name == "testing" && sel.Sel.Name == "T"
}

// bodyHasGatedSkip reports whether body contains a call
// `<recv>.Skip(...)` or `<recv>.Skipf(...)` with a string literal argument
// containing "AGAINROM_", at any depth within the function (loops,
// branches, subtests).
func bodyHasGatedSkip(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "Skip" && sel.Sel.Name != "Skipf") {
			return true
		}
		for _, arg := range call.Args {
			lit, ok := arg.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			if strings.Contains(lit.Value, "AGAINROM_") {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// gatedFunctions propagates direct gates through the reverse call graph.
// Each function enters the queue once, so helper cycles terminate. Sorted
// seeds and callers make Via deterministic when several routes reach a gate.
func gatedFunctions(bodies map[string]*ast.FuncDecl) map[string]string {
	names := make([]string, 0, len(bodies))
	for name := range bodies {
		names = append(names, name)
	}
	sort.Strings(names)

	via := map[string]string{}
	callers := map[string][]string{}
	var queue []string
	for _, name := range names {
		body := bodies[name].Body
		if bodyHasGatedSkip(body) {
			via[name] = ""
			queue = append(queue, name)
		}
		ast.Inspect(body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || bodies[ident.Name] == nil {
				return true
			}
			// A local variable or parameter can shadow a package helper.
			if ident.Obj != nil && ident.Obj.Kind != ast.Fun {
				return true
			}
			callers[ident.Name] = append(callers[ident.Name], name)
			return true
		})
	}
	for next := 0; next < len(queue); next++ {
		callee := queue[next]
		for _, caller := range callers[callee] {
			if _, found := via[caller]; found {
				continue
			}
			via[caller] = callee
			queue = append(queue, caller)
		}
	}
	return via
}

// ModuleRoot returns the directory containing go.mod, walking up from
// start (normally the current working directory of the calling test or
// command).
func ModuleRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
