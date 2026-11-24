package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// profileRuleBoolDebt lists pre-existing profile bool fields whose readers
// still branch into two bodies. A count may only fall; remove an entry once
// its field becomes a rule row.
var profileRuleBoolDebt = map[string]int{}

// TestProfileRulesAreRows guards that profile rules whose readers hold two
// bodies live as rule rows with function fields rather than bool flags.
func TestProfileRulesAreRows(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["pkg/sim/gameprofile.go"]; !ok {
		t.Fatal("pkg/sim/gameprofile.go is missing; the scan guards nothing")
	}
	flag := regexp.MustCompile(`\b(stopRefusedFlee|classRetreatMode|roundUphillTowardZero|countedOutcome|autoCastSideSight|savedProjectileAim|withdrawMoverPoint|victimHolds|raiseAtNext|packedStep|effectsFirst|nativeTick|nativeDecay|cellCastReach|recoveryLowByte|UnitLifeZero)\b`)
	presentFlag := regexp.MustCompile(`\b(PointLift|UnitLifeZero)\b`)
	for path, src := range files {
		if strings.HasPrefix(path, "pkg/sim/") {
			if m := flag.FindString(src); m != "" {
				t.Errorf("%s names %s, a flag that chose a rule's body; hold the rule as a row", path, m)
			}
		}
		if strings.HasPrefix(path, "pkg/data/") || strings.HasPrefix(path, "pkg/game/") || strings.HasPrefix(path, "pkg/ui/") {
			if m := presentFlag.FindString(src); m != "" {
				t.Errorf("%s names %s, a flag that chose a rule's body; hold the rule as a row", path, m)
			}
		}
	}

	m, err := loadTypeCheckedModule(root)
	if err != nil {
		t.Fatal(err)
	}
	cp := m.Packages[simPath]
	if cp == nil {
		t.Fatalf("package %s is missing; the scan guards nothing", simPath)
	}
	found := findProfileRuleBoolFindings(m.Fset, cp)
	for _, v := range checkProfileRuleBools(found, profileRuleBoolDebt) {
		t.Error(v)
	}

	t.Run("synthetic", func(t *testing.T) {
		checkSyntheticProfileRuleBools(t)
	})
}

type profileRuleBoolFinding struct {
	Field string
	File  string
	Line  int
	Why   string
}

func (f profileRuleBoolFinding) String() string {
	return fmt.Sprintf("%s:%d %s: %s", f.File, f.Line, f.Field, f.Why)
}

func checkProfileRuleBools(found []profileRuleBoolFinding, debt map[string]int) []string {
	var out []string
	held := map[string]int{}
	for _, f := range found {
		if _, ok := debt[f.Field]; ok {
			held[f.Field]++
			continue
		}
		out = append(out, f.String()+"; hold the rule as a row with function fields")
	}
	names := make([]string, 0, len(debt))
	for name := range debt {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		switch got := held[name]; {
		case got > debt[name]:
			out = append(out, fmt.Sprintf("%s holds %d two-body reads, above debt %d", name, got, debt[name]))
		case got < debt[name]:
			out = append(out, fmt.Sprintf("%s holds %d two-body reads, below debt %d; lower profileRuleBoolDebt", name, got, debt[name]))
		}
	}
	return out
}

