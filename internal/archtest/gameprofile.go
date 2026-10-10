package archtest

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

// The one-game-profile scan. The game a session runs is chosen once, by the
// profile pkg/base detects, and code reads the profile's edition data or calls
// the campaign service chosen from it. Outside the allowed sites, production
// code does not choose by game, however the choice is written.
//
// A game identity is a base.Game or base.Campaign value, an Edition identity
// field (Game, SaveTag, Campaign, CutsceneArchive, Town), a conversion of one,
// its string form (fmt.Sprint, String, a case or space change), a
// concatenation holding one, a local defined from one, or a string constant
// some edition carries as an identity. Reading Edition data that is not an
// identity is not a choice.

// The shapes, type-checked: a Game or Campaign constant named; an equality
// or order comparison on an identity or against an identity constant, or a
// comparison function (strings, bytes, slices, cmp, reflect.DeepEqual) taking
// one; len of an identity; a switch on an identity or a case naming an
// identity constant; an index into a map keyed by Game or Campaign, or by an
// identity or an identity constant; a return of an identity constant; a call answering one bool from a Game or Campaign receiver or
// argument, or such a function taken as a value; a read of a bool Edition
// field; a non-bool Edition field compared with a constant, a datum used as a
// flag; a use of a bool variable named for a game (rom1, rom2, first game,
// second game); and == or != against nil on a field named second or Second, a
// second-campaign state used as a game flag.
const profilePackage = "pkg/base/"

const basePath = "againrom/pkg/base"

// ProfileAllowed are the sites outside pkg/base that may hold a finding, each
// with its reason. A key is a file, or a file and one of its functions as
// file#function. The two campaign service implementations, and the one
// picker that chooses between them from the edition's campaign model.
var ProfileAllowed = map[string]string{
	"pkg/game/campaignfirst.go":              "the first game's campaign service refuses a second-campaign record and another game's save",
	"pkg/game/campaignsecond.go":             "the second game's campaign service guards its own state and refuses another game's save",
	"pkg/game/campaignservice.go#campaignOf": "the one picker of the campaign service",
}

// ProfileDebt is the findings not yet moved, per file. A count may only
// fall; a file whose count reaches zero leaves the map in the same commit. It
// is empty: every choice reads the edition's data or calls the campaign
// service, and CheckProfile runs with the empty table so any new choice fails.
var ProfileDebt = map[string]int{}

// ProfileFinding is one site and the shape found there. Func is the
// enclosing function declaration's name, empty at package level.
type ProfileFinding struct {
	File  string
	Line  int
	Func  string
	Shape string
}

func (f ProfileFinding) String() string { return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Shape) }

func profileAllowed(f ProfileFinding) bool {
	return ProfileAllowed[f.File] != "" || f.Func != "" && ProfileAllowed[f.File+"#"+f.Func] != ""
}

// CheckProfile turns the findings into violations: a finding in a file that is
// neither allowed nor in debt, a debt file holding more findings than its
// count, and a debt entry whose count no longer matches.
func CheckProfile(found []ProfileFinding, debt map[string]int) []string {
	var out []string
	held := map[string]int{}
	for _, f := range found {
		if profileAllowed(f) {
			continue
		}
		if _, ok := debt[f.File]; ok {
			held[f.File]++
			continue
		}
		out = append(out, f.String()+"; read the profile's edition or call the campaign service")
	}
	names := make([]string, 0, len(debt))
	for name := range debt {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		switch got := held[name]; {
		case got > debt[name]:
			out = append(out, fmt.Sprintf("%s holds %d game-profile findings, above its debt of %d", name, got, debt[name]))
		case got < debt[name]:
			out = append(out, fmt.Sprintf("%s holds %d game-profile findings, below its debt of %d; lower ProfileDebt", name, got, debt[name]))
		}
	}
	return out
}

