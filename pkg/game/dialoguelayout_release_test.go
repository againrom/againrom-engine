package game

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// The dialogue panel as `DLG-PANEL-035`, `DLG-PORTRAIT-036`, `DLG-RECT-037`,
// `DLG-LINE-038` and `DLG-BUTTON-039` state it at 640x480, checked on the frame
// the App presents for a map event, an inn NPC and the shop keeper, on the
// installed frame art and font. Every rectangle and colour is written here from
// the claims and never read from the layout under test.

var (
	dlgBody   = image.Rect(76, 124, 556, 348)  // frame body, 480x224
	dlgBand   = image.Rect(76, 124, 564, 356)  // body and the 8 px shadow band
	dlgPane   = image.Rect(106, 178, 194, 286) // portrait surface, 88x108
	dlgText   = image.Rect(204, 160, 506, 297) // text control 300x135 and a pixel of shadow
	dlgButton = image.Rect(276, 296, 356, 322)

	dlgWhite  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	dlgShadow = color.RGBA{R: 8, G: 8, B: 8, A: 255}
	dlgGold   = color.RGBA{R: 185, G: 159, B: 73, A: 255}
	dlgLight  = color.RGBA{R: 41, G: 68, B: 57, A: 255}
	dlgDark   = color.RGBA{G: 12, B: 8, A: 255}
)

const (
	dlgTextLeft, dlgTextTop, dlgTextWidth = 204, 160, 300
	dlgPitch, dlgIndent                   = 17, 10
)

const dlgShopPath = "main/text/shop/npc31m31.txt"

// Shipped dialogue blocks per line count, from the claim's census over both
// roots' five text families: 688 English and 732 Russian blocks.
var (
	dlgLinesEN = [7]int{48, 154, 172, 104, 76, 64, 70}
	dlgLinesRU = [7]int{74, 171, 145, 115, 87, 76, 64}
)