func findProfileRuleBoolFindings(fset *token.FileSet, cp *CheckedPackage) []profileRuleBoolFinding {
	pkg := packageOfChecked(cp)
	if pkg == nil {
		return nil
	}
	profileObj := pkg.Scope().Lookup("gameProfile")
	if profileObj == nil {
		return nil
	}
	profileNamed, ok := profileObj.Type().(*types.Named)
	if !ok {
		return nil
	}

	profileTypes := map[*types.Named]bool{}
	ruleIndexTypes := map[*types.Named]bool{}
	profileFields := map[types.Object]bool{}
	var collectProfile func(n *types.Named)
	collectProfile = func(n *types.Named) {
		if n == nil || profileTypes[n] || n.Obj().Pkg() != pkg {
			return
		}
		st, ok := n.Underlying().(*types.Struct)
		if !ok {
			return
		}
		profileTypes[n] = true
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			profileFields[f] = true
			ft := f.Type()
			if ptr, ok := ft.(*types.Pointer); ok {
				ft = ptr.Elem()
			}
			named, ok := ft.(*types.Named)
			if !ok || named.Obj().Pkg() != pkg {
				continue
			}
			switch named.Underlying().(type) {
			case *types.Struct:
				collectProfile(named)
			case *types.Basic:
				ruleIndexTypes[named] = true
			}
		}
	}
	collectProfile(profileNamed)

	// Rule-row structs are element structs of tables indexed by a profile rule field.
	ruleRowTypes := map[*types.Named]bool{}
	for _, f := range cp.Files[:cp.NumProd] {
		ast.Inspect(f, func(n ast.Node) bool {
			ix, ok := n.(*ast.IndexExpr)
			if !ok {
				return true
			}
			indexedByProfile := false
			if named, ok := cp.Info.TypeOf(ix.Index).(*types.Named); ok && ruleIndexTypes[named] {
				indexedByProfile = true
			}
			if sel, ok := ast.Unparen(ix.Index).(*ast.SelectorExpr); ok {
				if s := cp.Info.Selections[sel]; s != nil && profileFields[s.Obj()] {
					indexedByProfile = true
				}
			}
			if !indexedByProfile {
				return true
			}
			var elem types.Type
			switch u := typeUnder(cp.Info.TypeOf(ix.X)).(type) {
			case *types.Array:
				elem = u.Elem()
			case *types.Slice:
				elem = u.Elem()
			default:
				return true
			}
			if named, ok := elem.(*types.Named); ok && named.Obj().Pkg() == pkg {
				if _, ok := named.Underlying().(*types.Struct); ok {
					ruleRowTypes[named] = true
				}
			}
			return true
		})
	}

	var out []profileRuleBoolFinding
	trackedBools := map[types.Object]string{}
	recordBools := func(n *types.Named, isRow bool) {
		st := n.Underlying().(*types.Struct)
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			b, ok := typeUnder(f.Type()).(*types.Basic)
			if !ok || b.Kind() != types.Bool {
				continue
			}
			key := n.Obj().Name() + "." + f.Name()
			trackedBools[f] = key
			if isRow {
				pos := fset.Position(f.Pos())
				out = append(out, profileRuleBoolFinding{
					Field: key,
					File:  pos.Filename,
					Line:  pos.Line,
					Why:   "rule row carries a bool field",
				})
			}
		}
	}
	for n := range profileTypes {
		recordBools(n, false)
	}
	for n := range ruleRowTypes {
		recordBools(n, true)
	}

	for i, f := range cp.Files[:cp.NumProd] {
		rel := cp.RelPaths[i]
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Body != nil {
					out = append(out, scanBodyForProfileTwoBodyBools(fset, rel, x.Body, cp.Info, pkg, trackedBools, ruleRowTypes)...)
				}
			case *ast.FuncLit:
				if x.Body != nil {
					out = append(out, scanBodyForProfileTwoBodyBools(fset, rel, x.Body, cp.Info, pkg, trackedBools, ruleRowTypes)...)
				}
			}
			return true
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Field != out[j].Field {
			return out[i].Field < out[j].Field
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

func packageOfChecked(cp *CheckedPackage) *types.Package {
	for _, obj := range cp.Info.Defs {
		if obj != nil && obj.Pkg() != nil {
			return obj.Pkg()
		}
	}
	return nil
}

func scanBodyForProfileTwoBodyBools(fset *token.FileSet, rel string, body *ast.BlockStmt, info *types.Info, pkg *types.Package, tracked map[types.Object]string, ruleRows map[*types.Named]bool) []profileRuleBoolFinding {
	localBools := map[types.Object]string{}
	fieldOfExpr := func(e ast.Expr) []string {
		var keys []string
		ast.Inspect(e, func(n ast.Node) bool {
			if _, isLit := n.(*ast.FuncLit); isLit {
				return false
			}
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if sel := info.Selections[x]; sel != nil {
					if key, ok := tracked[sel.Obj()]; ok && !slices.Contains(keys, key) {
						keys = append(keys, key)
					}
				}
			case *ast.Ident:
				if key, ok := localBools[info.Uses[x]]; ok && !slices.Contains(keys, key) {
					keys = append(keys, key)
				}
			}
			return true
		})
		return keys
	}
	recordAlias := func(lhs, rhs []ast.Expr) {
		if len(lhs) != len(rhs) {
			return
		}
		for i, l := range lhs {
			id, ok := l.(*ast.Ident)
			if !ok {
				continue
			}
			b, ok := typeUnder(info.TypeOf(rhs[i])).(*types.Basic)
			if !ok || b.Kind() != types.Bool {
				continue
			}
			keys := fieldOfExpr(rhs[i])
			if len(keys) == 1 {
				if obj := info.ObjectOf(id); obj != nil {
					localBools[obj] = keys[0]
				}
			}
		}
	}
	for range 2 {
		ast.Inspect(body, func(n ast.Node) bool {
			if _, isLit := n.(*ast.FuncLit); isLit {
				return false
			}
			switch x := n.(type) {
			case *ast.AssignStmt:
				if x.Tok == token.DEFINE || x.Tok == token.ASSIGN {
					recordAlias(x.Lhs, x.Rhs)
				}
			case *ast.ValueSpec:
				lhs := make([]ast.Expr, len(x.Names))
				for i, name := range x.Names {
					lhs[i] = name
				}
				recordAlias(lhs, x.Values)
			}
			return true
		})
	}

	var out []profileRuleBoolFinding
	isRuleRowKey := func(key string) bool {
		typeName, _, _ := strings.Cut(key, ".")
		for n := range ruleRows {
			if n.Obj().Name() == typeName {
				return true
			}
		}
		return false
	}
	add := func(pos token.Pos, keys []string, why string) {
		line := fset.Position(pos).Line
		for _, k := range keys {
			if isRuleRowKey(k) {
				continue
			}
			out = append(out, profileRuleBoolFinding{Field: k, File: rel, Line: line, Why: why})
		}
	}

	// Catch branchless polarity splits such as flag&&a() || !flag&&b().
	ast.Inspect(body, func(n ast.Node) bool {
		if _, isLit := n.(*ast.FuncLit); isLit {
			return false
		}
		bin, ok := n.(*ast.BinaryExpr)
		if !ok || bin.Op != token.LOR {
			return true
		}
		pos, neg := polarityRefs(bin, info, tracked, localBools)
		var both []string
		for k := range pos {
			if neg[k] {
				both = append(both, k)
			}
		}
		if len(both) > 0 {
			sort.Strings(both)
			add(bin.Pos(), both, "selects between two expressions on both polarities")
			return false
		}
		return true
	})

	var walkStmts func(stmts []ast.Stmt)
	walkStmts = func(stmts []ast.Stmt) {
		for i, st := range stmts {
			switch x := st.(type) {
			case *ast.IfStmt:
				keys := fieldOfExpr(x.Cond)
				if len(keys) > 0 {
					if x.Else != nil {
						add(x.Pos(), keys, "branches into if/else bodies")
					} else if i+1 < len(stmts) && blockAlwaysExits(x.Body) && !isTrivialGuardExit(x.Body, info, pkg) {
						add(x.Pos(), keys, "branches into early-exit and fallthrough bodies")
					}
				}
				walkStmts(x.Body.List)
				switch e := x.Else.(type) {
				case *ast.BlockStmt:
					walkStmts(e.List)
				case *ast.IfStmt:
					walkStmts([]ast.Stmt{e})
				}
			case *ast.SwitchStmt:
				if len(x.Body.List) >= 2 {
					var keys []string
					if x.Tag != nil {
						keys = append(keys, fieldOfExpr(x.Tag)...)
					}
					for _, raw := range x.Body.List {
						cc, ok := raw.(*ast.CaseClause)
						if !ok {
							continue
						}
						for _, expr := range cc.List {
							for _, k := range fieldOfExpr(expr) {
								if !slices.Contains(keys, k) {
									keys = append(keys, k)
								}
							}
						}
					}
					if len(keys) > 0 {
						add(x.Pos(), keys, "branches across switch cases")
					}
				}
				for _, raw := range x.Body.List {
					if cc, ok := raw.(*ast.CaseClause); ok {
						walkStmts(cc.Body)
					}
				}
			case *ast.BlockStmt:
				walkStmts(x.List)
			case *ast.ForStmt:
				walkStmts(x.Body.List)
			case *ast.RangeStmt:
				walkStmts(x.Body.List)
			case *ast.TypeSwitchStmt:
				for _, raw := range x.Body.List {
					if cc, ok := raw.(*ast.CaseClause); ok {
						walkStmts(cc.Body)
					}
				}
			case *ast.LabeledStmt:
				walkStmts([]ast.Stmt{x.Stmt})
			}
		}
	}
	walkStmts(body.List)
	return out
}

