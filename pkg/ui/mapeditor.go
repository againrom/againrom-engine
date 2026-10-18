package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
	Player, Group  int // a unit's owner slot and group number; 0 is none
	// Hover is the status strip's line while the pointer is on the record's
	// cell, in the font's encoding; empty shows nothing.
	Hover string
}

type InspectionDocument struct {
	Viewer        *Viewer
	Source, Name  string
	Width, Height int
	Records       []InspectionRecord
	SourceBytes   []byte // exact current model bytes, not a re-encoded map
	ScriptNotice  string
	Drops         []image.Point          // the cells the map's script drops the party on
	Script        *InspectionScriptModel // the script the trigger editor reads; nil: none
	Edits         InspectionEdits        // optional; host owns persistence and safety
}

type InspectionEdits interface {
	Dirty() bool
	CanUndo() bool
	CanRedo() bool
	// Move, Clone, Delete, Fields and SetField address a record by its
	// InspectionRecord Kind and Index. Players support Delete and Fields.
	Move(kind string, index int, cell image.Point) (*InspectionDocument, error)
	Clone(kind string, index int, cell image.Point) (*InspectionDocument, error)
	Delete(kind string, index int) (*InspectionDocument, error)
	Fields(kind string, index int) []InspectionField
	SetField(kind string, index int, key string, value int64) (*InspectionDocument, error)
	Undo() (*InspectionDocument, error)
	Redo() (*InspectionDocument, error)
	SaveAs(path string) (string, error)
}

// InspectionScript is implemented by edits whose host edits the map's script:
// the drop cell and the targets of its win, lose and protect arms. Goals lists
// the arms; SetGoalTarget points goal at the Unit or Structure record picked
// on the map.
type InspectionScript interface {
	Goals() []InspectionGoal
	SetGoalTarget(goal int, kind string, index int) (*InspectionDocument, error)
	SetDropCell(cell image.Point) (*InspectionDocument, error)
}

// InspectionGoal is one row of the script panel. Pick is the record kind a
// click on the map must name. Kind and Index address the script record the
// row edits, Index -1 for the row that adds one. Cell is where the current
// target stands, when HasCell.
type InspectionGoal struct {
	Arm, Text string
	Pick      string
	Kind      string
	Index     int
	Cell      image.Point
	HasCell   bool
}

// InspectionField is one settable field of a selected record and its value.
type InspectionField struct {
	Key, Label string
	Value      int64
	Text       bool
	TextValue  string
	Choices    []InspectionChoice
}

// InspectionChoice is one value a field can take; Label is in the font's bytes.
type InspectionChoice struct {
	Value int64
	Label string
}

type EditorMap struct{ Key, Label string }
type InspectionOpener func(string) (*InspectionDocument, error)

// MapEditor is an authored-map canvas. It has no game App, command queue,
// mission or simulation ticking seam. Host callbacks own edits and file writes.
type MapEditor struct {
	doc                    *InspectionDocument
	font                   *text.Font
	maps                   []EditorMap
	open                   InspectionOpener
	hostText               func(string) string
	pathDialog             editorPathDialog
	selected               int
	filter                 int
	editorMode             // the window's mode: catalogue, prompt, pick, brush and report
	message                string
	down, moved            bool
	press                  image.Point
	window                 image.Point
	focusedReference       int // 1-based on the selected record; 0 means none
	recordDrag             *editorRecordDrag
	cancelledDrag          bool // consume the remainder of a cancelled press
	textSmoothingEnabled   bool
	textOverlay            textOverlay
	textLayers             []textLayer
	textCaptured, textKept int
	editorChrome
}

