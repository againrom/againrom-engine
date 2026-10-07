package game

import (
	"fmt"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// worldMapAtChapter loads the city after mission finished with every installed
// offer taken, through the LOAD window, and opens the world map through the
// gates. It answers a 1920x1080 window, where the native frame is placed at
// scale 2.25.
func worldMapAtChapter(t *testing.T, finished int) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("world map text smoothing")
	app.Layout(1024, 768)
	t.Cleanup(app.StopAudio)
	store := SaveStore{Dir: t.TempDir()}
	saved := 0
	save, list, load := f.SaveSeams(store, OriginalStore{}, func() time.Time {
		saved++
		return time.Date(2001, 1, 1, 0, 0, saved, 0, time.UTC)
	})
	app.SetSaveSeams(save, list, load)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Scroll witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.FinishMission(finished, f.Carried, nil, nil)
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		for _, offer := range f.Town.Offers(building) {
			if offer.Mission > 0 {
				if _, ok := f.Town.Take(building, offer.Index); !ok {
					t.Fatalf("could not accept installed offer %+v", offer)
				}
			}
		}
	}
	if _, err := save(false); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"load", "enter"} {
		if err := app.HeadlessKey(k); err != nil {
			t.Fatal(err)
		}
	}
	s := f.TownScreen().(*townScreen)
	if app.Screen() != ui.ScreenTown || !s.AtTownSquare() {
		t.Fatalf("the city did not load: screen %s", app.Screen())
	}
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	app.Layout(1920, 1080)
	return f, app, s
}

// The world map's scroll titles are smoothed wherever any of a title shows. A
// title wider than its scroll is drawn into the scroll's clipped interior and
// its last glyph can straddle the clip edge; the part of that glyph inside the
// scroll is still on the map, and a glyph left to the raster steps at window
// scale. After mission 120 the scrolls are the ones the owner reported
// (missions 130 and 131 beside the town's own); after mission 100 the RU title
// of mission 111 overflows its scroll, and after mission 130 the EN title of
// mission 141 does.
func TestReleaseWorldMapScrollTitlesAreSmoothedWhereverAnyOfThemShows(t *testing.T) {
	for _, finished := range []int{100, 120, 130} {
		t.Run(fmt.Sprintf("after mission %d", finished), func(t *testing.T) {
			_, a, s := worldMapAtChapter(t, finished)
			missions := s.WorldMapView().Missions
			if len(missions) == 0 {
				t.Fatalf("the map offers no mission after mission %d", finished)
			}
			states := []struct {
				name  string
				point func() (int, int, error)
			}{
				{"no pointer on a scroll", nil},
				{"pointer on the first mission scroll", func() (int, int, error) { return a.HeadlessWorldMapMissionPoint(missions[0].Number) }},
				{"pointer on the town scroll", a.HeadlessWorldMapTownPoint},
			}
			for _, state := range states {
				if state.point != nil {
					x, y, err := state.point()
					if err != nil {
						t.Fatal(err)
					}
					if err = a.HeadlessPointer("hover", x, y); err != nil {
						t.Fatal(err)
					}
				}
				for range 3 {
					if err := a.HeadlessStep(); err != nil {
						t.Fatal(err)
					}
				}
				a.Draw(ebiten.NewImage(1920, 1080))
				pix, _, err := a.HeadlessFrame()
				if err != nil {
					t.Fatal(err)
				}
				captured, kept, fallbacks := a.TextSettle()
				visible, partly := visibleGlyphs(text.Captured(), pix)
				t.Logf("%s: captured %d, kept %d, showing in the frame %d (%d in part), readback frames %d",
					state.name, captured, kept, visible, partly, fallbacks)
				if visible == 0 {
					t.Errorf("%s: no glyph shows on the map", state.name)
				}
				if kept < visible {
					t.Errorf("%s: the overlay smoothed %d glyphs and %d show in the frame: %d stay as the raster stepped them",
						state.name, kept, visible, visible-kept)
				}
				if fallbacks != 0 {
					t.Errorf("%s: the frame needed %d readbacks; the pixel log should decide it", state.name, fallbacks)
				}
			}
		})
	}
}
