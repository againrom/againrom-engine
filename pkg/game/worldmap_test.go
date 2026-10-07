package game

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func worldMapRegistryFixture() []byte {
	return synth.Reg(kindRoot, []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{{Name: "ObjectCount", Kind: kindInt, Int: 4}}},
		{Name: "MapObject1", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{2, 3}},
			{Name: "MapRect", Kind: kindIntArray, Ints: []int32{1, 2, 4, 5}},
		}},
		{Name: "MapObject2", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{20, 30}},
			{Name: "MapRect", Kind: kindIntArray, Ints: []int32{10, 12, -1, 8}},
		}},
		{Name: "MapObject3", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{32, 23}},
			{Name: "MapRect", Kind: kindIntArray, Ints: []int32{30, 20, 8, 7}},
			{Name: "Picture", Kind: 0, Str: "absent-marker"},
		}},
		// The fourth declared object is absent. Its index must remain reserved.
		{Name: "MissionObjects", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mission30", Kind: kindInt, Int: 3},
			{Name: "Mission31", Kind: kindInt, Int: 2},
			{Name: "Mission32", Kind: kindInt, Int: 9},
		}},
	})
}

func TestGlobalMapKeepsDeclaredIndicesAndConvertsRectanglesOnce(t *testing.T) {
	r, err := reg.Parse(worldMapRegistryFixture())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadGlobalMap(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Objects) != 4 {
		t.Fatalf("objects = %d, want declared 4", len(got.Objects))
	}
	if got.Objects[0].Region != image.Rect(1, 2, 5, 7) {
		t.Fatalf("MapObject1 rectangle = %v, want (1,2)-(5,7)", got.Objects[0].Region)
	}
	if got.Objects[1].Valid || got.Objects[1].Problem == "" {
		t.Fatalf("negative-width object = %+v, want invalid with a reason", got.Objects[1])
	}
	if got.Objects[2].Point != image.Pt(32, 23) || got.Objects[3].Valid {
		t.Fatalf("later object identities shifted: object3=%+v object4=%+v", got.Objects[2], got.Objects[3])
	}
	wantMappings := map[int]int{30: 2, 31: 1, 32: 8}
	if !reflect.DeepEqual(got.Missions, wantMappings) {
		t.Fatalf("mission mappings = %v, want %v", got.Missions, wantMappings)
	}
}

// worldMapPalette is a 3-entry palette matching PathMap.bmp's own contract:
// index 0 impassable, index 1 corridor, index 2 node (`TOWN-116`, High).
var worldMapPalette = color.Palette{color.Black, color.White, color.RGBA{R: 255, A: 255}}

func TestWorldMapNodeGraphFindsEveryNodeAndItsDirectEdges(t *testing.T) {
	// Node A sits between B (east) and C (west) on the same row. Opposite
	// leaving directions keep the two corridors clear of each other at A,
	// which the eight-neighbour walk would otherwise merge. The edge point
	// lists are asserted as literals, not derived from the graph builder.
	mask := image.NewPaletted(image.Rect(0, 0, 13, 1), worldMapPalette)
	a, b, c := image.Pt(6, 0), image.Pt(10, 0), image.Pt(2, 0)
	mask.SetColorIndex(a.X, a.Y, 2)
	mask.SetColorIndex(b.X, b.Y, 2)
	mask.SetColorIndex(c.X, c.Y, 2)
	for x := 7; x <= 9; x++ {
		mask.SetColorIndex(x, 0, 1)
	}
	for x := 3; x <= 5; x++ {
		mask.SetColorIndex(x, 0, 1)
	}

	g := buildWorldMapNodeGraph(mask)
	if g == nil || len(g.nodes) != 3 {
		t.Fatalf("graph = %+v, want 3 nodes", g)
	}
	ai, bi, ci := g.index[a], g.index[b], g.index[c]

	wantAB := []image.Point{{7, 0}, {8, 0}, {9, 0}}
	wantAC := []image.Point{{5, 0}, {4, 0}, {3, 0}}
	foundAB, foundAC := false, false
	for _, e := range g.edges[ai] {
		switch e.to {
		case bi:
			if !reflect.DeepEqual(e.points, wantAB) {
				t.Fatalf("A-B edge points = %v, want %v", e.points, wantAB)
			}
			foundAB = true
		case ci:
			if !reflect.DeepEqual(e.points, wantAC) {
				t.Fatalf("A-C edge points = %v, want %v", e.points, wantAC)
			}
			foundAC = true
		}
	}
	if !foundAB || !foundAC {
		t.Fatalf("node A edges = %+v, want edges to both B and C", g.edges[ai])
	}
	// B's own walk independently discovers the same corridor, reversed.
	foundBA := false
	for _, e := range g.edges[bi] {
		if e.to == ai {
			want := []image.Point{{9, 0}, {8, 0}, {7, 0}}
			if !reflect.DeepEqual(e.points, want) {
				t.Fatalf("B-A edge points = %v, want %v", e.points, want)
			}
			foundBA = true
		}
	}
	if !foundBA {
		t.Fatalf("node B edges = %+v, want an edge back to A", g.edges[bi])
	}
}

