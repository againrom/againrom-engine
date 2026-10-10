package game

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"testing"

	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The character generator trace is a behaviour record of the generator on one
// install: New Game, a scripted pointer and keyboard path over every pre-create
// and detail control, the tip popups and their cycles, Back, Reset and the
// commit. Each tick records the composed frame's hash, the message line and
// the sound and music requests; the commit records the party it built. The
// recorded lines live in testdata/chargentrace, one file per install text, and
// AGAINROM_TOWN_TRACE_WRITE=1 rewrites them.

// chargenTraceMaskPoints answers, for each wanted byte of the installed mask at
// addr, the pixel at the middle of that byte's raster run, offset by at.
func chargenTraceMaskPoints(t *testing.T, f *FrontEnd, addr string, at image.Point, want []byte) map[byte]image.Point {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatal(err)
	}
	mask, err := terrain.DecodeBMP8(raw)
	if err != nil {
		t.Fatal(err)
	}
	runs := map[byte][]image.Point{}
	for y := 0; y < mask.Bounds().Dy(); y++ {
		for x := 0; x < mask.Bounds().Dx(); x++ {
			c := mask.ColorIndexAt(x, y)
			runs[c] = append(runs[c], image.Pt(x, y).Add(at))
		}
	}
	out := map[byte]image.Point{}
	for _, c := range want {
		r := runs[c]
		if len(r) == 0 {
			t.Fatalf("%s: mask byte %d has no pixel", addr, c)
		}
		out[c] = r[len(r)/2]
	}
	return out
}

type chargenTrace struct {
	*townTrace
}

func (tr chargenTrace) where() string {
	where := fmt.Sprintf("screen%d", tr.a.Screen())
	if s, ok := tr.a.HeadlessChargenState(); ok {
		where += fmt.Sprintf("+%s+focus%d", s.Stage, s.Focus)
	}
	return where
}

func (tr chargenTrace) record(kind string) {
	frame := "none"
	if tr.a.Screen() == ui.ScreenChargen || tr.a.Screen() == ui.ScreenMenu {
		frame = tr.frame()
	}
	tr.lines = append(tr.lines, fmt.Sprintf("%04d %s %s frame=%s msg=%q snd=[%s]",
		tr.tick, kind, tr.where(), frame, tr.a.HeadlessMessage(), tr.sound.take()))
	tr.tick++
}

func (tr chargenTrace) hover(p image.Point, n int) {
	for i := 0; i < n; i++ {
		if err := tr.a.HeadlessPointer("hover", p.X, p.Y); err != nil {
			tr.t.Fatal(err)
		}
		tr.record(fmt.Sprintf("hover%d,%d", p.X, p.Y))
	}
}

func (tr chargenTrace) edge(action string, p image.Point) {
	if err := tr.a.HeadlessPointer(action, p.X, p.Y); err != nil {
		tr.t.Fatal(err)
	}
	tr.record(fmt.Sprintf("%s%d,%d", action, p.X, p.Y))
}

func (tr chargenTrace) click(p image.Point) {
	tr.edge("press", p)
	tr.edge("release", p)
}

func (tr chargenTrace) key(name string) {
	if err := tr.a.HeadlessKey(name); err != nil {
		tr.t.Fatal(err)
	}
	tr.record("key-" + name)
}

func (tr chargenTrace) typed(text string, backspace bool) {
	if err := tr.a.HeadlessType(text, backspace); err != nil {
		tr.t.Fatal(err)
	}
	tr.record(fmt.Sprintf("type%q-%t", text, backspace))
}

func (tr chargenTrace) expect(stage string) {
	tr.t.Helper()
	s, ok := tr.a.HeadlessChargenState()
	if !ok || s.Stage != stage {
		tr.t.Fatalf("tick %d stands at %s, want %s", tr.tick, tr.where(), stage)
	}
}

// party records the commit's party: each member's name, hero, weapon, worn and
// carried codes and figure.
func (tr chargenTrace) party(kind string) {
	var b []byte
	for _, m := range tr.f.liveParty {
		weapon := ""
		if m.Weapon != nil {
			weapon = m.Weapon.Name
		}
		b = fmt.Appendf(b, "%s|%+v|%s|%v|%v|%s|%s|%d|%v|%v\n", m.Name, m.Hero, weapon, m.Worn, m.Carried,
			m.Body, m.FigureDir, m.FigureFace, m.Mage, m.KnownSpells)
	}
	sum := sha256.Sum256(b)
	tr.lines = append(tr.lines, fmt.Sprintf("%04d %s %s members=%d party=%s msg=%q",
		tr.tick, kind, tr.where(), len(tr.f.liveParty), hex.EncodeToString(sum[:8]), tr.a.HeadlessMessage()))
	tr.tick++
}

