package ui

import (
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
)

// TestWorldMapCardTextUsesItsOwnOrigins is the card text geometry witness.
// The expected frame is composed once with no font, then the three strings are
// drawn at literal coordinates taken from the card contract. It therefore does
// not ask drawWorldMapCard where its text went. Moving any production text
// draw by one pixel makes the whole-frame comparison fail.
func TestWorldMapScrollsCollapseAndExpandOnlyUnderThePointer(t *testing.T) {
	font := panelFont()
	v := WorldMapView{Font: font, DetailFont: font, Words: AuthoredWords(), Missions: []WorldMapMission{{Title: "MISSION", Briefing: "Find the man who sold the cloak to the shopkeeper", Payment: 37}}, Hovered: 0, Selected: 0}
	r := WorldMapCardRect(0)
	if r.Min.X >= WorldMapCardRect(-1).Min.X || WorldMapCardRect(-1).Max.X != 640 {
		t.Fatal("town must remain at the right edge")
	}
	if got := worldMapCardRect(v, 0, 0); got != r {
		t.Fatal("selection or region hover expanded the scroll")
	}
	closed := ComposeWorldMap(v)
	v.ScrollHovered = true
	expanded := worldMapCardRect(v, 0, 0)
	if expanded.Dy() <= r.Dy() {
		t.Fatal("hover did not reveal the briefing")
	}
	for _, line := range worldMapCardDetail(v, 0) {
		w, _ := font.Measure(line)
		if w > worldMapCardW-20 {
			t.Fatalf("unwrapped line %q", line)
		}
	}
	p := image.Pt(r.Min.X+30, r.Max.Y+2)
	if i, ok := WorldMapCardAt(v, p); !ok || i != 0 {
		t.Fatal("expanded paper loses hover and click capture")
	}
	if got := ComposeWorldMap(v); got.RGBAAt(p.X, p.Y) == closed.RGBAAt(p.X, p.Y) {
		t.Fatal("expanded paper was not drawn")
	}
	v.ScrollHovered = false
	if _, ok := WorldMapCardAt(v, p); ok {
		t.Fatal("collapsed scroll retained its old hit area")
	}
}

type fakeWorldMapTown struct {
	view    WorldMapView
	hovers  []image.Point
	clicks  []image.Point
	moves   []int
	chooses int
	ticks   int
	backed  bool
}

func (f *fakeWorldMapTown) Header() string        { return "world map" }
func (f *fakeWorldMapTown) Rows() []TownRow       { return nil }
func (f *fakeWorldMapTown) Footer() []string      { return nil }
func (f *fakeWorldMapTown) Choose(int) TownAction { return TownAction{} }
func (f *fakeWorldMapTown) Back() bool            { f.backed = true; return true }
func (f *fakeWorldMapTown) AtWorldMap() bool      { return true }
func (f *fakeWorldMapTown) WorldMapView() WorldMapView {
	return f.view
}
func (f *fakeWorldMapTown) WorldMapHover(p image.Point) { f.hovers = append(f.hovers, p) }
func (f *fakeWorldMapTown) WorldMapMove(delta int)      { f.moves = append(f.moves, delta) }
func (f *fakeWorldMapTown) WorldMapChoose() TownAction  { f.chooses++; return TownAction{} }
func (f *fakeWorldMapTown) WorldMapClick(p image.Point) TownAction {
	f.clicks = append(f.clicks, p)
	return TownAction{}
}
func (f *fakeWorldMapTown) WorldMapTick() TownAction { f.ticks++; return TownAction{} }

func TestWorldMapCardsHavePriorityAndRegionsUseObjectOrder(t *testing.T) {
	v := WorldMapView{Missions: []WorldMapMission{
		{Enabled: true, Object: 7, Region: image.Rect(0, 0, 640, 480)},
		{Enabled: true, Object: 3, Region: image.Rect(20, 20, 80, 80)},
	}}
	card := WorldMapCardRect(0)
	p := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	if got, ok := WorldMapCardAt(v, p); !ok || got != 0 {
		t.Fatalf("mission card hit = (%d,%v), want (0,true)", got, ok)
	}
	if got, ok := WorldMapRegionAt(v, image.Pt(30, 30)); !ok || got != 1 {
		t.Fatalf("overlapping registry regions hit = (%d,%v), want lower object index 1", got, ok)
	}
	if got, ok := WorldMapCardAt(v, image.Pt(630, 470)); ok || got != 0 {
		t.Fatalf("outside cards = (%d,%v), want no hit", got, ok)
	}
}