// TestWorldMapRoutePrefersLeastAccumulatedLengthOverFewestHops builds a mask
// where the direct one-hop edge between A and B is a 63-pixel detour loop
// (weight 64), and a two-hop chain through M is 33 corridor pixels total
// (weight 35). The route must take the
// two-hop chain (`TOWN-119`, High: least accumulated segment length, not a
// straight line and not a per-frame pixel search) even though it is not the
// fewest-hops choice. Every corridor leaves its shared node in a direction
// that stays clear of the node's other corridor by construction: opposite
// (A-M west vs the detour's east) or separated by two rows/columns
// everywhere else, which the eight-neighbour walk would otherwise merge.
func TestWorldMapRoutePrefersLeastAccumulatedLengthOverFewestHops(t *testing.T) {
	mask := image.NewPaletted(image.Rect(0, 0, 35, 35), worldMapPalette)
	a, m, b := image.Pt(20, 0), image.Pt(5, 0), image.Pt(5, 20)
	mask.SetColorIndex(a.X, a.Y, 2)
	mask.SetColorIndex(m.X, m.Y, 2)
	mask.SetColorIndex(b.X, b.Y, 2)

	line := func(x0, x1, y int) {
		for x := x0; x <= x1; x++ {
			mask.SetColorIndex(x, y, 1)
		}
	}
	column := func(x, y0, y1 int) {
		for y := y0; y <= y1; y++ {
			mask.SetColorIndex(x, y, 1)
		}
	}

	// A-M: straight west from A, 14 corridor pixels, weight 15.
	line(6, 19, 0)
	// M-B: one south-west step off M's row, then straight south, 19
	// corridor pixels, weight 20. Indirect total: 35.
	mask.SetColorIndex(4, 1, 1)
	column(4, 2, 19)
	// Direct A-B: east, south, west, north around a 63-pixel loop that
	// stays at least two cells from both A-M (row 0) and M-B (column 4)
	// throughout. Weight 64.
	line(21, 30, 0)
	column(30, 1, 25)
	line(6, 29, 25)
	column(5, 21, 24)

	g := buildWorldMapNodeGraph(mask)
	if g == nil || len(g.nodes) != 3 {
		t.Fatalf("graph = %+v, want 3 nodes", g)
	}
	route, ok := g.Route(a, b)
	if !ok {
		t.Fatal("no route found between A and B")
	}
	var want []image.Point
	want = append(want, a)
	for x := 19; x >= 6; x-- {
		want = append(want, image.Pt(x, 0))
	}
	want = append(want, m)
	want = append(want, image.Pt(4, 1))
	for y := 2; y <= 19; y++ {
		want = append(want, image.Pt(4, y))
	}
	want = append(want, b)
	if !reflect.DeepEqual(route, want) {
		t.Fatalf("route = %v (%d points), want the 35-weight two-hop chain through M (%d points)", route, len(route), len(want))
	}
}

func TestWorldMapRouteFallsBackWhenEndpointIsNotANodeOrMaskIsMissing(t *testing.T) {
	mask := image.NewPaletted(image.Rect(0, 0, 20, 12), worldMapPalette)
	mask.SetColorIndex(0, 0, 2)
	mask.SetColorIndex(10, 0, 2)
	for x := 1; x <= 9; x++ {
		mask.SetColorIndex(x, 0, 1)
	}
	g := buildWorldMapNodeGraph(mask)

	// DIV-129: the destination is not itself a node pixel. No shipped
	// registry MapPoint exercises this, and the fallback is a straight
	// sampled line.
	got := worldMapRoute(g, image.Pt(0, 0), image.Pt(20, 0), 4)
	if want := []image.Point{{0, 0}, {4, 0}, {8, 0}, {12, 0}, {16, 0}, {20, 0}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("off-node route = %v, want the sampled-line fallback %v", got, want)
	}

	fallback := worldMapRoute(nil, image.Pt(0, 0), image.Pt(20, 0), 10)
	if want := []image.Point{{0, 0}, {10, 0}, {20, 0}}; !reflect.DeepEqual(fallback, want) {
		t.Fatalf("missing-graph route = %v, want %v", fallback, want)
	}
}

