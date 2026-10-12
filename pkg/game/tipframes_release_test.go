package game

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// tipFrame is one composed tip: the panel alone on a blank 640x480 canvas
// and the screen frame it draws on.
type tipFrame struct {
	view  ui.TipPanelView
	panel *image.RGBA
	frame *image.RGBA
}

func tipPanelAlone(v ui.TipPanelView, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	ui.ComposeTipPanel(dst, v)
	return dst
}

func tipDigest(pic *image.RGBA) string {
	if pic == nil {
		return "-"
	}
	sum := sha256.Sum256(pic.Pix)
	return hex.EncodeToString(sum[:8])
}

// tipFrameLine is one frame's record: the view's rectangle and text digest,
// the panel's pixels and the screen frame's pixels.
func tipFrameLine(name string, f tipFrame) string {
	text := sha256.Sum256([]byte(f.view.Text))
	return fmt.Sprintf("%s rect=%v text=%s panel=%s frame=%s", name, f.view.Rect, hex.EncodeToString(text[:6]), tipDigest(f.panel), tipDigest(f.frame))
}

// generatorTipFrames drives the installed generator through its tip steps on
// a headless App: the three pre-create steps, the detailed page's class tip
// for picture, and the text after the first skill click.
func generatorTipFrames(t *testing.T, f *FrontEnd, picture int) map[string]tipFrame {
	t.Helper()
	out := map[string]tipFrame{}
	setup := f.ChargenSetup()
	setup.TipsOn = true
	c := ui.NewChargen(setup)
	app := f.App("tip frames")
	t.Cleanup(app.StopAudio)
	app.Layout(640, 480)
	if err := app.OpenChargen(c, func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(10, res), nil }); err != nil {
		t.Fatal(err)
	}
	click := func(p image.Point) {
		t.Helper()
		for _, action := range []string{"press", "release"} {
			if err := app.HeadlessPointer(action, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	prePoint := func(owner string) image.Point {
		t.Helper()
		for y := 479; y >= 0; y-- {
			for x := 0; x < 640; x++ {
				p := image.Pt(x, y)
				if got, ok := ui.PreCreateControlAt(c, p); ok && got == owner && !c.TipPanel().Covers(p) {
					return p
				}
			}
		}
		t.Fatalf("no installed pre-create pixel is owned by %q", owner)
		return image.Point{}
	}
	record := func(name string) {
		v := c.TipPanel()
		if !v.Showing() {
			t.Fatalf("%s: no tip panel", name)
		}
		out[name] = tipFrame{view: v, panel: tipPanelAlone(v, 640, 480), frame: ui.ComposeChargenFrame(c)}
	}
	for step, press := range []string{fmt.Sprintf("choice %d", picture), "difficulty 1", ""} {
		record(fmt.Sprintf("generator-select-%d-picture%d", step, picture))
		if press != "" {
			click(prePoint(press))
		}
	}
	click(prePoint("forward"))
	if c.Stage() != ui.DetailedStage {
		t.Fatal("OK did not open the detailed page")
	}
	record(fmt.Sprintf("generator-detail-picture%d", picture))
	column := f.generator().Detail.ColumnRect.Rectangle()
	for y := column.Min.Y; y < column.Max.Y && c.TipStep() == 0; y += 4 {
		for x := column.Min.X; x < column.Max.X && c.TipStep() == 0; x += 4 {
			if p := image.Pt(x, y); !c.TipPanel().Covers(p) {
				click(p)
			}
		}
	}
	if c.TipStep() != 1 {
		t.Fatal("no click in the skill column advanced the detailed tip")
	}
	record(fmt.Sprintf("generator-after-picture%d", picture))
	return out
}

// TestReleaseTipFrameDigests composes every tip the engine shows on the
// installed root and records its rectangle, text and pixels: the rooms, the
// shop's second text, the generator steps and the first mission tip. Two
// revisions that draw the same tips print the same lines; the lines go to
// AGAINROM_TIP_FRAME_DIGESTS when it names a file.
func TestReleaseTipFrameDigests(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: tip frames need a lawful install")
	}
	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	second := archives.Game() == base.GameROM2
	frames := map[string]tipFrame{}
	if second {
		for picture := range 4 {
			f := secondGameFront(t)
			f.Options = OptionsStore{}
			f.PersistenceContext.tipsOff = false
			for k, v := range generatorTipFrames(t, f, picture) {
				frames[k] = v
			}
		}
	} else {
		for _, room := range []struct {
			name   string
			choose int
		}{{"tavern", 0}, {"shop", 1}, {"school", 2}} {
			f := missionTipFront(t)
			s := f.TownScreen().(*townScreen)
			if room.name == "tavern" {
				sq := s.TownSquareView()
				frames["room-square"] = tipFrame{view: sq.Tip, panel: tipPanelAlone(sq.Tip, 640, 480), frame: ui.ComposeTownSquare(sq)}
			}
			s.Choose(room.choose)
			switch room.name {
			case "shop":
				v := s.ShopScreen()
				frames["room-shop"] = tipFrame{view: v.TipPanel, panel: tipPanelAlone(v.TipPanel, 640, 480), frame: ui.ComposeShopScreen(v, image.Point{}, false, nil, false)}
			default:
				v := s.TownSurface()
				frames["room-"+room.name] = tipFrame{view: v.Tip, panel: tipPanelAlone(v.Tip, 640, 480), frame: ui.ComposeTownSurface(v)}
			}
		}
		f := missionTipFront(t)
		app := roomTipTownApp(t, f)
		s := roomTipEnterShop(t, f, app)
		roomTipSteps(t, app, 3)
		roomTipPutOnTable(t, s)
		roomTipSteps(t, app, 1)
		v := s.ShopScreen()
		frames["room-shop-second"] = tipFrame{view: v.TipPanel, panel: tipPanelAlone(v.TipPanel, 640, 480), frame: ui.ComposeShopScreen(v, image.Point{}, false, nil, false)}

		for picture := range 4 {
			g := missionTipFront(t)
			for k, v := range generatorTipFrames(t, g, picture) {
				frames[k] = v
			}
		}

		m := missionTipFront(t)
		mapp, live := openMissionTipMap(t, m, 10)
		pageMissionDialogue(t, mapp, live, "start dialogue")
		mv := live.view.MissionTip()
		frames["mission-10-start"] = tipFrame{view: mv, panel: tipPanelAlone(mv, 1024, 768), frame: ui.ComposeMissionTip(live.view, 1024, 768)}
	}
	names := make([]string, 0, len(frames))
	for k, v := range frames {
		if !v.view.Showing() {
			t.Errorf("%s: no tip panel", k)
		}
		names = append(names, k)
	}
	sort.Strings(names)
	lines := make([]string, 0, len(names))
	for _, k := range names {
		lines = append(lines, tipFrameLine(k, frames[k]))
		t.Log(lines[len(lines)-1])
	}
	if out := os.Getenv("AGAINROM_TIP_FRAME_DIGESTS"); out != "" {
		if err := os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// withTipRow runs the rest of the test with the installed game's table
// changed by change on its row for event.
func withTipRow(t *testing.T, e base.Edition, event, room, class string, change func(*ui.TipRow)) {
	t.Helper()
	old := tipTables[e.Tips]
	changed := *old
	changed.Rows = append([]ui.TipRow(nil), old.Rows...)
	row := changed.Row(event, room, class)
	if row == nil {
		t.Fatalf("the %s table has no %s row", e.Tips, event)
	}
	change(row)
	tipTables[e.Tips] = &changed
	t.Cleanup(func() { tipTables[e.Tips] = old })
}

// TestReleaseTipFramesFollowTheirRows is the digests' loss control: a row
// whose rectangle or text changes changes its family's frame, so equal
// digests come from equal rows, not from a frame that ignores them.
func TestReleaseTipFramesFollowTheirRows(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: tip frames need a lawful install")
	}
	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	e := archives.Game().Edition()
	generator := func(t *testing.T) string {
		var f *FrontEnd
		if archives.Game() == base.GameROM2 {
			f = secondGameFront(t)
			f.Options = OptionsStore{}
			f.PersistenceContext.tipsOff = false
		} else {
			f = missionTipFront(t)
		}
		c := ui.NewChargen(f.ChargenSetup())
		return tipDigest(ui.ComposeChargenFrame(c))
	}
	t.Run("generator", func(t *testing.T) {
		before := generator(t)
		withTipRow(t, e, "pre-create-enter", "", "", func(r *ui.TipRow) { r.Rect[1] += 8; r.Rect[3] += 8 })
		if generator(t) == before {
			t.Error("a moved pre-create row left the generator frame unchanged")
		}
	})
	t.Run("generator text", func(t *testing.T) {
		before := generator(t)
		withTipRow(t, e, "pre-create-enter", "", "", func(r *ui.TipRow) {
			r.Text = tipTables[e.Tips].Row("first-skill-click", "", "").Text
		})
		if generator(t) == before {
			t.Error("a retexted pre-create row left the generator frame unchanged")
		}
	})
	if archives.Game() == base.GameROM2 {
		return
	}
	square := func(t *testing.T) string {
		f := missionTipFront(t)
		return tipDigest(ui.ComposeTownSquare(f.TownScreen().(*townScreen).TownSquareView()))
	}
	t.Run("room", func(t *testing.T) {
		before := square(t)
		withTipRow(t, e, "room-enter", "square", "", func(r *ui.TipRow) { r.Rect[0] -= 8; r.Rect[2] -= 8 })
		if square(t) == before {
			t.Error("a moved square row left the square frame unchanged")
		}
	})
	t.Run("room text", func(t *testing.T) {
		before := square(t)
		withTipRow(t, e, "room-enter", "square", "", func(r *ui.TipRow) { r.Text.File = "main/text/tips/inn.txt" })
		if square(t) == before {
			t.Error("a retexted square row left the square frame unchanged")
		}
	})
	mission := func(t *testing.T) (ui.TipPanelView, string) {
		f := missionTipFront(t)
		app, live := openMissionTipMap(t, f, 10)
		pageMissionDialogue(t, app, live, "start dialogue")
		return live.view.MissionTip(), tipDigest(ui.ComposeMissionTip(live.view, 1024, 768))
	}
	t.Run("mission", func(t *testing.T) {
		v, before := mission(t)
		if !v.Showing() {
			t.Fatal("the start dialogue raised no mission tip")
		}
		withTipRow(t, e, "dialogue-close", "", "", func(r *ui.TipRow) { r.Rect[0] += 8; r.Rect[2] += 8 })
		if _, after := mission(t); after == before {
			t.Error("a moved mission row left the mission popup unchanged")
		}
	})
	t.Run("mission text", func(t *testing.T) {
		_, before := mission(t)
		withTipRow(t, e, "dialogue-close", "", "", func(r *ui.TipRow) { r.Text.File = "main/text/battle/m%d/tips%02d.txt.missing" })
		if v, after := mission(t); after == before || v.Showing() {
			t.Error("a mission row naming no shipped text still showed the popup")
		}
	})
}