// TestReleaseChargenTraceIsUnchanged pins the first game's generator, frame
// by frame, across a move of its code.
func TestReleaseChargenTraceIsUnchanged(t *testing.T) {
	f := releaseFront(t)
	if f.ChargenAssets == nil {
		t.Skip("the install ships no character generation")
	}
	f.randomService().SetStreamSeed(random.Music, 3)
	f.PersistenceContext.tipsOff = false
	sounds := &traceSounds{}
	f.SoundPlayer, f.SpeechPlayer, f.AmbientPlayer = sounds, sounds, sounds
	f.SoundBank = OpenSounds(f.Archives.Root)
	a := f.App("chargen-trace")
	a.Layout(640, 480)
	a.SetMusic(traceMusicSource{inner: f.MusicBank, log: sounds}, traceMusicDevice{log: sounds}, f.randomService().Stream(random.Music))
	a.SetNewGameChargen(func() *ui.ChargenEntry {
		return &ui.ChargenEntry{Model: ui.NewChargen(f.ChargenSetup()), Begin: func(res ui.ChargenResult) (ui.MapOpener, error) {
			return f.NewGameOpener(10, res), nil
		}}
	})
	tr := chargenTrace{&townTrace{t: t, f: f, a: a, sound: sounds}}

	pre := chargenTraceMaskPoints(t, f, f.generator().PreCreate.Mask.Key, image.Point{},
		[]byte{20, 40, 60, 80, 100, 120, 140, 160, 180})
	fighter := chargenTraceMaskPoints(t, f, graphicsPrefix+"interface/chrgen/fighter/mask.bmp", image.Pt(160, 0),
		[]byte{255, 191, 152, 127, 102})
	mage := chargenTraceMaskPoints(t, f, graphicsPrefix+"interface/chrgen/mag/mask.bmp", image.Pt(160, 0),
		[]byte{255, 191, 152, 127, 102})
	off := image.Pt(320, 20)
	name := image.Pt(290, 322)
	accept, reset, back := image.Pt(554, 67), image.Pt(554, 114), image.Pt(554, 161)
	tipClose := func(r image.Rectangle) image.Point {
		c := ui.TipPanelCloseRect(r)
		return c.Min.Add(c.Size().Div(2))
	}
	tipToggle := func(r image.Rectangle) image.Point {
		c := ui.TipPanelToggleRect(r)
		return c.Min.Add(c.Size().Div(2))
	}

	if err := a.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	tr.record("new-game")
	tr.expect(ui.ChargenStagePreCreate)

	// Pre-create: rest, every control hovered, the cycle and sparkle running.
	tr.hover(off, 40)
	for _, c := range []byte{20, 40, 60, 80, 100, 120, 140, 160, 180} {
		tr.hover(pre[c], 6)
		tr.hover(off, 30)
	}
	tr.hover(name, 4)
	// Tip steps: a hero, then a level; each hero and level once.
	tr.click(pre[100])
	tr.hover(off, 40)
	tr.click(pre[60])
	tr.hover(off, 40)
	for _, c := range []byte{80, 120, 140, 20, 40} {
		tr.hover(pre[c], 3)
		tr.click(pre[c])
		tr.hover(off, 8)
	}
	// A double click on a hero continues; Back returns.
	tr.click(pre[80])
	tr.click(pre[80])
	tr.expect(ui.ChargenStageDetailed)
	tr.click(back)
	tr.expect(ui.ChargenStagePreCreate)
	// The name: focus, type, erase, keyboard focus moves.
	tr.click(name)
	tr.typed("Ab", false)
	tr.hover(off, 40)
	tr.typed("", true)
	tr.typed("cdefghijklm", false)
	tr.hover(off, 10)
	for i := 0; i < 12; i++ {
		tr.typed("", true)
	}
	tr.key("enter")
	tr.hover(off, 4)
	tr.typed("Trace", false)
	for _, k := range []string{"down", "down", "down", "up", "enter", "down", "down", "down", "down", "down", "down", "down", "up"} {
		tr.key(k)
	}
	// The toggle and close of the pre-create popup.
	tr.hover(tipToggle(preCreateTipRect), 4)
	tr.click(tipToggle(preCreateTipRect))
	tr.click(tipToggle(preCreateTipRect))
	tr.click(tipClose(preCreateTipRect))
	tr.hover(off, 20)
	tr.click(pre[100])
	tr.click(pre[180])
	tr.expect(ui.ChargenStageDetailed)

	// Detail, female fighter: rest, the skill cycle, every skill hovered.
	tr.hover(off, 40)
	for _, c := range []byte{255, 191, 152, 127, 102} {
		tr.hover(fighter[c], 5)
		tr.hover(off, 30)
	}
	tr.click(fighter[152])
	tr.hover(off, 10)
	tr.click(fighter[102])
	tr.click(fighter[102])
	tr.hover(off, 4)
	// Every stat button: hovered, pressed, released.
	for stat := 0; stat < 4; stat++ {
		_, value, lower, raise, ok := ui.DetailedAttributeBoxes(f.generator(), stat)
		if !ok {
			t.Fatal("no attribute box")
		}
		for _, r := range []image.Rectangle{value, raise, lower} {
			p := r.Min.Add(r.Size().Div(2))
			tr.hover(p, 3)
			tr.click(p)
			tr.click(p)
		}
		plate, _, _, _, _ := ui.DetailedAttributeBoxes(f.generator(), stat)
		tr.hover(plate.Min.Add(plate.Size().Div(2)), 3)
	}
	tr.hover(image.Pt(84, 192), 3)
	// The held raise repeats until the pool or the bound refuses it.
	_, _, _, raise, _ := ui.DetailedAttributeBoxes(f.generator(), 0)
	hold := raise.Min.Add(raise.Size().Div(2))
	tr.edge("press", hold)
	for i := 0; i < 60; i++ {
		tr.edge("move", hold)
	}
	tr.edge("release", hold)
	_, _, lower, _, _ := ui.DetailedAttributeBoxes(f.generator(), 1)
	for i := 0; i < 20; i++ {
		tr.click(lower.Min.Add(lower.Size().Div(2)))
	}
	tr.hover(image.Pt(80, 360), 4)
	tr.hover(image.Pt(560, 360), 4)
	for _, p := range []image.Point{accept, reset, back} {
		tr.hover(p, 3)
		tr.edge("press", p)
		tr.hover(off, 2)
		tr.edge("release", off)
	}
	tr.click(reset)
	tr.hover(tipToggle(chargenTipRect), 3)
	tr.click(tipClose(chargenTipRect))
	tr.hover(off, 10)
	for _, k := range []string{"down", "down", "enter", "down", "down", "down", "down", "down", "enter", "up"} {
		tr.key(k)
	}
	tr.key("escape")
	tr.expect(ui.ChargenStagePreCreate)

	// The male mage's detail page, then the reserved and accepted names.
	tr.click(pre[140])
	tr.click(pre[180])
	tr.expect(ui.ChargenStageDetailed)
	tr.hover(off, 40)
	for _, c := range []byte{255, 191, 152, 127, 102} {
		tr.hover(mage[c], 5)
		tr.click(mage[c])
		tr.hover(off, 5)
	}
	tr.click(back)
	tr.click(name)
	for i := 0; i < 12; i++ {
		tr.typed("", true)
	}
	tr.click(pre[180])
	tr.hover(off, 4)
	tr.typed("self", false)
	tr.click(pre[180])
	tr.click(accept)
	tr.hover(off, 4)
	tr.click(back)
	tr.click(name)
	tr.typed("", true)
	tr.typed("lfin", false)
	tr.click(pre[180])
	tr.expect(ui.ChargenStageDetailed)
	tr.click(mage[191])
	tr.click(accept)
	if a.Screen() == ui.ScreenChargen {
		t.Fatalf("accept left the generator showing: %q", a.HeadlessMessage())
	}
	tr.party("accept")

	// Pre-create Escape leaves to the menu.
	a2 := f.App("chargen-trace-escape")
	a2.Layout(640, 480)
	a2.SetNewGameChargen(func() *ui.ChargenEntry {
		return &ui.ChargenEntry{Model: ui.NewChargen(f.ChargenSetup())}
	})
	tr.a = a2
	if err := a2.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	tr.record("new-game")
	tr.click(pre[160])
	if a2.Screen() != ui.ScreenMenu {
		t.Fatalf("the amulet left screen %v", a2.Screen())
	}
	if err := a2.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	tr.record("new-game")
	tr.key("escape")
	tr.compare("chargentrace")
}
