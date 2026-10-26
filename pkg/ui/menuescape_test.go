package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/render/menu"
)

// The second game's menu keeps Escape: the menu stays and asks nothing
// (s1.31), and the Hall of Fame it opens stays until its Ok (s1.33,
// R2-ENGINE-565). The first game's table still exits and leaves the hall.
func TestMenuEscapeFollowsTheMenuTable(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	hall := func() EndingView {
		return EndingView{HallAvailable: true, Hall: []EndingHallRow{{Name: "Diamond", Score: 70006}}}
	}
	for _, c := range []struct {
		name      string
		assets    *menu.Assets
		menuExits bool
		hallStays bool
	}{
		{"second", secondAppAssets(t), false, true},
		{"first", appAssets(t), true, false},
	} {
		a := NewApp("t", c.assets, appRows(1), okLoader(t))
		a.Layout(640, 480)
		a.SetHallOfFame(hall)
		if exit := a.step(appInput{Escape: true, AnyKey: true}, now); exit != c.menuExits || a.Screen() != ScreenMenu {
			t.Errorf("%s: Escape on the menu exits %v on %v, want exit %v", c.name, exit, a.Screen(), c.menuExits)
		}
		clickButton(a, buttonOf(t, c.assets, menu.ActionHallOfFame), now)
		if a.Screen() != ScreenEnding {
			t.Fatalf("%s: Hall of Fame opens %v", c.name, a.Screen())
		}
		a.step(appInput{Escape: true, AnyKey: true}, now)
		want := ScreenMenu
		if c.hallStays {
			want = ScreenEnding
		}
		if a.Screen() != want {
			t.Errorf("%s: Escape on the hall leaves %v, want %v", c.name, a.Screen(), want)
		}
	}
}

// After a return to the second game's menu the button under the resting
// pointer shows no caption until the pointer moves (s1.38, s1.40); the
// first game's menu selects it at once.
func TestMenuReturnHoldsTheCaptionUntilThePointerMoves(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, c := range []struct {
		name   string
		assets *menu.Assets
		held   bool
	}{
		{"second", secondAppAssets(t), true},
		{"first", appAssets(t), false},
	} {
		for _, action := range []menu.Action{menu.ActionCredits, menu.ActionCutscenes} {
			a := NewApp("t", c.assets, appRows(1), okLoader(t))
			a.Layout(640, 480)
			b := buttonOf(t, c.assets, action)
			clickButton(a, b, now)
			if a.Screen() == ScreenMenu {
				t.Fatalf("%s: button %d opened nothing", c.name, b)
			}
			x, y := centreOf(b)
			a.step(appInput{CursorX: x, CursorY: y, Escape: true, AnyKey: true}, now)
			for range 3 {
				a.step(appInput{CursorX: x, CursorY: y}, now)
			}
			if a.Screen() != ScreenMenu {
				t.Fatalf("%s: Escape from button %d leaves %v", c.name, b, a.Screen())
			}
			want := b
			if c.held {
				want = 0
			}
			if got := a.sel.State().Selected; got != want {
				t.Errorf("%s: after Escape from button %d the resting pointer selects %d, want %d", c.name, b, got, want)
			}
			a.step(appInput{CursorX: x + 1, CursorY: y}, now)
			if got := a.sel.State().Selected; got != b {
				t.Errorf("%s: the moved pointer selects %d, want %d", c.name, got, b)
			}
		}
	}
}

// A credits line that starts with a quote names a picture: it shows the
// loaded logo and never its own text (R2-ENGINE-442). A plain first-game line
// names a logo by its word.
func TestCreditsPictureLineShowsItsLogo(t *testing.T) {
	for line, want := range map[string]struct {
		name    string
		picture bool
	}{
		`"nival.bmp"`:     {"nival", true},
		` "Monolith.BMP"`: {"monolith", true},
		"Nival":           {"nival", false},
		"GAME DESIGN":     {"game design", false},
	} {
		if name, picture := CreditsLogoName(line); name != want.name || picture != want.picture {
			t.Errorf("%q names %q picture %v, want %q %v", line, name, picture, want.name, want.picture)
		}
	}
	a := newTestApp(t, appRows(1), okLoader(t))
	a.flow.menuFont = solidFont15()
	logo := image.NewRGBA(image.Rect(0, 0, 40, 10))
	for i := range logo.Pix {
		logo.Pix[i] = 0xff
	}
	a.SetCredits(func() CreditsView {
		return CreditsView{Lines: []string{`"nival.bmp"`, `"buka.bmp"`}, Logos: map[string]*image.RGBA{"nival": logo}}
	})
	a.openCredits(ScreenMenu)
	a.media.steps = creditsStart
	pix := a.composeCredits()
	if got := pix.RGBAAt(320, 5); got != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("the logo's middle reads %v, want its white", got)
	}
	// The unloaded buka line draws nothing at its row.
	pitch := a.creditsPitch()
	for y := pitch; y < 2*pitch; y++ {
		for x := 0; x < pix.Rect.Dx(); x++ {
			if c := pix.RGBAAt(x, y); c.R|c.G|c.B != 0 {
				t.Fatalf("the unloaded picture line draws at %d,%d", x, y)
			}
		}
	}
}
