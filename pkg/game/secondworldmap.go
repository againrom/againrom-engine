package game

import (
	"fmt"
	"image"

	"againrom/pkg/ui"
)

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

// worldMapNow is the open map, built once per opening from the available
// list in its order and the catalog points.

// placePoint is where place l stands on map a: a town at its catalog
// entry's map point, a mission at its record.
func (t *destinationTownMap) placePoint(a *worldMapAssets, l partyPlace) (image.Point, bool) {
	if l.kind != placeTown {
		return a.desc.point(l.kind, l.id)
	}
	e, ok := TownCatalog(t.host.in.profile()).Town(l.id)
	if !ok {
		return image.Point{}, false
	}
	p, _, ok := townMapPoint(e, a)
	return p, ok
}

// leaveWorldMap does nothing: the map has no cancel (R2-ENGINE-428).

// arriveWorldMap ends a travel: the party stands at the target, the map
// closes and the record at the target is entered, the first available record
// at that point (R2-ENGINE-432). A town becomes current; a mission opens.

// enterLocation is EnterLocation and the 0x468 arm: a town becomes current on
// its square; a mission opens (R2-ENGINE-272).
func (t *destinationTownMap) enterLocation(l partyPlace) ui.TownAction {
	c := t.host.state()
	if l.kind == placeTown {
		c.current, c.room, c.mapOpen = l, secondTownSquare, nil
		return ui.TownAction{}
	}
	return ui.TownAction{Open: t.host.openMission(l.id)}
}

func (t *destinationTownMap) atMap() bool {
	c := t.host.state()
	return c != nil && c.current == (partyPlace{}) && c.talk.payload == nil
}

func (t *destinationTownMap) WorldMapView() ui.WorldMapView {
	s := t.worldMapNow()
	if s == nil {
		return ui.WorldMapView{Hovered: -1, Selected: -1, Problem: "world map is unavailable"}
	}
	return s.view(worldMapViewInput{font: t.host.in.Font.Value(), detail: t.host.in.tipFont(), words: t.host.in.Words,
		text: func(kind string, id int) (string, string) { return s.assets.worldMapRecordText(t.host.in, kind, id) }})
}

func (t *destinationTownMap) worldMapNow() *worldMapState {
	if t == nil || !t.host.AtWorldMap() || t.host.mapArt == nil {
		return nil
	}
	c := t.host.state()
	if c.mapOpen == nil {
		// A LOAD at the map keeps no opening: the map opens in select mode
		// with the party at the first available record (DIV-3071).
		from := partyPlace{}
		if len(c.available) > 0 {
			from = c.available[0]
		}
		c.mapOpen = &secondMapOpening{from: from}
	}
	if t.host.worldMap != nil && t.host.worldMapOf == c && t.host.worldMapAt == c.mapOpen {
		return t.host.worldMap
	}
	a := t.host.mapArt()
	if a == nil || a.desc == nil {
		return nil
	}
	name := secondMapSelect
	if c.mapOpen.route {
		name = secondMapRoute
	}
	s := &worldMapState{assets: a, selected: -1, hovered: -1}
	for i, m := range a.desc.Modes {
		if m.Name == name {
			s.mode = i
		}
	}
	s.current, _ = t.placePoint(a, c.mapOpen.from)
	for _, l := range c.available {
		m := ui.WorldMapMission{Number: l.id, Kind: l.kind, Enabled: true}
		if p, ok := t.placePoint(a, l); ok {
			m.Anchor = p
		} else {
			m.Enabled, m.Problem = false, fmt.Sprintf("Destination %d/%d has no map point", l.kind, l.id)
		}
		s.missions = append(s.missions, m)
	}
	s.openTravel()
	t.host.worldMap, t.host.worldMapOf, t.host.worldMapAt = s, c, c.mapOpen
	return s
}

func (t *destinationTownMap) chooseWorldRecord(i int) { t.worldMapNow().startTravel(i) }

func (t *destinationTownMap) leaveWorldMap() {}

func (t *destinationTownMap) arriveWorldMap() ui.TownAction {
	s, c := t.worldMapNow(), t.host.state()
	i := s.selected
	s.route, s.shown, s.cross, s.selected = nil, 0, 0, -1
	if i < 0 || i >= len(s.missions) || i >= len(c.available) {
		return ui.TownAction{}
	}
	s.current = s.missions[i].Anchor
	for k, m := range s.missions {
		if m.Enabled && m.Anchor == s.current {
			i = k
			break
		}
	}
	if !s.missions[i].Enabled {
		return ui.TownAction{Msg: s.missions[i].Problem}
	}
	return t.enterLocation(c.available[i])
}

func (t *destinationTownMap) tick() ui.TownAction { return worldMapTick(t) }
