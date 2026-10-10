package game

import (
	"fmt"
	"image"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

var globalMapRegistry = scenarioPrefix + "globalmap.reg"

const (
	globalMapMain     = mainPrefix + "graphics/global.map/"
	globalMapGraphics = graphicsPrefix + "global.map/"
	// worldMapFallbackStep spaces the DIV-129 straight-line fallback used
	// when the node graph cannot route an endpoint. The graph route itself
	// is the concatenation of `PathMap.bmp` corridor pixels and needs no
	// sampling (`TOWN-119`, High).
	worldMapFallbackStep = 10
	// worldMapRouteReveal is how many route coordinates one paint reveals
	// (`TOWN-120`, High).
	worldMapRouteReveal = 8
)

type globalMapObject struct {
	Point   image.Point
	Region  image.Rectangle
	Picture string
	Valid   bool
	Problem string
}

type globalMapData struct {
	Objects  []globalMapObject
	Missions map[int]int
}

// ReadGlobalMap converts the registry's declared object collection and dynamic
// mission mapping. Invalid rows retain their declared indices.
func ReadGlobalMap(r *reg.Reg) (*globalMapData, error) {
	if r == nil || r.Root == nil {
		return nil, fmt.Errorf("global map registry is empty")
	}
	count, ok := r.GetInt("General", "ObjectCount")
	if !ok || count < 0 {
		return nil, fmt.Errorf("global map ObjectCount is absent or negative")
	}
	out := &globalMapData{Objects: make([]globalMapObject, int(count)), Missions: make(map[int]int)}
	for i := range out.Objects {
		section := "MapObject" + strconv.Itoa(i+1)
		point, pok := r.GetIntArray(section, "MapPoint")
		rect, rok := r.GetIntArray(section, "MapRect")
		obj := globalMapObject{}
		if !pok || len(point) != 2 || !rok || len(rect) != 4 {
			obj.Problem = section + ": missing MapPoint[2] or MapRect[4]"
			out.Objects[i] = obj
			continue
		}
		x, y, w, h := int64(rect[0]), int64(rect[1]), int64(rect[2]), int64(rect[3])
		maxInt := int64(^uint(0) >> 1)
		minInt := -maxInt - 1
		if w <= 0 || h <= 0 || x+w > maxInt || y+h > maxInt || x < minInt || y < minInt {
			obj.Problem = section + ": invalid or overflowing MapRect"
			out.Objects[i] = obj
			continue
		}
		obj.Point = image.Pt(int(point[0]), int(point[1]))
		obj.Region = image.Rect(int(x), int(y), int(x+w), int(y+h))
		obj.Picture, _ = r.GetString(section, "Picture")
		obj.Valid = true
		out.Objects[i] = obj
	}

	var missionSection *reg.Node
	for _, n := range r.Root.Children {
		if n.Dir && strings.EqualFold(n.Name, "MissionObjects") {
			missionSection = n
			break
		}
	}
	if missionSection == nil {
		return out, nil
	}
	for _, n := range missionSection.Children {
		if n.Dir || n.Type != reg.TypeInt || len(n.Name) <= len("Mission") ||
			!strings.EqualFold(n.Name[:len("Mission")], "Mission") {
			continue
		}
		mission, err := strconv.Atoi(n.Name[len("Mission"):])
		if err != nil || mission <= 0 {
			continue
		}
		object := int(n.Int) - 1
		out.Missions[mission] = object
	}
	return out, nil
}

type worldMapAssets struct {
	data       *globalMapData
	background *image.RGBA
	path       *image.Paletted
	graph      *worldMapNodeGraph
	ball       *image.RGBA
	// flag, available and cross hold every decoded frame of the
	// current-position Flag, the task Flag1 and the destination Cross, in
	// stream order: each is drawn from a frame counter (`TOWN-529`,
	// `TOWN-530`), so the picker at paint time needs the whole sheet.
	flag        []*image.RGBA
	available   []*image.RGBA
	cross       []*image.RGBA
	scroll      [3]*image.RGBA
	pressed     [3]*image.RGBA
	markers     map[int]*image.RGBA
	markerTried map[int]bool
	texts       map[int]worldMapText
	problem     string
	reads       map[string]int
}

// flagFrame and availableFrame pick frame n, wrapping around the sheet's own
// length (`TOWN-529`). crossFrame holds the sheet's last frame once n reaches
// it (`TOWN-530`): the Cross plays once and stays on its last frame, the one
// with the full cross (`TOWN-531`). Each returns nil for an undecoded or
// empty sheet, which worldMapOptionalImage reports as absent.
func (a *worldMapAssets) flagFrame(n int) *image.RGBA      { return worldMapPickFrame(a.flag, n) }
func (a *worldMapAssets) availableFrame(n int) *image.RGBA { return worldMapPickFrame(a.available, n) }
func (a *worldMapAssets) crossFrame(n int) *image.RGBA {
	if last := len(a.cross) - 1; n > last {
		n = last
	}
	return worldMapPickFrame(a.cross, n)
}

func worldMapPickFrame(frames []*image.RGBA, n int) *image.RGBA {
	if len(frames) == 0 {
		return nil
	}
	n %= len(frames)
	if n < 0 {
		n += len(frames)
	}
	return frames[n]
}

type worldMapText struct {
	title string
	brief string
}

func (f *FrontEnd) worldMapAssets() *worldMapAssets {
	return f.Presentation.worldMapAssets(&f.InstallResources)
}

func (p *Presentation) worldMapAssets(in *InstallResources) *worldMapAssets {
	if p.worldMapCache.Tried() {
		return p.worldMapCache.Value()
	}
	p.worldMapCache.begin()
	return p.worldMapCache.store(loadWorldMapAssets(in.Archives))
}

// coldWorldMapData is the campaign map's data as the first-use cache holds it,
// or as a first read of the install would produce it when the cache has not
// been tried. It takes the cache by value, so a first read is neither stored
// nor visible to the owner of the cache.
func coldWorldMapData(cache lazy[*worldMapAssets], archives *Archives) *globalMapData {
	assets := cache.Value()
	if !cache.Tried() {
		assets = loadWorldMapAssets(archives)
	}
	if assets != nil {
		return assets.data
	}
	return nil
}

// loadWorldMapAssets reads the campaign map's manifest and pictures from the
// install's archives.
func loadWorldMapAssets(archives *Archives) *worldMapAssets {
	a := &worldMapAssets{
		markers: make(map[int]*image.RGBA), markerTried: make(map[int]bool),
		texts: make(map[int]worldMapText), reads: make(map[string]int),
	}
	if archives == nil || archives.Containers == nil {
		a.problem = "world map assets are unavailable"
		return a
	}
	src := archives.Containers
	if raw, err := a.read(src, globalMapRegistry); err == nil {
		if parsed, err := reg.Parse(raw); err == nil {
			a.data, err = ReadGlobalMap(parsed)
			if err != nil {
				a.problem = err.Error()
			}
		} else {
			a.problem = fmt.Sprintf("%s: %v", globalMapRegistry, err)
		}
	} else {
		a.problem = err.Error()
	}
	a.background = a.bmp(src, globalMapMain+"gmap.bmp", false)
	a.path = a.mask(src, globalMapGraphics+"pathmap.bmp")
	a.graph = buildWorldMapNodeGraph(a.path)
	a.ball = a.bmp(src, globalMapGraphics+"ballmap.bmp", true)
	// Hero.bmp is read by no path this build follows any more: `TOWN-120`
	// (High) states the own-paint routine never reads it, and the
	// current-position marker it stood in for is `Flag` (`DIV-130`).
	a.flag = a.sprite16Frames(src, globalMapGraphics+"flag/sprites.16a")
	a.available = a.sprite16Frames(src, globalMapGraphics+"flag1/sprites.16a")
	a.cross = a.sprite16Frames(src, globalMapGraphics+"cross/sprites.16a")
	for i := 0; i < 3; i++ {
		a.scroll[i] = a.bmp(src, fmt.Sprintf("%sscroll0%d.bmp", globalMapGraphics, i+1), true)
		a.pressed[i] = a.bmp(src, fmt.Sprintf("%sscrollp%d.bmp", globalMapGraphics, i+1), true)
	}
	return a
}

func (a *worldMapAssets) read(src entrySource, path string) ([]byte, error) {
	a.reads[path]++
	return src.ReadFile(path)
}

func (a *worldMapAssets) bmp(src entrySource, path string, transparent bool) *image.RGBA {
	raw, err := a.read(src, path)
	if err != nil {
		return nil
	}
	pic, err := chargenRGBA(raw, path)
	if err != nil {
		return nil
	}
	if transparent {
		return keyBlack(pic)
	}
	return pic
}

func (a *worldMapAssets) mask(src entrySource, path string) *image.Paletted {
	raw, err := a.read(src, path)
	if err != nil {
		return nil
	}
	mask, err := terrain.DecodeBMP8(raw)
	if err != nil {
		return nil
	}
	return mask
}

// sprite16Frames reads the whole .16a sheet rather than frame 0 alone, for
// the three markers drawn from a frame counter (`Flag`, `Flag1`, `Cross`;
// `TOWN-529`, `TOWN-530`).
func (a *worldMapAssets) sprite16Frames(src entrySource, path string) []*image.RGBA {
	raw, err := a.read(src, path)
	if err != nil {
		return nil
	}
	return decodeWorldMap16Frames(raw)
}

// decodeWorldMap16Frames shares loadItemIcon's own pixel-and-palette
// resolution (cursorPixel, cursor.go) over every frame instead of frame 0.
func decodeWorldMap16Frames(raw []byte) []*image.RGBA {
	sprite, err := spr16.DecodeA(raw, true)
	if err != nil {
		return nil
	}
	out := make([]*image.RGBA, 0, len(sprite.Frames))
	for _, f := range sprite.Frames {
		if f.Width <= 0 || f.Height <= 0 {
			continue
		}
		pic := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
		for i, p := range f.Pixels {
			if !p.Painted {
				continue
			}
			c := cursorPixel(sprite.Palette, p)
			o := i * 4
			pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = c.R, c.G, c.B, c.A
		}
		out = append(out, pic)
	}
	return out
}

func (a *worldMapAssets) marker(src entrySource, mission int, obj globalMapObject) *image.RGBA {
	if a.markerTried[mission] {
		return a.markers[mission]
	}
	a.markerTried[mission] = true
	if obj.Picture == "" || strings.EqualFold(obj.Picture, "nothing") {
		return nil
	}
	path := globalMapMain + obj.Picture + ".256"
	raw, err := a.read(src, path)
	if err != nil {
		return nil
	}
	sheet, err := spr256.Decode(raw)
	if err != nil || !sheet.HasPalette || len(sheet.Frames) == 0 {
		return nil
	}
	frame := sheet.Frames[0]
	if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height {
		return nil
	}
	pic := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for i, p := range frame.Pixels {
		if !p.Opaque || int(p.Index) >= len(sheet.Palette) {
			continue
		}
		c := sheet.Palette[p.Index]
		o := i * 4
		pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = c.R, c.G, c.B, 0xff
	}
	a.markers[mission] = pic
	return pic
}

func WorldMapTextPaths(mission int) (title, briefing string, ok bool) {
	if mission <= 0 {
		return "", "", false
	}
	base := fmt.Sprintf("%stext/battle/m%d/", mainPrefix, mission)
	return base + "title.txt", base + "briefmap.txt", true
}

func (a *worldMapAssets) missionText(src entrySource, mission int) worldMapText {
	if got, ok := a.texts[mission]; ok {
		return got
	}
	titlePath, briefPath, _ := WorldMapTextPaths(mission)
	t := worldMapText{title: fmt.Sprintf("Mission %d", mission), brief: "Briefing unavailable"}
	if raw, err := a.read(src, titlePath); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		t.title = string(raw)
	}
	if raw, err := a.read(src, briefPath); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		t.brief = string(raw)
	}
	a.texts[mission] = t
	return t
}