// The shop keeper's dialogue, then the first town's tavern NPC, through ordinary
// town input; then the first dialogue a shipped mission raises by itself. Each is
// compared with the frame the App presents.
func TestReleaseDialogueLayout(t *testing.T) {
	f := releaseFront(t)
	app, s := openFirstTownShopDialogue(t, f, "dialogue layout")
	shop := func() (*image.RGBA, string) {
		t.Helper()
		if path, _ := s.townTextPath(); s.room != roomTalk || path != dlgShopPath {
			t.Fatalf("room %d text %q, want the dialogue %s", s.room, path, dlgShopPath)
		}
		return dialogueFrame(t, app), townPartText(t, f, s)
	}
	shopFrame, shopText := shop()
	shopRoom := closeTownDialogue(t, app, s)
	t.Run("shop", func(t *testing.T) {
		checkDialogueFrame(t, "shop", f, shopFrame, dialogueBackdropRoom(shopRoom), shopText)
	})

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	offers := f.Town.Offers(TownTavern)
	if len(offers) == 0 {
		t.Fatal("the first town's tavern offers no NPC")
	}
	for _, target := range []string{"TAVERN", fmt.Sprintf("NPC %d", offers[0].NPC)} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	want := fmt.Sprintf("main/text/inn/npc/npc%02dm%02d.txt", offers[0].NPC, offers[0].Mission)
	if path, _ := s.townTextPath(); s.room != roomTalk || path != want {
		t.Fatalf("room %d text %q, want the dialogue %s", s.room, path, want)
	}
	tavernFrame, tavernText := dialogueFrame(t, app), townPartText(t, f, s)
	tavernRoom := closeTownDialogue(t, app, s)
	t.Run("tavern", func(t *testing.T) {
		checkDialogueFrame(t, "tavern", f, tavernFrame, dialogueBackdropRoom(tavernRoom), tavernText)
	})

	t.Run("map event", func(t *testing.T) {
		mf := releaseFront(t)
		mf.SoundPlayer, mf.MusicPlayer, mf.AmbientPlayer, mf.CutsceneAudioPlayer = nil, nil, nil, nil
		mf.SetDeterministicFrames(true)
		mapp := mf.App("dialogue layout map event")
		mapp.Layout(640, 480)
		if err := mapp.OpenMission(mf.MissionOpenerWith(10, mf.NextParty())); err != nil {
			t.Fatal(err)
		}
		for n := 0; ; n++ {
			if _, kind, open := mf.LiveNotice(); open && kind == ui.NoticeDialogue {
				break
			}
			if n == 1500 {
				t.Fatal("mission 10 raised no dialogue notice within 1500 ticks")
			}
			if err := mapp.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		// The map screen draws straight onto the display, so no frame of it can
		// be read back. The notice the viewer holds is composed here with the
		// layout, font, text and speaker it draws with and laid over a flat
		// ground at the panel's origin, which is where it draws it.
		body, _, _ := mf.LiveNotice()
		portrait, face := mf.live.view.NoticeSpeaker()
		if !portrait {
			t.Fatal("the map dialogue has no portrait pane: every shipped dialogue names a speaker")
		}
		layout := mf.Words.OnLayout(ui.AuthoredDialogueLayout())
		layout.Frame = mf.gameMenuArt()
		room := image.NewRGBA(image.Rect(0, 0, 640, 480))
		draw.Draw(room, room.Bounds(), &image.Uniform{C: color.RGBA{R: 120, G: 110, B: 100, A: 255}}, image.Point{}, draw.Src)
		frame := image.NewRGBA(room.Bounds())
		copy(frame.Pix, room.Pix)
		ui.ComposeDialogueNotice(frame, layout.WithPortrait(true), mf.Font.Value(), body, face, layout.Box.Min)
		if got, want := layout.Box.Min, image.Pt(76, 124); got != want {
			t.Fatalf("the panel's origin is %v, want %v", got, want)
		}
		checkDialogueFrame(t, "map event", mf, frame, room, body)
	})
}

// Every shipped dialogue block wraps to the number of lines the claim's census
// gives, and none to more than seven.
func TestReleaseDialogueLinesPerBlock(t *testing.T) {
	f := releaseFront(t)
	font := f.Font.Value()
	layout := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(true)
	want, blocks := dlgLinesEN, 688
	if font.Selector == text.SelectorConverting {
		want, blocks = dlgLinesRU, 732
	}
	var got [8]int
	total := 0
	for _, e := range f.Archives.Containers.Entries() {
		addr := strings.ToLower(e.Address)
		if !strings.HasPrefix(addr, "main/text/") || !dialogueFamily(addr) {
			continue
		}
		payload, err := f.Archives.Containers.ReadFile(e.Address)
		if err != nil {
			t.Fatalf("%s: %v", e.Address, err)
		}
		for _, candidate := range dialogueProjectionCandidates(t, payload) {
			body := candidate.body
			total++
			n := len(ui.NoticeLayoutOf(layout, font, body))
			if n < 1 || n > 7 {
				t.Errorf("%s: a block wraps to %d lines", e.Address, n)
				continue
			}
			got[n]++
		}
	}
	if total != blocks {
		t.Errorf("%d blocks, want %d", total, blocks)
	}
	if !slices.Equal(got[1:], want[:]) {
		t.Errorf("blocks per line count 1..7 = %v, want %v", got[1:], want)
	}
}

// dialogueFamily is whether an entry address is one of the five families whose
// blocks the claim counts: mission events, inn NPCs, mercenaries, the shop and the
// training hall's one shipped speaker.
func dialogueFamily(addr string) bool {
	for _, family := range []string{"text/battle/", "text/inn/npc/", "text/inn/mercenary/", "text/shop/", "text/training/npc34m"} {
		if strings.Contains(addr, family) {
			return true
		}
	}
	return false
}

// dialogueBlocks are the bodies of the tags of payload that carry a part number:
// a body runs from the tag's closing bracket to the next tag.
func dialogueBlocks(payload []byte) []string {
	var out []string
	s := string(payload)
	for i := 0; i < len(s); {
		open := strings.IndexByte(s[i:], '<')
		if open < 0 {
			break
		}
		open += i
		end := strings.IndexByte(s[open:], '>')
		if end < 0 {
			break
		}
		end += open
		body := s[end+1:]
		if next := strings.IndexByte(body, '<'); next >= 0 {
			body = body[:next]
		}
		if containsFold(s[open+1:end], partMark) {
			out = append(out, body)
		}
		i = end + 1
	}
	return out
}

type dlgCandidate struct{ tail, body string }

// Projection selects every part-bearing candidate, with acceptance supplied.
// LF/CR/NUL extraction and six-byte trim premises are DIALOGUE-062/069/070.
func dialogueProjectionCandidates(t *testing.T, payload []byte) []dlgCandidate {
	t.Helper()
	source := string(payload)
	if nul := strings.IndexByte(source, 0); nul >= 0 {
		source = source[:nul]
	}
	var candidates []dlgCandidate
	for cursor := 0; cursor < len(source); {
		open := strings.IndexByte(source[cursor:], '<')
		if open < 0 {
			break
		}
		open += cursor
		close := strings.IndexByte(source[open:], '>')
		if close < 0 {
			t.Fatal("projection unterminated header")
		}
		close += open
		cursor = close + 1
		if !containsFold(source[open+1:close], "part=") {
			continue
		}
		tail := source[cursor:]
		lf := strings.IndexByte(tail, '\n')
		if lf < 0 {
			t.Fatal("projection candidate missing header LF")
		}
		body := tail[lf+1:]
		if next := strings.IndexByte(body, '<'); next >= 0 {
			cr := strings.LastIndexByte(body[:next], '\r')
			if cr < 0 {
				t.Fatal("projection candidate missing bounded CR")
			}
			body = body[:cr]
		}
		candidates = append(candidates, dlgCandidate{tail: tail, body: body})
	}
	return candidates
}

func expectedInstalledDialoguePart(t *testing.T, payload []byte, part int, audience EventAudience) (string, bool) {
	t.Helper()
	if nul := strings.IndexByte(string(payload), 0); nul >= 0 {
		payload = payload[:nul]
	}
	_, tail, ok := eventPartTail(payload, part, audience)
	if !ok {
		return "", false
	}
	lf := strings.IndexByte(tail, '\n')
	if lf < 0 {
		t.Fatal("accepted installed header lacks LF")
	}
	expected := tail[lf+1:]
	if next := strings.IndexByte(expected, '<'); next >= 0 {
		cr := strings.LastIndexByte(expected[:next], '\r')
		if cr < 0 {
			t.Fatal("accepted installed tail lacks bounded CR")
		}
		expected = expected[:cr]
	}
	return expected, true
}

// townPartText binds the actual stored text to its accepted raw resource tail.
func townPartText(t *testing.T, f *FrontEnd, town *townScreen) string {
	t.Helper()
	_, payload, _ := town.townDialogueLayout()
	expected, ok := expectedInstalledDialoguePart(t, payload, town.said, HeroAudience(f.Carried))
	if !ok {
		t.Fatal("town selected no raw resource tail")
	}
	if town.dialogue.text != expected {
		t.Fatalf("town delivered %q want raw accepted interval %q", town.dialogue.text, expected)
	}
	return town.dialogue.text
}

// closeTownDialogue answers the open dialogue with its own control until the room
// shows, and returns the frame of that room: the same room the dialogue's panel
// was drawn over.
func closeTownDialogue(t *testing.T, app *ui.App, s *townScreen) *image.RGBA {
	t.Helper()
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room == roomTalk {
		t.Fatal("the dialogue did not close")
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	return dialogueFrame(t, app)
}

// dialogueFrame is a copy of the frame the App presents at 640x480.
func dialogueFrame(t *testing.T, app *ui.App) *image.RGBA {
	t.Helper()
	pix, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	if pix.Bounds().Size() != image.Pt(640, 480) {
		t.Fatalf("frame is %v, want 640x480", pix.Bounds())
	}
	out := image.NewRGBA(image.Rect(0, 0, 640, 480))
	draw.Draw(out, out.Bounds(), pix, pix.Bounds().Min, draw.Src)
	return out
}

// checkDialogueFrame runs every claim check on one dialogue frame. room is the
// same room without the dialogue when the room can be had.
func checkDialogueFrame(t *testing.T, kind string, f *FrontEnd, pix, room *image.RGBA, src string) {
	t.Helper()
	art := f.gameMenuArt()
	font := f.Font.Value()
	if art == nil || font == nil {
		t.Fatal("no installed frame art or font")
	}
	layout := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(true)
	checkDialoguePanel(t, kind, pix, art)
	if room != nil {
		checkDialogueShadow(t, kind, pix, room, art)
	}
	checkDialogueBorder(t, kind, pix, art)
	checkDialogueButton(t, kind, pix, font, layout.ButtonLabel)
	checkDialogueText(t, kind, pix, font, src, ui.NoticeLayoutOf(layout, font, src))
}

type dlgTile struct {
	piece int
	at    image.Point
}

// dialogueTiling is the frame's tiling of the 480x224 body: `DLG-PANEL-035`'s
// corners, its (480-96)/96 tiles of pieces 2 and 7 along the top and bottom, its
// (224-96)/64 tiles of pieces 4 and 5 down the sides, and the interior of piece 0
// from (48,48) in as many tiles.
func dialogueTiling() []dlgTile {
	l, t, r, b := dlgBody.Min.X, dlgBody.Min.Y, dlgBody.Max.X, dlgBody.Max.Y
	tiles := []dlgTile{{1, image.Pt(l, t)}, {3, image.Pt(r-48, t)}, {6, image.Pt(l, b-48)}, {8, image.Pt(r-48, b-48)}}
	across, down := (dlgBody.Dx()-96)/96, (dlgBody.Dy()-96)/64
	for i := 0; i < across; i++ {
		tiles = append(tiles, dlgTile{2, image.Pt(l+48+96*i, t)}, dlgTile{7, image.Pt(l+48+96*i, b-48)})
	}
	for j := 0; j < down; j++ {
		tiles = append(tiles, dlgTile{4, image.Pt(l, t+48+64*j)}, dlgTile{5, image.Pt(r-48, t+48+64*j)})
		for i := 0; i < across; i++ {
			tiles = append(tiles, dlgTile{0, image.Pt(l+48+96*i, t+48+64*j)})
		}
	}
	return tiles
}

// checkDialoguePanel compares every opaque pixel of the frame's nine pieces, where
// the tiling puts it, with the frame: apart from the portrait surface, the text
// control and the button, which are drawn over the frame.
func checkDialoguePanel(t *testing.T, kind string, pix *image.RGBA, art *ui.MenuPanelArt) {
	t.Helper()
	for i, want := range []image.Point{{96, 64}, {48, 48}, {96, 48}, {48, 48}, {48, 64}, {48, 64}, {48, 48}, {96, 48}, {48, 48}} {
		if got := art.Pieces[i].Bounds().Size(); got != want {
			t.Fatalf("frame piece %d measures %v, the claim's %v", i, got, want)
		}
	}
	checked, bad := panelDifferences(t, kind, pix, art, image.Point{}, true)
	t.Logf("%s: %d frame pixels checked, %d differ", kind, checked, bad)
	if checked < 40000 {
		t.Fatalf("%s: only %d frame pixels are checked", kind, checked)
	}
	if bad > 0 {
		t.Errorf("%s: %d of %d frame pixels differ from the tiling", kind, bad, checked)
	}
	// The comparison must be able to tell one pixel: the tiling moved by one
	// pixel is not what the frame shows.
	if _, moved := panelDifferences(t, kind, pix, art, image.Pt(1, 0), false); moved < 1000 {
		t.Errorf("%s: the tiling moved one pixel differs in only %d pixels: the comparison cannot tell", kind, moved)
	}
}

// panelDifferences counts the opaque pixels of the frame's pieces, laid by the
// tiling moved by shift, that the frame does not show, and how many it compared.
func panelDifferences(t *testing.T, kind string, pix *image.RGBA, art *ui.MenuPanelArt, shift image.Point, report bool) (checked, bad int) {
	t.Helper()
	for _, tl := range dialogueTiling() {
		piece := art.Pieces[tl.piece]
		b := piece.Bounds()
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				want := piece.RGBAAt(b.Min.X+x, b.Min.Y+y)
				p := tl.at.Add(image.Pt(x, y))
				if want.A != 0xff || p.In(dlgPane) || p.In(dlgText) || p.In(dlgButton) {
					continue
				}
				checked++
				if q := p.Add(shift); pix.RGBAAt(q.X, q.Y) != want {
					if report && bad < 5 {
						t.Errorf("%s: piece %d at %v: pixel %v is %v, want %v", kind, tl.piece, tl.at, q, pix.RGBAAt(q.X, q.Y), want)
					}
					bad++
				}
			}
		}
	}
	return checked, bad
}

