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
	Player, Group  int // a unit's or structure's owner slot and a unit's group number; 0 is none
	// ScriptID is the number a script parameter names a unit or structure
	// by; 0 for any other record.
	ScriptID int
	// Hover is the status strip's line while the pointer is on the record's
	// cell, in the font's encoding; empty shows nothing.
	Hover string
	// Definition names the definition a unit's placement sets, in the font's
	// encoding; empty for a unit whose class alone selects it, and for any
	// other record.
	Definition string

	// Footprint is the cells the record takes in the world the game builds
	// from the map; empty: the cells of Cell and Size.
	Footprint image.Rectangle
}

type InspectionDocument struct {
	Viewer        *Viewer
	Source, Name  string
	Width, Height int
	Records       []InspectionRecord
	AddedRecord   RecordRef
	SourceBytes   []byte // exact current model bytes, not a re-encoded map
	ScriptNotice  string
	// AgainromOnly marks a map whose script holds an Againrom operation: the
	// original cannot play its Againrom nodes (DIV-4009).
	AgainromOnly bool
	Drops        []image.Point          // the cells the map's script drops the party on
	Script       *InspectionScriptModel // the script the trigger editor reads; nil: none
	Edits        InspectionEdits        // optional; host owns persistence and safety
	// Passability is, per cell and row-major, the arms that close the cell
	// to a ground mover in the world the game builds from this map: zero for
	// an open cell. nil: the host derives none and the view draws nothing.
	Passability []PassArm
	// PassArms names the bits of Passability, in the order the status line
	// lists them; the host reads them from its one arm table.
	PassArms []PassArmRow
	// Refused is, in a placement's preview, each record the placement
	// refuses, with its footprint where it would have stood.
	Refused []SelectionRefusal
	// Playable is the cells inside the map's border ring; empty when the
	// ring covers the map or the host names none.
	Playable image.Rectangle
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

// InspectionSaveFolder is implemented by edits whose host knows where the
// Save As window starts; an empty answer leaves the window's own start.
type InspectionSaveFolder interface {
	SaveFolder() string
}

// InspectionInPlace is implemented by edits that can save over the map's own
// host file. OwnFile says whether there is one; SaveInPlace writes it.
type InspectionInPlace interface {
	OwnFile() (string, bool)
	SaveInPlace() (string, error)
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
	OpenKind  string
	OpenIndex int
}

// InspectionField is one control or derived read-only block of a record.
type InspectionField struct {
	Column          int
	Action          bool
	Icons           map[int64]*image.RGBA
	Section, Parent string
	ReadOnly        bool
	Lines           []InspectionLine
	Key, Label      string
	Meaning         string
	Value           int64
	Text            bool
	TextValue       string
	Choices         []InspectionChoice
	// Group is the heading the field shows under; "" for none.
	Group string
	// Inherit marks a value the record may leave to its template: Absent is
	// the value the record stores for that, which an empty field writes, and
	// Inherited is the value the game uses meanwhile. An Absent of 0 means
	// the record cannot store 0 itself.
	Inherit   bool
	Absent    int64
	Inherited string
	// Shut lists choice values the list shows but refuses, each label
	// carrying its reason.
	Shut []int64
	// Toward marks a player's relation field: the relation toward player
	// Toward, counted from 1; zero for every other field.
	Toward int
}

// InspectionChoice is one value a field can take; Label is in the font's bytes.
type InspectionChoice struct {
	Value int64
	Label string
}

// EditorMap is one row of the map list: its key, its label, and whether the
// list shows it among the multiplayer maps (else the single-player maps).
type EditorMap struct {
	Key, Label  string
	Multiplayer bool
}
type InspectionOpener func(string) (*InspectionDocument, error)

// MapEditor is an authored-map canvas. It has no game App, command queue,
// mission or simulation ticking seam. Host callbacks own edits and file writes.
type MapEditor struct {
	mcp                         editorMCP
	doc                         *InspectionDocument
	font                        *text.Font
	maps                        []EditorMap
	open                        InspectionOpener
	hostText                    func(string) string
	hostDecode                  func(string) string
	heroRoles                   map[int64]string // hero N's role in host text; empty: the label is a number
	nativeInput                 func() editorNativeInput
	pathDialog                  editorPathDialog
	selected                    int
	filter                      int
	editorMode                  // the window's mode: catalogue, prompt, pick, brush and report
	message                     string
	down, moved                 bool
	press                       image.Point
	window                      image.Point
	focusedReference            int // 1-based on the selected record; 0 means none
	recordDrag                  *editorRecordDrag
	cancelledDrag               bool // consume the remainder of a cancelled press
	textSmoothingEnabled        bool
	textOverlay                 textOverlay
	textLayers                  []textLayer
	textCaptured, textKept      int
	placementKey                string
	placementDoc, placementView *InspectionDocument
	placementErr                error
	dragCheckKey                string // the move dragCheck answers, on dragCheckDoc
	dragCheckDoc                *InspectionDocument
	dragCheck                   map[RecordRef]bool
	cancelRestore               func()
	listRestore                 func()
	selection                   selectionState
	input                       editorInputState
	place                       placeGrid
	relations                   relationMatrix
	lastSeed                    uint32 // the seed the last whole-map command took
	panels                      editorPanels
	tree                        editorTree
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
func (e *MapEditor) SetHostTextEncoder(encode func(string) string, decode ...func(string) string) {
	e.hostText = encode
	if len(decode) > 0 {
		e.hostDecode = decode[0]
	}
}

// SetHeroRoles supplies the roles in host text the hero band's labels name:
// hero N with a role reads "Hero N: role" (HeroLabels).
func (e *MapEditor) SetHeroRoles(roles map[int64]string) { e.heroRoles = roles }

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
	e.saveStatus = saveNotice{}
	e.cancelRestore = nil
	e.listRestore = nil
	e.Layout(e.window.X, e.window.Y)
	e.leave(leaveDrag | leaveReport | leaveCatalog | leavePrompt | leavePanel)
	e.selected, e.filter, e.listShown = -1, 0, false
	e.focusedReference, e.objectTop, e.groupSel = 0, 0, editorGroup{}
	e.objectBack = nil
	e.messagePage = 0
	e.listWheel, e.objectWheel = 0, 0
	e.message = ""
	e.brush.endStroke()
	e.clearSelection()
	e.Fit()
	if _, ok := e.mapPanel(); ok {
		e.openPanel()
	}
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
	if index != e.selected {
		e.panels.detached = ""
	}
	e.messagePage = 0
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
	e.triggerDetails = 0
	e.focusedReference, e.objectWheel = 0, 0
	e.groupSel, e.objectBack = editorGroup{}, nil
	e.replaceGroup(nil)
	e.noteSelection("", nil)
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

func (e *MapEditor) rows() []int { return e.recordsOf(e.filter) }

// recordsOf are the records the records list filter shows, in its order.
func (e *MapEditor) recordsOf(filter int) []int {
	if e.doc == nil {
		return nil
	}
	if filter == editorScriptsFilter {
		var out []int
		for _, section := range e.scriptSections() {
			out = append(out, section...)
		}
		return out
	}
	var out []int
	for i, r := range e.doc.Records {
		show := filter == 0
		switch filter {
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
	if in.SecondaryDown && e.pick != pickNone {
		e.leave(leavePick)
		return
	}
	// A right press clears the brush as Esc does; in Forest mode the right
	// button thins instead.
	if in.SecondaryDown && e.brushActive() && !e.forestOn() && !e.brush.down {
		if e.listKind() == "brush" {
			e.cancelPopup(popList)
		} else {
			e.leave(leaveBrush)
		}
		return
	}
	if e.pathDialog.pending != nil {
		return
	}
	if in.Alt && in.WheelY != 0 && e.brushActive() {
		// Alt+wheel sizes the brush; the plain wheel still zooms and scrolls.
		e.brushWheel(in.WheelY)
		in.WheelY = 0
	}
	if e.chromePointer(in, now) {
		e.hoverOK = false
		if e.doc != nil {
			e.doc.Viewer.dragging = false
		}
		return
	}
	if e.doc == nil {
		e.down = in.PrimaryDown
		return
	}
	// The map's own window starts right of the left panel.
	in.CursorX -= e.mapArea().Min.X
	x, y := e.doc.Viewer.windowToFrame(in.CursorX, in.CursorY)
	rail := e.doc.Viewer.cam.ViewW
	p := image.Pt(x, y)
	e.hoverAt(p)
	if e.placePointer(in, p, rail) {
		return
	}
	if e.brushPointer(in, p, rail) {
		return
	}
	if e.marqueePointer(in, p, rail) {
		return
	}
	if e.dragRecord(in, p, rail) {
		return
	}
	if e.boxPointer(in, p, rail) {
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
	e.mcpWatch()
	if e.updateMCP() {
		e.mcpTrackNative()
		return nil
	}
	if e.exitAsked {
		return ebiten.Termination
	}
	e.deferSaves = true
	defer func() { e.deferSaves = false }()
	if ran, err := e.runDeferredSave(); ran {
		if err == ebiten.Termination {
			return err
		}
		return nil
	}
	if e.PollPathDialog() {
		return nil
	}
	in := readEditorInput
	if e.nativeInput != nil {
		in = e.nativeInput
	}
	native := in()
	if e.pathDialog.rearm {
		if !native.keysDown && !native.canvas.PrimaryDown && !native.canvas.SecondaryDown && !native.canvas.MiddleDown {
			e.pathDialog.rearm = false
		}
		return nil
	}
	if native.closing {
		if err := e.Key("close"); err != nil {
			return err
		}
	}
	if handled, err := e.modNativeKeys(native); handled {
		return err
	}
	typing, policy := e.updateTextInput(native, time.Now())
	for _, k := range []struct {
		key  ebiten.Key
		name string
	}{{ebiten.KeyEscape, "escape"}, {ebiten.KeyEnter, "enter"}, {ebiten.KeyNumpadEnter, "enter"}, {ebiten.KeyBackspace, "backspace"}, {ebiten.KeyPageUp, "pageup"}, {ebiten.KeyPageDown, "pagedown"}, {ebiten.KeyHome, "home"}, {ebiten.KeyEnd, "end"}, {ebiten.KeyArrowUp, "up"}, {ebiten.KeyArrowDown, "down"}} {
		if native.justPressed(k.key) {
			name := k.name
			if (name == "up" || name == "down") && e.messagePage != 0 && native.pressed(ebiten.KeyAlt) {
				name = "card" + name // the dialog editor's card moves, even while typing
			}
			if typing && name != "escape" && name != "enter" && name != "pageup" && name != "pagedown" && name != "cardup" && name != "carddown" {
				continue
			}
			if name == "enter" && native.pressed(ebiten.KeyShift) && e.multilinePrompt() {
				name = "newline"
			}
			if name == "enter" && native.pressed(ebiten.KeyControl) && e.nativeUnitWindow() {
				name = "applyobject"
			}
			if err := e.Key(name); err == ebiten.Termination {
				return err
			}
		}
	}
	if !e.prompt.open() && !e.objectTyping() && !e.treeTyping() {
		for _, p := range sidePanels {
			if native.justPressed(p.hotkey) {
				_ = e.Key(p.key)
			}
		}
		if native.pressed(ebiten.KeyControl) {
			if native.justPressed(ebiten.KeyZ) {
				key := "undo"
				if native.pressed(ebiten.KeyShift) {
					key = "redo"
				}
				_ = e.Key(key)
			}
			if native.justPressed(ebiten.KeyY) {
				_ = e.Key("redo")
			}
			if native.justPressed(ebiten.KeyS) {
				_ = e.Key("saveas")
			}
			if native.justPressed(ebiten.KeyC) {
				_ = e.Key("copy")
			}
			if native.justPressed(ebiten.KeyV) {
				_ = e.Key("paste")
			}
			if native.justPressed(ebiten.KeyE) {
				_ = e.Key("randomterrain")
			}
			if native.justPressed(ebiten.KeyR) && native.pressed(ebiten.KeyShift) {
				_ = e.Key("randomheights")
			}
		}
		for n := 0; n < 9; n++ {
			if native.justPressed(ebiten.KeyDigit1 + ebiten.Key(n)) {
				key := "group"
				if native.pressed(ebiten.KeyControl) {
					key = "togroup"
				}
				_ = e.Key(key + string(rune('1'+n)))
			}
		}
		if native.justPressed(ebiten.KeyO) {
			_ = e.Key("open")
		}
		if e.pathDialog.pending != nil {
			return nil
		}
		for _, k := range []struct {
			key  ebiten.Key
			name string
		}{{ebiten.KeyC, "clone"}, {ebiten.KeyDelete, "delete"}, {ebiten.KeyE, "field"}, {ebiten.KeyS, "sack"}, {ebiten.KeyD, "drop"}, {ebiten.KeyG, "goals"}, {ebiten.KeyT, "terrain"}, {ebiten.KeyH, "height"}, {ebiten.KeyB, "objects"}, {ebiten.KeyQ, "passability"}, {ebiten.KeyLeftBracket, "radiusdown"}, {ebiten.KeyRightBracket, "radiusup"}, {ebiten.KeyN, "new"}, {ebiten.KeyU, "unit"}, {ebiten.KeyJ, "structure"}, {ebiten.KeyK, "stock"}, {ebiten.KeyI, "recipe"}, {ebiten.KeyP, "players"}, {ebiten.KeyA, "player"}, {ebiten.KeyF5, "testplay"}, {ebiten.KeyM, "metadata"}, {ebiten.KeyX, "pages"}, {ebiten.KeyW, "package"}, {ebiten.KeyF1, "help"}, {ebiten.KeyF2, "scripts"}, {ebiten.KeyL, "drawarea"}, {ebiten.KeyV, "heightlabels"}, {ebiten.KeyZ, "passview"}, {ebiten.KeyY, "playertint"}} {
			if native.justPressed(k.key) && !native.pressed(ebiten.KeyControl) {
				_ = e.Key(k.name)
			}
		}
		if native.justPressed(ebiten.KeyF3) {
			_ = e.Key("casts")
		}
		if native.justPressed(ebiten.KeyF) {
			e.Fit()
		}
		if native.justPressed(ebiten.KeyR) && !native.pressed(ebiten.KeyControl) {
			_ = e.Key("report")
		}
	}
	if native.justPressed(ebiten.KeyTab) && (e.prompt.fields() || e.objectTyping() || e.nativeUnitWindow() && !e.objectHidden) {
		_ = e.Key("tab")
	}
	canvas := e.mcpCanvas(native.canvas)
	if typing && !policy.panArrows {
		canvas.PanLeft, canvas.PanRight, canvas.PanUp, canvas.PanDown = false, false, false, false
	}
	e.Pointer(canvas, time.Now())
	e.windowCursor()
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
		// a strip of equal height, it sits at scale 1 below the bar. Its
		// window is the map area's columns; mapOrigin adds the left panel.
		v.frameW, v.frameH = fw, fh
		v.place = frame.Fit(fw, fh, fw, h)
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
		e.applyPlayerTint()
		v.drawFrame(v.canvas)
		if f, ok := e.placement(); ok {
			e.drawPlacement(v.canvas, f)
		}
		e.drawMarkers(v.canvas)
		ox, oy := e.mapOrigin()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(v.place.Scale(), v.place.Scale())
		op.GeoM.Translate(ox, oy)
		screen.DrawImage(v.canvas, op)
	}
	if kept := e.drawChrome(screen); len(kept) > 0 {
		e.textOverlay.draw(screen, kept, 1, 0, 0)
	}
	e.drawMCP(screen)
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
	e.drawPassability(clip)
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
	tint := func(refused bool) color.RGBA {
		if refused {
			return editorWarning
		}
		return editorGold
	}
	f, placing := e.placement()
	if placing && !f.Dragged {
		// The footprints switch off leaves the anchor cell alone.
		cells := f.Cells
		if len(cells) == 0 || e.view.hideFootprints {
			cells = []placementCells{{Rect: image.Rectangle{Min: f.Cell, Max: f.Cell.Add(image.Pt(1, 1))}, Refused: f.Refused}}
		}
		for _, c := range cells {
			out = append(out, e.cellBoxLines(c.Rect, tint(c.Refused))...)
		}
	}
	for i, r := range e.doc.Records {
		member := e.inSelection(i)
		dragRefused := false
		if d := e.recordDrag; d != nil && d.moved && (i == d.record || d.group && member) {
			// preview only; no model mutation until release
			offset := d.cell.Sub(d.origin)
			r.Cell, r.Footprint = r.Cell.Add(offset), r.Footprint.Add(offset)
			dragRefused = placing && f.Dragged && f.RefusedRecords[RecordRef{Kind: r.Kind, Index: r.Index}]
		}
		issue := !e.view.hideIssues && (r.Warning != "" || e.report.flagged[findingKey(r.Kind, r.Index)])
		chosen := i == e.selected || member
		if !r.Spatial || (!issue && !chosen) {
			continue
		}
		wx, wy := e.cellCenter(r.Cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		c := editorWarning
		if chosen {
			c = editorGold
		}
		out = append(out, editorLine{float32(x - 5), float32(y), float32(x + 5), float32(y), c}, editorLine{float32(x), float32(y - 5), float32(x), float32(y + 5), c})
		if chosen && !e.view.hideFootprints {
			if dragRefused {
				c = editorWarning
			}
			out = append(out, e.cellBoxLines(r.footprint(), c)...)
		}
	}
	for _, cell := range e.reportCrosses() {
		wx, wy := e.cellCenter(cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		out = append(out, editorLine{float32(x - 5), float32(y), float32(x + 5), float32(y), editorGold}, editorLine{float32(x), float32(y - 5), float32(x), float32(y + 5), editorGold})
	}
	out = append(out, e.dropToolLines()...)
	for _, cell := range e.dropMarkers() {
		if e.view.hideDrops && !e.dropToolOn() {
			break
		}
		wx, wy := e.cellCenter(cell)
		x, y := v.cam.WorldToScreen(wx, wy)
		for _, d := range [][4]float32{{-8, -8, 8, -8}, {8, -8, 8, 8}, {8, 8, -8, 8}, {-8, 8, -8, -8}, {-8, -8, 8, 8}, {-8, 8, 8, -8}} {
			out = append(out, editorLine{float32(x) + d[0], float32(y) + d[1], float32(x) + d[2], float32(y) + d[3], editorCondition})
		}
	}
	out = append(out, e.borderLines()...)
	out = append(out, e.areaLines()...)
	out = append(out, e.wizardAreaLines()...)
	out = append(out, e.referenceMarkerLines()...)
	out = append(out, e.brushMarkerLines()...)
	out = append(out, e.groupMarkerLines()...)
	out = append(out, e.marqueeLines()...)
	return append(e.gridLines(), out...)
}

// cellBoxLines are the outlines, in c, of every map cell of r; a cell off the
// map is not drawn.
func (e *MapEditor) cellBoxLines(r image.Rectangle, c color.RGBA) []editorLine {
	v := e.doc.Viewer
	r = r.Intersect(image.Rect(0, 0, e.doc.Width, e.doc.Height))
	var out []editorLine
	for cy := r.Min.Y; cy < r.Max.Y; cy++ {
		for cx := r.Min.X; cx < r.Max.X; cx++ {
			verts := v.cellSpace().vertices(v.cam, cx, cy)
			for _, edge := range [][2]int{{0, 1}, {1, 3}, {3, 2}, {2, 0}} {
				a, b := verts[edge[0]], verts[edge[1]]
				out = append(out, editorLine{a.DstX, a.DstY, b.DstX, b.DstY, c})
			}
		}
	}
	return out
}

func (e *MapEditor) Run() error {
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetWindowTitle("Againrom Map Editor")
	ApplyWindowRow(EditorWindow)
	return ebiten.RunGame(e)
}
