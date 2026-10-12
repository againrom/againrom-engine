package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// AreaArmTableFile holds the one table of area-arm properties. Elsewhere in
// production pkg/ code an arm site, a comparison or switch case of an arm or
// row expression (arm, spellArm, SpellArm, spellArmIs, or a name arm, spell
// or id) against an arm the table names, may only fall in count per file.
// The scan is lexical: an arm in another variable or argument is not seen.
const AreaArmTableFile = "pkg/sim/areaarms.go"

// SpellValuesFile holds the one range and one duration function and every
// formula they read. No other production pkg/ file names its other
// functions, computes with a range or duration column, or reads a range or
// duration formula entry.
const SpellValuesFile = "pkg/sim/spellvalues.go"

var spellValueEntries = map[string]bool{"spellRangeOf": true, "spellDurationOf": true}

var spellValueColumns = map[string]bool{"MaxRange": true, "SpellDuration": true, "AreaDuration": true}

var spellValueFormulas = map[string]bool{"FormulaRange": true, "FormulaDuration": true}

var integerTypes = map[string]bool{"int": true, "int16": true, "int32": true, "int64": true, "uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true}

var armExprNames = map[string]bool{"arm": true, "spell": true, "id": true, "spellid": true}

// AreaArmIDs reads every arm key of the table file.
func AreaArmIDs(table string) (map[int64]bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), AreaArmTableFile, table, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	out := map[int64]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "arm" {
			if v, ok := intLiteral(kv.Value); ok {
				out[v] = true
			}
		}
		return true
	})
	if len(out) == 0 {
		return nil, fmt.Errorf("%s names no arm", AreaArmTableFile)
	}
	return out, nil
}

func intLiteral(e ast.Expr) (int64, bool) {
	lit, ok := ast.Unparen(e).(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	v, err := strconv.ParseInt(lit.Value, 0, 64)
	return v, err == nil
}

func armExpr(e ast.Expr) bool {
	switch x := ast.Unparen(e).(type) {
	case *ast.CallExpr:
		if fn, ok := x.Fun.(*ast.Ident); ok && integerTypes[fn.Name] && len(x.Args) == 1 {
			// A conversion such as uint16(arm).
			return armExpr(x.Args[0])
		}
		switch calleeName(x) {
		case "spellArm", "SpellArm", "arm":
			return true
		}
	case *ast.Ident:
		return armExprNames[strings.ToLower(x.Name)]
	case *ast.SelectorExpr:
		return armExprNames[strings.ToLower(x.Sel.Name)]
	}
	return false
}

func calleeName(c *ast.CallExpr) string {
	switch fn := c.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}
	return ""
}

// AreaArmSites counts each production file's arm sites outside the table.
func AreaArmSites(files map[string]string, arms map[int64]bool) (map[string]int, []Violation) {
	out := map[string]int{}
	var vs []Violation
	for _, name := range sortedNames(files) {
		if name == AreaArmTableFile {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		named := func(e ast.Expr) bool {
			v, ok := intLiteral(e)
			return ok && arms[v]
		}
		n := 0
		ast.Inspect(f, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.BinaryExpr:
				if (x.Op == token.EQL || x.Op == token.NEQ) && (armExpr(x.X) && named(x.Y) || armExpr(x.Y) && named(x.X)) {
					n++
				}
			case *ast.SwitchStmt:
				if x.Tag == nil || !armExpr(x.Tag) {
					return true
				}
				for _, s := range x.Body.List {
					for _, e := range s.(*ast.CaseClause).List {
						if named(e) {
							n++
						}
					}
				}
			case *ast.CallExpr:
				if calleeName(x) == "spellArmIs" && len(x.Args) == 2 && named(x.Args[1]) {
					n++
				}
			}
			return true
		})
		if n != 0 {
			out[name] = n
		}
	}
	return out, vs
}

// CheckAreaArmSites reports a count above or below its committed one.
func CheckAreaArmSites(got, committed map[string]int) []Violation {
	var vs []Violation
	for _, name := range sortedNames(got) {
		switch want := committed[name]; {
		case got[name] > want:
			vs = append(vs, Violation{From: name, Reason: fmt.Sprintf("%d arm sites, committed %d: an area arm's property belongs in a row of %s", got[name], want, AreaArmTableFile)})
		case got[name] < want:
			vs = append(vs, Violation{From: name, Reason: fmt.Sprintf("%d arm sites, committed %d: lower CommittedAreaArmSites", got[name], want)})
		}
	}
	for _, name := range sortedNames(committed) {
		if _, ok := got[name]; !ok {
			vs = append(vs, Violation{From: name, Reason: fmt.Sprintf("0 arm sites, committed %d: drop the row from CommittedAreaArmSites", committed[name])})
		}
	}
	return vs
}

// CheckSpellValues reports every range or duration formula outside its file.
func CheckSpellValues(files map[string]string) []Violation {
	values, ok := files[SpellValuesFile]
	if !ok {
		return []Violation{{From: SpellValuesFile, Reason: "the spell value file is missing"}}
	}
	vf, err := parser.ParseFile(token.NewFileSet(), SpellValuesFile, values, parser.SkipObjectResolution)
	if err != nil {
		return []Violation{{From: SpellValuesFile, Reason: "source does not parse: " + err.Error()}}
	}
	leaves := map[string]bool{}
	for _, d := range vf.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && !spellValueEntries[fn.Name.Name] {
			leaves[fn.Name.Name] = true
		}
	}
	var vs []Violation
	for entry := range spellValueEntries {
		if !strings.Contains(values, "func "+entry+"(") {
			vs = append(vs, Violation{From: SpellValuesFile, Reason: "does not declare " + entry})
		}
	}
	for _, name := range sortedNames(files) {
		if name == SpellValuesFile {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		inSim := strings.HasPrefix(name, "pkg/sim/") && !strings.Contains(strings.TrimPrefix(name, "pkg/sim/"), "/")
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				if inSim && leaves[x.Name] {
					vs = append(vs, Violation{From: at(x), Reason: "names " + x.Name + "; read a range or duration through spellRangeOf or spellDurationOf"})
				}
			case *ast.BinaryExpr:
				switch x.Op {
				case token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.SHL, token.SHR:
					if column := spellValueColumn(x.X); column != "" {
						vs = append(vs, Violation{From: at(x), Reason: "computes with " + column + "; the formula belongs in " + SpellValuesFile})
					} else if column := spellValueColumn(x.Y); column != "" {
						vs = append(vs, Violation{From: at(x), Reason: "computes with " + column + "; the formula belongs in " + SpellValuesFile})
					}
				}
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "SpellFormulaAt" && len(x.Args) > 0 {
					if k, ok := x.Args[0].(*ast.SelectorExpr); ok && spellValueFormulas[k.Sel.Name] {
						vs = append(vs, Violation{From: at(x), Reason: "reads the " + k.Sel.Name + " table; the formula belongs in " + SpellValuesFile})
					}
				}
			}
			return true
		})
	}
	return vs
}

// spellValueColumn is the range or duration column an operand reads, or "".
func spellValueColumn(e ast.Expr) string {
	switch x := ast.Unparen(e).(type) {
	case *ast.SelectorExpr:
		if spellValueColumns[x.Sel.Name] {
			return x.Sel.Name
		}
	case *ast.CallExpr:
		if fn, ok := x.Fun.(*ast.Ident); ok && integerTypes[fn.Name] && len(x.Args) == 1 {
			return spellValueColumn(x.Args[0])
		}
	}
	return ""
}
