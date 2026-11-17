package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

var missionHeightScreens = []MissionScreen{
	{ColumnW: 160, NoticeOnScreen: true},
	{ColumnW: 160, NoticeOnScreen: true, PickUsableInExploredGround: true},
	{ColumnW: 176, NoticeOnScreen: true},
}

func TestMissionFrameUsesLowWindowHeight(t *testing.T) {
	v := geometryViewer(t)
	for _, c := range []struct {
		window, frame image.Point
		scale         float64
	}{
		{image.Pt(1280, 720), image.Pt(1280, 720), 1},
		{image.Pt(1024, 768), image.Pt(1024, 768), 1},
		{image.Pt(1920, 1080), image.Pt(1280, 720), 1.5},
		{image.Pt(2560, 1440), image.Pt(1280, 720), 2},
		{image.Pt(3200, 1800), image.Pt(1280, 720), 2.5},
		{image.Pt(3840, 2160), image.Pt(1280, 720), 3},
		{image.Pt(1920, 1200), image.Pt(1229, 768), 1920.0 / 1229},
		{image.Pt(2048, 1536), image.Pt(1024, 768), 2},
		{image.Pt(1600, 900), image.Pt(1366, 768), 1600.0 / 1366},
		{image.Pt(800, 600), image.Pt(1024, 720), 800.0 / 1024},
		{image.Pt(1024, 767), image.Pt(1024, 720), 1},
		{image.Pt(1280, 768), image.Pt(1280, 768), 1},
		{image.Pt(1280, 720), image.Pt(1280, 720), 1},
	} {
		v.Layout(c.window.X, c.window.Y)
		if got := v.FrameSize(); got != c.frame {
			t.Errorf("window %v: frame %v, want %v", c.window, got, c.frame)
		}
		if got := v.Placement().Scale(); got != c.scale {
			t.Errorf("window %v: scale %v, want %v", c.window, got, c.scale)
		}
		if c.window == image.Pt(1280, 720) {
			ox, oy := v.Placement().Origin()
			if ox != 0 || oy != 0 {
				t.Errorf("native low frame origin (%v,%v), want (0,0)", ox, oy)
			}
		}
	}
	before, size := v.Placement(), v.FrameSize()
	v.Layout(0, 0)
	if v.Placement() != before || v.FrameSize() != size {
		t.Fatal("minimising changed the last usable layout")
	}
}

func TestMissionFrameMapRowsFollowHeight(t *testing.T) {
	for _, screen := range missionHeightScreens {
		v := geometryViewer(t)
		v.SetMissionScreen(screen)
		hideBottomPanels(v)
		for _, c := range []struct{ height, rows int }{{720, 22}, {768, 24}, {1080, 22}, {1200, 24}, {600, 22}} {
			v.Layout(1024, c.height)
			if got := v.Camera().ViewH / 32; got != c.rows {
				t.Errorf("column %d window height %d: map rows %d, want %d", screen.ColumnW, c.height, got, c.rows)
			}
			v.SetPackShown(true)
			pack, _, ok := v.packBarShown()
			if !ok || pack.Max.Y != v.FrameSize().Y || v.ViewportSize().Y != pack.Min.Y {
				t.Errorf("column %d: pack %v visible %v, map %v, frame %v", screen.ColumnW, pack, ok, v.ViewportSize(), v.FrameSize())
			}
			v.SetPackShown(false)
		}
	}
}

func TestMissionFrameColumnKeepsSlots(t *testing.T) {
	for _, screen := range missionHeightScreens {
		var card *image.RGBA
		for _, height := range []int{720, 768} {
			v := columnViewer(t)
			v.SetMissionScreen(screen)
			v.Layout(1024, height)
			rec := recordColumnLayers(t)
			dst := ebiten.NewImage(1024, height)
			v.Draw(dst)
			dst.Dispose()
			for _, c := range []struct {
				layer    string
				at, size image.Point
			}{
				{"minimap", image.Pt(866, 0), image.Pt(158, 158)},
				{"controlPanel", image.Pt(848, 158), image.Pt(176, 80)},
				{"panel", image.Pt(848, 238), image.Pt(176, 242)},
				{"filler", image.Pt(848, 480), image.Pt(176, 46)},
				{"missionCard", image.Pt(848, height-242), image.Pt(176, 242)},
			} {
				got, ok := rec[c.layer]
				if !ok || got.origin != c.at || got.size != c.size {
					t.Errorf("column %d height %d layer %s: %v present %v, want origin %v size %v", screen.ColumnW, height, c.layer, got, ok, c.at, c.size)
				}
			}
			pic := v.missionCardPic
			if pic == nil {
				t.Error("no lower card picture")
				continue
			}
			if card != nil && !bytes.Equal(card.Pix, pic.Pix) {
				t.Error("frame height changed the native lower card pixels")
			}
			card = pic
		}
	}
}

