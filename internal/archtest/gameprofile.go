package archtest

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

// The one-game-profile scan. The game a session runs is chosen once, by the
// profile pkg/base detects, and code reads the profile's edition data or calls
// the campaign service chosen from it. Outside the allowed files, production
// code names no game constant, compares no game value and does not test a
// second-campaign state for nil to learn which game is running.
//
// It finds three shapes, type-checked:
//   - a use of base.GameROM1 or base.GameROM2: a game named in code;
//   - an == or != whose operand is a base.Game: a game compared;
//   - an == or != against nil whose operand is a field named second or
//     Second: a second-campaign state used as a game flag.
const profilePackage = "pkg/base/"

// ProfileAllowed are the files outside pkg/base that may hold a finding, each
// with its reason.
var ProfileAllowed = map[string]string{
	"pkg/game/campaignservice.go": "the one place that picks the campaign service from the profile",
	"pkg/game/campaignsecond.go":  "the second game's campaign service guards its own state",
}

// ProfileDebt is the findings this story could not move, per file. A count
// may only fall; a file whose count reaches zero leaves the map in the same
// commit.
var ProfileDebt = map[string]int{
	"cmd/terraintool/main.go":           2,
	"pkg/game/archives.go":              1,
	"pkg/game/base.go":                  2,
	"pkg/game/campaignsession.go":       2,
	"pkg/game/currentsave.go":           9,
	"pkg/game/currentscriptbindings.go": 2,
	"pkg/game/currentsecondcampaign.go": 10,
	"pkg/game/currentsession.go":        14,
	"pkg/game/cutscene.go":              2,
	"pkg/game/frontend.go":              6,
	"pkg/game/mapload.go":               3,
	"pkg/game/mission.go":               6,
	"pkg/game/missionentry.go":          1,
	"pkg/game/missiontransition.go":     4,
	"pkg/game/originalsave.go":          6,
	"pkg/game/questobjectives.go":       2,
	"pkg/game/resume.go":                13,
	"pkg/game/save.go":                  1,
	"pkg/game/secondcensus.go":          3,
	"pkg/game/secondcompletion.go":      2,
	"pkg/game/secondgamenotices.go":     2,
	"pkg/game/secondgametext.go":        2,
	"pkg/game/table.go":                 3,
	"pkg/game/townscreen.go":            2,
	"pkg/game/world.go":                 3,
	"pkg/mapload/cheatfactory.go":       4,
	"pkg/mapload/currentplayers.go":     2,
	"pkg/mapload/spawn.go":              2,
	"pkg/mapload/spell.go":              2,
}

// ProfileFinding is one site and the shape found there.
type ProfileFinding struct {
	File  string
	Line  int
	Shape string
}

func (f ProfileFinding) String() string { return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Shape) }

// CheckProfile turns the findings into violations: a finding in a file that is
// neither allowed nor in debt, a debt file holding more findings than its
// count, and a debt entry whose count no longer matches.
func CheckProfile(found []ProfileFinding, debt map[string]int) []string {
	var out []string
	held := map[string]int{}
	for _, f := range found {
		if ProfileAllowed[f.File] != "" {
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
	var out []ProfileFinding
	for _, cp := range m.Packages {
		for i, f := range cp.Files[:cp.NumProd] {
			rel := cp.RelPaths[i]
			if strings.HasPrefix(rel, profilePackage) || strings.HasPrefix(rel, "internal/archtest/") {
				continue
			}
			out = append(out, profileFindings(m.Fset, f, cp.Info, rel)...)
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

func profileFindings(fset *token.FileSet, f *ast.File, info *types.Info, rel string) []ProfileFinding {
	var out []ProfileFinding
	add := func(n ast.Node, shape string) {
		out = append(out, ProfileFinding{File: rel, Line: fset.Position(n.Pos()).Line, Shape: shape})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if (n.Sel.Name == "GameROM1" || n.Sel.Name == "GameROM2") && isNamed(info.TypeOf(n), "againrom/pkg/base", "Game") {
				add(n, "names a game")
			}
		case *ast.BinaryExpr:
			if n.Op != token.EQL && n.Op != token.NEQ {
				return true
			}
			if isNamed(info.TypeOf(n.X), "againrom/pkg/base", "Game") || isNamed(info.TypeOf(n.Y), "againrom/pkg/base", "Game") {
				add(n, "compares a game")
			}
			if isNil(info, n.Y) && isSecondField(n.X) || isNil(info, n.X) && isSecondField(n.Y) {
				add(n, "tests a second-campaign state for nil")
			}
		}
		return true
	})
	return out
}

func isNil(info *types.Info, e ast.Expr) bool {
	tv, ok := info.Types[ast.Unparen(e)]
	return ok && tv.IsNil()
}

func isSecondField(e ast.Expr) bool {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	return ok && (sel.Sel.Name == "second" || sel.Sel.Name == "Second")
}
