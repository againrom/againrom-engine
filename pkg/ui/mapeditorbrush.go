package ui

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"math"
)

// BrushTerrain is one terrain the host lets the window paint. Swatch is a
// picture of the terrain the list shows beside its name; nil shows the name
// alone.
type BrushTerrain struct {
	Class  int
	Name   string
	Swatch *image.RGBA
}

// BrushObject is one row of the objects list the host lets the window paint:
// the class code a cell takes (0 is the row that clears objects) and its name.
type BrushObject struct {
	Code int
	Name string
}

// BrushPlane names the grid layer a stroke writes.
type BrushPlane uint8

const (
	BrushPlaneTerrain BrushPlane = iota
	BrushPlaneHeight
	BrushPlaneObjects
	BrushPlanePassability
)

// BrushShape names the cells a brush covers around each centre.
type BrushShape uint8

const (
	BrushCircle BrushShape = iota // the cells within the radius of the centre
	BrushSquare                   // the cells within the radius on both axes
)

// BrushFillMode names how a terrain stroke fills the centre of the brush:
// the tile pattern the document chooses by position, one named variant of the
// terrain's tiles, or a variant drawn at random for each cell.
type BrushFillMode uint8

const (
	BrushFillPattern BrushFillMode = iota
	BrushFillRandom
	BrushFillVariant
)

// BrushFill is the centre fill of a terrain stroke; Variant counts from 0 and
// is read only for BrushFillVariant.
type BrushFill struct {
	Mode    BrushFillMode
	Variant int
}

// BrushHeightMode names how a Raise or Lower stroke lays its change: Step
// adds it once to every cell the stroke covers, Level sets every covered
// cell to the pressed cell's height plus it, Smooth adds it at every centre
// faded from the centre to the rim. The zero value is Step.
type BrushHeightMode uint8

const (
	HeightModeStep BrushHeightMode = iota
	HeightModeLevel
	HeightModeSmooth
	heightModeCount
)

// BrushStroke is one pointer stroke as data: the plane, the value at each cell
// (a terrain class, a height change, an object class code, or 1 to set and 0
// to clear the tile word's impassable flag), the brush radius in cells, its
// shape, the centre fill of a terrain stroke, the cells the pointer visited
// and, for an object stroke in Forest mode, the forest settings (a stroke
// with ticked kinds plants or thins them instead of writing Value).
type BrushStroke struct {
	Plane  BrushPlane
	Value  int
	Radius int
	Shape  BrushShape
	Fill   BrushFill
	Cells  []image.Point
	Forest BrushForest
	// Set, on the height plane, makes Value the height the first cell takes
	// and the cells within Radius blend toward it, instead of a change added.
	Set bool
	// Flat, with Set, makes every cell the brush covers along Cells take
	// Value, with no blend: the Sample height row's stroke.
	Flat bool
	// Mode, on the height plane without Set, is how the change Value is laid.
	Mode BrushHeightMode
}

// InspectionBrushFill is the optional seam of a host that offers more than
// one tile for a terrain: BrushVariants is how many look-alike tiles the
// terrain class has.
type InspectionBrushFill interface {
	BrushVariants(class int) int
}

// InspectionBrush is the optional seam of a host whose document paints terrain,
// heights and objects. A stroke is one edit: one undo step, one document.
// BrushCells is every map cell a stroke touches, each once, and
// BrushMaxRadius the largest radius a stroke takes.
type InspectionBrush interface {
	BrushTerrains() []BrushTerrain
	BrushObjects() []BrushObject
	Paint(BrushStroke) (*InspectionDocument, error)
	BrushCells(BrushStroke) []image.Point
	BrushMaxRadius() int
}

type brushMode int

const (
	brushOff brushMode = iota
	brushTerrain
	brushHeight
	brushObjects
	brushPassability
	brushModeCount
)

// The brush settings the window offers: the radius a stroke starts with, the
// strength (the height change of a Raise or Lower stroke) it starts with,
// and the strength's bound, the row step bound (TERR-245).
const (
	brushDefaultRadius = 2
	brushHeightStep    = 8
	brushMaxStrength   = 31
)