type worldMapState struct {
	assets        *worldMapAssets
	missions      []ui.WorldMapMission
	selected      int
	hovered       int
	scrollHovered bool
	route         []image.Point
	shown         int
	// returnMission identifies a completed mission's pending homeward trip.
	// It is separate from selected: home is MapObject 0, not a mission scroll.
	returnMission int
	// current is the party's own current-position field (`TOWN-120`/
	// `TOWN-121`, High), copied in from townScreen.worldPosition on entry
	// and written back only at arrival (arriveWorldMap).
	current image.Point
	// frame is the current-position Flag's animation counter, advanced once
	// per paint — once per WorldMapTick call, which the reveal cadence now
	// gates the same way (`TOWN-120`, High; `DIV-136`).
	frame int
	// cross is the destination Cross's animation counter: zero when a route
	// starts, one more per paint while the route is held (`TOWN-120`, High).
	// The animation has passed its end once the counter reaches the sheet's
	// frame count, and arrival waits for that as well as for the reveal
	// (`TOWN-121`, High; DIV-1545).
	cross int
}

// crossPassedEnd reports whether the Cross animation has played through its
// sheet. A state with no Cross sheet has nothing to wait for.
func (s *worldMapState) crossPassedEnd() bool {
	return s.assets == nil || s.cross >= len(s.assets.cross)
}

