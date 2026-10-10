package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// plaqueCensus hashes and optionally writes each recorded picture.
type plaqueCensus struct {
	out, lang string
	hashes    map[string]string
}

func (c *plaqueCensus) record(t *testing.T, name string, pic *image.RGBA) {
	t.Helper()
	if pic == nil || pic.Bounds().Empty() {
		t.Fatal("census picture is empty:", name)
	}
	if _, dup := c.hashes[name]; dup {
		t.Fatal("census name repeated:", name)
	}
	c.hashes[name] = fmt.Sprintf("%x", sha256.Sum256(pic.Pix))
	if c.out != "" {
		writeWidgetKitPNG(t, filepath.Join(c.out, c.lang, name+".png"), pic)
	}
}

func (c *plaqueCensus) frame(t *testing.T, app *ui.App, name string) {
	t.Helper()
	pic, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal(name, note, err)
	}
	c.record(t, name, pic)
}

func (c *plaqueCensus) pointer(t *testing.T, app *ui.App, name, action string, p image.Point) {
	t.Helper()
	if err := app.HeadlessPointer(action, p.X, p.Y); err != nil {
		t.Fatal(name, err)
	}
	c.frame(t, app, name)
}

// press records hover, press, out, back and a release outside.
func (c *plaqueCensus) press(t *testing.T, app *ui.App, name string, inside, outside image.Point) {
	t.Helper()
	c.pointer(t, app, name+"-hover", "hover", inside)
	c.pointer(t, app, name+"-press", "press", inside)
	c.pointer(t, app, name+"-press-out", "move", outside)
	c.pointer(t, app, name+"-press-back", "move", inside)
	c.pointer(t, app, name+"-release-out", "move", outside)
	c.pointer(t, app, name+"-released", "release", outside)
}

func centreOf(r image.Rectangle) image.Point { return r.Min.Add(r.Size().Div(2)) }

