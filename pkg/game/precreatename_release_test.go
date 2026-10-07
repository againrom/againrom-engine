package game

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// preCreateHeroNames are the pictures' own names in the page's picture order,
// male fighter, male mage, female fighter, female mage, as TEXT-074 gives
// them for each root: npcnames.txt entries 20, 22, 21 and 23.
var preCreateHeroNames = map[bool][4]string{
	false: {"Danath", "Fergard", "Naira", "Reniesta"},
	true:  {"Данас", "Фергард", "Найра", "Рениеста"},
}

// preCreateCP866 is s in code page 866, the RU root's text bytes.
func preCreateCP866(t *testing.T, s string) string {
	t.Helper()
	b, err := charmap.CodePage866.NewEncoder().String(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// preCreateNames is the running install's four picture names as the field
// holds them, and whether the install is the RU one.
func preCreateNames(t *testing.T, f *FrontEnd) (names [4]string, ru bool) {
	t.Helper()
	ru = f.textSelector() == text.SelectorConverting
	names = preCreateHeroNames[ru]
	if !ru {
		return names, false
	}
	for i, s := range names {
		names[i] = preCreateCP866(t, s)
	}
	if names[0] != "\x84\xa0\xad\xa0\xe1" {
		t.Fatalf("code page 866 gives % x for the first name, TEXT-073 gives 84 a0 ad a0 e1", names[0])
	}
	return names, true
}

// preCreateNameOpen presses the main menu's NEW GAME, armed as cmd/againrom's
// default door arms it, and returns the generator the press opened.
func preCreateNameOpen(t *testing.T, f *FrontEnd, app *ui.App) *ui.Chargen {
	t.Helper()
	var c *ui.Chargen
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		c = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: c, Begin: func(res ui.ChargenResult) (ui.MapOpener, error) {
			return f.NewGameOpener(10, res), nil
		}}
	})
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenChargen || c == nil {
		t.Fatalf("NEW GAME: screen %s, error %v", app.Screen(), err)
	}
	return c
}

// A new campaign's name field opens at npcnames.txt entry 20 with the first
// picture lit (TEXT-073): EN Danath, RU Данас.
func TestReleasePreCreateOpensAtTheFirstPicturesName(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	app := f.App("pre-create name")
	app.Layout(640, 480)
	names, _ := preCreateNames(t, f)
	c := preCreateNameOpen(t, f, app)
	state, ok := app.HeadlessChargenState()
	if !ok || state.Stage != ui.ChargenStagePreCreate || state.Name != names[0] || c.PreChoice() != 0 {
		t.Fatalf("NEW GAME opened the %s page at %q (% x), picture %d; want % x, picture 0",
			state.Stage, state.Name, state.Name, c.PreChoice(), names[0])
	}
}

// preCreateInk counts the pixels of frame inside r that hold exactly ink.
func preCreateInk(frame *image.RGBA, r image.Rectangle, ink color.RGBA) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if frame.RGBAAt(x, y) == ink {
				n++
			}
		}
	}
	return n
}

// preCreateFullLevel draws s at origin in ink on a blank canvas, where a
// full-level glyph pixel is the ink itself, and reports how many such pixels
// the draw has and how many of them frame does not hold in ink.
func preCreateFullLevel(font *text.Font, frame *image.RGBA, s string, origin image.Point, ink color.RGBA) (n, off int) {
	probe := image.NewRGBA(frame.Bounds())
	font.Draw(probe, s, origin.X, origin.Y, ink)
	b := probe.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if probe.RGBAAt(x, y) != ink {
				continue
			}
			n++
			if frame.RGBAAt(x, y) != ink {
				off++
			}
		}
	}
	return n, off
}

