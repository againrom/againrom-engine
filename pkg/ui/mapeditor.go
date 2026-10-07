package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"time"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const editorRailWidth = 320
const editorListTop, editorRowHeight = 184, 20

var editorInk = color.RGBA{0x20, 0x1b, 0x16, 255}
var editorPanel = color.RGBA{0x30, 0x29, 0x1f, 255}
var editorPaper = color.RGBA{0xd6, 0xc4, 0x9b, 255}
var editorLight = color.RGBA{0xf3, 0xe5, 0xbe, 255}
var editorGold = color.RGBA{0xb8, 0x95, 0x4f, 255}
var editorWarning = color.RGBA{0xb8, 0x5e, 0x46, 255}
var editorCondition = color.RGBA{0x78, 0xb6, 0xc2, 255}

type InspectionRecord struct {
	Kind           string
	Section, Index int // original section and file-order index (cell index for type 3)
	Label          string
	Cell, Size     image.Point
	Spatial        bool
	Details        []string
	Warning        string
	Preview        *terrain.StaticFrame
	Lines          []InspectionLine
	References     []InspectionReference
}

type InspectionDocument struct {
	Viewer        *Viewer
	Source, Name  string
	Width, Height int
	Records       []InspectionRecord
	SourceBytes   []byte // exact current model bytes, not a re-encoded map
	ScriptNotice  string
	Edits         InspectionEdits // optional; host owns persistence and safety
}

type InspectionEdits interface {
	Dirty() bool
	CanUndo() bool
	CanRedo() bool
	MoveUnit(index int, cell image.Point) (*InspectionDocument, error)
	Undo() (*InspectionDocument, error)
	Redo() (*InspectionDocument, error)
	SaveAs(path string) (string, error)
}

type EditorMap struct{ Key, Label string }
type InspectionOpener func(string) (*InspectionDocument, error)

// MapEditor is an authored-map canvas. It has no game App, command queue,
// mission or simulation ticking seam. Host callbacks own edits and file writes.
type MapEditor struct {
	doc                            *InspectionDocument
	font                           *text.Font
	maps                           []EditorMap
	open                           InspectionOpener
	hostText                       func(string) string
	selected, offset, detailOffset int
	listWheel, detailWheel         float64 // fractional rows belong to their viewport
	filter                         int
	catalog, pathMode              bool
	pathSave                       bool
	path, message                  string
	down, moved                    bool
	press                          image.Point
	window                         image.Point
	panelImage                     *ebiten.Image
	focusedReference               int // 1-based on the selected record; 0 means none
	unitDrag                       *editorUnitDrag
	cancelledDrag                  bool // consume the remainder of a cancelled press
	textSmoothingEnabled           bool
	textOverlay                    textOverlay
	textLayers                     []textLayer
	textCaptured, textKept         int
}

func NewMapEditor(font *text.Font, maps []EditorMap, open InspectionOpener) *MapEditor {
	return &MapEditor{font: font, maps: maps, open: open, selected: -1, catalog: true, window: image.Pt(1280, 800), textSmoothingEnabled: true}
}

func (e *MapEditor) SetTextSmoothing(enabled bool) { e.textSmoothingEnabled = enabled }

func (e *MapEditor) TextSettle() (captured, kept, fallbacks int) {
	return e.textCaptured, e.textKept, 0
}

func (e *MapEditor) TextFates() []TextFate {
	return (&App{textLayers: e.textLayers}).TextFates()
}

func (e *MapEditor) Document() *InspectionDocument { return e.doc }
func (e *MapEditor) Message() string               { return e.message }
func (e *MapEditor) Selected() int                 { return e.selected }

// SetHostTextEncoder supplies the install's UTF-8-to-font encoding outside the
// UI tier. Record and item strings are already font-encoded by the loader.
func (e *MapEditor) SetHostTextEncoder(encode func(string) string) { e.hostText = encode }