// finishCross assigns the Cross counter the sheet's frame count plus one, as
// the original's progress helper does to its Cross counter whatever the counter
// held (`TOWN-486`, High). The frame drawn until the next tick follows that
// counter.
func (s *worldMapState) finishCross() {
	if s.assets != nil {
		s.cross = len(s.assets.cross) + 1
	}
}

// skipTravel is the skip a click that misses every scroll makes (`TOWN-121`,
// High): once the reveal has begun, on the outward and on the homeward trip
// alike, it sets the reveal to the route's own end and the Cross counter past
// its end, so that the next tick arrives. Before the first reveal tick it does
// nothing (DIV-1516).
func (s *worldMapState) skipTravel() {
	if (s.selected >= 0 || s.returnMission != 0) && s.shown > 0 {
		s.shown = len(s.route)
		s.finishCross()
	}
}

// worldMapHomePoint is the first MapObject's own point: the party's home
// position before any travel this game, and what AtHome compares the
// current-position field against for the Flag1 hover gate (`TOWN-120`/
// `TOWN-122`, High).
func worldMapHomePoint(a *worldMapAssets) image.Point {
	if a != nil && a.data != nil && len(a.data.Objects) > 0 && a.data.Objects[0].Valid {
		return a.data.Objects[0].Point
	}
	return image.Pt(320, 240)
}

func (t *townScreen) enterWorldMap() {
	a := t.art.worldMap()
	if !t.worldPositionSet {
		t.worldPosition, t.worldPositionSet = worldMapHomePoint(a), true
		if first, restored := t.sess.Town.firstMapPoint(); restored && !first && a != nil && a.data != nil {
			if object, ok := a.data.Missions[t.sess.Town.selectedMission()]; ok && object >= 0 && object < len(a.data.Objects) && a.data.Objects[object].Valid {
				t.worldPosition = a.data.Objects[object].Point
			}
		}
	}
	if t.worldSelectedOnce == nil {
		for _, mission := range t.sess.Town.selectedMarkerMissions() {
			t.markWorldSelected(mission)
		}
	}
	s := &worldMapState{assets: a, selected: -1, hovered: -1, current: t.worldPosition}
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	for _, mission := range t.sess.Town.Available() {
		m := ui.WorldMapMission{Number: mission, Title: fmt.Sprintf("Mission %d", mission), Enabled: true}
		if src != nil {
			words := a.missionText(src, mission)
			m.Title, m.Briefing = words.title, words.brief
		}
		m.Payment = t.sess.Town.campaignPayment(mission)
		object, mapped := -1, false
		if a.data != nil {
			object, mapped = a.data.Missions[mission]
		}
		if !mapped || object < 0 || a.data == nil || object >= len(a.data.Objects) {
			m.Enabled = false
			m.Problem = fmt.Sprintf("Mission %d has no valid map location", mission)
		} else {
			obj := a.data.Objects[object]
			m.Object = object
			m.Anchor, m.Region = obj.Point, obj.Region
			if !obj.Valid {
				m.Enabled = false
				m.Problem = obj.Problem
			}
			// Marker is resolved in WorldMapView, gated on selection history
			// (`DIV-128`), so a mission selected during THIS visit shows its
			// marker immediately rather than only on the next entry.
		}
		s.missions = append(s.missions, m)
	}
	t.worldMap = s
	// An imported away-from-home relation has the selected mission's point
	// (SAV-CAMPPOS-072). Apply the same homeward presentation rather than
	// offering inaccessible scrolls indefinitely (DIV-137).
	if s.current != worldMapHomePoint(a) && t.sess.Town.restoredCampaign() {
		s.returnMission = t.sess.Town.selectedMission()
		s.route = worldMapRoute(a.graph, s.current, worldMapHomePoint(a), worldMapFallbackStep)
	}
}

func (s *worldMapState) hideScrolls() bool {
	return s.returnMission != 0 || s.current != worldMapHomePoint(s.assets)
}

