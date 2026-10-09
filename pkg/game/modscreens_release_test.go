package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/menu"
	"againrom/pkg/ui"
)

// modTextColour is the letter colour of the town shell and of every modal text
// of the game; a mod screen draws its words in it.
var modTextColour = color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff}

// modScreenApp is an App on the lawful install, with the example mod's screens
// registered when withMod is set, started through the launcher's own steps.
func modScreenApp(t *testing.T, withMod bool) (*FrontEnd, *ui.App, mod.ScreenData) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	var screens mod.ScreenData
	if withMod {
		entries, err := mod.Resolve(modItemsDir, []string{"heavy-armor"})
		if err != nil {
			t.Fatal(err)
		}
		res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err := f.SetMods(res.Rules, res.Set, false); err != nil {
			t.Fatal(err)
		}
		if err := f.SetModItems(res.Items); err != nil {
			t.Fatal(err)
		}
		screens = res.Screens
	}
	app := f.App("mod screens")
	app.Layout(frame.W, frame.H)
	if err := app.SetModScreens(ModScreens(screens)); err != nil {
		t.Fatal(err)
	}
	return f, app, screens
}

func modFrame(t *testing.T, app *ui.App) *image.RGBA {
	t.Helper()
	pix, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatalf("frame of the %s screen: %v", app.Screen(), err)
	}
	return pix
}

// modTemplate paints s in the install font and the game's letter colour on an
// empty image; its non-zero pixels are the glyph pixels a draw of s must leave
// on a frame.
func modTemplate(t *testing.T, f *FrontEnd, s string) *image.RGBA {
	t.Helper()
	font := f.Font.Value()
	if font == nil {
		t.Fatal("the install has no font")
	}
	encoded := EncodeInstallText(s, font.Selector)
	w, h := font.Measure(encoded)
	img := image.NewRGBA(image.Rect(0, 0, w+1, h+1))
	font.Draw(img, encoded, 0, 0, modTextColour)
	return img
}

// modButtonTemplate is s as the shared push button draws its caption at
// rest: the grey (210,210,210) ramp, each pixel packed to RGB565 and expanded
// (MENU-115).
func modButtonTemplate(t *testing.T, f *FrontEnd, s string) *image.RGBA {
	t.Helper()
	font := f.Font.Value()
	if font == nil {
		t.Fatal("the install has no font")
	}
	encoded := EncodeInstallText(s, font.Selector)
	w, h := font.Measure(encoded)
	img := image.NewRGBA(image.Rect(0, 0, w+1, h+1))
	font.Draw(img, encoded, 0, 0, color.RGBA{210, 210, 210, 255})
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] == 0 {
			continue
		}
		r, g, b := img.Pix[i]>>3, img.Pix[i+1]>>2, img.Pix[i+2]>>3
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = uint8(int(r)*255/31), uint8(int(g)*255/63), uint8(int(b)*255/31), 255
	}
	return img
}

// findTemplate reports where inside region every glyph pixel of tmpl appears
// on pix with the colour the template gives it.
func findTemplate(pix, tmpl *image.RGBA, region image.Rectangle) (image.Point, bool) {
	var painted []image.Point
	for y := 0; y < tmpl.Bounds().Dy(); y++ {
		for x := 0; x < tmpl.Bounds().Dx(); x++ {
			if tmpl.RGBAAt(x, y).A != 0 {
				painted = append(painted, image.Pt(x, y))
			}
		}
	}
	if len(painted) == 0 {
		return image.Point{}, false
	}
	b := pix.Bounds()
	for oy := region.Min.Y; oy < region.Max.Y; oy++ {
	next:
		for ox := region.Min.X; ox < region.Max.X; ox++ {
			for _, p := range painted {
				q := image.Pt(ox+p.X, oy+p.Y)
				if !q.In(b) || pix.RGBAAt(q.X, q.Y) != tmpl.RGBAAt(p.X, p.Y) {
					continue next
				}
			}
			return image.Pt(ox, oy), true
		}
	}
	return image.Point{}, false
}

func requireTemplate(t *testing.T, what string, pix, tmpl *image.RGBA, region image.Rectangle) image.Point {
	t.Helper()
	at, ok := findTemplate(pix, tmpl, region)
	if !ok {
		t.Fatalf("%s is not drawn inside %v", what, region)
	}
	return at
}

func requireNoTemplate(t *testing.T, what string, pix, tmpl *image.RGBA, region image.Rectangle) {
	t.Helper()
	if at, ok := findTemplate(pix, tmpl, region); ok {
		t.Fatalf("%s is drawn at %v where it must not be", what, at)
	}
}

