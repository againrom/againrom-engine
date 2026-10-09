package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/ui"
)

// townGuardPixel is one fully painted pixel of a guard frame: where it lands on
// the square and the colour it must show.
type townGuardPixel struct {
	at image.Point
	c  color.RGBA
}

// townGuardOracle is the installed guard sheet decoded from its archive path
// without the production loader, one list of fully painted pixels per frame.
// The guards are the last layer of the square, so those pixels are what the
// player sees whatever lies beneath them.
type townGuardOracle [][]townGuardPixel

func newTownGuardOracle(t *testing.T, f *FrontEnd) townGuardOracle {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile("graphics/interface/townbirds/guards/sprites.16a")
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := spr16.DecodeA(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	var o townGuardOracle
	for _, fr := range sheet.Frames {
		var opaque []townGuardPixel
		for i, px := range fr.Pixels {
			if px.Painted && px.Level == 15 {
				c := sheet.Palette[px.Index]
				opaque = append(opaque, townGuardPixel{image.Pt(184+i%fr.Width, 158+i/fr.Width), color.RGBA{c.R, c.G, c.B, 255}})
			}
		}
		o = append(o, opaque)
	}
	// The oracle must tell its own frames apart, or it could not fail.
	for k, opaque := range o {
		pix := image.NewRGBA(image.Rect(0, 0, 640, 480))
		for _, p := range opaque {
			pix.SetRGBA(p.at.X, p.at.Y, p.c)
		}
		if got := o.shown(pix); got != k {
			t.Fatalf("the oracle reads its own frame %d as %d", k, got)
		}
	}
	return o
}

// shown is the frame whose fully painted pixels are all on screen, or -1.
func (o townGuardOracle) shown(pix *image.RGBA) int {
	found := -1
	for k, opaque := range o {
		match := true
		for _, p := range opaque {
			if pix.RGBAAt(p.at.X, p.at.Y) != p.c {
				match = false
				break
			}
		}
		if match {
			if found >= 0 {
				return -2
			}
			found = k
		}
	}
	return found
}

// townFigureRects are the painted figures of the three rooms, as rectangles of
// the room picture.
var townFigureRects = map[townRoom][]struct {
	name string
	r    image.Rectangle
}{
	roomShop: {{"merchant", image.Rect(277, 112, 353, 288)}},
	roomSchool: {
		{"left trainer", image.Rect(0, 200, 172, 424)},
		{"right trainer", image.Rect(320, 200, 480, 424)},
		{"diamond", image.Rect(200, 60, 280, 136)},
		{"column picture", image.Rect(168, 176, 316, 384)},
	},
	roomTavern: {
		{"candle", image.Rect(160, 48, 240, 168)},
		{"cauldron", image.Rect(420, 160, 480, 412)},
		{"tender", image.Rect(240, 152, 420, 364)},
	},
}

// townFigureState is everything a press on a figure must leave as it was.
type townFigureState struct {
	room                 townRoom
	screen, line         string
	gold, table          int
	available, offers    string
	cues, sounds, speech int
	chosen               string
}

// townFiguresRig is the installed first town loaded through App's own LOAD,
// with a fixed clock, the ambient sheets cleared, the tip popups off and every
// sound and speech request recorded, so that nothing plays aloud.
type townFiguresRig struct {
	t              *testing.T
	f              *FrontEnd
	app            *ui.App
	s              *townScreen
	now            time.Time
	sounds         *exteriorRecorder
	speech         *tavernInteriorRecorder
	guards         townGuardOracle
	guard1, guard2 audio.Sample
	rest           image.Point
}

func newTownFiguresRig(t *testing.T) *townFiguresRig {
	t.Helper()
	f := releaseFront(t)
	if f.TownSquareArt.Value() == nil || len(f.TownSquareArt.Value().Problems) != 0 {
		t.Fatalf("exterior load: %v", f.TownSquareArt.Err())
	}
	// The square art is the process's shared install data, so the ambient
	// families are cleared on this front end's own copy: nothing but the guards
	// paints in their rectangle.
	f.TownSquareArt = resolved(withoutAmbience(f.TownSquareArt.Value()), nil)

	r := &townFiguresRig{t: t, f: f, now: time.Unix(100, 0), sounds: &exteriorRecorder{}, speech: &tavernInteriorRecorder{}}
	f.SoundPlayer, f.SpeechPlayer = r.sounds, r.speech
	f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil
	f.SoundBank = OpenSounds(f.Archives.Root)
	f.TownAnimationNow = func() time.Time { return r.now }
	f.TownAnimationRandom = func(int) int { return 0 }
	f.ShopRandom = func(int) int { return 0 }
	f.tipsOff = true
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
	if _, err = store.Write(r.now, payload); err != nil {
		t.Fatal(err)
	}
	r.app = f.App("town figures")
	r.app.Layout(640, 480)
	r.app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	for _, step := range []func() error{func() error { return r.app.HeadlessKey("load") }, func() error { return r.app.HeadlessActivate("@first") }} {
		if err = step(); err != nil {
			t.Fatal(err)
		}
	}
	if r.app.Screen() != ui.ScreenTown {
		t.Fatalf("App LOAD opened screen %s, want the town", r.app.Screen())
	}
	r.s = f.TownScreen().(*townScreen)
	r.guards = newTownGuardOracle(t, f)
	sfx, err := OpenContainers(filepath.Join(f.Archives.Root, "sfx.res"))
	if err != nil {
		t.Fatal(err)
	}
	decode := func(path string) audio.Sample {
		raw, e := sfx.ReadFile("sfx/" + path)
		if e != nil {
			t.Fatal(e)
		}
		sample, e := audio.DecodeWAV(raw, audio.DeviceRate)
		if e != nil {
			t.Fatal(e)
		}
		return sample
	}
	r.guard1, r.guard2 = decode("town/guard1.wav"), decode("town/guard2.wav")
	if reflect.DeepEqual(r.guard1, r.guard2) {
		t.Fatal("the two halberd samples decode alike, so a request cannot be told from the other")
	}
	r.rest = r.maskPoint(0)
	return r
}

// maskPoint is the first pixel of the square whose installed mask byte is code.
func (r *townFiguresRig) maskPoint(code byte) image.Point {
	r.t.Helper()
	mask := r.f.TownSquareArt.Value().Mask
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if mask.ColorIndexAt(x, y) == code {
				return image.Pt(x, y)
			}
		}
	}
	r.t.Fatalf("the square mask holds no byte %#x", code)
	return image.Point{}
}

