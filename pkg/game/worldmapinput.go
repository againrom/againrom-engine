package game

import (
	"image"

	"againrom/pkg/ui"
)

// worldMapCampaign is a campaign's side of the one world-map builder: its
// open map, its view, and what choosing a record, leaving the map and
// arriving do to the campaign. The input rules below are the builder's and
// follow the map description's pick rule.
type worldMapCampaign interface {
	worldMapNow() *worldMapState
	WorldMapView() ui.WorldMapView
	chooseWorldRecord(i int)
	leaveWorldMap()
	arriveWorldMap() ui.TownAction
}

// worldMapHover is the pointer. The nearest pick targets the record nearest
// it (R2-ENGINE-431). The scroll pick hovers a card first, then a record's
// region (`TOWN-118`).
func worldMapHover(c worldMapCampaign, p image.Point) {
	s := c.worldMapNow()
	if s == nil {
		return
	}
	if s.pickNearest() {
		s.aim(p)
		return
	}
	v := c.WorldMapView()
	s.scrollHovered = false
	if i, ok := ui.WorldMapCardAt(v, p); ok {
		s.scrollHovered = true
		s.hovered = i
		return
	}
	if i, ok := ui.WorldMapRegionAt(v, p); ok {
		s.hovered = i
		return
	}
	s.hovered = -1
}

// worldMapMove is Up and Down. Under the nearest pick a key skips a running
// travel (R2-ENGINE-428). Under the scroll pick it chooses the next enabled
// mission before the selected one for delta -1 (Up, the wheel away from the
// user) and after it for +1 (Down, the wheel toward the user). Reaching the
// mission the party is already travelling to chooses nothing, so its route and
// reveal keep running (DIV-1515), as a click on its scroll leaves them
// (DIV-1479).
func worldMapMove(c worldMapCampaign, delta int) {
	s := c.worldMapNow()
	if s == nil || len(s.missions) == 0 || delta == 0 {
		return
	}
	if s.pickNearest() {
		s.skipTravel()
		return
	}
	if s.hideScrolls() {
		return
	}
	i := s.selected
	if i < 0 && delta < 0 {
		// A backward move from the before-first sentinel begins after the last
		// item, just as a forward move begins before the first.
		i = 0
	}
	for attempts := 0; attempts < len(s.missions); attempts++ {
		i = (i + delta + len(s.missions)) % len(s.missions)
		if s.missions[i].Enabled {
			if !s.travellingTo(i) {
				c.chooseWorldRecord(i)
			}
			return
		}
	}
}

// worldMapChoose is Enter. The original's key-down and character slots call
// the progress helper a missed scroll click calls, without reading the key
// value or the selection (`TOWN-485`, `TOWN-486`, High; `TOWN-487` binds
// Return through the campaign; R2-ENGINE-428). So Enter does what a click
// that misses every scroll does at the same moment (skipTravel). It never
// opens a record itself and reports no problem (DIV-1546).
func worldMapChoose(c worldMapCampaign) ui.TownAction {
	if s := c.worldMapNow(); s != nil {
		s.skipTravel()
	}
	return ui.TownAction{}
}

// worldMapClick is a press. Under the nearest pick a press skips a running
// travel, or starts the travel to the target (R2-ENGINE-431, R2-ENGINE-432).
//
// Under the scroll pick it chooses through the mission scrolls only
// (`TOWN-118`, High): a scroll click starts travel to its mission, and a click
// on another mission's scroll restarts travel toward that one. A click on the
// scroll of the mission already being travelled to leaves the route and its
// reveal running (DIV-1479); the town scroll leaves the map. A click that
// lands on no card is the miss-a-scroll skip arm (`TOWN-121`; DIV-1516).
// Arrival opens the record by itself, with no second click.
func worldMapClick(c worldMapCampaign, p image.Point) ui.TownAction {
	s := c.worldMapNow()
	if s == nil {
		return ui.TownAction{}
	}
	if s.pickNearest() {
		if s.travelling() {
			s.skipTravel()
			return ui.TownAction{}
		}
		s.aim(p)
		if s.hovered >= 0 {
			c.chooseWorldRecord(s.hovered)
		}
		return ui.TownAction{}
	}
	v := c.WorldMapView()
	if i, ok := ui.WorldMapCardAt(v, p); ok {
		if i < 0 {
			c.leaveWorldMap()
			return ui.TownAction{}
		}
		if !s.travellingTo(i) {
			c.chooseWorldRecord(i)
		}
		return ui.TownAction{}
	}
	s.skipTravel()
	return ui.TownAction{}
}

// worldMapTick is one paint (`TOWN-120`, R2-ENGINE-432). It arrives, with no
// second click, the moment the reveal has reached the route's own end and the
// Cross animation has passed its end — from the tick that finishes the later
// of the two or from a skip (`TOWN-121`; DIV-1545). The call rate is authored
// (`DIV-136`).
func worldMapTick(c worldMapCampaign) ui.TownAction {
	s := c.worldMapNow()
	if s == nil || !s.tick() {
		return ui.TownAction{}
	}
	return c.arriveWorldMap()
}
