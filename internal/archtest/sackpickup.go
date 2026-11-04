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

// The sack pickup scan: one World transaction in pkg/sim/sackpickup.go builds
// every sack pickup (the order's admission, the arrival test, the take, the
// quest documents' move and the completion). Its internals are called only
// from their transaction functions; outside pkg/sim no production code calls
// them, declares a take of its own, builds the pickup command, writes the
// ground sack list or tests arrival by hand. The documents' holder is the
// one starting-hero lookup's.
const (
	sackPickupFile = "pkg/sim/sackpickup.go"
	sackPickupTake = "pickUpSack"
)

// sackPickupCallers names, for each internal of the transaction, the
// functions of sackpickup.go that may call it.
var sackPickupCallers = map[string][]string{
	"TakeSack":           {sackPickupTake},
	"MoveCarried":        {sackPickupTake},
	"CompleteSackPickup": {sackPickupTake, "SettleSackPickup"},
	"cancelSackPickup":   {"SettleSackPickup"},
	"CancelSackPickup":   nil,
	"sackPickupArrived":  {"PickUpSackUnderfoot", "SettleSackPickup", "SackPickupArrived"},
	"SackPickupArrived":  nil,
	sackPickupTake:       {"PickUpSackUnderfoot", "SettleSackPickup", "CheatPickupAll"},
	"cheatPickupEach":    {"CheatPickupAll"},
}

// sackPickupEntries are the named entries: a function outside pkg/sim that
// keeps a shape the scan refuses elsewhere, with its reason.
var sackPickupEntries = map[string]string{
	// A LOAD's ground import replaces the World's sack list; it takes nothing.
	"pkg/game/originalground.go applyOriginalGround ground": "",
	// The tool's walk stops on crossing and centre alone. The arrival test also
	// waits for the strike cycle and order progress, so calling it would change
	// the ticks the tool spends and prints; the take is Mission.PickUpUnderfoot's.
	"cmd/missionrun/main.go pickUpAt arrival": "",
}

// LoadSackPickupSources returns the non-test Go sources under pkg and cmd,
// keyed by module-relative slash path.
func LoadSackPickupSources(moduleRoot string) (map[string]string, error) {
	out := make(map[string]string)
	for _, top := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(moduleRoot, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "testdata" {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(moduleRoot, path)
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

// CheckSackPickup reports every second builder of a sack pickup in files.
func CheckSackPickup(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the sack pickup scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	takes := 0
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		inSim := strings.HasPrefix(name, "pkg/sim/") && !strings.Contains(name[len("pkg/sim/"):], "/")
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if inSim {
				if name == sackPickupFile && fn.Name.Name == sackPickupTake {
					takes++
				}
				vs = append(vs, sackPickupSimCalls(name, fn, at)...)
				continue
			}
			vs = append(vs, sackPickupOutside(name, fn, at)...)
		}
	}
	if takes != 1 {
		vs = append(vs, Violation{From: sackPickupFile, Reason: "the sack pickup scan found " + strconv.Itoa(takes) + " " + sackPickupTake + " functions, want one"})
	}
	return vs
}

// sackPickupSimCalls reports a call of a transaction internal from a pkg/sim
// function that is not one of its transaction callers.
func sackPickupSimCalls(name string, fn *ast.FuncDecl, at func(ast.Node) string) []Violation {
	var vs []Violation
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		callee := calleeName(call)
		callers, restricted := sackPickupCallers[callee]
		if !restricted {
			return true
		}
		allowed := false
		for _, c := range callers {
			allowed = allowed || name == sackPickupFile && fn.Name.Name == c
		}
		if !allowed {
			vs = append(vs, Violation{From: at(call), Reason: fn.Name.Name + " calls " + callee + "; only the sack pickup transaction in " + sackPickupFile + " calls it"})
		}
		return true
	})
	return vs
}

// sackPickupOutside reports the shapes of a second pickup in a function
// outside pkg/sim.
func sackPickupOutside(name string, fn *ast.FuncDecl, at func(ast.Node) string) []Violation {
	var vs []Violation
	entry := func(shape string) bool {
		_, ok := sackPickupEntries[name+" "+fn.Name.Name+" "+shape]
		return ok
	}
	lower := strings.ToLower(fn.Name.Name)
	if strings.HasPrefix(lower, "takesack") || strings.HasPrefix(lower, "completesackpickup") || lower == "sackpickuparrived" {
		vs = append(vs, Violation{From: at(fn), Reason: fn.Name.Name + " is a second sack take or arrival test; call the World's sack pickup"})
	}
	fine, centre, transit := false, false, false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			callee := calleeName(n)
			if _, internal := sackPickupCallers[callee]; internal {
				vs = append(vs, Violation{From: at(n), Reason: fn.Name.Name + " calls " + callee + "; only the World's sack pickup calls it"})
			}
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "PickUp" && isIdent(sel.X, "sim") {
				vs = append(vs, Violation{From: at(n), Reason: fn.Name.Name + " builds the pickup command; queue OrderSackPickup's order"})
			}
			if callee == "ReplaceGroundSacks" && !entry("ground") {
				vs = append(vs, Violation{From: at(n), Reason: fn.Name.Name + " writes the World's sack list; a take is the World's sack pickup"})
			}
			if callee == "ActorFinePosition" {
				fine = true
			}
			if callee == "primaryPlayerHero" && fn.Name.Name != "startingHeroIndex" && insideLoop(fn.Body, n) {
				vs = append(vs, Violation{From: at(n), Reason: fn.Name.Name + " looks the starting hero up in a loop; call startingHeroIndex"})
			}
		case *ast.BinaryExpr:
			if lit, ok := n.Y.(*ast.BasicLit); ok && lit.Value == "128" && (n.Op == token.EQL || n.Op == token.NEQ) {
				centre = true
			}
		case *ast.SelectorExpr:
			if n.Sel.Name == "Transit" {
				transit = true
			}
		case *ast.CompositeLit:
			if sel, ok := n.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "SackDocuments" && isIdent(sel.X, "sim") && fn.Name.Name != "questDocuments" {
				vs = append(vs, Violation{From: at(n), Reason: fn.Name.Name + " names the quest documents' holder; call questDocuments"})
			}
		}
		return true
	})
	if fine && centre && transit && !entry("arrival") {
		vs = append(vs, Violation{From: at(fn), Reason: fn.Name.Name + " tests arrival by hand; the World's sack pickup applies the arrival test"})
	}
	return vs
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

// insideLoop reports whether n lies in the body of a for or range statement
// of body.
func insideLoop(body *ast.BlockStmt, n ast.Node) bool {
	inside := false
	ast.Inspect(body, func(m ast.Node) bool {
		var loop *ast.BlockStmt
		switch m := m.(type) {
		case *ast.ForStmt:
			loop = m.Body
		case *ast.RangeStmt:
			loop = m.Body
		}
		if loop != nil && loop.Pos() <= n.Pos() && n.End() <= loop.End() {
			inside = true
		}
		return !inside
	})
	return inside
}
