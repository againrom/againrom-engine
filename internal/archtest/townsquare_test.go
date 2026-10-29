package archtest

import (
	"os"
	"strings"
	"testing"
)

// Both games' town squares have one view adapter.
func TestTownSquareHasOneViewAdapter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckTownSquareView(files) {
		t.Error(v)
	}
}

// The guard finds a second producer of the square's view, a doll built
// elsewhere and a game or host branch in the adapter.
func TestTownSquareGuardFindsASecondAdapter(t *testing.T) {
	one := "package game\n" +
		"func (t *townScreen) TownSquareView() ui.TownSquareView { v := ui.TownSquareView{}; t.paneSquareView(&v, nil, \"\"); return v }\n" +
		"func (t *townScreen) paneSquareView(v *ui.TownSquareView, d *town.Description, room string) { v.Dolls = append(v.Dolls, ui.TownSquareDoll{}) }\n" +
		"func (v townRoomView) Paint(dst *image.RGBA) { v.t.scene(v.d, v.room).Paint(dst, \"\") }\n"
	if vs := CheckTownSquareView(map[string]string{"pkg/game/town.go": one}); len(vs) != 0 {
		t.Fatalf("one adapter: %v", vs)
	}
	for _, c := range []struct{ name, source, want string }{
		{"first-game view", "package game\nfunc (t *townScreen) firstSquareView() ui.TownSquareView { return ui.TownSquareView{Scene: townSquareScene{t}} }\n", "firstSquareView"},
		{"doll elsewhere", "package game\nfunc dolls() []ui.TownSquareDoll { return []ui.TownSquareDoll{{}} }\n", "dolls"},
		{"game branch", "package game\nfunc (v townRoomView) TipAt(p image.Point) (int, bool) { if v.t.in.profile().SecondGame() { return 0, false }; return 1, true }\n", "SecondGame"},
		{"host branch", "package game\nfunc (v townRoomView) Size() image.Point { if v.t.hostRules().Pages == \"entry\" { return image.Point{} }; return image.Pt(1, 1) }\n", "hostRules"},
	} {
		files := map[string]string{"pkg/game/town.go": one, "pkg/game/other.go": c.source}
		found := false
		for _, v := range CheckTownSquareView(files) {
			found = found || strings.Contains(v.From+" "+v.Reason, c.want)
		}
		if !found {
			t.Errorf("%s: the guard did not find %s", c.name, c.want)
		}
	}
}
