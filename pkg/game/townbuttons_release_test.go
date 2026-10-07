package game

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// TestReleaseSchoolAndTavernButtonArtLoads is 1017's install-gated witness
// that LoadTownSchoolArt and LoadTownTavernArt resolve every shipped button
// bitmap this story added, against a real install rather than a synthetic
// fixture (schoolmask_release_test.go's own pattern; golden rule 2 forbids
// this from a plain go test).
func TestReleaseSchoolAndTavernButtonArtLoads(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil {
		t.Fatalf("production school art did not resolve: %v", f.TownSchoolArt.Err())
	}
	for i := range f.TownSchoolArt.Value().Buttons {
		for state, name := range []string{"off", "on"} {
			if f.TownSchoolArt.Value().Buttons[i][state] == nil {
				t.Errorf("school button %d state %s did not load", i, name)
			}
		}
	}
	if f.TownTavernArt.Value() == nil {
		t.Fatalf("production tavern art did not resolve: %v", f.TownTavernArt.Err())
	}
	if f.TownTavernArt.Value().Upper == nil {
		t.Error("tavern area picture did not load")
	}
	for i := range f.TownTavernArt.Value().Buttons {
		for state, name := range []string{"off", "on"} {
			if f.TownTavernArt.Value().Buttons[i][state] == nil {
				t.Errorf("tavern button %d state %s did not load", i, name)
			}
		}
	}
	if f.TownTavernArt.Value().CommandUpper == nil {
		t.Error("tavern four-command panel body did not load")
	}
	wantSizes := [4]image.Point{{120, 52}, {140, 46}, {140, 46}, {120, 52}}
	for i, button := range f.TownTavernArt.Value().CommandButtons {
		if button == nil {
			t.Errorf("tavern four-command button %d did not load", i)
			continue
		}
		if got := button.Bounds().Size(); got != wantSizes[i] {
			t.Errorf("tavern four-command button %d size = %v, want %v", i, got, wantSizes[i])
		}
	}
}

// TestReleaseSchoolButtonPressShowsItsOwnOnState checks the school's own
// native pressed pair.
func TestReleaseSchoolButtonPressShowsItsOwnOnState(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil || f.TownTavernArt.Value() == nil {
		t.Fatalf("production surface art did not resolve: school %v tavern %v",
			f.TownSchoolArt.Err(), f.TownTavernArt.Err())
	}
	cases := []struct {
		name string
		kind ui.TownSurfaceKind
		v    ui.TownSurfaceView
	}{
		{"school", ui.TownSurfaceSchool, ui.TownSurfaceView{
			Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(), SchoolClass: -1,
			Buttons:   []ui.TownSurfaceButton{{Label: "Train", Enabled: true}, {Label: "EXIT", Enabled: true}},
			HoverCell: -1, Font: f.Font.Value(),
		}},
	}
	for _, c := range cases {
		for i := range c.v.Buttons {
			released := c.v
			released.Press = ui.TownSurfaceControl{}
			pressed := c.v
			pressed.Press = ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: i}

			off := ui.ComposeTownSurface(released)
			on := ui.ComposeTownSurface(pressed)
			well := ui.TownSurfaceButtonRect(c.kind, i)

			differ := 0
			for y := well.Min.Y; y < well.Max.Y; y++ {
				for x := well.Min.X; x < well.Max.X; x++ {
					if off.RGBAAt(x, y) != on.RGBAAt(x, y) {
						differ++
					}
				}
			}
			if differ == 0 {
				t.Errorf("%s button %d: pressed art is pixel-identical to released art over its own well %v",
					c.name, i, well)
			}
		}
	}
}

// Literal rest and disabled inks guard label and value placement.
var (
	townButtonInkText     = color.RGBA{R: 200, G: 184, B: 144, A: 255}
	townButtonInkDisabled = color.RGBA{R: 0x68, G: 0x62, B: 0x59, A: 0xff}
	townButtonCanvasFill  = color.RGBA{R: 0x0d, G: 0x0e, B: 0x13, A: 0xff}
)