// travellingTo reports whether the party is on its way to mission i: the
// mission is selected and its route is held. Arrival, an immediate open and
// the town scroll clear both, so a held route is travel not yet arrived.
func (s *worldMapState) travellingTo(i int) bool {
	return s.selected == i && len(s.route) > 0
}

func (t *townScreen) AtWorldMap() bool { return t != nil && t.room == roomGates }

// beginWorldMapReturn is the owner-directed post-Victory trigger (DIV-137).
// TOWN-121 supplies route completion into destination object zero. Campaign
// completion has already happened; neither this method nor arrival pays again.
func (t *townScreen) beginWorldMapReturn(mission int) {
	a := t.art.worldMap()
	t.atSquare()
	t.worldMap = nil
	t.worldPosition, t.worldPositionSet = worldMapHomePoint(a), true
	if a.data == nil {
		return // Registry-less diagnostics have no mission location to traverse.
	}
	object, ok := a.data.Missions[mission]
	if !ok || object < 0 || object >= len(a.data.Objects) || !a.data.Objects[object].Valid {
		return
	}
	t.worldPosition = a.data.Objects[object].Point
	t.room = roomGates
	t.enterWorldMap() // Town.Available now excludes the completed mission.
	t.worldMap.returnMission = mission
	t.worldMap.route = worldMapRoute(a.graph, t.worldPosition, worldMapHomePoint(a), worldMapFallbackStep)
}

func (t *townScreen) WorldMapView() ui.WorldMapView {
	if t == nil || t.worldMap == nil {
		return ui.WorldMapView{Hovered: -1, Selected: -1, Problem: "world map is unavailable"}
	}
	s, a := t.worldMap, t.worldMap.assets
	cardBase := ui.WorldMapCardPageBase(s.selected)
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	missions := append([]ui.WorldMapMission(nil), s.missions...)
	for i := range missions {
		// `DIV-128`: a picture-bearing mission's own marker paints only once
		// it has been selected at least once this game (`TOWN-123`/
		// `TOWN-040`, High).
		if missions[i].Enabled && src != nil && t.worldSelectedOnce[missions[i].Number] {
			obj := a.data.Objects[missions[i].Object]
			missions[i].Marker = worldMapOptionalImage(a.marker(src, missions[i].Number, obj))
		}
	}
	var markers []ui.WorldMapMarker
	if src != nil && a.data != nil {
		shown := make(map[int]bool, len(missions))
		for _, m := range missions {
			shown[m.Number] = true
		}
		for _, mission := range sortedSelectedMissions(t.worldSelectedOnce) {
			object, ok := a.data.Missions[mission]
			if shown[mission] || !ok || object < 0 || object >= len(a.data.Objects) {
				continue
			}
			obj := a.data.Objects[object]
			if picture := worldMapOptionalImage(a.marker(src, mission, obj)); picture != nil {
				markers = append(markers, ui.WorldMapMarker{Anchor: obj.Point, Picture: picture})
			}
		}
	}
	return ui.WorldMapView{
		Markers:    markers,
		Background: worldMapOptionalImage(a.background), Ball: worldMapOptionalImage(a.ball),
		Flag: worldMapOptionalImage(a.flagFrame(s.frame)), Available: worldMapOptionalImage(a.availableFrame(t.worldFlag1Frame)),
		Cross: worldMapOptionalImage(a.crossFrame(s.cross)),
		Scroll: [3]image.Image{
			worldMapOptionalImage(a.scroll[0]), worldMapOptionalImage(a.scroll[1]), worldMapOptionalImage(a.scroll[2]),
		},
		Pressed: [3]image.Image{
			worldMapOptionalImage(a.pressed[0]), worldMapOptionalImage(a.pressed[1]), worldMapOptionalImage(a.pressed[2]),
		},
		Font: t.art.documentFont(), DetailFont: t.in.tipFont(), ScrollHovered: s.scrollHovered, Words: t.in.Words, Missions: missions, CardBase: cardBase,
		Hovered: s.hovered, Selected: s.selected, Route: append([]image.Point(nil), s.route...),
		RouteShown: s.shown, Position: s.current, AtHome: s.current == worldMapHomePoint(a),
		HideScrolls: s.hideScrolls(),
		Returning:   s.returnMission != 0, Destination: worldMapHomePoint(a),
		Problem: a.problem,
	}
}

func worldMapOptionalImage(pic *image.RGBA) image.Image {
	if pic == nil {
		return nil
	}
	return pic
}

func (t *townScreen) WorldMapHover(p image.Point) {
	if t == nil || t.worldMap == nil {
		return
	}
	v := t.WorldMapView()
	t.worldMap.scrollHovered = false
	if i, ok := ui.WorldMapCardAt(v, p); ok {
		t.worldMap.scrollHovered = true
		t.worldMap.hovered = i
		return
	}
	if i, ok := ui.WorldMapRegionAt(v, p); ok {
		t.worldMap.hovered = i
		return
	}
	t.worldMap.hovered = -1
}

// WorldMapMove selects the next enabled mission before the selected one for
// delta -1 (Up, the wheel away from the user) and after it for +1 (Down, the
// wheel toward the user). Reaching the mission the party is already travelling
// to selects nothing, so its route and reveal keep running (DIV-1515), as a
// click on its scroll leaves them (DIV-1479).
func (t *townScreen) WorldMapMove(delta int) {
	if t == nil || t.worldMap == nil || len(t.worldMap.missions) == 0 || delta == 0 {
		return
	}
	if t.worldMap.hideScrolls() {
		return
	}
	i := t.worldMap.selected
	if i < 0 && delta < 0 {
		// A backward move from the before-first sentinel begins after the last
		// item, just as a forward move begins before the first.
		i = 0
	}
	for attempts := 0; attempts < len(t.worldMap.missions); attempts++ {
		i = (i + delta + len(t.worldMap.missions)) % len(t.worldMap.missions)
		if t.worldMap.missions[i].Enabled {
			if !t.worldMap.travellingTo(i) {
				t.selectWorldMission(i)
			}
			return
		}
	}
}

