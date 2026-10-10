package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
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
	for _, f := range profileFindings(fset, files["user.go"], info, "pkg/user/user.go") {
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
