package game

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"againrom/pkg/ui"
)

// The first game's detailed generator draws its four commands as the shop's
// four-button composition (owner, DIV-2868 after DIV-483): the 176x238 shop
// menu over TownWideUpperRegion and, top to bottom, Accept, Restore, Reset and
// Back at the shop's four plaque rectangles, each plaque's bitmap drawn only
// while it is held.

// chargenPanelOrder is the owner's top-to-bottom order as the description's
// roles.
var chargenPanelOrder = [4]string{"play", "restore", "reset", "back"}

func chargenPanelInks() []color.RGBA {
	return []color.RGBA{{R: 255, G: 230, B: 150, A: 255}, {R: 255, G: 255, B: 224, A: 255}}
}

func chargenPanelInk(c color.RGBA) bool {
	for _, ink := range chargenPanelInks() {
		if inkShade(c, ink) {
			return true
		}
	}
	return false
}

func TestReleaseChargenCommandPanelIsTheShopComposition(t *testing.T) {
	f := releaseFront(t)
	setup := f.ChargenSetup()
	if setup.PreCreate == nil || setup.PreCreate.Art == nil || setup.PreCreate.Art.NavArt == nil {
		t.Fatal("production chargen art did not resolve the panel body")
	}
	art := setup.PreCreate.Art
	l := f.generator()
	rects := map[string]image.Rectangle{}
	for i, cmd := range l.Detail.Commands {
		rects[cmd.Role] = cmd.Rect.Rectangle()
		if art.NavButtons[i][0] != nil || art.NavButtons[i][1] == nil {
			t.Errorf("command %s: off %v, on %v; want only the pressed picture", cmd.Role, art.NavButtons[i][0] != nil, art.NavButtons[i][1] != nil)
		}
	}
	for i, role := range chargenPanelOrder {
		if got, want := rects[role], ui.ShopButtonRect(i); got != want {
			t.Errorf("%s at %v, want the shop's plaque %d %v", role, got, i, want)
		}
	}
	if b := art.NavArt.Bounds(); b.Dx() != ui.TownWideUpperRegion.Dx() || b.Dy() != ui.TownWideUpperRegion.Dy() {
		t.Fatalf("panel body %v, want the shop menu's %v", b.Size(), ui.TownWideUpperRegion.Size())
	}

	// At rest every opaque body pixel is the body's, or a caption inside a
	// plaque.
	c := ui.NewChargen(setup)
	c.SelectPreChoice(0)
	c.Forward()
	if c.Stage() != ui.DetailedStage {
		t.Fatal("Forward did not reach the detail page")
	}
	frame := ui.ComposeChargenFrame(c)
	region := ui.TownWideUpperRegion
	in := func(p image.Point) bool {
		for _, r := range rects {
			if p.In(r) {
				return true
			}
		}
		return false
	}
	bad := 0
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			p := image.Pt(x, y)
			want := color.RGBAModel.Convert(art.NavArt.At(x-region.Min.X, y-region.Min.Y)).(color.RGBA)
			got := frame.RGBAAt(x, y)
			if want.A == 0 || got == want || in(p) && chargenPanelInk(got) {
				continue
			}
			if bad++; bad <= 5 {
				t.Errorf("rest pixel %v = %v, body %v", p, got, want)
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d rest pixels are neither the body nor a caption", bad)
	}

	// Held, a plaque shows its own bitmap over the body.
	a, model, begun := chargenPanelApp(t, f)
	for i, role := range chargenPanelOrder {
		r := ui.ShopButtonRect(i)
		mid := r.Min.Add(r.Size().Div(2))
		chargenPanelPointer(t, a, "hover", mid)
		chargenPanelPointer(t, a, "press", mid)
		held, _, err := a.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		on := art.NavButtons[slices.IndexFunc(l.Detail.Commands, func(c ui.GeneratorCommand) bool { return c.Role == role })][1]
		bad := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				want := color.RGBAModel.Convert(on.At(x-r.Min.X, y-r.Min.Y)).(color.RGBA)
				got := held.RGBAAt(x, y)
				if want.A == 0 || got == want || chargenPanelInk(got) {
					continue
				}
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("%s held: %d pixels are neither its bitmap nor its caption", role, bad)
		}
		chargenPanelPointer(t, a, "move", image.Pt(320, 20))
		chargenPanelPointer(t, a, "release", image.Pt(320, 20))
	}

	// Each press does its command.
	click := func(i int) {
		r := ui.ShopButtonRect(i)
		mid := r.Min.Add(r.Size().Div(2))
		for _, action := range []string{"hover", "press", "release"} {
			chargenPanelPointer(t, a, action, mid)
		}
	}
	preset, _ := model.Result()
	presetStats := slices.Clone(preset.Stats)
	click(2) // Reset
	if r, _ := model.Result(); !slices.Equal(r.Stats, []int{25, 25, 25, 25}) {
		t.Errorf("Reset gave %v, want 25 each", r.Stats)
	}
	click(1) // Restore
	if r, _ := model.Result(); !slices.Equal(r.Stats, presetStats) {
		t.Errorf("Restore gave %v, want the preset %v", r.Stats, presetStats)
	}
	click(3) // Back
	if s, _ := a.HeadlessChargenState(); s.Stage != ui.ChargenStagePreCreate {
		t.Fatalf("Back left the page at %q", s.Stage)
	}
	pre := chargenTraceMaskPoints(t, f, l.PreCreate.Mask.Key, image.Point{}, []byte{180})
	secondGeneratorClick(t, a, pre[180])
	click(0) // Accept
	if a.Screen() == ui.ScreenChargen || !*begun {
		t.Fatalf("Accept left screen %v, begun %t, message %q", a.Screen(), *begun, a.HeadlessMessage())
	}
}

// chargenPanelApp opens the generator through the App with a typed name and
// stands on the first hero's detail page.
func chargenPanelApp(t *testing.T, f *FrontEnd) (*ui.App, *ui.Chargen, *bool) {
	t.Helper()
	a := f.App("chargen panel")
	a.Layout(640, 480)
	begun := new(bool)
	var model *ui.Chargen
	a.SetNewGameChargen(func() *ui.ChargenEntry {
		model = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: model, Begin: func(res ui.ChargenResult) (ui.MapOpener, error) {
			*begun = true
			return f.NewGameOpener(10, res), nil
		}}
	})
	if err := a.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	secondGeneratorClick(t, a, image.Pt(290, 322))
	if err := a.HeadlessType("Panel", false); err != nil {
		t.Fatal(err)
	}
	pre := chargenTraceMaskPoints(t, f, f.generator().PreCreate.Mask.Key, image.Point{}, []byte{180})
	secondGeneratorClick(t, a, pre[180])
	if s, _ := a.HeadlessChargenState(); s.Stage != ui.ChargenStageDetailed {
		t.Fatalf("forward stood at %q", s.Stage)
	}
	return a, model, begun
}

func chargenPanelPointer(t *testing.T, a *ui.App, action string, p image.Point) {
	t.Helper()
	if err := a.HeadlessPointer(action, p.X, p.Y); err != nil {
		t.Fatal(err)
	}
	if a.Screen() == ui.ScreenChargen {
		if _, _, err := a.HeadlessFrame(); err != nil {
			t.Fatal(err)
		}
	}
}
