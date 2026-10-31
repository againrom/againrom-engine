package archtest

import (
	"os"
	"strings"
	"testing"
)

const mainMenuFixture = `package menu

type Layout struct {
	Hover    [8]int
	Captions [8]string
	Actions  Actions
}

type Assets struct {
	Base     *int
	Layout   Layout
	Captions [8]*int
}

func Load(set int, l Layout) (*Assets, error) { return &Assets{Layout: l}, nil }
`

const mainMenuImportLine = `import "againrom/pkg/render/menu"` + "\n"

// TestCheckMainMenuNamesEachShape plants each shape; the controls pass.
func TestCheckMainMenuNamesEachShape(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a second loader in the menu package", "pkg/render/menu/second.go",
			"package menu\nfunc LoadSecond(set int) (*Assets, error) { return nil, nil }\n", "LoadSecond is a second main-menu loader"},
		{"a loader method in the menu package", "pkg/render/menu/second.go",
			"package menu\nfunc (a *Assets) LoadMore() {}\n", "LoadMore is a second main-menu loader"},
		{"a nil check on an Assets field", "pkg/render/menu/state.go",
			"package menu\nfunc f(a *Assets) bool { return a.Base == nil }\n", "the Base field is compared with nil"},
		{"a nil check on an indexed field", "pkg/render/menu/state.go",
			"package menu\nfunc f(a *Assets) bool { return nil != a.Captions[1] }\n", "the Captions field is compared with nil"},
		{"a default chosen by an empty value", "pkg/render/menu/actions.go",
			"package menu\nfunc f(a *Assets) bool { return a.Layout.Actions == (Actions{}) }\n", "the Actions field is compared with nil or an empty value"},
		{"a default chosen by a row name", "pkg/render/menu/actions.go",
			"package menu\nfunc f(a *Assets) bool { return a.Layout.Actions == SecondActions }\n", "the Actions field is compared with SecondActions"},
		{"an Assets literal beside the loader", "pkg/render/menu/second.go",
			"package menu\nvar a = &Assets{}\n", "an Assets literal outside menu.go"},
		{"a second loader named outside", "pkg/game/x.go",
			"package game\n" + mainMenuImportLine + "var f = menu.LoadSecond\n", "menu.LoadSecond is a second main-menu loader"},
		{"a second loader named through an alias", "pkg/game/x.go",
			"package game\nimport m \"againrom/pkg/render/menu\"\nfunc f() { m.LoadSecond(nil) }\n", "m.LoadSecond is a second main-menu loader"},
		{"a second loader named in a test", "pkg/ui/x_test.go",
			"package ui\n" + mainMenuImportLine + "func f() { menu.LoadSecond(nil) }\n", "menu.LoadSecond is a second main-menu loader"},
		{"a dot import", "pkg/game/x.go",
			"package game\nimport . \"againrom/pkg/render/menu\"\n", "a dot import of the menu package"},
		{"a loader chosen per game as a field", "pkg/game/x.go",
			"package game\n" + mainMenuImportLine + "type files struct{ loadMenu func(int) (*menu.Assets, error) }\n", "a function type returning *menu.Assets"},
		{"a loader chosen per game as a variable", "pkg/ui/x.go",
			"package ui\n" + mainMenuImportLine + "var pick = map[int]func() *menu.Assets{}\n", "a function type returning *menu.Assets"},
		{"a helper that builds the menu", "pkg/game/x.go",
			"package game\n" + mainMenuImportLine + "func menuFor(g int) (*menu.Assets, error) { return nil, nil }\n", "menuFor returns *menu.Assets"},
		{"the wrapper in another file", "pkg/game/other.go",
			"package game\n" + mainMenuImportLine + "func LoadMainMenuArt() (*menu.Assets, error) { return nil, nil }\n", "LoadMainMenuArt returns *menu.Assets"},
		{"a second call of the loader", "pkg/game/x.go",
			"package game\n" + mainMenuImportLine + "func f() { menu.Load(nil, menu.Layout{}) }\n", "menu.Load outside pkg/game/menuart.go"},
		{"a row named outside the game table", "pkg/ui/x.go",
			"package ui\n" + mainMenuImportLine + "var l = menu.SecondLayout\n", "menu.SecondLayout outside pkg/game/gamefiles.go"},
		{"an action table named outside the game table", "pkg/game/x.go",
			"package game\n" + mainMenuImportLine + "var l = menu.FirstActions\n", "menu.FirstActions outside pkg/game/gamefiles.go"},
		{"an Assets literal outside the menu package", "pkg/game/x.go",
			"package game\n" + mainMenuImportLine + "var a = &menu.Assets{}\n", "a menu.Assets literal outside the menu package"},
		{"the wrapper beside the one caller", "pkg/game/menuart.go",
			"package game\n" + mainMenuImportLine + "func LoadMainMenuArt() (*menu.Assets, error) { return menu.Load(nil, menu.Layout{}) }\n", ""},
		{"the game table naming the rows", "pkg/game/gamefiles.go",
			"package game\n" + mainMenuImportLine + "var a, b = menu.FirstLayout, menu.SecondLayout\n", ""},
		{"a test building a menu", "pkg/ui/x_test.go",
			"package ui\n" + mainMenuImportLine + "func load() (*menu.Assets, error) { a := &menu.Assets{}; _ = menu.FirstLayout; return menu.Load(nil, menu.FirstLayout) }\n", ""},
		{"a test comparing a field", "pkg/render/menu/x_test.go",
			"package menu\nfunc f(a *Assets) bool { return a.Base == nil }\n", ""},
		{"a field read that is no comparison", "pkg/render/menu/state.go",
			"package menu\nfunc f(a *Assets) int { n := len(a.Layout.Captions); return n }\n", ""},
		{"a comparison of a value that is no field", "pkg/render/menu/state.go",
			"package menu\nfunc f(a *Assets, b int) bool { return a == nil || b == 0 }\n", ""},
		{"the LOAD GAME button index", "pkg/ui/x.go",
			"package ui\n" + mainMenuImportLine + "var b = menu.LoadGameButton\n", ""},
		{"a file that names no menu", "pkg/game/x.go",
			"package game\nfunc LoadSecond() {}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckMainMenu(map[string]string{mainMenuFile: mainMenuFixture, tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckMainMenu = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckMainMenu = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestCheckMainMenuNeedsExactlyOneLoad(t *testing.T) {
	vs := CheckMainMenu(map[string]string{mainMenuFile: "package menu\ntype Layout struct{ Actions int }\ntype Assets struct{ Layout Layout }\n"})
	if len(vs) != 1 || !strings.Contains(vs[0].Reason, "exactly one") {
		t.Fatalf("a menu package with no Load = %v, want one violation naming the count", vs)
	}
	if vs := CheckMainMenu(map[string]string{"pkg/game/x.go": "package game\n"}); len(vs) != 1 || !strings.Contains(vs[0].Reason, "no menu.go scanned") {
		t.Fatalf("CheckMainMenu without menu.go = %v, want one violation", vs)
	}
}

func TestMainMenuHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadMainMenuSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckMainMenu(files) {
		t.Errorf("%s", v)
	}
}
