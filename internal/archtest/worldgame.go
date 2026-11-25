package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// The World game scan. A pkg/sim World asks once which game it runs:
// newGameProfile chooses its gameProfile on each path that builds a World,
// and every first/second-game difference reads a field of that profile.
// Outside the named sites, production code in pkg/sim and pkg/game does not
// ask the game again, however the question is written.
//
// A spell rule row carries no game: the rules a World applies are the
// spellRow its profile's spellGame selects. A Second field or key, and the
// per-row branches it fed (drainRuleOf, secondArm, spellDamageUnder), stay
// out of pkg/sim; AssignSecondGameArms, which writes the arm ids of a
// second-game table, stays at named sites.

// The shapes the scan refuses.
const (
	worldGameSecondGame  = "a SecondGame call"
	worldGameROM2Nil     = "a rom2 nil test"
	worldGameNative      = "a groupDialectNative call"
	worldGameSavedGroups = "a savedGroups nil test"
	worldGameDialect     = "a script dialect compared as the game"
	worldGameChooser     = "a game profile chosen or written"
	worldGameSpellRow    = "a spell row's format marker"
	worldGameRowBranch   = "a per-row game branch the spell rows replaced"
	worldGameSpellTable  = "a spell table's format read from one row"
	worldGameSpellArms   = "a use of AssignSecondGameArms"
)

// worldGamePackages are the trees the scan reads for every shape but
// AssignSecondGameArms, which it refuses in every production file.
var worldGamePackages = []string{"pkg/sim/", "pkg/game/"}

// WorldGameAllowed are the sites that may hold a shape, keyed by shape, then
// by file#function (a package-level declaration names its first value), each
// with its reason.
var WorldGameAllowed = map[string]map[string]string{
	worldGameROM2Nil: {
		"pkg/sim/currentrom2policy.go#currentROM2Policy":      "state: exports the second game's script state the World holds",
		"pkg/sim/rom2script.go#ROM2ScenarioValue":             "state: reads the Scenario bank",
		"pkg/sim/rom2script.go#ROM2ScenarioState":             "state: reads the Scenario bank",
		"pkg/sim/rom2script.go#SetROM2ScenarioState":          "state: writes the Scenario bank",
		"pkg/sim/rom2script.go#setROM2GroupForced":            "state: writes the Group activity rows",
		"pkg/sim/rom2statebinary.go#rom2ScriptStateFault":     "state: the byte form holds the state exactly when the dialect is the second game's",
		"pkg/sim/rom2statebinary.go#appendROM2ScriptState":    "state: writes the state the World holds",
		"pkg/sim/rom2statebinary.go#unmarshalROM2ScriptState": "state: decodes the state the wrapper carries",
	},
	worldGameSavedGroups: {
		"pkg/sim/carriers.go#groupsAsConstructed":             "decode: an omitted registry cannot equal a constructed one yet",
		"pkg/sim/savedgroups.go#ImportSavedGroups":            "import: preserve metadata only from an installed registry",
		"pkg/sim/carriers.go#constructGroups":                 "construction: keep the installed registry",
		"pkg/sim/creaturespell.go#fillOrderSlotWindows":       "decode: a wrapper can precede registry construction",
		"pkg/sim/savedgroups.go#savedOrder":                   "decode: no order exists before the registry is constructed",
		"pkg/sim/playerparticipants.go#RestoreCurrentPlayers": "decode: restore Players before constructing an omitted registry",
		"pkg/sim/savedgroupplayers.go#SavedGroupPlayers":      "decode: a base form can omit its registry before the Player wrapper",
	},
	worldGameDialect: {
		"pkg/sim/gameprofile.go#newGameProfile":                       "the one chooser of the World's game profile",
		"pkg/sim/rom2script.go#initializeROM2Script":                  "builds the second game's script state with the profile",
		"pkg/sim/rom2statebinary.go#rom2ScriptStateFault":             "the byte form holds the state exactly when the dialect is the second game's",
		"pkg/sim/script.go#newScriptOn":                               "the compiler admits each operation by the program's own dialect",
		"pkg/sim/world.go#newWorld":                                   "an empty first-game program is no program",
		"pkg/game/currentscriptbindings.go#restoreCurrentScriptRoles": "compiles the roles by the running program's own dialect",
	},
	worldGameSpellArms: {
		"pkg/sim/secondspell.go#AssignSecondGameArms":         "its declaration",
		"pkg/mapload/spell.go#SecondGameSpellArms":            "the second game's table is loaded",
		"pkg/sim/rom2statebinary.go#unmarshalROM2ScriptState": "the second game's wrapper decodes its table",
	},
	worldGameChooser: {
		"pkg/sim/gameprofile.go#newGameProfile":               "the one chooser of the World's game profile",
		"pkg/sim/rom2script.go#initializeROM2Script":          "the build path of mission start and the controlled script copy",
		"pkg/sim/binary.go#unmarshalBinary":                   "the decoder's base form",
		"pkg/sim/rom2statebinary.go#unmarshalROM2ScriptState": "the decoder's second-game wrapper",
		"pkg/sim/script.go#newScriptOn":                       "the compiler admits each operation by the rows of the program's own profile",
		"pkg/sim/gameprofile.go#HoldsScriptState":             "a caller with a program and no World reads the dialect's own profile field",
		"pkg/sim/spellrow.go#formSpellRow":                    "a wrapper decodes its spell rows before the World it builds has its game",
		"pkg/sim/skill.go#HeldSkillLevel":                     "a party load outside a World reads the skill award of the chooser's profile",
	},
}