func worldMapFixtureFS(t *testing.T) *vfs.FS {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, files []synth.File) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, synth.Archive(files), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	bg := image.NewRGBA(image.Rect(0, 0, 640, 480))
	// Both registered endpoints (the town start (2,3) and mission 30's
	// object at (32,23)) sit on graph nodes, matching TOWN-116: every
	// registry MapPoint lands on an index-2 pixel.
	mask := image.NewPaletted(image.Rect(0, 0, 640, 480), worldMapPalette)
	for x := 2; x <= 32; x++ {
		mask.SetColorIndex(x, 23, 1)
	}
	for y := 3; y <= 23; y++ {
		mask.SetColorIndex(2, y, 1)
	}
	mask.SetColorIndex(2, 3, 2)
	mask.SetColorIndex(32, 23, 2)
	main := write(MainArchive, []synth.File{
		{Path: "graphics/global.map/gmap.bmp", Data: synth.BMP24(bg)},
		{Path: "text/battle/m30/title.txt", Data: []byte("A SHIPPED TITLE")},
		{Path: "text/battle/m30/briefmap.txt", Data: []byte("A shipped briefing")},
	})
	graphics := write(GraphicsArchive, []synth.File{{Path: "global.map/pathmap.bmp", Data: synth.BMP8(mask)}})
	scenario := write(ScenarioArchive, []synth.File{{Path: "globalmap.reg", Data: worldMapRegistryFixture()}})
	fsys, err := vfs.Open([]string{main, graphics, scenario}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

func TestGatesShowLiveScrollTextSelectRouteAndUseTheMissionDoor(t *testing.T) {
	c := townCampaign(t)
	town := NewTown(c)
	town.Arrive()
	town.announceMission(30)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: worldMapFixtureFS(t)}, Font: resolved(missionFont(), nil)}, CampaignSession: CampaignSession{Town: town}}
	before, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	beforeBytes, err := EncodeSave(before, label)
	if err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	s.Choose(3)

	view := s.WorldMapView()
	if len(view.Missions) != 1 {
		t.Fatalf("mission scrolls = %d, want one live Town.Available mission", len(view.Missions))
	}
	m := view.Missions[0]
	if !m.Enabled || m.Number != 30 || m.Title != "A SHIPPED TITLE" || m.Briefing != "A shipped briefing" || m.Payment != 1000 {
		t.Fatalf("mission scroll = %+v", m)
	}
	if m.Marker != nil {
		t.Fatalf("missing optional mission marker crossed the adapter as %T", m.Marker)
	}

	// A scroll release selects and constructs the route. There is no second
	// click that opens it any more (`TOWN-121`, High; `DIV-135` closed):
	// arrival opens it by itself.
	card := ui.WorldMapCardRect(0)
	centre := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	if act := s.WorldMapClick(centre); act.Open != nil {
		t.Fatal("the scroll release opened the mission instead of selecting it")
	}
	view = s.WorldMapView()
	if view.Selected != 0 || len(view.Route) < 2 || view.Route[0] != image.Pt(2, 3) || view.Route[len(view.Route)-1] != image.Pt(32, 23) {
		t.Fatalf("selected route = selected %d, %v", view.Selected, view.Route)
	}
	if !view.AtHome {
		t.Fatal("the party's own field did not read as home before travel began")
	}
	// The graph route is pixel-exact along the mask's own L-shaped
	// corridor, not a ten-pixel sample of a straight line between the
	// endpoints (`TOWN-119`, High).
	if want := 1 + (23 - 3) + (32 - 2); len(view.Route) != want {
		t.Fatalf("route length = %d, want %d corridor pixels from (2,3) to (32,23)", len(view.Route), want)
	}
	for _, p := range view.Route {
		if p.X != 2 && p.Y != 23 {
			t.Fatalf("route point %v left the mask's own L-shaped corridor", p)
		}
	}
	routeLen := len(view.Route)
	beforeRoute := view.RouteShown
	if act := s.WorldMapTick(); act.Open != nil {
		t.Fatal("the first tick opened the mission before the route finished revealing")
	}
	if after := s.WorldMapView().RouteShown; after != beforeRoute+8 {
		t.Fatalf("route animation prefix = %d after tick, want %d (TOWN-120: 8 coordinates per paint)", after, beforeRoute+8)
	}
	// A click that misses every scroll fast-forwards the rest of the reveal
	// (`TOWN-121`'s own skip arm, High); the following tick opens the
	// destination.
	miss := image.Pt(2, 2)
	if _, ok := ui.WorldMapCardAt(s.WorldMapView(), miss); ok {
		t.Fatal("setup: the chosen miss point hits a card")
	}
	s.WorldMapClick(miss)
	if shown := s.WorldMapView().RouteShown; shown != routeLen {
		t.Fatalf("miss click did not skip to the route's own end: shown %d, want %d", shown, routeLen)
	}
	act := s.WorldMapTick()
	if act.Open == nil {
		t.Fatal("the completing tick did not open the destination mission")
	}
	view = s.WorldMapView()
	if view.Selected != -1 || len(view.Route) != 0 {
		t.Fatalf("arrival left selected=%d route=%v, want both cleared", view.Selected, view.Route)
	}
	if view.Position != image.Pt(32, 23) {
		t.Fatalf("arrival position = %v, want the destination (32,23)", view.Position)
	}

	// Away from home no scroll is actionable (TOWN-122). Homeward arrival
	// changes presentation, not purse, party or accepted offers.
	gold, available := town.Gold(), town.Available()
	townCard := ui.WorldMapCardRect(-1)
	s.WorldMapClick(image.Pt((townCard.Min.X+townCard.Max.X)/2, (townCard.Min.Y+townCard.Max.Y)/2))
	if s.AtTownSquare() {
		t.Fatal("away-from-home town scroll remained actionable")
	}
	s.beginWorldMapReturn(30)
	s.WorldMapTick()
	s.WorldMapClick(image.Pt(2, 2))
	s.WorldMapTick()
	if !s.AtTownSquare() || town.Gold() != gold || !reflect.DeepEqual(town.Available(), available) {
		t.Fatalf("town scroll changed campaign state: square=%v gold=%d available=%v", s.AtTownSquare(), town.Gold(), town.Available())
	}
	after, afterLabel, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	afterBytes, err := EncodeSave(after, afterLabel)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{30}; !reflect.DeepEqual(after.WorldSelectedOnce, want) {
		t.Fatalf("selected marker history = %v, want %v", after.WorldSelectedOnce, want)
	}
	withoutHistory := after
	withoutHistory.WorldSelectedOnce = nil
	if !reflect.DeepEqual(before, withoutHistory) {
		t.Fatal("world-map visit changed saved state beyond the selected-marker history")
	}
	if reflect.DeepEqual(beforeBytes, afterBytes) {
		t.Fatal("selecting a world-map mission did not change the encoded snapshot")
	}
}