// brushKeyModes is the brush mode each brush key switches.
var brushKeyModes = map[string]brushMode{"terrain": brushTerrain, "height": brushHeight, "objects": brushObjects, "passability": brushPassability}

// brushModeWord is the word the Tools menu and the list title give mode.
func (e *MapEditor) brushModeWord(mode brushMode) string {
	switch mode {
	case brushTerrain:
		return e.word("editor.brush.terrain")
	case brushHeight:
		return e.word("editor.brush.height")
	case brushObjects:
		return e.word("editor.brush.objects")
	case brushPassability:
		return e.word("editor.brush.passability")
	}
	return ""
}

// terrainWord is the word the terrain list gives the palette row the host
// names name; a name the table lacks reads as the host gave it.
func (e *MapEditor) terrainWord(name string) string {
	switch name {
	case "Grass":
		return e.word("editor.terrain.grass")
	case "Land":
		return e.word("editor.terrain.land")
	case "Sand":
		return e.word("editor.terrain.sand")
	case "Cracked":
		return e.word("editor.terrain.cracked")
	case "Flowers":
		return e.word("editor.terrain.flowers")
	case "Stones":
		return e.word("editor.terrain.stones")
	case "Savanna":
		return e.word("editor.terrain.savanna")
	case "Mountain":
		return e.word("editor.terrain.mountain")
	case "Water":
		return e.word("editor.terrain.water")
	case "Road":
		return e.word("editor.terrain.road")
	}
	return name
}

// objectWord is the word the objects list gives o: its class name, the clear
// row's word, or the class code where the class has no name.
func (e *MapEditor) objectWord(o BrushObject) string {
	switch {
	case o.Code == 0:
		return e.word("editor.brush.noobject")
	case o.Name == "":
		return fmt.Sprintf(e.word("editor.brush.object"), o.Code)
	}
	return o.Name
}

type brushState struct {
	mode    brushMode
	sel     [brushModeCount]int // selected row per mode
	radius  int                 // meaningful once radiusSet
	shape   BrushShape
	fill    BrushFill // the terrain brush's centre fill, kept for the session
	fillSet bool      // the fill was chosen; until then the brush fills at random
	stroke  []image.Point
	down    bool
	hover   image.Point
	hovered bool
	wheel   float64 // the Alt+wheel's turn not yet a whole step
	forest  forestState
	set     *heightSet // the Set height stroke in progress
	// sample is the height the press of a Sample height stroke took, while
	// sampled holds that stroke.
	sample  int
	sampled bool
	// heightMode is the Raise and Lower rows' mode; strength their change,
	// 0 until set (brushHeightStep).
	heightMode BrushHeightMode
	strength   int

	radiusSet bool
}

// heightSet is a Set height gesture: the cell the press took as the anchor,
// the pointer row it started at, the anchor's height then, the height the
// drag has reached and the pointer now.
type heightSet struct {
	anchor  image.Point
	pressY  int
	base    int
	target  int
	pointer image.Point
}

// heightSetRow and heightSampleRow are the Set height and Sample height
// rows' places in the height brush's list.
const (
	heightSetRow    = 2
	heightSampleRow = 3
)

// maxBrushHeight is the highest stored height.
const maxBrushHeight = 127

// settingHeight reports whether the height brush's Set height row is chosen.
func (e *MapEditor) settingHeight() bool {
	return e.brush.mode == brushHeight && e.brush.sel[brushHeight] == heightSetRow
}

// samplingHeight reports whether the height brush's Sample height row is
// chosen: the press takes the pressed cell's height and the drag sets every
// cell the brush covers to it.
func (e *MapEditor) samplingHeight() bool {
	return e.brush.mode == brushHeight && e.brush.sel[brushHeight] == heightSampleRow
}

// dragTarget is the height a pointer at p has dragged the gesture to: the
// anchor's height plus the rows the pointer rose above the press, in 0..127.
func (g *heightSet) dragTarget(p image.Point) int {
	return min(max(g.base+g.pressY-p.Y, 0), maxBrushHeight)
}

func (e *MapEditor) brushHost() InspectionBrush {
	if !e.editable() {
		return nil
	}
	b, _ := e.doc.Edits.(InspectionBrush)
	return b
}

