package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"strings"

	rendertext "againrom/pkg/render/text"
)

// WorldMapMission is one live campaign offer as the Client can present it.
// The mission number and opener remain on the Town & Economy side of the
// TownWorldMapScreen seam. This value carries only pixels, text and geometry.
type WorldMapMission struct {
	Number   int
	Title    string
	Briefing string
	Payment  int
	Anchor   image.Point
	Region   image.Rectangle
	Object   int
	// Kind is the record's kind as its campaign numbers it; a map's text
	// keys are chosen by it.
	Kind   int
	Marker image.Image
	// MarkerAt places Marker. Its zero value keeps drawMarker's own rule: a
	// whole-map overlay at its own coordinates, a smaller picture centred.
	MarkerAt WorldMapPlace
	Enabled  bool
	Problem  string
}

// WorldMapPlace places a sprite at a point. Its zero value centres the
// sprite on the point; TopLeft puts the sprite's top-left corner at the point
// plus Offset. A map's description supplies every placement.
type WorldMapPlace struct {
	Offset  image.Point
	TopLeft bool
}

// WorldMapTextBlock is a text block a map draws over its records: one name
// line at NameAt, and Body wrapped to Width in lines from LinesAt, LineStep
// apart, at most MaxLines of them.
type WorldMapTextBlock struct {
	Name     string
	Body     string
	NameAt   image.Point
	LinesAt  image.Point
	LineStep int
	Width    int
	MaxLines int
	NameFont *rendertext.Font
	LineFont *rendertext.Font
}

// WorldMapTextLines are the lines a text block draws its body in.
func WorldMapTextLines(t *WorldMapTextBlock) []string {
	if t == nil || t.LineFont == nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(t.Body, "\r", ""), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, wrapTooltipLine(line, t.LineFont, t.Width)...)
	}
	if t.MaxLines > 0 && len(lines) > t.MaxLines {
		lines = lines[:t.MaxLines]
	}
	return lines
}

// WorldMapMarker is one cached mission marker picture. It is drawn whether or
// not its mission still has a scroll, because a marker outlives the mission's
// offer (`TOWN-123`: nothing removes a marker in play).
type WorldMapMarker struct {
	Anchor  image.Point
	Picture image.Image
}

// WorldMapView is the complete 640x480 presentation snapshot for the campaign
// map. RouteShown is a prefix length so animation state stays on the screen
// owner while composition remains a pure function. Flag and Cross already
// carry the current animation frame (`TOWN-120`, High); the owner picks which
// frame outside this package, once per WorldMapTick call.
type WorldMapView struct {
	Background image.Image
	Ball       image.Image
	// Flag is the party's own current-position marker (`TOWN-120`, High),
	// drawn at Position. Available is the task Flag1, drawn while AtHome at
	// the hovered mission's anchor, or the selected mission's from the choice
	// until arrival (`TOWN-528`). Cross is the destination marker.
	Flag        image.Image
	Available   image.Image
	Cross       image.Image
	FlagAt      WorldMapPlace
	AvailableAt WorldMapPlace
	CrossAt     WorldMapPlace
	// MarkersOverRoute draws the mission markers after the route dots
	// instead of before them.
	MarkersOverRoute bool
	// Text is drawn last, over every sprite, when present.
	Text *WorldMapTextBlock
	// PointerPicks reports that the pointer targets the nearest record and a
	// press travels to it, rather than the scroll cards choosing.
	PointerPicks  bool
	Scroll        [3]image.Image
	Pressed       [3]image.Image
	Font          *rendertext.Font
	DetailFont    *rendertext.Font
	ScrollHovered bool
	Words         Words

	Missions []WorldMapMission
	// Markers holds the cached markers of missions that have no scroll in
	// Missions, such as a completed mission.
	Markers  []WorldMapMarker
	CardBase int
	Hovered  int
	Selected int

	Route      []image.Point
	RouteShown int
	// Position is the party's own current-position field (`TOWN-120`/
	// `TOWN-121`, High): it changes only at travel completion, never as the
	// route reveals. AtHome reports whether Position equals the first
	// MapObject's own point — the Flag1 hover gate (`TOWN-122`, High).
	Position image.Point
	AtHome   bool
	// HideScrolls applies TOWN-122's home-only list gate to painting and hits.
	// Returning carries the destination-zero route, which has no mission row.
	HideScrolls bool
	Returning   bool
	Destination image.Point

	Problem string
	Message string
}