func NewMapEditor(font *text.Font, maps []EditorMap, open InspectionOpener) *MapEditor {
	return &MapEditor{font: font, maps: maps, open: open, selected: -1, editorMode: editorMode{catalog: true}, window: image.Pt(1280, 800), textSmoothingEnabled: true,
		editorChrome: editorChrome{obj: objectEdit{record: -1, focus: -1, open: -1, cell: -1, pickField: -1}}}
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

// Open opens the map named source. With unsaved changes it asks first.
func (e *MapEditor) Open(source string) error {
	if e.Dirty() {
		e.askLeave(leaveStep{open: true, source: source})
		return nil
	}
	return e.openMap(source)
}

func (e *MapEditor) openMap(source string) error {
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
	e.leave(leaveDrag | leaveReport | leaveCatalog | leavePrompt | leavePanel)
	e.selected, e.filter, e.listShown = -1, 0, false
	if _, ok := e.mapPanel(); ok {
		e.openPanel()
	}
	e.focusedReference, e.objectTop, e.groupSel = 0, 0, editorGroup{}
	e.objectBack = nil
	e.listWheel, e.objectWheel = 0, 0
	e.message = ""
	e.brush.endStroke()
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
	if _, ok := e.scriptVariableRow(index); ok {
		e.selected, e.objectTop, e.objectHidden = index, 0, false
		e.groupSel, e.objectBack = editorGroup{}, nil
		e.obj.open, e.obj.focus = -1, -1
		return true
	}
	if e.doc == nil || index < 0 || index >= len(e.doc.Records) {
		return false
	}
	e.selected, e.objectTop, e.objectHidden = index, 0, false
	e.focusedReference, e.objectWheel = 0, 0
	e.groupSel, e.objectBack = editorGroup{}, nil
	r := e.doc.Records[index]
	if jump && r.Spatial {
		e.centerOn(r.Cell)
	}
	if c, ok := e.triggerCentre(index); jump && ok {
		e.centerOn(c)
	}
	return true
}

// centerOn scrolls the map to cell at full zoom.
func (e *MapEditor) centerOn(cell image.Point) {
	v := e.doc.Viewer
	x, y := e.cellCenter(cell)
	v.cam.SetZoom(1)
	v.cam.X = x - float64(v.cam.ViewW)/2
	v.cam.Y = y - float64(v.cam.ViewH)/2
	v.cam.Clamp()
}

// cellCenter is the world centre of cell, clamped into the map: the point
// where the game draws a path leg on that cell.
func (e *MapEditor) cellCenter(cell image.Point) (float64, float64) {
	cell.X = max(0, min(cell.X, e.doc.Width-1))
	cell.Y = max(0, min(cell.Y, e.doc.Height-1))
	return e.doc.Viewer.cellWorldCentre(cell)
}

func (e *MapEditor) rows() []int {
	if e.doc == nil {
		return nil
	}
	if e.filter == editorScriptsFilter {
		var out []int
		for _, section := range e.scriptSections() {
			out = append(out, section...)
		}
		return out
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
		case editorPlayersFilter:
			show = r.Kind == "Player"
		}
		if show {
			out = append(out, i)
		}
	}
	return out
}

