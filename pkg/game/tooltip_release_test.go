package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/internal/witness"
	"againrom/pkg/ui"
)

func releaseTooltipMenuWitness(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	before := f.live.world.Hash()
	for _, ms := range []int{0, 100, 200, 300, 400, 500} {
		// Delay control in the same 640x480 panel as the speed witness. A
		// click changes the dialog's copy; OK applies it.
		prior := app.TooltipDelay()
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, 205, 246); err != nil {
				t.Fatal(err)
			}
		}
		if app.TooltipDelay() != prior {
			t.Fatalf("menu click applied delay%d before OK", app.TooltipDelay())
		}
		label := app.HeadlessRows()[8].Text
		for _, action := range []string{"page-return", "game-options"} {
			if err := app.HeadlessGameMenuAction(action); err != nil {
				t.Fatal(err)
			}
		}
		if app.TooltipDelay() != ms {
			t.Fatalf("menu click: delay%d want%d", app.TooltipDelay(), ms)
		}
		if !strings.Contains(label, fmt.Sprint(ms)) || f.Font.Value().Advance(label) > 446 {
			t.Fatal("delay row clipped", label)
		}
		cold := releaseFront(t)
		cold.Options = OptionsStore{Path: f.Options.Path}
		if got := cold.App("cold tooltip preferences").TooltipDelay(); got != ms {
			t.Fatal("fresh installed frontend lost delay", got, ms)
		}
	}
	if f.live.world.Hash() != before {
		t.Fatal("tooltip setting moved paused simulation")
	}
}

func TestReleaseTooltipShopAndChargenUseInstalledTextAndSharedDelay(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.SetTipsOff(true)
	f.Carried = MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	f.Town = NewTown(f.Campaign.Value())
	f.Shop = NewShop(5000)
	f.Shop.Generate(f.Table, 1182)
	s := f.TownScreen().(*townScreen)
	s.room = roomShop
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfPick, Index: roomArmour})
	a := f.App("installed tooltip")
	a.Layout(640, 480)
	a.SetTown(s)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town", Label: "town"}} }, func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("town"); err != nil {
		t.Fatal(err)
	}
	x, y, err := a.HeadlessShopPoint("shelf", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	before, _, err := a.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame})
	if err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick <= 25; tick++ {
		if tick > 0 {
			if err = a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		pic, state, _ := a.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartTooltip})
		if state.Tooltip.Visible != (tick == 25) || state.Tooltip.Target == "" {
			t.Fatal("shop delay boundary", tick, state.Tooltip)
		}
		if state.Tooltip.Visible && (pic == nil || !state.Tooltip.Bounds.In(image.Rect(0, 0, 640, 480))) {
			t.Fatal("installed item popup escaped frame", state.Tooltip)
		}
		if state.Tooltip.Visible && (state.Tooltip.Bounds.Min.Y != max(0, y-pic.Bounds().Dy()) || state.Tooltip.Bounds.Min.X != x) {
			t.Fatal("installed shelf tooltip lost its lower-left anchor or top-edge clamp", x, y, state.Tooltip)
		}
	}
	after, _, err := a.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame})
	if err != nil || bytes.Equal(before.Pix, after.Pix) {
		t.Fatal("shop did not paint delayed installed popup", err)
	}
	tooltipReleasePNG(t, "shop", after)
	// A fresh production generator gives long authored stat descriptions,
	// class descriptions and skill icons, all through the same resolver.
	g := f.App("installed generator hover")
	g.Layout(640, 480)
	g.SetTooltipDelayPreference(0, nil)
	if err = g.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if err = g.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, stage := range []string{ui.ChargenStagePreCreate, ui.ChargenStageDetailed} {
		state, ok := g.HeadlessChargenState()
		if !ok || state.Stage != stage {
			t.Fatal("generator stage", state)
		}
		for py := 8; py < 480; py += 16 {
			for px := 8; px < 640; px += 16 {
				if err = g.HeadlessPointer("hover", px, py); err != nil {
					t.Fatal(err)
				}
				pic, h, _ := g.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartTooltip})
				if h.Tooltip.Target == "" || seen[h.Tooltip.Target] {
					continue
				}
				if !h.Tooltip.Visible || pic == nil || !h.Tooltip.Bounds.In(image.Rect(0, 0, 640, 480)) {
					t.Fatal("installed description escaped frame or missed zero delay", h.Tooltip)
				}
				seen[h.Tooltip.Target] = true
				if len(seen) <= 35 {
					frame, _, e := g.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame})
					if e != nil {
						t.Fatal(e)
					}
					tooltipReleasePNG(t, fmt.Sprintf("%s-%02d", stage, len(seen)), frame)
				}
			}
		}
		if stage == ui.ChargenStagePreCreate {
			for n := 0; n < 20; n++ {
				state, _ = g.HeadlessChargenState()
				control, found := state.Control(ui.ChargenControlForward, "")
				if !found {
					t.Fatal("no forward")
				}
				if state.Focus == control.Focus {
					break
				}
				if err = g.HeadlessKey("down"); err != nil {
					t.Fatal(err)
				}
			}
			if err = g.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(seen) < 24 {
		t.Fatal("insufficient installed generator tooltip population", len(seen))
	}
	t.Logf("installed item delay boundary and %d generator text targets compose inside frame", len(seen))
}

func tooltipReleasePNG(t *testing.T, name string, pic *image.RGBA) {
	t.Helper()
	dir := os.Getenv("AGAINROM_TOOLTIP_ARTIFACTS")
	if dir == "" {
		return
	}
	dir = filepath.Join(dir, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	witness.PNG(t, filepath.Join(dir, name+".png"), pic)
}
