package ui

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/render/text"
)

func secondTestGenerator(t *testing.T) *GeneratorDescription {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "game", "generators", "rom2.json"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := DecodeGenerator(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// The second game's description names the card shadow, the hovered and held
// caption, the lingering skill light and the mixed name field; the first
// game's names no shadow, its own caption ink and offset, no linger and no mix.
func TestSecondGeneratorNamesItsTextDetails(t *testing.T) {
	d := secondTestGenerator(t).Detail
	if s := d.CardShadow; s == nil || s.Ink != (GeneratorInk{8, 8, 8}) || s.At != (GeneratorPoint{1, 1}) {
		t.Errorf("card shadow %+v, want (8,8,8) at (1,1)", s)
	}
	if k := d.CommandHoverInk; k == nil || *k != (GeneratorInk{255, 255, 116}) {
		t.Errorf("command hover ink %v, want (255,255,116)", k)
	}
	if p := d.CommandSink; p == nil || *p != (GeneratorPoint{1, 1}) {
		t.Errorf("command sink %v, want (1,1)", p)
	}
	if !d.SkillStates.Linger || !secondTestGenerator(t).PreCreate.Name.Over {
		t.Error("the second game's skill light does not linger or its name field does not mix")
	}
	first := testGenerator()
	if f := first.Detail; f.CardShadow != nil || f.SkillStates.Linger || first.PreCreate.Name.Over {
		t.Error("the first game's description names a second-game text detail")
	}
	if f := first.Detail; f.CommandHoverInk == nil || f.CommandHoverInk.RGBA() != plaqueCommandInk.Hover || f.CommandSink == nil || f.CommandSink.Pt() != plaqueSink {
		t.Errorf("the first game's caption ink %v and offset %v, want %v and %v", f.CommandHoverInk, f.CommandSink, plaqueCommandInk.Hover, plaqueSink)
	}
}

// Every label and value of a card built through CardView draws over its
// shadow one pixel right and down; the first game's card draws none.
func TestCardTextDrawsOverItsShadow(t *testing.T) {
	shadow := color.RGBA{8, 8, 8, 255}
	subject := PanelSubject{Name: "ALDOR", HP: 139, MaxHP: 142, DetailLevel: 7, DetailSet: true}
	card := func(l *GeneratorDescription) (*image.RGBA, PanelLayout) {
		p := &ChargenPresentation{Layout: l, Font: chargenTestFont()}
		layout := p.CardView(subject, image.Rect(0, 238, 160, 480), image.Point{}, nil, nil).statsLayout()
		return RenderCharacterPanel(layout, chargenTestFont(), subject), layout
	}
	plain, layout := card(testGenerator())
	if layout.Shadow != nil || countChargenPixel(plain, shadow) != 0 {
		t.Fatalf("the first game's card draws a shadow: %+v", layout.Shadow)
	}
	shaded, layout := card(secondTestGenerator(t))
	if layout.Shadow == nil {
		t.Fatal("the second game's card has no shadow")
	}
	ink := func(c color.RGBA) bool {
		return c == layout.LabelColor || c == layout.ValueColor || c == characterCardHeading
	}
	checked := 0
	b := plain.Bounds()
	for y := b.Min.Y; y < b.Max.Y-1; y++ {
		for x := b.Min.X; x < b.Max.X-1; x++ {
			if !ink(plain.RGBAAt(x, y)) || ink(plain.RGBAAt(x+1, y+1)) {
				continue
			}
			checked++
			if got := shaded.RGBAAt(x+1, y+1); got != shadow {
				t.Fatalf("below-right of the text pixel (%d,%d): %v, want the shadow %v", x, y, got, shadow)
			}
		}
	}
	if checked == 0 {
		t.Fatal("the card drew no text edge")
	}
}

// A hovered command caption turns the description's hover ink over the
// unchanged plaque; held, the on picture draws and the caption moves by the
// sink.
func TestSecondGeneratorCommandCaptionHover(t *testing.T) {
	l := secondTestGenerator(t)
	art := &ChargenPresentation{Layout: l}
	on := color.RGBA{R: 90, A: 255}
	for i, cmd := range l.Detail.Commands {
		art.NavButtons[i][1] = panelTestPicture(cmd.Rect.Rectangle(), on)
	}
	c := NewChargen(ChargenSetup{PreCreate: &ChargenPreCreate{Art: art},
		Detailed: &ChargenDetailed{Play: "A", Reset: "R", Back: "B", Restore: "S"}})
	under := color.RGBA{B: 70, A: 255}
	compose := func(hover, pressed chargenControl) *image.RGBA {
		dst := panelTestPicture(image.Rect(0, 0, 640, 480), under)
		p := chargenCommandPanel(c, hover, pressed)
		p.drawButtons(dst, chargenTestFont())
		return dst
	}
	r := l.Detail.Commands[0].Rect.Rectangle()
	hover := compose(chargenPlay, chargenNone)
	ink := color.RGBA{255, 255, 116, 255}
	inked, plaque := image.Rectangle{}, 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			switch hover.RGBAAt(x, y) {
			case ink:
				inked = inked.Union(image.Rect(x, y, x+1, y+1))
			case under:
				plaque++
			}
		}
	}
	if inked.Empty() || plaque+inked.Dx()*inked.Dy() != r.Dx()*r.Dy() {
		t.Fatalf("hovered plaque: caption %v in the hover ink, %d plaque pixels unchanged of %d", inked, plaque, r.Dx()*r.Dy())
	}
	held := compose(chargenPlay, chargenPlay)
	moved := inked.Add(image.Pt(1, 1))
	if got := held.RGBAAt(moved.Min.X, moved.Min.Y); got != ink {
		t.Errorf("held caption corner %v = %v, want the hover ink one pixel right and down", moved.Min, got)
	}
	if got := held.RGBAAt(r.Min.X, r.Min.Y); got != on {
		t.Errorf("held plaque = %v, want the on picture %v", got, on)
	}
}