func (e *MapEditor) Pointer(in Input, now time.Time) {
	if e.pathDialog.pending != nil {
		return
	}
	if e.chromePointer(in, now) {
		if e.doc != nil {
			e.doc.Viewer.dragging = false
		}
		return
	}
	if e.doc == nil {
		e.down = in.PrimaryDown
		return
	}
	x, y := e.doc.Viewer.windowToFrame(in.CursorX, in.CursorY)
	rail := e.doc.Viewer.cam.ViewW
	p := image.Pt(x, y)
	e.hoverAt(p)
	if e.brushPointer(in, p, rail) {
		return
	}
	if e.dragRecord(in, p, rail) {
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
	if e.blocksCanvas() {
		return
	}
	e.doc.Viewer.step(in, now)
	if released {
		if e.pick != pickNone {
			e.pickAt(p)
			return
		}
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

func (e *MapEditor) Update() error {
	if e.exitAsked {
		return ebiten.Termination
	}
	if e.PollPathDialog() {
		return nil
	}
	if e.pathDialog.rearm {
		if len(inpututil.AppendPressedKeys(nil)) == 0 && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
			e.pathDialog.rearm = false
		}
		return nil
	}
	if ebiten.IsWindowBeingClosed() {
		if err := e.Key("close"); err != nil {
			return err
		}
	}
	if e.prompt.open() || e.objectTyping() {
		e.TextInput(string(ebiten.AppendInputChars(nil)))
	}
	for _, k := range []struct {
		key  ebiten.Key
		name string
	}{{ebiten.KeyEscape, "escape"}, {ebiten.KeyEnter, "enter"}, {ebiten.KeyNumpadEnter, "enter"}, {ebiten.KeyBackspace, "backspace"}, {ebiten.KeyPageUp, "pageup"}, {ebiten.KeyPageDown, "pagedown"}, {ebiten.KeyHome, "home"}, {ebiten.KeyEnd, "end"}, {ebiten.KeyArrowUp, "up"}, {ebiten.KeyArrowDown, "down"}} {
		if inpututil.IsKeyJustPressed(k.key) {
			name := k.name
			if name == "enter" && ebiten.IsKeyPressed(ebiten.KeyShift) && e.multilinePrompt() {
				name = "newline"
			}
			if err := e.Key(name); err == ebiten.Termination {
				return err
			}
		}
	}
	if !e.prompt.open() && !e.objectTyping() {
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
		if e.pathDialog.pending != nil {
			return nil
		}
		for _, k := range []struct {
			key  ebiten.Key
			name string
		}{{ebiten.KeyC, "clone"}, {ebiten.KeyDelete, "delete"}, {ebiten.KeyE, "field"}, {ebiten.KeyS, "sack"}, {ebiten.KeyD, "drop"}, {ebiten.KeyG, "goals"}, {ebiten.KeyT, "terrain"}, {ebiten.KeyH, "height"}, {ebiten.KeyB, "objects"}, {ebiten.KeyLeftBracket, "radiusdown"}, {ebiten.KeyRightBracket, "radiusup"}, {ebiten.KeyN, "new"}, {ebiten.KeyU, "unit"}, {ebiten.KeyJ, "structure"}, {ebiten.KeyK, "stock"}, {ebiten.KeyI, "recipe"}, {ebiten.KeyP, "players"}, {ebiten.KeyA, "player"}, {ebiten.KeyF5, "testplay"}, {ebiten.KeyM, "metadata"}, {ebiten.KeyX, "pages"}, {ebiten.KeyW, "package"}, {ebiten.KeyF1, "help"}, {ebiten.KeyF2, "scripts"}, {ebiten.KeyL, "drawarea"}} {
			if inpututil.IsKeyJustPressed(k.key) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
				_ = e.Key(k.name)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF) {
			e.Fit()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
			_ = e.Key("report")
		}
	} else {
		if (e.prompt.fields() || e.objectTyping()) && inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			_ = e.Key("tab")
		}
		if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyA) {
			_ = e.Key("clear")
		}
	}
	e.Pointer(readInput(), time.Now())
	return nil
}

func (e *MapEditor) Layout(w, h int) (int, int) {
	e.window = image.Pt(w, h)
	if e.doc != nil && w > 0 && h > 0 {
		v := e.doc.Viewer
		area := e.mapArea()
		fw, fh := area.Dx(), area.Dy()
		if (v.frameW != fw || v.frameH != fh) && v.canvas != nil {
			disposeFrameCanvas(v.canvas)
			v.canvas = nil
		}
		// The frame is the map area: centred in the window between a bar and
		// a strip of equal height, it sits at scale 1 below the bar.
		v.frameW, v.frameH = fw, fh
		v.place = frame.Fit(fw, fh, w, h)
		v.syncMapViewport()
		v.cam.SetMinimumZoom(math.Min(camera.ZoomMin, e.fitZoom()))
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
	if e.doc != nil {
		v := e.doc.Viewer
		v.syncMapViewport()
		if v.canvas == nil {
			v.canvas = ebiten.NewImage(v.frameW, v.frameH)
		}
		v.canvas.Clear()
		v.drawFrame(v.canvas)
		e.drawMarkers(v.canvas)
		ox, oy := v.place.Origin()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(v.place.Scale(), v.place.Scale())
		op.GeoM.Translate(ox, oy)
		screen.DrawImage(v.canvas, op)
	}
	if kept := e.drawChrome(screen); len(kept) > 0 {
		e.textOverlay.draw(screen, kept, 1, 0, 0)
	}
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
			// Break after the line's last space when it has one, so a
			// word moves whole; the bytes stay in order either way.
			cut := strings.LastIndexByte(line, ' ') + 1
			if cut <= 0 || r == ' ' {
				cut = len(line)
			}
			lines = append(lines, line[:cut])
			line = line[cut:]
		}
		line += string([]byte{r})
	}
	return append(lines, line)
}

