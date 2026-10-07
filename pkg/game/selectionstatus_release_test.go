package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/mapload"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// The mission column's upper character pane states the selection as global
// strings 47 to 50 give it (`TEXT-UI-047`): two lines for no character
// selected, and from two the two words and the count each on a line of its own,
// neither for one, which shows the unit itself. The witness selects through the
// pointer as a player does: a press on one party member, Shift presses on four
// more, and a right press on empty ground. It reads both boxes after every
// input, as the drawn frames do, on the installed art and font. The expected
// pictures are the installed pane art drawn with no lines and then the lines
// drawn directly where the owner's capture of the original EN game has them
// (`DIV-1795`): the card font in gold over a one-pixel flat shadow, each line
// centred on the middle of the 176-pixel column, the first line's cell 54
// pixels below the pane's top and 12 pixels between the tops of the lines. The
// lower statistics card is the bare page in every state. On EN the drawn
// rectangle of every line is also pinned to the capture's own ink rectangles
// for the five-selected and the none-selected pictures. The words come from the
// claim and not from the install's own reader.
func TestReleaseMissionPaneAndCardStateTheSelectedCount(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)

	ru := f.textSelector() == text.SelectorConverting
	plain := [4]string{"No units", "selected", "Units", "selected:"}
	if ru {
		plain = [4]string{"Персонаж", "не выбран", "Выбрано", "персонажей:"}
	}
	words := plain
	if ru {
		for i, w := range plain {
			b, err := charmap.CodePage866.NewEncoder().String(w)
			if err != nil {
				t.Fatal(err)
			}
			words[i] = b
		}
	}
	linesOf := func(w [4]string, count int) [3]string {
		switch {
		case count == 0:
			return [3]string{w[0], w[1]}
		case count >= 2:
			return [3]string{w[2], w[3], fmt.Sprint(count)}
		}
		return [3]string{}
	}
	lines := func(count int) [3]string { return linesOf(words, count) }

	party := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	group := make([]mapload.PartyMember, 5)
	for i := range group {
		group[i] = party[0]
	}
	group = mapload.CloneParty(group)
	for i := range group {
		group[i].ID = fmt.Sprintf("status-line-%d", i)
	}
	app := f.App("mission selection lines")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, group)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	ids := live.mission.ids
	if len(ids) != len(group) {
		t.Fatalf("the mission seated %d of the 5 party members", len(ids))
	}
	step := func() {
		t.Helper()
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
	pointer := func(x, y int, edges ...string) {
		t.Helper()
		for _, edge := range edges {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
	}

	for range 4 {
		step()
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	first, ok := live.entity(ids[0])
	if !ok {
		t.Fatal("the first party member has no entity")
	}
	inspectionCentre(live, int(first.X), int(first.Y))
	step()
	gx, gy, err := app.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	// The pointer rests on the command panel while a box is read: over a unit
	// or a structure on the map the boxes show that one instead of the
	// selection.
	cx, cy, err := app.HeadlessCommandPoint(0)
	if err != nil {
		t.Fatal(err)
	}
	away := func() {
		t.Helper()
		pointer(cx, cy, "hover")
	}

	chrome := f.Font.Value()
	font := f.tipFont()
	if chrome == nil || font == nil {
		t.Fatal("the install gave the mission no font")
	}
	if font == chrome || font.Height() != 10 {
		t.Fatalf("the caption font is the %d-row chrome font, want the install's small 10-row font", font.Height())
	}
	ink := color.RGBA{R: 0xbd, G: 0x9e, B: 0x4a, A: 0xff}
	shadow := color.RGBA{R: 8, G: 8, B: 8, A: 0xff}
	panes := f.characterPanes()
	const seam, bodyW, bodyH = 16, 160, 242
	local := image.Rect(seam, 0, seam+bodyW, bodyH)
	// stated draws a state's lines by this test's own arithmetic: each line
	// centred on the column's middle, an odd remainder on the left, 12 pixels between the tops of the lines.
	stated := func(box *image.RGBA, l [3]string) {
		for i, line := range l {
			if line == "" {
				continue
			}
			w, _ := font.Measure(line)
			x, y := (seam+bodyW-w+1)/2, 54+12*i
			font.DrawFlat(box, line, x+1, y+1, shadow)
			font.Draw(box, line, x, y, ink)
		}
	}
	bareBoxes := func() (pane, card *image.RGBA) {
		pane = image.NewRGBA(image.Rect(0, 0, seam+bodyW, bodyH))
		ui.DrawTownCharacterRegion(pane, ui.TownCharacterView{
			Session: ui.CharacterPaneMission, ScreenH: ui.MissionFrameH, ModeFlag: true,
			PaneRect: local, PackOpen: true, BookOpen: true, CornerArt: f.characterPaneCorners(),
			FigurePane: panes.Figure, StatsPane: panes.Stats, Font: chrome, CardFont: font,
		})
		card = image.NewRGBA(pane.Bounds())
		ui.DrawCharacterPaneBody(card, ui.TownCharacterView{
			Statistics: true, PaneRect: local,
			FigurePane: panes.Figure, StatsPane: panes.Stats, Font: chrome, CardFont: font,
		})
		return pane, card
	}
	wantBoxes := func(l [3]string) (pane, card *image.RGBA) {
		pane, card = bareBoxes()
		stated(pane, l)
		return pane, card
	}
	// drawnLines is the bounding rectangle of each run of consecutive rows on
	// which got holds a pixel that bare does not, top to bottom: the rectangle
	// each line of text was drawn in.
	drawnLines := func(got, bare *image.RGBA) []image.Rectangle {
		var out []image.Rectangle
		b := got.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			minX, maxX := b.Max.X, b.Min.X-1
			for x := b.Min.X; x < b.Max.X; x++ {
				if got.RGBAAt(x, y) != bare.RGBAAt(x, y) {
					minX, maxX = min(minX, x), max(maxX, x)
				}
			}
			if maxX < minX {
				continue
			}
			row := image.Rect(minX, y, maxX+1, y+1)
			if n := len(out); n > 0 && out[n-1].Max.Y == y {
				out[n-1] = out[n-1].Union(row)
			} else {
				out = append(out, row)
			}
		}
		return out
	}
	// pinned reads the drawn rectangle of each line off the pane and, on EN,
	// holds it to the owner's capture: the ink rectangles, shadow excluded, in
	// pane-local pixels, of "NO UNITS" and "SELECTED" with none selected and of
	// "UNITS", "SELECTED:" and "5" with five. A rectangle read here is the ink
	// plus the one-pixel shadow right and below.
	pinned := func(what string, got, bare *image.RGBA, count int) {
		t.Helper()
		rects := drawnLines(got, bare)
		l := lines(count)
		n := 0
		for _, line := range l {
			if line != "" {
				n++
			}
		}
		if len(rects) != n {
			t.Fatalf("%s: %d drawn lines %v, want %d", what, len(rects), rects, n)
		}
		for i, r := range rects {
			t.Logf("%s: line %d %q drawn in %v", what, i+1, linesOf(plain, count)[i], r)
		}
		if ru {
			return
		}
		capture := map[int][]image.Rectangle{
			0: {image.Rect(66, 55, 110, 63), image.Rect(64, 67, 111, 75)},
			5: {image.Rect(75, 55, 100, 63), image.Rect(63, 67, 112, 75), image.Rect(85, 79, 90, 87)},
		}
		for i, ink := range capture[count] {
			if want := image.Rect(ink.Min.X, ink.Min.Y, ink.Max.X+1, ink.Max.Y+1); rects[i] != want {
				t.Errorf("%s: line %d %q drawn in %v, the capture has %v", what, i+1, linesOf(plain, count)[i], rects[i], want)
			}
		}
	}
	// heldInk counts the pixels the two lines paint at full level when drawn
	// alone at their centred placement, and how many of them box does not hold.
	heldInk := func(box *image.RGBA, l [3]string) (painted, missing int) {
		probe := image.NewRGBA(box.Bounds())
		stated(probe, l)
		b := probe.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if probe.RGBAAt(x, y) != ink {
					continue
				}
				painted++
				if box.RGBAAt(x, y) != ink {
					missing++
				}
			}
		}
		return painted, missing
	}
	same := func(what string, got, want *image.RGBA) {
		t.Helper()
		if got.Bounds() != want.Bounds() {
			t.Fatalf("%s: composed %v, want %v", what, got.Bounds(), want.Bounds())
		}
		if bytes.Equal(got.Pix, want.Pix) {
			return
		}
		for y := 0; y < got.Bounds().Dy(); y++ {
			for x := 0; x < got.Bounds().Dx(); x++ {
				if g, w := got.RGBAAt(x, y), want.RGBAAt(x, y); g != w {
					t.Fatalf("%s: (%d,%d) is %v, want %v", what, x, y, g, w)
				}
			}
		}
	}
	// look reads both boxes the way a frame does and checks them against the
	// state's own lines, and returns copies for a later comparison.
	look := func(what string, count int) (pane, card []byte) {
		t.Helper()
		upper, _, err := app.HeadlessCharacterPane()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		lower, err := app.HeadlessMissionCard()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if got := len(app.HeadlessSelection()); got != count {
			t.Fatalf("%s: the selection holds %d characters, want %d", what, got, count)
		}
		if count == 1 {
			for _, box := range []struct {
				name string
				pic  *image.RGBA
			}{{"pane", upper}, {"card", lower}} {
				for _, stale := range [][3]string{lines(0), lines(5)} {
					if painted, missing := heldInk(box.pic, stale); painted == 0 || missing == 0 {
						t.Fatalf("%s: the %s states %q with one character selected (%d of %d pixels)",
							what, box.name, stale, painted-missing, painted)
					}
				}
			}
			t.Logf("%s: the unit itself, with neither the zero lines nor the count lines in either box", what)
		} else {
			wantPane, wantCard := wantBoxes(lines(count))
			same(what+": upper pane", upper, wantPane)
			same(what+": lower card", lower, wantCard)
			barePane, bareCard := bareBoxes()
			same(what+": lower card is the bare page", lower, bareCard)
			pinned(what+": upper pane", upper, barePane, count)
			paneInk, _ := heldInk(upper, lines(count))
			if paneInk == 0 {
				t.Fatalf("%s: the lines %q paint %d pixels in the upper pane, so the comparison cannot tell them from none",
					what, linesOf(plain, count), paneInk)
			}
			t.Logf("%s: %q, %d ink pixels in the upper pane and none in the lower card",
				what, linesOf(plain, count), paneInk)
		}
		return append([]byte(nil), upper.Pix...), append([]byte(nil), lower.Pix...)
	}

	// Nothing selected: a right press on empty ground with nothing armed
	// clears any selection the mission opened with.
	pointer(gx, gy, "right-press", "right-release")
	away()
	firstPane, firstCard := look("nothing selected", 0)

	if err := app.HeadlessSelectEntity(uint32(ids[0])); err != nil {
		t.Fatal(err)
	}
	away()
	onePane, oneCard := look("one selected", 1)
	if bytes.Equal(onePane, firstPane) || bytes.Equal(oneCard, firstCard) {
		t.Fatal("one selected leaves a box as it was with none selected")
	}

	for k := 1; k < len(ids); k++ {
		x, y, err := app.HeadlessEntityPoint(uint32(ids[k]))
		if err != nil {
			t.Fatal(err)
		}
		pointer(x, y, "shift-press", "shift-release")
		away()
		look(fmt.Sprintf("%d selected", k+1), k+1)
	}

	pointer(gx, gy, "right-press", "right-release")
	away()
	againPane, againCard := look("nothing selected again", 0)
	if !bytes.Equal(againPane, firstPane) || !bytes.Equal(againCard, firstCard) {
		t.Fatal("emptying the selection again left a box unlike the first one with none selected")
	}
}