func dialogueBackdropRoom(room *image.RGBA) *image.RGBA {
	out := image.NewRGBA(room.Bounds())
	for y := room.Bounds().Min.Y; y < room.Bounds().Max.Y; y++ {
		for x := room.Bounds().Min.X; x < room.Bounds().Max.X; x++ {
			out.SetRGBA(x, y, packedBackdropColour(room.RGBAAt(x, y)))
		}
	}
	return out
}

// checkDialogueShadow compares the frame with the same room without the panel.
// The 8 px band right of and below the body holds the second, 8 px moved
// stamp of pieces 3, 5, 6, 7 and 8: a written pixel of a stamp is darkened, every
// other pixel of the band, and everything within 16 px of the panel, is the
// room's. What the stamp puts there is not the claim's; that it darkens is what
// the original's screenshot shows.
func checkDialogueShadow(t *testing.T, kind string, pix, room *image.RGBA, art *ui.MenuPanelArt) {
	t.Helper()
	foot := map[image.Point]bool{}
	for _, tl := range dialogueTiling() {
		if !slices.Contains([]int{3, 5, 6, 7, 8}, tl.piece) {
			continue
		}
		piece := art.Pieces[tl.piece]
		b := piece.Bounds()
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				if piece.RGBAAt(b.Min.X+x, b.Min.Y+y).A != 0 {
					foot[tl.at.Add(image.Pt(x+8, y+8))] = true
				}
			}
		}
	}
	darker, bad := 0, 0
	report := func(format string, args ...any) {
		if bad < 5 {
			t.Errorf(kind+": "+format, args...)
		}
		bad++
	}
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			p := image.Pt(x, y)
			if p.In(dlgBody) || !p.In(dlgBand.Inset(-16)) {
				continue
			}
			got, want := pix.RGBAAt(x, y), room.RGBAAt(x, y)
			switch {
			case !foot[p]:
				if got != want {
					report("pixel %v is %v, the room's %v: nothing is drawn there", p, got, want)
				}
			case got.R > want.R || got.G > want.G || got.B > want.B:
				report("shadow pixel %v is %v, lighter than the room's %v", p, got, want)
			case got != want:
				darker++
			case int(want.R)+int(want.G)+int(want.B) >= 40:
				report("shadow pixel %v is the room's %v, not darkened", p, want)
			}
		}
	}
	t.Logf("%s: shadow footprint %d pixels, %d darkened, %d wrong", kind, len(foot), darker, bad)
	if darker < 1500 {
		t.Errorf("%s: only %d pixels of the band are darkened", kind, darker)
	}
}