func polarityRefs(e ast.Expr, info *types.Info, tracked map[types.Object]string, localBools map[types.Object]string) (pos, neg map[string]bool) {
	pos, neg = map[string]bool{}, map[string]bool{}
	var walk func(n ast.Expr, negated bool)
	walk = func(n ast.Expr, negated bool) {
		n = ast.Unparen(n)
		switch x := n.(type) {
		case *ast.UnaryExpr:
			if x.Op == token.NOT {
				walk(x.X, !negated)
				return
			}
			walk(x.X, negated)
		case *ast.BinaryExpr:
			walk(x.X, negated)
			walk(x.Y, negated)
		case *ast.SelectorExpr:
			if sel := info.Selections[x]; sel != nil {
				if key, ok := tracked[sel.Obj()]; ok {
					if negated {
						neg[key] = true
					} else {
						pos[key] = true
					}
				}
			}
		case *ast.Ident:
			if key, ok := localBools[info.Uses[x]]; ok {
				if negated {
					neg[key] = true
				} else {
					pos[key] = true
				}
			}
		}
	}
	walk(e, false)
	return pos, neg
}

func blockAlwaysExits(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) == 0 {
		return false
	}
	return stmtAlwaysExits(b.List[len(b.List)-1])
}

func stmtAlwaysExits(st ast.Stmt) bool {
	switch x := st.(type) {
	case *ast.ReturnStmt, *ast.BranchStmt:
		return true
	case *ast.BlockStmt:
		return blockAlwaysExits(x)
	case *ast.IfStmt:
		if x.Else == nil || !blockAlwaysExits(x.Body) {
			return false
		}
		return stmtAlwaysExits(x.Else)
	}
	return false
}

