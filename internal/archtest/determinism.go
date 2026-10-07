package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// bannedSimImports names the standard-library packages pkg/sim's production
// sources may not import. The DAG check above cannot hold these out and does
// not try: it permits every import whose first path segment has no dot,
// which is every package in the standard library there is.
//
// A path UNDER one of these is banned with it — math/rand/v2 is the same
// generator in a newer package, os/exec is the same operating system — so the
// match is on the path root at a SEGMENT BOUNDARY. A hypothetical math/random
// is not math/rand, and a check that cannot tell the two apart is reporting on
// spelling rather than on packages.
var bannedSimImports = []string{"os", "time", "math/rand"}

// bannedSimTypes names the floating-point type identifiers.
var bannedSimTypes = map[string]bool{
	"float32":    true,
	"float64":    true,
	"complex64":  true,
	"complex128": true,
}

// CheckSimDeterminism is the behavioural half of the determinism wall: a scan of
// pkg/sim's production sources for the nondeterminism the import DAG cannot see.
// files maps a source path to that file's text; the live set comes from
// LoadSimSources, and a synthetic set drives the negative cases, since the real
// package is clean and so can only ever show the check passing.
//
// It reports, per file in path order and then in source order:
//
//   - an import of os, time or math/rand, or of any package under one of them;
//   - any float32, float64, complex64 or complex128 identifier, declared or used;
//   - any floating-point or imaginary literal.
//
// It judges PARSED SYNTAX, not file text: what it looks at are import specs,
// identifiers and numeric literals, and a banned name written in a comment or
// inside a string is none of those. (The sources are parsed without
// parser.ParseComments as well, but that is belt and braces — a comment is not
// an identifier either way.) pkg/sim/rng.go names math/rand in prose to explain
// why it is not the generator, and a scan that made that a violation would be
// paid for by deleting the explanation.
//
// What it proves is LEXICAL: no such import, type name or literal occurs in the
// files given. It does not prove that nondeterminism cannot reach the simulation
// another way — a float arriving as another package's untyped constant (math.Pi)
// is invisible to it, as is whatever a permitted standard-library package does
// inside itself.
//
// An empty file set is itself a violation: a scan that has stopped finding
// sources reports nothing, and "no violations" must not be how that reads.
//
// Violations from this check name the source file and line in From, where the
// DAG check names a package.
func CheckSimDeterminism(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{
			From:   "pkg/sim",
			Reason: "no sources scanned - the determinism scan found nothing to read",
		}}
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	var vs []Violation
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(p token.Pos) string {
			return name + ":" + strconv.Itoa(fset.Position(p).Line)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.ImportSpec:
				imp, uerr := strconv.Unquote(node.Path.Value)
				if uerr != nil {
					return false
				}
				if root, banned := bannedSimImport(imp); banned {
					vs = append(vs, Violation{
						From:   at(node.Pos()),
						Import: imp,
						Reason: "imports " + root + ", which the determinism core must not use",
					})
				}
				return false
			case *ast.Ident:
				if bannedSimTypes[node.Name] {
					vs = append(vs, Violation{
						From:   at(node.Pos()),
						Reason: "names the floating-point type " + node.Name,
					})
				}
			case *ast.BasicLit:
				switch node.Kind {
				case token.FLOAT:
					vs = append(vs, Violation{
						From:   at(node.Pos()),
						Reason: "floating-point literal " + node.Value,
					})
				case token.IMAG:
					vs = append(vs, Violation{
						From:   at(node.Pos()),
						Reason: "imaginary literal " + node.Value,
					})
				}
			}
			return true
		})
	}
	return vs
}

// bannedSimImport reports whether imp is one of the banned packages or lies
// under one, and which root it matched.
func bannedSimImport(imp string) (string, bool) {
	for _, b := range bannedSimImports {
		if imp == b || strings.HasPrefix(imp, b+"/") {
			return b, true
		}
	}
	return "", false
}

// LoadSimSources reads pkg/sim's PRODUCTION sources from the module tree
// rooted at moduleRoot, keyed by their module-relative slash path.
func LoadSimSources(moduleRoot string) (map[string]string, error) {
	dir := filepath.Join(moduleRoot, "pkg", "sim")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			return nil, rerr
		}
		out["pkg/sim/"+name] = string(b)
	}
	return out, nil
}