// 1009, DIV-105: `TOWN-118` (High) states a mission is selected by clicking
// its scroll entry, and the map-rectangle override only hit-tests a region —
// it calls neither mission selection nor route construction. A click inside
// a mission's own map region must select and open nothing; the card gesture
// is unchanged.
//
// The state is built directly rather than through a loaded install: the map
// region (400,400)-(440,440) is chosen clear of every card rectangle
// (WorldMapCardRect covers at most two rows starting at y=8), so a hit there
// cannot also be a card hit — the one confound that would make this pass
// for the wrong reason.
func TestWorldMapRegionClickSelectsAndOpensNothing(t *testing.T) {
	s := (&FrontEnd{}).bindTown(&townScreen{worldMap: &worldMapState{
		selected: -1, hovered: -1, assets: &worldMapAssets{},
		current: image.Pt(320, 240),
		missions: []ui.WorldMapMission{
			{Number: 30, Title: "T", Enabled: true, Region: image.Rect(400, 400, 440, 440), Anchor: image.Pt(420, 420)},
		},
	}})

	inside := image.Pt(420, 420)
	if _, ok := ui.WorldMapRegionAt(s.WorldMapView(), inside); !ok {
		t.Fatal("setup: the probe point does not hit the fixture's own map region")
	}
	if act := s.WorldMapClick(inside); act.Open != nil {
		t.Fatal("a click inside the mission's own map region opened it")
	}
	if got := s.WorldMapView().Selected; got != -1 {
		t.Fatalf("a click inside the mission's own map region selected %d, want none", got)
	}

	// The scroll card still selects and, on a second release, opens — the
	// override changed nothing about the card's own gesture.
	card := ui.WorldMapCardRect(0)
	center := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	if act := s.WorldMapClick(center); act.Open != nil {
		t.Fatal("the first scroll release opened the mission instead of selecting it")
	}
	if got := s.WorldMapView().Selected; got != 0 {
		t.Fatalf("the scroll card selected %d, want 0", got)
	}
}