// oracleTownButtonText reproduces townShellTextLayout and drawTownShellText
// (townshell.go:429-449) by re-deriving the same centred, tail-truncated
// placement from the exported font primitives, rather than by calling
// either: both are unexported, and an oracle built by calling the code
// under test cannot disagree with it.
func oracleTownButtonText(dst *image.RGBA, font *text.Font, s string, r image.Rectangle, c color.RGBA) {
	if font == nil || s == "" {
		return
	}
	for {
		w, _ := font.Measure(s)
		if w <= r.Dx()-6 || len(s) <= 1 {
			break
		}
		s = s[:len(s)-1]
	}
	w, h := font.Measure(s)
	font.Draw(dst, s, r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-h)/2, c)
}

// oracleTownButtonWell independently composes one complete button well. A
// nil panel is the school's opaque off/on plaque path. A non-nil panel is the
// tavern's shop body; its native command bitmap is drawn over it only when
// art is non-nil, which is the pressed-and-hovered state.
func oracleTownButtonWell(panel image.Image, panelAt image.Point, art image.Image, well image.Rectangle, font *text.Font, b ui.TownSurfaceButton, pressed bool) *image.RGBA {
	dst := image.NewRGBA(well)
	draw.Draw(dst, well, &image.Uniform{C: townButtonCanvasFill}, image.Point{}, draw.Src)
	op := draw.Src
	if panel != nil {
		src := panel.Bounds().Min.Add(well.Min.Sub(panelAt))
		draw.Draw(dst, well, panel, src, draw.Src)
		op = draw.Over
	}
	if art != nil {
		draw.Draw(dst, well, art, art.Bounds().Min, op)
	}
	ink := townButtonInkText
	if !b.Enabled {
		ink = townButtonInkDisabled
	}
	label := image.Rect(well.Min.X, well.Min.Y+2, well.Max.X, well.Min.Y+24)
	value := image.Rect(well.Min.X, well.Min.Y+23, well.Max.X, well.Max.Y-2)
	if pressed && b.Enabled {
		label, value = label.Add(image.Pt(0, 1)), value.Add(image.Pt(0, 1))
	}
	oracleTownButtonText(dst, font, b.Label, label, ink)
	oracleTownButtonText(dst, font, b.Value, value, ink)
	return dst
}

// diffWells reports every point in well where a and b disagree, capped at 5
// examples, alongside the total count.
func diffWells(a, b *image.RGBA, well image.Rectangle) (n int, examples []image.Point) {
	for y := well.Min.Y; y < well.Max.Y; y++ {
		for x := well.Min.X; x < well.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
				if len(examples) < 5 {
					examples = append(examples, image.Pt(x, y))
				}
			}
		}
	}
	return n, examples
}

