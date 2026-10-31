package archtest

import (
	"os"
	"strings"
	"testing"
)

// optionConsts is the constant file the scan reads its names from.
const optionConsts = "package ui\ntype GameOption uint8\nconst (\n\tGameOptionDayNight GameOption = iota\n\tGameOptionHealth\n\tGameOptionSmoothing\n\tGameOptionAnimation\n\tgameOptionCount\n)\n" +
	"type GameOptionValues [gameOptionCount]int\n" +
	"var gameOptionTable = [gameOptionCount]gameOptionDefinition{GameOptionDayNight: {needs: []GameOption{GameOptionHealth}}}\n"

// TestCheckGameOptionRowsNamesEachShape plants each shape of a per-option
// decision outside the two tables and shows the scan names it.
func TestCheckGameOptionRowsNamesEachShape(t *testing.T) {
	game := func(body string) string {
		return "package game\nimport \"againrom/pkg/ui\"\nfunc f(o ui.GameOption, v ui.GameOptionValues) int {\n" + body + "\n\treturn 0\n}\n"
	}
	cases := []struct {
		name, file, src string
		want            string // a substring of the one violation; "" for none
	}{
		{"a switch over an option parameter", "pkg/game/a.go", game("\tswitch o {\n\tdefault:\n\t}"), "a switch over a GameOption value"},
		{"a switch over a conversion", "pkg/game/a.go", game("\tswitch ui.GameOption(v[0]) {\n\t}"), "a switch over a GameOption value"},
		{"a case naming a constant", "pkg/game/a.go", game("\tswitch v[0] {\n\tcase int(ui.GameOptionHealth):\n\t}"), "a case names GameOptionHealth"},
		{"an if comparing with a constant", "pkg/game/a.go", game("\tif o == ui.GameOptionHealth {\n\t}"), "a comparison with GameOptionHealth"},
		{"a comparison through a conversion", "pkg/game/a.go", game("\ti := 0\n\t_ = i < int(ui.GameOptionSmoothing)"), "a comparison with GameOptionSmoothing"},
		{"an option value against a literal", "pkg/game/a.go", game("\tif o == 3 {\n\t}"), "a GameOption value with a literal"},
		{"a range variable against a literal", "pkg/game/a.go", game("\tfor _, p := range ui.GameOptions() {\n\t\tif 2 < p {\n\t\t}\n\t}"), "a GameOption value with a literal"},
		{"arithmetic on a constant", "pkg/ui/a.go", "package ui\nfunc g(i int) GameOption { return GameOptionSmoothing + GameOption(i) }\n", "arithmetic on GameOptionSmoothing"},
		{"an index by a constant", "pkg/game/a.go", game("\t_ = v[ui.GameOptionAnimation]"), "an index by GameOptionAnimation"},
		{"an index of another array", "pkg/ui/a.go", "package ui\nfunc g(w GameOptionWords) string { return w.Labels[GameOptionHealth] }\n", "an index by GameOptionHealth"},
		{"a positional values literal", "pkg/game/a.go", game("\tv = ui.GameOptionValues{1, 1}"), "a positional GameOptionValues literal"},
		{"an option list literal", "pkg/game/a.go", game("\tfor range [...]ui.GameOption{ui.GameOptionSmoothing} {\n\t}"), "a composite literal lists GameOptionSmoothing"},
		{"a per-option mapping", "pkg/game/a.go", game("\t_ = PendingGameOption{ui.GameOptionHealth, 1}"), "a composite literal lists GameOptionHealth"},
		{"a keyed literal outside the table", "pkg/game/a.go", game("\t_ = [2]int{ui.GameOptionHealth: 1}"), "a composite literal keyed by GameOptionHealth"},
		{"a package-level list", "pkg/game/a.go", "package game\nvar graphics = []ui.GameOption{ui.GameOptionSmoothing}\n", "a composite literal lists GameOptionSmoothing"},
		{"a menu action per option", "pkg/ui/a.go", "package ui\nconst (\n\tgameMenuDayNight gameMenuAction = iota\n\tgameMenuHealth\n)\n", "a menu action named for GameOptionHealth"},
		{"the option action block", "pkg/ui/a.go", "package ui\nconst (\n\tgameMenuTooltipDelay gameMenuAction = iota\n\tgameMenuDayNight\n)\n", ""},
		{"a function in a table file", "pkg/game/gameoptions.go", game("\t_ = v[ui.GameOptionAnimation]"), "an index by GameOptionAnimation"},
		{"the game table itself", "pkg/game/gameoptions.go",
			"package game\nvar gameOptionRows = [4]gameOptionRow{ui.GameOptionHealth: {initial: ui.GameOptionHealth.Default}}\n", ""},
		{"a constant passed to a call", "pkg/ui/a.go", "package ui\nfunc g(f *flow) { f.cycleGameOption(GameOptionHealth) }\n", ""},
		{"a walk by variable", "pkg/game/a.go", game("\tfor o := range v {\n\t\t_ = v[o]\n\t}"), ""},
		{"a test file", "pkg/game/a_test.go", game("\tif o == ui.GameOptionHealth {\n\t}"), ""},
		{"another package", "pkg/render/a.go", game("\tif o == ui.GameOptionHealth {\n\t}"), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckGameOptionRows(map[string]string{gameOptionConstFile: optionConsts, tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckGameOptionRows = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) {
				t.Fatalf("CheckGameOptionRows = %v, want one naming %q", vs, tc.want)
			}
		})
	}
	if vs := CheckGameOptionRows(map[string]string{"pkg/game/a.go": game("")}); len(vs) != 1 {
		t.Fatalf("CheckGameOptionRows without the constants = %v, want one violation", vs)
	}
}

// TestGameOptionsHaveOneRowEach is the live-tree half.
func TestGameOptionsHaveOneRowEach(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadTreeSources(root, "pkg/game", "pkg/ui")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckGameOptionRows(files) {
		t.Errorf("%s", v)
	}
}
