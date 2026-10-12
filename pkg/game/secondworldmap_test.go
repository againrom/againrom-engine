package game

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// secondMapState is a second-game map over the installed description's rules
// with fixture sheets: the select-mode party, route-mode party, Cross and the
// two marker rules, each with its own pixel colour.
func secondMapState(t *testing.T, mode string, party image.Point, records ...ui.WorldMapMission) *worldMapState {
	t.Helper()
	d := worldMapDescriptions["rom2"]
	sheet := func(n int, r uint8) []*image.RGBA {
		frames := make([]*image.RGBA, n)
		for i := range frames {
			frames[i] = image.NewRGBA(image.Rect(0, 0, 2, 2))
			for p := 0; p < 4; p++ {
				frames[i].SetRGBA(p%2, p/2, color.RGBA{R: r, G: uint8(i + 1), A: 255})
			}
		}
		return frames
	}
	a := &worldMapAssets{desc: d, modes: make([]worldMapModeArt, len(d.Modes)), cross: sheet(3, 30)}
	a.modes[0].party, a.modes[1].party = sheet(4, 10), sheet(4, 20)
	a.rules = [][]*image.RGBA{sheet(2, 40), sheet(2, 50)}
	s := &worldMapState{assets: a, selected: -1, hovered: -1, current: party, missions: records}
	for i, m := range d.Modes {
		if m.Name == mode {
			s.mode = i
		}
	}
	if d.Modes[s.mode].Name != mode {
		t.Fatalf("no %s mode", mode)
	}
	s.openTravel()
	return s
}

func secondRecord(kind, id int, at image.Point) ui.WorldMapMission {
	return ui.WorldMapMission{Number: id, Kind: kind, Anchor: at, Enabled: true}
}

// The pointer targets the available record nearest it by squared distance
// with no limit, a tie keeping the earlier record; a disabled record and a
// pointer outside the map change nothing; a travel holds the target
// (R2-ENGINE-431). The first game's scroll pick does not target by distance.
func TestSecondMapTargetsTheNearestRecord(t *testing.T) {
	a, b := image.Pt(100, 100), image.Pt(300, 100)
	s := secondMapState(t, secondMapSelect, image.Pt(200, 300), secondRecord(1, 21, a), secondRecord(2, 2, b))
	for _, tc := range []struct {
		at   image.Point
		want int
	}{{image.Pt(110, 90), 0}, {image.Pt(290, 400), 1}, {image.Pt(200, 100), 0}, {image.Pt(639, 479), 1}, {image.Pt(-1, -1), 1}} {
		s.aim(tc.at)
		if s.hovered != tc.want {
			t.Fatalf("pointer %v targets %d, want %d", tc.at, s.hovered, tc.want)
		}
	}
	s.missions[1].Enabled = false
	s.aim(image.Pt(299, 100))
	if s.hovered != 0 {
		t.Fatal("a disabled record was targeted")
	}
	s.missions[1].Enabled = true
	s.aim(image.Pt(299, 100))
	if s.route == nil || s.route[len(s.route)-1] != b {
		t.Fatal("a new target did not preview its route", s.route)
	}
	s.startTravel(s.hovered)
	s.aim(image.Pt(101, 100))
	if s.hovered != 1 || s.selected != 1 {
		t.Fatal("a travel retargeted")
	}

	scroll := secondMapState(t, secondMapSelect, image.Pt(200, 300), secondRecord(1, 21, a), secondRecord(2, 2, b))
	scroll.assets.desc = worldMapDescriptions["rom1"]
	worldMapHover(scrollCampaign{scroll}, image.Pt(110, 90))
	if scroll.hovered == 0 {
		t.Fatal("the scroll pick targeted the record nearest the pointer")
	}
}

// scrollCampaign is a campaign with no effects for the builder's input rules.
type scrollCampaign struct{ s *worldMapState }

func (c scrollCampaign) worldMapNow() *worldMapState { return c.s }
func (c scrollCampaign) WorldMapView() ui.WorldMapView {
	return c.s.view(worldMapViewInput{})
}
func (c scrollCampaign) chooseWorldRecord(i int)       { c.s.startTravel(i) }
func (c scrollCampaign) leaveWorldMap()                {}
func (c scrollCampaign) arriveWorldMap() ui.TownAction { return ui.TownAction{Msg: "arrived"} }