// WorldGameFinding is one site and the shape found there. Func is the
// enclosing function, or the first name of a package-level declaration.
type WorldGameFinding struct {
	File  string
	Line  int
	Func  string
	Shape string
}

func (f WorldGameFinding) String() string {
	return fmt.Sprintf("%s:%d %s (%s)", f.File, f.Line, f.Shape, f.Func)
}

// worldGameScope is what the scan looks for in one file.
type worldGameScope int

const (
	// worldGameAnywhere: only a use of AssignSecondGameArms.
	worldGameAnywhere worldGameScope = iota
	// worldGameSession: pkg/game, every shape but those of pkg/sim's fields.
	worldGameSession
	// worldGameWorld: pkg/sim, every shape.
	worldGameWorld
)

// FindWorldGame lists every shape in the production files of pkg/sim and
// pkg/game, and every use of AssignSecondGameArms in any production file.
// files is keyed by module-relative slash path, as LoadRandomnessSources
// returns it.
func FindWorldGame(files map[string]string) ([]WorldGameFinding, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		if worldGameScanned(name, files[name]) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var out []WorldGameFinding
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("%s does not parse: %w", name, err)
		}
		scope := worldGameScopeOf(name)
		for _, decl := range f.Decls {
			owner := worldGameOwner(decl)
			aliases := map[string]bool{}
			ast.Inspect(decl, func(n ast.Node) bool {
				if a, ok := n.(*ast.AssignStmt); ok {
					for i, value := range a.Rhs {
						if worldGameName(value) == "savedGroups" && i < len(a.Lhs) {
							if id, ok := a.Lhs[i].(*ast.Ident); ok {
								aliases[id.Name] = true
							}
						}
					}
				}
				return true
			})
			add := func(n ast.Node, shape string) {
				out = append(out, WorldGameFinding{File: name, Line: fset.Position(n.Pos()).Line, Func: owner, Shape: shape})
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.ValueSpec:
					// A declared name is not a use; its values are read.
					for _, v := range x.Values {
						ast.Inspect(v, func(m ast.Node) bool { worldGameNode(m, scope, aliases, add); return true })
					}
					return false
				default:
					worldGameNode(n, scope, aliases, add)
				}
				return true
			})
		}
	}
	return out, nil
}

func worldGameScanned(name, source string) bool {
	if strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
		return false
	}
	return worldGameScopeOf(name) != worldGameAnywhere || strings.Contains(source, "AssignSecondGameArms") || strings.Contains(source, "groupDialectNative")
}

func worldGameScopeOf(name string) worldGameScope {
	switch {
	case strings.HasPrefix(name, worldGamePackages[0]):
		return worldGameWorld
	case strings.HasPrefix(name, worldGamePackages[1]):
		return worldGameSession
	}
	return worldGameAnywhere
}

func worldGameOwner(decl ast.Decl) string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return d.Name.Name
	case *ast.GenDecl:
		for _, s := range d.Specs {
			if v, ok := s.(*ast.ValueSpec); ok && len(v.Names) > 0 {
				return v.Names[0].Name
			}
		}
	}
	return ""
}

