package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestLiveEngineWordsStayInTheirTables measures the real tree: no production
// file of pkg/ui or pkg/game embeds a file or branches on the font selector,
// except the allowed file and the falling debt list.
func TestLiveEngineWordsStayInTheirTables(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	found, err := LoadEngineWordsFindings(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, v := range CheckEngineWords(found, EngineWordsDebt) {
		t.Error(v)
	}
}

// TestEngineWordsFindingsSeeEachShape holds the scan against synthetic
// sources, so a scan that went blind fails here rather than passing a dirty
// tree.
func TestEngineWordsFindingsSeeEachShape(t *testing.T) {
	const src = `package ui

import (
	_ "embed"
	"againrom/pkg/render/text"
)

//go:embed words_ru.json
var table []byte

type flow struct{ menuFont *text.Font }

func (f *flow) menuSelector() int { return 0 }

func a(f *flow) bool { return f.menuFont.Selector == text.SelectorConverting }
func b(f *flow) bool { return f.menuSelector() != text.SelectorConverting }
func c(selector int) bool { return selector == 1 }
func d(f *flow) bool { return (f.menuSelector()) == 0 }
func clean(f *flow, n int) bool { return n == 1 && f.menuFont != nil }
`
	found, err := EngineWordsFindings("pkg/ui/x.go", src)
	if err != nil {
		t.Fatal(err)
	}
	var shapes []string
	for _, f := range found {
		shapes = append(shapes, f.Shape)
	}
	want := []string{shapeEmbed, shapeSelectorBranch, shapeSelectorBranch, shapeSelectorBranch, shapeSelectorBranch}
	if strings.Join(shapes, "|") != strings.Join(want, "|") {
		t.Fatalf("shapes %v, want %v (%v)", shapes, want, found)
	}

	if v := CheckEngineWords(found, nil); len(v) != 5 {
		t.Fatalf("an undebted file gave %d violations, want 5: %v", len(v), v)
	}
	debt := map[string]int{"pkg/ui/x.go": 4}
	if v := CheckEngineWords(found, debt); len(v) != 1 || !strings.Contains(v[0], shapeEmbed) {
		t.Fatalf("a debt file's embed must still fail and its branches pass: %v", v)
	}
	debt["pkg/ui/x.go"] = 5
	if v := CheckEngineWords(found[1:], debt); len(v) != 1 || !strings.Contains(v[0], "below its debt") {
		t.Fatalf("a fallen count must lower the debt: %v", v)
	}
	debt["pkg/ui/x.go"] = 3
	if v := CheckEngineWords(found[1:], debt); len(v) != 1 || !strings.Contains(v[0], "above its debt") {
		t.Fatalf("a risen count must fail: %v", v)
	}
	if v := CheckEngineWords([]EngineWordsFinding{{File: "pkg/game/townsquare.go", Line: 1, Shape: shapeEmbed}}, nil); len(v) != 0 {
		t.Fatalf("the allowed file failed: %v", v)
	}
}