// TestReleaseSchoolAndTavernButtonArtIsPlacedAndLabelled is the placement
// and labelling half of the same witness (townsquare_release_test.go's own
// byte-exact pattern for placement). It replaces a byte-exact well
// comparison this story's round-1 review found: the shipped
// b{1,2}{off,on}.bmp and button{1,2,3}{off,on}.bmp plaques carry no baked
// text (DIV-159), so a byte-exact composed-well-equals-raw-bitmap assertion
// positively forbids a label from ever being drawn and would pass over that
// defect as readily as over the fix.
//
// ROUND 3 REBUILD (W-1). The round-2 version unioned the label and value
// rectangles under one counter and accepted any single differing pixel
// inside the union as "a label was drawn"; production is correct, but seven
// independent mutations to townshell.go's label/value drawing (moved,
// swapped, wrong colour, truncated to one character, disabled styling
// removed, no text drawn at all, drawn over the wrong well) all left it
// green. This version instead builds an independent pixel-exact oracle
// (oracleTownButtonWell, above) for the whole well and requires the composed
// well to equal it everywhere: placement, content, position and ink are all
// one assertion rather than five heuristics, and any of the seven mutations
// changes some pixel the oracle does not.
//
// One button per room (EXIT) is Enabled: false, so the disabled ink path is
// exercised; a fixture where every button were enabled could never catch a
// removal of the disabled styling.
func TestReleaseSchoolAndTavernButtonArtIsPlacedAndLabelled(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil || f.TownTavernArt.Value() == nil {
		t.Fatalf("production surface art did not resolve: school %v tavern %v",
			f.TownSchoolArt.Err(), f.TownTavernArt.Err())
	}
	cases := []struct {
		name    string
		kind    ui.TownSurfaceKind
		buttons []ui.TownSurfaceButton
		v       ui.TownSurfaceView
		panel   image.Image
		panelAt image.Point
		art     func(i, state int) image.Image
	}{
		{"school", ui.TownSurfaceSchool, []ui.TownSurfaceButton{
			{Label: "Train", Value: "518", Enabled: true},
			{Label: "EXIT", Value: "100", Enabled: false},
		}, ui.TownSurfaceView{
			Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(), SchoolClass: -1,
			HoverCell: -1, Font: f.Font.Value(),
		}, nil, image.Point{}, func(i, state int) image.Image { return f.TownSchoolArt.Value().Buttons[i][state] }},
		{"tavern", ui.TownSurfaceTavern, []ui.TownSurfaceButton{
			{Label: "Sleep", Enabled: true},
			{Label: "Hire", Value: "250", Enabled: true},
			{Label: "Talk", Enabled: true},
			{Label: "EXIT", Value: "100", Enabled: false},
		}, ui.TownSurfaceView{
			Kind: ui.TownSurfaceTavern, TavernArt: f.TownTavernArt.Value(),
			HoverCell: -1, Font: f.Font.Value(),
		}, f.TownTavernArt.Value().CommandUpper, ui.TownWideUpperRegion.Min,
			func(i, state int) image.Image {
				if state == 0 {
					return nil
				}
				return f.TownTavernArt.Value().CommandButtons[i]
			}},
	}
	for _, c := range cases {
		c.v.Buttons = c.buttons
		n := len(c.buttons)
		composed := make([]*image.RGBA, n)
		wells := make([]image.Rectangle, n)
		for i := 0; i < n; i++ {
			composed[i] = ui.ComposeTownSurface(c.v)
			wells[i] = ui.TownSurfaceButtonRect(c.kind, i)
		}
		for i := 0; i < n; i++ {
			well := wells[i]
			oracle := oracleTownButtonWell(c.panel, c.panelAt, c.art(i, 0), well, f.Font.Value(), c.buttons[i], false)
			if diff, ex := diffWells(composed[i], oracle, well); diff != 0 {
				t.Errorf("%s button %d (%q/%q, enabled=%v): composed well %v differs from the oracle "+
					"render (art + own label/value at own ink) in %d pixel(s), first few at %v",
					c.name, i, c.buttons[i].Label, c.buttons[i].Value, c.buttons[i].Enabled, well, diff, ex)
			}
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				if wells[i].Dx() != wells[j].Dx() || wells[i].Dy() != wells[j].Dy() {
					continue
				}
				same := true
				for y := 0; y < wells[i].Dy() && same; y++ {
					for x := 0; x < wells[i].Dx(); x++ {
						a := composed[i].RGBAAt(wells[i].Min.X+x, wells[i].Min.Y+y)
						b := composed[j].RGBAAt(wells[j].Min.X+x, wells[j].Min.Y+y)
						if a != b {
							same = false
							break
						}
					}
				}
				if same {
					t.Errorf("%s buttons %d and %d: pixel-identical over their own wells %v and %v",
						c.name, i, j, wells[i], wells[j])
				}
			}
		}
	}
}

// TestReleaseSchoolButtonPressedArtIsPlacedAndLabelled is the pressed-state
// half for the one panel that ships ON bitmaps.
func TestReleaseSchoolButtonPressedArtIsPlacedAndLabelled(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil {
		t.Fatalf("production school art did not resolve: %v", f.TownSchoolArt.Err())
	}
	buttons := []ui.TownSurfaceButton{
		{Label: "Train", Value: "518", Enabled: true},
		{Label: "EXIT", Value: "100", Enabled: false},
	}
	base := ui.TownSurfaceView{Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(),
		SchoolClass: -1, HoverCell: -1, Font: f.Font.Value(), Buttons: buttons}
	for i := range buttons {
		v := base
		v.Press = ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: i}
		composed := ui.ComposeTownSurface(v)
		well := ui.TownSurfaceButtonRect(ui.TownSurfaceSchool, i)
		oracle := oracleTownButtonWell(nil, image.Point{}, f.TownSchoolArt.Value().Buttons[i][1], well, f.Font.Value(), buttons[i], true)
		if diff, ex := diffWells(composed, oracle, well); diff != 0 {
			t.Errorf("school button %d pressed (%q/%q, enabled=%v): composed well %v differs from the "+
				"oracle render (ON art + own label/value at own ink) in %d pixel(s), first few at %v",
				i, buttons[i].Label, buttons[i].Value, buttons[i].Enabled, well, diff, ex)
		}
	}
}

