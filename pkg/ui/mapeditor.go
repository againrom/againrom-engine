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
	Drops         []image.Point   // the cells the map's script drops the party on
	Edits         InspectionEdits // optional; host owns persistence and safety
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
	AddSack(cell image.Point, item int64) (*InspectionDocument, error)
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

type InspectionChoice struct {
	Value int64
	Label string
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
	pathDialog                     editorPathDialog
	selected, offset, detailOffset int
	listWheel, detailWheel         float64 // fractional rows belong to their viewport
	filter                         int
	editorMode                     // the window's mode: catalogue, prompt, pick, brush and report
	message                        string
	down, moved                    bool
	press                          image.Point
	window                         image.Point
	panelImage                     *ebiten.Image
	focusedReference               int // 1-based on the selected record; 0 means none
	recordDrag                     *editorRecordDrag
	cancelledDrag                  bool // consume the remainder of a cancelled press
	textSmoothingEnabled           bool
	textOverlay                    textOverlay
	textLayers                     []textLayer
	textCaptured, textKept         int
}

func NewMapEditor(font *text.Font, maps []EditorMap, open InspectionOpener) *MapEditor {
	return &MapEditor{font: font, maps: maps, open: open, selected: -1, editorMode: editorMode{catalog: true}, window: image.Pt(1280, 800), textSmoothingEnabled: true}
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
	e.leave(leaveDrag | leaveReport | leaveCatalog | leavePrompt)
	e.selected, e.offset, e.detailOffset, e.filter = -1, 0, 0, 0
	e.focusedReference = 0
	e.listWheel, e.detailWheel = 0, 0
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
		e.centerOn(r.Cell)
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
	bottom := min(e.window.Y/2+30, e.helpTop(e.window.Y)-132)
	return max(1, min(12, (bottom-editorListTop)/editorRowHeight))
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
		case editorPlayersFilter:
			show = r.Kind == "Player"
		}
		if show {
			out = append(out, i)
		}
	}
	return out
}

func (e *MapEditor) rowCount() int {
	if e.brushActive() {
		return len(e.brushRows())
	}
	if e.catalog {
		return len(e.maps)
	}
	if e.report.open {
		return len(e.report.rows)
	}
	return len(e.rows())
}

