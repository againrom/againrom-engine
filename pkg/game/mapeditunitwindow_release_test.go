package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/internal/firstgame"
	"againrom/internal/witness"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// TestReleaseMapEditorUnitWindowFrames opens the unit window of the first
// person and the first creature of an installed map in its normal and its
// detailed mode, rendered headless at 1920x1080, and writes each frame as a
// PNG into AGAINROM_UNIT_WINDOW_FRAMES. Without that directory it only drives
// the same steps: both units open the one unit window, in both modes.
func TestReleaseMapEditorUnitWindowFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir, lang := os.Getenv("AGAINROM_UNIT_WINDOW_FRAMES"), installLanguage(x)
	now := time.Unix(1, 0)
	for _, who := range []string{"human", "animal"} {
		e := x.Editor()
		// The right panel minimized: the unit window floats at its own size.
		e.SetPanelProfile([]byte(`{"right":{"minimized":true,"width":520,"view":"plain"}}`), nil, nil)
		e.Layout(1920, 1080)
		if err := e.Open("scenario/10.alm"); err != nil {
			t.Fatal(err)
		}
		closeMapPanel(t, e)
		s := objectEdits(t, e)
		m, err := s.model.Map()
		if err != nil {
			t.Fatal(err)
		}
		unit := -1
		for i, u := range m.AuthoredUnits {
			arm := mapload.Resolve(u, x.table).Arm
			if who == "human" && (arm == mapload.ArmHumansByType || arm == mapload.ArmServerID) || who == "animal" && arm == mapload.ArmUnits {
				unit = i
				break
			}
		}
		if unit < 0 {
			t.Fatalf("the map has no %s", who)
		}
		e.Select(recordOf(e.Document(), "Unit", unit), true)
		for _, mode := range []string{"normal", "detailed"} {
			if mode == "detailed" {
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlViewSwitch})
				if !ok {
					t.Fatalf("%s: the unit window offers no view switch", who)
				}
				p := r.Min.Add(r.Size().Div(2))
				e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
				e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
				now = now.Add(time.Second)
			}
			if e.PlainView() != (mode == "normal") {
				t.Fatalf("%s: the window is not in its %s mode", who, mode)
			}
			screen := ebiten.NewImage(1920, 1080)
			e.Draw(screen)
			screen.Dispose()
			pic, _, err := e.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			if dir != "" {
				witness.PNG(t, filepath.Join(dir, fmt.Sprintf("unit-%s-%s-%s.png", who, mode, lang)), pic)
			}
		}
	}
}
