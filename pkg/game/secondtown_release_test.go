package game

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

// secondTownMaskPoint is the middle pixel, in raster order, of one square
// mask code.
func secondTownMaskPoint(t *testing.T, mask *image.Paletted, code uint8) image.Point {
	t.Helper()
	var points []image.Point
	for y := mask.Rect.Min.Y; y < mask.Rect.Max.Y; y++ {
		for x := mask.Rect.Min.X; x < mask.Rect.Max.X; x++ {
			if mask.ColorIndexAt(x, y) == code {
				points = append(points, image.Pt(x, y))
			}
		}
	}
	if len(points) == 0 {
		t.Fatalf("square mask has no code %#x", code)
	}
	return points[len(points)/2]
}

func secondTownShot(t *testing.T, app *ui.App, name string) *image.RGBA {
	t.Helper()
	pix, note, err := app.HeadlessFrame()
	if err != nil || note != "" || pix == nil {
		t.Fatalf("%s frame: %q %v", name, note, err)
	}
	if dir := os.Getenv("AGAINROM_SHOT_DIR"); dir != "" {
		out, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer out.Close()
		if err := png.Encode(out, pix); err != nil {
			t.Fatal(err)
		}
	}
	return pix
}

func secondTownClick(t *testing.T, app *ui.App, p image.Point) {
	t.Helper()
	for _, edge := range []string{"hover", "press", "release"} {
		if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
}

// TestReleaseSecondFirstTownSquareAndTavern plays a fresh second-game
// campaign through its first town by the square's own mask: the square and
// its highlights from the install's town art, the closed shop and the
// roomless school, the tavern over the install's inn art, the mission-10
// TALK, and the gates to the destination list.
func TestReleaseSecondFirstTownSquareAndTavern(t *testing.T) {
	f := secondGameFront(t)
	square, tavern := f.TownSquareArt.Value(), f.TownTavernArt.Value()
	if square == nil || square.Background == nil || square.Mask == nil || square.Exterior == nil {
		t.Fatal("second-game town square art did not load", f.TownSquareArt.Err())
	}
	if tavern == nil || tavern.Center == nil || tavern.CommandUpper == nil {
		t.Fatal("second-game tavern art did not load", f.TownTavernArt.Err())
	}
	app := f.App("first town")
	app.Layout(640, 480)
	screen := f.TownScreen().(*secondCampaignScreen)
	app.SetTown(screen)
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("new game did not reach the first town", err, app.Screen())
	}
	c := f.Town.second
	if c.current != firstSecondTown || !screen.AtTownSquare() || screen.TownSquareView().Art != square {
		t.Fatal("first town did not open the square", c.current)
	}
	pix := secondTownShot(t, app, "square")
	if pix.Bounds().Size() != image.Pt(640, 480) {
		t.Fatal("square frame size", pix.Bounds())
	}
	for _, p := range []image.Point{{0, 0}, {639, 0}, {600, 10}, {0, 240}} {
		r, g, b, _ := square.Background.At(p.X, p.Y).RGBA()
		if got := pix.RGBAAt(p.X, p.Y); got.R != uint8(r>>8) || got.G != uint8(g>>8) || got.B != uint8(b>>8) {
			t.Fatalf("square pixel %v is %v, not the installed picture", p, got)
		}
	}
	for _, door := range []struct {
		name     string
		code     uint8
		selector int
	}{{"tavern", 0x80, 2}, {"shop", 0x90, 1}, {"school", 0xc0, -1}, {"gates", 0xa0, 8}} {
		p := secondTownMaskPoint(t, square.Mask, door.code)
		if err := app.HeadlessPointer("hover", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		if got := screen.TownSquareView().Selector; got != door.selector {
			t.Fatalf("%s hover selector %d, want %d", door.name, got, door.selector)
		}
		secondTownShot(t, app, "square-hover-"+door.name)
	}
	for _, code := range []uint8{0x90, 0xc0, 0xa0} {
		secondTownClick(t, app, secondTownMaskPoint(t, square.Mask, code))
		if c.current != firstSecondTown || c.room != secondTownSquare || !screen.AtTownSquare() {
			t.Fatalf("closed door %#x left the square", code)
		}
	}
	secondTownClick(t, app, secondTownMaskPoint(t, square.Mask, 0x80))
	if c.room != secondTownInn || !screen.AtTownSurface() {
		t.Fatal("tavern click did not open the tavern")
	}
	v := screen.TownSurface()
	if v.TavernArt != tavern || len(v.Cells) != len(c.speakers()) {
		t.Fatal("tavern does not draw the installed inn art and speakers", len(v.Cells))
	}
	for i, o := range c.speakers() {
		if v.Cells[i].Semantic != fmt.Sprintf("NPC %d", o.npc) {
			t.Fatalf("cell %d is %q", i, v.Cells[i].Semantic)
		}
	}
	if !v.RosterUnpainted || len(v.Buttons) != 4 || v.Buttons[0].Enabled || v.Buttons[1].Enabled || v.Buttons[2].Enabled || !v.Buttons[3].Enabled {
		t.Fatal("tavern entry selection or buttons", v.RosterUnpainted, v.Buttons)
	}
	secondTownShot(t, app, "tavern")
	if err := app.HeadlessActivate("NPC 517"); err != nil {
		t.Fatal(err)
	}
	if !c.has(secondLocation{kind: 1, id: 10}) {
		t.Fatal("TALK did not make mission 10 available")
	}
	secondTownShot(t, app, "tavern-talk")
	for page := 0; app.HeadlessActivate("notice") == nil; page++ {
		if page == 64 {
			t.Fatal("TALK exceeded 64 pages")
		}
	}
	selected := screen.TownSurface()
	for _, cell := range selected.Cells {
		if cell.Selected != (cell.Semantic == "NPC 517") || selected.RosterUnpainted {
			t.Fatal("the spoken-to speaker is not the one selection", cell.Semantic, cell.Selected)
		}
	}
	secondTownShot(t, app, "tavern-selected")
	if err := app.HeadlessKey("escape"); err != nil || !screen.AtTownSquare() {
		t.Fatal("tavern did not return to the square", err)
	}
	secondTownClick(t, app, secondTownMaskPoint(t, square.Mask, 0xa0))
	if c.current != (secondLocation{}) || screen.AtTownSquare() || app.Screen() != ui.ScreenTown {
		t.Fatal("open gates did not leave town", c.current)
	}
	if err := app.HeadlessActivate("mission 10"); err != nil {
		t.Fatal("destination list lacks mission 10", err)
	}
	secondTownShot(t, app, "destinations")
}

// TestReleaseSecondFirstTownColdLoadKeepsTheRoom saves the first town at
// the square and in the tavern and loads each in a fresh front end.
func TestReleaseSecondFirstTownColdLoadKeepsTheRoom(t *testing.T) {
	for _, inn := range []bool{false, true} {
		f, app := secondTownNew(t, inn, true)
		out := t.TempDir()
		before := captureSecondCampaign(f.Town.second)
		secondTownNamedSave(t, f, app, out, "Room")
		cold, _ := secondTownCold(t, out, "Room.sav")
		after := captureSecondCampaign(cold.Town.second)
		if after.Current != before.Current || *after.Room != *before.Room || after.Bank != before.Bank || len(after.Available) != len(before.Available) {
			t.Fatalf("cold LOAD changed the town: %+v %+v", after.Current, after.Available)
		}
		screen := cold.TownScreen().(*secondCampaignScreen)
		if inn != screen.AtTownSurface() || inn == screen.AtTownSquare() {
			t.Fatalf("cold LOAD of inn=%v landed square=%v tavern=%v", inn, screen.AtTownSquare(), screen.AtTownSurface())
		}
	}
}