// WorldMapChoose is Enter. The original's key-down and character slots call
// the progress helper a missed scroll click calls, without reading the key
// value or the selection (`TOWN-485`, `TOWN-486`, High; `TOWN-487` binds Return
// through the campaign). So Enter does what a click that misses every scroll
// does at the same moment (skipTravel): nothing at zero progress, otherwise the
// reveal to the route's end and the Cross counter assigned its frame count plus
// one, and the next tick arrives. It never opens a mission itself and reports
// no mission problem (DIV-1546).
func (t *townScreen) WorldMapChoose() ui.TownAction {
	if t == nil || t.worldMap == nil {
		return ui.TownAction{}
	}
	t.worldMap.skipTravel()
	return ui.TownAction{}
}

// WorldMapClick selects through the mission scrolls only (`TOWN-118`, High):
// a mission is selected by clicking its scroll entry, and the map-rectangle
// override hit-tests a region without calling mission selection or route
// construction (DIV-105). A scroll click starts travel to its mission, and a
// click on another mission's scroll restarts travel toward that one. A click
// on the scroll of the mission already being travelled to leaves the route and
// its reveal running (DIV-1479). There is no second click that opens a mission
// — arrival opens it by itself (`TOWN-121`, High; `DIV-135` closed). A click
// that lands on no card while travel is in progress, a mission's map region
// included, is the miss-a-scroll skip arm: once the reveal has begun it sets
// the reveal to the route's own end and the Cross counter past its end, and
// the very next WorldMapTick call opens the destination. `TOWN-121` skips only
// at non-zero route progress, so the same click before the first tick does
// nothing (DIV-1516).
func (t *townScreen) WorldMapClick(p image.Point) ui.TownAction {
	if t == nil || t.worldMap == nil {
		return ui.TownAction{}
	}
	v := t.WorldMapView()
	if i, ok := ui.WorldMapCardAt(v, p); ok {
		if i < 0 {
			t.worldMap.route, t.worldMap.shown, t.worldMap.cross, t.worldMap.selected = nil, 0, 0, -1
			t.atSquare()
			return ui.TownAction{}
		}
		if t.worldMap.travellingTo(i) {
			return ui.TownAction{}
		}
		t.selectWorldMission(i)
		return ui.TownAction{}
	}
	t.worldMap.skipTravel()
	return ui.TownAction{}
}

// WorldMapTick advances the Flag's animation counter, and while a route is
// held the Cross counter by one and the route reveal by 8 coordinates,
// clamped to the route's own length (`TOWN-120`, High). It opens the
// destination, with no second click, the moment the reveal has reached the
// route's own end and the Cross animation has passed its end — from the tick
// that finishes the later of the two or from a skip that WorldMapClick set up
// (`TOWN-121`, High; DIV-1545). The call rate that gates how often this fires
// is authored (`DIV-136`): `TOWN-121` grades the driving message's own
// real-time cadence Unknown.
func (t *townScreen) WorldMapTick() ui.TownAction {
	if t == nil || t.worldMap == nil {
		return ui.TownAction{}
	}
	s := t.worldMap
	s.frame++
	t.worldFlag1Frame++
	if (s.selected < 0 && s.returnMission == 0) || len(s.route) == 0 {
		return ui.TownAction{}
	}
	s.cross++
	// shown may already equal len(s.route) here — the skip arm in
	// WorldMapClick and a Cross animation that outlasts the reveal both leave
	// it there — so this call still falls through to the arrival test rather
	// than treating that as idle.
	shown := s.shown + worldMapRouteReveal
	if shown > len(s.route) {
		shown = len(s.route)
	}
	s.shown = shown
	if shown < len(s.route) || !s.crossPassedEnd() {
		return ui.TownAction{}
	}
	return t.arriveWorldMap()
}

