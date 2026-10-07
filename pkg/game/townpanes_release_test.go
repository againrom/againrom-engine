package game

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseTownColumnPanesDrawShippedBodyAndSeam(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil || f.TownTavernArt.Value() == nil {
		t.Fatalf("production surface art did not resolve: school %v tavern %v",
			f.TownSchoolArt.Err(), f.TownTavernArt.Err())
	}
	if f.TownTavernArt.Value().LeftStatsSeam == nil || f.TownTavernArt.Value().LeftPictureSeam == nil {
		t.Fatalf("production tavern art did not resolve its own left-column seams")
	}
	panes := f.characterPanes()
	if panes.Figure.Body == nil || panes.Figure.Seam == nil || panes.Stats.Body == nil || panes.Stats.Seam == nil {
		t.Fatalf("production character panes did not resolve: %+v", panes)
	}
	filler := f.characterPaneFiller()
	if filler.Body == nil || filler.Seam == nil {
		t.Fatalf("production mission filler did not resolve its body and closing seam: %+v", filler)
	}
	if got, want := filler.Body.Bounds(), image.Rect(0, 0, 160, 46); got != want {
		t.Errorf("mission filler body bounds = %v, want %v", got, want)
	}
	if got, want := filler.Seam.Bounds(), image.Rect(0, 0, 16, 46); got != want {
		t.Errorf("mission filler seam bounds = %v, want %v", got, want)
	}
	art := f.shopArt()
	if art == nil || art.Menu == nil {
		t.Fatalf("production shop art did not resolve its own menu plaque")
	}

	assertRegion := func(t *testing.T, frame *image.RGBA, rect image.Rectangle, src image.Image, exclude ...image.Rectangle) {
		t.Helper()
		b := src.Bounds()
		if b.Dx() != rect.Dx() || b.Dy() != rect.Dy() {
			t.Fatalf("source is %dx%d, the rectangle it fills is %dx%d", b.Dx(), b.Dy(), rect.Dx(), rect.Dy())
		}
		mismatches, total := 0, 0
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				p := image.Pt(x, y)
				excluded := false
				for _, e := range exclude {
					if p.In(e) {
						excluded = true
						break
					}
				}
				if excluded {
					continue
				}
				total++
				got := frame.RGBAAt(x, y)
				sr, sg, sb, sa := src.At(b.Min.X+x-rect.Min.X, b.Min.Y+y-rect.Min.Y).RGBA()
				want := color.RGBA{R: uint8(sr >> 8), G: uint8(sg >> 8), B: uint8(sb >> 8), A: uint8(sa >> 8)}
				if got != want {
					mismatches++
					if mismatches <= 5 {
						t.Errorf("%v: composed pixel %#v != source pixel %#v", p, got, want)
					}
				}
			}
		}
		if mismatches > 0 {
			t.Fatalf("%d of %d checked pixel(s) are not the shipped source", mismatches, total)
		}
		if total == 0 {
			t.Fatalf("the exclusion rectangles cover the whole region %v, nothing was checked", rect)
		}
	}

	// assertSeamRegion is assertRegion's own keyed-aware sibling (round-2
	// adversarial review, item 2). before is the SAME frame composed with
	// this one seam suppressed (its own TownPane carrying a nil Seam),
	// through the identical production composer — drawTownPane's own
	// nil-Seam branch leaves seamRect exactly as already composed, which is
	// the production "underneath" state for a keyed source's own key
	// pixels. Where src's own pixel is opaque (alpha 255, keyBlack's own
	// untouched case) the composed frame must equal src; where key (alpha
	// 0) it must equal before, not the source's own zero-value pixel.
	assertSeamRegion := func(t *testing.T, frame, before *image.RGBA, rect image.Rectangle, src image.Image) {
		t.Helper()
		b := src.Bounds()
		if b.Dx() != rect.Dx() || b.Dy() != rect.Dy() {
			t.Fatalf("source is %dx%d, the rectangle it fills is %dx%d", b.Dx(), b.Dy(), rect.Dx(), rect.Dy())
		}
		mismatches, total := 0, 0
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				total++
				sr, sg, sb, sa := src.At(b.Min.X+x-rect.Min.X, b.Min.Y+y-rect.Min.Y).RGBA()
				got := frame.RGBAAt(x, y)
				var want color.RGBA
				if sa == 0 {
					want = before.RGBAAt(x, y)
				} else {
					want = color.RGBA{R: uint8(sr >> 8), G: uint8(sg >> 8), B: uint8(sb >> 8), A: uint8(sa >> 8)}
				}
				if got != want {
					mismatches++
					if mismatches <= 5 {
						t.Errorf("%v: composed pixel %#v != want %#v (shipped seam pixel alpha %d)", image.Pt(x, y), got, want, sa)
					}
				}
			}
		}
		if mismatches > 0 {
			t.Fatalf("%d of %d checked pixel(s) do not follow the shipped seam's own key/opaque split", mismatches, total)
		}
	}

	characterPanelControls := ui.TownCharacterPersistentControls()

	tavernFrame := ui.ComposeTownSurface(ui.TownSurfaceView{
		Kind: ui.TownSurfaceTavern, TavernArt: f.TownTavernArt.Value(), Font: f.Font.Value(), HoverCell: -1,
		Hero: ui.TownCharacterView{HasSubject: false, FigurePane: panes.Figure, StatsPane: panes.Stats, Font: f.Font.Value()},
	})

	// tavernWithout composes the tavern frame with one named TavernArt seam
	// field suppressed, everything else production. A shallow copy of
	// *f.TownTavernArt is safe: TownTavernArt's own fields are all
	// image.Image or value types, so nilling one field on the copy cannot
	// mutate the cached original any other subtest reads.
	tavernWithout := func(mutate func(*ui.TownTavernArt)) *image.RGBA {
		artCopy := *f.TownTavernArt.Value()
		mutate(&artCopy)
		return ui.ComposeTownSurface(ui.TownSurfaceView{
			Kind: ui.TownSurfaceTavern, TavernArt: &artCopy, Font: f.Font.Value(), HoverCell: -1,
			Hero: ui.TownCharacterView{HasSubject: false, FigurePane: panes.Figure, StatsPane: panes.Stats, Font: f.Font.Value()},
		})
	}
	t.Run("tavern left upper", func(t *testing.T) {
		assertRegion(t, tavernFrame, image.Rect(0, 0, 160, 238), f.TownTavernArt.Value().LeftStats)
		before := tavernWithout(func(a *ui.TownTavernArt) { a.LeftStatsSeam = nil })
		assertSeamRegion(t, tavernFrame, before, image.Rect(160, 0, 176, 238), f.TownTavernArt.Value().LeftStatsSeam)
	})
	t.Run("tavern left lower", func(t *testing.T) {
		assertRegion(t, tavernFrame, image.Rect(0, 238, 160, 480), f.TownTavernArt.Value().LeftPicture)
		before := tavernWithout(func(a *ui.TownTavernArt) { a.LeftPictureSeam = nil })
		assertSeamRegion(t, tavernFrame, before, image.Rect(160, 238, 176, 480), f.TownTavernArt.Value().LeftPictureSeam)
	})

	// tavernCharacterWithout composes the tavern frame with the shared
	// character pane's own Seam field suppressed for the named mode alone;
	// the other mode's own pane is left exactly as production supplies it.
	tavernCharacterWithout := func(statistics bool) *image.RGBA {
		figure, stats := panes.Figure, panes.Stats
		if statistics {
			stats.Seam = nil
		} else {
			figure.Seam = nil
		}
		return ui.ComposeTownSurface(ui.TownSurfaceView{
			Kind: ui.TownSurfaceTavern, TavernArt: f.TownTavernArt.Value(), Font: f.Font.Value(), HoverCell: -1,
			Hero: ui.TownCharacterView{Statistics: statistics, HasSubject: false, FigurePane: figure, StatsPane: stats, Font: f.Font.Value()},
		})
	}
	t.Run("tavern character panel figure mode", func(t *testing.T) {
		assertRegion(t, tavernFrame, ui.TownCharacterRegion, panes.Figure.Body, characterPanelControls...)
		before := tavernCharacterWithout(false)
		assertSeamRegion(t, tavernFrame, before, image.Rect(464, 238, 480, 480), panes.Figure.Seam)
	})

	tavernStats := ui.ComposeTownSurface(ui.TownSurfaceView{
		Kind: ui.TownSurfaceTavern, TavernArt: f.TownTavernArt.Value(), Font: f.Font.Value(), HoverCell: -1,
		Hero: ui.TownCharacterView{Statistics: true, HasSubject: false, FigurePane: panes.Figure, StatsPane: panes.Stats, Font: f.Font.Value()},
	})
	t.Run("tavern character panel stats mode", func(t *testing.T) {
		assertRegion(t, tavernStats, ui.TownCharacterRegion, panes.Stats.Body, characterPanelControls...)
		before := tavernCharacterWithout(true)
		assertSeamRegion(t, tavernStats, before, image.Rect(464, 238, 480, 480), panes.Stats.Seam)
	})

	schoolFrame := ui.ComposeTownSurface(ui.TownSurfaceView{
		Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(), SchoolClass: -1, Font: f.Font.Value(), HoverCell: -1,
		Hero: ui.TownCharacterView{HasSubject: false, FigurePane: panes.Figure, StatsPane: panes.Stats, Font: f.Font.Value()},
	})
	t.Run("school character panel figure mode", func(t *testing.T) {
		assertRegion(t, schoolFrame, ui.TownCharacterRegion, panes.Figure.Body, characterPanelControls...)
		figure := panes.Figure
		figure.Seam = nil
		before := ui.ComposeTownSurface(ui.TownSurfaceView{
			Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(), SchoolClass: -1, Font: f.Font.Value(), HoverCell: -1,
			Hero: ui.TownCharacterView{HasSubject: false, FigurePane: figure, StatsPane: panes.Stats, Font: f.Font.Value()},
		})
		assertSeamRegion(t, schoolFrame, before, image.Rect(464, 238, 480, 480), panes.Figure.Seam)
	})

	// school stats mode (round-2 adversarial review, item 6): the story's
	// own spec.md claimed eight subtests while the committed test carried
	// seven, with the school's own Statistics mode the one production state
	// the population never composed at all — the same shared TownCharacterView
	// the tavern's own "stats mode" subtest above already exercises, applied
	// to the school's SchoolArt instead of TavernArt.
	schoolStats := ui.ComposeTownSurface(ui.TownSurfaceView{
		Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(), SchoolClass: -1, Font: f.Font.Value(), HoverCell: -1,
		Hero: ui.TownCharacterView{Statistics: true, HasSubject: false, FigurePane: panes.Figure, StatsPane: panes.Stats, Font: f.Font.Value()},
	})
	t.Run("school character panel stats mode", func(t *testing.T) {
		assertRegion(t, schoolStats, ui.TownCharacterRegion, panes.Stats.Body, characterPanelControls...)
		stats := panes.Stats
		stats.Seam = nil
		before := ui.ComposeTownSurface(ui.TownSurfaceView{
			Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(), SchoolClass: -1, Font: f.Font.Value(), HoverCell: -1,
			Hero: ui.TownCharacterView{Statistics: true, HasSubject: false, FigurePane: panes.Figure, StatsPane: stats, Font: f.Font.Value()},
		})
		assertSeamRegion(t, schoolStats, before, image.Rect(464, 238, 480, 480), panes.Stats.Seam)
	})

	shopFrame := ui.ComposeShopScreen(ui.ShopScreenView{
		Art: art, Font: f.Font.Value(), Chosen: -1,
		Character: ui.TownCharacterView{HasSubject: false, FigurePane: panes.Figure, StatsPane: panes.Stats, Font: f.Font.Value()},
	}, image.Point{}, false, nil, false)
	t.Run("shop upper (shopmenu.bmp, no separate seam)", func(t *testing.T) {
		// Menu includes its keyed left edge. Verify the room remains visible
		// there, using the same screen with only Menu suppressed underneath.
		artCopy := *art
		artCopy.Menu = image.NewRGBA(art.Menu.Bounds())
		want := ui.ComposeShopScreen(ui.ShopScreenView{
			Art: &artCopy, Font: f.Font.Value(), Chosen: -1,
			Character: ui.TownCharacterView{HasSubject: false, FigurePane: panes.Figure, StatsPane: panes.Stats, Font: f.Font.Value()},
		}, image.Point{}, false, nil, false)
		draw.Draw(want, ui.TownWideUpperRegion, art.Menu, art.Menu.Bounds().Min, draw.Over)
		var excludes []image.Rectangle
		for i := 0; i < 4; i++ {
			excludes = append(excludes, ui.ShopButtonRect(i))
		}
		assertRegion(t, shopFrame, ui.TownWideUpperRegion, want.SubImage(ui.TownWideUpperRegion), excludes...)
	})
	t.Run("shop character panel figure mode", func(t *testing.T) {
		// The authored Book plaque (548,242)-(632,272) this case used to
		// exclude in addition to characterPanelControls is deleted (owner
		// finding 3, round-2 adversarial return): rect B's own decoded art
		// is the shop's only Book control now, and characterPanelControls
		// already covers it, on top of the same three controls the tavern
		// and school also draw.
		assertRegion(t, shopFrame, ui.TownCharacterRegion, panes.Figure.Body, characterPanelControls...)
		figure := panes.Figure
		figure.Seam = nil
		before := ui.ComposeShopScreen(ui.ShopScreenView{
			Art: art, Font: f.Font.Value(), Chosen: -1,
			Character: ui.TownCharacterView{HasSubject: false, FigurePane: figure, StatsPane: panes.Stats, Font: f.Font.Value()},
		}, image.Point{}, false, nil, false)
		assertSeamRegion(t, shopFrame, before, image.Rect(464, 238, 480, 480), panes.Figure.Seam)
	})
}