func (e *MapEditor) Open(source string) error {
	if e.Dirty() {
		e.message = "Unsaved changes. Save As or undo changes before opening another map."
		return fmt.Errorf("%s", e.message)
	}
	if e.open == nil {
		e.message = "no map opener"
		return fmt.Errorf("%s", e.message)
	}
	next, err := e.open(source)
	if err == nil && (next == nil || next.Viewer == nil) {
		err = fmt.Errorf("map opener returned no view")
	}
	if err != nil {
		e.message = err.Error()
		return err
	}
	next.Viewer.editorView = true
	e.doc = next
	e.Layout(e.window.X, e.window.Y)
	e.selected, e.offset, e.detailOffset, e.filter = -1, 0, 0, 0
	e.focusedReference = 0
	e.listWheel, e.detailWheel = 0, 0
	e.catalog, e.pathMode = false, false
	e.message = ""
	e.cancelUnitDrag()
	e.Fit()
	return nil
}

func (e *MapEditor) Fit() {
	if e.doc == nil {
		return
	}
	c := e.doc.Viewer.cam
	z := e.fitZoom()
	c.SetMinimumZoom(math.Min(camera.ZoomMin, z))
	c.SetZoom(z)
	c.X, c.Y = 0, 0
	c.Clamp()
}

func (e *MapEditor) fitZoom() float64 {
	c := e.doc.Viewer.cam
	return math.Min(float64(c.ViewW)/c.WorldW(), float64(c.ViewH)/c.WorldH())
}

func (e *MapEditor) Select(index int, jump bool) bool {
	if e.doc == nil || index < 0 || index >= len(e.doc.Records) {
		return false
	}
	e.selected, e.detailOffset = index, 0
	e.focusedReference = 0
	e.detailWheel = 0
	rows := e.rows()
	row := -1
	for n, id := range rows {
		if id == index {
			row = n
			break
		}
	}
	if row >= 0 && (row < e.offset || row >= e.offset+e.visibleRows()) {
		e.offset = max(0, row-e.visibleRows()/2)
	}
	r := e.doc.Records[index]
	if jump && r.Spatial {
		v := e.doc.Viewer
		x, y := e.cellCenter(r.Cell)
		v.cam.SetZoom(1)
		v.cam.X = x - float64(v.cam.ViewW)/2
		v.cam.Y = y - float64(v.cam.ViewH)/2
		v.cam.Clamp()
	}
	return true
}

func (e *MapEditor) cellCenter(cell image.Point) (float64, float64) {
	v := e.doc.Viewer
	cell.X = max(0, min(cell.X, e.doc.Width-1))
	cell.Y = max(0, min(cell.Y, e.doc.Height-1))
	x, y := cell.X*32+16, cell.Y*32+16
	if v.Mode() == ModeDisplaced {
		left, top := v.proj.WorldCorner(cell.X, cell.Y)
		right, bottom := v.proj.WorldCorner(cell.X+1, cell.Y+1)
		x, y = (left+right)/2, (top+bottom)/2
	}
	return float64(x), float64(y)
}

var editorFilters = []string{"All", "Units", "Build", "Sacks", "Scenery", "Other", "Issues", "Triggers"}

func (e *MapEditor) listBottom() int { return editorListTop + e.visibleRows()*editorRowHeight }
func (e *MapEditor) visibleRows() int {
	return max(3, min(12, (e.window.Y/2+30-editorListTop)/editorRowHeight))
}

func (e *MapEditor) rows() []int {
	if e.doc == nil {
		return nil
	}
	var out []int
	for i, r := range e.doc.Records {
		show := e.filter == 0
		switch e.filter {
		case 1:
			show = r.Kind == "Unit"
		case 2:
			show = r.Kind == "Structure"
		case 3:
			show = r.Kind == "Sack" || r.Kind == "Stock"
		case 4:
			show = r.Kind == "Scenery"
		case 5:
			show = !r.Spatial
		case 6:
			show = r.Warning != ""
		case 7:
			show = r.Kind == "Trigger" || (r.Kind == "Section" && r.Section == 7)
		}
		if show {
			out = append(out, i)
		}
	}
	return out
}