// TownWorldMapScreen is the optional graphical replacement for the gates'
// legacy mission list. It keeps campaign and opener decisions outside pkg/ui.
type TownWorldMapScreen interface {
	AtWorldMap() bool
	WorldMapView() WorldMapView
	WorldMapHover(image.Point)
	WorldMapMove(int)
	WorldMapChoose() TownAction
	WorldMapClick(image.Point) TownAction
	// WorldMapTick advances one paint's worth of animation and route reveal,
	// and reports the arrival action once travel completes — with no second
	// click (`TOWN-121`, High).
	WorldMapTick() TownAction
}

func townWorldMapScreen(t TownScreen) (TownWorldMapScreen, bool) {
	w, ok := t.(TownWorldMapScreen)
	return w, ok && w.AtWorldMap()
}

const (
	worldMapCardW = 164
	worldMapCardH = 40
	// worldMapTextDrop lowers the title and detail lines so they sit centred
	// between the scroll's two rolls.
	worldMapTextDrop   = 2
	worldMapCardPage   = 3
	worldMapBallStride = 8
)

// The town scroll stays on the right; mission scrolls fill the row from left.
func WorldMapCardRect(slot int) image.Rectangle {
	if slot < 0 {
		return image.Rect(520, 0, 640, worldMapCardH)
	}
	x := 4 + slot*(worldMapCardW+4)
	return image.Rect(x, 0, x+worldMapCardW, worldMapCardH)
}

func worldMapCardDetail(v WorldMapView, index int) []string {
	if !v.ScrollHovered || v.Hovered != index {
		return nil
	}
	font := v.DetailFont
	if font == nil {
		font = v.Font
	}
	if font == nil {
		return nil
	}
	detail, payment := v.Words.WorldHomeDetail, 0
	width := WorldMapCardRect(-1).Dx() - 20
	if index >= 0 && index < len(v.Missions) {
		m := v.Missions[index]
		detail, payment, width = m.Briefing, m.Payment, worldMapCardW-20
		if m.Problem != "" {
			detail = m.Problem
		}
	} else if detail == "" {
		detail = AuthoredWords().WorldHomeDetail
	}
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(detail, "\r", ""), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, wrapTooltipLine(line, font, width)...)
		}
	}
	if payment > 0 {
		label := v.Words.WorldPayment
		if label == "" {
			label = AuthoredWords().WorldPayment
		}
		lines = append(lines, wrapTooltipLine(fmt.Sprintf("%s: %d", label, payment), font, width)...)
	}
	return lines
}

func worldMapCardRect(v WorldMapView, index, slot int) image.Rectangle {
	r := WorldMapCardRect(slot)
	font := v.DetailFont
	if font == nil {
		font = v.Font
	}
	if font != nil {
		if n := len(worldMapCardDetail(v, index)); n > 0 {
			r.Max.Y += n*font.Height() + worldMapTextDrop
		}
	}
	return r
}

func WorldMapCardPageBase(index int) int {
	if index < 0 {
		return 0
	}
	return (index / worldMapCardPage) * worldMapCardPage
}

// WorldMapCardAt gives cards priority over map regions. The separate town card
// is reported as -1; false means the pointer belongs to the map below them.
func WorldMapCardAt(v WorldMapView, p image.Point) (int, bool) {
	if v.HideScrolls {
		return 0, false
	}
	if p.In(worldMapCardRect(v, -1, -1)) {
		return -1, true
	}
	base := minInt(v.CardBase, len(v.Missions))
	if base < 0 {
		base = 0
	}
	end := minInt(len(v.Missions), base+worldMapCardPage)
	for i := base; i < end; i++ {
		if p.In(worldMapCardRect(v, i, i-base)) {
			return i, true
		}
	}
	return 0, false
}