// TestReleaseSchoolAndTavernUpperRegionMatchesShippedArtOutsideButtonWells
// guards the ornaments the buttons must not cover. School retains its
// 160-wide body plus keyed seam. Tavern uses the shop's complete 176-wide
// body; its four native rectangles are the only excluded pixels.
func TestReleaseSchoolAndTavernUpperRegionMatchesShippedArtOutsideButtonWells(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil || f.TownTavernArt.Value() == nil {
		t.Fatalf("production surface art did not resolve: school %v tavern %v",
			f.TownSchoolArt.Err(), f.TownTavernArt.Err())
	}
	compareOutside := func(t *testing.T, got, want *image.RGBA, kind ui.TownSurfaceKind, count int) {
		t.Helper()
		diff := 0
		var examples []image.Point
		for y := ui.TownWideUpperRegion.Min.Y; y < ui.TownWideUpperRegion.Max.Y; y++ {
			for x := ui.TownWideUpperRegion.Min.X; x < ui.TownWideUpperRegion.Max.X; x++ {
				p := image.Pt(x, y)
				inside := false
				for i := 0; i < count; i++ {
					if p.In(ui.TownSurfaceButtonRect(kind, i)) {
						inside = true
						break
					}
				}
				if !inside && got.RGBAAt(x, y) != want.RGBAAt(x, y) {
					diff++
					if len(examples) < 5 {
						examples = append(examples, p)
					}
				}
			}
		}
		if diff != 0 {
			t.Fatalf("%d panel pixel(s) outside button rectangles differ; first few %v", diff, examples)
		}
	}

	t.Run("school", func(t *testing.T) {
		buttons := []ui.TownSurfaceButton{{Label: "Train", Value: "518", Enabled: true},
			{Label: "EXIT", Value: "100", Enabled: false}}
		v := ui.TownSurfaceView{Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(),
			SchoolClass: -1, Buttons: buttons, HoverCell: -1, Font: f.Font.Value()}
		artCopy := *f.TownSchoolArt.Value()
		artCopy.UpperSeam = nil
		beforeV := v
		beforeV.SchoolArt = &artCopy
		want := ui.ComposeTownSurface(beforeV)
		draw.Draw(want, ui.TownUpperRegion, f.TownSchoolArt.Value().Upper, f.TownSchoolArt.Value().Upper.Bounds().Min, draw.Src)
		draw.Draw(want, image.Rect(464, 0, 480, 238), f.TownSchoolArt.Value().UpperSeam,
			f.TownSchoolArt.Value().UpperSeam.Bounds().Min, draw.Over)
		compareOutside(t, ui.ComposeTownSurface(v), want, ui.TownSurfaceSchool, len(buttons))
	})

	t.Run("tavern shop composition", func(t *testing.T) {
		buttons := []ui.TownSurfaceButton{{Label: "Sleep", Enabled: true},
			{Label: "Hire", Value: "250", Enabled: true}, {Label: "Talk", Enabled: true},
			{Label: "EXIT", Value: "100", Enabled: false}}
		v := ui.TownSurfaceView{Kind: ui.TownSurfaceTavern, TavernArt: f.TownTavernArt.Value(),
			Buttons: buttons, HoverCell: -1, Font: f.Font.Value()}
		artCopy := *f.TownTavernArt.Value()
		artCopy.CommandUpper = image.NewRGBA(image.Rect(0, 0, 176, 238))
		artCopy.CommandButtons = [4]image.Image{}
		beforeV := v
		beforeV.TavernArt = &artCopy
		want := ui.ComposeTownSurface(beforeV)
		draw.Draw(want, ui.TownWideUpperRegion, f.TownTavernArt.Value().CommandUpper,
			f.TownTavernArt.Value().CommandUpper.Bounds().Min, draw.Over)
		compareOutside(t, ui.ComposeTownSurface(v), want, ui.TownSurfaceTavern, len(buttons))
	})
}