func (r *townFiguresRig) hover(p image.Point) {
	r.t.Helper()
	if err := r.app.HeadlessPointer("hover", p.X, p.Y); err != nil {
		r.t.Fatal(err)
	}
}

func (r *townFiguresRig) click(p image.Point) {
	r.t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := r.app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			r.t.Fatal(err)
		}
	}
}

// paint composes one frame through App.Draw after dt.
func (r *townFiguresRig) paint(dt time.Duration) *image.RGBA {
	r.t.Helper()
	return exteriorPaint(r.t, r.app, &r.now, dt)
}

// frame is a copy of the frame App composes now, at the unchanged clock.
func (r *townFiguresRig) frame() *image.RGBA {
	r.t.Helper()
	pix, note, err := r.app.HeadlessFrame()
	if err != nil || note != "" {
		r.t.Fatalf("frame %q: %v", note, err)
	}
	out := image.NewRGBA(pix.Rect)
	copy(out.Pix, pix.Pix)
	return out
}

func (r *townFiguresRig) guardRequests() (guard1, guard2 int) {
	for _, sample := range r.sounds.samples {
		switch {
		case reflect.DeepEqual(sample, r.guard1):
			guard1++
		case reflect.DeepEqual(sample, r.guard2):
			guard2++
		}
	}
	return guard1, guard2
}

// state reads the room's state; deep adds the roster selection, which builds
// the whole surface view.
func (r *townFiguresRig) state(deep bool) townFigureState {
	st := townFigureState{
		room:      r.s.room,
		screen:    fmt.Sprint(r.app.Screen()),
		line:      r.app.HeadlessMessage(),
		gold:      r.f.Town.Gold(),
		available: fmt.Sprint(r.f.Town.Available()),
		offers:    fmt.Sprint(r.f.Town.Offers(TownShop), r.f.Town.Offers(TownSchool), r.f.Town.Offers(TownTavern)),
		cues:      r.sounds.ones + r.speech.ones,
		sounds:    len(r.sounds.samples),
		speech:    len(r.speech.samples),
	}
	if r.f.Shop != nil {
		st.table = len(r.f.Shop.Table())
	}
	if deep && (r.s.room == roomTavern || r.s.room == roomSchool) {
		for i, cell := range r.s.TownSurface().Cells {
			if cell.Selected {
				st.chosen += fmt.Sprintf("%d ", i)
			}
		}
	}
	return st
}