// checkDialogueBorder compares the portrait border with the frame: its 1626
// written pixels, each drawn where its stored row counted from the bottom of the
// 88x108 surface puts it, except a black one, which the surface's keyed blit
// leaves out. The order of the rows is read off the original's screenshot and is
// not the claim's.
func checkDialogueBorder(t *testing.T, kind string, pix *image.RGBA, art *ui.MenuPanelArt) {
	t.Helper()
	if b := art.Portrait; b == nil || b.Bounds().Size() != image.Pt(88, 108) {
		t.Fatalf("%s: the installed portrait border is %v, want 88x108", kind, b)
	}
	written, matched, keyed, bad := borderDifferences(t, kind, pix, art, image.Point{}, true)
	t.Logf("%s: border writes %d pixels: %d shown, %d black and keyed out, %d wrong", kind, written, matched, keyed, bad)
	if written != 1626 {
		t.Errorf("%s: the border writes %d pixels, the claim's 1626", kind, written)
	}
	if bad > 0 || matched < 1500 {
		t.Errorf("%s: %d border pixels differ, %d shown", kind, bad, matched)
	}
	// The comparison must be able to tell the row order and one pixel: the
	// border moved one pixel right, or laid top-down, is not what the frame shows.
	if _, _, _, moved := borderDifferences(t, kind, pix, art, image.Pt(1, 0), false); moved < 500 {
		t.Errorf("%s: the border moved one pixel differs in only %d pixels: the comparison cannot tell", kind, moved)
	}
}

