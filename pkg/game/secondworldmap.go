package game

import "againrom/pkg/ui"

// The second game's campaign is a world-map campaign: the map opens whenever
// the party stands at no location, after leaving a town or after a victory
// (R2-ENGINE-272, R2-ENGINE-422, R2-ENGINE-428), and leaves through the
// record the travel ends at.

// secondMapOpening is one opening of the map: the record the party left,
// whose point is the party's point on the map (R2-SESSION-177; DIV-3071), and
// the route mode leaving town 1 selects (R2-ENGINE-272).
type secondMapOpening struct {
	from  partyPlace
	route bool
}

func (c *secondCampaign) openMap(from partyPlace) {
	c.mapOpen = &secondMapOpening{from: from, route: from == (partyPlace{kind: placeTown, id: 1})}
}

// enteredWithoutMap is the mission a victory enters without the map: the
// first available record when no town is available and the stage is below
// the credits (R2-ENGINE-422 step 5); zero when the map opens.
func (c *secondCampaign) enteredWithoutMap() int {
	if c.bank[768] >= 120 || len(c.available) == 0 || c.available[0].kind != placeMission {
		return 0
	}
	for _, l := range c.available {
		if l.kind == placeTown {
			return 0
		}
	}
	return c.available[0].id
}

// secondMapModes are the description modes the campaign opens its map in.
const (
	secondMapSelect = "select"
	secondMapRoute  = "route"
)

// enterLocation is EnterLocation and the 0x468 arm: a town becomes current on
// its square; a mission opens (R2-ENGINE-272).
func (t *townScreen) enterLocation(l partyPlace) ui.TownAction {
	c := t.state()
	if l.kind == placeTown {
		c.current, c.room, c.mapOpen = l, secondTownSquare, nil
		// Every location entry clears its slots; a mission's entry clears
		// them as it seats.
		enterLocation(&c.bank)
		if err := t.sess.enteredTown(t.in.townInstall()); err != nil {
			return ui.TownAction{Msg: err.Error()}
		}
		return ui.TownAction{}
	}
	// A bag the map still holds, as after a LOAD at the map, goes with the
	// hero (DIV-4191).
	t.sess.packBag(c, t.in.townInstall().table)
	return ui.TownAction{Open: t.openMission(l.id)}
}

type worldMapPage struct{ host *townScreen }

func (p *worldMapPage) policy() WorldMapPagePolicy {
	return WorldMapDescriptionFor(p.host.in.profile()).Page
}

func (p *worldMapPage) atMap() bool {
	if p == nil {
		return false
	}
	switch p.policy().Open {
	case "gates":
		return p.host.room == roomGates
	case "unlocated":
		c := p.host.state()
		return c != nil && c.current == (partyPlace{}) && p.host.talkPage().payload == nil
	}
	return false
}

func (p *worldMapPage) worldMapNow() *worldMapState {
	if p == nil {
		return nil
	}
	t := p.host
	if p.policy().Build == "entry" {
		return t.worldMap
	}
	if !p.atMap() || t.mapArt == nil {
		return nil
	}
	c := t.state()
	if c.mapOpen == nil {
		// A LOAD at the map keeps no opening: the map opens in select mode
		// with the party at the first available record (DIV-3071).
		from := partyPlace{}
		if len(c.available) > 0 {
			from = c.available[0]
		}
		c.mapOpen = &secondMapOpening{from: from}
	}
	if t.worldMap != nil && t.worldMapOf == c && t.worldMapAt == c.mapOpen {
		return t.worldMap
	}
	if p.openWorldMap(p.opening(), p.records()) == nil {
		return nil
	}
	t.worldMapOf, t.worldMapAt = c, c.mapOpen
	return t.worldMap
}

func (p *worldMapPage) entry(l partyPlace) worldMapEntry {
	e := worldMapEntry{place: l, point: "catalog"}
	if l.kind == placeTown {
		e.town, _ = TownCatalog(p.host.in.profile()).Town(l.id)
	}
	return e
}