// Key and Pointer are the same input doors used by Update and headless proof.
func (e *MapEditor) Key(key string) error {
	if e.pathMode && key != "escape" && key != "enter" && key != "backspace" && key != "close" {
		return nil
	}
	switch key {
	case "close":
		if e.Dirty() {
			e.message = "Unsaved changes. Save As or undo changes before closing."
			return nil
		}
		return ebiten.Termination
	case "escape":
		if e.unitDrag != nil {
			e.cancelUnitDrag()
			return nil
		}
		if e.pathMode {
			e.pathMode = false
			e.message = ""
			return nil
		}
		if e.message != "" {
			e.message = ""
			return nil
		}
		if e.catalog && e.doc != nil {
			e.catalog = false
			e.offset = 0
			e.listWheel = 0
			return nil
		}
		if e.Dirty() {
			e.message = "Unsaved changes. Save As or undo changes before closing."
			return nil
		}
		return ebiten.Termination
	case "open":
		e.cancelUnitDrag()
		e.pathMode = true
		e.pathSave = false
		e.path = ""
		e.message = ""
	case "catalog":
		e.cancelUnitDrag()
		e.catalog = !e.catalog
		e.offset = 0
		e.listWheel = 0
		e.message = ""
	case "enter":
		if e.pathMode {
			if e.pathSave {
				return e.saveAs(strings.TrimSpace(e.path))
			}
			return e.Open(strings.TrimSpace(e.path))
		}
	case "saveas":
		if e.editable() {
			e.cancelUnitDrag()
			e.pathMode, e.pathSave = true, true
			e.path, e.message = "", ""
		}
	case "undo", "redo":
		e.cancelUnitDrag()
		if e.editable() {
			if key == "undo" && !e.doc.Edits.CanUndo() || key == "redo" && !e.doc.Edits.CanRedo() {
				return nil
			}
			var next *InspectionDocument
			var err error
			if key == "undo" {
				next, err = e.doc.Edits.Undo()
			} else {
				next, err = e.doc.Edits.Redo()
			}
			return e.adoptEdit(next, err)
		}
	case "backspace":
		if e.pathMode {
			r := []rune(e.path)
			if len(r) > 0 {
				e.path = string(r[:len(r)-1])
			}
		}
	case "fit":
		e.Fit()
	case "triggers":
		e.filter, e.offset = 7, 0
		e.listWheel = 0
		if e.selected >= 0 {
			e.Select(e.selected, false)
		}
	case "pageup":
		e.listWheel = 0
		e.offset = max(0, e.offset-e.visibleRows())
	case "pagedown":
		e.listWheel = 0
		e.offset = min(max(0, e.rowCount()-e.visibleRows()), e.offset+e.visibleRows())
	case "home":
		e.listWheel = 0
		e.offset = 0
	case "end":
		e.listWheel = 0
		e.offset = max(0, e.rowCount()-e.visibleRows())
	}
	return nil
}

func (e *MapEditor) rowCount() int {
	if e.catalog {
		return len(e.maps)
	}
	return len(e.rows())
}

func (e *MapEditor) Pointer(in Input, now time.Time) {
	x, y := in.CursorX, in.CursorY
	rail := e.window.X - editorRailWidth
	if e.doc != nil {
		x, y = e.doc.Viewer.windowToFrame(x, y)
		rail = e.doc.Viewer.cam.ViewW
	}
	p := image.Pt(x, y)
	if e.dragUnit(in, p, rail) {
		return
	}
	if in.PrimaryDown && !e.down {
		e.press = p
		e.moved = false
	}
	if e.down && (math.Abs(float64(x-e.press.X)) > 4 || math.Abs(float64(y-e.press.Y)) > 4) {
		e.moved = true
	}
	released := e.down && !in.PrimaryDown && !e.moved
	e.down = in.PrimaryDown
	if e.pathMode {
		return
	}
	if x >= rail {
		if e.doc != nil {
			e.doc.Viewer.dragging = false
		}
		if in.WheelY != 0 {
			if y >= e.listBottom() && !e.catalog {
				e.detailOffset = max(0, e.detailOffset+editorWheelRows(&e.detailWheel, in.WheelY))
			} else {
				e.offset = max(0, min(max(0, e.rowCount()-e.visibleRows()), e.offset+editorWheelRows(&e.listWheel, in.WheelY)))
			}
		}
		if released && e.press.X >= rail {
			e.railClick(x-rail, y)
		}
		return
	}
	if e.doc == nil || e.catalog || e.pathMode {
		return
	}
	e.doc.Viewer.step(in, now)
	if released && e.press.X < rail {
		e.selectAt(p)
	}
}

