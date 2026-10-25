package game

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"againrom/pkg/ui"
)

// secondPanelSame counts the pixels of r in frame equal to pic laid at r's
// corner, and the pixels compared.
func secondPanelSame(frame *image.RGBA, pic image.Image, r image.Rectangle, skip func(image.Point) bool) (same, all int) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			p := image.Pt(x, y)
			if skip != nil && skip(p) {
				continue
			}
			all++
			at := pic.Bounds().Min.Add(p.Sub(r.Min))
			if color.RGBAModel.Convert(pic.At(at.X, at.Y)).(color.RGBA) == frame.RGBAAt(x, y) {
				same++
			}
		}
	}
	return same, all
}

// The second game's detail page draws its commands on the hero-creation
// folder's own four-plaque brown panel (R2-ENGINE-286, R2-ASSET-090): the
// panel copied at (480,0), Accept, Reset, Back and Restore top to bottom in
// its four recesses (DIV-3014). The points the original was clicked at,
// (533,65), (533,112) and (533,158), are Accept, Reset and Back; a held
// plaque shows its pressed picture; Reset keeps its rule per language,
// Restore returns the opening draft and Back leaves the page.
func TestReleaseSecondGameGeneratorCommandPanelIsTheFourPlaquePanel(t *testing.T) {
	f := secondGameFront(t)
	if f.ChargenAssets == nil || f.ChargenAssets.Presentation == nil {
		t.Fatal("the second game's install loaded no generator")
	}
	f.Options = OptionsStore{}
	app := f.App("second generator panel")
	app.Layout(640, 480)
	var c *ui.Chargen
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		c = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: c, Begin: f.NewGameBegin(f.Base().Profile.Mission())}
	})
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenChargen || c == nil {
		t.Fatalf("NEW GAME: screen %s, error %v", app.Screen(), err)
	}
	l := f.generator()
	ok := byte(l.PreCreate.Forward.Mask)
	pre := chargenTraceMaskPoints(t, f, l.PreCreate.Mask.Key, image.Point{}, []byte{ok})
	secondGeneratorClick(t, app, pre[ok])
	if secondGeneratorStage(app) != ui.ChargenStageDetailed {
		t.Fatal("OK did not open the detail page")
	}
	opening, _ := c.Result()
	pool := c.Remaining()
	art := f.ChargenAssets.Presentation
	rects := map[string]image.Rectangle{}
	for _, cmd := range l.Detail.Commands {
		rects[cmd.Role] = cmd.Rect.Rectangle()
	}
	frameAt := func(action string, p image.Point) *image.RGBA {
		t.Helper()
		if err := app.HeadlessPointer(action, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		frame, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		return frame
	}

	t.Run("panel at rest", func(t *testing.T) {
		frame := frameAt("move", image.Pt(320, 470))
		inPlaque := func(p image.Point) bool {
			for _, r := range rects {
				if p.In(r) {
					return true
				}
			}
			return false
		}
		if same, all := secondPanelSame(frame, art.NavArt, l.Detail.Nav.Rect.Rectangle(), inPlaque); same != all {
			t.Errorf("panel body: %d of %d pixels are ButtonsArea's", same, all)
		}
		for _, cmd := range l.Detail.Commands {
			r := rects[cmd.Role]
			shift := art.NavArt.Bounds().Min.Sub(l.Detail.Nav.Rect.Rectangle().Min)
			same, all := 0, 0
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					at := image.Pt(x, y).Add(shift)
					all++
					if color.RGBAModel.Convert(art.NavArt.At(at.X, at.Y)).(color.RGBA) == frame.RGBAAt(x, y) {
						same++
					}
				}
			}
			if same*10 < all*8 {
				t.Errorf("%s at rest: %d of %d pixels are the panel's recess", cmd.Role, same, all)
			}
		}
	})
	t.Run("the original's points", func(t *testing.T) {
		restore := rects["restore"]
		for p, want := range map[image.Point]string{{533, 65}: "play", {533, 112}: "reset", {533, 158}: "back",
			restore.Min.Add(restore.Size().Div(2)): "restore"} {
			if got, _ := ui.DetailedControlAt(c, p); got != want {
				t.Errorf("%v is %q, want %q", p, got, want)
			}
		}
	})
	t.Run("held Accept shows its on picture", func(t *testing.T) {
		frameAt("move", image.Pt(533, 65))
		frame := frameAt("press", image.Pt(533, 65))
		pressed := art.NavButtons[0][1]
		key := color.RGBAModel.Convert(pressed.At(pressed.Bounds().Min.X, pressed.Bounds().Min.Y))
		transparent := func(p image.Point) bool {
			return color.RGBAModel.Convert(pressed.At(pressed.Bounds().Min.X+p.X-rects["play"].Min.X, pressed.Bounds().Min.Y+p.Y-rects["play"].Min.Y)) == key
		}
		if same, all := secondPanelSame(frame, pressed, rects["play"], transparent); all == 0 || same*10 < all*8 {
			t.Errorf("held Accept: %d of %d pixels are its pressed picture", same, all)
		}
		frameAt("move", image.Pt(320, 470))
		frameAt("release", image.Pt(320, 470))
		if secondGeneratorStage(app) != ui.ChargenStageDetailed {
			t.Fatal("a release off Accept left the detail page")
		}
	})
	t.Run("Reset, Restore and Back at the original's points", func(t *testing.T) {
		_, _, lower, _, _ := ui.DetailedAttributeBoxes(l, 3)
		_, _, _, raise, _ := ui.DetailedAttributeBoxes(l, 2)
		secondGeneratorClick(t, app, lower.Min.Add(lower.Size().Div(2)))
		secondGeneratorClick(t, app, raise.Min.Add(raise.Size().Div(2)))
		if r, _ := c.Result(); slices.Equal(r.Stats, opening.Stats) {
			t.Fatal("the draft did not change")
		}
		secondGeneratorClick(t, app, image.Pt(533, 112))
		r, _ := c.Result()
		wantReset, wantFree := []int{25, 25, 25, 25}, 100
		if f.Base().Profile.Language == "russian" {
			wantReset, wantFree = opening.Stats, pool
		}
		if !slices.Equal(r.Stats, wantReset) || c.Remaining() != wantFree {
			t.Errorf("Reset at (533,112) gave %v, %d free; want %v, %d free", r.Stats, c.Remaining(), wantReset, wantFree)
		}
		restore := rects["restore"]
		secondGeneratorClick(t, app, restore.Min.Add(restore.Size().Div(2)))
		if r, _ := c.Result(); !slices.Equal(r.Stats, opening.Stats) || !slices.Equal(r.Choices, opening.Choices) || c.Remaining() != pool {
			t.Errorf("Restore gave %+v, %d free; want %+v, %d free", r, c.Remaining(), opening, pool)
		}
		secondGeneratorClick(t, app, image.Pt(533, 158))
		if s := secondGeneratorStage(app); s != ui.ChargenStagePreCreate {
			t.Errorf("Back at (533,158) left the page on %q, want pre-create", s)
		}
	})
}
