package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// SpellArmTableFile holds the one table of spell-arm capabilities. Outside
// it, production pkg/ code asks a row and never tests an arm's identity.
const SpellArmTableFile = "pkg/sim/spellarms.go"

// The shapes of a spell-arm identity test SpellArmSites names.
const (
	ShapeArmCompare = "compare"     // == or != of an arm expression with a literal or const
	ShapeArmSwitch  = "switch"      // a case literal or const of a switch on an arm expression
	ShapeArmLookup  = "lookup"      // a literal or const arm handed to an arm lookup
	ShapeIDConst    = "id-const"    // a comparison with a const named ...SpellID
	ShapeArmName    = "const-name"  // a const named ...SpellID or ...Arm
	ShapeArmTable   = "arm-table"   // a package-level table named ...Arms
	ShapeArmField   = "field-match" // == or != of a row's Arm field with a literal or const
	ShapeIDMatch    = "id-match"    // a spell row's ID compared with, or found in, literals or consts
	ShapeBookID     = "book-id"     // a literal or const id handed to a book spell lookup
	ShapeIDTable    = "id-table"    // a pkg/game table of spell answers keyed by literal ids
)

// SpellArmAllowList names the sites the guard admits, keyed "file shape",
// with their count and reason.
var SpellArmAllowList = map[string]struct {
	Count  int
	Reason string
}{
	"pkg/sim/creaturecast.go arm-table": {1, "the second game's creature aims are keyed by row id with no Spells " +
		"column read, and its ids 5 and 16 run no arm, so an arm row cannot carry their aims"},
}

// bookLookups are the calls taking a book spell id, with its argument index.
var bookLookups = map[string]int{"bookSpell": 1, "bookSpellRefusal": 2, "beginBookSpellOnce": 2}

// armCalls are the calls whose result is an arm: SpellRule.arm and ArmID,
// World.spellArm and SpellArm, and pkg/game's own row lookup.
var armCalls = map[string]bool{"arm": true, "ArmID": true, "spellArm": true, "SpellArm": true, "spellArmOf": true}

// armLookups are the calls taking an arm, with the arm's argument index.
var armLookups = map[string]int{
	"armSpellID": 0, "spellIDOfArm": 0, "HasEffectArm": 1, "spellArmIs": 1, "AutoCastLevelOfArm": 0,
	"autoCastAt": 2, "spellArmOf": 0, "MagnitudeArm": 0, "DurationArm": 0, "runsArm": 0,
}

// SpellArmSites reports every spell-arm identity test outside the table
// file, one finding per site, less the allow list's admitted counts.
func SpellArmSites(files map[string]string) []Violation {
	allow := map[string]int{}
	for key, a := range SpellArmAllowList {
		allow[key] = a.Count
	}
	return spellArmSitesAllowing(files, allow)
}