// isTrivialGuardExit reports a one-statement exit that performs no work:
// bare return/continue/break, or returning only zero literals or pass-through locals.
func isTrivialGuardExit(b *ast.BlockStmt, info *types.Info, pkg *types.Package) bool {
	if len(b.List) != 1 {
		return false
	}
	switch x := b.List[0].(type) {
	case *ast.BranchStmt:
		return true
	case *ast.ReturnStmt:
		for _, r := range x.Results {
			if !isPassThroughReturnExpr(r, info, pkg) {
				return false
			}
		}
		return true
	}
	return false
}

func isPassThroughReturnExpr(e ast.Expr, info *types.Info, pkg *types.Package) bool {
	e = ast.Unparen(e)
	if tv, ok := info.Types[e]; ok {
		if tv.IsNil() {
			return true
		}
		if tv.Value != nil {
			switch tv.Value.String() {
			case "false", "0", `""`:
				return true
			}
		}
	}
	if lit, ok := e.(*ast.CompositeLit); ok && len(lit.Elts) == 0 {
		return true
	}
	if id, ok := e.(*ast.Ident); ok {
		if v, ok := info.Uses[id].(*types.Var); ok && v.Parent() != nil && v.Parent() != pkg.Scope() {
			return true
		}
	}
	return false
}

