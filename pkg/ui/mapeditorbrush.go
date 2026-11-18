package ui

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"math"

	"againrom/pkg/intmath"
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
// the class code a cell takes (0 is the row that clears objects), its name and
// the bar palette it lists in.
type BrushObject struct {
	Code    int
	Name    string
	Palette Palette
}

// BrushPlane names the grid layer a stroke writes.
type BrushPlane uint8

const (
	BrushPlaneTerrain BrushPlane = iota
	BrushPlaneHeight
	BrushPlaneObjects
	BrushPlanePassability
	BrushPlaneBridge
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
	brushBridge
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
var brushKeyModes = map[string]brushMode{"terrain": brushTerrain, "height": brushHeight, "objects": brushObjects, "passability": brushPassability, "bridge": brushBridge}

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

// brushRows are the brush window's list rows; none for a window without a
// list.
func (e *MapEditor) brushRows() []string {
	if !e.brushWindowOf().list {
		return nil
	}
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
			for _, o := range e.brushObjectRows() {
				rows = append(rows, e.objectWord(o))
			}
		}
		return rows
	}
	return nil
}

// brushPreviewW is the width of the picture beside a brush window's list.
const brushPreviewW = 160

// brushPicture is the picture beside the brush window's list: the chosen
// ground's swatch in the Terrain window, the chosen object's picture in the
// Objects window and the Trees window's One kind mode, through the host's
// record-preview seam under the key "object"; nil while the clear row is
// chosen. ok is false for a window that keeps no column for a picture, or a
// host that draws none.
func (e *MapEditor) brushPicture() (pic *image.RGBA, ok bool) {
	h := e.brushHost()
	if h == nil || !e.brushWindowOf().picture {
		return nil, false
	}
	sel := e.brush.sel[e.brush.mode]
	if e.brush.mode == brushTerrain {
		if terrains := h.BrushTerrains(); sel >= 0 && sel < len(terrains) {
			return terrains[sel].Swatch, true
		}
		return nil, true
	}
	if _, ok := h.(InspectionNewRecords); !ok {
		return nil, false
	}
	objects := e.brushObjectRows()
	if sel < 0 || sel >= len(objects) || objects[sel].Code == 0 {
		return nil, true
	}
	return e.placeTipPicture("brush", "object", int64(objects[sel].Code)), true
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
func (e *MapEditor) brushWheel(source wheelSource) {
	wheel := source.wheelNotches()
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

// brushWindow is one brush window as data for the one brush window builder,
// the brush branch of listSpecBase: the word of its title, the words its
// title's (?) says, its rows in order, and whether it shows its list and a
// picture beside the list.
type brushWindow struct {
	titleWord string
	helpWords []string
	rows      []brushRowKind
	list      bool
	picture   bool
}

// brushRowKind is a row of a brush window.
type brushRowKind uint8

const (
	brushRowTool     brushRowKind = iota // One kind or Forest
	brushRowSize                         // the radius, less and more
	brushRowShape                        // circle or square; the bridge's rectangle
	brushRowStrength                     // a Raise or Lower stroke's change
	brushRowFill                         // the ground's centre fill
	brushRowDensity                      // the share a forest stroke takes
)

// brushWindowOf is the shown brush's window. The Objects window holds no
// tool: Forest is the Trees window's. Forest mode ticks its kinds in the
// grid alone and shows no list; the Bridge window shows none either. The
// Terrain window's fill row shows for a ground with look-alike tiles.
func (e *MapEditor) brushWindowOf() brushWindow {
	sized := []brushRowKind{brushRowSize, brushRowShape}
	switch e.brush.mode {
	case brushTerrain:
		w := brushWindow{titleWord: "editor.brush.terrain", helpWords: []string{"editor.brush.terrain.help"}, rows: sized, list: true, picture: true}
		if e.brushVariants() > 1 {
			w.rows = append(w.rows, brushRowFill)
		}
		return w
	case brushHeight:
		return brushWindow{titleWord: "editor.brush.height", helpWords: []string{"editor.brush.height.help"}, rows: append(sized, brushRowStrength), list: true}
	case brushPassability:
		return brushWindow{titleWord: "editor.brush.passability", rows: sized, list: true}
	case brushBridge:
		return brushWindow{titleWord: "editor.brush.bridge", helpWords: []string{"editor.brush.bridge.help"}, rows: sized}
	case brushObjects:
		if !e.brush.forest.trees && !e.brush.forest.on {
			return brushWindow{titleWord: "editor.bar.objects", helpWords: []string{"editor.bar.objects.help"}, rows: sized, list: true, picture: true}
		}
		tool := append([]brushRowKind{brushRowTool}, sized...)
		if e.brush.forest.on {
			return brushWindow{titleWord: "editor.bar.trees", helpWords: []string{"editor.bar.trees.help"}, rows: append(tool, brushRowDensity)}
		}
		return brushWindow{titleWord: "editor.bar.trees", helpWords: []string{"editor.bar.trees.help"}, rows: tool, list: true, picture: true}
	}
	return brushWindow{}
}

// brushPopupRows are the brush window's rows above its grid and list.
func (e *MapEditor) brushPopupRows() []popupRow {
	var rows []popupRow
	for _, k := range e.brushWindowOf().rows {
		rows = append(rows, e.brushRow(k))
	}
	return rows
}

// brushRow is row k of the brush window: its word, value, buttons and (?).
func (e *MapEditor) brushRow(k brushRowKind) popupRow {
	steps := []string{e.word("editor.brush.less"), e.word("editor.brush.more")}
	switch k {
	case brushRowTool:
		tool := e.word("editor.brush.tool.one")
		if e.brush.forest.on {
			tool = e.word("editor.brush.tool.forest")
		}
		return popupRow{Kind: popupText, Label: e.word("editor.brush.tool"), Value: tool,
			Buttons: []string{e.word("editor.brush.tool.one"), e.word("editor.brush.tool.forest")}}
	case brushRowSize:
		return popupRow{Kind: popupText, Label: e.word("editor.brush.size"), Value: fmt.Sprint(e.brushRadius()), Buttons: steps,
			Help: e.word("editor.brush.size.help")}
	case brushRowShape:
		if e.brush.mode == brushBridge {
			return popupRow{Kind: popupText, Label: e.word("editor.brush.shape"), Value: e.word("editor.brush.rectangle")}
		}
		return popupRow{Kind: popupText, Label: e.word("editor.brush.shape"), Value: e.brushShapeWord(e.brush.shape),
			Buttons: []string{e.brushShapeWord(BrushCircle), e.brushShapeWord(BrushSquare)}}
	case brushRowStrength:
		return popupRow{Kind: popupText, Label: e.word("editor.brush.strength"), Value: fmt.Sprint(e.brushStrength()), Buttons: steps,
			Help: e.word("editor.brush.strength.help")}
	case brushRowFill:
		return popupRow{Kind: popupText, Label: e.word("editor.brush.fill"), Value: e.brushFillWord(),
			Buttons: append([]string{e.word("editor.brush.fill.pattern"), e.word("editor.brush.fill.random")}, steps...),
			Help:    e.word("editor.brush.fill.help")}
	}
	buttons := make([]string, 0, len(forestShares)+1)
	for i := range forestShares {
		buttons = append(buttons, e.forestPresetWord(i))
	}
	return popupRow{Kind: popupText, Label: e.word("editor.brush.density"), Value: fmt.Sprintf(e.word("editor.brush.density.value"), e.forestShare()),
		Buttons: append(buttons, e.word("editor.brush.density.typed")), Help: e.word("editor.brush.density.help")}
}

// brushFit gives the brush window its natural size through listFit: the label column as wide
// as its widest label with its (?), the mode row's head as wide as that
// column, and the window as wide as its widest row, its list's widest row
// beside the picture column, its grid's columns, its empty-palette notes, its
// title and its button row, so every label, value and name shows whole.
func (e *MapEditor) brushFit(s *popupSpec) {
	e.listFit(s, listFitIn{
		Counts: rowHasControl,
		Value: func(r popupRow, m listFitMeasure) int {
			w := max(popupMinField, m.Advance(r.Value)+2*(editTextX+2))
			for _, bw := range rowButtonWidths(m.Window, r, m.Body, m.Pitch) {
				w += bw + 4
			}
			return w
		},
		Heads:     true,
		GridSlack: 2,
		Rest: func(_ int, m listFitMeasure) int {
			need := 0
			if s.List != nil {
				for _, r := range s.List.Rows() {
					text := m.Advance(r.Text) + 2*listTextX
					if r.Icon != nil || r.IconCell {
						text += m.Pitch + listTextX
					}
					need = max(need, text+widgetSpriteSize+2+s.ListAside)
				}
			}
			for _, n := range e.emptyPaletteNote() {
				need = max(need, m.Advance(n)+8)
			}
			if e.font != nil {
				need = max(need, e.font.Advance(s.Title)+2*m.Pitch)
			}
			return need
		},
	})
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

// brushRowButton is a press on button k of the brush window's row: One kind
// or Forest on the tool row, less or more on the size and strength rows,
// Circle or Square on the shape row, the fill row's buttons, and the density
// row's presets and its button that asks for a typed per cent.
func (e *MapEditor) brushRowButton(row, k int) {
	rows := e.brushWindowOf().rows
	if row < 0 || row >= len(rows) {
		return
	}
	switch rows[row] {
	case brushRowTool:
		e.setForest(k == 1)
	case brushRowSize:
		e.setBrushRadius(e.brushRadius() + 2*k - 1)
	case brushRowShape:
		if e.brush.mode != brushBridge {
			e.brush.shape = BrushShape(k)
		}
	case brushRowStrength:
		e.setBrushStrength(e.brushStrength() + 2*k - 1)
	case brushRowFill:
		e.setBrushFillButton(k)
	case brushRowDensity:
		e.forestDensityButton(k)
	}
}

// brushKey handles the keys that choose a brush: T, H, B and Q switch their
// mode on or off, the bracket keys set the radius, Up and Down choose the row.
func (e *MapEditor) brushKey(key string) bool {
	switch key {
	case "terrain", "height", "objects", "passability", "bridge":
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
	}
}

// objectBrush is the bar's Objects entry (forest false) and Trees entry
// (forest true): the object brush on in that palette's first mode, or off
// when it is on in that mode already. The Trees window keeps its One kind
// mode, whose rows are the tree classes; the Objects window's rows are the
// other classes.
func (e *MapEditor) objectBrush(forest bool) {
	if e.brushHost() == nil || e.catalog {
		return
	}
	if e.brush.mode == brushObjects && e.brush.forest.on == forest && e.brush.forest.trees == forest {
		e.closeBarWindow()
		return
	}
	e.closeBarWindow()
	e.chooseBrush(brushObjects)
	e.barOwner = "objects"
	if forest {
		e.barOwner = "trees"
	}
	delete(e.barSets, e.barOwner)
	if e.brush.forest.trees != forest {
		e.brush.sel[brushObjects] = 0
	}
	e.brush.forest.trees = forest
	e.setForest(forest)
}

// setForest puts the object brush in its Forest mode or its one-kind mode.
// The two modes list different rows, so a row chosen in one names another
// class in the other: a switched list starts on its clear row.
func (e *MapEditor) setForest(on bool) {
	if e.brush.forest.on != on {
		e.brush.sel[brushObjects] = 0
	}
	e.brush.forest.on = on
}

// chooseBrush is a toolbar tool: it turns mode on, or every brush off
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
	thin := e.brushErases() && in.SecondaryDown && !rightWas && !in.PrimaryDown
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
	n := max(intmath.Abs(dx), intmath.Abs(dy))
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
	case brushBridge:
		op.Plane, op.Value = BrushPlaneBridge, 1
		if thin {
			op.Value = 0
		}
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
		objects := e.brushObjectRows()
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
	var key uint64
	if e.selected >= 0 && e.selected < len(e.doc.Records) {
		key = e.doc.Records[e.selected].Key
	}
	nv := next.Viewer
	nv.editorView = true
	nv.wheelZoomPercent = 0
	nv.flat, nv.unshaded, nv.showGrid = old.flat, old.unshaded, old.showGrid
	e.doc = next
	e.cancelRestore, e.listRestore = nil, nil
	e.clearSelection()
	e.Layout(e.window.X, e.window.Y)
	nv.syncWorld()
	nv.cam.Zoom, nv.cam.X, nv.cam.Y = old.cam.Zoom, old.cam.X, old.cam.Y
	nv.cam.Clamp()
	e.refreshBarReport()
	e.selected = e.linkedRecord(RecordRef{}, key)
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
	ox, oy := e.mapOrigin()
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
	if e.brush.mode == brushBridge {
		marker.Plane = BrushPlaneBridge
	}
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