// worldGameNode reports the shape one node holds. A write of a field named
// game, and a selector or key named Second, are read only in pkg/sim, where
// the World's profile and the spell row live; elsewhere the names mean other
// things. A Second read through a constant index is the whole table's format.
func worldGameNode(n ast.Node, scope worldGameScope, aliases map[string]bool, add func(ast.Node, string)) {
	if id, ok := n.(*ast.Ident); ok && id.Name == "AssignSecondGameArms" {
		add(id, worldGameSpellArms)
		return
	}
	if id, ok := n.(*ast.Ident); ok && id.Name == "groupDialectNative" {
		add(id, worldGameNative)
		return
	}
	if scope == worldGameAnywhere {
		return
	}
	world := scope == worldGameWorld
	switch x := n.(type) {
	case *ast.SelectorExpr:
		if world && x.Sel.Name == "Second" {
			if ix, ok := x.X.(*ast.IndexExpr); ok {
				if _, lit := ix.Index.(*ast.BasicLit); lit {
					add(x, worldGameSpellTable)
					return
				}
			}
			add(x, worldGameSpellRow)
		}
	case *ast.CallExpr:
		switch worldGameName(x.Fun) {
		case "SecondGame":
			add(x, worldGameSecondGame)
		case "newGameProfile":
			add(x, worldGameChooser)
		}
	case *ast.BinaryExpr:
		if x.Op != token.EQL && x.Op != token.NEQ {
			return
		}
		l, r := worldGameName(x.X), worldGameName(x.Y)
		switch {
		case worldGameNil(x.X) && r == "rom2", worldGameNil(x.Y) && l == "rom2":
			add(x, worldGameROM2Nil)
		case worldGameNil(x.X) && (r == "savedGroups" || aliases[r]), worldGameNil(x.Y) && (l == "savedGroups" || aliases[l]):
			add(x, worldGameSavedGroups)
		case worldGameDialectName(l), worldGameDialectName(r):
			add(x, worldGameDialect)
		}
	case *ast.CaseClause:
		for _, e := range x.List {
			if worldGameDialectName(worldGameName(e)) {
				add(e, worldGameDialect)
			}
		}
	case *ast.AssignStmt:
		for _, l := range x.Lhs {
			if sel, ok := l.(*ast.SelectorExpr); ok && world && sel.Sel.Name == "game" {
				add(l, worldGameChooser)
			}
		}
	case *ast.KeyValueExpr:
		if id, ok := x.Key.(*ast.Ident); ok && world && id.Name == "game" {
			add(x, worldGameChooser)
		}
		if id, ok := x.Key.(*ast.Ident); ok && world && id.Name == "Second" {
			add(x, worldGameSpellRow)
		}
	case *ast.Ident:
		if world && (x.Name == "drainRuleOf" || x.Name == "secondArm" || x.Name == "spellDamageUnder") {
			add(x, worldGameRowBranch)
		}
		if x.Name == "secondGameGroupRoute" || x.Name == "secondGameCreatureAim" {
			add(x, worldGameChooser)
		}
	}
}

// worldGameName is the name an identifier or a selector ends in.
func worldGameName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	case *ast.ParenExpr:
		return worldGameName(x.X)
	}
	return ""
}

func worldGameNil(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}

func worldGameDialectName(name string) bool { return name == "ScriptROM1" || name == "ScriptROM2" }

// CheckWorldGame turns the findings into violations: a shape at a site that
// is not allowed. Registry presence is allowed only at named construction
// and decode guards.
func CheckWorldGame(found []WorldGameFinding, allowed map[string]map[string]string) []string {
	var out []string
	for _, f := range found {
		if a := allowed[f.Shape]; f.Shape != worldGameNative && (a[f.File] != "" || a[f.File+"#"+f.Func] != "") {
			continue
		}
		switch f.Shape {
		case worldGameNative, worldGameSavedGroups:
			out = append(out, f.String()+"; every World uses its Group registry; only named construction and decode guards may test its presence")
		case worldGameSpellRow, worldGameSpellTable, worldGameRowBranch:
			out = append(out, f.String()+"; a spell rule row carries no game; read the World's spellRow")
		case worldGameSpellArms:
			out = append(out, f.String()+"; a second-game table's arm ids are written where it is loaded or decoded; name the site with its reason")
		default:
			out = append(out, f.String()+"; read a field of the World's game profile")
		}
	}
	return out
}
