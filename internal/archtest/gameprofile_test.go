package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"
)

// TestLiveGameChoiceStaysInTheProfile measures the real tree: no production
// file outside pkg/base names or compares a game, or tests a second-campaign
// state for nil, except the allowed files and the falling debt list.
func TestLiveGameChoiceStaysInTheProfile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	found, err := LoadProfileFindings(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, v := range CheckProfile(found, ProfileDebt) {
		t.Error(v)
	}
}

// TestProfileFindingsSeeEachShape holds the scan against synthetic sources, so
// a scan that went blind fails here rather than passing a dirty tree.
func TestProfileFindingsSeeEachShape(t *testing.T) {
	const basePkg = `package base
type Game string
const (
	GameROM1 Game = "rom1"
	GameROM2 Game = "rom2"
)`
	const user = `package user

import "againrom/pkg/base"

type campaign struct{}
type town struct{ second *campaign }

func named() base.Game { return base.GameROM2 }

func compared(a, b base.Game) bool { return a == b }

func flag(t *town) bool { return t.second != nil }

func clean(t *town, g base.Game) (base.Game, *campaign) { return g, t.second }
`
	fset := token.NewFileSet()
	files := map[string]*ast.File{}
	for name, src := range map[string]string{"base.go": basePkg, "user.go": user} {
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = f
	}
	basePackage, err := (&types.Config{}).Check("againrom/pkg/base", fset, []*ast.File{files["base.go"]}, nil)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: importerFunc(func(path string) (*types.Package, error) { return basePackage, nil })}
	if _, err := conf.Check("againrom/user", fset, []*ast.File{files["user.go"]}, info); err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, f := range profileFindings(fset, files["user.go"], info, "pkg/user/user.go", nil) {
		got[f.Shape]++
	}
	want := map[string]int{"names a game": 1, "compares a game": 1, "tests a second-campaign state for nil": 1}
	for shape, n := range want {
		if got[shape] != n {
			t.Errorf("%s: %d findings, want %d (all: %v)", shape, got[shape], n, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("findings %v, want only %v", got, want)
	}

	found := []ProfileFinding{{File: "pkg/game/a.go", Line: 1}, {File: "pkg/game/b.go", Line: 2}, {File: "pkg/game/campaignsecond.go", Line: 3}}
	if v := CheckProfile(found, map[string]int{"pkg/game/b.go": 1}); len(v) != 1 {
		t.Errorf("a finding outside the debt must fail once, got %v", v)
	}
	if v := CheckProfile(found, map[string]int{"pkg/game/a.go": 1, "pkg/game/b.go": 2}); len(v) != 1 {
		t.Errorf("a debt above its findings must fail once, got %v", v)
	}
}

// profileMutationBase is a game profile package shaped like pkg/base: the
// games, the campaign models, an edition with identity fields, data and a
// flag, and the bool questions a game answers.
const profileMutationBase = `package base
type Game string
const (
	GameROM1 Game = "rom1"
	GameROM2 Game = "rom2"
)
type Campaign int
const (
	CampaignChapters Campaign = iota
	CampaignDestinations
)
type Edition struct {
	Game Game
	SaveTag Game
	Campaign Campaign
	Town string
	Rooms string
	CutsceneArchive string
	CompanionObjectiveMission int
	SecondMaps bool
}
var first = Edition{Game: GameROM1, Town: "rom1", Rooms: "rom1", CompanionObjectiveMission: 40}
var second = Edition{Game: GameROM2, SaveTag: GameROM2, Campaign: CampaignDestinations, Rooms: "rom1", CutsceneArchive: "video", SecondMaps: true}
func (g Game) Edition() Edition {
	if g == GameROM2 {
		return second
	}
	return first
}
func (g Game) Known() bool { return g == "" || g == GameROM1 || g == GameROM2 }
func SameGame(a, b Game) bool { return a == b }
`

// profileMutationStd stands in for the standard packages a mutation
// imports, by path.
var profileMutationStd = map[string]string{
	"strings": `package strings
func HasPrefix(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }
`,
	"fmt": `package fmt
func Sprint(a ...any) string { return "" }
`,
	"reflect": `package reflect
func DeepEqual(x, y any) bool { return false }
`,
}

// profileMutationScan type-checks body as one file of a package importing
// the profile package, and answers its findings.
func profileMutationScan(t *testing.T, body string) []ProfileFinding {
	t.Helper()
	fset := token.NewFileSet()
	check := func(path, name, src string, imp types.Importer, info *types.Info) (*ast.File, *types.Package) {
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		p, err := (&types.Config{Importer: imp}).Check(path, fset, []*ast.File{f}, info)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return f, p
	}
	newInfo := func() *types.Info {
		return &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{},
			Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	}
	baseInfo := newInfo()
	baseFile, basePkg := check("againrom/pkg/base", "base.go", profileMutationBase, nil, baseInfo)
	std := map[string]*types.Package{}
	src := "package user\n\nimport \"againrom/pkg/base\"\n"
	for path, stub := range profileMutationStd {
		if strings.Contains(body, path+".") {
			_, std[path] = check(path, path+".go", stub, nil, newInfo())
			src += "import \"" + path + "\"\n"
		}
	}
	imp := importerFunc(func(path string) (*types.Package, error) {
		if p := std[path]; p != nil {
			return p, nil
		}
		return basePkg, nil
	})
	src += "var _ base.Game\n\n" + body + "\n"
	info := newInfo()
	f, _ := check("againrom/user", "user.go", src, imp, info)
	ids := profileIdentityStrings([]*ast.File{baseFile}, baseInfo)
	return profileFindings(fset, f, info, "pkg/user/user.go", ids)
}

// TestProfileFindingsRejectEveryForm holds the scan against each way of
// choosing a game: the ten a review placed in a production file, of which the
// scan once saw only the first, the forms the live tree held beside them, and
// the nine a second review placed in a production file that the scan then
// passed. Each must be found with its shape; reading edition data must not
// be.
func TestProfileFindingsRejectEveryForm(t *testing.T) {
	mutations := []struct{ name, shape, body string }{
		{"case naming a game constant", "names a game",
			`func f(g base.Game) int { switch g { case base.GameROM2: return 2 }; return 1 }`},
		{"switch on a game with a string case", "branches on a game",
			`func f(g base.Game) int { switch g { case "rom2": return 2 }; return 1 }`},
		{"save tag string compared", "compares a game",
			`func f(tag string) bool { return tag == "rom2" }`},
		{"archive name compared", "compares a game",
			`func f(archive string) bool { return archive == "video" }`},
		{"string form of a game compared", "compares a game",
			`func f(g base.Game) bool { return string(g) == "rom2" }`},
		{"strings prefix of a game", "compares a game",
			`func f(g base.Game) bool { return strings.HasPrefix(string(g), "rom2") }`},
		{"map keyed by game", "looks up a game",
			`var m = map[base.Game]int{}
func f(g base.Game) int { return m[g] }`},
		{"edition campaign compared", "compares a game",
			`func f(g base.Game) bool { return g.Edition().Campaign == base.CampaignDestinations }`},
		{"edition flag", "reads an edition flag",
			`func f(g base.Game) bool { return g.Edition().SecondMaps }`},
		{"bool method on a game", "asks a game",
			`func f(g base.Game) bool { return g.Known() }`},
		{"bool function of two games", "asks a game",
			`func f(a, b base.Game) bool { return base.SameGame(a, b) }`},
		{"local defined from an edition archive", "compares a game",
			`func f(g base.Game, archive string) string { if only := g.Edition().CutsceneArchive; only != "" { return only }; return archive }`},
		{"switch on a string tag with a game case", "branches on a game",
			`func f(tag string) int { switch tag { case "rom1": return 1 }; return 0 }`},
		{"bool field named for a game", "reads a game flag",
			`type audience struct{ SecondGame bool }
func f(a audience) bool { return a.SecondGame }`},
		{"method value of a game question", "takes a game question as a value",
			`func f(g base.Game) bool { known := g.Known; return known() }`},
		{"string-keyed map looked up by a save tag", "looks up a game",
			`var m = map[string]int{}
func f(g base.Game) int { return m[string(g.Edition().SaveTag)] }`},
		{"printed game compared with a variable", "compares a game",
			`func f(g base.Game, x string) bool { return fmt.Sprint(g) == x }`},
		{"deep equality of two games", "compares a game",
			`func f(a, b base.Game) bool { return reflect.DeepEqual(a, b) }`},
		{"length of the edition town", "measures a game",
			`func f(g base.Game) bool { return len(g.Edition().Town) > 0 }`},
		{"length of the edition cutscene archive", "measures a game",
			`func f(g base.Game) bool { return len(g.Edition().CutsceneArchive) == 0 }`},
		{"games ordered", "compares a game",
			`func f(g base.Game) bool { return g > base.Game("") }`},
		{"int edition field used as a flag", "tests edition data as a flag",
			`func f(g base.Game) bool { return g.Edition().CompanionObjectiveMission > 0 }`},
		{"concatenations holding a game compared", "compares a game",
			`func f(g, h base.Game) bool { return g.Edition().Town+"x" == h.Edition().Town+"x" }`},
	}
	for _, m := range mutations {
		got := profileMutationScan(t, m.body)
		found := false
		for _, f := range got {
			found = found || f.Shape == m.shape
		}
		if !found {
			t.Errorf("%s: no %q finding (all: %v)", m.name, m.shape, got)
		}
	}
	clean := `func f(g base.Game, rooms map[string]int) (string, int, base.Game) {
	e := g.Edition()
	return e.CutsceneArchive + ".res", rooms[e.Rooms] + rooms[e.Town] + e.CompanionObjectiveMission, e.SaveTag
}`
	if got := profileMutationScan(t, clean); len(got) != 0 {
		t.Errorf("reading edition data was found as a choice: %v", got)
	}
}

// TestProfileAllowedNamesAFunction: an allowed file and function admits only
// that function's findings.
func TestProfileAllowedNamesAFunction(t *testing.T) {
	found := []ProfileFinding{{File: "pkg/game/campaignservice.go", Line: 1, Func: "campaignOf"}, {File: "pkg/game/campaignservice.go", Line: 2, Func: "other"}}
	if v := CheckProfile(found, nil); len(v) != 1 {
		t.Errorf("only the picker's finding is allowed, got %v", v)
	}
}