func TestWorldMapCardsPageWithoutADataLimit(t *testing.T) {
	v := WorldMapView{Missions: make([]WorldMapMission, 10), CardBase: 7}
	p := WorldMapCardRect(0).Min.Add(image.Pt(3, 3))
	if got, ok := WorldMapCardAt(v, p); !ok || got != 7 {
		t.Fatalf("first card on second page = (%d,%v), want campaign index 7", got, ok)
	}
}

func TestWorldMapCompositionDrawsBackgroundRouteFlagAndSeparateTownScroll(t *testing.T) {
	bg := image.NewRGBA(image.Rect(0, 0, 640, 480))
	bg.SetRGBA(500, 400, color.RGBA{R: 9, G: 19, B: 29, A: 255})
	cross := image.NewRGBA(image.Rect(0, 0, 5, 1))
	for x := 0; x < 5; x++ {
		cross.SetRGBA(x, 0, color.RGBA{B: 255, A: 255})
	}
	flag := image.NewRGBA(image.Rect(0, 0, 1, 1))
	flag.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	v := WorldMapView{
		Background: bg,
		Cross:      cross,
		Flag:       flag,
		Missions:   []WorldMapMission{{Enabled: true, Anchor: image.Pt(300, 300), Title: "TITLE", Briefing: "DETAIL"}},
		Selected:   0,
		Route:      []image.Point{{100, 200}, {200, 250}, {300, 300}},
		RouteShown: 2,
		// Position is the party's own field (`TOWN-120`/`TOWN-121`), not the
		// route's leading revealed point: it stays put during travel, so it
		// is set away from every route point and from both scroll cards
		// (drawn last, over the top-left corner) here to isolate the Flag
		// stamp's own location from the route stamps and the card paper.
		Position: image.Pt(50, 400),
	}
	got := ComposeWorldMap(v)
	if got.RGBAAt(500, 400) != (color.RGBA{R: 9, G: 19, B: 29, A: 255}) {
		t.Fatal("background was not copied into the native world-map frame")
	}
	if c := got.RGBAAt(100, 200); c != worldMapRoute {
		t.Fatalf("route stamp = %v, want %v", c, worldMapRoute)
	}
	if c := got.RGBAAt(200, 250); c != (color.RGBA{}) {
		t.Fatalf("shown route point off the ball stride = %v, want the background image's own (unset) pixel there (no leading marker any more)", c)
	}
	if c := got.RGBAAt(298, 300); c != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("selected destination cross = %v, want blue cross pixel", c)
	}
	if c := got.RGBAAt(300, 300); c != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("destination centre = %v, want the cross's own blue pixel (Flag no longer draws at the destination)", c)
	}
	if c := got.RGBAAt(50, 400); c != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("current-position flag = %v, want red flag pixel at Position", c)
	}
	if WorldMapCardRect(-1).Overlaps(WorldMapCardRect(0)) {
		t.Fatal("the town transition scroll overlaps the first mission scroll")
	}
}

func TestWorldMapCompositionTreatsTypedNilImagesAsMissing(t *testing.T) {
	var missing *image.RGBA
	v := WorldMapView{
		Background: missing, Ball: missing, Flag: missing, Available: missing, Cross: missing,
		Scroll:  [3]image.Image{missing, missing, missing},
		Pressed: [3]image.Image{missing, missing, missing},
		Missions: []WorldMapMission{{
			Enabled: true, Anchor: image.Pt(300, 300), Marker: missing, Title: "M", Briefing: "B",
		}},
		Selected: 0, Route: []image.Point{{100, 200}}, RouteShown: 1,
		Position: image.Pt(100, 200), Hovered: -1,
	}
	got := ComposeWorldMap(v)
	if got.Bounds() != image.Rect(0, 0, 640, 480) {
		t.Fatalf("typed-nil composition bounds = %v", got.Bounds())
	}
	if c := got.RGBAAt(500, 400); c != worldMapFallback {
		t.Fatalf("typed-nil background = %v, want fallback %v", c, worldMapFallback)
	}
	if c := got.RGBAAt(100, 200); c != (color.RGBA{R: 242, G: 235, B: 211, A: 255}) {
		t.Fatalf("typed-nil current-position flag = %v, want the flag fallback at Position", c)
	}
	if c := got.RGBAAt(300, 300); c != worldMapMark {
		t.Fatalf("typed-nil selected destination cross = %v, want cross fallback %v", c, worldMapMark)
	}
}

