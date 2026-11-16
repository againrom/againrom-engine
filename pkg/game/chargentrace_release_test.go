package game

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"strings"
	"testing"

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
		hero := strings.Replace(fmt.Sprintf("%+v", m.Hero), " AuthoredHealthMax:0}", "}", 1)
		b = fmt.Appendf(b, "%s|%s|%s|%v|%v|%s|%s|%d|%v|%v\n", m.Name, hero, weapon, m.Worn, m.Carried,
			m.Body, m.FigureDir, m.FigureFace, m.Mage, m.KnownSpells)
	}
	sum := sha256.Sum256(b)
	tr.lines = append(tr.lines, fmt.Sprintf("%04d %s %s members=%d party=%s msg=%q",
		tr.tick, kind, tr.where(), len(tr.f.liveParty), hex.EncodeToString(sum[:8]), tr.a.HeadlessMessage()))
	tr.tick++
}
