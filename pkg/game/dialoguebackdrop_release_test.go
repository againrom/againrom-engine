package game

import (
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// The original's shop dialogue "Interested to hear the news?" as the owner's
// EN screenshot shows it at 640x480: picture coordinates inside npc31's window
// (34,21)-(106,117) and the colour there, an RGB565 value with its low bits
// clear. Written out here rather than read from the install, so the witness
// does not grade the code by the art it reads.
var dialogueBackdropWant = []struct {
	at image.Point
	c  color.RGBA
}{
	{image.Pt(56, 54), color.RGBA{R: 72, G: 60, B: 48, A: 0xff}},
	{image.Pt(60, 45), color.RGBA{R: 48, G: 40, B: 24, A: 0xff}},
	{image.Pt(95, 30), color.RGBA{R: 48, G: 36, B: 16, A: 0xff}},
	{image.Pt(91, 43), color.RGBA{R: 32, G: 24, B: 8, A: 0xff}},
	{image.Pt(43, 68), color.RGBA{R: 32, G: 24, B: 8, A: 0xff}},
	{image.Pt(43, 47), color.RGBA{R: 32, G: 24, B: 8, A: 0xff}},
	{image.Pt(97, 57), color.RGBA{R: 16, G: 12, A: 0xff}},
	{image.Pt(50, 31), color.RGBA{R: 16, G: 16, A: 0xff}},
	{image.Pt(97, 39), color.RGBA{R: 40, G: 32, B: 16, A: 0xff}},
}

// A dialogue speaker stands on the installed backdrop cut at the speaker's own
// window (DIV-1485), through ordinary town input on the shipped first town:
// the shopkeeper's quest dialogue that opens when the shop is entered, then
// the tavern NPC's dialogue through that NPC's own row.
func TestReleaseDialoguePortraitBackdrop(t *testing.T) {
	f := releaseFront(t)
	app, s := openFirstTownShopDialogue(t, f, "dialogue portrait backdrop")
	window := checkDialogueBackdrop(t, f, app, s, "main/text/shop/npc31m31.txt")
	pix, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range dialogueBackdropWant {
		at := window.at.Add(p.at.Sub(window.src.Min))
		got := pix.RGBAAt(at.X, at.Y)
		got.R, got.G, got.B = got.R&0xf8, got.G&0xfc, got.B&0xf8
		if got != p.c {
			t.Errorf("shopkeeper picture %v shows %v in RGB565, the original %v", p.at, got, p.c)
		}
	}

	// Accept through the dialogue's own control, leave the shop, and talk to
	// the tavern's NPC.
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
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
	checkDialogueBackdrop(t, f, app, s, fmt.Sprintf("main/text/inn/npc/npc%02dm%02d.txt", offers[0].NPC, offers[0].Mission))
}

// openFirstTownShopDialogue starts the first town from a saved game and enters
// its shop through ordinary App input, which opens the shopkeeper's quest
// dialogue. Town animation is pinned to one instant and the shop's roll to zero,
// so the frames it presents are reproducible.
func openFirstTownShopDialogue(t *testing.T, f *FrontEnd, name string) (*ui.App, *townScreen) {
	t.Helper()
	r := &tavernInteriorRecorder{}
	f.SpeechPlayer = r
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil
	f.Carried = f.NextParty()
	f.arriveInTown()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(200, 0), payload); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.ShopRandom = func(int) int { return 0 }
	app := f.App(name)
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	for _, target := range []string{"load game", "@first", "SHOP"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	return app, f.TownScreen().(*townScreen)
}

type dialogueWindow struct {
	at  image.Point
	src image.Rectangle
}

// checkDialogueBackdrop reads the open town dialogue's picture window from the
// presented frame: a transparent face pixel must show t_back.bmp read from the
// install at the same picture coordinate, an opaque one the face, and a pixel
// under the engine's portrait border that border.
func checkDialogueBackdrop(t *testing.T, f *FrontEnd, app *ui.App, s *townScreen, wantPath string) dialogueWindow {
	t.Helper()
	path, _ := s.townTextPath()
	if s.room != roomTalk || path != wantPath {
		t.Fatalf("room %d text %q; want the dialogue %s", s.room, path, wantPath)
	}
	pix, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	layout, payload, _ := s.townDialogueLayout()
	speaker, named := EventPartSpeaker(payload, s.said, HeroAudience(f.Carried))
	face, win, found := s.speakerFace(speaker)
	if !named || !found || face == nil {
		t.Fatalf("%s: speaker npc%d named %v found %v", wantPath, speaker, named, found)
	}
	back, err := readChargenBMP(f.Archives.Containers, tipBackPath)
	if err != nil {
		t.Fatal(err)
	}
	l := layout.WithFaceWindow(win)
	// The picture surface is 88x108 and bottom-up (read off the owner's
	// screenshot of the original): the window's (8,7) is measured from the
	// surface's bottom edge, so a 96-row window starts at surface row 5 and a
	// 92-row one at row 9, and the border's stored rows come out upside down.
	surface := l.Box.Min.Add(l.Portrait.Min)
	w := dialogueWindow{at: surface.Add(image.Pt(8, 108-7-l.PortraitWindow.Dy())), src: l.PortraitWindow}
	borderAt := func(p image.Point) color.RGBA {
		q := p.Sub(surface)
		if q.X < 0 || q.Y < 0 || q.X >= 88 || q.Y >= 108 {
			return color.RGBA{}
		}
		return l.Frame.Portrait.RGBAAt(q.X, 107-q.Y)
	}
	fb := face.Bounds().Min
	var backdrop, figure, covered, keyed, bad int
	for y := 0; y < w.src.Dy(); y++ {
		for x := 0; x < w.src.Dx(); x++ {
			p := w.at.Add(image.Pt(x, y))
			src := w.src.Min.Add(image.Pt(x, y))
			want := face.RGBAAt(fb.X+src.X, fb.Y+src.Y)
			switch b := borderAt(p); {
			case b.A == 0xff:
				want = b
				covered++
			case b.A != 0:
				covered++
				continue
			case want.A == 0:
				want = back.RGBAAt(src.X, src.Y)
				backdrop++
			case want.A == 0xff:
				figure++
			default:
				t.Fatalf("%s: face pixel %v is partly transparent", wantPath, src)
			}
			// The surface is blitted keyed on black at the screen's depth: a
			// picture pixel of 5-6-5 value 0 is not drawn, so the black fill
			// under it shows, or the frame's body above the fill.
			if want.R>>3 == 0 && want.G>>2 == 0 && want.B>>3 == 0 && borderAt(p).A != 0xff {
				keyed++
				continue
			}
			want = dialoguePackedPicture(want)
			if got := pix.RGBAAt(p.X, p.Y); got != want {
				if bad < 5 {
					t.Errorf("%s: picture pixel %v shows %v, want %v", wantPath, src, got, want)
				}
				bad++
			}
		}
	}
	t.Logf("%s: speaker npc%d window %v at %v: %d backdrop, %d figure, %d border, %d keyed pixels; %d differ",
		wantPath, speaker, w.src, w.at, backdrop, figure, covered, keyed, bad)
	// The border art writes 404 pixels inside a 72x96 window and 118 inside a
	// 72x92 one (`DLG-PORTRAIT-036`), counted in the border's own rows.
	if want := map[int]int{96: 404, 92: 118}[w.src.Dy()]; covered != want {
		t.Errorf("%s: the border covers %d pixels of the %d-row window, want %d", wantPath, covered, w.src.Dy(), want)
	}
	if bad > 0 || backdrop < 1000 || figure < 1000 {
		t.Fatalf("%s: %d of %d picture pixels differ (%d backdrop, %d figure)", wantPath, bad, w.src.Dx()*w.src.Dy(), backdrop, figure)
	}
	return w
}

func dialoguePackedPicture(c color.RGBA) color.RGBA {
	return color.RGBA{uint8(int(c.R>>3) * 255 / 31), uint8(int(c.G>>2) * 255 / 63), uint8(int(c.B>>3) * 255 / 31), c.A}
}
