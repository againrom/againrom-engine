package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestLiveTipsLiveOnlyInTheTipTables measures the real tree: no production
// pkg/ source and no description outside the tip tables names a tip text or
// a tip rectangle.
func TestLiveTipsLiveOnlyInTheTipTables(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	descriptions, err := LoadDescriptions(root)
	if err != nil {
		t.Fatal(err)
	}
	tables := 0
	for name := range descriptions {
		if strings.HasPrefix(name, TipTableDir) {
			tables++
		}
	}
	if tables != 2 {
		t.Errorf("%d tip tables under %s, want one per game (2)", tables, TipTableDir)
	}
	for _, v := range CheckTipHomes(files, descriptions) {
		t.Error(v)
	}
}

// TestTipHomesAreFound: each shape of a second tip home is a finding, and
// what the tables, a hover slot and the generator's cycles hold is not.
func TestTipHomesAreFound(t *testing.T) {
	goFiles := map[string]string{
		"pkg/ui/panel.go": `package ui
import "image"
var MissionTipRect = image.Rect(10, 20, 370, 188)
const roomRectForTip = 4
func f() string { return "main/text/tips/inn.txt" }
func g() string { return "main/text/battle/m%d/tips%02d.txt" }
func h() string { return "tips8" }
type T struct { A string ` + "`json:\"tips\"`" + ` }
func ok() string { return "tipstore" }
`,
	}
	jsonFiles := map[string]string{
		"pkg/game/towns/x.json":      `{"tip": {"text": "a", "rect": [0, 0, 1, 1]}, "hotspots": [{"tip": 236}], "rooms": [{"tip": {"text": "b"}}]}`,
		"pkg/game/generators/x.json": `{"tip-cycles": {"pre-create-cycle": [[1]]}, "tips": [], "words": {"tip-font": "text"}, "s": {"file": "main/text/town.txt", "section": "tips5"}}`,
		"pkg/game/tips/x.json":       `{"tips": "x", "rows": [{"text": {"file": "main/text/tips/a.txt"}}]}`,
	}
	got := map[string]bool{}
	for _, v := range CheckTipHomes(goFiles, jsonFiles) {
		got[v.From] = true
	}
	want := []string{
		"pkg/ui/panel.go:3", "pkg/ui/panel.go:4", "pkg/ui/panel.go:5", "pkg/ui/panel.go:6", "pkg/ui/panel.go:7",
		"pkg/game/towns/x.json:.tip", "pkg/game/towns/x.json:.rooms[0].tip",
		"pkg/game/generators/x.json:.tips", "pkg/game/generators/x.json:.s.section",
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("no finding at %s", w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("findings %v, want exactly %v", got, want)
	}
}