// A high-resolution wheel may report less than one notch per update. Preserve
// the remainder instead of discarding it on every float-to-int conversion.
func editorWheelRows(remainder *float64, wheel float64) int {
	if math.IsNaN(wheel) || math.IsInf(wheel, 0) {
		return 0
	}
	*remainder -= math.Max(-100000, math.Min(100000, wheel)) * 3
	rows := int(*remainder)
	*remainder -= float64(rows)
	return rows
}

func (e *MapEditor) railClick(x, y int) {
	if e.pathMode {
		return
	}
	if y >= 42 && y < 64 {
		_ = e.Key([]string{"undo", "redo", "saveas"}[min(2, max(0, x/108))])
		return
	}
	if !e.catalog && !e.pathMode && e.message == "" {
		for _, line := range e.detailLayout(e.window.Y) {
			if line.Reference > 0 && image.Pt(x, y).In(line.Rect) {
				e.FocusReference(line.Reference)
				return
			}
		}
	}
	if y >= 8 && y < 36 {
		switch {
		case x < 108:
			_ = e.Key("catalog")
		case x < 216:
			_ = e.Key("open")
		default:
			e.Fit()
		}
		return
	}
	if !e.catalog && y >= 128 && y < 176 {
		idx := (y-128)/24*4 + x/80
		if idx < len(editorFilters) {
			e.filter = idx
			e.offset = 0
			e.listWheel = 0
			if e.selected >= 0 {
				e.Select(e.selected, false)
			}
		}
		return
	}
	if y < editorListTop || y >= e.listBottom() {
		return
	}
	row := e.offset + (y-editorListTop)/editorRowHeight
	if e.catalog {
		if row < len(e.maps) {
			_ = e.Open(e.maps[row].Key)
		}
	} else {
		rows := e.rows()
		if row < len(rows) {
			e.Select(rows[row], true)
		}
	}
}

func (e *MapEditor) Update() error {
	if ebiten.IsWindowBeingClosed() {
		if err := e.Key("close"); err != nil {
			return err
		}
	}
	if e.pathMode {
		e.TextInput(string(ebiten.AppendInputChars(nil)))
	}
	for _, k := range []struct {
		key  ebiten.Key
		name string
	}{{ebiten.KeyEscape, "escape"}, {ebiten.KeyEnter, "enter"}, {ebiten.KeyNumpadEnter, "enter"}, {ebiten.KeyBackspace, "backspace"}, {ebiten.KeyPageUp, "pageup"}, {ebiten.KeyPageDown, "pagedown"}, {ebiten.KeyHome, "home"}, {ebiten.KeyEnd, "end"}} {
		if inpututil.IsKeyJustPressed(k.key) {
			if err := e.Key(k.name); err == ebiten.Termination {
				return err
			}
		}
	}
	if !e.pathMode {
		if ebiten.IsKeyPressed(ebiten.KeyControl) {
			if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
				key := "undo"
				if ebiten.IsKeyPressed(ebiten.KeyShift) {
					key = "redo"
				}
				_ = e.Key(key)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyY) {
				_ = e.Key("redo")
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyS) {
				_ = e.Key("saveas")
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyO) {
			_ = e.Key("open")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF) {
			e.Fit()
		}
	}
	e.Pointer(readInput(), time.Now())
	return nil
}

func (e *MapEditor) Layout(w, h int) (int, int) {
	e.window = image.Pt(w, h)
	if e.doc != nil {
		v := e.doc.Viewer
		if w > 0 && h > 0 {
			if v.frameW != w || v.frameH != h {
				if v.canvas != nil {
					disposeFrameCanvas(v.canvas)
					v.canvas = nil
				}
			}
			v.frameW, v.frameH = w, h
			v.place = frame.Fit(w, h, w, h)
			v.syncMapViewport()
			v.cam.SetMinimumZoom(math.Min(camera.ZoomMin, e.fitZoom()))
		}
	}
	return w, h
}