// LoadProfileFindings walks every production file of the module outside
// pkg/base.
func LoadProfileFindings(root string) ([]ProfileFinding, error) {
	m, err := loadTypeCheckedModule(root)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	if cp := m.Packages[basePath]; cp != nil {
		ids = profileIdentityStrings(cp.Files[:cp.NumProd], cp.Info)
	}
	var out []ProfileFinding
	for _, cp := range m.Packages {
		for i, f := range cp.Files[:cp.NumProd] {
			rel := cp.RelPaths[i]
			if strings.HasPrefix(rel, profilePackage) || strings.HasPrefix(rel, "internal/archtest/") {
				continue
			}
			out = append(out, profileFindings(m.Fset, f, cp.Info, rel, ids)...)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

// profileIdentityFields are the Edition fields that say which game an edition
// is, as against what it holds.
var profileIdentityFields = map[string]bool{"Game": true, "SaveTag": true, "Campaign": true, "CutsceneArchive": true, "Town": true}

// profileIdentityStrings are the string values pkg/base gives a game: its
// Game constants and every string an Edition literal sets in an identity
// field. The empty string is no identity.
func profileIdentityStrings(files []*ast.File, info *types.Info) map[string]bool {
	ids := map[string]bool{}
	addConst := func(v constant.Value) {
		if v != nil && v.Kind() == constant.String && constant.StringVal(v) != "" {
			ids[constant.StringVal(v)] = true
		}
	}
	for _, obj := range info.Defs {
		if c, ok := obj.(*types.Const); ok && isNamed(c.Type(), basePath, "Game") {
			addConst(c.Val())
		}
	}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isNamed(info.TypeOf(lit), basePath, "Edition") {
				return true
			}
			for _, el := range lit.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if key, ok := kv.Key.(*ast.Ident); ok && profileIdentityFields[key.Name] {
					addConst(info.Types[kv.Value].Value)
				}
			}
			return true
		})
	}
	return ids
}

// stringsComparisons are the strings and bytes functions that compare their
// arguments.
var stringsComparisons = map[string]bool{
	"HasPrefix": true, "HasSuffix": true, "Contains": true, "EqualFold": true, "Compare": true,
	"Index": true, "LastIndex": true, "Cut": true, "CutPrefix": true, "CutSuffix": true,
	"TrimPrefix": true, "TrimSuffix": true, "Count": true, "Equal": true,
}

// profileComparisons are the standard functions, by package, that compare
// their arguments.
var profileComparisons = map[string]map[string]bool{
	"strings": stringsComparisons,
	"bytes":   stringsComparisons,
	"reflect": {"DeepEqual": true},
	"slices":  {"Contains": true, "Index": true, "Equal": true, "Compare": true},
	"cmp":     {"Compare": true, "Less": true},
}

// profileStringForms are the standard functions, by package, whose result
// is the string form of their game argument.
var profileStringForms = map[string]map[string]bool{
	"fmt":     {"Sprint": true, "Sprintf": true, "Sprintln": true},
	"strings": {"ToLower": true, "ToUpper": true, "TrimSpace": true, "Clone": true},
}

type profileScan struct {
	info    *types.Info
	ids     map[string]bool
	derived map[types.Object]bool
	// called holds each call's function expression, so a game question is
	// found once: as a call, or as a value when it is not called.
	called map[ast.Expr]bool
}

func profileFindings(fset *token.FileSet, f *ast.File, info *types.Info, rel string, ids map[string]bool) []ProfileFinding {
	s := &profileScan{info: info, ids: ids, derived: map[types.Object]bool{}, called: map[ast.Expr]bool{}}
	s.markDerived(f)
	var out []ProfileFinding
	var fn string
	add := func(n ast.Node, shape string) {
		out = append(out, ProfileFinding{File: rel, Line: fset.Position(n.Pos()).Line, Func: fn, Shape: shape})
	}
	for _, decl := range f.Decls {
		fn = ""
		if d, ok := decl.(*ast.FuncDecl); ok {
			fn = d.Name.Name
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if s.editionFlag(n) {
					add(n, "reads an edition flag")
				}
				if !s.called[n] && s.gameQuestion(s.info.Uses[n.Sel]) {
					add(n, "takes a game question as a value")
				}
				s.called[n.Sel] = true
			case *ast.Ident:
				if s.namesGame(n) {
					add(n, "names a game")
				}
				if s.gameFlag(n) {
					add(n, "reads a game flag")
				}
				if !s.called[n] && s.gameQuestion(s.info.Uses[n]) {
					add(n, "takes a game question as a value")
				}
			case *ast.BinaryExpr:
				if !profileComparison(n.Op) {
					return true
				}
				if s.identity(n.X) || s.identity(n.Y) || s.identityConst(n.X) || s.identityConst(n.Y) {
					add(n, "compares a game")
				}
				if s.editionDatum(n.X) && s.constant(n.Y) || s.editionDatum(n.Y) && s.constant(n.X) {
					add(n, "tests edition data as a flag")
				}
				if n.Op != token.EQL && n.Op != token.NEQ {
					return true
				}
				if isNil(info, n.Y) && isSecondField(n.X) || isNil(info, n.X) && isSecondField(n.Y) {
					add(n, "tests a second-campaign state for nil")
				}
			case *ast.SwitchStmt:
				if n.Tag != nil && s.identity(n.Tag) {
					add(n, "branches on a game")
					return true
				}
				for _, st := range n.Body.List {
					cc, ok := st.(*ast.CaseClause)
					if !ok {
						continue
					}
					for _, e := range cc.List {
						if s.identityConst(e) {
							add(e, "branches on a game")
						}
					}
				}
			case *ast.IndexExpr:
				if m, ok := typeUnder(info.TypeOf(n.X)).(*types.Map); ok && isGameType(m.Key()) || s.identity(n.Index) && !s.dataName(n.Index) {
					add(n, "looks up a game")
				} else if s.identityConst(n.Index) {
					add(n, "looks up a game by a literal key")
				}
			case *ast.ReturnStmt:
				for _, r := range n.Results {
					if s.identityConst(r) {
						add(r, "answers a game literal")
					}
				}
			case *ast.CallExpr:
				s.called[ast.Unparen(n.Fun)] = true
				if s.stringsCompare(n) {
					add(n, "compares a game")
				}
				if s.asksGame(n) {
					add(n, "asks a game")
				}
				if s.measuresGame(n) {
					add(n, "measures a game")
				}
			}
			return true
		})
	}
	return out
}

