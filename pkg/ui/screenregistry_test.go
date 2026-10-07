package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// screenConstNamesFromSource parses flow.go and returns every identifier
// declared in the const block that starts with "ScreenMenu Screen = iota" —
// the same source screenRegistry's own comment names. It is read from source
// rather than from a remembered list so a Screen value ADDED to that block
// is caught here without anyone updating this test: the parse finds a new
// identifier, screenConstName does not know it, and the lookup below fails.
func screenConstNamesFromSource(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "flow.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing flow.go: %v", err)
	}
	var names []string
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		found := false
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if name.Name == "ScreenMenu" {
					found = true
				}
				if found {
					names = append(names, vs.Names[i].Name)
				}
			}
		}
		if found {
			return names
		}
	}
	t.Fatal("flow.go: no const block declaring ScreenMenu found — the Screen enum moved, and this test no longer reads it")
	return nil
}

// TestEveryScreenValueIsRegistered fails closed: a Screen value flow.go
// declares with no entry in screenRegistry is an error here, never a silent
// gap (contract B1). It also requires the two maps to agree in the other
// direction — a registry entry naming a Screen value flow.go no longer
// declares is equally a defect, since it would be an entry nothing tests.
func TestEveryScreenValueIsRegistered(t *testing.T) {
	names := screenConstNamesFromSource(t)
	if len(names) == 0 {
		t.Fatal("parsed zero Screen constants from flow.go")
	}
	seen := make(map[Screen]bool, len(names))
	for _, name := range names {
		s, ok := screenConstName[name]
		if !ok {
			t.Errorf("flow.go declares Screen constant %s, which screenConstName does not know: register it in screenregistry.go", name)
			continue
		}
		seen[s] = true
		entry, ok := screenRegistry[s]
		if !ok {
			t.Errorf("Screen %s (%s) has no screenRegistry entry: give it a Composer+Test or a Reason", name, s)
			continue
		}
		hasComposer := entry.Composer != "" || entry.Test != ""
		if hasComposer == (entry.Reason != "") {
			t.Errorf("Screen %s: entry must set exactly one of {Composer+Test, Reason}, got %+v", name, entry)
		}
		if hasComposer && (entry.Composer == "" || entry.Test == "") {
			t.Errorf("Screen %s: a composing entry needs BOTH Composer and Test, got %+v", name, entry)
		}
	}
	if len(seen) != len(screenRegistry) {
		for s := range screenRegistry {
			if !seen[s] {
				t.Errorf("screenRegistry has an entry for %s, which flow.go's const block no longer declares", s)
			}
		}
	}
}

// TestEveryRegisteredSyntheticTestExists is a light guard against a typo in
// screenRegistry's own Test field: every named function must be a real Go
// identifier reachable from this package's test binary. It cannot call the
// test by name (Go gives no such reflection), so it checks the string
// against go/parser's own scan of this package's _test.go files instead —
// the same shape TestEveryScreenValueIsRegistered uses for flow.go.
func TestEveryRegisteredSyntheticTestExists(t *testing.T) {
	declared := testFuncNames(t, ".")
	for s, entry := range screenRegistry {
		if entry.Test == "" {
			continue
		}
		if !declared[entry.Test] {
			t.Errorf("Screen %s names synthetic test %q, which no _test.go file in this package declares", s, entry.Test)
		}
	}
}

// TestEveryRegisteredGeometryTestExists is TestEveryRegisteredSyntheticTestExists's
// own guard, extended to screenEntry.GeometryTests (round 2 adversarial
// review, finding 2: a Test name proves selection only, and this story adds
// a second, separate list for the tests that actually pin a destination
// rectangle). An entry with no "." is looked up in this package's own test
// files; an entry of the form "<import path>.<FuncName>" is looked up in
// that package's own directory instead, the shape
// pkg/render/menu.TestSelectionAndCompose needs — the menu overlay's own
// destination rectangles are pinned in pkg/render/menu, not here.
func TestEveryRegisteredGeometryTestExists(t *testing.T) {
	cache := map[string]map[string]bool{}
	for s, entry := range screenRegistry {
		for _, ref := range entry.GeometryTests {
			dir, name := splitGeometryTestRef(ref)
			declared, ok := cache[dir]
			if !ok {
				declared = testFuncNames(t, dir)
				cache[dir] = declared
			}
			if !declared[name] {
				t.Errorf("Screen %s names geometry test %q, which no _test.go file under %q declares", s, ref, dir)
			}
		}
	}
}

// TestSharedGeometryPointsAtARealComposingScreen guards
// screenEntry.SharesGeometry/SharesGeometryWith (round 3 punch list, finding
// 1): SharesGeometryWith must name a DIFFERENT, registered, composing Screen
// carrying at least one GeometryTests entry of its own. A reference to
// itself, an unregistered value, a Reason-only (refusing) entry, or a
// composing entry with an empty GeometryTests would let ScreenCensus's
// EffectiveGeometryTests report a witness that is not really there —
// exactly the defect this whole mechanism exists to stop cmd/screencensus
// from printing.
func TestSharedGeometryPointsAtARealComposingScreen(t *testing.T) {
	for s, entry := range screenRegistry {
		if !entry.SharesGeometry {
			continue
		}
		if entry.SharesGeometryWith == s {
			t.Errorf("Screen %s: SharesGeometryWith names itself", s)
			continue
		}
		target, ok := screenRegistry[entry.SharesGeometryWith]
		if !ok {
			t.Errorf("Screen %s: SharesGeometryWith names %s, which has no screenRegistry entry", s, entry.SharesGeometryWith)
			continue
		}
		if target.Composer == "" {
			t.Errorf("Screen %s: SharesGeometryWith names %s, which does not compose (Reason: %q)", s, entry.SharesGeometryWith, target.Reason)
		}
		if len(target.GeometryTests) == 0 {
			t.Errorf("Screen %s: SharesGeometryWith names %s, whose own GeometryTests is empty: nothing to share", s, entry.SharesGeometryWith)
		}
	}
}

// splitGeometryTestRef splits a GeometryTests entry into the directory to
// parse and the bare function name to look up in it. "TestFoo" (no dot)
// means this package's own directory; "pkg/render/menu.TestFoo" means that
// module-relative import path's own directory, split on the LAST dot so a
// package name itself containing no dot (every package name in this
// module) is unambiguous. go test's own working directory is this
// package's directory (pkg/ui), two levels below the module root, so the
// import path is resolved relative to "../..".
func splitGeometryTestRef(ref string) (dir, name string) {
	i := strings.LastIndex(ref, ".")
	if i < 0 {
		return ".", ref
	}
	importPath := ref[:i]
	return filepath.Join("../..", filepath.FromSlash(importPath)), ref[i+1:]
}

// testFuncNames parses every _test.go file in dir and returns the set of
// top-level Test... function names it declares. dir "." is this package's
// own directory.
func testFuncNames(t *testing.T, dir string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		name := fi.Name()
		return len(name) > 8 && name[len(name)-8:] == "_test.go"
	}, 0)
	if err != nil {
		t.Fatalf("parsing package directory %q: %v", dir, err)
	}
	names := make(map[string]bool)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Recv != nil {
					continue
				}
				names[fd.Name.Name] = true
			}
		}
	}
	return names
}