func (p *worldMapPage) opening() worldMapEntry {
	if p.policy().Point == "from-record" {
		return p.entry(p.host.state().mapOpen.from)
	}
	return worldMapEntry{}
}

func (p *worldMapPage) records() []worldMapEntry {
	t := p.host
	var records []worldMapEntry
	for _, source := range p.policy().Records {
		switch source {
		case "available-missions":
			for _, id := range t.sess.Town.Available() {
				records = append(records, worldMapEntry{place: partyPlace{id: id}, point: "registry", region: true, text: "mission-files", payment: t.sess.Town.campaignPayment(id), admission: "mission"})
			}
		case "travel-towns":
			for _, e := range t.sess.Town.travelTowns() {
				records = append(records, worldMapEntry{place: partyPlace{placeTown, e.ID}, label: e.Name, town: e, region: true, text: "town-label", admission: "town"})
			}
		case "destinations":
			for _, l := range t.state().available {
				records = append(records, p.entry(l))
			}
		}
	}
	return records
}

func (p *worldMapPage) openWorldMap(from worldMapEntry, records []worldMapEntry) *worldMapState {
	t, row := p.host, p.policy()
	var a *worldMapAssets
	if row.Build == "opening" {
		a = t.mapArt()
		if a == nil || a.desc == nil {
			return nil
		}
	} else {
		a = t.art.worldMap()
	}
	builder := worldMapEntryBuilder{assets: a, in: t.in}
	s := builder.open(from, records)
	name := row.Mode
	if row.RouteMode != "" && t.state().mapOpen.route {
		name = row.RouteMode
	}
	if a.desc != nil && name != "first" {
		for i, m := range a.desc.Modes {
			if m.Name == name {
				s.mode = i
			}
		}
	}
	if row.Point == "retained" {
		home := worldMapHome(t.partyTown(), a)
		if !t.worldPositionSet {
			t.worldPosition, t.worldPositionSet = home, true
			if first, restored := t.sess.Town.firstMapPoint(); restored && !first {
				if m := builder.record(worldMapEntry{place: partyPlace{id: t.sess.Town.selectedMission()}, point: "registry"}); m.Enabled {
					t.worldPosition = m.Anchor
				}
			}
		}
		s.current, s.homeTown = t.worldPosition, t.partyTown()
		if s.current != home && t.sess.Town.restoredCampaign() {
			s.returnMission = t.sess.Town.selectedMission()
			s.route = worldMapRoute(a.mode(s.mode).graph, s.current, home, worldMapFallbackStep)
		}
	}
	if row.History && t.worldSelectedOnce == nil {
		for _, mission := range t.sess.Town.selectedMarkerMissions() {
			t.markWorldSelected(mission)
		}
	}
	if row.TravelOnOpen {
		s.openTravel()
	}
	t.worldMap = s
	return s
}

func (p *worldMapPage) WorldMapView() ui.WorldMapView {
	s := p.worldMapNow()
	if s == nil {
		return ui.WorldMapView{Hovered: -1, Selected: -1, Problem: "world map is unavailable"}
	}
	t, row := p.host, p.policy()
	words, font, task := t.in.Words, t.in.Font.Value(), 0
	if row.Font == "document" {
		font = t.art.documentFont()
	}
	if row.HomeTitle && s.homeTown.Name != "" {
		words.WorldHomeTitle = s.homeTown.Name
	}
	if row.Task {
		task = t.worldFlag1Frame
	}
	in := worldMapViewInput{font: font, detail: t.in.tipFont(), words: words, task: task}
	if row.RecordText {
		in.text = func(kind string, id int) (string, string) { return s.assets.worldMapRecordText(t.in, kind, id) }
	}
	v := s.view(in)
	if row.History && t.in.Archives != nil && t.in.Archives.Containers != nil {
		a := s.assets
		for i := range v.Missions {
			if v.Missions[i].Enabled && v.Missions[i].Kind != placeTown && t.worldSelectedOnce[v.Missions[i].Number] {
				obj := a.data.Objects[v.Missions[i].Object]
				v.Missions[i].Marker = worldMapOptionalImage(a.marker(v.Missions[i].Number, obj))
			}
		}
		if a.data != nil {
			shown := make(map[int]bool, len(v.Missions))
			for _, m := range v.Missions {
				if m.Kind != placeTown {
					shown[m.Number] = true
				}
			}
			for _, mission := range sortedSelectedMissions(t.worldSelectedOnce) {
				object, ok := a.data.Missions[mission]
				if shown[mission] || !ok || object < 0 || object >= len(a.data.Objects) {
					continue
				}
				obj := a.data.Objects[object]
				if picture := worldMapOptionalImage(a.marker(mission, obj)); picture != nil {
					v.Markers = append(v.Markers, ui.WorldMapMarker{Anchor: obj.Point, Picture: picture})
				}
			}
		}
	}
	return v
}