// borderDifferences lays the border art at the pane, moved by shift, and counts
// its written pixels, those the frame shows, those left out as black, and those
// it does not show.
func borderDifferences(t *testing.T, kind string, pix *image.RGBA, art *ui.MenuPanelArt, shift image.Point, report bool) (written, matched, keyed, bad int) {
	t.Helper()
	b := art.Portrait
	for row := 0; row < 108; row++ {
		for col := 0; col < 88; col++ {
			c := b.RGBAAt(b.Bounds().Min.X+col, b.Bounds().Min.Y+row)
			if c.A == 0 {
				continue
			}
			written++
			if c.A != 0xff {
				t.Fatalf("%s: border pixel (%d,%d) is partly transparent: %v", kind, col, row, c)
			}
			if c.R>>3 == 0 && c.G>>2 == 0 && c.B>>3 == 0 {
				keyed++
				continue
			}
			c = dialoguePackedPicture(c)
			p := dlgPane.Min.Add(image.Pt(col, 107-row)).Add(shift)
			if got := pix.RGBAAt(p.X, p.Y); got != c {
				if report && bad < 5 {
					t.Errorf("%s: border pixel (%d,%d) shows %v at %v, want %v", kind, col, row, got, p, c)
				}
				bad++
				continue
			}
			matched++
		}
	}
	return written, matched, keyed, bad
}