// TestReleasePlaqueButtonCensus hashes the plaque, stat, tip close and Save
// dialog button states App input reaches; AGAINROM_PLAQUE_CENSUS_DIR keeps
// PNGs and a hash list for comparing two builds.
func TestReleasePlaqueButtonCensus(t *testing.T) {
	f := releaseFront(t)
	root, err := editorPhysicalDirectory(f.Archives.Root)
	if err != nil {
		t.Fatal(err)
	}
	census := &plaqueCensus{lang: strings.ToLower(filepath.Base(root)), hashes: map[string]string{}}
	census.out = os.Getenv("AGAINROM_PLAQUE_CENSUS_DIR")
	if census.out != "" {
		rel, err := filepath.Rel(root, filepath.Clean(census.out))
		if !filepath.IsAbs(census.out) || err == nil && !strings.HasPrefix(rel, "..") {
			t.Fatal("census output must be absolute and outside the installed root")
		}
	}

	t.Run("generator", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil
		app := f.App("plaque census generator")
		t.Cleanup(app.StopAudio)
		app.SetCutscenes(nil)
		app.Layout(640, 480)
		c := ui.NewChargen(f.ChargenSetup())
		if err := app.OpenChargen(c, func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(10, res), nil }); err != nil {
			t.Fatal(err)
		}
		away := image.Pt(4, 4)
		census.pointer(t, app, "generator-precreate", "hover", away)
		if tip := c.TipPanel(); tip.Showing() {
			census.press(t, app, "generator-precreate-tip-close", centreOf(ui.TipPanelCloseRect(tip.Rect)), away)
		}
		c.CloseTip()
		c.Forward()
		census.pointer(t, app, "generator-detailed", "hover", away)
		if tip := c.TipPanel(); tip.Showing() {
			census.press(t, app, "generator-detailed-tip-close", centreOf(ui.TipPanelCloseRect(tip.Rect)), away)
			c.CloseTip()
			census.pointer(t, app, "generator-detailed-tip-closed", "hover", away)
		}
		for i, r := range ui.ChargenDetailedNavLabelRects(c) {
			if r.Empty() {
				t.Fatal("generator command", i, "has no rectangle")
			}
			census.press(t, app, fmt.Sprintf("generator-command%d", i), centreOf(r), away)
		}
		for stat := 0; stat < 4; stat++ {
			_, _, lower, raise, ok := ui.DetailedAttributeBoxes(stat)
			if !ok {
				t.Fatal("no attribute row", stat)
			}
			census.press(t, app, fmt.Sprintf("generator-stat%d-minus", stat), centreOf(lower), away)
			census.press(t, app, fmt.Sprintf("generator-stat%d-plus", stat), centreOf(raise), away)
		}
		// Freed points keep the raise enabled while held.
		_, _, lower, raise, _ := ui.DetailedAttributeBoxes(0)
		for i := 0; i < 2; i++ {
			census.pointer(t, app, fmt.Sprintf("generator-free%d-press", i), "press", centreOf(lower))
			census.pointer(t, app, fmt.Sprintf("generator-free%d-release", i), "release", centreOf(lower))
		}
		census.press(t, app, "generator-stat0-plus-held", centreOf(raise), away)
	})

	for _, room := range []struct {
		name, door string
		want       townRoom
	}{{"tavern", "TAVERN", roomTavern}, {"school", "SCHOOL", roomSchool}, {"shop", "SHOP", roomShop}} {
		t.Run(room.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			now := time.Unix(200, 0)
			f.TownAnimationNow = func() time.Time { return now }
			f.ShopRandom = func(int) int { return 0 }
			app, s := roomExitApp(t, f, 0)
			t.Cleanup(app.StopAudio)
			roomExitEnter(t, app, s, room.door, room.want)
			for n := 0; s.room == roomTalk && n < 64; n++ {
				if err := app.HeadlessActivate("dialogue"); err != nil {
					t.Fatal(err)
				}
			}
			if s.room != room.want {
				t.Fatalf("%s: room %d, want %d", room.name, s.room, room.want)
			}
			away := image.Pt(320, 2)
			census.pointer(t, app, room.name, "hover", away)
			var tip ui.TipPanelView
			if room.want == roomShop {
				tip = s.ShopScreen().TipPanel
			} else {
				tip = s.TownSurface().Tip
			}
			if tip.Showing() {
				close := centreOf(ui.TipPanelCloseRect(tip.Rect))
				census.press(t, app, room.name+"-tip-close", close, away)
				census.pointer(t, app, room.name+"-tip-close-again", "press", close)
				census.pointer(t, app, room.name+"-tip-closed", "release", close)
			}
			if room.want == roomShop {
				for i := 0; i < 4; i++ {
					x, y, err := app.HeadlessShopPoint("button", i)
					if err != nil {
						t.Logf("shop button %d: %v", i, err)
						continue
					}
					census.press(t, app, fmt.Sprintf("shop-button%d", i), image.Pt(x, y), away)
				}
				return
			}
			v := s.TownSurface()
			for i, b := range v.Buttons {
				at := centreOf(ui.TownSurfaceButtonRect(v.Kind, i))
				name := fmt.Sprintf("%s-button%d", room.name, i)
				if !b.Enabled {
					census.pointer(t, app, name+"-disabled-hover", "hover", at)
					census.pointer(t, app, name+"-disabled-press", "press", at)
					census.pointer(t, app, name+"-disabled-released", "release", away)
					continue
				}
				census.press(t, app, name, at, away)
			}
		})
	}

	t.Run("compose", func(t *testing.T) {
		f := releaseFront(t)
		s, ok := f.TownScreen().(*townScreen)
		if !ok {
			t.Fatal("TownScreen did not return a town screen")
		}
		for _, room := range []struct {
			name string
			door int
		}{{"tavern", 0}, {"school", 2}} {
			s.Choose(room.door)
			v := s.TownSurface()
			v.Tip = ui.TipPanelView{}
			census.record(t, "compose-"+room.name, ui.ComposeTownSurface(v))
			for i, b := range v.Buttons {
				h := v
				h.Hover, h.HasHover = centreOf(ui.TownSurfaceButtonRect(v.Kind, i)), true
				census.record(t, fmt.Sprintf("compose-%s-button%d-hover", room.name, i), ui.ComposeTownSurface(h))
				if !b.Enabled {
					continue
				}
				h.Press = ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: i}
				census.record(t, fmt.Sprintf("compose-%s-button%d-press", room.name, i), ui.ComposeTownSurface(h))
			}
			s.Back()
		}
		s.Choose(1)
		v := s.ShopScreen()
		v.TipPanel = ui.TipPanelView{}
		census.record(t, "compose-shop", ui.ComposeShopScreen(v, image.Point{}, false, nil, false))
		points := []image.Point{{554, 41}, {553, 90}, {553, 137}, {554, 186}}
		for i, p := range points {
			census.record(t, fmt.Sprintf("compose-shop-button%d-hover", i), ui.ComposeShopScreen(v, p, true, nil, false))
			if !v.Live[i] {
				continue
			}
			h := v
			h.Press = ui.ShopControl{Kind: ui.ShopControlButton, Index: i}
			census.record(t, fmt.Sprintf("compose-shop-button%d-press", i), ui.ComposeShopScreen(h, p, true, nil, false))
			census.record(t, fmt.Sprintf("compose-shop-button%d-press-out", i), ui.ComposeShopScreen(h, image.Pt(320, 2), true, nil, false))
		}
	})

	t.Run("save-dialog", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		stateDir := t.TempDir()
		f.Options = OptionsStore{Path: filepath.Join(stateDir, "options.txt")}
		app := f.App("plaque census save")
		t.Cleanup(app.StopAudio)
		app.Layout(640, 480)
		if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
			t.Fatal(err)
		}
		rows := make([]ui.SaveEntry, 12)
		for i := range rows {
			rows[i] = ui.SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("slot-%02d", i)}
		}
		app.SetSaveDialogSeams(ui.SaveDialogSeams{Directory: stateDir,
			List: func(directory string) (ui.SaveDirectory, error) {
				return ui.SaveDirectory{Path: directory, Entries: rows}, nil
			},
			Prepare: func(ui.SaveRequest) (ui.PreparedSave, error) {
				return ui.PreparedSave{}, fmt.Errorf("census writes no save")
			},
		})
		openMissionGameMenu(t, app)
		if err := app.HeadlessGameMenuAction("save"); err != nil || app.Screen() != ui.ScreenSave {
			t.Fatal("Save dialog", app.Screen(), err)
		}
		away := image.Pt(320, 380)
		census.pointer(t, app, "save-dialog", "hover", away)
		for _, b := range []struct {
			name string
			at   image.Point
		}{{"delete", image.Pt(196, 445)}, {"save", image.Pt(320, 445)}, {"cancel", image.Pt(444, 445)}, {"open", image.Pt(487, 65)}, {"up", image.Pt(570, 65)}} {
			census.press(t, app, "save-dialog-"+b.name, b.at, away)
		}
	})

	names := make([]string, 0, len(census.hashes))
	for name := range census.hashes {
		names = append(names, name)
	}
	sort.Strings(names)
	var list strings.Builder
	for _, name := range names {
		fmt.Fprintf(&list, "%s %s\n", census.hashes[name], name)
	}
	t.Logf("%s: %d census pictures\n%s", census.lang, len(names), list.String())
	if census.out != "" {
		if err := os.MkdirAll(census.out, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(census.out, census.lang+"-hashes.txt"), []byte(list.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