func (p *worldMapPage) chooseWorldRecord(i int) {
	s := p.worldMapNow()
	if s == nil || i < 0 || i >= len(s.missions) || !s.startTravel(i) {
		return
	}
	m := s.missions[i]
	if p.policy().History && m.Kind != placeTown {
		if p.host.sess != nil {
			p.host.sess.Town.selectMission(m.Number)
		}
		p.host.markWorldSelected(m.Number)
	}
}

func (p *worldMapPage) leaveWorldMap() {
	if p.policy().Leave == "square" {
		s := p.worldMapNow()
		s.route, s.shown, s.cross, s.selected = nil, 0, 0, -1
		p.host.atSquare()
	}
}

var worldMapLocationEntries = map[string]func(*townScreen, ui.WorldMapMission) ui.TownAction{
	"town-state":     (*townScreen).arriveInTown,
	"mission-action": func(t *townScreen, m ui.WorldMapMission) ui.TownAction { return t.worldMapMissionAction(m.Number) },
	"location": func(t *townScreen, m ui.WorldMapMission) ui.TownAction {
		return t.enterLocation(partyPlace{m.Kind, m.Number})
	},
}

func (p *worldMapPage) arriveWorldMap() ui.TownAction {
	s, t, row := p.worldMapNow(), p.host, p.policy().Arrival
	i := s.selected
	s.route, s.shown, s.selected = nil, 0, -1
	if row.ClearCross {
		s.cross = 0
	}
	if row.ReturnHome && s.returnMission != 0 {
		s.returnMission = 0
		s.current = s.home()
		t.worldPosition, t.worldPositionSet = s.current, true
		if t.sess.Town.progress != nil {
			t.sess.Town.progress.firstMapPoint = true
		}
		t.atSquare()
		return ui.TownAction{}
	}
	if i < 0 || i >= len(s.missions) {
		return ui.TownAction{}
	}
	if row.Snap {
		if i >= len(t.state().available) {
			return ui.TownAction{}
		}
		s.current = s.missions[i].Anchor
		for k, m := range s.missions {
			if m.Enabled && m.Anchor == s.current {
				i = k
				break
			}
		}
	}
	m := s.missions[i]
	entry := row.Mission
	if m.Kind == placeTown {
		entry = row.Town
	} else if row.KeepPosition {
		s.current, t.worldPosition = m.Anchor, m.Anchor
	}
	if !m.Enabled {
		return ui.TownAction{Msg: m.Problem}
	}
	if row.Snap {
		l := t.state().available[i]
		m.Kind, m.Number = l.kind, l.id
	}
	return worldMapLocationEntries[entry](t, m)
}

func (p *worldMapPage) tick() ui.TownAction {
	if p == nil {
		return ui.TownAction{}
	}
	if p.policy().Task {
		if p.worldMapNow() == nil {
			return ui.TownAction{}
		}
		p.host.worldFlag1Frame++
	}
	return worldMapTick(p)
}