func TestMissionCardFitsShortFrameAndKeepsBasePlacement(t *testing.T) {
	for i, screen := range missionHeightScreens[:2] {
		t.Run([]string{"first game", "second game"}[i], func(t *testing.T) {
			for _, c := range []struct {
				height, top, panelOverlap, fillerOverlap int
			}{{720, 478, 2, 46}, {768, 526, 0, 0}} {
				v := columnViewer(t)
				v.SetMissionScreen(screen)
				v.Layout(1024, c.height)
				box, ok := missionCardBoxRect(v.FrameSize())
				want := image.Rect(864, c.top, 1024, c.height)
				if !ok || box != want || !box.In(image.Rectangle{Max: v.FrameSize()}) || box.Dy() != 242 {
					t.Errorf("height %d: card %v visible %v, want whole %v", c.height, box, ok, want)
				}
				panel, _ := characterPanelBoxRect(v.FrameSize())
				filler, _ := columnFillerRect(v.FrameSize())
				if panel != image.Rect(864, 238, 1024, 480) || filler != image.Rect(864, 480, 1024, 526) {
					t.Errorf("height %d moved panel %v or filler %v", c.height, panel, filler)
				}
				if box.Intersect(panel).Dy() != c.panelOverlap || box.Intersect(filler).Dy() != c.fillerOverlap {
					t.Errorf("height %d: card overlaps panel by %d and filler by %d, want %d and %d",
						c.height, box.Intersect(panel).Dy(), box.Intersect(filler).Dy(), c.panelOverlap, c.fillerOverlap)
				}
				rec := recordColumnLayers(t)
				var order []string
				previous := blitColumnLayer
				blitColumnLayer = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions, layer string) {
					order = append(order, layer)
					previous(dst, src, op, layer)
				}
				t.Cleanup(func() { blitColumnLayer = previous })
				dst := ebiten.NewImage(1024, c.height)
				v.Draw(dst)
				dst.Dispose()
				if got := rec["missionCard"]; got.origin != image.Pt(848, c.top) || got.size != image.Pt(176, 242) {
					t.Errorf("height %d: card draw %+v, want origin (848,%d) size (176,242)", c.height, got, c.top)
				}
				cardIndex, panelIndex, fillerIndex := -1, -1, -1
				for j, layer := range order {
					switch layer {
					case "missionCard":
						cardIndex = j
					case "panel":
						panelIndex = j
					case "filler":
						fillerIndex = j
					}
				}
				if panelIndex < 0 || fillerIndex < 0 || cardIndex <= panelIndex || cardIndex <= fillerIndex {
					t.Errorf("height %d: card must draw above panel and filler; order %v", c.height, order)
				}
			}
		})
	}
}

func TestMissionFrameKeepsModeCornerHidden(t *testing.T) {
	for i, screen := range missionHeightScreens[:2] {
		t.Run([]string{"first game", "second game"}[i], func(t *testing.T) {
			for _, window := range []image.Point{{1280, 720}, {1920, 1080}, {1024, 768}, {1920, 1200}} {
				v := columnViewer(t)
				v.SetMissionScreen(screen)
				body := TownPane{Body: solidPic(160, 242, missionPaneCornerColours.body)}
				v.SetCharacterPaneArt(body, body, missionPaneArt())
				v.Layout(window.X, window.Y)
				before := missionPaneCompose(t, v)
				for _, col := range []color.RGBA{missionPaneCornerColours.humanMode, missionPaneCornerColours.textMode} {
					if at, drawn := findColour(before, col); drawn {
						t.Errorf("window %v: rect C is drawn at panel pixel %v", window, at)
					}
				}
				if _, _, shown := v.missionCardPresent(); !shown {
					t.Fatalf("window %v: no lower card before pressing rect C", window)
				}
				pane, _ := v.characterPaneRect()
				corner := CharacterPaneCornerRect(pane, CharacterPaneMode)
				want := image.Rect(v.FrameSize().X-32, 238, v.FrameSize().X, 274)
				if corner != want {
					t.Fatalf("window %v: rect C is %v, want %v", window, corner, want)
				}
				x, y, err := headlessControlPoint(corner, v.place.WindowSize(), v.place.FrameToWindow, v.place.WindowToFrame, "mode corner test")
				if err != nil {
					t.Fatal(err)
				}
				fx, fy := v.windowToFrame(x, y)
				mode := v.characterPaneStatistics()
				if v.pressCharacterPaneCorner(fx, fy) {
					t.Errorf("window %v: rect C consumed a press at frame (%d,%d)", window, fx, fy)
				}
				if v.characterPaneStatistics() != mode || !bytes.Equal(before.Pix, missionPaneCompose(t, v).Pix) {
					t.Errorf("window %v: pressing rect C changed the hero panel", window)
				}
				if _, _, shown := v.missionCardPresent(); !shown {
					t.Errorf("window %v: pressing rect C hid the lower card", window)
				}
			}
		})
	}
}