// checkDialogueButton compares the button with the frame: the light and dark
// bevel lines of `DLG-BUTTON-039` with L,T = (276,296) and R',B' = (355,321), and
// the label centred on (316,308) by its width, its cell's top eight rows above
// that, in gold with a flat shadow two pixels down and right.
func checkDialogueButton(t *testing.T, kind string, pix *image.RGBA, font *text.Font, label string) {
	t.Helper()
	l, top, r, b := 276, 296, 355, 321
	bad := 0
	set := func(c color.RGBA, x, y int) {
		if got := pix.RGBAAt(x, y); got != c {
			if bad < 5 {
				t.Errorf("%s: button pixel (%d,%d) is %v, want %v", kind, x, y, got, c)
			}
			bad++
		}
	}
	hline := func(c color.RGBA, y, x0, x1 int) {
		for x := x0; x <= x1; x++ {
			set(c, x, y)
		}
	}
	vline := func(c color.RGBA, x, y0, y1 int) {
		for y := y0; y <= y1; y++ {
			set(c, x, y)
		}
	}
	vline(dlgDark, r, top+2, b-2)
	vline(dlgDark, r-1, top+1, b-1)
	hline(dlgDark, b, l+2, r-2)
	hline(dlgDark, b-1, l+1, r-1)
	set(dlgDark, r-2, b-2)
	hline(dlgLight, top, l+2, r-2)
	vline(dlgLight, l, top+2, b-2)
	set(dlgLight, l+1, top+1)
	if label == "" {
		t.Fatalf("%s: the dialogue button has no label", kind)
	}
	x, y := 316-font.Advance(label)/2, 308-8
	if !runMatches(pix, dialogueRunPixels(font, label, x, y, dlgGold, 2)) {
		t.Errorf("%s: the label %q is not drawn in gold at (%d,%d)", kind, label, x, y)
	}
	if runMatches(pix, dialogueRunPixels(font, label, x+1, y, dlgGold, 2)) {
		t.Errorf("%s: the label also matches one pixel to the right: the comparison cannot tell", kind)
	}
}

// dialogueRunPixels is what drawing s at (x,y) paints, as the original draws every
// run of text: a flat shadow shade pixels right and down, then the ink over it.
func dialogueRunPixels(f *text.Font, s string, x, y int, ink color.RGBA, shade int) map[image.Point]color.RGBA {
	w, h := f.Measure(s)
	pad := shade + 2
	c := image.NewRGBA(image.Rect(0, 0, w+2*pad, h+2*pad))
	f.DrawFlat(c, s, pad+shade, pad+shade, dlgShadow)
	f.Draw(c, s, pad, pad, ink)
	out := map[image.Point]color.RGBA{}
	for cy := 0; cy < c.Bounds().Dy(); cy++ {
		for cx := 0; cx < c.Bounds().Dx(); cx++ {
			if p := c.RGBAAt(cx, cy); p.A == 0xff {
				if ink == dlgGold {
					p.R = uint8(int(p.R>>3) * 255 / 31)
					p.G = uint8(int(p.G>>2) * 255 / 63)
					p.B = uint8(int(p.B>>3) * 255 / 31)
				}
				out[image.Pt(x-pad+cx, y-pad+cy)] = p
			}
		}
	}
	return out
}

// runMatches is whether every pixel a run paints is that pixel of the frame.
func runMatches(pix *image.RGBA, run map[image.Point]color.RGBA) bool {
	if len(run) == 0 {
		return false
	}
	for p, c := range run {
		if !p.In(pix.Bounds()) || pix.RGBAAt(p.X, p.Y) != c {
			return false
		}
	}
	return true
}

type dlgLine struct {
	words     []string
	first     bool // a paragraph's first line: indented
	justified bool
}

// dialogueLines validates visible words against CRLF pieces and derives their
// marker placement. The visible control may clamp a final source suffix.
func dialogueLines(t *testing.T, kind, src string, lines []string) []dlgLine {
	t.Helper()
	var out []dlgLine
	li := 0
	first := true
	blanks := func(r rune) bool { return r == ' ' || r == '\t' }
	for _, piece := range strings.Split(src, "\r\n") {
		rest := strings.FieldsFunc(piece, blanks)
		if len(rest) == 0 {
			if li >= len(lines) {
				break
			}
			if lines[li] != "" {
				t.Fatalf("%s empty piece line%d=%q", kind, li, lines[li])
			}
			out = append(out, dlgLine{first: first})
			li++
			first = false
			continue
		}
		for len(rest) > 0 && li < len(lines) {
			words := strings.FieldsFunc(lines[li], blanks)
			if len(words) == 0 || len(words) > len(rest) || !slices.Equal(words, rest[:len(words)]) {
				t.Fatalf("%s line%d %q does not continue %q", kind, li, lines[li], rest)
			}
			rest = rest[len(words):]
			out = append(out, dlgLine{words: words, first: first, justified: len(rest) > 0 && len(words) > 1})
			first = len(rest) == 0
			li++
		}
		if li == len(lines) {
			break
		}
	}
	if li != len(lines) {
		t.Fatalf("%s %d of %d lines follow raw source", kind, li, len(lines))
	}
	return out
}