func (e *MapEditor) Pointer(in Input, now time.Time) {
	if e.pathDialog.pending != nil {
		return
	}
	x, y := in.CursorX, in.CursorY
	rail := e.window.X - editorRailWidth
	if e.doc != nil {
		x, y = e.doc.Viewer.windowToFrame(x, y)
		rail = e.doc.Viewer.cam.ViewW
	}
	p := image.Pt(x, y)
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
	if e.prompt.open() {
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
	if e.doc == nil || e.blocksCanvas() {
		return
	}
	e.doc.Viewer.step(in, now)
	if released && e.press.X < rail {
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

func (e *MapEditor) railClick(x, y int) {
	if e.prompt.open() {
		return
	}
	if y >= 42 && y < 64 {
		_ = e.Key([]string{"undo", "redo", "saveas"}[min(2, max(0, x/108))])
		return
	}
	if e.listView() && !e.prompt.open() && e.message == "" {
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
	if e.listView() && !e.brushActive() && e.filter != editorPlayersFilter && y >= 128 && y < 176 {
		idx := (y-128)/24*4 + x/80
		if idx < len(editorFilters) {
			e.filter = idx
			e.resetList(0)
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
	} else if e.report.open {
		e.selectFinding(row)
	} else if e.brushActive() {
		e.brushRowClick(row)
	} else {
		rows := e.rows()
		if row < len(rows) {
			e.Select(rows[row], true)
		}
	}
}

func (e *MapEditor) Update() error {
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
	if e.prompt.open() {
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
	if !e.prompt.open() {
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
		}{{ebiten.KeyC, "clone"}, {ebiten.KeyDelete, "delete"}, {ebiten.KeyE, "field"}, {ebiten.KeyS, "sack"}, {ebiten.KeyD, "drop"}, {ebiten.KeyG, "goals"}, {ebiten.KeyT, "terrain"}, {ebiten.KeyH, "height"}, {ebiten.KeyB, "objects"}, {ebiten.KeyLeftBracket, "radiusdown"}, {ebiten.KeyRightBracket, "radiusup"}, {ebiten.KeyN, "new"}, {ebiten.KeyU, "unit"}, {ebiten.KeyJ, "structure"}, {ebiten.KeyK, "stock"}, {ebiten.KeyI, "recipe"}, {ebiten.KeyP, "players"}, {ebiten.KeyA, "player"}, {ebiten.KeyF5, "testplay"}, {ebiten.KeyM, "metadata"}, {ebiten.KeyX, "pages"}, {ebiten.KeyW, "package"}, {ebiten.KeyF1, "help"}} {
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
		if e.prompt.fields() && inpututil.IsKeyJustPressed(ebiten.KeyTab) {
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
		} else if e.filter == editorPlayersFilter {
			line(fmt.Sprintf("Players %d", counts["Player"]), 10, 108, 300, editorPaper)
		} else {
			line(fmt.Sprintf("Scenery %d / %d issues", counts["Scenery"], warnings), 10, 108, 300, editorPaper)
		}
	} else {
		line("Choose an installed map or open an ALM path.", 10, 75, 300, editorPaper)
	}
	if e.report.open {
		line(editorClip(e.font, e.report.note, 300), 10, 143, 300, editorGold)
	} else if e.brushActive() {
		line(editorClip(e.font, e.brushStatus(), 300), 10, 143, 300, editorGold)
	} else if !e.catalog && e.filter == editorPlayersFilter {
		line("PLAYERS", 10, 130, 300, editorGold)
		line("Name, colour, control, relations", 10, 154, 300, editorPaper)
	} else if !e.catalog {
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
		line("NEW AND INSTALLED MAPS", 10, 143, 300, editorGold)
	}
	rows := e.rows()
	rowCount := len(rows)
	if e.catalog {
		rowCount = len(e.maps)
	} else if e.report.open {
		rowCount = len(e.report.rows)
	} else if e.brushActive() {
		rowCount = len(e.brushRows())
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
		} else if e.report.open {
			f := e.report.rows[n]
			label = reportLabel(f)
			if len(e.report.goals) > 0 {
				label = goalLabel(e.report.goals[n])
			}
			if f.Error {
				c = editorWarning
			}
			if n == e.report.sel {
				draw.Draw(pic, image.Rect(0, y, 320, y+20), image.NewUniform(editorInk), image.Point{}, draw.Src)
				c = editorGold
			}
		} else if e.brushActive() {
			label = e.brushRows()[n]
			if n == e.brush.sel[e.brush.mode] {
				draw.Draw(pic, image.Rect(0, y, 320, y+20), image.NewUniform(editorInk), image.Point{}, draw.Src)
				c = editorGold
			}
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
	if e.report.open && e.report.sel < len(e.report.rows) {
		f := e.report.rows[e.report.sel]
		step := 16
		if e.font != nil {
			step = max(16, e.font.Height()+2)
		}
		text := f.Line()
		if len(e.report.goals) > 0 {
			text = goalLabel(e.report.goals[e.report.sel])
		}
		for n, row := range editorWrap(e.font, text, 300) {
			if y := bottom + 40 + n*step; y+step <= e.helpTop(h)-6 {
				line(row, 10, y, 300, editorLight)
			}
		}
	} else if e.selected >= 0 && e.doc != nil && !e.catalog && !e.brushActive() {
		r := e.doc.Records[e.selected]
		if bottom+40+e.helpStep() <= e.helpTop(h)-6 {
			line(fmt.Sprintf("%s #%d / type %d", r.Kind, r.Index, r.Section), 10, bottom+40, 300, editorGold)
		}
		if r.Spatial {
			if bottom+61+e.helpStep() <= e.helpTop(h)-6 {
				line(fmt.Sprintf("Cell %d,%d  /  %dx%d", r.Cell.X, r.Cell.Y, r.Size.X, r.Size.Y), 10, bottom+61, 300, editorLight)
			}
		} else if r.Kind == "Trigger" {
			if bottom+61+e.helpStep() <= e.helpTop(h)-6 {
				line("Blue: conditions / Gold: actions", 10, bottom+61, 300, editorPaper)
			}
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
						if c.A != 0 && detailTop+y < e.helpTop(h)-6 {
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
	for n, help := range e.helpLines() {
		line(editorClip(e.font, help, 300), 10, e.helpTop(h)+n*e.helpStep(), 300, editorPaper)
	}
	line(e.saveHelp(), 10, e.saveTop(h), 300, editorPaper)
	if e.prompt.open() || e.message != "" {
		draw.Draw(pic, image.Rect(0, bottom+30, 320, e.helpTop(h)-6), image.NewUniform(editorInk), image.Point{}, draw.Src)
		for _, row := range e.modalLayout(h) {
			line(row.text, 10, row.y, 300, row.color)
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
		if e.recordDrag != nil && e.recordDrag.moved && i == e.recordDrag.record {
			r.Cell = e.recordDrag.cell // preview only; no model mutation until release
		}
		if !r.Spatial || (r.Warning == "" && i != e.selected && !e.report.flagged[findingKey(r.Kind, r.Index)]) {
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
	for _, cell := range e.reportCrosses() {
		wx, wy := e.cellCenter(cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		out = append(out, editorLine{float32(x - 5), float32(y), float32(x + 5), float32(y), editorGold}, editorLine{float32(x), float32(y - 5), float32(x), float32(y + 5), editorGold})
	}
	for _, cell := range e.dropMarkers() {
		wx, wy := e.cellCenter(cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		for _, d := range [][4]float32{{-8, -8, 8, -8}, {8, -8, 8, 8}, {8, 8, -8, 8}, {-8, 8, -8, -8}, {-8, -8, 8, 8}, {-8, 8, 8, -8}} {
			out = append(out, editorLine{float32(x) + d[0], float32(y) + d[1], float32(x) + d[2], float32(y) + d[3], editorCondition})
		}
	}
	out = append(out, e.referenceMarkerLines()...)
	out = append(out, e.brushMarkerLines()...)
	return out
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