// brushActive reports whether a brush mode owns the canvas and the list.
func (e *MapEditor) brushActive() bool {
	return e.brush.mode != brushOff && e.brushHost() != nil && e.free() && len(e.report.goals) == 0
}

func (e *MapEditor) brushRows() []string {
	switch e.brush.mode {
	case brushTerrain:
		var rows []string
		if h := e.brushHost(); h != nil {
			for _, t := range h.BrushTerrains() {
				rows = append(rows, e.terrainWord(t.Name))
			}
		}
		return rows
	case brushHeight:
		// The first row raises the cells, the second lowers them, the third
		// sets a height and the fourth paints a sampled one.
		return []string{e.word("editor.brush.raise"), e.word("editor.brush.lower"), e.word("editor.brush.setheight"), e.word("editor.brush.sampleheight")}
	case brushPassability:
		// The first row sets the impassable flag, the second clears it.
		return []string{e.word("editor.brush.block"), e.word("editor.brush.unblock")}
	case brushObjects:
		var rows []string
		if h := e.brushHost(); h != nil {
			for _, o := range h.BrushObjects() {
				word := e.objectWord(o)
				if e.forestOn() && o.Code != 0 {
					word = map[bool]string{false: "[ ] ", true: "[x] "}[e.brush.forest.ticks[o.Code]] + word
				}
				rows = append(rows, word)
			}
		}
		return rows
	}
	return nil
}

// brushPreviewW is the width of the picture beside the Objects list, wider
// than the Add window's previewW.
const brushPreviewW = 160

// brushAside is the column the Objects list keeps for the highlighted object's
// picture: none for another brush, or for a host that draws no pictures.
func (e *MapEditor) brushAside() int {
	if e.brush.mode != brushObjects {
		return 0
	}
	if _, ok := e.brushHost().(InspectionNewRecords); !ok {
		return 0
	}
	return brushPreviewW + popupPad
}

// drawBrushPreview draws the highlighted object's picture beside the Objects
// list, through the host's record-preview seam under the key "object".
func (e *MapEditor) drawBrushPreview(pic *image.RGBA, s popupSpec) {
	if e.brushAside() == 0 || s.ListAside == 0 {
		return
	}
	host, ok := e.brushHost().(InspectionNewRecords)
	if !ok {
		return
	}
	objects := e.brushHost().BrushObjects()
	sel := e.brush.sel[brushObjects]
	if sel < 0 || sel >= len(objects) || objects[sel].Code == 0 {
		return
	}
	e.drawAside(pic, s, host.NewRecordPreview("brush", "object", int64(objects[sel].Code)))
}

func (e *MapEditor) brushRadius() int {
	if !e.brush.radiusSet {
		return min(brushDefaultRadius, e.brushMaxRadius())
	}
	return e.brush.radius
}

// brushMaxRadius is the largest radius the host takes.
func (e *MapEditor) brushMaxRadius() int {
	if h := e.brushHost(); h != nil {
		return max(0, h.BrushMaxRadius())
	}
	return 0
}

// setBrushRadius is the one way the size changes: the bracket keys, the
// Alt+wheel and the list's minus and plus buttons all come here.
func (e *MapEditor) setBrushRadius(r int) {
	e.brush.radius, e.brush.radiusSet = min(e.brushMaxRadius(), max(0, r)), true
}

// brushWheel turns the Alt+wheel into size steps, one per notch, up for a
// turn away from the user.
func (e *MapEditor) brushWheel(wheel float64) {
	if math.IsNaN(wheel) || math.IsInf(wheel, 0) {
		return
	}
	e.brush.wheel += max(-1000, min(1000, wheel))
	steps := int(e.brush.wheel)
	e.brush.wheel -= float64(steps)
	if steps != 0 {
		e.setBrushRadius(e.brushRadius() + steps)
	}
}

// brushShapeWord is the word the list gives shape.
func (e *MapEditor) brushShapeWord(shape BrushShape) string {
	if shape == BrushSquare {
		return e.word("editor.brush.square")
	}
	return e.word("editor.brush.circle")
}

// brushLabelWidth is the label column of the brush list's rows.
const brushLabelWidth = 90