// TestWorldMapMarkerPaintsOnlyAfterSelection checks `DIV-128`
// (`TOWN-123`/`TOWN-040`, High): a picture-bearing mission's own marker
// paints only once it has been selected on the map at least once. The
// picture decode itself is stubbed through markerTried/markers so the test
// exercises only the gate WorldMapView applies, not spr256 decoding.
func TestWorldMapMarkerPaintsOnlyAfterSelection(t *testing.T) {
	pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
	a := &worldMapAssets{
		data: &globalMapData{Objects: []globalMapObject{
			{Point: image.Pt(5, 5), Valid: true},
			{Point: image.Pt(9, 9), Valid: true, Picture: "boss"},
		}, Missions: map[int]int{30: 1}},
		markers: map[int]*image.RGBA{30: pic}, markerTried: map[int]bool{30: true},
	}
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: worldMapFixtureFS(t)}}}
	s := f.bindTown(&townScreen{worldMap: &worldMapState{
		assets: a, selected: -1, hovered: -1, current: a.data.Objects[0].Point,
		missions: []ui.WorldMapMission{{Number: 30, Enabled: true, Object: 1, Anchor: a.data.Objects[1].Point}},
	}})
	if v := s.WorldMapView(); v.Missions[0].Marker != nil {
		t.Fatal("marker painted before the mission was ever selected")
	}
	s.markWorldSelected(30)
	if v := s.WorldMapView(); v.Missions[0].Marker == nil {
		t.Fatal("marker did not paint once the mission was selected")
	}
}

// TestMarkWorldSelectedKeepsOnlyValidPictureMappings checks the sole writer of
// the persisted marker-history set. A mission number is not enough: its
// global-map row must resolve to a valid object carrying actual marker art.
func TestMarkWorldSelectedKeepsOnlyValidPictureMappings(t *testing.T) {
	tests := []struct {
		name    string
		mission int
		data    *globalMapData
		want    bool
	}{
		{name: "missing data", mission: 10},
		{name: "missing mission mapping", mission: 10, data: &globalMapData{Missions: map[int]int{}}},
		{name: "negative object index", mission: 10, data: &globalMapData{Missions: map[int]int{10: -1}}},
		{name: "object index past population", mission: 10, data: &globalMapData{Missions: map[int]int{10: 1}, Objects: []globalMapObject{{Valid: true, Picture: "marker"}}}},
		{name: "invalid object", mission: 10, data: &globalMapData{Missions: map[int]int{10: 0}, Objects: []globalMapObject{{Picture: "marker"}}}},
		{name: "empty picture", mission: 10, data: &globalMapData{Missions: map[int]int{10: 0}, Objects: []globalMapObject{{Valid: true}}}},
		{name: "nothing picture", mission: 10, data: &globalMapData{Missions: map[int]int{10: 0}, Objects: []globalMapObject{{Valid: true, Picture: "NoThInG"}}}},
		{name: "valid picture", mission: 10, data: &globalMapData{Missions: map[int]int{10: 0}, Objects: []globalMapObject{{Valid: true, Picture: "marker"}}}, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &townScreen{worldMap: &worldMapState{assets: &worldMapAssets{data: tc.data}}}
			s.markWorldSelected(tc.mission)
			got := s.worldSelectedOnce[tc.mission]
			if got != tc.want {
				t.Fatalf("worldSelectedOnce[%d] = %v, want %v; set=%v", tc.mission, got, tc.want, s.worldSelectedOnce)
			}
			if !tc.want && s.worldSelectedOnce != nil {
				t.Fatalf("rejected mapping allocated marker history %v", s.worldSelectedOnce)
			}
		})
	}
}

// TestWorldMapNextSelectionStartsFromCurrentPosition checks the constraint
// that the party's own position must actually move: selecting a destination
// routes from worldMapState.current, not from the town's own first
// MapObject on every selection.
func TestWorldMapNextSelectionStartsFromCurrentPosition(t *testing.T) {
	s := (&FrontEnd{}).bindTown(&townScreen{worldMap: &worldMapState{
		selected: -1, hovered: -1, assets: &worldMapAssets{}, current: image.Pt(50, 50),
		missions: []ui.WorldMapMission{{Number: 1, Enabled: true, Anchor: image.Pt(60, 60)}},
	}})
	s.selectWorldMission(0)
	if len(s.worldMap.route) == 0 || s.worldMap.route[0] != image.Pt(50, 50) {
		t.Fatalf("route = %v, want it to start from the party's own current position (50,50)", s.worldMap.route)
	}
}