// TestWorldMapCompositionPaintsRouteOverMissionMarkers checks the TOWN-120
// (High) paint order: cached mission markers are painted before the route.
// A mission anchor placed exactly on a revealed route point must show the
// route's colour, not the marker's.
func TestWorldMapCompositionPaintsRouteOverMissionMarkers(t *testing.T) {
	var route []image.Point
	for i := 0; i < 9; i++ {
		route = append(route, image.Pt(10+i*20, 5))
	}
	marker := image.NewRGBA(image.Rect(0, 0, 1, 1))
	marker.SetRGBA(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	// The mission anchor sits on route[0], which the ball stride stamps.
	// Position is set away from every route point so the current-position
	// flag (`TOWN-120`) cannot also land on the shared pixel.
	v := WorldMapView{HideScrolls: true,
		Missions:   []WorldMapMission{{Enabled: true, Anchor: route[0], Marker: marker}},
		Selected:   -1,
		Route:      route,
		RouteShown: 9,
		Position:   image.Pt(999, 999),
	}
	got := ComposeWorldMap(v)
	if c := got.RGBAAt(route[0].X, route[0].Y); c != worldMapRoute {
		t.Fatalf("shared pixel = %v, want the route colour painted over the marker", c)
	}
}

// TestWorldMapCompositionStampsBallOnlyAtEveryEighthRevealedCoordinate
// checks TOWN-120 (High): a BallMap.bmp stamp is drawn at every eighth
// revealed route coordinate, not at every one of them. A route of 17 points
// with RouteShown 9 must stamp indices 0 and 8 and no other.
func TestWorldMapCompositionStampsBallOnlyAtEveryEighthRevealedCoordinate(t *testing.T) {
	var route []image.Point
	for i := 0; i < 17; i++ {
		route = append(route, image.Pt(10+i*20, 5))
	}
	v := WorldMapView{HideScrolls: true,
		Missions:   []WorldMapMission{{Enabled: true, Anchor: image.Pt(600, 470)}},
		Selected:   -1,
		Route:      route,
		RouteShown: 9,
		// Position no longer tracks the route's leading revealed point
		// (`TOWN-120`/`TOWN-121`): it stays wherever the party actually is,
		// set here off every route point.
		Position: image.Pt(999, 999),
	}
	got := ComposeWorldMap(v)
	for i, p := range route {
		want := worldMapFallback
		if i == 0 || i == 8 {
			want = worldMapRoute
		}
		if c := got.RGBAAt(p.X, p.Y); c != want {
			t.Fatalf("route point %d (%v) = %v, want %v", i, p, c, want)
		}
	}
}

func TestApplicationDrivesAndDrawsTheWorldMapThroughTown(t *testing.T) {
	town := &fakeWorldMapTown{view: WorldMapView{Missions: []WorldMapMission{{
		Enabled: true, Anchor: image.Pt(300, 300), Region: image.Rect(280, 280, 320, 320),
	}}}}
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the world map town")
	}
	// Each step is spaced past worldMapTickInterval so the cadence gate
	// (`DIV-136`) fires a tick on every one of them, exactly as the
	// pre-cadence test expected once per driven frame.
	now := time.Unix(1_700_000_000, 0)
	step := func(in appInput) {
		now = now.Add(worldMapTickInterval)
		a.step(in, now)
	}
	card := WorldMapCardRect(0)
	p := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true})
	if !reflect.DeepEqual(town.clicks, []image.Point{p}) || town.ticks != 1 {
		t.Fatalf("world map clicks=%v ticks=%d, want one of each", town.clicks, town.ticks)
	}
	step(appInput{Down: true, CursorX: p.X, CursorY: p.Y})
	step(appInput{WheelY: 1, CursorX: p.X, CursorY: p.Y})
	step(appInput{Up: true, CursorX: p.X, CursorY: p.Y})
	step(appInput{Enter: true, CursorX: p.X, CursorY: p.Y})
	if !reflect.DeepEqual(town.moves, []int{1, -1, -1}) || town.chooses != 1 || town.ticks != 5 {
		t.Fatalf("moves=%v chooses=%d ticks=%d", town.moves, town.chooses, town.ticks)
	}
	a.canvas = ebiten.NewImage(frame.W, frame.H)
	a.drawTown()
	if len(town.hovers) != 5 {
		t.Fatalf("hover updates = %d, want one per driven frame", len(town.hovers))
	}
}

// TestWorldMapTickCadenceIsGatedByWallClock checks `DIV-136`: two steps
// closer together than worldMapTickInterval must fire only the first tick, a
// property that could not be seen while WorldMapTick fired unconditionally.
func TestWorldMapTickCadenceIsGatedByWallClock(t *testing.T) {
	town := &fakeWorldMapTown{}
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the world map town")
	}
	now := time.Unix(1_700_000_000, 0)
	a.step(appInput{}, now)
	if town.ticks != 1 {
		t.Fatalf("ticks after the first driven frame = %d, want 1", town.ticks)
	}
	a.step(appInput{}, now.Add(worldMapTickInterval/2))
	if town.ticks != 1 {
		t.Fatalf("ticks after a frame inside the interval = %d, want still 1", town.ticks)
	}
	a.step(appInput{}, now.Add(worldMapTickInterval))
	if town.ticks != 2 {
		t.Fatalf("ticks after a frame at the interval = %d, want 2", town.ticks)
	}
}