// arriveWorldMap is `TOWN-121`'s own paint-completion step (High): copy
// destination to current, clear the route and selection, and open the
// destination mission. It is reached only once route reveal has consumed the
// whole route and the Cross animation has passed its end, whether by
// WorldMapTick's own counters or by a skip that set them.
func (t *townScreen) arriveWorldMap() ui.TownAction {
	s := t.worldMap
	i := s.selected
	s.route, s.shown, s.cross, s.selected = nil, 0, 0, -1
	if s.returnMission != 0 {
		s.returnMission = 0
		s.current = worldMapHomePoint(s.assets)
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
	m := s.missions[i]
	s.current = m.Anchor
	t.worldPosition = m.Anchor
	if !m.Enabled {
		return ui.TownAction{Msg: m.Problem}
	}
	return t.worldMapMissionAction(m.Number)
}

// selectWorldMission (re)starts travel to mission i from the party's own
// current position (`TOWN-118`, High, for the selection call; `DIV-137` for
// the position it starts from — the position no longer resets to the town's
// own point on every selection, only on a return to town).
func (t *townScreen) selectWorldMission(i int) {
	if t == nil || t.worldMap == nil || i < 0 || i >= len(t.worldMap.missions) {
		return
	}
	m := t.worldMap.missions[i]
	t.worldMap.selected = i
	t.worldMap.route, t.worldMap.shown, t.worldMap.cross = nil, 0, 0
	if !m.Enabled {
		return
	}
	t.worldMap.route = worldMapRoute(t.worldMap.assets.graph, t.worldMap.current, m.Anchor, worldMapFallbackStep)
	if t.sess != nil {
		t.sess.Town.selectMission(m.Number)
	}
	t.markWorldSelected(m.Number)
}

// markWorldSelected records a registered or selected mission when its MapObject
// carries actual marker art (`TOWN-123`/`TOWN-040`, High). This is the sole
// writer-side population rule for the persisted marker-history set; snapshot
// encode/decode deliberately preserves older superset saves unchanged.
func (t *townScreen) markWorldSelected(mission int) {
	if t == nil {
		return
	}
	var data *globalMapData
	if t.worldMap != nil && t.worldMap.assets != nil {
		data = t.worldMap.assets.data
	}
	if data == nil && t.sess != nil {
		if assets := t.art.worldMap(); assets != nil {
			data = assets.data
		}
	}
	if !worldMapMarkerHasPicture(data, mission) {
		return
	}
	if t.worldSelectedOnce == nil {
		t.worldSelectedOnce = make(map[int]bool)
	}
	t.worldSelectedOnce[mission] = true
}

// worldMapMarkerHasPicture is markWorldSelected's own population rule,
// factored out so a second caller can ask the same question without a
// *townScreen and without mutating worldSelectedOnce (originalsave.go's own
// frontWorldMapFilteredMarkers).
func worldMapMarkerHasPicture(data *globalMapData, mission int) bool {
	if data == nil {
		return false
	}
	object, ok := data.Missions[mission]
	if !ok || object < 0 || object >= len(data.Objects) {
		return false
	}
	row := data.Objects[object]
	return row.Valid && row.Picture != "" && !strings.EqualFold(row.Picture, "nothing")
}

// frontWorldMapFilteredMarkers computes what enterWorldMap's own lazy
// reconstruction (above) would populate a fresh townScreen's
// worldSelectedOnce with, straight from Town's own persisted
// selectedMarkerMissions(), without requiring a *townScreen or mutating
// anything.
func frontWorldMapFilteredMarkers(front *FrontEnd) []int {
	if front == nil || front.Town == nil {
		return nil
	}
	var data *globalMapData
	if assets := front.worldMapAssets(); assets != nil {
		data = assets.data
	}
	return worldMapMarkerMissions(front.Town, data)
}

// worldMapMarkerMissions is the sorted list of the town's selected-marker
// missions that have a picture in data.
func worldMapMarkerMissions(town *Town, data *globalMapData) []int {
	var out []int
	for _, mission := range town.selectedMarkerMissions() {
		if worldMapMarkerHasPicture(data, mission) {
			out = append(out, mission)
		}
	}
	sortInts(out)
	return out
}

// The away point is committed before the opener runs. A refused opener must
// therefore return home too; leaving it idle would hide every retry scroll.
// No completion, payment or party carry occurs on this recovery path.
func (t *townScreen) worldMapMissionAction(mission int) ui.TownAction {
	open := t.openMission(mission)
	return ui.TownAction{
		Msg:  fmt.Sprintf("travelling to mission %d", mission),
		Info: true,
		Open: func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect,
			ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
			v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
			if err != nil {
				t.beginWorldMapReturn(mission)
			}
			return v, tick, order, cadence, affect, advance, attack, grab, stance, march, err
		},
	}
}

// worldMapRoute builds the travel route the way the original does: the node
// graph's least-accumulated-length chain, expanded to its own corridor
// pixels (`TOWN-119`, High). When the graph is absent, or either endpoint is
// not itself a graph node, or the two endpoints have no connecting chain, it
// falls back to a straight sampled line (DIV-129, authored: no shipped
// registry `MapPoint` exercises this path).
func worldMapRoute(graph *worldMapNodeGraph, start, end image.Point, step int) []image.Point {
	if step < 1 {
		step = 1
	}
	if route, ok := graph.Route(start, end); ok {
		return route
	}
	return sampledLine(start, end, step)
}