func (e *MapEditor) Draw(screen *ebiten.Image) {
	screen.Fill(editorInk)
	e.textCaptured, e.textKept = 0, 0
	e.textLayers = e.textLayers[:0]
	if e.textSmoothingEnabled {
		text.ResetCapture()
	}
	if e.doc == nil {
		calls := e.drawPanel(screen, e.window.X-editorRailWidth, e.window.Y)
		if len(calls) > 0 {
			e.textOverlay.draw(screen, calls, 1, 0, 0)
		}
		return
	}
	v := e.doc.Viewer
	v.syncMapViewport()
	if v.canvas == nil {
		v.canvas = ebiten.NewImage(v.frameW, v.frameH)
	}
	v.canvas.Clear()
	v.drawFrame(v.canvas)
	e.drawMarkers(v.canvas)
	calls := e.drawPanel(v.canvas, v.cam.ViewW, v.frameH)
	ox, oy := v.place.Origin()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(v.place.Scale(), v.place.Scale())
	op.GeoM.Translate(ox, oy)
	screen.DrawImage(v.canvas, op)
	if len(calls) > 0 {
		e.textOverlay.draw(screen, calls, v.place.Scale(), ox, oy)
	}
}

func (e *MapEditor) drawPanel(dst *ebiten.Image, x, h int) []text.DrawCall {
	pic, blit, calls, kept := e.panelPicture(h)
	for i := range calls {
		calls[i].X += x
		calls[i].Clip = calls[i].Clip.Add(image.Pt(x, 0)).Intersect(dst.Bounds())
	}
	for i := range kept {
		kept[i].X += x
		kept[i].Clip = kept[i].Clip.Add(image.Pt(x, 0)).Intersect(dst.Bounds())
	}
	log := &pixelLog{}
	log.reset(dst.Bounds())
	log.over(pic, image.Pt(x, 0))
	log.end()
	e.textLayers = append(e.textLayers, textLayer{calls, dst.Bounds(), log})
	e.textCaptured, e.textKept = len(calls), len(kept)
	if e.panelImage == nil || e.panelImage.Bounds().Dy() != h {
		if e.panelImage != nil {
			e.panelImage.Dispose()
		}
		e.panelImage = ebiten.NewImage(editorRailWidth, h)
	}
	e.panelImage.WritePixels(blit.Pix)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), 0)
	dst.DrawImage(e.panelImage, op)
	return kept
}

func (e *MapEditor) panelPicture(h int) (pic, blit *image.RGBA, calls, kept []text.DrawCall) {
	if !e.textSmoothingEnabled {
		pic = e.PanelImage(h)
		return pic, pic, nil, nil
	}
	calls = text.Record(func() { pic = e.PanelImage(h) })
	blit = image.NewRGBA(pic.Bounds())
	copy(blit.Pix, pic.Pix)
	kept = textsmooth.Settle(blit, calls)
	return pic, blit, calls, text.MarkErased(kept)
}