// A whole-map overlay keeps its own coordinates, a smaller picture is centred
// on its mission point, and a cached marker without a scroll still paints.
func TestWorldMapCompositionPlacesMarkersAndPaintsOnesWithoutAScroll(t *testing.T) {
	red := color.RGBA{R: 200, A: 255}
	green := color.RGBA{G: 200, A: 255}
	overlay := image.NewRGBA(image.Rect(0, 0, 640, 480))
	overlay.SetRGBA(150, 80, red)
	small := image.NewRGBA(image.Rect(0, 0, 3, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			small.SetRGBA(x, y, green)
		}
	}
	v := WorldMapView{
		HideScrolls: true, Selected: -1, Hovered: -1, Position: image.Pt(5, 5),
		Missions: []WorldMapMission{{Enabled: true, Anchor: image.Pt(181, 96), Marker: overlay}},
		Markers:  []WorldMapMarker{{Anchor: image.Pt(400, 300), Picture: small}},
	}
	got := ComposeWorldMap(v)
	if c := got.RGBAAt(150, 80); c != red {
		t.Fatalf("overlay pixel (150,80) = %v, want %v at the overlay's own coordinates", c, red)
	}
	if c := got.RGBAAt(150-640/2+181, 80-480/2+96); c == red {
		t.Fatal("the overlay was centred on its mission point")
	}
	if c := got.RGBAAt(400, 300); c != green {
		t.Fatalf("scrollless small marker centre = %v, want %v", c, green)
	}
	if c := got.RGBAAt(398, 298); c == green {
		t.Fatal("the small marker spilled outside its 3x3 footprint")
	}
}

// TestWorldMapTaskFlagFollowsHoverThenStaysOnTheChosenTask checks TOWN-528 and
// the owner's observation of the original: while the party stands on the first
// MapObject, Flag1 is drawn at the hovered task before a choice and at the
// chosen task from the choice through the route, whatever the pointer does. Away
// from the first MapObject, and on the homeward trip, it is not drawn. Its frame
// stands with the top-left corner 4 pixels left of and 32 above the anchor.
func TestWorldMapTaskFlagFollowsHoverThenStaysOnTheChosenTask(t *testing.T) {
	missions := []WorldMapMission{
		{Enabled: true, Anchor: image.Pt(200, 300)},
		{Enabled: true, Anchor: image.Pt(400, 300)},
		{Enabled: false, Anchor: image.Pt(500, 300)},
	}
	route := []image.Point{{320, 240}, {260, 270}, {200, 300}}
	for _, tc := range []struct {
		name              string
		hovered, selected int
		atHome, returning bool
		shown             int
		want              int
	}{
		{"nothing hovered or chosen", -1, -1, true, false, 0, -1},
		{"hover before a choice", 1, -1, true, false, 0, 1},
		{"hover over a task that cannot be chosen", 2, -1, true, false, 0, -1},
		{"the choice, before the first step", -1, 0, true, false, 0, 0},
		{"a route step with the pointer on another task", 1, 0, true, false, 1, 0},
		{"the route's last step with the pointer away", -1, 0, true, false, len(route), 0},
		{"after arrival at the task", 1, -1, false, false, 0, -1},
		{"the homeward trip", 0, -1, false, true, 1, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := WorldMapView{Missions: missions, Hovered: tc.hovered, Selected: tc.selected, AtHome: tc.atHome,
				Returning: tc.returning, Route: route, RouteShown: tc.shown, Position: image.Pt(20, 460), HideScrolls: true}
			if got := worldMapTaskFlag(v); got != tc.want {
				t.Fatalf("task flag at mission %d, want %d", got, tc.want)
			}
		})
	}

	flag := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := range flag.Pix {
		flag.Pix[i] = 0xff
	}
	v := WorldMapView{Missions: missions, Hovered: 1, Selected: 0, AtHome: true, Available: flag,
		Route: route, RouteShown: 1, Position: image.Pt(20, 460), HideScrolls: true}
	got := ComposeWorldMap(v)
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if c := got.RGBAAt(196, 268); c != white {
		t.Fatalf("Flag1 corner at the chosen task's anchor + (-4,-32) = %v, want the flag's pixel", c)
	}
	if c := got.RGBAAt(396, 268); c == white {
		t.Fatal("Flag1 drawn at the hovered task while another task is chosen")
	}
}