// diffBounds is the bounding box of the pixels two frames differ at, and whether
// they differ at all.
func diffBounds(a, b *image.RGBA) (image.Rectangle, bool) {
	var box image.Rectangle
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box, !box.Empty()
}

func writeModShot(t *testing.T, name string, pix *image.RGBA) {
	t.Helper()
	dir := os.Getenv("AGAINROM_SHOT_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	locale := strings.ReplaceAll(filepath.Base(os.Getenv("AGAINROM_ASSETS")), string(filepath.Separator), "_")
	out, err := os.Create(filepath.Join(dir, name+"-"+locale+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, pix); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func modPointer(t *testing.T, app *ui.App, x, y int) {
	t.Helper()
	for _, edge := range []string{"hover", "press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseModScreenSlotsAgreeWithTheMenus(t *testing.T) {
	if ui.MaxModMainEntries != mod.MaxMainMenuScreens || ui.MaxModGameEntries != mod.MaxGameMenuScreens {
		t.Fatalf("the menus have %d and %d slots, the mod loader allows %d and %d",
			ui.MaxModMainEntries, ui.MaxModGameEntries, mod.MaxMainMenuScreens, mod.MaxGameMenuScreens)
	}
}

// The unmodded main menu is exactly the brooch the install draws; with the mod
// it differs only in the bottom left corner, where the entry is drawn in the
// install font; the entry opens the screen, which shows the mod's words; Back
// and Escape return to a menu frame equal to the one before.
func TestReleaseModScreenMainMenuEntryOpensTheScreenAndBackReturns(t *testing.T) {
	f, app, screens := modScreenApp(t, true)
	screen := screens.Screens[0]
	_, plain, _ := modScreenApp(t, false)

	// Without the mod the menu frame is the installed brooch, pixel for pixel.
	want := f.Assets.Compose(menu.State{})
	got := modFrame(t, plain)
	if box, differs := diffBounds(got, want); differs {
		t.Fatalf("the unmodded menu differs from the installed brooch inside %v", box)
	}
	if err := plain.HeadlessActivate(screen.MenuLabel); err == nil {
		t.Fatal("an unmodded menu opened the mod entry")
	}

	// With the mod: the entry is the only change.
	menuFrame := modFrame(t, app)
	box, differs := diffBounds(menuFrame, want)
	if !differs {
		t.Fatal("the mod's entry is not drawn on the main menu")
	}
	if !box.In(image.Rect(0, 300, 260, 480)) {
		t.Fatalf("the entry changes pixels outside the bottom left corner: %v", box)
	}
	label := modButtonTemplate(t, f, screen.MenuLabel)
	requireTemplate(t, "the entry label", menuFrame, label, box)
	// Loss control: the same label is not on the unmodded menu.
	requireNoTemplate(t, "the entry label on the unmodded menu", got, label, image.Rect(0, 300, 260, 480))

	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("screen %s", app.Screen())
	}
	x, y, err := app.HeadlessModEntryPoint(0)
	if err != nil {
		t.Fatal(err)
	}
	modPointer(t, app, x, y)
	if app.Screen() != ui.ScreenMod {
		t.Fatalf("selecting the entry opened %s", app.Screen())
	}

	// The screen shows its title and each paragraph in the install font.
	page := modFrame(t, app)
	requireTemplate(t, "the title", page, modTemplate(t, f, screen.Title), image.Rect(96, 52, 544, 120))
	body := image.Rect(100, 110, 520, 360)
	for i, p := range screen.Paragraphs {
		words := strings.Join(strings.Fields(p)[:2], " ")
		requireTemplate(t, fmt.Sprintf("paragraph %d", i+1), page, modTemplate(t, f, words), body)
		// Loss control: the menu frame has none of it.
		requireNoTemplate(t, fmt.Sprintf("paragraph %d on the menu", i+1), menuFrame, modTemplate(t, f, words), body)
	}
	requireNoTemplate(t, "the title on the menu", menuFrame, modTemplate(t, f, screen.Title), image.Rect(96, 52, 544, 120))
	writeModShot(t, "mod-screen", page)
	writeModShot(t, "menu-with-entry", menuFrame)

	// Back returns to the menu frame, and so does Escape.
	bx, by, err := app.HeadlessModBackPoint()
	if err != nil {
		t.Fatal(err)
	}
	modPointer(t, app, bx, by)
	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("Back returned to %s", app.Screen())
	}
	if again := modFrame(t, app); !bytes.Equal(again.Pix, menuFrame.Pix) {
		t.Fatal("the menu frame after Back differs from the frame before the screen")
	}
	if err := app.HeadlessActivate(screen.MenuLabel); err != nil || app.Screen() != ui.ScreenMod {
		t.Fatalf("activate by label: %v on %s", err, app.Screen())
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMenu {
		t.Fatalf("Escape: %v on %s", err, app.Screen())
	}
	// The other menu buttons still work: Credits opens and Escape returns.
	if err := app.HeadlessActivate("credits"); err != nil || app.Screen() != ui.ScreenCredits {
		t.Fatalf("credits: %v on %s", err, app.Screen())
	}
}

// The in-game menu over a mission gains one row for the mod's screen, below the
// shipped seven. The row opens the screen with the world held; Escape returns to
// the menu with the row selected; the shipped rows are untouched.
func TestReleaseModScreenInGameMenuRowOpensTheScreen(t *testing.T) {
	f, app, screens := modScreenApp(t, true)
	screen := screens.Screens[0]
	g, plain, _ := modScreenApp(t, false)
	for _, c := range []struct {
		f   *FrontEnd
		app *ui.App
	}{{f, app}, {g, plain}} {
		party := c.f.ChargenParty(ui.ChargenResult{Name: "Mod screens", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
		if err := c.app.OpenMission(c.f.MissionOpenerWith(10, party)); err != nil {
			t.Fatalf("open mission 10: %v", err)
		}
		if err := c.app.HeadlessKey("escape"); err != nil || c.app.Screen() != ui.ScreenGameMenu {
			t.Fatalf("Escape in the mission: %v on %s", err, c.app.Screen())
		}
	}
	shipped, rows := plain.HeadlessRows(), app.HeadlessRows()
	if len(shipped) != 7 || len(rows) != 8 {
		t.Fatalf("the menu has %d rows unmodded and %d under the mod, want 7 and 8", len(shipped), len(rows))
	}
	for i := range shipped {
		if rows[i] != shipped[i] {
			t.Fatalf("shipped row %d changed: %+v vs %+v", i, rows[i], shipped[i])
		}
	}
	installLabel := EncodeInstallText(screen.MenuLabel, f.Font.Value().Selector)
	if rows[7].Text != installLabel || !rows[7].Choosable {
		t.Fatalf("the mod row is %+v, want %q", rows[7], installLabel)
	}

	// The panel differs from the unmodded panel only on the added row's rectangle
	// (panel top 60, first row 40 below, pitch 30: the eighth row is y 310..340).
	panel, base := app.GameMenuPanel(), plain.GameMenuPanel()
	if panel == nil || base == nil {
		t.Fatal("no game menu panel")
	}
	box, differs := diffBounds(panel, base)
	if !differs || box.Min.Y < 310 || box.Max.Y > 340 {
		t.Fatalf("the mod row changes pixels at %v (differs %v), want only the row at y 310..340", box, differs)
	}
	requireTemplate(t, "the row label", panel, modTemplate(t, f, screen.MenuLabel), box.Inset(-8))
	requireNoTemplate(t, "the row label on the unmodded panel", base, modTemplate(t, f, screen.MenuLabel), image.Rect(100, 100, 440, 400))
	shot := image.NewRGBA(panel.Bounds())
	draw.Draw(shot, shot.Bounds(), &image.Uniform{C: color.RGBA{A: 0xff}}, image.Point{}, draw.Src)
	draw.Draw(shot, shot.Bounds(), panel, image.Point{}, draw.Over)
	writeModShot(t, "game-menu-with-entry", shot)

	before := f.live.world.Hash()
	if err := app.HeadlessActivate(installLabel); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMod || app.HeadlessGameplayScreen() != ui.ScreenMap {
		t.Fatalf("screen %s, gameplay %s", app.Screen(), app.HeadlessGameplayScreen())
	}
	page := modFrame(t, app)
	requireTemplate(t, "the title", page, modTemplate(t, f, screen.Title), image.Rect(96, 52, 544, 120))
	for i := 0; i < 40; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Hash() != before {
		t.Fatal("the world advanced while a mod screen was open")
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenGameMenu {
		t.Fatalf("Escape from the mod screen: %v on %s", err, app.Screen())
	}
	if after := app.HeadlessRows(); len(after) != 8 || after[7] != rows[7] {
		t.Fatalf("the menu rows after returning: %+v", after)
	}
	if f.live.world.Hash() != before {
		t.Fatal("the world changed on returning to the menu")
	}
	// Loss control: the hold is the menu's, not the harness's. With the menu
	// closed the same steps advance the world.
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatalf("closing the menu: %v on %s", err, app.Screen())
	}
	for i := 0; i < 40; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Hash() == before {
		t.Fatal("the world does not advance with the menu closed, so the hold above proves nothing")
	}
}