// brushPopupRows are the list's rows above the palette: the size with its
// minus and plus buttons, and the shape with a button for each; the Height
// tool adds the strength with its minus and plus buttons.
func (e *MapEditor) brushPopupRows() []popupRow {
	rows := []popupRow{
		{Kind: popupText, Label: e.word("editor.brush.size"), Value: fmt.Sprint(e.brushRadius()), Buttons: []string{"-", "+"}},
		{Kind: popupText, Label: e.word("editor.brush.shape"), Value: e.brushShapeWord(e.brush.shape),
			Buttons: []string{e.brushShapeWord(BrushCircle), e.brushShapeWord(BrushSquare)}},
	}
	if e.brush.mode == brushHeight {
		rows = append(rows, popupRow{Kind: popupText, Label: e.word("editor.brush.strength"), Value: fmt.Sprint(e.brushStrength()), Buttons: []string{"-", "+"}})
	}
	if e.brush.mode == brushTerrain && e.brushVariants() > 1 {
		rows = append(rows, popupRow{Kind: popupText, Label: e.word("editor.brush.fill"), Value: e.brushFillWord(),
			Buttons: []string{e.word("editor.brush.fill.pattern"), e.word("editor.brush.fill.random"), "-", "+"}})
	}
	if e.brush.mode == brushObjects {
		rows = append(rows, e.forestPopupRows()...)
	}
	return rows
}

// brushVariants is how many look-alike tiles the selected terrain has, 0 when
// the host offers no choice. A class with one has nothing to choose from: the
// fill row is not shown and the stroke keeps the pattern.
func (e *MapEditor) brushVariants() int {
	h, _ := e.brushHost().(InspectionBrushFill)
	terrains := []BrushTerrain(nil)
	if b := e.brushHost(); b != nil {
		terrains = b.BrushTerrains()
	}
	if sel := e.brush.sel[brushTerrain]; h != nil && sel < len(terrains) {
		return h.BrushVariants(terrains[sel].Class)
	}
	return 0
}

// brushFill is the fill a stroke takes: the kept choice with its variant held
// inside the selected terrain's variants.
func (e *MapEditor) brushFill() BrushFill {
	f := e.brush.fill
	if !e.brush.fillSet {
		f = BrushFill{Mode: BrushFillRandom}
	}
	if n := e.brushVariants(); n < 2 {
		return BrushFill{}
	} else if f.Mode == BrushFillVariant {
		f.Variant = max(0, min(f.Variant, n-1))
	}
	return f
}

// brushFillWord is the word the fill row gives the current fill.
func (e *MapEditor) brushFillWord() string {
	switch f := e.brushFill(); f.Mode {
	case BrushFillRandom:
		return e.word("editor.brush.fill.random")
	case BrushFillVariant:
		return fmt.Sprintf(e.word("editor.brush.fill.variant"), f.Variant+1, e.brushVariants())
	}
	return e.word("editor.brush.fill.pattern")
}

// setBrushFillButton is a press on button k of the fill row: Pattern, Random,
// and minus or plus through the variants (plus from Pattern or Random starts
// at the first, minus at the last).
func (e *MapEditor) setBrushFillButton(k int) {
	n := e.brushVariants()
	f := e.brushFill()
	switch k {
	case 0:
		f = BrushFill{}
	case 1:
		f = BrushFill{Mode: BrushFillRandom}
	case 2, 3:
		step := 2*k - 5
		switch {
		case f.Mode != BrushFillVariant && step > 0:
			f = BrushFill{Mode: BrushFillVariant}
		case f.Mode != BrushFillVariant:
			f = BrushFill{Mode: BrushFillVariant, Variant: n - 1}
		default:
			f.Variant = (f.Variant + step + n) % n
		}
	}
	e.brush.fill, e.brush.fillSet = f, true
}

// brushRowButton is a press on button k of the list's row: minus or plus on
// the size row, Circle or Square on the shape row, minus or plus on the
// Height tool's strength row.
func (e *MapEditor) brushRowButton(row, k int) {
	switch row {
	case 0:
		e.setBrushRadius(e.brushRadius() + 2*k - 1)
	case 1:
		e.brush.shape = BrushShape(k)
	case 2, 3, 4:
		if e.brush.mode == brushHeight && row == 2 {
			e.setBrushStrength(e.brushStrength() + 2*k - 1)
		} else if e.brush.mode == brushObjects {
			e.forestRowButton(row, k)
		} else if row == 2 {
			e.setBrushFillButton(k)
		}
	}
}