// In select mode with no travel the target carries MissionFlag at (x-5,y-41)
// and every other record not at the party's point FlagOnMap at (x-5,y-37),
// over the route; a travel and route mode draw no record marker
// (R2-ENGINE-430).
func TestSecondMapMarkersByModeAndPhase(t *testing.T) {
	party := image.Pt(200, 300)
	s := secondMapState(t, secondMapSelect, party, secondRecord(1, 21, image.Pt(100, 100)), secondRecord(2, 2, image.Pt(300, 120)), secondRecord(1, 30, party))
	s.aim(image.Pt(100, 101))
	v := s.view(worldMapViewInput{})
	target, other := v.Missions[0], v.Missions[1]
	if !v.MarkersOverRoute || target.Marker != image.Image(s.assets.rules[0][0]) || target.MarkerAt != (ui.WorldMapPlace{Offset: image.Pt(-5, -41), TopLeft: true}) {
		t.Fatalf("target marker %v at %+v", target.Marker, target.MarkerAt)
	}
	if other.Marker != image.Image(s.assets.rules[1][0]) || other.MarkerAt != (ui.WorldMapPlace{Offset: image.Pt(-5, -37), TopLeft: true}) {
		t.Fatalf("other marker %v at %+v", other.Marker, other.MarkerAt)
	}
	if v.Missions[2].Marker != nil {
		t.Fatal("the record at the party's point carries a marker")
	}
	frame := ui.ComposeWorldMap(v)
	if c := frame.RGBAAt(95, 59); c.R != 40 {
		t.Fatalf("MissionFlag pixel at the target's (x-5,y-41) = %v", c)
	}
	if c := frame.RGBAAt(295, 83); c.R != 50 {
		t.Fatalf("FlagOnMap pixel at the other's (x-5,y-37) = %v", c)
	}
	if c := frame.RGBAAt(180, 256); c.R != 10 {
		t.Fatalf("YourFlag pixel at the party's (x-20,y-44) = %v", c)
	}
	s.tick()
	if v := s.view(worldMapViewInput{}); v.Missions[0].Marker != image.Image(s.assets.rules[0][1]) {
		t.Fatal("the marker frame did not advance with the paint")
	}
	s.startTravel(0)
	if v := s.view(worldMapViewInput{}); v.Missions[0].Marker != nil || v.Missions[1].Marker != nil {
		t.Fatal("a travel draws record markers")
	}
	route := secondMapState(t, secondMapRoute, party, secondRecord(1, 10, image.Pt(100, 100)), secondRecord(1, 21, image.Pt(300, 120)))
	route.selected, route.route = -1, nil
	v = route.view(worldMapViewInput{})
	if v.Missions[0].Marker != nil || v.Missions[1].Marker != nil || v.FlagAt != (ui.WorldMapPlace{Offset: image.Pt(-10, -24), TopLeft: true}) {
		t.Fatal("route mode draws record markers or its party flag elsewhere")
	}
}

// Before a travel the whole route previews; a press starts the travel and
// each paint reveals 8 more route points; the travel ends once the reveal
// passes the route and the Cross counter its sheet plus two, and enters the
// record (R2-ENGINE-431, R2-ENGINE-432).
func TestSecondMapTravelRevealsAndEnds(t *testing.T) {
	party, at := image.Pt(100, 300), image.Pt(100, 100)
	s := secondMapState(t, secondMapSelect, party, secondRecord(1, 21, at))
	c := scrollCampaign{s}
	worldMapHover(c, at)
	if v := s.view(worldMapViewInput{}); v.RouteShown != len(v.Route) || len(v.Route) < 2 {
		t.Fatalf("preview shows %d of %d route points", v.RouteShown, len(v.Route))
	}
	worldMapClick(c, at)
	if s.selected != 0 || s.shown != 0 {
		t.Fatal("a press did not start the travel")
	}
	ticks := 0
	for ; ticks < 100; ticks++ {
		if act := worldMapTick(c); act.Msg == "arrived" {
			break
		}
		if s.shown != min(8*(ticks+1), len(s.route)) {
			t.Fatalf("tick %d reveals %d", ticks+1, s.shown)
		}
	}
	reveal := (len(s.route) + 7) / 8
	if want := max(reveal, len(s.assets.cross)+2); ticks+1 != want {
		t.Fatalf("travel ended on tick %d, want %d", ticks+1, want)
	}
}

// A press or a key during a travel skips it: the next paint ends it, at the
// travel's start as well (R2-ENGINE-432). The first game's skip waits for the
// first reveal (DIV-1516).
func TestSecondMapSkipEndsTheTravelOnTheNextPaint(t *testing.T) {
	for _, skip := range []func(worldMapCampaign){
		func(c worldMapCampaign) { worldMapClick(c, image.Pt(5, 5)) },
		func(c worldMapCampaign) { worldMapMove(c, 1) },
		func(c worldMapCampaign) { worldMapChoose(c) },
	} {
		s := secondMapState(t, secondMapSelect, image.Pt(100, 400), secondRecord(1, 21, image.Pt(100, 50)))
		c := scrollCampaign{s}
		worldMapClick(c, image.Pt(100, 50))
		skip(c)
		if act := worldMapTick(c); act.Msg != "arrived" {
			t.Fatal("a skip at the travel's start did not end it on the next paint")
		}
	}
	s := secondMapState(t, secondMapSelect, image.Pt(100, 400), secondRecord(1, 21, image.Pt(100, 50)))
	s.assets.desc = worldMapDescriptions["rom1"]
	s.startTravel(0)
	s.skipTravel()
	if s.shown != 0 {
		t.Fatal("the scroll rules skipped before the first reveal")
	}
}