// enter opens one door of the square through App and settles the room.
func (r *townFiguresRig) enter(door string, room townRoom) {
	r.t.Helper()
	roomExitEnter(r.t, r.app, r.s, door, room)
	if r.s.room != room && r.s.room != roomTalk {
		r.t.Fatalf("%s opened room %d", door, r.s.room)
	}
}

// endConversation presses OK, or Escape, until the open conversation is over.
func (r *townFiguresRig) endConversation(escape bool) {
	r.t.Helper()
	for n := 0; r.s.room == roomTalk && n < 64; n++ {
		var err error
		if escape {
			err = r.app.HeadlessKey("escape")
		} else {
			err = r.app.HeadlessActivate("dialogue")
		}
		if err != nil {
			r.t.Fatal(err)
		}
	}
	if r.s.room == roomTalk {
		r.t.Fatal("the conversation did not end")
	}
}

// exitPoint is the middle of the room's Exit button.
func (r *townFiguresRig) exitPoint(room townRoom) image.Point {
	r.t.Helper()
	switch room {
	case roomShop:
		x, y, err := r.app.HeadlessShopPoint("button", 3)
		if err != nil {
			r.t.Fatal(err)
		}
		return image.Pt(x, y)
	case roomTavern:
		b := ui.TownSurfaceButtonRect(ui.TownSurfaceTavern, tavernButtonExit)
		return b.Min.Add(b.Max).Div(2)
	case roomSchool:
		b := ui.TownSurfaceButtonRect(ui.TownSurfaceSchool, 1)
		return b.Min.Add(b.Max).Div(2)
	}
	r.t.Fatalf("room %d has no Exit button", room)
	return image.Point{}
}

// leave closes the open room as a player does: a press on its Exit button, or
// Escape with the pointer resting on a spot of the room. It returns where the
// pointer stays.
func (r *townFiguresRig) leave(room townRoom, escape bool) image.Point {
	r.t.Helper()
	at := r.rest
	if escape {
		r.hover(at)
		if err := r.app.HeadlessKey("escape"); err != nil {
			r.t.Fatal(err)
		}
	} else {
		at = r.exitPoint(room)
		r.click(at)
	}
	if r.s.room != roomSquare {
		r.t.Fatalf("leaving room %d (escape %v) reached room %d, want the square", room, escape, r.s.room)
	}
	if r.app.Screen() != ui.ScreenTown || r.app.HeadlessMessage() != "" {
		r.t.Fatalf("after leaving room %d: screen %s, line %q; want the town and no line", room, r.app.Screen(), r.app.HeadlessMessage())
	}
	return at
}

// press clicks every figure of the open room on a grid of points. Every press
// must leave the state as it was, and the frame too at the first and last
// point of each figure; a point that a live control claims is left out.
func (r *townFiguresRig) press(room townRoom, what string) {
	r.t.Helper()
	still := func(p image.Point, before townFigureState, pix *image.RGBA) {
		r.t.Helper()
		if after := r.state(pix != nil); after != before {
			r.t.Fatalf("%s: a press at %v changed the room from %+v to %+v", what, p, before, after)
		}
		if pix != nil && !bytes.Equal(pix.Pix, r.frame().Pix) {
			r.t.Fatalf("%s: a press at %v changed the screen", what, p)
		}
	}
	r.hover(image.Pt(2, 2))
	if a, b := r.frame(), r.frame(); !bytes.Equal(a.Pix, b.Pix) {
		r.t.Fatalf("%s: the room's screen is not steady at a fixed clock", what)
	}
	view := ui.TownSurfaceView{}
	if room != roomShop {
		view = r.s.TownSurface()
	}
	for _, fig := range townFigureRects[room] {
		var points []image.Point
		for y := fig.r.Min.Y + 6; y < fig.r.Max.Y; y += 22 {
			for x := fig.r.Min.X + 6; x < fig.r.Max.X; x += 22 {
				p := image.Pt(x, y)
				if room == roomShop {
					if c, ok := ui.ShopControlAt(p); !ok || c.Kind != ui.ShopControlMerchant {
						r.t.Fatalf("%s: %v is inside the %s but answers %+v", what, p, fig.name, c)
					}
				} else if _, hit := ui.TownSurfaceControlAt(view, p); hit {
					continue
				}
				points = append(points, p)
			}
		}
		if len(points) < 3 {
			r.t.Fatalf("%s: only %d free point(s) inside the %s", what, len(points), fig.name)
		}
		for i, p := range points {
			var pix *image.RGBA
			if i == 0 || i == len(points)-1 {
				r.hover(image.Pt(2, 2))
				pix = r.frame()
			}
			before := r.state(pix != nil)
			r.click(p)
			if pix != nil {
				r.hover(image.Pt(2, 2))
			}
			still(p, before, pix)
		}
		r.t.Logf("%s: %d presses on the %s changed nothing", what, len(points), fig.name)
	}
}