// brushKey handles the keys that choose a brush: T, H, B and Q switch their
// mode on or off, the bracket keys set the radius, Up and Down choose the row.
func (e *MapEditor) brushKey(key string) bool {
	switch key {
	case "terrain", "height", "objects", "passability":
		mode := brushKeyModes[key]
		if e.brushHost() == nil || e.catalog {
			return true
		}
		e.toggleBrush(mode)
		return true
	case "radiusup", "radiusdown":
		if e.brushActive() {
			r := e.brushRadius() + 1
			if key == "radiusdown" {
				r -= 2
			}
			e.setBrushRadius(r)
		}
		return true
	case "up", "down":
		if !e.brushActive() {
			return false
		}
		d := 1
		if key == "up" {
			d = -1
		}
		m := e.syncList("brush", e.listWindow())
		if editorListStep(m, d) {
			e.brushRowClick(m.Selection())
		}
		return true
	}
	return false
}

// brushRowClick selects the palette row at list row n.
func (e *MapEditor) brushRowClick(n int) {
	if n >= 0 && n < len(e.brushRows()) {
		e.brush.sel[e.brush.mode] = n
		if h := e.brushHost(); h != nil && e.forestOn() {
			if objects := h.BrushObjects(); n < len(objects) && objects[n].Code != 0 {
				e.forestTick(objects[n].Code)
			}
		}
	}
}

func (e *MapEditor) brushStatus() string {
	rows := e.brushRows()
	sel := e.brush.sel[e.brush.mode]
	if sel >= len(rows) {
		return ""
	}
	if e.forestOn() {
		return fmt.Sprintf(e.word("editor.brush.forest.status"), len(e.forestTicked()), e.forestShare(), e.brushRadius())
	}
	if e.raisingHeight() {
		return fmt.Sprintf(e.word("editor.brush.height.status"), e.brushModeWord(e.brush.mode), rows[sel], e.heightModeWord(e.brush.heightMode), e.brushRadius(), e.brushStrength())
	}
	return fmt.Sprintf(e.word("editor.brush.status"), e.brushModeWord(e.brush.mode), rows[sel], e.brushRadius())
}

// objectBrush is the bar's Objects entry (forest false) and Trees entry
// (forest true): the object brush on in that mode, or off when it is on in
// that mode already.
func (e *MapEditor) objectBrush(forest bool) {
	if e.brushHost() == nil || e.catalog {
		return
	}
	if e.brush.mode == brushObjects && e.brush.forest.on == forest {
		e.toggleBrush(brushObjects)
		return
	}
	e.chooseBrush(brushObjects)
	e.brush.forest.on = forest
}

// chooseBrush is a Tools menu row: it turns mode on, or every brush off
// for brushOff, and leaves a brush already on as it is.
func (e *MapEditor) chooseBrush(mode brushMode) {
	if e.brushHost() == nil || e.catalog {
		return
	}
	switch {
	case mode == brushOff && e.brush.mode != brushOff:
		e.toggleBrush(e.brush.mode)
	case mode != brushOff && e.brush.mode != mode:
		e.toggleBrush(mode)
	}
}

// brushCell is the grid cell under the frame position p, if it is on the map.
func (e *MapEditor) brushCell(p image.Point, rail int) (image.Point, bool) {
	if p.X < 0 || p.Y < 0 || p.X >= rail || p.Y >= e.doc.Viewer.cam.ViewH {
		return image.Point{}, false
	}
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok || x < 0 || y < 0 || x >= e.doc.Width || y >= e.doc.Height {
		return image.Point{}, false
	}
	return image.Pt(x, y), true
}

