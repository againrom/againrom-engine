package game

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			if err := app.HeadlessPointer(edge, 221, 282); err != nil {
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
	before, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick <= 25; tick++ {
		if tick > 0 {
			if err = a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		state, pic := a.HeadlessTooltip()
		if state.Visible != (tick == 25) || state.Target == "" {
			t.Fatal("shop delay boundary", tick, state)
		}
		if state.Visible && (pic == nil || !state.Bounds.In(image.Rect(0, 0, 640, 480))) {
			t.Fatal("installed item popup escaped frame", state)
		}
		if state.Visible && (state.Bounds.Min.Y != max(0, y-pic.Bounds().Dy()) || state.Bounds.Min.X != x) {
			t.Fatal("installed shelf tooltip lost its lower-left anchor or top-edge clamp", x, y, state)
		}
	}
	after, _, err := a.HeadlessFrame()
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
				h, pic := g.HeadlessTooltip()
				if h.Target == "" || seen[h.Target] {
					continue
				}
				if !h.Visible || pic == nil || !h.Bounds.In(image.Rect(0, 0, 640, 480)) {
					t.Fatal("installed description escaped frame or missed zero delay", h)
				}
				seen[h.Target] = true
				if len(seen) <= 35 {
					frame, _, e := g.HeadlessFrame()
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
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, pic)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
}