// TestWorldMapArrivalClearsSelectionMovesPositionAndOpensWithNoSecondClick
// checks `TOWN-121` (High; `DIV-135` closed): the tick that finishes the
// reveal opens the destination by itself, and leaves the party's own
// position at the destination with the route and selection cleared.
func TestWorldMapArrivalClearsSelectionMovesPositionAndOpensWithNoSecondClick(t *testing.T) {
	s := (&FrontEnd{}).bindTown(&townScreen{worldMap: &worldMapState{
		selected: 0, hovered: -1, assets: &worldMapAssets{}, current: image.Pt(1, 1), shown: 1,
		route:    []image.Point{{1, 1}},
		missions: []ui.WorldMapMission{{Number: 42, Enabled: true, Anchor: image.Pt(9, 9)}},
	}})
	act := s.WorldMapTick()
	if act.Open == nil {
		t.Fatal("the completing tick did not open the destination mission")
	}
	if s.worldMap.selected != -1 || len(s.worldMap.route) != 0 {
		t.Fatalf("selected=%d route=%v after arrival, want both cleared", s.worldMap.selected, s.worldMap.route)
	}
	if s.worldPosition != image.Pt(9, 9) || s.worldMap.current != image.Pt(9, 9) {
		t.Fatalf("worldPosition=%v current=%v after arrival, want both (9,9)", s.worldPosition, s.worldMap.current)
	}
}

// TestWorldMapRepeatedScrollClickKeepsTravelRunning checks DIV-1479. Two more
// clicks on the travelled mission's scroll between every reveal tick leave the
// route and its revealed prefix as they are, and the party arrives and the
// mission opens on the tick one click alone gives. The first click and a click
// on another mission's scroll behave as before: each starts travel from the
// party's own position with nothing revealed.
func TestWorldMapRepeatedScrollClickKeepsTravelRunning(t *testing.T) {
	home := image.Pt(320, 240) // worldMapHomePoint with no registry loaded
	anchors := map[int]image.Point{30: image.Pt(0, 0), 40: image.Pt(620, 470)}
	newScreen := func() *townScreen {
		return (&FrontEnd{}).bindTown(&townScreen{worldMap: &worldMapState{
			selected: -1, hovered: -1, assets: &worldMapAssets{}, current: home,
			missions: []ui.WorldMapMission{
				{Number: 30, Title: "A", Enabled: true, Anchor: anchors[30]},
				{Number: 40, Title: "B", Enabled: true, Anchor: anchors[40]},
			},
		}})
	}
	scroll := func(slot int) image.Point {
		r := ui.WorldMapCardRect(slot)
		return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	}
	start := func(s *townScreen, slot, mission int) []image.Point {
		t.Helper()
		if act := s.WorldMapClick(scroll(slot)); act.Open != nil {
			t.Fatalf("the click on mission %d's scroll opened it before travel", mission)
		}
		v := s.WorldMapView()
		if v.Selected != slot || v.RouteShown != 0 || len(v.Route) < 2 || v.Route[0] != home || v.Route[len(v.Route)-1] != anchors[mission] {
			t.Fatalf("mission %d's scroll click: selected %d, shown %d, route %v", mission, v.Selected, v.RouteShown, v.Route)
		}
		return v.Route
	}
	arrive := func(s *townScreen, between func(tick int)) (int, ui.TownAction) {
		t.Helper()
		for tick := 1; tick <= 100; tick++ {
			if act := s.WorldMapTick(); act.Open != nil {
				return tick, act
			}
			between(tick)
		}
		t.Fatal("the travel never arrived")
		return 0, ui.TownAction{}
	}

	control := newScreen()
	route := start(control, 0, 30)
	want := (len(route) + worldMapRouteReveal - 1) / worldMapRouteReveal
	if ticks, act := arrive(control, func(int) {}); ticks != want || act.Msg != "travelling to mission 30" {
		t.Fatalf("one click: arrival on tick %d with %q, want tick %d to mission 30", ticks, act.Msg, want)
	}

	repeated := newScreen()
	start(repeated, 0, 30)
	clicks := 0
	ticks, act := arrive(repeated, func(tick int) {
		for range 2 {
			before := repeated.WorldMapView()
			if act := repeated.WorldMapClick(scroll(0)); act.Open != nil {
				t.Fatal("a repeated scroll click opened the mission")
			}
			clicks++
			after := repeated.WorldMapView()
			if after.RouteShown != before.RouteShown || after.Selected != 0 || !reflect.DeepEqual(after.Route, route) {
				t.Fatalf("repeated click %d after tick %d restarted the travel: shown %d -> %d, route %d -> %d points",
					clicks, tick, before.RouteShown, after.RouteShown, len(route), len(after.Route))
			}
		}
		if got, want := repeated.WorldMapView().RouteShown, min(tick*worldMapRouteReveal, len(route)); got != want {
			t.Fatalf("after tick %d the route shows %d coordinates, want %d", tick, got, want)
		}
	})
	if ticks != want || act.Msg != "travelling to mission 30" || repeated.worldPosition != anchors[30] {
		t.Fatalf("after %d repeated clicks: arrival on tick %d at %v with %q, want tick %d at %v", clicks, ticks, repeated.worldPosition, act.Msg, want, anchors[30])
	}

	other := newScreen()
	start(other, 0, 30)
	other.WorldMapTick()
	other.WorldMapTick()
	start(other, 1, 40)
	if _, act := arrive(other, func(int) {}); act.Msg != "travelling to mission 40" || other.worldPosition != anchors[40] {
		t.Fatalf("after another mission's scroll: arrival at %v with %q, want mission 40 at %v", other.worldPosition, act.Msg, anchors[40])
	}
}