// checkDialogueText compares every line of the text with the frame as
// `DLG-LINE-038` places it: line tops 17 apart from 160, left 204 (214 on a
// paragraph's first line), the justify width 300 (290 there), each word of a
// justified line at the whole part of a running sum of the words' widths and the
// gap, a paragraph's last line whole from its left, ink white and shadow (8,8,8).
// The default supplied PC53 policy has a double store between words. Native
// precision is Unknown; screenshot-fitted retained precision is not an oracle.
func checkDialogueText(t *testing.T, kind string, pix *image.RGBA, font *text.Font, src string, lines []string) {
	t.Helper()
	if len(lines) < 1 || len(lines) > 7 {
		t.Fatalf("%s: %d lines, the control shows one to seven", kind, len(lines))
	}
	placed := dialogueLines(t, kind, src, lines)
	painted := map[image.Point]bool{}
	mark := func(run map[image.Point]color.RGBA) {
		for p := range run {
			painted[p] = true
		}
	}
	justified, words := 0, 0
	for i, pl := range placed {
		if len(pl.words) == 0 {
			continue
		}
		y := dlgTextTop + dlgPitch*i
		x0, w := dlgTextLeft, dlgTextWidth
		if pl.first {
			x0, w = x0+dlgIndent, w-dlgIndent
		}
		if !pl.justified {
			line := strings.Join(pl.words, " ")
			run := dialogueRunPixels(font, line, x0, y, dlgWhite, 1)
			if !runMatches(pix, run) {
				t.Errorf("%s: line %d %q is not drawn whole at (%d,%d)", kind, i, line, x0, y)
			}
			mark(run)
			words += len(pl.words)
			continue
		}
		sum := 0
		widths := make([]float64, len(pl.words))
		for k, word := range pl.words {
			m := font.Advance(word)
			widths[k], sum = float64(m), sum+m
		}
		gap := float64(w-sum) / float64(len(pl.words)-1)
		acc := float64(x0)
		for k, word := range pl.words {
			x := int(acc)
			run := dialogueRunPixels(font, word, x, y, dlgWhite, 1)
			if !runMatches(pix, run) {
				t.Errorf("%s: line %d word %d %q is not drawn at %d", kind, i, k, word, x)
			}
			mark(run)
			words++
			acc = (acc + widths[k]) + gap
		}
		justified++
	}
	// The comparison must be able to tell one pixel: the first word of the first
	// line moved either way is not what the frame shows.
	firstLine := 0
	for firstLine < len(placed) && len(placed[firstLine].words) == 0 {
		firstLine++
	}
	if firstLine == len(placed) {
		t.Fatal("raw source draws no words")
	}
	x := dlgTextLeft
	if placed[firstLine].first {
		x += dlgIndent
	}
	for _, dx := range []int{-1, 1} {
		if runMatches(pix, dialogueRunPixels(font, placed[firstLine].words[0], x+dx, dlgTextTop+dlgPitch*firstLine, dlgWhite, 1)) {
			t.Errorf("%s first reached word matches offset%d", kind, dx)
		}
	}
	extra := 0
	for y := dlgText.Min.Y; y < dlgText.Max.Y; y++ {
		for x := dlgText.Min.X; x < dlgText.Max.X; x++ {
			if p := image.Pt(x, y); pix.RGBAAt(x, y) == dlgWhite && !painted[p] {
				if extra < 5 {
					t.Errorf("%s: white ink at %v that no line paints", kind, p)
				}
				extra++
			}
		}
	}
	t.Logf("%s: %d lines, %d words, %d justified lines, %d stray white pixels", kind, len(placed), words, justified, extra)
}