// PanelImage composes the native rail, including art and opaque covers.
func (e *MapEditor) PanelImage(h int) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, editorRailWidth, h))
	draw.Draw(pic, pic.Bounds(), image.NewUniform(editorPanel), image.Point{}, draw.Src)
	line := func(s string, x, y, w int, c color.RGBA) {
		if e.font == nil {
			return
		}
		for _, s := range editorWrap(e.font, s, w) {
			e.font.Draw(pic, s, x, y, c)
			y += max(16, e.font.Height()+2)
		}
	}
	for i, s := range []string{"Maps", "Open path", "Fit"} {
		r := image.Rect(i*108, 8, min((i+1)*108-4, 316), 36)
		draw.Draw(pic, r, image.NewUniform(editorGold), image.Point{}, draw.Src)
		line(s, r.Min.X+8, 12, 90, editorInk)
	}
	for i, s := range []string{"Undo", "Redo", "Save As"} {
		enabled := e.editable()
		if enabled && i == 0 {
			enabled = e.doc.Edits.CanUndo()
		} else if enabled && i == 1 {
			enabled = e.doc.Edits.CanRedo()
		}
		bg, ink := editorInk, editorPaper
		if enabled {
			bg, ink = editorGold, editorInk
		}
		r := image.Rect(i*108, 42, min((i+1)*108-4, 316), 64)
		draw.Draw(pic, r, image.NewUniform(bg), image.Point{}, draw.Src)
		line(s, r.Min.X+8, 44, 90, ink)
	}
	if e.doc != nil {
		status := "Saved: "
		if e.Dirty() {
			status = "* Modified: "
		}
		line(editorClip(e.font, status+e.editorHostText(e.doc.Source), 300), 10, 67, 300, editorLight)
		counts := map[string]int{}
		warnings := 0
		for _, r := range e.doc.Records {
			counts[r.Kind]++
			if r.Warning != "" {
				warnings++
			}
		}
		line(fmt.Sprintf("Units %d Build %d Sacks %d", counts["Unit"], counts["Structure"], counts["Sack"]), 10, 88, 300, editorPaper)
		if e.filter == 7 {
			line(fmt.Sprintf("Triggers %d / C%d A%d", counts["Trigger"], counts["Condition"], counts["Action"]), 10, 108, 300, editorPaper)
		} else {
			line(fmt.Sprintf("Scenery %d / %d issues", counts["Scenery"], warnings), 10, 108, 300, editorPaper)
		}
	} else {
		line("Choose an installed map or open an ALM path.", 10, 75, 300, editorPaper)
	}
	if !e.catalog {
		for i, s := range editorFilters {
			x, y := (i%4)*80, (i/4)*24+128
			c := editorPaper
			if i == e.filter {
				draw.Draw(pic, image.Rect(x, y, x+79, y+22), image.NewUniform(editorInk), image.Point{}, draw.Src)
				c = editorGold
			}
			line(editorClip(e.font, s, 74), x+4, y+2, 74, c)
		}
	} else {
		line("INSTALLED MAPS", 10, 143, 300, editorGold)
	}
	rows := e.rows()
	rowCount := len(rows)
	if e.catalog {
		rowCount = len(e.maps)
	}
	if rowCount == 0 && e.filter == 7 && e.doc != nil {
		line(e.doc.ScriptNotice, 10, editorListTop, 300, editorPaper)
	}
	for j := 0; j < e.visibleRows(); j++ {
		n := e.offset + j
		if n >= rowCount {
			break
		}
		y := editorListTop + j*editorRowHeight
		label := ""
		c := editorPaper
		if e.catalog {
			label = e.maps[n].Label
		} else {
			id := rows[n]
			r := e.doc.Records[id]
			label = fmt.Sprintf("%s #%d  %s", r.Kind, r.Index, r.Label)
			if r.Warning != "" {
				label = "! " + label
				c = editorWarning
			}
			if id == e.selected {
				draw.Draw(pic, image.Rect(0, y, 320, y+20), image.NewUniform(editorInk), image.Point{}, draw.Src)
				c = editorGold
			}
		}
		if e.font != nil {
			clipped := editorWrap(e.font, label, 300)
			if len(clipped) > 0 {
				e.font.Draw(pic, clipped[0], 10, y+2, c)
			}
		}
	}
	bottom := e.listBottom()
	line(editorClip(e.font, fmt.Sprintf("%d-%d / %d   PgUp PgDn", min(e.offset+1, rowCount), min(e.offset+e.visibleRows(), rowCount), rowCount), 300), 10, bottom+4, 300, editorPaper)
	draw.Draw(pic, image.Rect(8, bottom+29, 312, bottom+30), image.NewUniform(editorGold), image.Point{}, draw.Src)
	if e.selected >= 0 && e.doc != nil && !e.catalog {
		r := e.doc.Records[e.selected]
		line(fmt.Sprintf("%s #%d / type %d", r.Kind, r.Index, r.Section), 10, bottom+40, 300, editorGold)
		if r.Spatial {
			line(fmt.Sprintf("Cell %d,%d  /  %dx%d", r.Cell.X, r.Cell.Y, r.Size.X, r.Size.Y), 10, bottom+61, 300, editorLight)
		} else if r.Kind == "Trigger" {
			line("Blue: conditions / Gold: actions", 10, bottom+61, 300, editorPaper)
		}
		detailTop := bottom + 86
		if r.Preview != nil && h >= 700 {
			preview := r.Preview.RGBA()
			if preview != nil && !preview.Bounds().Empty() {
				scale := math.Min(1, math.Min(300/float64(preview.Bounds().Dx()), 64/float64(preview.Bounds().Dy())))
				w, hp := max(1, int(float64(preview.Bounds().Dx())*scale)), max(1, int(float64(preview.Bounds().Dy())*scale))
				for y := 0; y < hp; y++ {
					for x := 0; x < w; x++ {
						c := preview.RGBAAt(preview.Rect.Min.X+int(float64(x)/scale), preview.Rect.Min.Y+int(float64(y)/scale))
						if c.A != 0 {
							pic.SetRGBA(10+x, detailTop+y, c)
						}
					}
				}
				detailTop += 70
			}
		}
		for _, row := range e.detailLayout(h) {
			c := inspectionRoleColor(row.Role)
			if row.Reference > 0 && row.Reference == e.focusedReference {
				draw.Draw(pic, row.Rect, image.NewUniform(editorInk), image.Point{}, draw.Src)
			}
			line(row.Text, row.Rect.Min.X, row.Rect.Min.Y, 300, c)
		}
	}
	line(editorClip(e.font, "Drag unit: move / ground: pan", 300), 10, h-39, 300, editorPaper)
	line("Ctrl+Z / Ctrl+Y / Ctrl+S", 10, h-21, 300, editorPaper)
	if e.pathMode || e.message != "" {
		draw.Draw(pic, image.Rect(0, bottom+30, 320, h-45), image.NewUniform(editorInk), image.Point{}, draw.Src)
		for _, row := range e.modalLayout(h) {
			line(row.text, 10, row.y, 300, row.color)
		}
		if e.message != "" {
			line("Map and history retained. Esc: back", 10, h-72, 300, editorPaper)
		}
	}
	return pic
}