func typeUnder(t types.Type) types.Type {
	if t == nil {
		return nil
	}
	return t.Underlying()
}

func isGameType(t types.Type) bool {
	return isNamed(t, basePath, "Game") || isNamed(t, basePath, "Campaign")
}

// markDerived records each local defined from an identity, so a comparison
// of the local is a comparison of the identity.
func (s *profileScan) markDerived(f *ast.File) {
	mark := func(lhs []ast.Expr, rhs []ast.Expr) {
		if len(lhs) != len(rhs) {
			return
		}
		for i, l := range lhs {
			id, ok := l.(*ast.Ident)
			if !ok || !s.identity(rhs[i]) {
				continue
			}
			if obj := s.info.ObjectOf(id); obj != nil {
				s.derived[obj] = true
			}
		}
	}
	// Two passes reach a local defined from a local.
	for pass := 0; pass < 2; pass++ {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if n.Tok == token.DEFINE {
					mark(n.Lhs, n.Rhs)
				}
			case *ast.ValueSpec:
				lhs := make([]ast.Expr, len(n.Names))
				for i, name := range n.Names {
					lhs[i] = name
				}
				mark(lhs, n.Values)
			}
			return true
		})
	}
}

// identity reports whether e is a game identity.
func (s *profileScan) identity(e ast.Expr) bool {
	e = ast.Unparen(e)
	if isGameType(s.info.TypeOf(e)) {
		return true
	}
	switch e := e.(type) {
	case *ast.SelectorExpr:
		return profileIdentityFields[e.Sel.Name] && s.editionField(e)
	case *ast.Ident:
		return s.derived[s.info.ObjectOf(e)]
	case *ast.BinaryExpr:
		return e.Op == token.ADD && (s.identity(e.X) || s.identity(e.Y))
	case *ast.CallExpr:
		if tv, ok := s.info.Types[e.Fun]; ok && tv.IsType() && len(e.Args) == 1 {
			return s.identity(e.Args[0])
		}
		if sel, ok := ast.Unparen(e.Fun).(*ast.SelectorExpr); ok && sel.Sel.Name == "String" {
			if selection := s.info.Selections[sel]; selection != nil && selection.Kind() == types.MethodVal && isGameType(selection.Recv()) {
				return true
			}
		}
		if fn := s.stdFunc(e.Fun); fn != nil && profileStringForms[fn.Pkg().Path()][fn.Name()] {
			for _, a := range e.Args {
				if s.identity(a) {
					return true
				}
			}
		}
	}
	return false
}

// stdFunc is the package-level or method function fun names, or nil.
func (s *profileScan) stdFunc(fun ast.Expr) *types.Func {
	sel, ok := ast.Unparen(fun).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	fn, ok := s.info.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return nil
	}
	return fn
}

func profileComparison(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
		return true
	}
	return false
}

// constant reports whether e is a constant value.
func (s *profileScan) constant(e ast.Expr) bool {
	tv, ok := s.info.Types[ast.Unparen(e)]
	return ok && tv.Value != nil
}

// editionDatum reports whether e reads a non-bool Edition field that is not
// an identity: data, which a comparison with a constant turns into a flag.
func (s *profileScan) editionDatum(e ast.Expr) bool {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	if !ok || !s.editionField(sel) || profileIdentityFields[sel.Sel.Name] {
		return false
	}
	b, ok := typeUnder(s.info.TypeOf(sel)).(*types.Basic)
	return !ok || b.Kind() != types.Bool
}

