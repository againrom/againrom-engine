package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// RandomnessRoots are the module-relative trees whose production sources may
// draw only through pkg/random: every package under them.
var RandomnessRoots = []string{"pkg", "cmd"}

// bannedRandomImports are the generators and random seeds outside pkg/random.
// A path under one of them is banned with it.
var bannedRandomImports = []string{"math/rand", "crypto/rand", "hash/maphash"}

// lcgConstants are the original generator's multiplier and increment. Written
// anywhere but pkg/random they are a second copy of the recurrence.
var lcgConstants = map[uint64]bool{214013: true, 2531011: true}

// randomSinks are the pkg/random calls that take a seed or a session.
var randomSinks = map[string]bool{
	"NewGo": true, "NewStream": true, "NewService": true, "SetStreamSeed": true,
	"SetLaunch": true, "Fresh": true, "Begin": true, "Prepare": true,
	"ReseedValue": true, "DeriveSeed": true, "MissionLoadState": true,
	"NewDraws": true, "NewOriginalDraws": true,
}

// generatorFields are the field names a generator keeps its seed or state in.
var generatorFields = map[string]bool{"State": true, "Seed": true}

// The rules a violation breaks, as the allow list names them.
const (
	ruleConstant = "constant"
	ruleClock    = "clock"
)

// RandomnessAllowance admits one rule in one file, with its reason. Rule is
// an import path, ruleConstant or ruleClock.
type RandomnessAllowance struct {
	File, Rule, Reason string
}

// RandomnessAllowed is the explicit allow list.
var RandomnessAllowed = []RandomnessAllowance{
	{"pkg/random/gosource.go", "math/rand", "the seeded mode's own generator; every stream takes its seed from the session"},
	{"pkg/random/msvc.go", ruleConstant, "the original's recurrence lives here once"},
	{"pkg/ui/panel.go", "hash/maphash", "the hash seed of a layout memo key; it orders no draw and decides no hashed state"},
}

func randomnessAllowed(file, rule string) bool {
	for _, a := range RandomnessAllowed {
		if a.File == file && a.Rule == rule {
			return true
		}
	}
	return false
}

// RandomnessDebt counts, by file, the violations that could not yet move into
// pkg/random. A count may only fall; a fall lowers it here in the same commit.
var RandomnessDebt = map[string]int{}

// SanctionedClockSeeds names the one function, by file, that may read the
// clock to choose a session seed. Every stream then derives from that seed.
var SanctionedClockSeeds = map[string]string{"pkg/game/randomsession.go": "clockSessionSeed"}

// CheckRandomness scans production sources, keyed by module-relative slash
// path, for randomness that bypasses the random service, per file in path
// order and then in source order: an import under bannedRandomImports; the
// original generator's multiplier or increment as a literal; a clock value
// reaching a pkg/random seed argument, a pkg/random composite literal or a
// State or Seed field, directly or through local and package variables; and
// a seed function that reads package time, other than the sanctioned one.
// It judges parsed syntax: a name in a comment or a string is none of these.
// RandomnessAllowed admits a rule in a file.
func CheckRandomness(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "randomness", Reason: "no sources scanned"}}
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
		type found struct {
			pos token.Pos
			v   Violation
		}
		var hits []found
		report := func(p token.Pos, rule, imp, reason string) {
			if !randomnessAllowed(name, rule) {
				hits = append(hits, found{p, Violation{From: name + ":" + strconv.Itoa(fset.Position(p).Line), Import: imp, Reason: reason}})
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.ImportSpec:
				imp, _ := strconv.Unquote(node.Path.Value)
				for _, banned := range bannedRandomImports {
					if imp == banned || strings.HasPrefix(imp, banned+"/") {
						report(node.Pos(), banned, imp, "imports a generator outside pkg/random")
					}
				}
				return false
			case *ast.BasicLit:
				if node.Kind == token.INT {
					if v, err := strconv.ParseUint(strings.ReplaceAll(node.Value, "_", ""), 0, 64); err == nil && lcgConstants[v] {
						report(node.Pos(), ruleConstant, "", "writes the original generator's constant "+node.Value)
					}
				}
			}
			return true
		})
		global := clockTaint(f, nil)
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Body == nil {
					continue
				}
				if strings.Contains(strings.ToLower(d.Name.Name), "seed") && readsTime(d.Body) &&
					SanctionedClockSeeds[name] != d.Name.Name {
					report(d.Pos(), ruleClock, "", "seed function "+d.Name.Name+" reads the clock")
				}
				checkClockSinks(d.Body, clockTaint(d.Body, global), report)
			case *ast.GenDecl:
				checkClockSinks(d, global, report)
			}
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].pos < hits[j].pos })
		for _, x := range hits {
			vs = append(vs, x.v)
		}
	}
	return vs
}

