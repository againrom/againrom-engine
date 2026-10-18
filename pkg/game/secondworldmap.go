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
	from  secondLocation
	route bool
}

func (c *secondCampaign) openMap(from secondLocation) {
	c.mapOpen = &secondMapOpening{from: from, route: from == (secondLocation{kind: 2, id: 1})}
}

// enteredWithoutMap is the mission a victory enters without the map: the
// first available record when no town is available and the stage is below
// the credits (R2-ENGINE-422 step 5); zero when the map opens.
func (c *secondCampaign) enteredWithoutMap() int {
	if c.bank[768] >= 120 || len(c.available) == 0 || c.available[0].kind != 1 {
		return 0
	}
	for _, l := range c.available {
		if l.kind == 2 {
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

func (t *secondCampaignScreen) AtWorldMap() bool {
	c := t.state()
	return c != nil && c.current == (secondLocation{}) && c.payload == nil
}

// worldMapNow is the open map, built once per opening from the available
// list in its order and the catalog points.
func (t *secondCampaignScreen) worldMapNow() *worldMapState {
	if t == nil || !t.AtWorldMap() || t.mapArt == nil {
		return nil
	}
	c := t.state()
	if c.mapOpen == nil {
		// A LOAD at the map keeps no opening: the map opens in select mode
		// with the party at the first available record (DIV-3071).
		from := secondLocation{}
		if len(c.available) > 0 {
			from = c.available[0]
		}
		c.mapOpen = &secondMapOpening{from: from}
	}
	if t.worldMap != nil && t.worldMapOf == c && t.worldMapAt == c.mapOpen {
		return t.worldMap
	}
	a := t.mapArt()
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
	s.current, _ = a.desc.point(c.mapOpen.from.kind, c.mapOpen.from.id)
	for _, l := range c.available {
		m := ui.WorldMapMission{Number: l.id, Kind: l.kind, Enabled: true}
		if p, ok := a.desc.point(l.kind, l.id); ok {
			m.Anchor = p
		} else {
			m.Enabled, m.Problem = false, fmt.Sprintf("Destination %d/%d has no map point", l.kind, l.id)
		}
		s.missions = append(s.missions, m)
	}
	s.openTravel()
	t.worldMap, t.worldMapOf, t.worldMapAt = s, c, c.mapOpen
	return s
}

func (t *secondCampaignScreen) WorldMapView() ui.WorldMapView {
	s := t.worldMapNow()
	if s == nil {
		return ui.WorldMapView{Hovered: -1, Selected: -1, Problem: "world map is unavailable"}
	}
	return s.view(worldMapViewInput{font: t.install.Font.Value(), detail: t.install.tipFont(), words: t.install.Words,
		text: func(kind string, id int) (string, string) { return s.assets.worldMapRecordText(t.install, kind, id) }})
}

func (t *secondCampaignScreen) WorldMapHover(p image.Point)               { worldMapHover(t, p) }
func (t *secondCampaignScreen) WorldMapMove(delta int)                    { worldMapMove(t, delta) }
func (t *secondCampaignScreen) WorldMapChoose() ui.TownAction             { return worldMapChoose(t) }
func (t *secondCampaignScreen) WorldMapClick(p image.Point) ui.TownAction { return worldMapClick(t, p) }
func (t *secondCampaignScreen) WorldMapTick() ui.TownAction               { return worldMapTick(t) }
func (t *secondCampaignScreen) chooseWorldRecord(i int)                   { t.worldMapNow().startTravel(i) }

// leaveWorldMap does nothing: the map has no cancel (R2-ENGINE-428).
func (t *secondCampaignScreen) leaveWorldMap() {}

// arriveWorldMap ends a travel: the party stands at the target, the map
// closes and the record at the target is entered, the first available record
// at that point (R2-ENGINE-432). A town becomes current; a mission opens.
func (t *secondCampaignScreen) arriveWorldMap() ui.TownAction {
	s, c := t.worldMapNow(), t.state()
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

// enterLocation is EnterLocation and the 0x468 arm: a town becomes current on
// its square; a mission opens (R2-ENGINE-272).
func (t *secondCampaignScreen) enterLocation(l secondLocation) ui.TownAction {
	c := t.state()
	if l.kind == 2 {
		c.current, c.room, c.mapOpen = l, secondTownSquare, nil
		return ui.TownAction{}
	}
	return ui.TownAction{Open: t.open(l.id)}
}