func spellArmSitesAllowing(files map[string]string, allow map[string]int) []Violation {
	type parsed struct {
		fset *token.FileSet
		f    *ast.File
	}
	var vs []Violation
	trees := map[string]parsed{}
	consts := map[string]bool{}
	for _, name := range sortedNames(files) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		trees[name] = parsed{fset, f}
		ast.Inspect(f, func(n ast.Node) bool {
			if g, ok := n.(*ast.GenDecl); ok && g.Tok == token.CONST {
				for _, s := range g.Specs {
					for _, id := range s.(*ast.ValueSpec).Names {
						consts[id.Name] = true
					}
				}
			}
			return true
		})
	}
	var fixed func(e ast.Expr) bool
	fixed = func(e ast.Expr) bool {
		switch x := ast.Unparen(e).(type) {
		case *ast.BasicLit:
			return x.Kind == token.INT
		case *ast.UnaryExpr:
			return (x.Op == token.SUB || x.Op == token.XOR) && fixed(x.X)
		case *ast.Ident:
			return consts[x.Name]
		case *ast.SelectorExpr:
			_, pkg := x.X.(*ast.Ident)
			return pkg && consts[x.Sel.Name]
		case *ast.CallExpr:
			// A typed literal such as uint16(14).
			if fn, ok := x.Fun.(*ast.Ident); ok && integerTypes[fn.Name] && len(x.Args) == 1 {
				_, lit := ast.Unparen(x.Args[0]).(*ast.BasicLit)
				return lit
			}
		}
		return false
	}
	idConst := func(e ast.Expr) bool {
		switch x := ast.Unparen(e).(type) {
		case *ast.Ident:
			return strings.HasSuffix(x.Name, "SpellID") && consts[x.Name]
		case *ast.SelectorExpr:
			_, pkg := x.X.(*ast.Ident)
			return pkg && strings.HasSuffix(x.Sel.Name, "SpellID") && consts[x.Sel.Name]
		}
		return false
	}
	counts := map[string]int{}
	var found []Violation
	for _, name := range sortedNames(files) {
		p, ok := trees[name]
		if !ok || name == SpellArmTableFile {
			continue
		}
		at := func(n ast.Node, shape, why string) {
			counts[name+" "+shape]++
			found = append(found, Violation{From: name + ":" + strconv.Itoa(p.fset.Position(n.Pos()).Line),
				Reason: shape + ": " + why + "; ask a capability of a row of " + SpellArmTableFile})
		}
		inSim := strings.HasPrefix(name, "pkg/sim/") && !strings.Contains(strings.TrimPrefix(name, "pkg/sim/"), "/")
		for _, d := range p.f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && (g.Tok == token.VAR || g.Tok == token.CONST) && inSim && name != AreaArmTableFile {
				for _, s := range g.Specs {
					spec := s.(*ast.ValueSpec)
					for i, id := range spec.Names {
						if strings.HasSuffix(id.Name, "Arms") || i < len(spec.Values) && sparseArray(spec.Values[i]) {
							at(id, ShapeArmTable, id.Name+" is a table keyed by arm")
						}
					}
				}
			}
		}
		inGame := strings.HasPrefix(name, "pkg/game/") && !strings.Contains(strings.TrimPrefix(name, "pkg/game/"), "/")
		for _, d := range p.f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.VAR && inGame {
				for _, s := range g.Specs {
					spec := s.(*ast.ValueSpec)
					for i, id := range spec.Names {
						if i < len(spec.Values) && spellIDTable(spec.Values[i]) {
							at(id, ShapeIDTable, id.Name+" answers a spell rule by literal ids")
						}
					}
				}
			}
		}
		ast.Inspect(p.f, func(n ast.Node) bool {
			if g, ok := n.(*ast.GenDecl); ok && g.Tok == token.CONST {
				for _, s := range g.Specs {
					for _, id := range s.(*ast.ValueSpec).Names {
						if strings.HasSuffix(id.Name, "SpellID") || strings.HasSuffix(id.Name, "Arm") {
							at(id, ShapeArmName, "const "+id.Name+" names a spell or arm")
						}
					}
				}
			}
			return true
		})
		for _, d := range p.f.Decls {
			scanArmBody(d, inSim, fixed, idConst, at)
		}
	}
	for _, v := range found {
		file, _, _ := strings.Cut(v.From, ":")
		shape, _, _ := strings.Cut(v.Reason, ":")
		if limit, ok := allow[file+" "+shape]; ok && counts[file+" "+shape] <= limit {
			continue
		}
		vs = append(vs, v)
	}
	for _, key := range sortedNames(allow) {
		file, _, _ := strings.Cut(key, " ")
		if _, scanned := files[file]; scanned && counts[key] < allow[key] {
			vs = append(vs, Violation{From: key, Reason: fmt.Sprintf("%d sites, allowed %d: lower SpellArmAllowList", counts[key], allow[key])})
		}
	}
	return vs
}

// sparseArray reports an array or slice literal keyed by integer literals
// whose values are names or functions, the shape of one capability indexed
// by arm.
func sparseArray(e ast.Expr) bool {
	c, ok := ast.Unparen(e).(*ast.CompositeLit)
	if !ok {
		return false
	}
	if _, ok := c.Type.(*ast.ArrayType); !ok {
		return false
	}
	keyed := false
	for _, el := range c.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			return false
		}
		if _, ok := ast.Unparen(kv.Value).(*ast.CompositeLit); ok {
			return false
		}
		if _, ok := intLiteral(kv.Key); ok {
			keyed = true
		}
	}
	return keyed
}

// spellIDTable reports a package-level value that reads a spell rule and
// holds a list of integer literals or a table keyed by them: spell answers
// chosen by literal ids.
func spellIDTable(e ast.Expr) bool {
	rule, ids := false, false
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			rule = rule || x.Sel.Name == "SpellRule"
		case *ast.Ident:
			rule = rule || x.Name == "SpellRule"
		case *ast.CompositeLit:
			ids = ids || literalIDs(x)
		}
		return true
	})
	return rule && ids
}

// literalIDs reports a slice, array or map literal whose elements, or whose
// keys, are all integer literals.
func literalIDs(c *ast.CompositeLit) bool {
	switch c.Type.(type) {
	case *ast.ArrayType, *ast.MapType:
	default:
		return false
	}
	if len(c.Elts) == 0 {
		return false
	}
	for _, el := range c.Elts {
		if kv, ok := el.(*ast.KeyValueExpr); ok {
			el = kv.Key
		}
		if _, ok := intLiteral(el); !ok {
			return false
		}
	}
	return true
}