func TestReleaseDialogueSpilledCorpus(t *testing.T) {
	f := releaseFront(t)
	font := f.Font.Value()
	layout := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(true)
	layout.Portrait = image.Rectangle{}
	layout.Button = image.Rectangle{}
	raw, err := f.Archives.Containers.ReadFile("main/text/battle/m10/event01.txt")
	if err != nil {
		t.Fatal(err)
	}
	rawBodies := dialogueBlocks(raw)
	if len(rawBodies) < 2 {
		t.Fatal("raw source witness lacks second candidate")
	}
	rawLines := ui.NoticeLayoutOf(layout, font, rawBodies[1])
	if len(rawLines) == 0 || rawLines[0] != "" {
		t.Fatal("raw leading CRLF lost its empty first line")
	}
	rawPlaced := dialogueLines(t, "raw-input seam", rawBodies[1], rawLines)
	rawSource, rawDrawn := 0, 0
	for _, piece := range strings.Split(rawBodies[1], "\r\n") {
		rawSource += len(strings.FieldsFunc(piece, func(r rune) bool { return r == ' ' || r == '\t' }))
	}
	for _, line := range rawPlaced {
		rawDrawn += len(line.words)
	}
	t.Logf("raw tag-body witness: %d of %d source words visible; leading empty piece and low-level clamp remain distinct from accepted delivery", rawDrawn, rawSource)
	wantBlocks, wantLines, wantJust, wantWords := 688, 2542, 1854, 11415
	if font.Selector == text.SelectorConverting {
		wantBlocks, wantLines, wantJust, wantWords = 732, 2650, 1906, 9550
	}
	blocks, lines, just, positions := 0, 0, 0, 0
	for _, entry := range f.Archives.Containers.Entries() {
		addr := strings.ToLower(entry.Address)
		if !strings.HasPrefix(addr, "main/text/") || !dialogueFamily(addr) {
			continue
		}
		payload, err := f.Archives.Containers.ReadFile(entry.Address)
		if err != nil {
			t.Fatal(err)
		}
		for _, candidate := range dialogueProjectionCandidates(t, payload) {
			body := candidate.body
			if delivered, ok := acceptedDialogueTail(candidate.tail); !ok || delivered != body {
				t.Fatalf("%s production accepted interval %q/%v differs from declared projection %q", addr, delivered, ok, body)
			}
			blocks++
			wrapped := ui.NoticeLayoutOf(layout, font, body)
			placed := dialogueLines(t, addr, body, wrapped)
			var sourceWords, drawnWords []string
			for _, piece := range strings.Split(body, "\r\n") {
				sourceWords = append(sourceWords, strings.FieldsFunc(piece, func(r rune) bool { return r == ' ' || r == '\t' })...)
			}
			for _, line := range placed {
				drawnWords = append(drawnWords, line.words...)
			}
			if !slices.Equal(drawnWords, sourceWords) {
				t.Fatalf("%s block%d reached %d of %d source words; visible clamp lost a source suffix", addr, blocks, len(drawnWords), len(sourceWords))
			}
			lines += len(placed)
			for _, bits := range []uint{53, 64} {
				layout.DialogueArithmetic = ui.DialogueArithmetic{Precision: bits}
				calls := text.Record(func() { ui.RenderNotice(layout, font, body, nil) })
				for i, line := range placed {
					if !line.justified {
						continue
					}
					if bits == 53 {
						just++
						positions += len(line.words)
					}
					var glyphs []text.DrawCall
					for _, c := range calls {
						if !c.Flat && c.Color == dlgWhite && c.Y == 36+17*i {
							glyphs = append(glyphs, c)
						}
					}
					x, w := 204, 300
					if line.first {
						x, w = 214, 290
					}
					sum := 0
					for _, word := range line.words {
						sum += font.Advance(word)
					}
					acc, gap := float64(x), float64(w-sum)/float64(len(line.words)-1)
					n := 0
					for _, word := range line.words {
						if n >= len(glyphs) {
							t.Fatalf("missing reached glyph %s PC%d line%d", addr, bits, i)
						}
						if glyphs[n].X+76 != int(acc) {
							t.Fatalf("%s PC%d block%d line%d word%q x%d want%d", addr, bits, blocks, i, word, glyphs[n].X+76, int(acc))
						}
						n += len(word)
						acc = (acc + float64(font.Advance(word))) + gap
					}
					if n != len(glyphs) {
						t.Fatalf("%s PC%d line%d captured%d bytes want%d", addr, bits, i, len(glyphs), n)
					}
				}
			}
		}
	}
	if blocks != wantBlocks || lines != wantLines || just != wantJust || positions != wantWords {
		t.Fatalf("corpus blocks/lines/justified/positions %d/%d/%d/%d want %d/%d/%d/%d", blocks, lines, just, positions, wantBlocks, wantLines, wantJust, wantWords)
	}
	t.Logf("declared candidate projection: named installed corpus %d blocks %d lines %d justified lines %d word coordinates; PC53/64 both match independent per-add double", blocks, lines, just, positions)
}
