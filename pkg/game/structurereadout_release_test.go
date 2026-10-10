package game

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// readoutPicture independently draws the centred lines inside the card.
func readoutPicture(f *text.Font, lines [3]string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 160, 242))
	for i, ys := range [3]int{28, 44, 54} {
		w, _ := f.Measure(lines[i])
		x := 72 - w/2
		ink := color.RGBA{R: 185, G: 159, B: 73, A: 255}
		if i == 2 {
			ink = color.RGBA{R: 107, G: 154, B: 120, A: 255}
		}
		f.DrawFlat(img, lines[i], x+1, ys+1, color.RGBA{R: 8, G: 8, B: 8, A: 255})
		f.Draw(img, lines[i], x, ys, ink)
	}
	return img
}

// Mission 20: widget 8 shows the hovered structure's building.txt name, the
// main.txt word Health and its health over its maximum; a plain click selects a
// destructible structure, which the readout then keeps showing, and never an
// indestructible one (MENU-070, MENU-071, MENU-072, MISSION-065).
func TestReleaseWidgetEightReadsTheHoveredThenTheSelectedStructure(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("widget eight readout")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	hero := live.mission.ids[0]
	if err := a.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	buildings := LoadTextTable(f.Archives.Containers, BuildingTextPath, f.textCode())
	mainTable := LoadTextTable(f.Archives.Containers, MainTextPath, f.textCode())
	if buildings == nil || mainTable == nil {
		t.Fatal("install carries no building.txt or main.txt")
	}
	health, ok := mainTable.At(19)
	if !ok {
		t.Fatal("main.txt has no string 19")
	}
	regBytes, err := f.Archives.Containers.ReadFile("graphics/structures/structures.reg")
	if err != nil {
		t.Fatal(err)
	}
	r, err := reg.Parse(regBytes)
	if err != nil {
		t.Fatal(err)
	}
	classes, err := data.LoadStructureClasses(r)
	if err != nil {
		t.Fatal(err)
	}
	pick := func(indestructible bool) (ui.InspectionSubject, int, int, int32, int, int) {
		for _, s := range live.world.Structures() {
			kind := int32(live.mission.state.Map.Objects[s.ID].Kind)
			c, ok := classes.ByID(kind)
			if ok && s.MaxHealth != 0 && c.Usable == 0 && (c.Indestructible != 0) == indestructible {
				return ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(s.ID)}, int(s.Col), int(s.Row), kind, int(int16(s.Field42)), int(s.MaxHealth)
			}
		}
		t.Fatalf("no structure with indestructible=%v in mission 20", indestructible)
		return ui.InspectionSubject{}, 0, 0, 0, 0, 0
	}
	ref, col, row, kind, hp, maxHP := pick(false)
	name, ok := buildings.At(int(kind) - 1)
	if !ok {
		t.Fatalf("building.txt has no line %d", kind-1)
	}
	want := readoutPicture(f.tipFont(), [3]string{name, health, fmt.Sprintf("%d/%d", hp, maxHP)})
	checkLayers := func(what string, readout bool) {
		t.Helper()
		layers, err := a.HeadlessMissionColumnLayers()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		gotReadout, gotCard := slices.Contains(layers, "structureReadout"), slices.Contains(layers, "missionCard")
		if gotReadout != readout || !gotCard {
			t.Errorf("%s: Draw layers %v, want structureReadout=%v missionCard=true", what, layers, readout)
		}
		t.Logf("%s: Draw layers %v", what, layers)
	}
	check := func(what string) {
		t.Helper()
		pic, at, lines, err := a.HeadlessStructureReadout()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		width := 0
		for _, line := range [3]string{name, health, fmt.Sprintf("%d/%d", hp, maxHP)} {
			w, _ := f.tipFont().Measure(line)
			width = max(width, w)
		}
		if wantAt := image.Pt(1024-88-width/2, 526+28); at != wantAt {
			t.Errorf("%s: readout at %v, want %v", what, at, wantAt)
		}
		if lines != [3]string{name, health, fmt.Sprintf("%d/%d", hp, maxHP)} {
			t.Errorf("%s: lines %q, want building.txt[%d] %q, main.txt[19] %q and %d/%d", what, lines, kind-1, name, health, hp, maxHP)
		}
		got := image.NewRGBA(want.Bounds())
		rect := pic.Bounds().Add(at.Sub(image.Pt(1024-160, 526)))
		if !rect.In(got.Bounds()) {
			t.Fatalf("%s: readout %v lies outside the card", what, rect)
		}
		draw.Draw(got, rect, pic, pic.Bounds().Min, draw.Src)
		for y := 0; y < got.Bounds().Dy(); y++ {
			for x := 0; x < got.Bounds().Dx(); x++ {
				if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
					t.Fatalf("%s: card pixel (%d,%d) is %v, want %v", what, x, y, got.RGBAAt(x, y), want.RGBAAt(x, y))
				}
			}
		}
		checkLayers(what, true)
		// The card's body art stays under the readout: no readout pixel
		// stands on black.
		card, cardAt, err := a.HeadlessDrawnMissionCard()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		black, total := 0, 0
		for y := 0; y < pic.Bounds().Dy(); y++ {
			for x := 0; x < pic.Bounds().Dx(); x++ {
				p := image.Pt(at.X-cardAt.X+x, at.Y-cardAt.Y+y)
				if !p.In(card.Bounds()) {
					continue
				}
				c := card.RGBAAt(p.X, p.Y)
				total++
				if c.A == 0 || (c.R == 0 && c.G == 0 && c.B == 0) {
					black++
				}
			}
		}
		if total == 0 || black*10 > total {
			t.Errorf("%s: %d of %d pixels under the readout are black or empty", what, black, total)
		}
		if dir := os.Getenv("AGAINROM_HOVER_PANEL_DIR"); dir != "" {
			shot := image.NewRGBA(card.Bounds())
			draw.Draw(shot, shot.Bounds(), card, card.Bounds().Min, draw.Src)
			draw.Draw(shot, pic.Bounds().Add(at.Sub(cardAt)), pic, pic.Bounds().Min, draw.Over)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			out, err := os.Create(filepath.Join(dir, "readout-"+what+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(out, shot); err != nil {
				t.Fatal(err)
			}
			out.Close()
		}
	}

	inspectionCentre(live, col, row)
	releaseHoverInspection(t, a, live, ref)
	check("hovered")

	// A plain click: the hero's selection gives way to the structure.
	x, y, err := live.view.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if got, ok := a.HeadlessSelectedStructure(); !ok || got != ref {
		t.Fatalf("selected structure %+v,%v after a plain click, want %+v", got, ok, ref)
	}
	if sel := a.HeadlessSelection(); len(sel) != 0 {
		t.Errorf("unit selection %v after a structure click, want none", sel)
	}
	gx, gy, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", gx, gy); err != nil {
		t.Fatal(err)
	}
	check("selected")
	heroState, ok := live.world.Entity(hero)
	if !ok {
		t.Fatal("selected hero disappeared")
	}
	inspectionCentre(live, int(heroState.X), int(heroState.Y))
	releaseHoverInspection(t, a, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(hero)})
	checkLayers("hovered unit over selected structure", false)

	// A selected unit replaces it.
	if err := a.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.HeadlessSelectedStructure(); ok {
		t.Error("the structure stays selected beside a selected unit")
	}
	gx, gy, err = a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", gx, gy); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := a.HeadlessStructureReadout(); err == nil {
		t.Error("a readout is drawn with a unit selected and nothing hovered")
	}
	checkLayers("restored selected unit", false)

	// An indestructible class is not selected: the click keeps the hero.
	well, wcol, wrow, _, _, _ := pick(true)
	inspectionCentre(live, wcol, wrow)
	x, y, err = live.view.InspectionPoint(well)
	if err != nil {
		t.Skipf("the indestructible structure has no exposed pixel: %v", err)
	}
	for _, edge := range []string{"hover", "press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if got, ok := a.HeadlessSelectedStructure(); ok {
		t.Errorf("an indestructible structure was selected: %+v", got)
	}
	if sel := a.HeadlessSelection(); !slices.Equal(sel, []uint32{uint32(hero)}) {
		t.Errorf("selection %v after a click on an indestructible structure, want the hero %d", sel, hero)
	}
}