// The pre-create name field driven through ui.App on the installed art: the
// prompt's and the name's left-aligned origins and inks (TEXT-077), the
// caret's flips in 20 ms ticks (TEXT-076), picture presses over a default and
// over an edit (TEXT-074), the ten-byte append and key 8 (TEXT-075), Back
// from the detailed page (TEXT-073), and the created hero's name in the
// information window, in an F2 SAV and after a cold LOAD of that SAV.
func TestReleasePreCreateNameThroughAppInput(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("pre-create name")
	app.Layout(640, 480)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	names, ru := preCreateNames(t, f)
	c := preCreateNameOpen(t, f, app)
	pre := f.ChargenSetup().PreCreate
	font := pre.Art.NameFont
	if font == nil {
		t.Fatal("the install gave the page no name font")
	}

	state := func() ui.HeadlessChargen {
		t.Helper()
		s, ok := app.HeadlessChargenState()
		if !ok {
			t.Fatalf("the generator is not showing: %s", app.Screen())
		}
		return s
	}
	expect := func(name string, picture int, why string) {
		t.Helper()
		if s := state(); s.Name != name || c.PreChoice() != picture {
			t.Fatalf("%s: field % x, picture %d; want % x, picture %d", why, s.Name, c.PreChoice(), name, picture)
		}
	}
	tick := func(n int) {
		t.Helper()
		for range n {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
	}
	typed := func(s string, backspace bool) {
		t.Helper()
		if err := app.HeadlessType(s, backspace); err != nil {
			t.Fatal(err)
		}
	}
	frame := func() *image.RGBA {
		t.Helper()
		pix, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		if pix.Bounds() != image.Rect(0, 0, 640, 480) {
			t.Fatalf("frame %v, want 640x480", pix.Bounds())
		}
		return pix
	}
	press := func(kind string) {
		t.Helper()
		s := state()
		if err := headlessChargenPress(app, &s, kind, ""); err != nil {
			t.Fatal(err)
		}
	}

	// A pixel the page's own hit test gives each picture, the owned pixel
	// nearest the centroid of all it owns; a pixel no control owns; and the
	// name field's centre.
	var owned [4][]image.Point
	var sums [4]image.Point
	idle := image.Pt(-1, -1)
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			p := image.Pt(x, y)
			owner, ok := ui.PreCreateControlAt(c, p)
			if !ok {
				if idle.X < 0 {
					idle = p
				}
				continue
			}
			var i int
			if _, err := fmt.Sscanf(owner, "choice %d", &i); err == nil && i >= 0 && i < 4 {
				owned[i] = append(owned[i], p)
				sums[i] = sums[i].Add(p)
			}
		}
	}
	var pictures [4]image.Point
	for i, pixels := range owned {
		if len(pixels) == 0 {
			t.Fatalf("no pixel answers for picture %d", i)
		}
		centre := sums[i].Div(len(pixels))
		pictures[i] = slices.MinFunc(pixels, func(a, b image.Point) int {
			da, db := a.Sub(centre), b.Sub(centre)
			return (da.X*da.X + da.Y*da.Y) - (db.X*db.X + db.Y*db.Y)
		})
	}
	fieldAt := image.Pt((224+362)/2, (310+337)/2)
	if owner, ok := ui.PreCreateControlAt(c, fieldAt); !ok || owner != "name" || idle.X < 0 {
		t.Fatalf("the field's centre belongs to %q; idle pixel %v", owner, idle)
	}
	rest := func() {
		t.Helper()
		if err := app.HeadlessPointer("hover", idle.X, idle.Y); err != nil {
			t.Fatal(err)
		}
	}
	click := func(p image.Point) {
		t.Helper()
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
		rest()
	}
	// A picture press, then more than the 500 ms in which a second release on
	// the same picture would go forward.
	picture := func(i int) {
		t.Helper()
		click(pictures[i])
		tick(30)
	}

	// TEXT-073: NEW GAME opens at entry 20 with the first picture lit. The tip
	// panel covers the field, so its Close is pressed first.
	expect(names[0], 0, "NEW GAME")
	if tip := c.TipPanel(); !tip.Rect.Empty() {
		r := ui.TipPanelCloseRect(tip.Rect)
		click(r.Min.Add(r.Size().Div(2)))
		if !c.TipPanel().Rect.Empty() {
			t.Fatal("the tip panel's Close left it open")
		}
	}
	rest()

	// TEXT-077: the prompt and the name are left-aligned font4 draws at
	// (224,305) and (224,321) in RGB(65,47,20) and RGB(101,39,61); the prompt
	// advances 127 pixels on EN and 132 on RU. TEXT-076 gives the seeds'
	// advances, where the caret starts.
	promptInk, nameInk := color.RGBA{65, 47, 20, 255}, color.RGBA{101, 39, 61, 255}
	wantPrompt, promptAdvance, advances := "Character name:", 127, [4]int{54, 59, 42, 63}
	if ru {
		wantPrompt, promptAdvance, advances = preCreateCP866(t, "Имя персонажа:"), 132, [4]int{47, 66, 50, 72}
	}
	if pre.Prompt != wantPrompt || font.Advance(pre.Prompt) != promptAdvance {
		t.Fatalf("prompt % x advancing %d, want % x advancing %d", pre.Prompt, font.Advance(pre.Prompt), wantPrompt, promptAdvance)
	}
	for i, s := range names {
		if font.Advance(s) != advances[i] {
			t.Fatalf("name % x advances %d, TEXT-076 gives %d", s, font.Advance(s), advances[i])
		}
	}
	opening := frame()
	for _, d := range []struct {
		s    string
		at   image.Point
		ink  color.RGBA
		what string
	}{{pre.Prompt, image.Pt(224, 305), promptInk, "prompt"}, {names[0], image.Pt(224, 321), nameInk, "name"}} {
		n, off := preCreateFullLevel(font, opening, d.s, d.at, d.ink)
		if n == 0 || off != 0 {
			t.Fatalf("%s at %v: %d of its %d full-level pixels are not its ink", d.what, d.at, off, n)
		}
		centred := image.Pt(224+(362-224-font.Advance(d.s))/2, d.at.Y)
		if _, off := preCreateFullLevel(font, opening, d.s, centred, d.ink); off == 0 {
			t.Fatalf("%s: a draw centred at %v matches too, so the check cannot tell", d.what, centred)
		}
		t.Logf("%s: %d full-level pixels in its ink from %v", d.what, n, d.at)
	}

	// TEXT-076: the caret glyph inks 13 full-level pixels in columns 0..2 of
	// rows 0..14 from the name's advance, and flips in the first frame more
	// than 500 ms after the last flip: every 26 ticks of 20 ms.
	caret := image.Rect(224+advances[0], 321, 224+advances[0]+3, 336)
	var counts []int
	for range 60 {
		tick(1)
		counts = append(counts, preCreateInk(frame(), caret, nameInk))
	}
	hidden, shown := slices.Min(counts), slices.Max(counts)
	if shown-hidden != 13 {
		t.Fatalf("caret pixels per tick %v, want two values 13 apart", counts)
	}
	var flips []int
	for i := 1; i < len(counts); i++ {
		if counts[i] != counts[i-1] {
			flips = append(flips, i)
		}
	}
	if len(flips) < 2 {
		t.Fatalf("caret pixels per tick %v flip %d times in 60 ticks", counts, len(flips))
	}
	for i := 1; i < len(flips); i++ {
		if flips[i]-flips[i-1] != 26 {
			t.Fatalf("caret flips at ticks %v, want 26 ticks apart", flips)
		}
	}
	caretShown := func() bool {
		t.Helper()
		switch preCreateInk(frame(), caret, nameInk) {
		case shown:
			return true
		case hidden:
			return false
		}
		t.Fatal("the caret is neither shown nor hidden")
		return false
	}

	// A character under the cap shows the caret and restarts its phase; key
	// 8 does not. Ten ticks into a hidden phase the caret would show again 16
	// ticks later; restarted, it shows at once and hides 27 ticks later.
	click(fieldAt)
	for _, want := range []bool{true, false} {
		for n := 0; caretShown() != want; n++ {
			if n == 60 {
				t.Fatalf("the caret did not turn %v in 60 ticks", want)
			}
			tick(1)
		}
	}
	tick(10)
	typed("x", false)
	expect(names[0]+"x", 0, "a character appends")
	typed("", true)
	expect(names[0], 0, "key 8 removes the last byte")
	for i := 1; i <= 26; i++ {
		if !caretShown() {
			t.Fatalf("the caret hid %d ticks after a character, want 27", i)
		}
		tick(1)
	}
	if caretShown() {
		t.Fatal("the caret still shows 27 ticks after a character")
	}

	// TEXT-074: a press writes the picture's name over a default when the
	// picture is not the last pressed one; a typed name survives.
	for _, i := range []int{2, 1, 3, 0} {
		picture(i)
		expect(names[i], i, "a press over a default")
	}
	click(fieldAt)
	typed("x", false)
	picture(3)
	expect(names[0]+"x", 3, "a typed name survives a press")
	click(fieldAt)
	typed("", true)
	expect(names[0], 3, "key 8 leaves a default")
	picture(3)
	expect(names[0], 3, "a press on the last pressed picture writes nothing")
	picture(1)
	expect(names[1], 1, "another picture writes over the default")

	// TEXT-075: typing stops at ten bytes and key 8 removes the last byte.
	// TEXT-076: a character at the cap neither shows the caret nor restarts it.
	click(fieldAt)
	room := 10 - len(names[1])
	typed("abcdefghij"[:room], false)
	capped := names[1] + "abcdefghij"[:room]
	expect(capped, 1, "typing to the cap")
	caret = image.Rect(224+font.Advance(capped), 321, 224+font.Advance(capped)+3, 336)
	shown = preCreateInk(frame(), caret, nameInk)
	hidden = shown - 13
	for i := 1; i <= 26; i++ {
		tick(1)
		if !caretShown() {
			t.Fatalf("the caret hid %d ticks after the last byte under the cap", i)
		}
	}
	tick(1)
	if caretShown() {
		t.Fatal("the caret still shows 27 ticks after the last byte under the cap")
	}
	tick(2)
	typed("k", false)
	expect(capped, 1, "a character at the cap")
	for i := 30; i < 53; i++ {
		if caretShown() {
			t.Fatalf("a character at the cap showed the caret %d ticks after the last byte under it", i)
		}
		tick(1)
	}
	if !caretShown() {
		t.Fatal("the caret did not show 53 ticks after the last byte under the cap")
	}
	for range room {
		typed("", true)
	}
	expect(names[1], 1, "key 8 back to the default")
	picture(3)
	expect(names[3], 3, "a press over the restored default")

	// TEXT-073: Back from the detailed page enters the page again: the first
	// picture is lit, a typed name stays, a default becomes entry 20.
	click(fieldAt)
	typed("s", false)
	press(ui.ChargenControlForward)
	if s := state(); s.Stage != ui.ChargenStageDetailed || c.CaretVisible() {
		t.Fatalf("Forward: the %s page, caret %v", s.Stage, c.CaretVisible())
	}
	press(ui.ChargenControlBack)
	expect(names[3]+"s", 0, "Back keeps a typed name")
	click(fieldAt)
	typed("", true)
	press(ui.ChargenControlForward)
	press(ui.ChargenControlBack)
	expect(names[0], 0, "Back turns a default into entry 20")
	picture(3)
	expect(names[3], 3, "the female mage's own name")

	// The created hero carries that name into the information window, an F2
	// SAV and a cold LOAD of that SAV.
	press(ui.ChargenControlForward)
	press(ui.ChargenControlPlay)
	if app.Screen() != ui.ScreenMap || f.liveMission != 10 || len(f.liveParty) == 0 || f.liveParty[0].Name != names[3] {
		t.Fatalf("Play: screen %s, mission %d, party %d", app.Screen(), f.liveMission, len(f.liveParty))
	}
	window := func(front *FrontEnd, a *ui.App, what string) {
		t.Helper()
		id := uint32(front.live.mission.ids[0])
		if s, ok := front.live.view.InspectionPanel(); !ok || s.ID != id {
			if err := a.HeadlessSelectEntity(id); err != nil {
				t.Fatalf("%s: %v", what, err)
			}
		}
		s, ok := front.live.view.InspectionPanel()
		if !ok || s.Kind != ui.InspectionUnit || s.ID != id || s.Name != names[3] {
			t.Fatalf("%s: the information window shows unit %d named % x, want the hero %d named % x", what, s.ID, s.Name, id, names[3])
		}
	}
	window(f, app, "Play")
	if _, _, up := f.LiveNotice(); up {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	missionAutosaveRow(t, f, store, f.liveMission)
	raw := cityRosterF2Save(t, app, store, "precreate")
	if got := fallenHeroWinSavedHero(t, raw).Name; got != names[3] {
		t.Fatalf("the F2 SAV names the hero % x, want % x", got, names[3])
	}
	cold, coldApp := fallenHeroWinColdLoad(t, store, "precreate.sav")
	if cold.liveMission != 10 || len(cold.liveParty) == 0 || cold.liveParty[0].Name != names[3] {
		t.Fatalf("cold LOAD: mission %d, party %d", cold.liveMission, len(cold.liveParty))
	}
	window(cold, coldApp, "cold LOAD")
}