// clockTaint answers the names in n that hold a clock value: assigned or
// declared from an expression that reads the clock or a name already held,
// to a fixed point. inherited names stay held.
func clockTaint(n ast.Node, inherited map[string]bool) map[string]bool {
	tainted := map[string]bool{}
	for k := range inherited {
		tainted[k] = true
	}
	for changed := true; changed; {
		changed = false
		mark := func(lhs []ast.Expr, rhs []ast.Expr) {
			for i, l := range lhs {
				id, ok := l.(*ast.Ident)
				if !ok || id.Name == "_" || tainted[id.Name] {
					continue
				}
				from := rhs
				if len(rhs) == len(lhs) {
					from = rhs[i : i+1]
				}
				for _, r := range from {
					if readsClock(r, tainted) {
						tainted[id.Name], changed = true, true
						break
					}
				}
			}
		}
		ast.Inspect(n, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				mark(node.Lhs, node.Rhs)
			case *ast.ValueSpec:
				lhs := make([]ast.Expr, len(node.Names))
				for i, id := range node.Names {
					lhs[i] = id
				}
				mark(lhs, node.Values)
			case *ast.FuncDecl:
				// A function's locals are its own; the file-level pass skips
				// them.
				return false
			}
			return true
		})
	}
	return tainted
}

// checkClockSinks reports each clock value reaching a seed: a pkg/random seed
// call's argument, an element of a pkg/random composite literal, or a State
// or Seed field assignment.
func checkClockSinks(n ast.Node, tainted map[string]bool, report func(token.Pos, string, string, string)) {
	ast.Inspect(n, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if sink := sinkName(node.Fun); sink != "" {
				reported := false
				for _, arg := range node.Args {
					if readsClock(arg, tainted) {
						report(arg.Pos(), ruleClock, "", "seeds "+sink+" from the clock")
						reported = true
					}
				}
				// One clock value is one violation, not one more per literal
				// inside the argument.
				return !reported
			}
		case *ast.CompositeLit:
			if typ := randomType(node.Type); typ != "" {
				for _, elt := range node.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						elt = kv.Value
					}
					if readsClock(elt, tainted) {
						report(elt.Pos(), ruleClock, "", "seeds random."+typ+" from the clock")
					}
				}
			}
		case *ast.AssignStmt:
			for i, l := range node.Lhs {
				sel, ok := l.(*ast.SelectorExpr)
				if !ok || !generatorFields[sel.Sel.Name] {
					continue
				}
				r := node.Rhs[0]
				if len(node.Rhs) == len(node.Lhs) {
					r = node.Rhs[i]
				}
				if readsClock(r, tainted) {
					report(r.Pos(), ruleClock, "", "writes the clock into a generator's "+sel.Sel.Name)
				}
			}
		}
		return true
	})
}

// randomType answers the pkg/random type a composite literal names, or "".
func randomType(t ast.Expr) string {
	if sel, ok := t.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "random" {
			return sel.Sel.Name
		}
	}
	return ""
}

// sinkName answers the pkg/random seed call fun names, or "".
func sinkName(fun ast.Expr) string {
	switch x := fun.(type) {
	case *ast.SelectorExpr:
		if randomSinks[x.Sel.Name] {
			return x.Sel.Name
		}
	case *ast.Ident:
		if randomSinks[x.Name] {
			return x.Name
		}
	}
	return ""
}

// readsClock reports whether n reads the clock or a name holding a clock
// value. A field name after a selector's dot is not a name read.
func readsClock(n ast.Node, tainted map[string]bool) bool {
	if readsTime(n) {
		return true
	}
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			ast.Inspect(node.X, func(m ast.Node) bool {
				if id, ok := m.(*ast.Ident); ok && tainted[id.Name] {
					found = true
				}
				return !found
			})
			return false
		case *ast.KeyValueExpr:
			ast.Inspect(node.Value, func(m ast.Node) bool {
				if id, ok := m.(*ast.Ident); ok && tainted[id.Name] {
					found = true
				}
				return !found
			})
			return false
		case *ast.Ident:
			found = found || tainted[node.Name]
		}
		return !found
	})
	return found
}

// readsTime reports whether n selects from package time or calls a Unix
// time accessor.
func readsTime(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "time" {
				found = true
			}
			switch sel.Sel.Name {
			case "UnixNano", "UnixMilli", "UnixMicro":
				found = true
			}
		}
		return !found
	})
	return found
}

// LoadRandomnessSources reads every production source under
// RandomnessRoots in moduleRoot, keyed by module-relative slash path. A
// testdata directory is not a package.
func LoadRandomnessSources(moduleRoot string) (map[string]string, error) {
	out := map[string]string{}
	for _, root := range RandomnessRoots {
		dir := filepath.Join(moduleRoot, filepath.FromSlash(root))
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(moduleRoot, path)
			if err != nil {
				return err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(rel)] = string(b)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