func TestWorldMapManifestCachesSuccessfulAndMissingNodesAcrossVisits(t *testing.T) {
	c := townCampaign(t)
	town := NewTown(c)
	town.Arrive()
	town.announceMission(30)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: worldMapFixtureFS(t)}}, CampaignSession: CampaignSession{Town: town}}
	s := f.TownScreen().(*townScreen)
	s.Choose(3)
	a := f.worldMapCache
	first := make(map[string]int, len(a.Value().reads))
	for path, count := range a.Value().reads {
		first[path] = count
	}
	s.Back()
	s.Choose(3)
	if !reflect.DeepEqual(a.Value().reads, first) {
		t.Fatalf("second visit retried manifest nodes: before=%v after=%v", first, a.Value().reads)
	}
}

func TestWorldMapAdapterNormalisesEveryMissingOptionalImage(t *testing.T) {
	c := townCampaign(t)
	town := NewTown(c)
	town.Arrive()
	town.announceMission(30)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: town}}
	s := f.TownScreen().(*townScreen)
	s.Choose(3)
	v := s.WorldMapView()
	for name, pic := range map[string]image.Image{
		"background": v.Background, "ball": v.Ball,
		"flag": v.Flag, "available": v.Available, "cross": v.Cross,
	} {
		if pic != nil {
			t.Fatalf("missing %s crossed the adapter as a non-nil interface: %T", name, pic)
		}
	}
	for i, pic := range append(v.Scroll[:], v.Pressed[:]...) {
		if pic != nil {
			t.Fatalf("missing scroll image %d crossed the adapter as a non-nil interface: %T", i, pic)
		}
	}
	if len(v.Missions) != 1 || v.Missions[0].Marker != nil {
		t.Fatalf("missing mission marker crossed the adapter: %+v", v.Missions)
	}
	frame := ui.ComposeWorldMap(v)
	if frame.Bounds() != image.Rect(0, 0, 640, 480) {
		t.Fatalf("missing-manifest fallback bounds = %v", frame.Bounds())
	}
}

func TestWorldMapKeyboardMovementWrapsAndSkipsDisabledMissions(t *testing.T) {
	s := &townScreen{worldMap: &worldMapState{
		assets:  &worldMapAssets{},
		current: image.Pt(320, 240),
		missions: []ui.WorldMapMission{
			{Enabled: true, Anchor: image.Pt(1, 1)},
			{Enabled: false, Anchor: image.Pt(2, 2)},
			{Enabled: true, Anchor: image.Pt(3, 3)},
		},
		selected: -1,
	}}
	s.WorldMapMove(-1)
	if s.worldMap.selected != 2 {
		t.Fatalf("backward move from no selection = %d, want last enabled mission", s.worldMap.selected)
	}
	s.WorldMapMove(1)
	if s.worldMap.selected != 0 {
		t.Fatalf("forward wrapped selection = %d, want first enabled mission", s.worldMap.selected)
	}
}

func TestTownScreenImplementsTheWorldMapPresentationSeam(t *testing.T) {
	var _ ui.TownWorldMapScreen = (*townScreen)(nil)
}