// WorldMapRegionAt applies the registry's linear object-order rule. Missions
// without a valid rectangle remain card-selectable but cannot win a map hit.
func WorldMapRegionAt(v WorldMapView, p image.Point) (int, bool) {
	best, object := -1, int(^uint(0)>>1)
	for i, m := range v.Missions {
		if !m.Enabled || m.Region.Empty() || !p.In(m.Region) {
			continue
		}
		// Object keeps the registry's declared MapObject index. Equal object
		// indices retain the live campaign order.
		candidate := m.Object
		if best < 0 || candidate < object {
			best, object = i, candidate
		}
	}
	return best, best >= 0
}

var (
	worldMapFallback = color.RGBA{R: 28, G: 38, B: 35, A: 255}
	worldMapInk      = color.RGBA{R: 43, G: 29, B: 18, A: 255}
	worldMapPaper    = color.RGBA{R: 214, G: 188, B: 129, A: 245}
	worldMapPaperHot = color.RGBA{R: 239, G: 218, B: 162, A: 255}
	worldMapBorder   = color.RGBA{R: 82, G: 54, B: 28, A: 255}
	worldMapRoute    = color.RGBA{R: 245, G: 214, B: 76, A: 255}
	worldMapMark     = color.RGBA{R: 190, G: 41, B: 31, A: 255}
)

// worldMapTaskFlag is the mission whose anchor carries the task Flag1, or -1.
// The flag is drawn only while the party stands on the first MapObject
// (`TOWN-528`). Before a choice it follows the hovered task; a chosen task
// keeps it from the choice through the route until arrival, whatever the
// pointer does.
func worldMapTaskFlag(v WorldMapView) int {
	if !v.AtHome || v.Returning {
		return -1
	}
	i := v.Hovered
	if v.Selected >= 0 {
		i = v.Selected
	}
	if i < 0 || i >= len(v.Missions) || !v.Missions[i].Enabled {
		return -1
	}
	return i
}

// ComposeWorldMap creates one native-size frame. It is CPU-only and therefore
// testable without an Ebitengine window or a lawful install.
func ComposeWorldMap(v WorldMapView) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: worldMapFallback}, image.Point{}, draw.Src)
	if worldMapImagePresent(v.Background) {
		draw.Draw(dst, dst.Bounds(), v.Background, v.Background.Bounds().Min, draw.Src)
	}

	// Paint order follows `TOWN-120` (High): cached mission markers, route
	// stamps, the animated destination cross, the task Flag1 while at home,
	// the animated current-position flag, then scroll cards below. A map whose
	// description draws markers over the route draws them after the stamps.
	// `DIV-128`: a mission with no cached marker (never selected, or its own
	// Picture is "nothing") paints nothing here — no placeholder dot.
	if !v.MarkersOverRoute {
		drawWorldMapMarkers(dst, v)
	}

	shown := v.RouteShown
	if shown < 0 {
		shown = 0
	}
	if shown > len(v.Route) {
		shown = len(v.Route)
	}
	for i := 0; i < shown; i += worldMapBallStride {
		stampAt(dst, v.Ball, v.Route[i], worldMapRoute, 2)
	}
	if v.MarkersOverRoute {
		drawWorldMapMarkers(dst, v)
	}
	if len(v.Route) > 0 && v.Returning && (v.Destination != v.Position || !v.AtHome) {
		placeSprite(dst, v.Cross, v.Destination, v.CrossAt, worldMapMark, 4)
	} else if len(v.Route) > 0 && v.Selected >= 0 && v.Selected < len(v.Missions) && v.Missions[v.Selected].Enabled && (v.Missions[v.Selected].Anchor != v.Position || !v.AtHome) {
		placeSprite(dst, v.Cross, v.Missions[v.Selected].Anchor, v.CrossAt, worldMapMark, 4)
	}
	if i := worldMapTaskFlag(v); i >= 0 {
		placeSprite(dst, v.Available, v.Missions[i].Anchor, v.AvailableAt, color.RGBA{R: 220, G: 210, B: 175, A: 255}, 3)
	}
	placeSprite(dst, v.Flag, v.Position, v.FlagAt, color.RGBA{R: 242, G: 235, B: 211, A: 255}, 4)

	words := v.Words
	authored := AuthoredWords()
	if words.WorldHomeTitle == "" {
		words.WorldHomeTitle = authored.WorldHomeTitle
	}
	if words.WorldHomeDetail == "" {
		words.WorldHomeDetail = authored.WorldHomeDetail
	}
	if words.WorldPayment == "" {
		words.WorldPayment = authored.WorldPayment
	}
	if !v.HideScrolls {
		drawWorldMapCard(dst, v, -1, -1, words.WorldHomeTitle, words.WorldHomeDetail, 0)
	}
	base := minInt(v.CardBase, len(v.Missions))
	if base < 0 {
		base = 0
	}
	end := minInt(len(v.Missions), base+worldMapCardPage)
	for i := base; !v.HideScrolls && i < end; i++ {
		m := v.Missions[i]
		detail := m.Briefing
		if m.Problem != "" {
			detail = m.Problem
		}
		drawWorldMapCard(dst, v, i, i-base, m.Title, detail, m.Payment)
	}
	drawWorldMapText(dst, v.Text)
	if v.Problem != "" {
		drawWorldMapLine(dst, v.Font, v.Problem, 12, 454, color.RGBA{R: 255, G: 220, B: 190, A: 255}, 102)
	} else if v.Message != "" {
		drawWorldMapLine(dst, v.Font, v.Message, 12, 454, color.RGBA{R: 255, G: 245, B: 210, A: 255}, 102)
	}
	return dst
}