// brushPointer drives a stroke: a press on the canvas starts it, the cells the
// pointer crosses join it and the release commits it as one edit. In Forest
// mode a right press starts a thinning stroke the same way. It reports
// whether it took the input, so the canvas does not pan under the brush.
func (e *MapEditor) brushPointer(in Input, p image.Point, rail int) bool {
	rightWas := e.brush.forest.right
	e.brush.forest.right = in.SecondaryDown
	if !e.brushActive() {
		e.brush.endStroke()
		return false
	}
	cell, on := e.brushCell(p, rail)
	e.brush.hover, e.brush.hovered = cell, on
	if e.brush.down && e.brush.set != nil {
		e.down = in.PrimaryDown
		e.brush.set.pointer = p
		e.brush.set.target = e.brush.set.dragTarget(p)
		if !in.PrimaryDown {
			e.brushCommit()
		}
		return true
	}
	if e.brush.down {
		e.down = in.PrimaryDown
		held := in.PrimaryDown
		if e.brush.forest.thin {
			held = in.SecondaryDown
		}
		if on {
			e.brush.extend(cell)
		}
		if !held {
			e.brushCommit()
		}
		return true
	}
	thin := e.forestOn() && in.SecondaryDown && !rightWas && !in.PrimaryDown
	if (in.PrimaryDown && !e.down || thin) && on {
		e.down = in.PrimaryDown
		e.brush.down = true
		e.brush.forest.thin = thin
		e.brush.stroke = []image.Point{cell}
		if e.settingHeight() {
			base := e.cellHeight(cell)
			e.brush.set = &heightSet{anchor: cell, pressY: p.Y, base: base, target: base, pointer: p}
		}
		if e.samplingHeight() {
			e.brush.sample, e.brush.sampled = e.cellHeight(cell), true
		}
		e.doc.Viewer.dragging = false
		return true
	}
	return false
}

// extend adds cell to the stroke, with the cells on the straight line from the
// last one so a quick drag leaves no gap.
func (b *brushState) extend(cell image.Point) {
	last := b.stroke[len(b.stroke)-1]
	if cell == last {
		return
	}
	dx, dy := cell.X-last.X, cell.Y-last.Y
	n := max(absInt(dx), absInt(dy))
	for i := 1; i <= n; i++ {
		step := image.Pt(last.X+(dx*i+cmp.Compare(dx, 0)*n/2)/n, last.Y+(dy*i+cmp.Compare(dy, 0)*n/2)/n)
		if step != b.stroke[len(b.stroke)-1] {
			b.stroke = append(b.stroke, step)
		}
	}
}

func (e *MapEditor) brushCommit() {
	cells, thin, set := e.brush.stroke, e.brush.forest.thin, e.brush.set
	sample, sampled := e.brush.sample, e.brush.sampled
	e.brush.endStroke()
	host := e.brushHost()
	if host == nil || len(cells) == 0 {
		return
	}
	op := BrushStroke{Radius: e.brushRadius(), Shape: e.brush.shape, Cells: cells}
	sel := e.brush.sel[e.brush.mode]
	switch e.brush.mode {
	case brushHeight:
		op.Plane = BrushPlaneHeight
		switch {
		case sel < heightSetRow:
			op.Value, op.Mode = e.brushStrength(), e.brush.heightMode
			if sel == 1 {
				op.Value = -op.Value
			}
		case set != nil:
			op.Value, op.Set, op.Cells = set.target, true, []image.Point{set.anchor}
		case sampled:
			op.Value, op.Set, op.Flat = sample, true, true
		default:
			return
		}
	case brushObjects:
		objects := host.BrushObjects()
		if sel >= len(objects) {
			return
		}
		op.Plane, op.Value = BrushPlaneObjects, objects[sel].Code
		if e.forestOn() {
			forest, ok := e.forestStroke(thin)
			if !ok {
				return
			}
			op.Value, op.Forest = 0, forest
		}
	case brushPassability:
		op.Plane, op.Value = BrushPlanePassability, 1
		if sel == 1 {
			op.Value = 0
		}
	default:
		terrains := host.BrushTerrains()
		if sel >= len(terrains) {
			return
		}
		op.Plane, op.Value, op.Fill = BrushPlaneTerrain, terrains[sel].Class, e.brushFill()
	}
	if _, err := e.commandPaint(op); err != nil {
		e.message = err.Error()
	}
}

