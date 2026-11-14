package game

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// sevenMissionMod is a mod of map files named as seven campaign missions, a
// missions file none, in the editor over a campaign that holds all seven.
func sevenMissionMod(t *testing.T, entry string) (*testPlayFixture, *ui.MapEditor, string) {
	t.Helper()
	f, e := graphFixtureIn(t, entry, nil, nil)
	numbers := []int{10, 20, 30, 31, 40, 41, 71}
	var nodes []synth.RegNode
	files := map[string]bool{"10.alm": true, "20.alm": true, "30.alm": true, "31.almx": true, "40.almx": true, "41.almx": true, "71.alm": true}
	for _, n := range numbers {
		nodes = append(nodes, synth.RegNode{Name: fmt.Sprintf("Mission%d", n), Kind: kindDir, Children: []synth.RegNode{{Name: "Payment", Kind: kindInt, Int: 100}}})
		address, _ := MissionMap(n)
		b := synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: fmt.Sprintf("map %d", n)})
		f.x.archives.Containers.Overlay(address, func() ([]byte, error) { return b, nil })
	}
	reg := synth.Reg(kindRoot, nodes)
	f.x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return reg, nil })
	maps := map[string][]byte{}
	for name := range files {
		maps[name] = synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: name})
	}
	dir := fileFormMod(t, maps)
	for _, n := range numbers {
		for _, lang := range []string{"en", "ru"} {
			writeFileAll(t, filepath.Join(dir, fmt.Sprintf("%d.text", n), lang, "title.txt"), []byte("Title"))
		}
	}
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	return f, e, dir
}

// The missions page of a mod that replaces seven missions by file name
// offers all seven, in the order the maps page lists them.
func TestModWindowMissionsPageOffersEveryMissionOfTheMod(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			f, e, _ := sevenMissionMod(t, entry)
			if f.x.mod == nil || f.x.mod.refusal != "" {
				t.Fatalf("the mod is not open: %+v", f.x.mod)
			}
			s := e.Document().Edits.(*mapInspectionEdits)
			var missions, maps []string
			for _, c := range func() []ui.InspectionChoice {
				for _, field := range s.WorkspaceFields("package") {
					if field.Key == "source" {
						return field.Choices
					}
				}
				return nil
			}() {
				missions = append(missions, strings.SplitN(c.Label, " ", 2)[0])
			}
			for _, c := range s.ModMaps().Choices {
				maps = append(maps, strings.SplitN(c.Label, ":", 2)[0])
			}
			if got, want := strings.Join(missions, " "), strings.Join(maps, " "); got != want || len(maps) != 7 {
				t.Fatalf("the missions list %q, the maps list %q", got, want)
			}
			// Each is a row of the page, whichever the selected one is.
			e.Layout(1280, 800)
			if !e.ModWindowOpen() {
				_ = e.Key("package")
			}
			// Turn to the page whose rows read "<file> / <mode>".
			for turn := 0; turn < 5 && !strings.Contains(strings.Join(e.PopupLines(), "\n"), maps[0]+" /"); turn++ {
				_ = e.Key("package")
			}
			lines := strings.Join(e.PopupLines(), "\n")
			for _, name := range maps {
				if !strings.Contains(lines, name+" /") {
					t.Fatalf("the missions page has no row of %s:\n%s", name, lines)
				}
			}
		})
	}
}