// The last hovered skill stays lit after the pointer leaves it, until
// another skill is hovered or a skill is pressed.
func TestSkillLightLingersUntilAnotherHoverOrAPress(t *testing.T) {
	setup := soundChargenSetup()
	l := *testGenerator()
	l.Detail.SkillStates.Linger = true
	setup.PreCreate.Art.Layout = &l
	a := newTestApp(t, appRows(1), okLoader(t))
	c := NewChargen(setup)
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	soundClick(a, &now, preControlRect(c, chargenForward).Min)
	if c.Stage() != DetailedStage {
		t.Fatalf("stage %v, want the detailed page", c.Stage())
	}
	move := func(p image.Point) {
		now = now.Add(time.Second)
		a.step(appInput{CursorX: p.X, CursorY: p.Y, Viewer: Input{CursorX: p.X, CursorY: p.Y}}, now)
	}
	lit := func() []bool {
		out := make([]bool, c.selectableSkills())
		for i := range out {
			_, out[i] = detailedSkillSelectedHover(c, chargenSkill0+chargenControl(i), chargenNone, chargenNone)
		}
		return out
	}
	away := detailedControlRect(c, chargenStatPlus0).Min
	move(detailedControlRect(c, chargenSkill2).Min)
	move(away)
	if got := lit(); !got[2] || got[0] || got[1] {
		t.Fatalf("after leaving skill 2: lit %v, want skill 2 alone", got)
	}
	move(detailedControlRect(c, chargenSkill1).Min)
	move(away)
	if got := lit(); !got[1] || got[2] {
		t.Fatalf("after leaving skill 1: lit %v, want skill 1 alone", got)
	}
	soundClick(a, &now, detailedControlRect(c, chargenSkill0).Min)
	move(away)
	for i, on := range lit() {
		if on {
			t.Errorf("after a skill press skill %d stays lit", i)
		}
	}
	plain := testGenerator()
	setup.PreCreate.Art.Layout = plain
	c2 := NewChargen(setup)
	c2.noteSkillHover(chargenSkill2)
	c2.noteSkillHover(chargenNone)
	if _, on := detailedSkillSelectedHover(c2, chargenSkill2, chargenNone, chargenNone); on {
		t.Error("the first game's skill light lingers")
	}
}

// The name field mixes a coverage glyph into the page where the description
// says so, and darkens it as before where it does not.
func TestNameFieldMixesCoverageGlyphs(t *testing.T) {
	coverage := chargenTestFont()
	for i := range coverage.Glyphs {
		g := &coverage.Glyphs[i]
		g.CoverageLevels = true
		for j := range g.Pixels {
			g.Pixels[j].Level = 5
		}
	}
	empty := &text.Font{Glyphs: make([]text.Glyph, 224)}
	frame := func(over bool, font *text.Font) *image.RGBA {
		l := *testGenerator()
		l.PreCreate.Name.Over = over
		art := &ChargenPresentation{Layout: &l, NameFont: font}
		bg := image.NewRGBA(image.Rect(0, 0, 640, 480))
		draw.Draw(bg, bg.Bounds(), &image.Uniform{C: color.RGBA{60, 50, 40, 255}}, image.Point{}, draw.Src)
		art.Background = bg
		c := NewChargen(ChargenSetup{Name: "N", PreCreate: &ChargenPreCreate{Prompt: "P", Art: art}})
		return composeChargenPage(c, chargenNone, chargenNone)
	}
	l := testGenerator()
	at := l.PreCreate.Name.TextAt.Pt()
	under := frame(false, empty).RGBAAt(at.X, at.Y)
	ink := l.PreCreate.Name.Ink.RGBA()
	if got, want := frame(true, coverage).RGBAAt(at.X, at.Y), text.Mix(under, ink, 5); got != want {
		t.Errorf("mixed name pixel = %v, want %v over %v", got, want, under)
	}
	if got, want := frame(false, coverage).RGBAAt(at.X, at.Y), text.Shade(ink, 5); got != want {
		t.Errorf("darkened name pixel = %v, want %v", got, want)
	}
}