// profileDataNames are the identity fields that name data: a lookup keyed by
// one reads the data the edition names.
var profileDataNames = map[string]bool{"Town": true, "CutsceneArchive": true}

// dataName reports whether e is an Edition field naming data, read as is.
func (s *profileScan) dataName(e ast.Expr) bool {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	return ok && profileDataNames[sel.Sel.Name] && s.editionField(sel)
}

// measuresGame reports whether call is len of an identity.
func (s *profileScan) measuresGame(call *ast.CallExpr) bool {
	id, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok || len(call.Args) != 1 {
		return false
	}
	if b, ok := s.info.Uses[id].(*types.Builtin); !ok || b.Name() != "len" {
		return false
	}
	return s.identity(call.Args[0])
}

// gameQuestion reports whether obj is a function or method answering one
// bool from a Game or Campaign receiver or parameter.
func (s *profileScan) gameQuestion(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Results().Len() != 1 {
		return false
	}
	if b, ok := typeUnder(sig.Results().At(0).Type()).(*types.Basic); !ok || b.Kind() != types.Bool {
		return false
	}
	if sig.Recv() != nil && isGameType(sig.Recv().Type()) {
		return true
	}
	for i := 0; i < sig.Params().Len(); i++ {
		if isGameType(sig.Params().At(i).Type()) {
			return true
		}
	}
	return false
}

// identityConst reports whether e is a string constant some edition carries
// as an identity.
func (s *profileScan) identityConst(e ast.Expr) bool {
	tv, ok := s.info.Types[ast.Unparen(e)]
	return ok && tv.Value != nil && tv.Value.Kind() == constant.String && s.ids[constant.StringVal(tv.Value)]
}

// editionField reports whether sel selects a field of base.Edition.
func (s *profileScan) editionField(sel *ast.SelectorExpr) bool {
	selection := s.info.Selections[sel]
	return selection != nil && selection.Kind() == types.FieldVal && isNamed(selection.Recv(), basePath, "Edition")
}

func (s *profileScan) editionFlag(sel *ast.SelectorExpr) bool {
	if !s.editionField(sel) {
		return false
	}
	b, ok := typeUnder(s.info.TypeOf(sel)).(*types.Basic)
	return ok && b.Kind() == types.Bool
}

// namesGame reports whether id is a use of a Game or Campaign constant.
func (s *profileScan) namesGame(id *ast.Ident) bool {
	c, ok := s.info.Uses[id].(*types.Const)
	return ok && isGameType(c.Type())
}

// gameFlag reports whether id is a use of a bool variable or field named for
// a game. A function so named is a dialect or layout helper, not a flag.
func (s *profileScan) gameFlag(id *ast.Ident) bool {
	v, ok := s.info.Uses[id].(*types.Var)
	if !ok {
		return false
	}
	name := strings.ToLower(v.Name())
	if !strings.Contains(name, "rom1") && !strings.Contains(name, "rom2") && !strings.Contains(name, "firstgame") && !strings.Contains(name, "secondgame") {
		return false
	}
	b, ok := typeUnder(v.Type()).(*types.Basic)
	return ok && b.Kind() == types.Bool
}

// stringsCompare reports whether call is a standard comparison function
// taking an identity or an identity constant.
func (s *profileScan) stringsCompare(call *ast.CallExpr) bool {
	fn := s.stdFunc(call.Fun)
	if fn == nil || !profileComparisons[fn.Pkg().Path()][fn.Name()] {
		return false
	}
	for _, a := range call.Args {
		if s.identity(a) || s.identityConst(a) {
			return true
		}
	}
	return false
}

// asksGame reports whether call answers one bool from a Game or Campaign
// receiver or argument.
func (s *profileScan) asksGame(call *ast.CallExpr) bool {
	sig, ok := typeUnder(s.info.TypeOf(call.Fun)).(*types.Signature)
	if !ok || sig.Results().Len() != 1 {
		return false
	}
	if b, ok := typeUnder(sig.Results().At(0).Type()).(*types.Basic); !ok || b.Kind() != types.Bool {
		return false
	}
	if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
		if selection := s.info.Selections[sel]; selection != nil && selection.Kind() == types.MethodVal && isGameType(selection.Recv()) {
			return true
		}
	}
	for i := 0; i < sig.Params().Len(); i++ {
		if isGameType(sig.Params().At(i).Type()) {
			return true
		}
	}
	return false
}

func isNil(info *types.Info, e ast.Expr) bool {
	tv, ok := info.Types[ast.Unparen(e)]
	return ok && tv.IsNil()
}

func isSecondField(e ast.Expr) bool {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	return ok && (sel.Sel.Name == "second" || sel.Sel.Name == "Second")
}
