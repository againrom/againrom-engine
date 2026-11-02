package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The second-campaign location scan in pkg/game: slots 752..767 are cleared
// only in enterLocation, and the departures are the rows of secondDepartures
// in seconddeparture.go. It refuses a bank slice 752:768 elsewhere; in
// secondcampaign.go a case on a departure mission and a literal store into
// the departure slots (532..559, 770..789); and, outside seconddeparture.go,
// a package-level SecondExit or secondDepartureRow table and any name holding
// DepartureCases.

const (
	secondEntryFunc       = "enterLocation"
	secondDepartureFile   = "pkg/game/seconddeparture.go"
	secondCampaignFile    = "pkg/game/secondcampaign.go"
	secondDepartureRowTyp = "secondDepartureRow"
)

var secondDepartureMissions = map[string]bool{
	"10": true, "20": true, "30": true, "31": true, "40": true, "50": true,
	"60": true, "70": true, "80": true, "100": true, "110": true,
}

// CheckSecondLocation reports every second path to the location entry or to a
// departure transition. files is keyed by module-relative slash path.
func CheckSecondLocation(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg/game", Reason: "no sources scanned - the second-campaign location scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/game/") || strings.Contains(name[len("pkg/game/"):], "/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				vs = append(vs, secondLocationFunc(name, d, at)...)
			case *ast.GenDecl:
				vs = append(vs, secondLocationDecl(name, d, at)...)
			}
		}
	}
	return vs
}

func secondLocationFunc(name string, fn *ast.FuncDecl, at func(ast.Node) string) []Violation {
	var vs []Violation
	if fn.Body == nil {
		return nil
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SliceExpr:
			if intLit(n.Low) == "752" && intLit(n.High) == "768" && fn.Name.Name != secondEntryFunc {
				vs = append(vs, Violation{From: at(n), Reason: "slots 752..767 are cleared here; call " + secondEntryFunc})
			}
		case *ast.CaseClause:
			if name != secondCampaignFile {
				break
			}
			for _, e := range n.List {
				if secondDepartureMissions[intLit(e)] {
					vs = append(vs, Violation{From: at(n), Reason: "a departure is declared as a case here; add a row to secondDepartures"})
					break
				}
			}
		case *ast.AssignStmt:
			if name != secondCampaignFile {
				break
			}
			for _, l := range n.Lhs {
				ix, ok := l.(*ast.IndexExpr)
				if !ok {
					continue
				}
				if sel, ok := ix.X.(*ast.SelectorExpr); !ok || sel.Sel.Name != "bank" {
					continue
				}
				if v, err := strconv.Atoi(intLit(ix.Index)); err == nil && (v >= 532 && v <= 559 || v >= 770 && v <= 789) {
					vs = append(vs, Violation{From: at(n), Reason: "a departure write is stored here; add it to the row's writes"})
				}
			}
		}
		return true
	})
	return vs
}

func secondLocationDecl(name string, d *ast.GenDecl, at func(ast.Node) string) []Violation {
	var vs []Violation
	if d.Tok != token.VAR && d.Tok != token.CONST {
		return nil
	}
	for _, s := range d.Specs {
		vspec, ok := s.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, id := range vspec.Names {
			if strings.Contains(id.Name, "DepartureCases") {
				vs = append(vs, Violation{From: at(id), Reason: "a second departure table; the rows of secondDepartures are the only declaration"})
			}
		}
		if name == secondDepartureFile {
			continue
		}
		held := false
		for _, v := range vspec.Values {
			ast.Inspect(v, func(n ast.Node) bool {
				if t, ok := n.(*ast.Ident); ok && (t.Name == "SecondExit" || t.Name == secondDepartureRowTyp) {
					held = true
				}
				return !held
			})
		}
		if held {
			vs = append(vs, Violation{From: at(vspec), Reason: "a second departure table; the rows of secondDepartures are the only declaration"})
		}
	}
	return vs
}

func intLit(e ast.Expr) string {
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.INT {
		return lit.Value
	}
	return ""
}
