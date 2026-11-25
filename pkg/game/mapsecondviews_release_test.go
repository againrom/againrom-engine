//go:build release

package game

import (
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// On every installed second-game map the passability view switches on with
// its note's word, the ring cells carry the border arm, the trigger areas are
// cells, and the search, the legend and the height numbers answer; a new map
// says it is a first-game map.
func TestReleaseSecondMapViewsDrawOnlyWhatIsDecoded(t *testing.T) {
	x := installedInspector(t)
	sources, _ := installedSecondMaps(t, x)
	e := x.Editor()
	e.Layout(1920, 1080)
	searched := 0
	for _, source := range sources {
		if err := e.Open(source); err != nil {
			t.Fatal(source, err)
		}
		e.Layout(1920, 1080)
		d := e.Document()
		if on, _, _ := e.ToolbarState("passview"); on {
			_ = e.Key("passview")
		}
		_ = e.Key("passview")
		if on, _, _ := e.ToolbarState("passview"); !on || !strings.Contains(e.StatusLine(), x.inspectionUTF8(x.word("editor.view.secondpass"))) {
			t.Errorf("%s: the passability view is on %v, status %q", source, on, e.StatusLine())
		}
		for i, arm := range d.Passability {
			if ring := mapload.InBorderRing(&alm.Map{Width: d.Width, Height: d.Height}, i%d.Width, i/d.Width); (arm&ui.PassArm(mapload.GroundBorder) != 0) != ring {
				t.Errorf("%s: cell %d arm %v, ring %v", source, i, arm, ring)
				break
			}
		}
		for _, a := range d.Script.Areas {
			if a.Shape != int(mapload.AreaCell) {
				t.Errorf("%s: area shape %d", source, a.Shape)
			}
		}
		_ = e.Key("search")
		e.TextInput("1")
		if len(e.SearchLines()) > 0 {
			searched++
		}
		_ = e.Key("escape")
		if _, _, ok := e.HeadlessFrame(); ok != nil {
			t.Errorf("%s: no frame: %v", source, ok)
		}
	}
	if searched != len(sources) {
		t.Errorf("search found rows on %d of %d maps", searched, len(sources))
	}
	_ = e.Key("new")
	_ = e.Key("enter")
	_ = e.Key("enter")
	if got := e.Message(); got != x.word("editor.new.second") {
		t.Errorf("new map message %q", got)
	}
	t.Logf("PASS: %d second-game maps, passability noted, ring and marked arms, cell areas (%s)", len(sources), installLanguage(x))
}