func checkSyntheticProfileRuleBools(t *testing.T) {
	t.Helper()
	cases := []struct {
		name       string
		src        string
		wantFields []string
	}{
		{
			name: "old countsListed early return",
			src: `package sim
type Entity struct{ dead bool }
type gameProfile struct{ countsListed bool }
type World struct{ game gameProfile }
func scriptListed(e Entity) bool { return !e.dead }
func scriptDead(e Entity) bool { return e.dead }
func (w *World) scriptAlive(e Entity) bool {
	if w.game.countsListed {
		return scriptListed(e)
	}
	return !scriptDead(e)
}
`,
			wantFields: []string{"gameProfile.countsListed"},
		},
		{
			name: "new arbitrary bool with if/else",
			src: `package sim
type gameProfile struct{ customMode bool }
type World struct{ game gameProfile }
func (w *World) pick() int {
	if w.game.customMode {
		return 2
	} else {
		return 1
	}
}
`,
			wantFields: []string{"gameProfile.customMode"},
		},
		{
			name: "branchless polarity split",
			src: `package sim
type gameProfile struct{ splitMode bool }
type World struct{ game gameProfile }
func a() bool { return true }
func b() bool { return false }
func (w *World) pick() bool {
	return w.game.splitMode && a() || !w.game.splitMode && b()
}
`,
			wantFields: []string{"gameProfile.splitMode"},
		},
		{
			name: "rule row carrying bool fields",
			src: `package sim
type blowRelationRule uint8
type blowRelationPolicy struct{ areaGated, areaBlowAllRows bool }
var blowRelationRules = [...]blowRelationPolicy{{areaGated: true, areaBlowAllRows: true}}
type gameProfile struct{ blowRelation blowRelationRule }
type World struct{ game gameProfile }
func (w *World) check() bool { return blowRelationRules[w.game.blowRelation].areaGated }
`,
			wantFields: []string{"blowRelationPolicy.areaGated", "blowRelationPolicy.areaBlowAllRows"},
		},
		{
			name: "single-body guard and function-field row pass",
			src: `package sim
type scriptCountRule uint8
type scriptCountRow struct{ alive func(int) bool }
var scriptCountRules = [...]scriptCountRow{{alive: func(v int) bool { return v > 0 }}}
type gameProfile struct {
	featureOn   bool
	scriptCount scriptCountRule
}
type World struct{ game gameProfile }
func (w *World) run(v int, buf []byte) ([]byte, bool) {
	if !w.game.featureOn {
		return buf, false
	}
	return buf, scriptCountRules[w.game.scriptCount].alive(v)
}
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "pkg/sim/synth.go", tc.src, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{
				Types:      map[ast.Expr]types.TypeAndValue{},
				Defs:       map[*ast.Ident]types.Object{},
				Uses:       map[*ast.Ident]types.Object{},
				Selections: map[*ast.SelectorExpr]*types.Selection{},
			}
			if _, err := (&types.Config{}).Check("againrom/pkg/sim", fset, []*ast.File{f}, info); err != nil {
				t.Fatal(err)
			}
			cp := &CheckedPackage{
				ImportPath: "againrom/pkg/sim",
				Files:      []*ast.File{f},
				RelPaths:   []string{"pkg/sim/synth.go"},
				NumProd:    1,
				Info:       info,
			}
			got := findProfileRuleBoolFindings(fset, cp)
			if len(tc.wantFields) == 0 {
				if len(got) != 0 {
					t.Fatalf("unexpected findings: %v", got)
				}
				return
			}
			for _, want := range tc.wantFields {
				matched := false
				for _, f := range got {
					if f.Field == want {
						matched = true
					}
				}
				if !matched {
					t.Fatalf("missing finding for %s (got %v)", want, got)
				}
			}
			if vs := checkProfileRuleBools(got, nil); len(vs) < len(tc.wantFields) {
				t.Fatalf("violations %v, want at least %d", vs, len(tc.wantFields))
			}
		})
	}
}