// scanArmBody names the identity tests of one declaration. A variable
// assigned from an arm call anywhere in it is an arm expression, and so is a
// name arm in the simulation; an Arm field is a spell row's when the value
// holding it is named for a spell or rule.
func scanArmBody(body ast.Node, inSim bool, fixed, idConst func(ast.Expr) bool, at func(ast.Node, string, string)) {
	vars := map[string]bool{}
	var isArm func(e ast.Expr) bool
	isArm = func(e ast.Expr) bool {
		switch x := ast.Unparen(e).(type) {
		case *ast.CallExpr:
			if fn, ok := x.Fun.(*ast.Ident); ok && integerTypes[fn.Name] && len(x.Args) == 1 {
				return isArm(x.Args[0])
			}
			return armCalls[calleeName(x)]
		case *ast.Ident:
			return inSim && x.Name == "arm" || vars[x.Name]
		}
		return false
	}
	var spellNamed func(e ast.Expr) bool
	spellNamed = func(e ast.Expr) bool {
		switch x := ast.Unparen(e).(type) {
		case *ast.Ident:
			n := strings.ToLower(x.Name)
			return strings.Contains(n, "spell") || strings.Contains(n, "rule")
		case *ast.SelectorExpr:
			return spellNamed(x.Sel)
		case *ast.IndexExpr:
			return spellNamed(x.X)
		}
		return false
	}
	armField := func(e ast.Expr) bool {
		x, ok := ast.Unparen(e).(*ast.SelectorExpr)
		return ok && x.Sel.Name == "Arm" && spellNamed(x.X)
	}
	idField := func(e ast.Expr) bool {
		x, ok := ast.Unparen(e).(*ast.SelectorExpr)
		return ok && x.Sel.Name == "ID" && spellNamed(x.X)
	}
	// spellID is a literal or const naming a spell; zero is the empty row.
	spellID := func(e ast.Expr) bool {
		v, lit := intLiteral(e)
		return fixed(e) && !(lit && v == 0)
	}
	fixedList := func(e ast.Expr) bool {
		c, ok := ast.Unparen(e).(*ast.CompositeLit)
		if !ok || len(c.Elts) == 0 {
			return false
		}
		for _, el := range c.Elts {
			if !spellID(el) {
				return false
			}
		}
		return true
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if len(x.Lhs) == len(x.Rhs) {
				for i, r := range x.Rhs {
					if id, ok := x.Lhs[i].(*ast.Ident); ok && isArm(r) {
						vars[id.Name] = true
					}
				}
			}
		case *ast.ValueSpec:
			for i, r := range x.Values {
				if i < len(x.Names) && isArm(r) {
					vars[x.Names[i].Name] = true
				}
			}
		}
		return true
	})
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BinaryExpr:
			switch x.Op {
			case token.LSS, token.GTR, token.LEQ, token.GEQ:
				if idConst(x.X) || idConst(x.Y) {
					at(x, ShapeIDConst, "a spell id is bounded by a named id")
				}
				return true
			case token.EQL, token.NEQ:
			default:
				return true
			}
			switch {
			case isArm(x.X) && fixed(x.Y) || isArm(x.Y) && fixed(x.X):
				at(x, ShapeArmCompare, "an arm is compared with a literal or const")
			case armField(x.X) && fixed(x.Y) || armField(x.Y) && fixed(x.X):
				at(x, ShapeArmField, "an Arm field is compared with a literal or const")
			case idConst(x.X) || idConst(x.Y):
				at(x, ShapeIDConst, "a spell id is compared with a named id")
			case idField(x.X) && spellID(x.Y) || idField(x.Y) && spellID(x.X):
				at(x, ShapeIDMatch, "a spell row's ID is compared with a literal or const")
			}
		case *ast.SwitchStmt:
			armTag := x.Tag != nil && (isArm(x.Tag) || armField(x.Tag))
			for _, s := range x.Body.List {
				for _, e := range s.(*ast.CaseClause).List {
					switch {
					case armTag && fixed(e):
						at(e, ShapeArmSwitch, "a switch on an arm names a literal or const case")
					case x.Tag != nil && idConst(e):
						at(e, ShapeArmSwitch, "a switch names a spell id case")
					}
				}
			}
		case *ast.CallExpr:
			name := calleeName(x)
			i, ok := armLookups[name]
			if !ok && strings.HasPrefix(name, "SpellArm") && name != "SpellArm" {
				i, ok = 0, true
			}
			if ok && i < len(x.Args) && (fixed(x.Args[i]) || idConst(x.Args[i])) {
				at(x.Args[i], ShapeArmLookup, name+" is handed a literal or const arm")
			}
			if i, ok := bookLookups[name]; ok && i < len(x.Args) && (fixed(x.Args[i]) || idConst(x.Args[i])) {
				at(x.Args[i], ShapeBookID, name+" is handed a literal or const spell id")
			}
			if name == "Contains" && len(x.Args) == 2 && fixedList(x.Args[0]) && idField(x.Args[1]) {
				at(x, ShapeIDMatch, "a spell row's ID is found in a literal list")
			}
		}
		return true
	})
}