// guardFrames paints one hub of 68 ms per wanted frame and requires the guards
// to show it.
func (r *townFiguresRig) guardFrames(what string, want ...int) {
	r.t.Helper()
	for i, frame := range want {
		if got := r.guards.shown(r.paint(68 * time.Millisecond)); got != frame {
			r.t.Fatalf("%s, hub %d: the guards show frame %d, want %d", what, i+1, got, frame)
		}
	}
}

func repeatFrame(frame, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = frame
	}
	return out
}

// checkGuardsAfterLeaving reads the guards on the square a visit has just left.
// They stand at the last frame of their sheet and stay there wherever the
// pointer rests, with or without a quest at the gate (TOWN-477); the one motion
// the pointer can start is the sweep over the gate with nothing on offer
// (TOWN-476). The visit requests the second halberd sample once, and the first
// only in that sweep.
func (r *townFiguresRig) checkGuardsAfterLeaving(what string, quest bool, at image.Point) {
	r.t.Helper()
	last := len(r.guards) - 1
	base1, base2 := r.guardRequests()
	if code := r.f.TownSquareArt.Value().Mask.ColorIndexAt(at.X, at.Y); code == 0xa0 {
		r.t.Fatalf("%s: the pointer rests at %v over the gate", what, at)
	}
	r.guardFrames(what+", pointer where the exit left it", repeatFrame(last, 8)...)
	r.hover(r.rest)
	r.guardFrames(what+", pointer over no door", repeatFrame(last, 8)...)
	for _, code := range []byte{0x80, 0x90, 0xc0, 0xb0} {
		r.hover(r.maskPoint(code))
		r.guardFrames(fmt.Sprintf("%s, pointer over mask byte %#x", what, code), repeatFrame(last, 3)...)
	}
	if guard1, guard2 := r.guardRequests(); guard1 != base1 || guard2 != base2+1 {
		r.t.Fatalf("%s: the visit requested %d guard1 and %d guard2, want none and one", what, guard1-base1, guard2-base2)
	}
	r.hover(r.maskPoint(0xa0))
	if quest {
		r.guardFrames(what+", pointer over the gate", repeatFrame(last, 8)...)
		if guard1, guard2 := r.guardRequests(); guard1 != base1 || guard2 != base2+1 {
			r.t.Fatalf("%s: the gate with a quest requested %d guard1 and %d guard2, want none and one", what, guard1-base1, guard2-base2)
		}
		return
	}
	sweep := []int{}
	for frame := last - 1; frame >= 0; frame-- {
		sweep = append(sweep, frame)
	}
	sweep = append(sweep, 0)
	r.guardFrames(what+", pointer over the gate", sweep...)
	if guard1, guard2 := r.guardRequests(); guard1 != base1+1 || guard2 != base2+1 {
		r.t.Fatalf("%s: the gate with nothing on offer requested %d guard1 and %d guard2, want one and one", what, guard1-base1, guard2-base2)
	}
}

// setQuest sets whether the gate has a mission on offer; the list is model
// state, not animation state.
func (r *townFiguresRig) setQuest(on bool) {
	if on {
		r.f.Town.announceMission(r.f.Town.currentMain())
		return
	}
	clearTownTestGateLatches(r.f.Town)
}