// Route mode opens travelling to the first record and starts again after a
// travel that did not leave the map (R2-ENGINE-272, R2-ENGINE-432).
func TestSecondMapRouteModeTravelsToTheFirstRecord(t *testing.T) {
	s := secondMapState(t, secondMapRoute, image.Pt(215, 234), secondRecord(1, 10, image.Pt(302, 158)), secondRecord(1, 21, image.Pt(100, 100)))
	if s.selected != 0 || !s.travelling() {
		t.Fatal("route mode did not open travelling to the first record")
	}
	s.selected, s.route = -1, nil
	s.tick()
	if s.selected != 0 || !s.travelling() {
		t.Fatal("route mode did not start its travel again")
	}
	if v := s.view(worldMapViewInput{text: func(string, int) (string, string) { return "name", "body" }}); v.Text != nil {
		t.Fatal("route mode draws text")
	}
}

// The second game's map enters the first available record at the target's
// point, and the campaign leaves through EnterLocation: a town becomes
// current on its square (R2-ENGINE-432, R2-ENGINE-272).
func TestSecondMapArrivalEntersTheFirstRecordAtTheTarget(t *testing.T) {
	f, _, screen := secondCampaignFixture(t, true)
	c := newSecondCampaign()
	c.current, c.available = secondLocation{2, 3}, []secondLocation{{2, 3}, {2, 2}, {1, 30}}
	f.Town.second = c
	if !c.leaveTown() || c.mapOpen.route || c.mapOpen.from != (secondLocation{2, 3}) || !screen.AtWorldMap() {
		t.Fatal("leaving town 3 did not open the map in select mode")
	}
	s := screen.worldMapNow()
	if want, _ := s.assets.desc.point(2, 3); s.current != want {
		t.Fatalf("party point %v, want town 3's %v", s.current, want)
	}
	s.missions[2].Anchor = s.missions[1].Anchor
	s.startTravel(2)
	s.skipTravel()
	if act := screen.WorldMapTick(); act.Open != nil || c.current != (secondLocation{2, 2}) || c.room != secondTownSquare || screen.AtWorldMap() {
		t.Fatalf("arrival entered %v, want town 2, the first record at the target", c.current)
	}
	c.current = secondLocation{2, 1}
	c.bank[769] = 1
	c.available = []secondLocation{{2, 1}, {1, 10}}
	if !c.leaveTown() || !c.mapOpen.route || !reflect.DeepEqual(c.available, []secondLocation{{1, 10}}) {
		t.Fatal("leaving town 1 did not open route mode")
	}
	if s := screen.worldMapNow(); s.modeSpec().Name != secondMapRoute || s.selected != 0 {
		t.Fatal("the route-mode map did not open travelling")
	}
}

// A victory with no town available enters the first available record without
// the map; with a town available, or at the credits stage, the map opens
// (R2-ENGINE-422).
func TestSecondVictoryEntersWithoutTheMapWhenNoTownIsAvailable(t *testing.T) {
	c := newSecondCampaign()
	c.bank[768] = 20
	c.available = []secondLocation{{1, 20}}
	if got := c.enteredWithoutMap(); got != 20 {
		t.Fatalf("no town available enters %d, want 20", got)
	}
	c.available = []secondLocation{{1, 21}, {2, 2}}
	if got := c.enteredWithoutMap(); got != 0 {
		t.Fatal("a town available entered without the map")
	}
	c.available, c.bank[768] = []secondLocation{{1, 20}}, 120
	if got := c.enteredWithoutMap(); got != 0 {
		t.Fatal("the credits stage entered a record")
	}
}

// A record's name and description come from the text source's keys by its
// kind, trimmed by two characters; a record without a key shows no text
// block (R2-ENGINE-434).
func TestSecondMapTextByKind(t *testing.T) {
	a := &worldMapAssets{desc: worldMapDescriptions["rom2"], recordTextTried: true, reads: map[string]int{},
		recordText: "#title21\r\nThe Ford\r\n#briefing21\r\nCross it.\r\n#town2\r\nKaarg\r\n#description2\r\nA town.\r\n"}
	if name, body := a.worldMapRecordText(nil, "1", 21); name != "The Ford" || body != "Cross it." {
		t.Fatalf("mission 21 text %q %q", name, body)
	}
	if name, body := a.worldMapRecordText(nil, "2", 2); name != "Kaarg" || body != "A town." {
		t.Fatalf("town 2 text %q %q", name, body)
	}
	if name, _ := a.worldMapRecordText(nil, "2", 1); name != "" {
		t.Fatal("town 1 has text")
	}
	s := secondMapState(t, secondMapSelect, image.Pt(10, 10), secondRecord(2, 1, image.Pt(100, 100)))
	s.aim(image.Pt(100, 100))
	if v := s.view(worldMapViewInput{text: func(kind string, id int) (string, string) { return a.worldMapRecordText(nil, kind, id) }}); v.Text != nil {
		t.Fatal("a record without a key draws a text block")
	}
}