// terrainDiffers reports whether next shows other tiles, heights or objects
// than the current document.
func (e *MapEditor) terrainDiffers(next *InspectionDocument) bool {
	if next == nil || next.Viewer == nil || e.doc == nil {
		return false
	}
	a, b := e.doc.Viewer.grid, next.Viewer.grid
	return !equalSlice(a.Tiles, b.Tiles) || !equalSlice(a.Altitudes, b.Altitudes) || !equalSlice(a.Overlay, b.Overlay)
}

func equalSlice[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// adoptTerrainEdit takes a document whose terrain changed. The viewer is the
// new document's own, since a viewer cannot replace its grid; the camera and
// the display switches carry over from the old one.
func (e *MapEditor) adoptTerrainEdit(next *InspectionDocument, err error) error {
	if err != nil {
		e.message = err.Error()
		return err
	}
	if next == nil {
		return nil
	}
	old := e.doc.Viewer
	var kind string
	index := -1
	if e.selected >= 0 && e.selected < len(e.doc.Records) {
		r := e.doc.Records[e.selected]
		kind, index = r.Kind, r.Index
	}
	nv := next.Viewer
	nv.editorView = true
	nv.flat, nv.unshaded, nv.showGrid = old.flat, old.unshaded, old.showGrid
	e.doc = next
	e.cancelRestore, e.listRestore = nil, nil
	e.clearSelection()
	e.Layout(e.window.X, e.window.Y)
	nv.syncWorld()
	nv.cam.Zoom, nv.cam.X, nv.cam.Y = old.cam.Zoom, old.cam.X, old.cam.Y
	nv.cam.Clamp()
	e.closeReport()
	e.selected = -1
	for i, r := range next.Records {
		if r.Kind == kind && r.Index == index {
			e.selected = i
			break
		}
	}
	e.refreshGroup()
	e.message = ""
	return nil
}

// CellPoint is the position, in the window, of the centre of a map cell
// under the current camera. A driver that feeds the pointer uses it.
func (e *MapEditor) CellPoint(cell image.Point) image.Point {
	if e.doc == nil {
		return image.Point{}
	}
	x, y := e.cellCenter(cell)
	sx, sy := e.doc.Viewer.cam.WorldToScreen(x, y)
	ox, oy := e.doc.Viewer.place.Origin()
	return image.Pt(int(sx+ox), int(sy+oy))
}

// ShowCell scrolls the map to cell at full zoom.
func (e *MapEditor) ShowCell(cell image.Point) {
	if e.doc != nil {
		e.centerOn(cell)
	}
}

// brushMarkerLines outlines the cells the brush covers, as the host's own
// brush walk gives them: the stroke so far, or the cells under the pointer.
// Only the outline's outer edges are drawn.
func (e *MapEditor) brushMarkerLines() []editorLine {
	if !e.brushActive() {
		return nil
	}
	centres := e.brush.stroke
	if !e.brush.down {
		if !e.brush.hovered {
			return nil
		}
		centres = []image.Point{e.brush.hover}
	}
	marker := BrushStroke{Radius: e.brushRadius(), Shape: e.brush.shape, Cells: centres}
	if e.settingHeight() || e.samplingHeight() {
		marker.Plane, marker.Set, marker.Flat = BrushPlaneHeight, true, e.samplingHeight()
	}
	cells := e.brushHost().BrushCells(marker)
	in := make(map[image.Point]bool, len(cells))
	for _, c := range cells {
		in[c] = true
	}
	var out []editorLine
	v := e.doc.Viewer
	c := color.RGBA{editorGold.R, editorGold.G, editorGold.B, 255}
	for _, cell := range cells {
		verts := v.cellSpace().vertices(v.cam, cell.X, cell.Y)
		for k, edge := range [4][2]int{{0, 1}, {1, 3}, {3, 2}, {2, 0}} {
			if in[cell.Add(cellSides[k])] {
				continue
			}
			a, b := verts[edge[0]], verts[edge[1]]
			out = append(out, editorLine{a.DstX, a.DstY, b.DstX, b.DstY, c})
		}
	}
	return out
}

// cellSides is the neighbour across each edge of a cell's vertices: north,
// east, south and west.
var cellSides = [4]image.Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// endStroke drops the stroke in progress.
func (b *brushState) endStroke() {
	b.stroke, b.down, b.forest.thin, b.set, b.sampled = nil, false, false, nil, false
}