func drawWorldMapCard(dst *image.RGBA, v WorldMapView, index, slot int, title, _ string, _ int) {
	r := worldMapCardRect(v, index, slot)
	top, paper, bottom := v.Scroll[0], v.Scroll[1], v.Scroll[2]
	if index < 0 {
		top, paper, bottom = v.Pressed[0], v.Pressed[2], v.Pressed[1]
	}
	if worldMapImagePresent(top) && worldMapImagePresent(paper) && worldMapImagePresent(bottom) {
		body := image.Rect(r.Min.X, r.Min.Y+12, r.Max.X, r.Max.Y-12)
		for y := body.Min.Y; y < body.Max.Y; y += paper.Bounds().Dy() {
			part := image.Rect(body.Min.X, y, body.Max.X, min(y+paper.Bounds().Dy(), body.Max.Y))
			draw.Draw(dst, part, paper, paper.Bounds().Min, draw.Over)
		}
		draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+12), top, top.Bounds().Min, draw.Over)
		draw.Draw(dst, image.Rect(r.Min.X, r.Max.Y-12, r.Max.X, r.Max.Y), bottom, bottom.Bounds().Min, draw.Over)
	} else {
		draw.Draw(dst, r, &image.Uniform{C: worldMapPaper}, image.Point{}, draw.Src)
		drawWorldMapBorder(dst, r, worldMapBorder)
	}
	clipped := dst.SubImage(r.Inset(7)).(*image.RGBA)
	if v.Font != nil {
		// The mission title files end in a line break; measuring it would put
		// the visible ink left of the scroll's centre.
		title = strings.TrimSpace(title)
		w, _ := v.Font.Measure(title)
		v.Font.Draw(clipped, title, max(r.Min.X+8, r.Min.X+(r.Dx()-w)/2), r.Min.Y+12+worldMapTextDrop, worldMapInk)
	}
	font := v.DetailFont
	if font == nil {
		font = v.Font
	}
	if font != nil {
		for i, line := range worldMapCardDetail(v, index) {
			w, _ := font.Measure(line)
			font.Draw(clipped, line, r.Min.X+(r.Dx()-w)/2, r.Min.Y+28+worldMapTextDrop+i*font.Height(), worldMapInk)
		}
	}
}