// The digest is the committed replacement for the throwaway probes that
// produced 1010's corpus counts. Every number is written out here, not
// derived from the graph the digest itself builds.
func TestWorldMapGraphDigestCountsPixelsNodesEdgesAndMapPoints(t *testing.T) {
	// Three nodes on one row: C(2,0) - A(6,0) - B(10,0), each pair joined by
	// three corridor pixels. 13x2 pixels: 6 corridor, 3 node, 17 impassable.
	mask := image.NewPaletted(image.Rect(0, 0, 13, 2), worldMapPalette)
	for _, p := range []image.Point{{X: 2}, {X: 6}, {X: 10}} {
		mask.SetColorIndex(p.X, p.Y, 2)
	}
	for _, x := range []int{3, 4, 5, 7, 8, 9} {
		mask.SetColorIndex(x, 0, 1)
	}
	// Two MapPoints: one on node A, one on a corridor pixel.
	points := []image.Point{{X: 6}, {X: 4}}

	got := worldMapGraphDigest(mask, points)
	want := "world map graph: 6 corridor, 3 node, 17 impassable pixel(s); 3 node(s), 4 directed edge(s), 0 duplicate edge(s); 3 of 3 node(s) reachable from node 0; 1 of 2 MapPoint(s) on a node"
	if got != want {
		t.Errorf("digest =\n%s\nwant\n%s", got, want)
	}

	if got := worldMapGraphDigest(nil, points); got != "world map graph: no path mask" {
		t.Errorf("absent mask digest = %q", got)
	}
}

// TestWorldMapCriterionDigestCountsPairsWhereTheTwoReadingsDisagree builds a
// graph on which the two readings of `TOWN-119` must choose differently, so
// the digest reports a non-zero count and cannot pass by reporting zero.
//
// Three nodes on one row, A(4,4) - M(14,4) - B(24,4), each neighbouring pair
// joined by a nine-pixel straight corridor. A and B are ALSO joined directly,
// by a thirty-seven-pixel corridor that leaves A leftward, runs along the
// bottom row and returns up the right edge. Accumulated segment length prefers
// A-M-B (10 + 10) over the direct segment (38); one-per-segment prefers the
// direct segment (one hop) over A-M-B (two). The two corridors leaving A go in
// OPPOSITE directions and every pair of distinct corridors keeps a Chebyshev
// gap of at least two, so the eight-neighbour walk cannot cross between them.
func TestWorldMapCriterionDigestCountsPairsWhereTheTwoReadingsDisagree(t *testing.T) {
	mask := image.NewPaletted(image.Rect(0, 0, 26, 9), worldMapPalette)
	nodes := []image.Point{{X: 4, Y: 4}, {X: 14, Y: 4}, {X: 24, Y: 4}}
	for _, p := range nodes {
		mask.SetColorIndex(p.X, p.Y, 2)
	}
	corridor := func(pts ...image.Point) {
		for _, p := range pts {
			mask.SetColorIndex(p.X, p.Y, 1)
		}
	}
	for x := 5; x <= 13; x++ {
		corridor(image.Pt(x, 4))
	}
	for x := 15; x <= 23; x++ {
		corridor(image.Pt(x, 4))
	}
	for x := 0; x <= 3; x++ {
		corridor(image.Pt(x, 4))
	}
	for y := 5; y <= 8; y++ {
		corridor(image.Pt(0, y))
	}
	for x := 1; x <= 25; x++ {
		corridor(image.Pt(x, 8))
	}
	for y := 4; y <= 7; y++ {
		corridor(image.Pt(25, y))
	}

	// The fixture is only worth anything if the two searches really do split
	// on it, so assert the chains themselves before asserting the count.
	g := buildWorldMapNodeGraph(mask)
	byLength, ok := g.chain(0, 2, shippedSegmentWeight)
	if !ok || len(byLength) != 3 {
		t.Fatalf("accumulated-length chain A->B = %v (ok %v), want the three-node chain through M", byLength, ok)
	}
	byHops, ok := g.chain(0, 2, hopWeight)
	if !ok || len(byHops) != 2 {
		t.Fatalf("one-per-segment chain A->B = %v (ok %v), want the two-node direct chain", byHops, ok)
	}

	got := worldMapCriterionDigest(mask, nodes)
	want := "world map criterion: 3 MapPoint(s) on nodes, 6 ordered pair(s); accumulated segment length and one-per-segment select different chains for 2 ordered pair(s), 1 unordered"
	if got != want {
		t.Errorf("criterion digest =\n%s\nwant\n%s", got, want)
	}

	if got := worldMapCriterionDigest(nil, nodes); got != "world map criterion: no graph" {
		t.Errorf("absent mask digest = %q", got)
	}
}