// Host paths/errors are UTF-8. Installed record and item labels are already in
// the font's source encoding and must not pass through this conversion twice.
func (e *MapEditor) editorHostText(s string) string {
	if e.hostText != nil {
		return e.hostText(s)
	}
	return s
}

func editorWrap(f *text.Font, s string, width int) []string {
	if f == nil {
		return []string{s}
	}
	var lines []string
	var line string
	for _, r := range []byte(strings.ReplaceAll(s, "\n", " ")) {
		if line != "" && f.Advance(line+string([]byte{r})) > width {
			lines = append(lines, line)
			line = ""
		}
		line += string([]byte{r})
	}
	return append(lines, line)
}

func editorClip(f *text.Font, s string, width int) string {
	if f == nil || f.Advance(s) <= width {
		return s
	}
	r := []byte(s)
	for len(r) > 0 && f.Advance(string(r)+"...") > width {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}

func (e *MapEditor) drawMarkers(dst *ebiten.Image) {
	v := e.doc.Viewer
	// Clip annotations to the canvas, including strokes at its very edge.
	clip := dst.SubImage(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH)).(*ebiten.Image)
	for _, s := range e.markerLines() {
		vector.StrokeLine(clip, s.x0, s.y0, s.x1, s.y1, 1, s.color, false)
	}
}

type editorLine struct {
	x0, y0, x1, y1 float32
	color          color.RGBA
}

func (e *MapEditor) markerLines() []editorLine {
	v := e.doc.Viewer
	var out []editorLine
	for i, r := range e.doc.Records {
		if e.unitDrag != nil && e.unitDrag.moved && i == e.unitDrag.record {
			r.Cell = e.unitDrag.cell // preview only; no model mutation until release
		}
		if !r.Spatial || (r.Warning == "" && i != e.selected) {
			continue
		}
		wx, wy := e.cellCenter(r.Cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		c := editorWarning
		if i == e.selected {
			c = editorGold
		}
		out = append(out, editorLine{float32(x - 5), float32(y), float32(x + 5), float32(y), c}, editorLine{float32(x), float32(y - 5), float32(x), float32(y + 5), c})
		if i == e.selected {
			for cy := r.Cell.Y; cy < r.Cell.Y+max(1, r.Size.Y); cy++ {
				for cx := r.Cell.X; cx < r.Cell.X+max(1, r.Size.X); cx++ {
					if cx < 0 || cy < 0 || cx >= e.doc.Width || cy >= e.doc.Height {
						continue
					}
					verts := flatTileVertices(v.cam, cx, cy)
					if v.Mode() == ModeDisplaced {
						verts = tileVertices(v.cam, v.proj, cx, cy)
					}
					for _, edge := range [][2]int{{0, 1}, {1, 3}, {3, 2}, {2, 0}} {
						a, b := verts[edge[0]], verts[edge[1]]
						out = append(out, editorLine{a.DstX, a.DstY, b.DstX, b.DstY, c})
					}
				}
			}
		}
	}
	out = append(out, e.referenceMarkerLines()...)
	return out
}

func (e *MapEditor) Run() error {
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowSizeLimits(640, 480, -1, -1)
	ebiten.SetWindowTitle("Againrom Map Editor")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(e)
}
