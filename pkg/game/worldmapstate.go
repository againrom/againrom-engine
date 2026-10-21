package game

import (
	"fmt"
	"image"
	"strings"

	rendertext "againrom/pkg/render/text"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// worldMapState is the one world-map builder's live state. A campaign fills
// its records and opens it in one of its description's modes; the rules it
// picks, travels and draws by are the description's (WorldMapDescription).
// Assets without a description follow the scroll rules with no preview.
type worldMapState struct {
	assets *worldMapAssets
	// mode is the description mode the map opened in.
	mode          int
	missions      []ui.WorldMapMission
	selected      int
	hovered       int
	scrollHovered bool
	route         []image.Point
	shown         int
	// returnMission identifies a completed mission's pending homeward trip.
	// It is separate from selected: home is the party's town, not a mission
	// scroll.
	returnMission int
	// homeTown is the catalog entry of the party's town, whose point is
	// home: where a homeward trip ends and the map's at-home rules hold. The
	// zero entry is the starting town's.
	homeTown town.Entry
	// current is the party's own current-position field (`TOWN-120`/
	// `TOWN-121`, High), copied in from the campaign on entry and written
	// back only at arrival.
	current image.Point
	// frame is the party flag's animation counter and marks the record
	// markers', each advanced once per paint — once per WorldMapTick call
	// (`TOWN-120`, R2-ENGINE-430; `DIV-136`).
	frame int
	marks int
	// cross advances on drawn paints; the description chooses whether it
	// starts at enter or at each route (`TOWN-530`).
	// The animation has passed its end once the counter reaches the sheet's
	// frame count plus the description's hold, and arrival waits for that as
	// well as for the reveal (`TOWN-121`, R2-ENGINE-432; DIV-1545).
	cross int
}

// home is the party's town's point on the map.
func (s *worldMapState) home() image.Point {
	if s == nil {
		return worldMapNoPoint
	}
	return worldMapHome(s.homeTown, s.assets)
}

// desc is the description the state follows, or nil.
func (s *worldMapState) desc() *WorldMapDescription {
	if s == nil || s.assets == nil {
		return nil
	}
	return s.assets.desc
}

// modeSpec is the description mode the map opened in.
func (s *worldMapState) modeSpec() WorldMapMode {
	if d := s.desc(); d != nil {
		if s.mode >= 0 && s.mode < len(d.Modes) {
			return d.Modes[s.mode]
		}
	}
	return WorldMapMode{}
}

func (s *worldMapState) pickNearest() bool {
	d := s.desc()
	return d != nil && d.Pick == "nearest"
}

func (s *worldMapState) graph() *worldMapNodeGraph { return s.assets.mode(s.mode).graph }

// travelling reports whether a travel runs: a chosen record or the homeward
// trip with its route held.
func (s *worldMapState) travelling() bool {
	return (s.selected >= 0 || s.returnMission != 0) && len(s.route) > 0
}

func (s *worldMapState) routeEnd() int {
	if d := s.desc(); d != nil && d.RoutePastEnd {
		return len(s.route) + 1
	}
	return len(s.route)
}

func (s *worldMapState) crossPassedEnd() bool {
	if s.assets == nil || len(s.assets.cross) == 0 {
		return true
	}
	hold := 0
	if d := s.desc(); d != nil {
		hold = d.CrossHold
	}
	return s.cross >= len(s.assets.cross)+hold
}

// finishCross assigns the Cross counter the sheet's frame count plus one, as
// the original's progress helper does to its Cross counter whatever the counter
// held (`TOWN-486`, R2-ENGINE-432). The frame drawn until the next tick follows
// that counter.
func (s *worldMapState) finishCross() {
	if s.assets != nil {
		s.cross = len(s.assets.cross) + 1
	}
}

// skipTravel is the skip a click that misses every scroll, a press during a
// travel or a key makes (`TOWN-121`, R2-ENGINE-432): it sets the reveal to the
// route's own end and the Cross counter past its end, so that the next tick
// arrives. Before the first reveal tick it does nothing unless the description
// lets a skip finish a travel at its start (DIV-1516).
func (s *worldMapState) skipTravel() {
	if !s.travelling() {
		return
	}
	if d := s.desc(); s.shown > 0 || d != nil && d.SkipAtStart {
		s.shown = s.routeEnd()
		s.finishCross()
	}
}

func (s *worldMapState) hideScrolls() bool {
	if d := s.desc(); d != nil && d.Cards == nil {
		return true
	}
	return s.returnMission != 0 || s.current != s.home()
}

// travellingTo reports whether the party is on its way to mission i: the
// mission is selected and its route is held. Arrival, an immediate open and
// the town scroll clear both, so a held route is travel not yet arrived.
func (s *worldMapState) travellingTo(i int) bool {
	return s.selected == i && len(s.route) > 0
}

// startTravel (re)starts travel to record i from the party's own current
// position and reports whether the record can be travelled to.
func (s *worldMapState) startTravel(i int) bool {
	if s == nil || i < 0 || i >= len(s.missions) {
		return false
	}
	m := s.missions[i]
	s.selected = i
	s.route, s.shown = nil, 0
	if d := s.desc(); d == nil || !d.CrossAtEnter {
		s.cross = 0
	}
	if !m.Enabled {
		return false
	}
	s.route = worldMapRoute(s.graph(), s.current, m.Anchor, worldMapFallbackStep)
	return true
}

// aim is the nearest pick (R2-ENGINE-431): while no travel runs, the pointer
// targets the enabled record nearest to it by squared distance, a tie keeping
// the earlier record, with no distance limit. A target at a new point
// recomputes the route from the party's point. A pointer outside the map
// keeps the target.
func (s *worldMapState) aim(p image.Point) {
	if s == nil || s.travelling() || !p.In(image.Rect(0, 0, 640, 480)) {
		return
	}
	best, dist := -1, 0
	for i, m := range s.missions {
		if !m.Enabled {
			continue
		}
		dx, dy := m.Anchor.X-p.X, m.Anchor.Y-p.Y
		if d := dx*dx + dy*dy; best < 0 || d < dist {
			best, dist = i, d
		}
	}
	if best < 0 {
		s.hovered, s.route = -1, nil
		return
	}
	if s.hovered >= 0 && s.hovered < len(s.missions) && s.missions[s.hovered].Anchor == s.missions[best].Anchor && s.route != nil {
		s.hovered = best
		return
	}
	s.hovered = best
	s.route = worldMapRoute(s.graph(), s.current, s.missions[best].Anchor, worldMapFallbackStep)
}

// tick advances one paint: the party and marker counters, and while a travel
// runs the Cross counter by one and the route reveal by 8 coordinates,
// clamped to the route's own length (`TOWN-120`, R2-ENGINE-432). It reports
// that the travel has ended: the reveal has reached the route's end and the
// Cross animation has passed its end.
func (s *worldMapState) tick() bool {
	s.frame++
	s.marks++
	if !s.travelling() {
		s.openTravel()
		return false
	}
	if s.crossDrawn() {
		s.cross++
	}
	s.shown = min(s.shown+worldMapRouteReveal, s.routeEnd())
	return s.shown >= s.routeEnd() && s.crossPassedEnd()
}

func (s *worldMapState) crossDrawn() bool {
	if !s.travelling() {
		return false
	}
	destination := s.home()
	if s.returnMission == 0 && s.selected < len(s.missions) {
		destination = s.missions[s.selected].Anchor
	}
	return destination != s.current || s.current != s.home()
}

// openTravel starts the travel to the first record when the mode says so
// and none runs: the map opens travelling, and a paint with no travel starts
// it again (R2-ENGINE-272, R2-ENGINE-432).
func (s *worldMapState) openTravel() {
	if s.modeSpec().Travel == "first" && len(s.missions) > 0 {
		s.startTravel(0)
	}
}

// worldMapViewInput is what a campaign lends the view: the fonts and words
// its scroll cards draw with and the task flag's counter.
type worldMapViewInput struct {
	font, detail *rendertext.Font
	words        ui.Words
	task         int
	text         func(kind string, id int) (name, body string)
}

// view is the state's presentation snapshot by its description.
func (s *worldMapState) view(in worldMapViewInput) ui.WorldMapView {
	a, d, mode := s.assets, s.desc(), s.modeSpec()
	art := a.mode(s.mode)
	v := ui.WorldMapView{
		Background: worldMapOptionalImage(art.background), Ball: worldMapOptionalImage(a.ball),
		Flag: worldMapOptionalImage(a.flagFrame(s.mode, s.frame)), Available: worldMapOptionalImage(a.availableFrame(in.task)),
		Cross: worldMapOptionalImage(a.crossFrame(s.cross)),
		Scroll: [3]image.Image{
			worldMapOptionalImage(a.scroll[0]), worldMapOptionalImage(a.scroll[1]), worldMapOptionalImage(a.scroll[2]),
		},
		Pressed: [3]image.Image{
			worldMapOptionalImage(a.pressed[0]), worldMapOptionalImage(a.pressed[1]), worldMapOptionalImage(a.pressed[2]),
		},
		Font: in.font, DetailFont: in.detail, ScrollHovered: s.scrollHovered, Words: in.words,
		Missions: append([]ui.WorldMapMission(nil), s.missions...), CardBase: ui.WorldMapCardPageBase(s.selected),
		Hovered: s.hovered, Selected: s.selected, Route: append([]image.Point(nil), s.route...),
		RouteShown: min(s.shown, len(s.route)), Position: s.current, AtHome: s.current == s.home(),
		HideScrolls: s.hideScrolls(),
		Returning:   s.returnMission != 0, Destination: s.home(),
		Problem: a.problem,
	}
	if d == nil {
		return v
	}
	v.FlagAt, v.CrossAt, v.AvailableAt = mode.Party.place(), d.Cross.place(), d.Task.place()
	v.PointerPicks = d.Pick == "nearest"
	// Rule markers stand over the route dots (R2-ENGINE-429); registry
	// pictures are whole-map overlays beneath them.
	v.MarkersOverRoute = len(d.Markers) > 0
	if d.Preview && !s.travelling() {
		v.RouteShown = len(v.Route)
	}
	if mode.Markers && !s.travelling() {
		for i := range v.Missions {
			v.Missions[i].Marker, v.Missions[i].MarkerAt = s.marker(i)
		}
	}
	if d.Text != nil && mode.Text && in.text != nil && s.hovered >= 0 && s.hovered < len(s.missions) {
		m := s.missions[s.hovered]
		if name, body := in.text(worldMapKind(m), m.Number); name != "" {
			t := d.Text
			v.Text = &ui.WorldMapTextBlock{Name: name, Body: body,
				NameAt: image.Pt(t.NameAt[0], t.NameAt[1]), LinesAt: image.Pt(t.LinesAt[0], t.LinesAt[1]),
				LineStep: t.LineStep, Width: t.Width, MaxLines: t.MaxLines, NameFont: in.font, LineFont: in.detail}
		}
	}
	return v
}

// marker is record i's marker by the description's rules (R2-ENGINE-430):
// the target's rule at the target, the other rule at every other record not
// at the party's point. A disabled record draws no marker.
func (s *worldMapState) marker(i int) (image.Image, ui.WorldMapPlace) {
	d := s.desc()
	m := s.missions[i]
	if !m.Enabled {
		return nil, ui.WorldMapPlace{}
	}
	for k, r := range d.Markers {
		target := s.hovered >= 0 && s.hovered < len(s.missions) && s.missions[s.hovered].Anchor == m.Anchor
		switch {
		case r.Who == "target" && target, r.Who == "other" && !target && m.Anchor != s.current:
			place := WorldMapPlace{At: &r.At}
			return worldMapOptionalImage(worldMapPickFrame(s.assets.rules[k], s.marks)), place.place()
		}
	}
	return nil, ui.WorldMapPlace{}
}

// worldMapKind is the text kind of a record: its campaign kind number.
func worldMapKind(m ui.WorldMapMission) string { return fmt.Sprint(m.Kind) }

// worldMapRecordText is a record's name and description from the
// description's text source, by the key formats of its kind; a key that
// finds nothing gives empty text (R2-ENGINE-434).
func (a *worldMapAssets) worldMapRecordText(in *InstallResources, kind string, id int) (string, string) {
	if a == nil || a.desc == nil || a.desc.Text == nil {
		return "", ""
	}
	t := a.desc.Text
	if !a.recordTextTried {
		a.recordTextTried = true
		if in != nil && in.Archives != nil && in.Archives.Containers != nil {
			if raw, err := a.read(in.Archives.Containers, t.Source); err == nil {
				a.recordText = string(in.textCode().Bytes(raw))
			}
		}
	}
	find := func(formats map[string]string) string {
		format, ok := formats[kind]
		if !ok {
			return ""
		}
		body := secondGameTextSection([]byte(a.recordText), fmt.Sprintf(format, id))
		if len(body) < t.Trim {
			return ""
		}
		return strings.TrimRight(body[:len(body)-t.Trim], "\x00")
	}
	return find(t.Name), find(t.Description)
}