func sampledLine(start, end image.Point, step int) []image.Point {
	dx, dy := end.X-start.X, end.Y-start.Y
	n := absInt(dx)
	if absInt(dy) > n {
		n = absInt(dy)
	}
	if n == 0 {
		return []image.Point{start}
	}
	var out []image.Point
	for i := 0; i <= n; i += step {
		out = append(out, image.Pt(start.X+dx*i/n, start.Y+dy*i/n))
	}
	if out[len(out)-1] != end {
		out = append(out, end)
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sortedSelectedMissions(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for n, ok := range m {
		if ok {
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

func sortedMissionNumbers(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// WorldMapWitness drives the production town/building/gates/mission path on a
// newly constructed FrontEnd. It is for the developer check command; tests use
// synthetic fixtures and never call it against an install.
func (f *FrontEnd) WorldMapWitness() (string, error) {
	if f == nil || f.Town == nil {
		return "", fmt.Errorf("world map witness: no town")
	}
	f.Town.Arrive()
	s := f.TownScreen().(*townScreen)
	type candidate struct {
		door    int
		row     int
		mission int
	}
	var chosen candidate
	found := false
	for _, c := range []struct {
		building TownBuilding
		door     int
	}{
		{TownTavern, 0}, {TownShop, 1}, {TownSchool, 2},
	} {
		for row, offer := range f.Town.Offers(c.building) {
			if offer.Mission > 0 {
				chosen = candidate{door: c.door, row: row, mission: offer.Mission}
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return "", fmt.Errorf("world map witness: chapter %d has no positive building offer", f.Town.Chapter())
	}

	s.Choose(chosen.door)
	if chosen.door == 0 {
		s.Choose(chosen.row)
	} else if s.room != roomTalk {
		// A missing automatic dialogue keeps the building open. Use its visible
		// offer row where the room supplies one.
		s.Choose(chosen.row + 1)
	}
	for presses := 0; presses < 128 && !containsMission(f.Town.Available(), chosen.mission); presses++ {
		if s.room != roomTalk {
			break
		}
		s.AdvanceTownDialogue()
	}
	if !containsMission(f.Town.Available(), chosen.mission) {
		return "", fmt.Errorf("world map witness: building did not accept mission %d", chosen.mission)
	}
	for s.room != roomSquare {
		if !s.Back() {
			return "", fmt.Errorf("world map witness: could not return from room %d", s.room)
		}
	}
	s.Choose(3)
	view := s.WorldMapView()
	i := -1
	for n, mission := range view.Missions {
		if mission.Number == chosen.mission {
			i = n
			break
		}
	}
	if i < 0 {
		return "", fmt.Errorf("world map witness: accepted mission %d has no scroll", chosen.mission)
	}
	m := view.Missions[i]
	if !m.Enabled || m.Title == "" || m.Briefing == "" {
		return "", fmt.Errorf("world map witness: mission %d scroll is incomplete: enabled=%v title=%d briefing=%d problem=%s",
			chosen.mission, m.Enabled, len(m.Title), len(m.Briefing), m.Problem)
	}
	if !view.AtHome {
		return "", fmt.Errorf("world map witness: gates entered away from home: position %v", view.Position)
	}
	card := ui.WorldMapCardRect(i)
	centre := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	if act := s.WorldMapClick(centre); act.Open != nil {
		return "", fmt.Errorf("world map witness: scroll release opened mission %d before travel completed", chosen.mission)
	}
	view = s.WorldMapView()
	if len(view.Route) < 2 || view.Selected != i {
		return "", fmt.Errorf("world map witness: mission %d produced route length %d and selection %d",
			chosen.mission, len(view.Route), view.Selected)
	}
	routeLen := len(view.Route)
	// One tick short of a full reveal, then a click that misses every scroll:
	// `TOWN-121`'s own skip arm fast-forwards the rest of the reveal
	// (High). No scroll is ever clicked a second time.
	if act := s.WorldMapTick(); act.Open != nil {
		return "", fmt.Errorf("world map witness: mission %d opened on the first tick", chosen.mission)
	}
	miss := image.Pt(2, 2)
	if _, ok := ui.WorldMapCardAt(s.WorldMapView(), miss); ok {
		return "", fmt.Errorf("world map witness: chosen miss point %v hits a card", miss)
	}
	s.WorldMapClick(miss)
	if shown := s.WorldMapView().RouteShown; shown != routeLen {
		return "", fmt.Errorf("world map witness: miss click did not skip to the route's own end: shown %d, want %d", shown, routeLen)
	}
	act := s.WorldMapTick()
	if act.Open == nil {
		return "", fmt.Errorf("world map witness: mission %d did not open on the completing tick", chosen.mission)
	}
	view = s.WorldMapView()
	if view.Selected != -1 || len(view.Route) != 0 {
		return "", fmt.Errorf("world map witness: arrival left selection %d and route length %d, want both cleared", view.Selected, len(view.Route))
	}
	if view.Position != m.Anchor {
		return "", fmt.Errorf("world map witness: arrival position %v, want the destination %v", view.Position, m.Anchor)
	}
	frame := ui.ComposeWorldMap(view)
	if frame == nil || frame.Bounds() != image.Rect(0, 0, 640, 480) {
		return "", fmt.Errorf("world map witness: composed frame has invalid bounds")
	}
	viewer, _, _, _, _, _, _, _, _, _, err := act.Open()
	if err != nil {
		return "", fmt.Errorf("world map witness: open mission %d: %w", chosen.mission, err)
	}
	if viewer == nil {
		return "", fmt.Errorf("world map witness: mission %d opener returned no viewer", chosen.mission)
	}
	return fmt.Sprintf("world map: chapter %d accepted mission %d; %d scroll(s); title %d bytes; briefing %d bytes; payment %d; route %d points; one click, skip completed travel, no second click; arrived at %v; frame composed; mission opened",
		f.Town.Chapter(), chosen.mission, len(view.Missions), len(m.Title), len(m.Briefing), m.Payment, routeLen, view.Position), nil
}

func containsMission(list []int, mission int) bool {
	for _, n := range list {
		if n == mission {
			return true
		}
	}
	return false
}

// WorldMapSweep checks every shipped mapping through the production adapter.
// It is separate from WorldMapWitness because the witness follows one campaign
// chapter while this check expands the live set temporarily to exercise every
// data row and both marker outcomes.
func (f *FrontEnd) WorldMapSweep() (string, error) {
	if f == nil || f.Town == nil || f.Archives == nil || f.Archives.Containers == nil {
		return "", fmt.Errorf("world map sweep: incomplete front end")
	}
	a := f.worldMapAssets()
	if a.data == nil {
		return "", fmt.Errorf("world map sweep: %s", a.problem)
	}
	type fixedAsset struct {
		name string
		ok   bool
	}
	fixed := []fixedAsset{
		{"background", a.background != nil}, {"path mask", a.path != nil},
		{"route stamp", a.ball != nil},
		{"current-position flag", len(a.flag) > 0}, {"hover flag", len(a.available) > 0},
		{"destination cross", len(a.cross) > 0},
	}
	for i := range a.scroll {
		fixed = append(fixed,
			fixedAsset{fmt.Sprintf("scroll %d", i+1), a.scroll[i] != nil},
			fixedAsset{fmt.Sprintf("pressed scroll %d", i+1), a.pressed[i] != nil},
		)
	}
	for _, asset := range fixed {
		if !asset.ok {
			return "", fmt.Errorf("world map sweep: shipped %s did not decode", asset.name)
		}
	}
	numbers := sortedMissionNumbers(a.data.Missions)
	mainCount, sideCount, mappedOnly, offeredCount := 0, 0, 0, 0
	for _, mission := range numbers {
		object := a.data.Missions[mission]
		if object < 0 || object >= len(a.data.Objects) || !a.data.Objects[object].Valid {
			return "", fmt.Errorf("world map sweep: mission %d maps to invalid object %d", mission, object+1)
		}
		if containsMission(f.Campaign.Value().Main, mission) {
			mainCount++
		} else if containsMission(f.Campaign.Value().Side, mission) {
			sideCount++
		} else {
			mappedOnly++
		}
		if containsMission(f.Campaign.Value().Offered, mission) {
			offeredCount++
		}
		title, brief, _ := WorldMapTextPaths(mission)
		for _, path := range []string{title, brief} {
			raw, err := f.Archives.Containers.ReadFile(path)
			if err != nil || len(raw) == 0 {
				return "", fmt.Errorf("world map sweep: mission %d missing non-empty %s", mission, path)
			}
		}
	}

	old := f.Town.available
	f.Town.available = make(map[int]bool, len(numbers))
	for _, mission := range numbers {
		f.Town.available[mission] = true
	}
	s := f.TownScreen().(*townScreen)
	// The gate this sweep checks is selection HISTORY (`DIV-128`), so any
	// prior selection (a real visit, or `WorldMapWitness` run earlier in the
	// same process — `cmd/worldmapcheck`) must not leak in and must not be
	// disturbed by this temporary check.
	savedSelected := s.worldSelectedOnce
	s.worldSelectedOnce = nil
	defer func() { s.worldSelectedOnce = savedSelected }()
	s.enterWorldMap()
	f.Town.available = old
	view := s.WorldMapView()
	if len(view.Missions) != len(numbers) {
		return "", fmt.Errorf("world map sweep: %d mappings produced %d scrolls", len(numbers), len(view.Missions))
	}
	// `DIV-128` (`TOWN-123`/`TOWN-040`, High): before any selection, no
	// mission's marker paints, picture-bearing or not — across every shipped
	// mapping, not only the one mission `WorldMapWitness` selects.
	for i, m := range view.Missions {
		if m.Marker != nil {
			return "", fmt.Errorf("world map sweep: mission %d painted a marker before ever being selected", numbers[i])
		}
	}
	frame := ui.ComposeWorldMap(view)
	if frame == nil || frame.Bounds() != image.Rect(0, 0, 640, 480) {
		return "", fmt.Errorf("world map sweep: composed frame has invalid bounds")
	}
	for _, mission := range numbers {
		s.markWorldSelected(mission)
	}
	view = s.WorldMapView()
	// pictureCount is missions whose object carries a picture, which paint a
	// marker once selected. noPictureCount is missions whose object's
	// Picture is "nothing" (or empty): no claim gives evidence ROM1 draws
	// anything for one, so this build paints no placeholder for it either —
	// these missions show nothing on the map surface at all, selected or
	// not (`DIV-107`).
	pictureCount, noPictureCount := 0, 0
	for i, m := range view.Missions {
		if !m.Enabled || m.Number != numbers[i] || m.Title == "" || m.Briefing == "" {
			return "", fmt.Errorf("world map sweep: scroll %d is incomplete: %+v", i, m)
		}
		object := a.data.Missions[m.Number]
		obj := a.data.Objects[object]
		hasPicture := obj.Picture != "" && !strings.EqualFold(obj.Picture, "nothing")
		selected, recorded := s.worldSelectedOnce[m.Number]
		if hasPicture && (!recorded || !selected) {
			return "", fmt.Errorf("world map sweep: mission %d picture %q was not retained by the selection writer", m.Number, obj.Picture)
		}
		if !hasPicture && recorded {
			return "", fmt.Errorf("world map sweep: mission %d without marker art entered selection history", m.Number)
		}
		if hasPicture && m.Marker == nil {
			return "", fmt.Errorf("world map sweep: mission %d picture %q did not decode once selected", m.Number, obj.Picture)
		}
		if !hasPicture && m.Marker != nil {
			return "", fmt.Errorf("world map sweep: mission %d has no picture but painted a marker", m.Number)
		}
		if hasPicture {
			pictureCount++
		} else {
			noPictureCount++
		}
	}
	return fmt.Sprintf("world map sweep: %d mappings, %d title/briefing pairs, %d main, %d side and %d mapping-only rows; %d building-offered; marker cache gate held before selection for all; %d picture marker(s) painted and %d with no picture drawing nothing, once selected; frame composed; all consumed",
		len(numbers), len(numbers), mainCount, sideCount, mappedOnly, offeredCount, pictureCount, noPictureCount), nil
}

// WorldMapGraphWitness reports the installed route graph's corpus counts. It
// reads the same assets the world map itself loads and changes no state.
func (f *FrontEnd) WorldMapGraphWitness() (string, error) {
	if f == nil || f.Archives == nil || f.Archives.Containers == nil {
		return "", fmt.Errorf("world map graph witness: incomplete front end")
	}
	a := f.worldMapAssets()
	if a.path == nil {
		return "", fmt.Errorf("world map graph witness: path mask did not decode: %s", a.problem)
	}
	var points []image.Point
	if a.data != nil {
		for _, o := range a.data.Objects {
			if o.Valid {
				points = append(points, o.Point)
			}
		}
	}
	return worldMapGraphDigest(a.path, points), nil
}

// WorldMapCriterionWitness reports how far apart the two readings of
// `TOWN-119` are on the shipped graph, measured through the same search the
// production route uses. See worldMapCriterionDigest for why it is committed
// rather than measured by a probe when a document needs the number.
func (f *FrontEnd) WorldMapCriterionWitness() (string, error) {
	if f == nil || f.Archives == nil || f.Archives.Containers == nil {
		return "", fmt.Errorf("world map criterion witness: incomplete front end")
	}
	a := f.worldMapAssets()
	if a.path == nil {
		return "", fmt.Errorf("world map criterion witness: path mask did not decode: %s", a.problem)
	}
	var points []image.Point
	if a.data != nil {
		for _, o := range a.data.Objects {
			if o.Valid {
				points = append(points, o.Point)
			}
		}
	}
	return worldMapCriterionDigest(a.path, points), nil
}