func drawWorldMapLine(dst *image.RGBA, font *rendertext.Font, s string, x, y int, c color.RGBA, maxBytes int) {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", " "), "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if maxBytes > 0 && len(s) > maxBytes {
		s = s[:maxBytes]
	}
	if font != nil {
		font.Draw(dst, s, x, y, c)
	}
}

func drawWorldMapBorder(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+2), &image.Uniform{C: c}, image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Max.Y-2, r.Max.X, r.Max.Y), &image.Uniform{C: c}, image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+2, r.Max.Y), &image.Uniform{C: c}, image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(r.Max.X-2, r.Min.Y, r.Max.X, r.Max.Y), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func stampAt(dst *image.RGBA, pic image.Image, p image.Point, fallback color.RGBA, radius int) {
	if worldMapImagePresent(pic) {
		copyCentred(dst, pic, p)
		return
	}
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				dst.SetRGBA(p.X+x, p.Y+y, fallback)
			}
		}
	}
}

// placeSprite draws pic at p by place. With no picture the fallback dot stays
// centred on p.
func placeSprite(dst *image.RGBA, pic image.Image, p image.Point, place WorldMapPlace, fallback color.RGBA, radius int) {
	if !worldMapImagePresent(pic) || !place.TopLeft {
		stampAt(dst, pic, p, fallback, radius)
		return
	}
	b := pic.Bounds()
	at := p.Add(place.Offset)
	draw.Draw(dst, b.Add(at.Sub(b.Min)), pic, b.Min, draw.Over)
}

// drawWorldMapMarkers draws every enabled mission's marker, then the cached
// markers of missions without a scroll.
func drawWorldMapMarkers(dst *image.RGBA, v WorldMapView) {
	for _, m := range v.Missions {
		if !m.Enabled || !worldMapImagePresent(m.Marker) {
			continue
		}
		if m.MarkerAt.TopLeft {
			placeSprite(dst, m.Marker, m.Anchor, m.MarkerAt, worldMapMark, 0)
			continue
		}
		drawMarker(dst, m.Marker, m.Anchor)
	}
	for _, m := range v.Markers {
		drawMarker(dst, m.Picture, m.Anchor)
	}
}

// drawWorldMapText draws a map's text block: the name, then each line.
func drawWorldMapText(dst *image.RGBA, t *WorldMapTextBlock) {
	if t == nil {
		return
	}
	if t.NameFont != nil && t.Name != "" {
		t.NameFont.Draw(dst, t.Name, t.NameAt.X, t.NameAt.Y, worldMapInk)
	}
	for i, line := range WorldMapTextLines(t) {
		t.LineFont.Draw(dst, line, t.LinesAt.X, t.LinesAt.Y+i*t.LineStep, worldMapInk)
	}
}

// drawMarker places a marker picture. A picture that covers the whole map is
// an overlay and keeps its own coordinates; the shipped registries carry only
// such pictures and no `PictureOffset` (`TOWN-042`). A smaller picture is
// centred on its mission point.
func drawMarker(dst *image.RGBA, src image.Image, anchor image.Point) {
	if dst == nil || !worldMapImagePresent(src) {
		return
	}
	b := src.Bounds()
	if b.Dx() >= dst.Bounds().Dx() && b.Dy() >= dst.Bounds().Dy() {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
		return
	}
	copyCentred(dst, src, anchor)
}

func copyCentred(dst *image.RGBA, src image.Image, p image.Point) {
	if dst == nil || !worldMapImagePresent(src) {
		return
	}
	b := src.Bounds()
	at := image.Pt(p.X-b.Dx()/2, p.Y-b.Dy()/2)
	draw.Draw(dst, b.Add(at.Sub(b.Min)), src, b.Min, draw.Over)
}

// worldMapImagePresent treats a nil pointer stored in image.Image as absent.
// The adapter normalises its own pointers before producing a view; this guard
// also protects the compositor from optional implementations supplied through
// another TownWorldMapScreen.
func worldMapImagePresent(pic image.Image) bool {
	if pic == nil {
		return false
	}
	v := reflect.ValueOf(pic)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return !v.IsNil()
	default:
		return true
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