// editorFit is how the editor fits its text: in the font's bytes, one byte
// one glyph, cut at the end.
func editorFit(f *text.Font) textfit.Spec {
	return textfit.Spec{Measure: f.Advance, Unit: textfit.Byte, Side: textfit.End}
}

func editorClip(f *text.Font, s string, width int) string {
	if f == nil {
		return s
	}
	return editorFit(f).Fit(s, width)
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
		if e.recordDrag != nil && e.recordDrag.moved && i == e.recordDrag.record {
			r.Cell = e.recordDrag.cell // preview only; no model mutation until release
		}
		issue := !e.view.hideIssues && (r.Warning != "" || e.report.flagged[findingKey(r.Kind, r.Index)])
		if !r.Spatial || (!issue && i != e.selected) {
			continue
		}
		wx, wy := e.cellCenter(r.Cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		c := editorWarning
		if i == e.selected {
			c = editorGold
		}
		out = append(out, editorLine{float32(x - 5), float32(y), float32(x + 5), float32(y), c}, editorLine{float32(x), float32(y - 5), float32(x), float32(y + 5), c})
		if i == e.selected && !e.view.hideFootprints {
			for cy := r.Cell.Y; cy < r.Cell.Y+max(1, r.Size.Y); cy++ {
				for cx := r.Cell.X; cx < r.Cell.X+max(1, r.Size.X); cx++ {
					if cx < 0 || cy < 0 || cx >= e.doc.Width || cy >= e.doc.Height {
						continue
					}
					verts := v.cellSpace().vertices(v.cam, cx, cy)
					for _, edge := range [][2]int{{0, 1}, {1, 3}, {3, 2}, {2, 0}} {
						a, b := verts[edge[0]], verts[edge[1]]
						out = append(out, editorLine{a.DstX, a.DstY, b.DstX, b.DstY, c})
					}
				}
			}
		}
	}
	for _, cell := range e.reportCrosses() {
		wx, wy := e.cellCenter(cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		out = append(out, editorLine{float32(x - 5), float32(y), float32(x + 5), float32(y), editorGold}, editorLine{float32(x), float32(y - 5), float32(x), float32(y + 5), editorGold})
	}
	for _, cell := range e.dropMarkers() {
		if e.view.hideDrops {
			break
		}
		wx, wy := e.cellCenter(cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		for _, d := range [][4]float32{{-8, -8, 8, -8}, {8, -8, 8, 8}, {8, 8, -8, 8}, {-8, 8, -8, -8}, {-8, -8, 8, 8}, {-8, 8, 8, -8}} {
			out = append(out, editorLine{float32(x) + d[0], float32(y) + d[1], float32(x) + d[2], float32(y) + d[3], editorCondition})
		}
	}
	out = append(out, e.areaLines()...)
	out = append(out, e.referenceMarkerLines()...)
	out = append(out, e.brushMarkerLines()...)
	out = append(out, e.groupMarkerLines()...)
	return append(e.gridLines(), out...)
}

func (e *MapEditor) Run() error {
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowSizeLimits(640, 480, -1, -1)
	ebiten.SetWindowTitle("Againrom Map Editor")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(e)
}