// The installed shop, school and tavern through App input: every painted figure
// is pressed on a grid of points and none of them answers (the shop's merchant
// after the shop's quest is taken, the two trainers, the diamond and the column
// picture of the school, the candle, cauldron and tender of the tavern;
// TOWN-478, TAVERN-CLICK-019). Leaving each room by its Exit button and by
// Escape, with and without a quest at the gate, shows the guards at the last
// frame of their sheet and moves them only when the pointer is over the gate
// with nothing on offer. The guards' frame is read from the installed sheet's
// own pixels in the composed screen.
func TestReleaseTownFiguresTakeNoPressAndGuardsRestAfterRoomExits(t *testing.T) {
	r := newTownFiguresRig(t)
	if len(r.guards) != 8 {
		t.Fatalf("the guard sheet holds %d frames, want 8", len(r.guards))
	}

	// The shop: its quest conversation opens on entry and takes the offer as it
	// opens, and App input ends the conversation whether it ends by OK or by
	// Escape. The merchant is pressed with the offer taken, where nothing is
	// left to offer.
	r.enter("SHOP", roomShop)
	if r.s.room != roomTalk || len(r.f.Town.Offers(TownShop)) != 0 || len(r.f.Town.Available()) == 0 {
		t.Fatalf("the first town's shop opened room %d with %d offer(s) and %v available, want its quest conversation with the offer taken", r.s.room, len(r.f.Town.Offers(TownShop)), r.f.Town.Available())
	}
	r.endConversation(false)
	if r.s.room != roomShop || len(r.f.Town.Offers(TownShop)) != 0 {
		t.Fatalf("the conversation left room %d with %d offer(s), want the shop and the offer taken", r.s.room, len(r.f.Town.Offers(TownShop)))
	}
	r.press(roomShop, "shop with the offer taken")
	before := r.state(false)
	x, y, err := r.app.HeadlessShopPoint("shelf_pick", 0)
	if err != nil {
		t.Fatal(err)
	}
	r.click(image.Pt(x, y))
	if after := r.state(false); after.cues <= before.cues {
		t.Fatalf("a press on a shelf pick played no cue (%+v to %+v), so this route cannot answer a press", before, after)
	}
	r.leave(roomShop, true)

	// The tavern and the school.
	r.enter("TAVERN", roomTavern)
	if r.s.room != roomTavern {
		t.Fatalf("the tavern opened room %d", r.s.room)
	}
	r.press(roomTavern, "tavern")
	before = r.state(false)
	if !hireControl(t, r.f, r.app, r.s) {
		t.Fatal("the first town lists no mercenary squad, so this route has no press that shows a line")
	}
	if after := r.state(false); after.line != "" {
		t.Fatalf("a hire left the line %q (was %q), want none", after.line, before.line)
	}
	r.leave(roomTavern, false)
	r.enter("SCHOOL", roomSchool)
	r.endConversation(false)
	if r.s.room != roomSchool {
		t.Fatalf("the school opened room %d", r.s.room)
	}
	r.press(roomSchool, "school")
	r.leave(roomSchool, true)

	// Leaving each room, with and without a quest at the gate.
	exits := 0
	for _, quest := range []bool{true, false} {
		for _, room := range []struct {
			door string
			room townRoom
		}{{"SHOP", roomShop}, {"TAVERN", roomTavern}, {"SCHOOL", roomSchool}} {
			for _, escape := range []bool{false, true} {
				r.enter(room.door, room.room)
				if r.s.room == roomTalk {
					r.endConversation(false)
				}
				if r.s.room != room.room {
					t.Fatalf("%s opened room %d", room.door, r.s.room)
				}
				r.setQuest(quest)
				how := "Exit button"
				if escape {
					how = "Escape"
				}
				at := r.leave(room.room, escape)
				r.checkGuardsAfterLeaving(fmt.Sprintf("%s left by %s, quest %v", room.door, how, quest), quest, at)
				exits++
			}
		}
	}
	guard1, guard2 := r.guardRequests()
	t.Logf("%d room exits checked against the installed guard sheet; %d guard1 and %d guard2 requests in the run", exits, guard1, guard2)
}
