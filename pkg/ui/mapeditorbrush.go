package ui

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
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

// BrushStroke is one pointer stroke as data: the plane, the value at each cell
// (a terrain class, a height change, an object class code, or 1 to set and 0
// to clear the tile word's impassable flag), the brush radius
// in cells and the cells the pointer visited.
type BrushStroke struct {
	Plane  BrushPlane
	Value  int
	Radius int
	Cells  []image.Point
}

// InspectionBrush is the optional seam of a host whose document paints terrain,
// heights and objects. A stroke is one edit: one undo step, one document.
type InspectionBrush interface {
	BrushTerrains() []BrushTerrain
	BrushObjects() []BrushObject
	Paint(BrushStroke) (*InspectionDocument, error)
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

// The brush settings the window offers: the radius a stroke starts with and
// its bounds, and the height change each visited cell adds or takes.
const (
	brushDefaultRadius = 2
	brushMaxRadius     = 8
	brushHeightStep    = 8
)

// brushKeyModes is the brush mode each brush key switches.
var brushKeyModes = map[string]brushMode{"terrain": brushTerrain, "height": brushHeight, "objects": brushObjects, "passability": brushPassability}

// brushModeWord is the word the Brushes menu and the list title give mode.
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
	return e.word("editor.brush.off")
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
	stroke  []image.Point
	down    bool
	hover   image.Point
	hovered bool

	radiusSet bool
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
		// The first row raises the cells, the second lowers them.
		return []string{e.word("editor.brush.raise"), e.word("editor.brush.lower")}
	case brushPassability:
		// The first row sets the impassable flag, the second clears it.
		return []string{e.word("editor.brush.block"), e.word("editor.brush.unblock")}
	case brushObjects:
		var rows []string
		if h := e.brushHost(); h != nil {
			for _, o := range h.BrushObjects() {
				rows = append(rows, e.objectWord(o))
			}
		}
		return rows
	}
	return nil
}

func (e *MapEditor) brushRadius() int {
	if !e.brush.radiusSet {
		return brushDefaultRadius
	}
	return e.brush.radius
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
			e.brush.radius, e.brush.radiusSet = min(brushMaxRadius, max(0, r)), true
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

func (e *MapEditor) brushStatus() string {
	rows := e.brushRows()
	sel := e.brush.sel[e.brush.mode]
	if sel >= len(rows) {
		return ""
	}
	return fmt.Sprintf(e.word("editor.brush.status"), e.brushModeWord(e.brush.mode), rows[sel], e.brushRadius())
}

// chooseBrush is a Brushes menu row: it turns mode on, or every brush off
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
// pointer crosses join it and the release commits it as one edit. It reports
// whether it took the input, so the canvas does not pan under the brush.
func (e *MapEditor) brushPointer(in Input, p image.Point, rail int) bool {
	if !e.brushActive() {
		e.brush.endStroke()
		return false
	}
	cell, on := e.brushCell(p, rail)
	e.brush.hover, e.brush.hovered = cell, on
	if e.brush.down {
		e.down = in.PrimaryDown
		if on {
			e.brush.extend(cell)
		}
		if !in.PrimaryDown {
			e.brushCommit()
		}
		return true
	}
	if in.PrimaryDown && !e.down && on {
		e.down = true
		e.brush.down = true
		e.brush.stroke = []image.Point{cell}
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
	cells := e.brush.stroke
	e.brush.endStroke()
	host := e.brushHost()
	if host == nil || len(cells) == 0 {
		return
	}
	op := BrushStroke{Radius: e.brushRadius(), Cells: cells}
	sel := e.brush.sel[e.brush.mode]
	switch e.brush.mode {
	case brushHeight:
		op.Plane, op.Value = BrushPlaneHeight, brushHeightStep
		if sel == 1 {
			op.Value = -brushHeightStep
		}
	case brushObjects:
		objects := host.BrushObjects()
		if sel >= len(objects) {
			return
		}
		op.Plane, op.Value = BrushPlaneObjects, objects[sel].Code
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
		op.Value = terrains[sel].Class
	}
	next, err := host.Paint(op)
	_ = e.adoptTerrainEdit(next, err)
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

// brushMarkerLines outlines the cells the brush covers: the stroke so far, or
// the cells under the pointer.
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
	r := e.brushRadius()
	seen := map[image.Point]bool{}
	var out []editorLine
	v := e.doc.Viewer
	c := color.RGBA{editorGold.R, editorGold.G, editorGold.B, 255}
	for _, ctr := range centres {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				cell := ctr.Add(image.Pt(dx, dy))
				if dx*dx+dy*dy > r*r || seen[cell] || cell.X < 0 || cell.Y < 0 || cell.X >= e.doc.Width || cell.Y >= e.doc.Height {
					continue
				}
				seen[cell] = true
				verts := v.cellSpace().vertices(v.cam, cell.X, cell.Y)
				for _, edge := range [][2]int{{0, 1}, {1, 3}, {3, 2}, {2, 0}} {
					a, b := verts[edge[0]], verts[edge[1]]
					out = append(out, editorLine{a.DstX, a.DstY, b.DstX, b.DstY, c})
				}
			}
		}
	}
	return out
}

// endStroke drops the stroke in progress.
func (b *brushState) endStroke() { b.stroke, b.down = nil, false }
