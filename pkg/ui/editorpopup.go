package ui

import (
	"cmp"
	"image"
	"image/color"
	"image/draw"
	"slices"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/render/text"
)

// The editor's popup window: one builder takes a popup description as data
// (title, rows of typed controls, an optional list, notes and buttons) and
// produces one layout that both the painter and the hit test read, so a
// control is drawn where it is pressed.

// popupRowKind is the control a popup row carries.
type popupRowKind uint8

const (
	popupText    popupRowKind = iota // a label and a read-only value
	popupHeading                     // a section heading across the row
	popupEdit                        // a label and an edit field
	popupChoice                      // a label and a drop-down
	popupPick                        // a label, an edit field and a pick-on-map button
	popupLink                        // a label and a value that opens what it names
	popupSwitch                      // a label and a button that switches its value
	popupMultiline
	// popupCard heads a card: a grip with the card's number (its label), a
	// drop-down (its value, none when empty) and a small cross (its button);
	// the card holds the Span rows after it. A drag from the grip moves the
	// card.
	popupCard
	popupDirection
)

// blockRow is the heading of a block of rows that opens and closes. It
// takes the tree's fold state, RowFold: the painter draws the state's mark
// from rowFoldMarks before the label, which stays the block's word.
func blockRow(label string, open bool) popupRow {
	fold := FoldClosed
	if open {
		fold = FoldOpen
	}
	return popupRow{Kind: popupHeading, Label: label, Fold: fold}
}

// popupRow is one row of a popup.
type popupRow struct {
	Column int
	Edit   *editorTextView
	Kind   popupRowKind
	// Fold marks a heading that opens and closes the rows under it, as a
	// tree row's Fold does; FoldNone is a plain heading.
	Fold         RowFold
	Label, Value string
	Ink          color.RGBA // the value's ink; zero is the popup's paper
	Focus        bool       // the keyboard edits this row
	Caret        int        // the caret's pixel offset in a focused edit
	// Buttons are the captions of the row's own buttons, at its right end;
	// a pick row without them carries the popup's PickLabel. Disabled marks
	// the buttons drawn greyed that take no press.
	// Armed marks the buttons drawn pressed in: the one that armed a map pick.
	Buttons  []string
	Disabled []bool
	Armed    []bool
	// Lines is a multi-line row's height in rows, zero for six; First is
	// the first text line it shows while it is not typed in.
	Lines, First int
	// Hint is an empty edit row's grey text: the value it stands for.
	Hint string
	// Icons are pictures drawn after the value, left to right, each square
	// at the row's height.
	Icons []*image.RGBA
	// Portrait is a picture drawn in the label's column under the label,
	// fitted to the control's height.
	Portrait *image.RGBA
	// PortraitTip is the hover line the portrait shows, empty for none.
	PortraitTip string
	// Span is how many rows after a card row its card holds. Struck marks a
	// removed card: a strip with its label and buttons, no grip or value.
	Span   int
	Struck bool
	// Place, when set, is the box on the card, relative to the card's
	// corner, that the row stands at: it takes no line among the rows and
	// never scrolls, and its label is not drawn. A placed edit row draws
	// its field there; a placed line of text draws nothing, the card
	// states it. Tip is the hover line the row shows whole, cut or not.
	Place image.Rectangle
	// RestLeft counts Place's x from the left of the room beside the card
	// instead of the card's; a placed pad draws its label over its buttons.
	RestLeft  bool
	Tip       string
	Choices   []InspectionChoice
	ChoicePad []image.Point
	Selected  int
	// Help is what the row's (?) mark says on hover; empty draws no mark.
	Help string
	// Hold makes a text row of Lines rows' height that draws nothing of its
	// own a place for what it holds: the spec's card, or a window's figure.
	Hold popupHold
	// HeadOf, when set, is 1 + the grid row whose head box the row stands
	// in, at Place counted from the box's corner, instead of on the card: a
	// matrix row's own controls left of its cells. Its label is not drawn.
	HeadOf int
}

// popupHold is what a holding row keeps its place for.
type popupHold uint8

const (
	holdNone popupHold = iota
	// holdCard stands the spec's card at the row's corner, the room right
	// of it the card's CardRest, instead of beside, above or under the rows.
	holdCard
	// holdFigure keeps the room the window draws its figure in.
	holdFigure
)

// cardHeld reports that a row of s holds its card.
func cardHeld(s uiWindow) bool {
	return s.Card.Subject != nil && slices.ContainsFunc(s.Body.Rows, func(r popupRow) bool { return r.Hold == holdCard })
}

func (r popupRow) withChoices(choices []InspectionChoice, pad []image.Point, value string) popupRow {
	r.Kind = popupChoice
	if len(pad) == 0 || len(pad) != len(choices) {
		return r
	}
	r.Kind, r.Choices, r.ChoicePad, r.Selected = popupDirection, choices, pad, -1
	for i, c := range choices {
		if strconv.FormatInt(c.Value, 10) == strings.TrimSpace(value) {
			r.Selected = i
		}
	}
	return r
}

func drawDirectionMark(dst *image.RGBA, box image.Rectangle, direction image.Point, ink color.RGBA) {
	c := box.Min.Add(box.Size().Div(2))
	radius := max(2, min(box.Dx(), box.Dy())/5)
	point := func(p image.Point) {
		draw.Draw(dst, image.Rect(p.X-1, p.Y-1, p.X+1, p.Y+1).Intersect(box.Inset(3)), &image.Uniform{C: ink}, image.Point{}, draw.Src)
	}
	if direction == (image.Point{}) {
		for y := -2; y <= 2; y++ {
			for x := -2; x <= 2; x++ {
				if x*x+y*y <= 4 {
					point(c.Add(image.Pt(x, y)))
				}
			}
		}
		return
	}
	for k := -radius; k <= radius; k++ {
		point(c.Add(direction.Mul(k)))
	}
	tip, across := c.Add(direction.Mul(radius)), image.Pt(-direction.Y, direction.X)
	for k := 0; k <= radius; k++ {
		point(tip.Sub(direction.Mul(k)).Add(across.Mul(k)))
		point(tip.Sub(direction.Mul(k)).Sub(across.Mul(k)))
	}
}

// placed reports that the row stands on the card at its Place.
func (r popupRow) placed() bool { return !r.Place.Empty() }

// popupRowDrag is a card held by its grip: Row is its card row and At the
// pointer.
type popupRowDrag struct {
	Row int
	At  image.Point
}

// popupGrid is a grid of picture cells as data: one widget for every grid
// of pictures a popup shows. A cell is drawn pressed while it is the chosen
// one and carries a tick while it is marked. Rows is how many rows show,
// zero for every row the room holds (fitGrid), and Top the first row shown;
// the columns follow the popup's width unless Cols fixes them.
type popupGrid struct {
	Cells  []popupGridCell
	Groups []popupGridGroup
	Side   int // a cell's side in pixels; zero is gridCellSide
	Rows   int
	Top    int
	Cols   int
	// LeastRows is the fewest rows a grid of every row shows in a popup of
	// its own size; zero is gridLeastRows.
	LeastRows int
	// RoomTop measures the room for the rows from the room's top, not the
	// popup's: the bounds move a popup that grows past them back inside.
	RoomTop bool
	// RowHeads and ColHeads head a matrix: RowHeads[r] stands left of grid
	// row r in a column HeadW pixels wide, ColHeads[c] above column c in a
	// row one cell high. With HeadH set the column heads stand upright,
	// reading from bottom to top, in a row HeadH pixels high, and the hover
	// line over one shows it whole.
	RowHeads, ColHeads []string
	HeadW, HeadH       int
	// HeadFlat lays the column heads flat in their HeadH row, a head wider
	// than its column on two lines when the row holds two. RowHigh[r], where
	// set, is matrix row r's height in place of the cell side; such a grid
	// shows every row.
	HeadFlat bool
	RowHigh  []int
	// CaptionH is the band under each cell's picture that holds its Caption.
	CaptionH int
}

// least is the fewest rows the grid shows when the room is short.
func (g *popupGrid) least() int {
	if g.LeastRows > 0 {
		return g.LeastRows
	}
	return gridLeastRows
}

// rowHigh is the height of grid row r.
func (g *popupGrid) rowHigh(r int) int {
	if r >= 0 && r < len(g.RowHigh) && g.RowHigh[r] > 0 {
		return g.RowHigh[r]
	}
	return g.side() + g.CaptionH
}

// pictureSide is the side of the box a cell's picture is drawn into.
func (g *popupGrid) pictureSide() int { return g.side() - 10 }

// rowsHigh is the height of the n rows from row top on.
func (g *popupGrid) rowsHigh(top, n int) int {
	h := 0
	for r := top; r < top+n; r++ {
		h += g.rowHigh(r)
	}
	return h
}

type popupGridGroup struct {
	Block, Head string
	Cells       []int
}

type popupGridRow struct {
	Head  string
	Cells []int
}

// rows wraps the grid's cell indices, with group and block heads on full rows.
func (g *popupGrid) rows(cols int) []popupGridRow {
	var rows []popupGridRow
	if len(g.Groups) == 0 {
		for first := 0; first < len(g.Cells); first += cols {
			row := popupGridRow{}
			if r := first / cols; r < len(g.RowHeads) {
				row.Head = g.RowHeads[r]
			}
			for n := first; n < min(first+cols, len(g.Cells)); n++ {
				row.Cells = append(row.Cells, n)
			}
			rows = append(rows, row)
		}
		return rows
	}
	block := ""
	for _, group := range g.Groups {
		if len(group.Cells) == 0 {
			continue
		}
		if group.Block != "" && group.Block != block {
			rows = append(rows, popupGridRow{Head: group.Block})
			block = group.Block
		}
		if group.Head != "" {
			rows = append(rows, popupGridRow{Head: group.Head})
		}
		for first := 0; first < len(group.Cells); first += cols {
			rows = append(rows, popupGridRow{Cells: group.Cells[first:min(first+cols, len(group.Cells))]})
		}
	}
	return rows
}

// popupGridCell is one cell of a grid. Label is drawn on a cell without a
// picture; an Inert cell is drawn flat and takes no press. Tip names the
// cell in the hover tooltip, empty for none.
type popupGridCell struct {
	Icon  *image.RGBA
	Label string
	Tip   string
	// TipPicture is the picture the hover tip shows at full size beside Tip.
	TipPicture *image.RGBA
	// TipPictures are pictures the hover tip shows side by side, each at the
	// size of the cell's own picture: a ground's variations.
	TipPictures []*image.RGBA
	// Caption names the cell under its picture, in the grid's CaptionH band.
	Caption string
	Pressed bool
	Ticked  bool
	Inert   bool
}

// gridCellSide is a grid cell's side when the grid names none.
const gridCellSide = 44

func (g *popupGrid) side() int {
	if g.Side > 0 {
		return g.Side
	}
	return gridCellSide
}

// height is the grid's height in the popup, its gap below included.
func (g *popupGrid) height() int {
	if g == nil {
		return 0
	}
	return g.rowsHigh(0, max(1, g.Rows)) + g.headHeight() + popupPad
}

// headHeight is the column heads' row height: HeadH for upright heads, one
// cell for flat ones, 0 when the columns carry none.
func (g *popupGrid) headHeight() int {
	switch {
	case len(g.ColHeads) == 0:
		return 0
	case g.HeadH > 0:
		return g.HeadH
	}
	return g.side()
}

// columns is how many cells a row of a grid width wide holds, its scroll bar
// aside.
func (g *popupGrid) columns(width int) int {
	if g.Cols > 0 {
		return g.Cols
	}
	return max(1, (width-widgetSpriteSize-2)/g.side())
}

// totalRows is how many rows the grid's cells fill at cols columns.
func (g *popupGrid) totalRows(cols int) int {
	if len(g.Groups) > 0 {
		return len(g.rows(cols))
	}
	return (len(g.Cells) + cols - 1) / cols
}

// gridLeastRows is the least rows a grid of every row shows in a popup of
// its own size, however little room is left: the rows the grids showed
// before they fitted the room.
const gridLeastRows = 3

// gridRoom is the room popup s's grid of every row grows into.
func gridRoom(s uiWindow) image.Rectangle {
	if s.Place.Room.Empty() {
		return s.Place.Bounds
	}
	return s.Place.Room
}

// gridLeastCols is the columns a grid of every row shows at least, where the
// room allows.
const gridLeastCols = 6

// fixedGridWidth is the width a grid with a fixed column count needs to show
// every column; zero for any other popup. A sized popup is held to it, since
// a column outside the window takes no press.
func fixedGridWidth(s uiWindow) int {
	g := s.Grid
	if g == nil || g.Cols <= 0 {
		return 0
	}
	return 2*popupPad + g.HeadW + g.Cols*g.side()
}

// gridWidth is the width popup s takes, w its own, widened only so that a
// grid of every row whose columns follow the width shows gridLeastCols
// columns, up to the room's width (the popup then stands left of its corner,
// inside the bounds). The grid scrolls the rest.
func gridWidth(s uiWindow, w int) int {
	g, room := s.Grid, gridRoom(s)
	if g == nil || g.Rows > 0 || g.Cols > 0 || room.Empty() {
		return w
	}
	need := 2*popupPad + g.HeadW + min(len(g.Cells), gridLeastCols)*g.side() + widgetSpriteSize + 2
	return max(w, min(need, room.Dx()))
}

// fitGrid is popup s's grid with the rows it shows at outer width w, the
// rest of the popup rest pixels high. A grid of every row shows the rows its
// cells fill, no more than fit a sized popup's height sizedH, or else the
// room below the popup's top, never fewer than gridLeastRows there; the
// grid scrolls the rest.
func fitGrid(s uiWindow, w, rest, sizedH int, sized bool) *popupGrid {
	g := s.Grid
	if g == nil || g.Rows > 0 {
		return g
	}
	total := g.totalRows(g.columns(w - 2*popupPad - g.HeadW))
	limit, room := total, gridRoom(s)
	free := func(h int) int { return (h - rest - g.headHeight() - popupPad) / g.rowHigh(0) }
	switch {
	case sized:
		limit = free(sizedH)
	case !room.Empty():
		top := max(s.Place.At.Y, room.Min.Y)
		if g.RoomTop {
			top = room.Min.Y
		}
		limit = max(g.least(), free(room.Max.Y-top))
	}
	fit := *g
	fit.Rows = max(1, min(total, limit))
	return &fit
}

// popupSpec is a popup as data. At is its top-left corner in window pixels,
// Width its width; the height follows the content and Bounds clamps both.
// Top is the first row shown and MaxRows how many rows show at most; List,
// when set, is a list box over that model below the rows.
type popupSpec struct {
	CentreSize  image.Point
	ColumnTops  []int
	ButtonFocus *int
	Columns     int
	ColumnWidth int
	Tabs        []string
	ActiveTab   int
	// TabSwitch draws the active tab pressed, a switch's chosen state,
	// instead of focused.
	TabSwitch bool
	Card      *PanelSubject
	CardFont  *text.Font
	CardAside int
	// CardScale draws the card that many times its size, 0 and 1 its own;
	// CardBelow stands the card under the rows, across the row width.
	CardScale int
	CardBelow bool
	// BandLines, set with CardBelow on one column, makes the band under the
	// rows that many lines that scroll after the last row: Top and MaxRows
	// count them, and the card and the rest beside it show where they fall
	// inside the rows' viewport.
	BandLines int
	Title     string
	// TitleHelp is what the title's (?) mark says on hover: what the window
	// or its page is for. Empty draws no mark.
	TitleHelp string
	// CloseLabel is the title row button's caption; empty is the close
	// button's x.
	CloseLabel string
	// SwitchButton is the caption of a button the button row holds at its
	// left end, apart from Buttons and as wide as its caption; empty is
	// none.
	SwitchButton string
	At           image.Point
	Width        int
	Bounds       image.Rectangle
	// Room is what a grid of every row (Rows 0) grows into while the popup
	// keeps its own size: the width that shows its least columns, then the
	// rows down to Room's bottom. Empty is Bounds.
	Room image.Rectangle
	// Slack lets a dragged popup hang past the bounds, keeping its title row
	// and popupKeepWide pixels of its width inside.
	Slack bool
	// Short marks a popup the player shortened from a place past the bottom
	// edge: it fits the height it was given down to the title row, one row
	// and the button row.
	Short   bool
	Rows    []popupRow
	Top     int
	MaxRows int
	// Page is the first row shown when the layout pass pages a window
	// that sets no MaxRows because it is taller than its bounds.
	Page     int
	List     *Picker
	ListRows int
	// ListLeast is the fewest list rows a sized popup keeps; zero is
	// popupMinRows.
	ListLeast int
	ListText  listRowText
	// ListAside is a column right of the list kept free for a picture.
	ListAside int
	// ListColumns parts the list into columns (listBox's Columns), each
	// head over its column, wrapped to its width.
	ListColumns []listColumn
	Notes       []string
	NoteInk     color.RGBA
	Buttons     []string
	Disabled    []bool
	// ButtonW is the button row's least button width; zero is the default.
	// A caption that needs more widens its button (popupButtonWidths).
	ButtonW int
	// PickLabel is the pick-on-map button's caption.
	PickLabel string
	PickArmed bool
	// LabelWidth is the label column; zero is two fifths of the width.
	LabelWidth int
	// Size is the outer size the user dragged the popup to; zero keeps the
	// size the content gives. The popup then spans that size, no smaller than
	// popupMinSize: a list shows as many rows as fit, a row-scrolled popup
	// shows as many rows, a multi-line field takes the free height and the
	// buttons stay on the bottom edge.
	Size image.Point
	// MinWidth caps the least width a sized popup is held to: a popup
	// whose own width was narrower than popupMinWidth keeps that width as
	// its least.
	MinWidth int
	// Grid, when set, stands between the rows and the list.
	Grid *popupGrid
	// Canvas, when set, is a drawing between the grid and the list.
	Canvas *popupCanvas
	// Drag is the card a row drag holds, nil for none.
	Drag *popupRowDrag
	// Fit makes the popup fit its content to its width, as a window in a
	// side panel does (popupFlowOf): what the width cannot hold flows onto
	// more lines instead of past the edge or over other text.
	Fit bool
}

// popupRowRect is the geometry of one shown row.
type popupRowRect struct {
	Index                int
	Rect, Label, Control image.Rectangle
	Button               image.Rectangle   // the first row button; empty otherwise
	Buttons              []image.Rectangle // every row button, left to right
	Off                  []bool            // the row's Disabled buttons
	// Grip is a card row's drag handle and Card the card's whole box.
	Grip, Card image.Rectangle
	// Help is the row's (?) mark; empty for a row without Help.
	Help image.Rectangle
}

// popupLayout is a built popup: every rectangle the painter fills and the
// hit test reads, and Window, the window as the pass laid it out.
type popupLayout struct {
	Window             *uiWindow
	Tabs               []image.Rectangle
	RowBar             image.Rectangle
	Rect, Title, Close image.Rectangle
	// TitleHelp is the title's (?) mark, left of Close; empty without one.
	TitleHelp image.Rectangle
	// SwitchButton is the button row's SwitchButton, empty for none.
	SwitchButton image.Rectangle
	Rows         []popupRowRect
	RowsViewport image.Rectangle
	List         listBox
	HasList      bool
	Aside        image.Rectangle // the column ListAside keeps
	// ListHeads are the list columns' head boxes and ListHeadLines their
	// lines; none without ListColumns.
	ListHeads     []image.Rectangle
	ListHeadLines [][]string
	Notes         []image.Rectangle
	Buttons       []image.Rectangle
	Pitch         int
	// ListRows and RowsFit are the list rows and the scrolled rows the
	// layout shows: the spec's own, or the ones a sized popup fits.
	ListRows, RowsFit int
	// Grid is the grid's box, GridBar its scroll bar, GridCells the shown
	// cells' boxes from cell GridFirst on, GridCols its columns, GridTop its
	// first shown row and GridRows the rows shown.
	Grid, GridBar                          image.Rectangle
	GridCells                              []image.Rectangle
	GridFirst, GridCols, GridTop, GridRows int
	// GridRowHeads and GridColHeads are the shown headings' boxes, from
	// row GridTop and column 0 on; GridInert marks the shown cells that
	// take no press.
	GridRowHeads, GridColHeads []image.Rectangle
	GridInert                  []bool
	GridIndices                []int
	GridHeadText               []string
	// Canvas is the canvas's view, CanvasBar its scroll bar, CanvasAt the
	// sheet point at the view's corner, CanvasBoxes each box's place in the
	// window, a row under the box's number that the draw reads and the hit
	// test asks, CanvasInert the boxes that take no press, and CanvasLegend
	// each legend word's place, its swatch first.
	Canvas, CanvasBar image.Rectangle
	CanvasAt          image.Point
	CanvasBoxes       controlLayout
	CanvasInert       []bool
	CanvasLegend      []image.Rectangle
	// Drop is the card, counted among the spec's card rows, that a held card
	// lands before, the card count for after the last, -1 for no drag;
	// Marker is the bar drawn at that gap while the drop moves the card.
	Drop   int
	Marker image.Rectangle
	// Card is the statistics card's box and CardRest the room the card's
	// picture takes: below the card in its column beside the rows, or left
	// of it when the card stands above the rows. Both are empty without a
	// card.
	Card, CardRest image.Rectangle
}

const (
	popupPad      = 6
	popupRowExtra = 8
	popupButtonW  = 96
	// popupCardColumn is the least width a fitted popup's rows keep beside
	// its card; below it the card stands above the rows.
	popupCardColumn = 300
	// popupFitField is the least width a fitted row's field keeps beside
	// the row's own buttons; below it the buttons take a line of their own.
	popupFitField = 2 * popupMinField
	// popupTipW is the least width of the box a row's own Tip reads in.
	popupTipW = 300
	// popupHelpW is the width of the box a (?) mark's help reads in,
	// wrapped to as many lines as it takes.
	popupHelpW = 380
)

// tabStrip is one strip of tabs: the captions, the tab shown and whether the
// strip switches (the tab shown stays pressed) or only names where the
// keyboard is. PerLine tabs share a line's width; 0: all of them.
type tabStrip struct {
	Captions []string
	Active   int
	Switch   bool
	PerLine  int
}

// rects is the strip's tabs from origin over width, each line pitch high.
func (t tabStrip) rects(origin image.Point, width, pitch int) []image.Rectangle {
	per := t.PerLine
	if per <= 0 {
		per = len(t.Captions)
	}
	var out []image.Rectangle
	for i := range t.Captions {
		left := origin.X + i%per*width/per
		right := origin.X + (i%per+1)*width/per
		top := origin.Y + i/per*pitch
		out = append(out, image.Rect(left, top, right, top+pitch))
	}
	return out
}

// draw draws the strip's tabs in rects and, where closeAt has area, the close
// button the strip ends in. part and at are what the pointer is over, as the
// layout's hit reports it: that tab, or the close button, is lit.
func (t tabStrip) draw(dst *image.RGBA, body *text.Font, rects []image.Rectangle, closeAt image.Rectangle, held *buttonPress, part popupPart, at int) {
	for i, tab := range rects {
		chosen := t.Switch && t.Active == i
		drawPushButton(dst, body, pushButton{Press: held, Rect: tab, Label: t.Captions[i], Literal: true, LabelRaise: labelRaise(body), Focus: t.Active == i && !t.Switch, Pressed: chosen, Inside: chosen, Hover: part == popupTab && at == i})
	}
	if !closeAt.Empty() {
		drawPushButton(dst, body, pushButton{Press: held, Rect: closeAt, Label: "x", Literal: true, LabelRaise: labelRaise(body), Hover: part == popupClose})
	}
}

// popupFlow is how a popup lays out what its width cannot hold: the tabs on
// each line and their lines, each button's line of the button row and the
// lines, each row's lines added for its label and its own buttons, the label
// column (0 for the default) and whether the card stands above the rows. A
// one-column popup without Fit keeps one line of each but the button row's.
type popupFlow struct {
	tabsPerLine, tabLines int
	buttonLine            []int
	buttonLines           int
	rowExtra              []int
	labelOver             []bool
	labelW                int
	// columnLabelW is each column's label column in a popup of several.
	columnLabelW []int
	cardTop      bool
	// cardH is the card's drawn height; cardBelow stands its band under
	// the rows instead of above them.
	cardH     int
	cardBelow bool
	// bandScrolls counts the band under the rows among the scrolled lines.
	bandScrolls bool
	// listHeads are the list columns' head lines and listHeadH the band
	// they take over the list.
	listHeads [][]string
	listHeadH int
}

// listColumn is one column of a list: its head and its share of the width.
type listColumn struct {
	Head  string
	Share int
}

// listShares are the shares of columns.
func listShares(columns []listColumn) []int {
	out := make([]int, len(columns))
	for i, c := range columns {
		out[i] = c.Share
	}
	return out
}

// listHeadLines are the heads of s's list columns, each wrapped to its
// column at popup width w, and the height of their band.
func listHeadLines(s uiWindow, body *text.Font, w int) ([][]string, int) {
	if s.List.Picker == nil || len(s.List.Columns) == 0 {
		return nil, 0
	}
	stops := columnStops(listShares(s.List.Columns), w-2*popupPad-widgetSpriteSize-s.List.Aside-2*listTextX)
	heads := make([][]string, len(s.List.Columns))
	lines := 1
	for k, c := range s.List.Columns {
		heads[k] = editorTextLines(body, c.Head, max(1, stops[k+1]-stops[k]-listColumnGap))
		lines = max(lines, len(heads[k]))
	}
	return heads, lines*fontHeight(body, 15) + popupPad
}

// cardSize is the card's drawn size: the character panel's own, times the
// spec's CardScale.
func cardSize(s uiWindow) image.Point {
	return image.Pt(compactPanelW, compactPanelH).Mul(max(1, s.Card.Scale))
}

// cardAside is the width of the card's column beside the rows, 0 without a
// card.
func cardAside(s uiWindow) int {
	if s.Card.Subject == nil || cardHeld(s) {
		return 0
	}
	return max(cardSize(s).X, s.Card.Aside)
}

// cardOnTop reports that a fitted popup width wide stands its card, aside
// pixels wide in its column, above the rows: the rows beside the card would
// keep less than popupCardColumn.
func cardOnTop(width, aside int) bool {
	return width-2*popupPad-aside-popupPad-widgetSpriteSize-2 < popupCardColumn
}

// rowButtonWidths are the widths of row's own buttons: the default button
// width, a caption's own width for a row that names its buttons, or a small
// square for a card's cross.
func rowButtonWidths(s uiWindow, row popupRow, body *text.Font, pitch int) []int {
	caps := rowButtons(s, row)
	ws := make([]int, len(caps))
	for k, c := range caps {
		ws[k] = popupButtonW
		if len(row.Buttons) > 0 && body != nil {
			ws[k] = max(48, body.Advance(c)+16)
		}
		if row.Kind == popupCard && !row.Struck {
			// A card's cross is a square; a caption that does not fit inside
			// the square takes its own width.
			ws[k] = pitch - 2
			if body != nil && body.Advance(c)+4 > ws[k] {
				ws[k] = body.Advance(c) + 16
			}
		}
	}
	return ws
}

// flowLines puts items of widths ws, gap pixels apart, on lines room wide,
// in order: each item's line, from first, and the count of lines.
func flowLines(ws []int, gap, room, first int) ([]int, int) {
	lines := make([]int, len(ws))
	line, used := first, 0
	for i, w := range ws {
		need := w
		if used > 0 {
			need += gap
		}
		if used > 0 && used+need > room {
			line, used = line+1, w
		} else {
			used += need
		}
		lines[i] = line
	}
	if len(ws) == 0 {
		return lines, 0
	}
	return lines, line - first + 1
}

// rowHasControl reports that row draws a field, a drop-down or a switch
// right of its label.
func rowHasControl(row popupRow) bool {
	switch row.Kind {
	case popupHeading, popupCard:
		return false
	case popupText, popupLink:
		return row.Value != ""
	}
	return true
}

// popupFlowOf is popup s's flow at outer width w. Every popup's button row
// takes as many lines as keep every button inside. With Fit set, and in
// every popup of several columns for its rows:
//   - the label column widens to the longest label, up to half the row, and
//     a set label column wider than two fifths of a narrow row gives way;
//     in a popup of several columns each column has its own;
//   - a label longer than its column takes a line of its own over its field;
//   - a row whose own buttons do not fit beside its label and a field of
//     popupFitField puts them on lines of their own under it.
//
// The tabs of every popup take as many lines as keep every caption whole.
// With Fit set the card stands above the rows when cardOnTop says so.
func popupFlowOf(s uiWindow, body *text.Font, w, pitch int) popupFlow {
	f := popupFlow{tabsPerLine: len(s.Tabs.Captions), rowExtra: make([]int, len(s.Body.Rows)), labelOver: make([]bool, len(s.Body.Rows))}
	if len(s.Tabs.Captions) > 0 {
		f.tabLines = 1
	}
	ws := popupButtonWidths(s, body)
	f.buttonLine = make([]int, len(ws))
	if len(ws) > 0 || s.Buttons.Switch != "" {
		f.buttonLines = 1
	}
	f.cardH, f.cardBelow = cardSize(s).Y, s.Card.Subject != nil && s.Card.Below
	f.cardTop, f.bandScrolls = f.cardBelow, bandLines(s) > 0
	f.listHeads, f.listHeadH = listHeadLines(s, body, w)
	inner := w - 2*popupPad
	// The switch button leads the first line, left of the buttons.
	if sw := switchButtonWidth(s, body); sw > 0 {
		lines, n := flowLines(append([]int{sw}, ws...), popupPad, inner, 0)
		f.buttonLine, f.buttonLines = lines[1:], n
	} else if len(ws) > 0 {
		f.buttonLine, f.buttonLines = flowLines(ws, popupPad, inner, 0)
	}
	// The tabs take as many lines as keep every caption whole.
	if body != nil {
		for f.tabsPerLine > 1 {
			share, whole := inner/f.tabsPerLine, true
			for _, t := range s.Tabs.Captions {
				whole = whole && body.Advance(t)+16 <= share
			}
			if whole {
				break
			}
			f.tabsPerLine--
		}
	}
	if f.tabsPerLine > 0 {
		f.tabLines = (len(s.Tabs.Captions) + f.tabsPerLine - 1) / f.tabsPerLine
	}
	columns := max(1, s.Body.Columns)
	if !s.Place.Fit && columns == 1 {
		return f
	}
	aside := cardAside(s)
	if s.Place.Fit {
		f.cardTop = s.Card.Subject != nil && (s.Card.Below || cardOnTop(w, aside))
	}
	rowW := inner
	if s.Body.MaxRows > 0 && (columns == 1 || popupRowCount(s) > s.Body.MaxRows) {
		rowW -= widgetSpriteSize + 2
	}
	if s.Card.Subject != nil && !f.cardTop {
		rowW -= aside + popupPad
	}
	rowW = popupColumnWidth(s, rowW)
	base := s.Body.LabelWidth
	if base <= 0 {
		base = rowW * 2 / 5
	}
	need := make([]int, columns)
	if body != nil {
		for i, row := range s.Body.Rows {
			if rowHasControl(row) && !row.placed() {
				c, _ := popupRowPosition(s, i)
				need[c] = max(need[c], body.Advance(row.Label)+8+helpMarkW(row, pitch))
			}
		}
	}
	f.columnLabelW = make([]int, columns)
	for c := range f.columnLabelW {
		f.columnLabelW[c] = max(min(base, rowW*2/5), min(need[c], rowW/2))
	}
	f.labelW = f.columnLabelW[0]
	for i, row := range s.Body.Rows {
		if row.Kind == popupHeading || row.Kind == popupCard || row.placed() {
			continue
		}
		c, _ := popupRowPosition(s, i)
		labelW := f.columnLabelW[c]
		if rowHasControl(row) && row.Label != "" && body != nil && body.Advance(row.Label)+8+helpMarkW(row, pitch) > labelW {
			f.labelOver[i], f.rowExtra[i] = true, 1
		}
		bws := rowButtonWidths(s, row, body, pitch)
		if len(bws) == 0 {
			continue
		}
		left := 0
		switch {
		case f.labelOver[i]:
			left = popupFitField
		case rowHasControl(row):
			left = labelW + 4 + popupFitField
		case row.Label != "" && body != nil:
			left = body.Advance(row.Label) + 8
		}
		beside := 0
		for _, bw := range bws {
			beside += bw + 4
		}
		if left+beside > rowW {
			_, n := flowLines(bws, 4, rowW, 1)
			f.rowExtra[i] += n
		}
	}
	return f
}

// helpMarkW is the width a row's (?) mark takes from the end of its label.
func helpMarkW(row popupRow, pitch int) int {
	if row.Help == "" || row.placed() {
		return 0
	}
	return pitch
}

// cardBand is the height the card takes above or under the rows, 0 while it
// stands beside them, scrolls among them, or there is none.
func (f popupFlow) cardBand() int {
	if !f.cardTop || f.bandScrolls {
		return 0
	}
	return f.cardH + popupPad
}

// rowHeight is row i's height: a multi-line field's lines, and the lines its
// own buttons take under it; a row placed on the card takes none.
func (f popupFlow) rowHeight(row popupRow, i, pitch int) int {
	if row.placed() {
		return 0
	}
	h := pitch
	if row.Kind == popupMultiline || pictureRow(row) {
		h = multilineRows(row) * pitch
	}
	if row.Kind == popupDirection {
		h = 3 * pitch
	}
	if i < len(f.rowExtra) {
		h += f.rowExtra[i] * pitch
	}
	return h
}

// pictureRow is a read-only row that holds a picture in its label column, or
// holds a place, and takes Lines rows of height for it.
func pictureRow(row popupRow) bool {
	return row.Kind == popupText && (row.Portrait != nil || row.Hold != holdNone) && row.Lines > 0
}

// multilineRows is a multi-line row's height in rows.
func multilineRows(row popupRow) int {
	if row.Lines > 0 {
		return row.Lines
	}
	return 6
}

func fontHeight(f *text.Font, fallback int) int {
	if f != nil && f.Height() > 0 {
		return f.Height()
	}
	return fallback
}

// popupMinField is the narrowest a row's field is kept in a sized popup.
const popupMinField = 60

// popupMinRows is the fewest list or scrolled rows a sized popup keeps.
const popupMinRows = 3

// popupMinWidth is the narrowest a sized popup gets: the width where its
// buttons and each row's own buttons still fit beside a label and a field.
func popupMinWidth(s uiWindow, body *text.Font) int {
	w := max(240, buttonRowWidth(s, body))
	for _, row := range s.Body.Rows {
		caps := rowButtons(s, row)
		if len(caps) == 0 {
			continue
		}
		need := 0
		for _, c := range caps {
			bw := popupButtonW
			if len(row.Buttons) > 0 && body != nil {
				bw = max(48, body.Advance(c)+16)
			}
			need += bw + 4
		}
		// A label, a field of at least popupMinField and the buttons.
		if s.Body.LabelWidth > 0 {
			w = max(w, s.Body.LabelWidth+4+popupMinField+need+2*popupPad)
		} else {
			w = max(w, (4+popupMinField+need)*5/3+2*popupPad)
		}
	}
	switch {
	case cardHeld(s):
		// The card stands at its row, inside the row's column.
	case s.Card.Subject != nil && s.Card.Below:
		// The band under the rows holds the card and, left of it, the
		// column CardAside asks for; a band that scrolls keeps clear of
		// the scroll bar.
		bar := 0
		if bandLines(s) > 0 {
			bar = widgetSpriteSize + 2
		}
		w = max(w, cardSize(s).X+2*popupPad+bar)
		if s.Card.Aside > 0 {
			w = max(w, cardSize(s).X+s.Card.Aside+3*popupPad+bar)
		}
	case s.Card.Subject != nil:
		w += cardAside(s) + popupPad
	}
	return w
}

func popupRowsHeight(s uiWindow, flow popupFlow, pitch, shown int) int {
	heights := make([]int, max(1, s.Body.Columns))
	for _, i := range popupShownRows(s, shown) {
		column, _ := popupRowPosition(s, i)
		heights[column] += flow.rowHeight(s.Body.Rows[i], i, pitch)
	}
	return slices.Max(heights)
}

func popupColumnWidth(s uiWindow, room int) int {
	if s.Body.Columns > 1 {
		room = room/s.Body.Columns - popupPad
	}
	if s.Body.ColumnWidth > 0 {
		room = min(room, s.Body.ColumnWidth)
	}
	return room
}

// popupFit is a sized popup's flexible part for outer height h: the list rows,
// the scrolled rows and the extra lines of the first multi-line field, and the
// least height that keeps them usable. s.Body.Rows is not modified.
func popupFit(s uiWindow, body *text.Font, flow popupFlow, pitch, titleH, h int) (listRows, rowsFit, extraLines, minH int) {
	fixed := titleH + popupPad + len(s.Notes.Lines)*pitch + popupPad + s.Grid.height() + s.Canvas.least() + flow.cardBand()
	fixed += flow.buttonLines * (pitch + 4 + popupPad)
	// A card beside the rows stays clear of the note and button lines.
	cardLeast := 0
	if s.Card.Subject != nil && !flow.cardTop {
		cardLeast = titleH + popupPad + flow.cardH + popupPad + len(s.Notes.Lines)*pitch + flow.buttonLines*(pitch+4+popupPad)
	}
	minRows := popupMinRows
	if s.Place.Short {
		minRows = 1
	}
	if slices.ContainsFunc(s.Body.Rows, func(r popupRow) bool { return r.Kind == popupDirection }) {
		minRows = 1
	}
	if s.Body.Columns > 1 && s.Body.MaxRows > 0 {
		if minRows == 1 {
			minH = max(fixed+popupRowsHeight(s, flow, pitch, 1), cardLeast)
			rowsFit = min(popupRowCount(s)-s.Body.Top, max(1, (max(h, minH)-fixed)/pitch))
			for rowsFit > 1 && fixed+popupRowsHeight(s, flow, pitch, rowsFit) > max(h, minH) {
				rowsFit--
			}
			return 0, rowsFit, 0, minH
		}
		minH = max(fixed+popupMinRows*pitch, cardLeast)
		rowsFit = min(popupRowCount(s)-s.Body.Top, max(popupMinRows, (h-fixed)/pitch))
		// A row taller than the pitch shows fewer rows.
		for rowsFit > popupMinRows && fixed+popupRowsHeight(s, flow, pitch, rowsFit) > max(h, minH) {
			rowsFit--
		}
		return 0, rowsFit, 0, minH
	}
	top := max(0, min(s.Body.Top, len(s.Body.Rows)))
	rowH := func(i int) int {
		return flow.rowHeight(s.Body.Rows[i], i, pitch)
	}
	flexRows := s.Body.MaxRows > 0
	if !flexRows {
		for i := top; i < len(s.Body.Rows); i++ {
			fixed += rowH(i)
		}
		fixed += bandLines(s) * pitch
	}
	lp, listBase := 0, 0
	if s.List.Picker != nil {
		listBase = newListBox(image.Rect(0, 0, 10, 0), 0, body).Rect.Dy()
		lp = newListBox(image.Rect(0, 0, 10, 0), 1, body).Rect.Dy() - listBase
		fixed += listBase + popupPad + flow.listHeadH
		// Paged rows above a list stand at their count; the list takes
		// the rest.
		for i, n := top, 0; flexRows && i < len(s.Body.Rows) && n < s.Body.MaxRows; i++ {
			if !s.Body.Rows[i].placed() {
				fixed += rowH(i)
				n++
			}
		}
	}
	// The least height holds the flexible part at its least rows.
	least, listLeast := fixed, popupMinRows
	if s.List.Least > 0 {
		listLeast = s.List.Least
	}
	if s.Place.Short {
		listLeast = 1
	}
	if s.List.Picker != nil {
		least += listLeast * lp
	}
	if flexRows && s.List.Picker == nil {
		for i := top; i < len(s.Body.Rows) && i < top+minRows; i++ {
			least += rowH(i)
		}
	}
	minH = max(least, cardLeast)
	h = max(h, minH)
	avail := h - fixed
	switch {
	case s.List.Picker != nil:
		listRows = max(listLeast, avail/lp)
		if flexRows {
			rowsFit = max(0, min(s.Body.MaxRows, popupRowCount(s)-top))
		}
	case flexRows:
		used, full := 0, false
		for i := top; i < len(s.Body.Rows); i++ {
			if s.Body.Rows[i].placed() {
				continue
			}
			if used+rowH(i) > avail && rowsFit >= minRows {
				full = true
				break
			}
			used += rowH(i)
			rowsFit++
		}
		if !full {
			// A scrolling band's lines fill the room the rows leave.
			rowsFit += max(0, min(bandLines(s), (avail-used)/pitch))
		}
		rowsFit = min(rowsFit, popupRowCount(s)-top)
	default:
		extraLines = max(0, avail/pitch)
	}
	return listRows, rowsFit, extraLines, minH
}

// layoutPopup lays out the window a popupSpec describes, through its tree.
func layoutPopup(s popupSpec, title, body *text.Font) popupLayout {
	return layoutWindow(specTree(s), title, body)
}

// layoutWindow is the one layout pass of an editor window: it reads the
// window's tree and builds its geometry. A window taller than its bounds
// gives way in order until it fits: the body pages, the list shows fewer
// rows down to its least, the notes fold behind the title's (?) mark. The
// layout keeps the window as laid out for the painter and the hit tests.
func layoutWindow(root uiNode, title, body *text.Font) popupLayout {
	s := windowOf(root)
	l := placeWindow(s, title, body)
	room := s.Place.Bounds.Dy()
	if s.Place.Short && s.Place.Size.Y > 0 {
		room = min(room, s.Place.Size.Y)
	}
	if !s.Place.Bounds.Empty() && l.Rect.Dy() > room {
		s, l = giveWay(s, l, room, title, body)
	}
	l.Window = &s
	return l
}

// giveWay takes the steps of layoutWindow's order on a window taller than
// room, each only while the window is still too tall.
func giveWay(s uiWindow, l popupLayout, room int, title, body *text.Font) (uiWindow, popupLayout) {
	// The body pages: a window that sets no MaxRows from its Page, one that
	// does with fewer rows; a sized window's own fit sets its rows.
	count, top := popupRowCount(s), s.Body.Page
	most := count - 1
	if s.Body.MaxRows > 0 {
		most, top = min(most, s.Body.MaxRows-1), s.Body.Top
	}
	sized := popupResizeEnabled && s.Place.Size.X > 0 && s.Place.Size.Y > 0
	fewest := popupMinRows
	if s.Place.Short {
		fewest = 1
	}
	if len(s.Body.ColumnTops) == 0 && most >= fewest && (!sized || s.Body.MaxRows == 0) {
		page := func(n int) uiWindow {
			t := s
			t.Body.MaxRows, t.Body.Top = n, max(0, min(top, count-n))
			return t
		}
		// The most rows that fit, fewest the least.
		n := sort.Search(most-fewest+1, func(i int) bool {
			return placeWindow(page(fewest+i), title, body).Rect.Dy() > room
		})
		s = page(max(fewest, fewest+n-1))
		if sized {
			// A sized window shows the rows its height holds; the page
			// keeps exactly those.
			if fit := placeWindow(s, title, body).RowsFit; fit >= fewest && fit < s.Body.MaxRows {
				s = page(fit)
			}
		}
		if l = placeWindow(s, title, body); l.Rect.Dy() <= room {
			return s, l
		}
	}
	least := cmp.Or(s.List.Least, popupMinRows)
	if s.Place.Short {
		least = 1
	}
	if s.List.Picker != nil && s.List.Rows > least {
		step := max(1, newListBox(image.Rect(0, 0, 10, 0), 1, body).Rect.Dy()-newListBox(image.Rect(0, 0, 10, 0), 0, body).Rect.Dy())
		s.List.Rows = max(least, s.List.Rows-(l.Rect.Dy()-room+step-1)/step)
		if l = placeWindow(s, title, body); l.Rect.Dy() <= room {
			return s, l
		}
	}
	if len(s.Notes.Lines) > 0 {
		s.Title.Help = strings.TrimSpace(s.Title.Help + " " + strings.Join(s.Notes.Lines, " "))
		s.Notes.Lines = nil
		l = placeWindow(s, title, body)
	}
	return s, l
}

// window is the window l was laid out from; a layout built without the
// pass reads it from spec.
func (l popupLayout) window(spec popupSpec) uiWindow {
	if l.Window != nil {
		return *l.Window
	}
	return specWindow(spec)
}

// placeWindow measures, arranges and places one window: title bar with its
// close button, the shown rows, the list box, the note lines and the button
// row. A window with a Size takes that size (no smaller than its content
// allows) and fits its flexible rows to it.
func placeWindow(s uiWindow, title, body *text.Font) popupLayout {
	// A card a row holds takes no place of its own: the rows are laid out as
	// if there were none, then the card stands at its row.
	subject := s.Card.Subject
	if cardHeld(s) {
		s.Card.Subject = nil
	}
	pitch := fontHeight(body, 15) + popupRowExtra
	titleH := fontHeight(title, 15) + popupRowExtra
	w := max(160, s.Place.Width)
	if s.Body.MaxRows > 0 && popupRowCount(s) > s.Body.MaxRows {
		w += widgetSpriteSize + 2
	}
	sized := popupResizeEnabled && s.Place.Size.X > 0 && s.Place.Size.Y > 0
	if !sized {
		// A popup at its own size is as wide as its button row; the row
		// wraps only where the bounds hold less.
		w = max(gridWidth(s, w), buttonRowWidth(s, body))
	}
	if sized {
		least := popupMinWidth(s, body)
		if s.Place.MinWidth > 0 {
			least = min(least, s.Place.MinWidth)
		}
		w = max(s.Place.Size.X, least, fixedGridWidth(s))
	}
	if !s.Place.Bounds.Empty() {
		w = min(w, s.Place.Bounds.Dx())
	}
	flow := popupFlowOf(s, body, w, pitch)
	titleH += flow.tabLines * pitch
	sizedH := 0
	if sized {
		lr, rf, extra, minH := popupFit(s, body, flow, pitch, titleH, s.Place.Size.Y)
		sizedH = max(s.Place.Size.Y, minH)
		if !s.Place.Bounds.Empty() {
			sizedH = min(sizedH, max(minH, s.Place.Bounds.Dy()))
		}
		lr, rf, extra, _ = popupFit(s, body, flow, pitch, titleH, sizedH)
		if s.List.Picker != nil {
			s.List.Rows = lr
		}
		if s.Body.MaxRows > 0 {
			s.Body.MaxRows = rf
		}
		if extra > 0 {
			s.Body.Rows = slices.Clone(s.Body.Rows)
			for i := max(0, s.Body.Top); i < len(s.Body.Rows); i++ {
				if s.Body.Rows[i].Kind == popupMultiline {
					s.Body.Rows[i].Lines = multilineRows(s.Body.Rows[i]) + extra
					break
				}
			}
		}
	}
	if !s.Place.Bounds.Empty() {
		w = min(w, s.Place.Bounds.Dx())
	}
	shown := popupRowCount(s) - s.Body.Top
	if len(s.Body.ColumnTops) > 0 {
		shown = 0
		for i := range s.Body.Rows {
			column, line := popupRowPosition(s, i)
			shown = max(shown, line-popupColumnTop(s, column)+1)
		}
	}
	if s.Body.MaxRows > 0 {
		shown = min(shown, s.Body.MaxRows)
	}
	shown = max(0, shown)
	rowsH, barH := shown*pitch, shown*pitch
	lines := make([]int, 0, len(s.Body.Rows))
	for i, row := range s.Body.Rows {
		if !row.placed() {
			lines = append(lines, i)
		}
	}
	for i := 0; i < shown && s.Body.Top+i < len(lines); i++ {
		at := lines[s.Body.Top+i]
		rowsH += flow.rowHeight(s.Body.Rows[at], at, pitch) - pitch
		barH += flow.rowExtra[at] * pitch
	}
	if s.Body.Columns > 1 {
		rowsH = popupRowsHeight(s, flow, pitch, shown)
		barH = rowsH
	}
	h := titleH + popupPad + flow.cardBand() + rowsH
	var list listBox
	if s.List.Picker != nil {
		list = newListBox(image.Rect(0, 0, w-2*popupPad-widgetSpriteSize, 0), max(1, s.List.Rows), body)
		h += list.Rect.Dy() + popupPad + flow.listHeadH
	}
	h += len(s.Notes.Lines)*pitch + popupPad
	h += flow.buttonLines * (pitch + 4 + popupPad)
	s.Grid = fitGrid(s, w, h, sizedH, sized)
	h += s.Grid.height()
	legend, head := s.Canvas.legendPlaces(body, w-2*popupPad)
	s.Canvas = fitCanvas(s, h, head, sizedH, sized)
	h += s.Canvas.height()
	if sized {
		h = max(h, sizedH)
	}
	r := image.Rectangle{Min: s.Place.At, Max: s.Place.At.Add(image.Pt(w, h))}
	if !s.Place.Bounds.Empty() && s.Place.Slack {
		r = slackInside(r, s.Place.Bounds, titleH)
	} else if !s.Place.Bounds.Empty() {
		if r.Max.X > s.Place.Bounds.Max.X {
			r = r.Add(image.Pt(s.Place.Bounds.Max.X-r.Max.X, 0))
		}
		if r.Max.Y > s.Place.Bounds.Max.Y {
			r = r.Add(image.Pt(0, s.Place.Bounds.Max.Y-r.Max.Y))
		}
		if r.Min.X < s.Place.Bounds.Min.X {
			r = r.Add(image.Pt(s.Place.Bounds.Min.X-r.Min.X, 0))
		}
		if r.Min.Y < s.Place.Bounds.Min.Y {
			r = r.Add(image.Pt(0, s.Place.Bounds.Min.Y-r.Min.Y))
		}
	}
	l := popupLayout{Rect: r, Pitch: pitch, ListRows: s.List.Rows, RowsFit: s.Body.MaxRows}
	l.Title = image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+titleH)
	if len(s.Tabs.Captions) > 0 {
		l.Title.Max.Y -= flow.tabLines * pitch
		l.Tabs = tabStrip{Captions: s.Tabs.Captions, PerLine: flow.tabsPerLine}.rects(image.Pt(r.Min.X+popupPad, l.Title.Max.Y), w-2*popupPad, pitch)
	}
	l.Close = image.Rect(r.Max.X-l.Title.Dy(), r.Min.Y+2, r.Max.X-2, l.Title.Max.Y-2)
	if s.Title.Help != "" {
		l.TitleHelp = l.Close.Sub(image.Pt(l.Close.Dx()+4, 0))
	}
	contentRight := r.Max.X - popupPad
	cardY := r.Min.Y + titleH + popupPad
	card := cardSize(s)
	// cardBand places the card in a band across the rows' width at y: the
	// card at its right, the room left of it its CardRest.
	cardBand := func(y int) {
		band := image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad, y+card.Y)
		l.Card = image.Rect(band.Max.X-card.X, band.Min.Y, band.Max.X, band.Max.Y)
		l.CardRest = image.Rect(band.Min.X, band.Min.Y, l.Card.Min.X-popupPad, band.Max.Y)
	}
	if s.Card.Subject != nil {
		switch {
		case flow.cardBelow:
		case flow.cardTop:
			cardBand(cardY)
		default:
			l.Card = image.Rect(r.Max.X-popupPad-card.X, cardY, r.Max.X-popupPad, cardY+card.Y)
			contentRight -= cardAside(s) + popupPad
		}
	}
	contentY := r.Min.Y + titleH + popupPad
	if !flow.cardBelow {
		contentY += flow.cardBand()
	}
	if s.Body.MaxRows > 0 && popupRowCount(s) > s.Body.MaxRows {
		l.RowBar = image.Rect(contentRight-widgetSpriteSize, contentY, contentRight, contentY+barH)
		contentRight -= widgetSpriteSize + 2
	}
	l.RowsViewport = image.Rect(r.Min.X+popupPad, contentY, contentRight, contentY+rowsH)
	labelW := s.Body.LabelWidth
	if labelW <= 0 {
		labelW = (contentRight - r.Min.X - popupPad) * 2 / 5
	}
	if flow.labelW > 0 {
		labelW = flow.labelW
	}
	y := contentY
	shownRows := popupShownRows(s, shown)
	columnExtra := make([]int, max(1, s.Body.Columns))
	for _, index := range shownRows {
		row := s.Body.Rows[index]
		if row.placed() {
			continue
		}
		left, right := r.Min.X+popupPad, contentRight
		if s.Body.Columns > 1 {
			column, line := popupRowPosition(s, index)
			columnW := (contentRight - r.Min.X - popupPad) / s.Body.Columns
			left, right = left+column*columnW, left+column*columnW+popupColumnWidth(s, contentRight-left)
			y = contentY + (line-popupColumnTop(s, column))*pitch + columnExtra[column]
			columnExtra[column] += flow.rowHeight(row, index, pitch) - pitch
		}
		if s.Body.Columns <= 1 {
			right = left + popupColumnWidth(s, right-left)
		}
		rowH := flow.rowHeight(row, index, pitch)
		extra, over := flow.rowExtra[index], flow.labelOver[index]
		rr := popupRowRect{Index: index, Rect: image.Rect(left, y, right, y+rowH)}
		rowLabelW := labelW
		if s.Body.Columns > 1 {
			column, _ := popupRowPosition(s, index)
			rowLabelW = flow.columnLabelW[column]
		}
		// top is the control's line: under the label when the label takes a
		// line of its own.
		top, fieldH := y, pitch
		if row.Kind == popupMultiline || pictureRow(row) {
			fieldH = multilineRows(row) * pitch
		}
		if row.Kind == popupDirection {
			fieldH = 3 * pitch
		}
		rr.Label = image.Rect(rr.Rect.Min.X, y, rr.Rect.Min.X+rowLabelW, y+pitch)
		rr.Control = image.Rect(rr.Label.Max.X+4, y+1, rr.Rect.Max.X, y+fieldH-1)
		if over {
			top, extra = y+pitch, extra-1
			rr.Label.Max.X = rr.Rect.Max.X
			rr.Control = image.Rect(rr.Rect.Min.X, top+1, rr.Rect.Max.X, top+fieldH-1)
		}
		// The row's first line: the label and its control, or a line of text.
		line := image.Rect(rr.Rect.Min.X, top, rr.Rect.Max.X, top+fieldH)
		switch row.Kind {
		case popupDirection:
			rr.Buttons = make([]image.Rectangle, len(row.ChoicePad))
			for k, pos := range row.ChoicePad {
				at := rr.Control.Min.Add(pos.Add(image.Pt(1, 1)).Mul(pitch))
				rr.Buttons[k] = image.Rectangle{Min: at, Max: at.Add(image.Pt(pitch-2, pitch-2))}
			}
		case popupHeading:
			rr.Label, rr.Control = line, image.Rectangle{}
		case popupText, popupLink:
			// A line of text runs across the row; a held place is no field.
			if row.Value == "" && !pictureRow(row) || row.Hold != holdNone {
				rr.Label, rr.Control = line, image.Rectangle{}
			}
		case popupCard:
			if row.Struck {
				rr.Label, rr.Control = line, image.Rectangle{}
			} else {
				rr.Grip = rr.Label
			}
			if !row.Struck && row.Value == "" {
				// A card with no drop-down: its number's line runs across.
				rr.Label, rr.Control = line, image.Rectangle{}
			}
		}
		if ws := rowButtonWidths(s, row, body, pitch); len(ws) > 0 {
			rr.Buttons = make([]image.Rectangle, len(ws))
			if extra > 0 {
				// The buttons take the lines under the first, right-aligned,
				// in order from the first line on.
				lines, _ := flowLines(ws, 4, rr.Rect.Dx(), 0)
				xs := make([]int, extra)
				for k := range xs {
					xs[k] = rr.Rect.Max.X
				}
				for k := len(ws) - 1; k >= 0; k-- {
					n := min(lines[k], extra-1)
					top := line.Max.Y + n*pitch
					rr.Buttons[k] = image.Rect(xs[n]-ws[k], top+1, xs[n], top+pitch-1)
					xs[n] -= ws[k] + 4
				}
			} else {
				x := rr.Rect.Max.X
				for k := len(ws) - 1; k >= 0; k-- {
					rr.Buttons[k] = image.Rect(x-ws[k], top+1, x, top+pitch-1)
					x -= ws[k] + 4
				}
				if rr.Control.Empty() {
					rr.Label.Max.X = min(rr.Label.Max.X, x)
				} else {
					rr.Control.Max.X = x
				}
			}
			rr.Button, rr.Off = rr.Buttons[0], row.Disabled
		}
		if row.Help != "" && !row.placed() {
			// The (?) mark ends the label's line; the label is cut before it.
			side := pitch - 4
			rr.Help = image.Rect(rr.Label.Max.X-side-2, rr.Label.Min.Y+2, rr.Label.Max.X-2, rr.Label.Min.Y+2+side)
			rr.Label.Max.X = rr.Help.Min.X - 2
		}
		l.Rows = append(l.Rows, rr)
		y += rowH
	}
	if s.Body.Columns > 1 {
		y = contentY + rowsH
	}
	switch {
	case s.Card.Subject != nil && flow.cardBelow && flow.bandScrolls:
		// The band follows the last row: below the viewport while rows
		// remain unshown, scrolled up by the band lines above the top.
		rowLines := popupRowCount(s) - bandLines(s)
		placed := 0
		for _, index := range shownRows {
			if !s.Body.Rows[index].placed() {
				placed++
			}
		}
		top := y
		switch {
		case s.Body.Top > rowLines:
			top = contentY - (s.Body.Top-rowLines)*pitch
		case s.Body.Top+placed < rowLines:
			top = contentY + rowsH
		}
		band := image.Rect(r.Min.X+popupPad, top+popupPad, contentRight, top+bandLines(s)*pitch)
		l.Card = image.Rect(band.Max.X-card.X, band.Min.Y, band.Max.X, band.Min.Y+card.Y)
		l.CardRest = image.Rect(band.Min.X, band.Min.Y, l.Card.Min.X-popupPad, max(band.Max.Y, l.Card.Max.Y))
		y = contentY + rowsH
	case s.Card.Subject != nil && flow.cardBelow:
		cardBand(y + popupPad)
		y = l.Card.Max.Y
	}
	if subject != nil && s.Card.Subject == nil {
		for _, rr := range l.Rows {
			if s.Body.Rows[rr.Index].Hold == holdCard {
				l.Card = image.Rectangle{Min: rr.Rect.Min, Max: rr.Rect.Min.Add(card)}
				l.CardRest = image.Rect(l.Card.Max.X+popupPad, rr.Rect.Min.Y, rr.Rect.Max.X, rr.Rect.Max.Y)
				break
			}
		}
	}
	for _, index := range shownRows {
		row := s.Body.Rows[index]
		if !row.placed() || row.HeadOf > 0 || subject == nil || l.Card.Empty() {
			continue
		}
		box := row.Place.Add(l.Card.Min)
		if row.RestLeft {
			box = row.Place.Add(image.Pt(l.CardRest.Min.X, l.Card.Min.Y))
		}
		rr := popupRowRect{Index: index, Rect: box}
		if row.Kind != popupText && row.Kind != popupLink {
			rr.Control = box
		}
		if row.Kind == popupDirection {
			// A placed pad: its label on the first line, the buttons under it.
			rr.Label = image.Rect(box.Min.X, box.Min.Y, box.Max.X, box.Min.Y+pitch)
			rr.Control = image.Rect(box.Min.X, box.Min.Y+pitch, box.Max.X, box.Max.Y)
			rr.Buttons = make([]image.Rectangle, len(row.ChoicePad))
			for k, pos := range row.ChoicePad {
				at := rr.Control.Min.Add(pos.Add(image.Pt(1, 1)).Mul(pitch))
				rr.Buttons[k] = image.Rectangle{Min: at, Max: at.Add(image.Pt(pitch-2, pitch-2))}
			}
		}
		l.Rows = append(l.Rows, rr)
	}
	l.Drop = -1
	layoutCards(s, &l)
	if g := s.Grid; g != nil {
		side, shown, top := g.side(), max(1, g.Rows), y
		l.GridRows = shown
		l.GridCols = g.columns(r.Dx() - 2*popupPad - g.HeadW)
		total := g.totalRows(l.GridCols)
		l.GridTop = max(0, min(g.Top, total-shown))
		x := r.Min.X + popupPad + g.HeadW
		for c := 0; c < len(g.ColHeads) && c < l.GridCols; c++ {
			l.GridColHeads = append(l.GridColHeads, image.Rect(x+c*side, y, x+(c+1)*side, y+g.headHeight()))
		}
		y += g.headHeight()
		l.Grid = image.Rect(x, y, x+l.GridCols*side, y+g.rowsHigh(l.GridTop, shown))
		if total > shown {
			l.GridBar = image.Rect(r.Max.X-popupPad-widgetSpriteSize, y, r.Max.X-popupPad, l.Grid.Max.Y)
		}
		l.GridFirst = l.GridTop * l.GridCols
		rows := g.rows(l.GridCols)
		for k, at := 0, y; k < shown && l.GridTop+k < len(rows); k++ {
			row, high := rows[l.GridTop+k], g.rowHigh(l.GridTop+k)
			if row.Head != "" || len(g.Groups) == 0 && l.GridTop+k < len(g.RowHeads) {
				left, right := r.Min.X+popupPad, x
				if len(g.Groups) > 0 {
					left, right = x, l.Grid.Max.X
				}
				l.GridRowHeads = append(l.GridRowHeads, image.Rect(left, at, right, at+high))
				l.GridHeadText = append(l.GridHeadText, row.Head)
			}
			for index, held := range s.Body.Rows {
				if held.HeadOf != l.GridTop+k+1 || !held.placed() {
					continue
				}
				box := held.Place.Add(image.Pt(r.Min.X+popupPad, at))
				rr := popupRowRect{Index: index, Rect: box, Control: box}
				if len(held.Buttons) > 0 {
					rr.Control, rr.Buttons, rr.Button, rr.Off = image.Rectangle{}, []image.Rectangle{box}, box, held.Disabled
				}
				l.Rows = append(l.Rows, rr)
			}
			for column, n := range row.Cells {
				corner := image.Pt(x+column*side, at)
				l.GridCells = append(l.GridCells, image.Rectangle{Min: corner, Max: corner.Add(image.Pt(side, high))}.Inset(1))
				l.GridInert = append(l.GridInert, g.Cells[n].Inert)
				if len(g.Groups) > 0 {
					l.GridIndices = append(l.GridIndices, n)
				}
			}
			at += high
		}
		y = top + g.height()
	}
	if c := s.Canvas; c != nil {
		y = l.placeCanvas(c, r, y, legend)
	}
	if s.List.Picker != nil {
		if len(flow.listHeads) > 0 {
			x := r.Min.X + popupPad + listTextX
			stops := columnStops(listShares(s.List.Columns), r.Dx()-2*popupPad-widgetSpriteSize-s.List.Aside-2*listTextX)
			for k := range flow.listHeads {
				l.ListHeads = append(l.ListHeads, image.Rect(x+stops[k], y, x+stops[k+1]-listColumnGap/2, y+flow.listHeadH-popupPad))
			}
			l.ListHeadLines = flow.listHeads
			y += flow.listHeadH
		}
		l.List = newListBox(image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad-widgetSpriteSize-s.List.Aside, y), max(1, s.List.Rows), body)
		l.List.Columns = listShares(s.List.Columns)
		l.HasList = true
		if s.List.Aside > 0 {
			l.Aside = image.Rect(r.Max.X-popupPad-s.List.Aside+popupPad, y, r.Max.X-popupPad, l.List.Rect.Max.Y)
		}
		y = l.List.Rect.Max.Y + popupPad
	}
	for range s.Notes.Lines {
		l.Notes = append(l.Notes, image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad, y+pitch))
		y += pitch
	}
	if n := len(s.Buttons.Captions); n > 0 || s.Buttons.Switch != "" {
		step := pitch + 4 + popupPad
		y += popupPad
		if sized {
			y = r.Max.Y - popupPad - (pitch + 4) - (flow.buttonLines-1)*step
		}
		if sw := switchButtonWidth(s, body); sw > 0 {
			l.SwitchButton = image.Rect(r.Min.X+popupPad, y, r.Min.X+popupPad+sw, y+pitch+4)
		}
		ws := popupButtonWidths(s, body)
		xs := make([]int, flow.buttonLines)
		for k := range xs {
			xs[k] = r.Max.X - popupPad
		}
		l.Buttons = make([]image.Rectangle, n)
		for i := n - 1; i >= 0; i-- {
			k := min(flow.buttonLine[i], flow.buttonLines-1)
			top := y + k*step
			l.Buttons[i] = image.Rect(xs[k]-ws[i], top, xs[k], top+pitch+4)
			xs[k] -= ws[i] + popupPad
		}
	}
	if s.Card.Subject != nil && !flow.cardTop {
		// The column under the card, down to the buttons, holds what the
		// card's builder draws beside it (the person's face).
		bottom := r.Max.Y - popupPad
		if len(l.Buttons) > 0 {
			bottom = l.Buttons[0].Min.Y - popupPad
		} else if !l.SwitchButton.Empty() {
			bottom = l.SwitchButton.Min.Y - popupPad
		}
		// No room under the card leaves it empty, never turned over the buttons.
		top := l.Card.Max.Y + popupPad
		l.CardRest = image.Rect(r.Max.X-popupPad-cardAside(s), top, r.Max.X-popupPad, max(top, bottom))
	}
	visible := l.Rows[:0]
	for _, rr := range l.Rows {
		if flow.bandScrolls && s.Body.Rows[rr.Index].placed() && !rr.Rect.In(l.RowsViewport) {
			continue // a card cell scrolled out of the viewport
		}
		if !s.Body.Rows[rr.Index].placed() {
			if !rr.Rect.In(l.RowsViewport) {
				continue
			}
			rr.Label = rr.Label.Intersect(l.RowsViewport)
			rr.Control = rr.Control.Intersect(l.RowsViewport)
			rr.Button = rr.Button.Intersect(l.RowsViewport)
			rr.Grip = rr.Grip.Intersect(l.RowsViewport)
			rr.Card = rr.Card.Intersect(l.RowsViewport)
			for k := range rr.Buttons {
				rr.Buttons[k] = rr.Buttons[k].Intersect(l.RowsViewport)
			}
		}
		visible = append(visible, rr)
	}
	l.Rows = visible
	return l
}

// layoutCards gives each shown card row its card's box over the rows it
// holds and, while a card is held, the card it lands before and the bar
// that marks that gap.
func layoutCards(s uiWindow, l *popupLayout) {
	ordinal := map[int]int{}
	held, cards := -1, 0
	for i, row := range s.Body.Rows {
		if row.Kind != popupCard {
			continue
		}
		ordinal[i] = cards
		if s.Place.Drag != nil && s.Place.Drag.Row == i {
			held = cards
		}
		cards++
	}
	var shown []int
	for i, rr := range l.Rows {
		row := s.Body.Rows[rr.Index]
		if row.Kind != popupCard {
			continue
		}
		card := rr.Rect
		for k := 1; k <= row.Span && i+k < len(l.Rows) && l.Rows[i+k].Index == rr.Index+k; k++ {
			card = card.Union(l.Rows[i+k].Rect)
		}
		l.Rows[i].Card = card
		shown = append(shown, i)
	}
	if held < 0 || len(shown) == 0 {
		return
	}
	at := s.Place.Drag.At.Y
	last := l.Rows[shown[len(shown)-1]]
	y := last.Card.Max.Y
	l.Drop = ordinal[last.Index] + 1
	for _, i := range shown {
		if c := l.Rows[i].Card; at < (c.Min.Y+c.Max.Y)/2 {
			l.Drop, y = ordinal[l.Rows[i].Index], c.Min.Y
			break
		}
	}
	if l.Drop != held && l.Drop != held+1 {
		l.Marker = image.Rect(last.Rect.Min.X, y-2, last.Rect.Max.X, y+1)
	}
}

// popupButtonWidths are the widths of the button row's buttons: the spec's
// button width, or the default, each widened to its caption where the
// caption needs more, so no caption paints past its button.
func popupButtonWidths(s uiWindow, body *text.Font) []int {
	bw := popupButtonW
	if s.Buttons.Least > 0 {
		bw = s.Buttons.Least
	}
	ws := make([]int, len(s.Buttons.Captions))
	for i, c := range s.Buttons.Captions {
		ws[i] = bw
		if body != nil {
			ws[i] = max(bw, body.Advance(c)+16)
		}
	}
	return ws
}

// switchButtonWidth is the width of s's SwitchButton, as wide as its
// caption; 0 for none.
func switchButtonWidth(s uiWindow, body *text.Font) int {
	if s.Buttons.Switch == "" {
		return 0
	}
	return popupButtonWidths(uiWindow{Buttons: uiButtons{Captions: []string{s.Buttons.Switch}, Least: 48}}, body)[0]
}

// rowButtons are the captions of row's buttons.
func rowButtons(s uiWindow, row popupRow) []string {
	if len(row.Buttons) > 0 {
		return row.Buttons
	}
	if row.Kind == popupPick {
		return []string{s.Pick.Label}
	}
	return nil
}

// rowButtonAt is the row and the button of that row under p.
func (l popupLayout) rowButtonAt(p image.Point) (row, button int, ok bool) {
	for _, r := range l.Rows {
		for k, b := range r.Buttons {
			if p.In(b) && !(k < len(r.Off) && r.Off[k]) {
				return r.Index, k, true
			}
		}
	}
	return 0, 0, false
}

// popupPart is what a press on a popup lands on.
type popupPart uint8

const (
	popupMiss       popupPart = iota // outside the popup
	popupBody                        // the popup's ground: consumed, no action
	popupTitle                       // the title bar: a drag moves the popup
	popupClose                       // the close button
	popupRowLine                     // a row's label or value
	popupRowHelp                     // a row's (?) mark: its help, never its action
	popupRowControl                  // a row's control
	popupRowButton                   // a row's own button
	popupListRow                     // a list row
	popupListBar                     // the list's scroll bar
	popupButton                      // a button of the button row
	popupGridPart                    // a grid cell; the index is the cell's
	popupGridBar                     // the grid's scroll bar
	popupRowGrip                     // a card's grip: a drag moves the card
	popupTab
	popupSwitchButton // the button row's SwitchButton
	popupCanvasBox    // a canvas box that takes a press; the index is the box's
	popupCanvasGround // the canvas's ground: a drag moves the sheet
	popupCanvasBar    // the canvas's scroll bar
)

// gridIndex maps a visible rectangle to its unchanged source cell index.
func (l popupLayout) gridIndex(shown int) int {
	if len(l.GridIndices) > 0 {
		return l.GridIndices[shown]
	}
	return l.GridFirst + shown
}

// hit reports the part under p and its row, list or button index.
func (l popupLayout) hit(p image.Point) (popupPart, int) {
	if !p.In(l.Rect) {
		return popupMiss, 0
	}
	if p.In(l.Close) {
		return popupClose, 0
	}
	if p.In(l.Title) {
		return popupTitle, 0
	}
	for i, tab := range l.Tabs {
		if p.In(tab) {
			return popupTab, i
		}
	}
	if row, _, ok := l.rowButtonAt(p); ok {
		return popupRowButton, row
	}
	// The row drawn last stands on top, as a row placed on a held card.
	for _, r := range slices.Backward(l.Rows) {
		switch {
		case p.In(r.Help):
			return popupRowHelp, r.Index
		case p.In(r.Grip):
			return popupRowGrip, r.Index
		case p.In(r.Control):
			return popupRowControl, r.Index
		case p.In(r.Rect):
			return popupRowLine, r.Index
		}
	}
	if p.In(l.GridBar) {
		return popupGridBar, 0
	}
	for i, c := range l.GridCells {
		if p.In(c) && !l.GridInert[i] {
			return popupGridPart, l.gridIndex(i)
		}
	}
	if p.In(l.CanvasBar) {
		return popupCanvasBar, 0
	}
	if p.In(l.Canvas) {
		return l.canvasHit(p)
	}
	if l.HasList {
		if p.In(l.List.Bar()) {
			return popupListBar, 0
		}
		if row, ok := l.List.RowAt(p); ok {
			return popupListRow, row
		}
	}
	for i, b := range l.Buttons {
		if p.In(b) {
			return popupButton, i
		}
	}
	if p.In(l.SwitchButton) {
		return popupSwitchButton, 0
	}
	return popupBody, 0
}

// popupEdge is the border of a popup a press lands on, as bits: a corner is
// two of them. A drag from a border resizes the popup.
type popupEdge uint8

const (
	edgeLeft popupEdge = 1 << iota
	edgeRight
	edgeTop
	edgeBottom
)

// popupEdgeBand is how many pixels inside the border count as the border;
// popupGripSide (in the placement store) is the side of the bottom-right
// square that resizes both ways.
const (
	popupEdgeBand = 8
	// popupTopBand is the top border's band, as deep as the others; the
	// title row below it drags.
	popupTopBand  = popupEdgeBand
	popupKeepWide = 48
)

// slackInside moves r the least that keeps its title row inside bounds and at
// least popupKeepWide pixels of its width inside them.
func slackInside(r, bounds image.Rectangle, titleH int) image.Rectangle {
	keep := min(popupKeepWide, r.Dx())
	dx, dy := 0, 0
	if r.Max.X < bounds.Min.X+keep {
		dx = bounds.Min.X + keep - r.Max.X
	} else if r.Min.X > bounds.Max.X-keep {
		dx = bounds.Max.X - keep - r.Min.X
	}
	if r.Min.Y < bounds.Min.Y {
		dy = bounds.Min.Y - r.Min.Y
	} else if r.Min.Y+titleH > bounds.Max.Y {
		dy = bounds.Max.Y - titleH - r.Min.Y
	}
	return r.Add(image.Pt(dx, dy))
}

// edgeAt is the border under p, none for a press inside the popup's body or
// on its close button.
func (l popupLayout) edgeAt(p image.Point) popupEdge {
	if !p.In(l.Rect) || p.In(l.Close) {
		return 0
	}
	var e popupEdge
	switch {
	case p.X < l.Rect.Min.X+popupEdgeBand:
		e |= edgeLeft
	case p.X >= l.Rect.Max.X-popupEdgeBand:
		e |= edgeRight
	}
	switch {
	case p.Y < l.Rect.Min.Y+popupTopBand:
		e |= edgeTop
	case p.Y >= l.Rect.Max.Y-popupEdgeBand:
		e |= edgeBottom
	}
	if p.X >= l.Rect.Max.X-popupGripSide && p.Y >= l.Rect.Max.Y-popupGripSide {
		return edgeRight | edgeBottom
	}
	// A row, list or button that reaches into the band keeps its press.
	if part, _ := l.hit(p); part != popupBody && part != popupTitle {
		return 0
	}
	return e
}

// popupInk is the editor popup's palette: the frame, title, label and value
// inks, an empty field's hint, and the push button's caption and bevels.
type popupInk struct {
	Ground, Edge, TitleGround, Title, Label, Value color.RGBA
	Hint, Button, BevelLight, BevelDark            color.RGBA
}

// drawPopup is the one editor popup painter: the frame, the title bar and its
// close button, each shown row's label and control, the list box, the notes
// and the buttons, each through the widget kit's builder.
func drawPopup(dst *image.RGBA, title, body *text.Font, frames []*image.RGBA, spec popupSpec, l popupLayout, ink popupInk, pointer image.Point, pointerOK bool, press ...*buttonPress) {
	s := l.window(spec)
	var held *buttonPress
	if len(press) > 0 {
		held = press[0]
	}
	if dst == nil {
		return
	}
	held.cover(l.Rect)
	drawFrame(dst, panelFrame(l.Rect, ink.Ground, ink.Edge))
	if !l.RowBar.Empty() {
		drawVScrollBar(dst, frames, vScrollBar{Rect: l.RowBar, Pos: s.Body.Top, Count: popupRowCount(s) - s.Body.MaxRows + 1}.withPointer(pointer, pointerOK))
	}
	if s.Card.Subject != nil && !l.Card.Empty() {
		font := s.Card.Font
		if font == nil {
			font = body
		}
		if s.Card.Scale > 1 {
			// A scaled card's glyphs scale with it: the text overlay draws
			// each at the card's scale, at the device's resolution.
			start := markCapture()
			if pic := RenderCharacterPanel(CompactPanelLayout(nil), font, *s.Card.Subject); pic != nil {
				c, to := l.Card, dst
				if bandLines(s) > 0 || cardHeld(s) {
					// A scrolling band, or a row holding the card, shows
					// only inside the rows' viewport.
					to = dst.SubImage(l.RowsViewport).(*image.RGBA)
				}
				editorBlit(to, pic, screenRect{X: float64(c.Min.X), Y: float64(c.Min.Y), W: float64(c.Dx()), H: float64(c.Dy())}, false)
				scaleCapture(start, c.Min, s.Card.Scale, to.Bounds())
			}
		} else {
			start := markCapture()
			if pic := RenderCharacterPanel(CompactPanelLayout(nil), font, *s.Card.Subject); pic != nil {
				at := l.Card.Min
				draw.Draw(dst, pic.Bounds().Add(at), pic, pic.Bounds().Min, draw.Over)
				shiftCapture(start, at)
			}
		}
	}
	if !l.Close.Empty() {
		drawFrame(dst, panelFrame(l.Title, ink.TitleGround, ink.Edge))
		drawPopupGrip(dst, l.Rect, ink.Edge)
		if title != nil {
			end := l.Close.Min.X
			if !l.TitleHelp.Empty() {
				end = l.TitleHelp.Min.X
				drawHelpMark(dst, body, l.TitleHelp, ink, pointerOK && pointer.In(l.TitleHelp))
			}
			clip := dst.SubImage(image.Rect(l.Title.Min.X+popupPad, l.Title.Min.Y, end-2, l.Title.Max.Y)).(*image.RGBA)
			title.Draw(clip, editorClip(title, s.Title.Text, clip.Bounds().Dx()), l.Title.Min.X+popupPad, l.Title.Min.Y+popupRowExtra/2, ink.Title)
		}
		closeLabel := "x"
		if s.Title.Close != "" {
			closeLabel = s.Title.Close
		}
		drawPushButton(dst, body, pushButton{Press: held, Rect: l.Close, Label: closeLabel, Literal: true, LabelRaise: labelRaise(body), Hover: pointerOK && pointer.In(l.Close)})
	}
	if len(l.Tabs) > 0 {
		over, at := popupMiss, 0
		if pointerOK {
			over, at = l.hit(pointer)
		}
		tabStrip{Captions: s.Tabs.Captions, Active: s.Tabs.Active, Switch: s.Tabs.Switch}.draw(dst, body, l.Tabs, image.Rectangle{}, held, over, at)
	}
	bodyText := func(r image.Rectangle, s string, c color.RGBA) {
		if body == nil || s == "" {
			return
		}
		body.Draw(dst.SubImage(r.Intersect(l.Rect)).(*image.RGBA), s, r.Min.X+2, r.Min.Y+popupRowExtra/2, c)
	}
	canvas := dst
	for _, rr := range l.Rows {
		row := s.Body.Rows[rr.Index]
		dst = canvas
		if !row.placed() {
			dst = canvas.SubImage(l.RowsViewport).(*image.RGBA)
		}
		value := row.Ink
		if value == (color.RGBA{}) {
			value = ink.Value
		}
		if row.placed() && row.Kind != popupDirection && row.HeadOf == 0 {
			if !rr.Control.Empty() {
				// An input on the card replaces the card's value under it.
				draw.Draw(dst, rr.Control.Intersect(l.Rect), &image.Uniform{C: ink.Ground}, image.Point{}, draw.Src)
				drawEditRow(dst, body, row, rr.Control, ink, value)
			}
			continue
		}
		if row.Kind == popupHeading {
			band := rr.Rect.Intersect(l.Rect)
			edge := ink.Edge
			if row.Focus {
				edge = ink.Title
			}
			drawFrame(dst, panelFrame(band, ink.TitleGround, edge))
			if title != nil {
				x := rr.Rect.Min.X + 4
				if row.Fold != FoldNone {
					// The fold mark, as a tree row draws it before its text.
					title.Draw(dst.SubImage(band).(*image.RGBA), rowFoldMarks[row.Fold], x, rr.Rect.Min.Y+popupRowExtra/2, ink.Title)
				}
				title.Draw(dst.SubImage(band).(*image.RGBA), row.Label, x+foldIndent(row), rr.Rect.Min.Y+popupRowExtra/2, ink.Title)
			}
			continue
		}
		label := rr.Label
		if row.Kind == popupCard {
			// The card's ground over the rows it holds, lighter while it is
			// held; a struck card is a strip in the hint ink.
			ground, edge := ink.TitleGround, ink.Edge
			if row.Struck {
				edge = ink.Hint
			} else if s.Place.Drag != nil && s.Place.Drag.Row == rr.Index {
				ground = ink.Ground
			}
			drawFrame(dst, panelFrame(rr.Card.Intersect(l.Rect), ground, edge))
			if row.Struck {
				bodyText(label, editorClip(body, row.Label, label.Dx()-2), ink.Hint)
				for k, b := range rr.Buttons {
					drawPushButton(dst, body, pushButton{Press: held, Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Hover: pointerOK && pointer.In(b)})
				}
				continue
			}
			// The grip: three strokes before the card's number.
			mid := (label.Min.Y + label.Max.Y) / 2
			for k := -1; k <= 1; k++ {
				stroke := image.Rect(label.Min.X+4, mid+4*k-1, label.Min.X+14, mid+4*k+1)
				draw.Draw(dst, stroke.Intersect(l.Rect), &image.Uniform{C: ink.Edge}, image.Point{}, draw.Src)
			}
			label.Min.X += 18
		}
		bodyText(label, editorClip(body, row.Label, label.Dx()-2), ink.Label)
		if row.Edit != nil {
			drawEditRow(dst, body, row, rr.Control, ink, value)
			drawRowPictures(dst, body, l, rr, row)
			for k, b := range rr.Buttons {
				drawPushButton(dst, body, pushButton{Press: held, Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Focus: row.Kind == popupPick && s.Pick.Armed || k < len(row.Armed) && row.Armed[k], Disabled: k < len(row.Disabled) && row.Disabled[k], Hover: pointerOK && pointer.In(b)})
			}
			continue
		}
		switch row.Kind {
		case popupDirection:
			for k, b := range rr.Buttons {
				selected := k == row.Selected
				drawPushButton(dst, body, pushButton{Press: held, Rect: b, Pressed: selected, Inside: selected, Focus: row.Focus && selected, Hover: pointerOK && pointer.In(b)})
				drawDirectionMark(dst, b, row.ChoicePad[k], value)
			}
		case popupMultiline:
			drawFrame(dst, panelFrame(rr.Control, ink.Ground, ink.Edge))
			if body != nil {
				var lines []string
				for _, line := range strings.Split(row.Value, "\n") {
					lines = append(lines, editorTextLines(body, line, rr.Control.Dx()-8)...)
				}
				visible := max(1, (rr.Control.Dy()-8)/fontHeight(body, 15))
				start := max(0, min(row.First, len(lines)-visible))
				if row.Focus {
					start = max(0, len(lines)-visible)
				}
				clip := dst.SubImage(rr.Control.Inset(3)).(*image.RGBA)
				y := rr.Control.Min.Y + 4
				for _, line := range lines[start:min(len(lines), start+visible)] {
					body.Draw(clip, line, rr.Control.Min.X+4, y, value)
					y += fontHeight(body, 15)
				}
				if row.Focus {
					x := min(rr.Control.Max.X-4, rr.Control.Min.X+4+body.Advance(lines[len(lines)-1]))
					for yy := y - fontHeight(body, 15); yy < y; yy++ {
						if image.Pt(x, yy).In(clip.Bounds()) {
							clip.SetRGBA(x, yy, value)
						}
					}
				}
			}
		case popupEdit, popupPick:
			drawEditRow(dst, body, row, rr.Control, ink, value)
		case popupSwitch:
			// A long value ends in an ellipsis inside its button.
			drawPushButton(dst, body, pushButton{Press: held, Rect: rr.Control, Label: editorClip(body, row.Value, rr.Control.Dx()-12), Literal: true, LabelRaise: labelRaise(body), Focus: row.Focus, Hover: pointerOK && pointer.In(rr.Control)})
		case popupChoice, popupCard:
			if !rr.Control.Empty() {
				drawDropdown(dst, body, frames, dropdown{Press: held, Rect: rr.Control, Text: row.Value, Focus: row.Focus, Fill: ink.Ground, Ink: value}, nil, pointer, pointerOK)
			}
		default:
			bodyText(rr.Control, editorClip(body, row.Value, rr.Control.Dx()-2), value)
		}
		drawRowPictures(dst, body, l, rr, row)
		for k, b := range rr.Buttons {
			if row.Kind == popupDirection {
				break
			}
			drawPushButton(dst, body, pushButton{Press: held, Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Focus: row.Kind == popupPick && s.Pick.Armed || k < len(row.Armed) && row.Armed[k], Disabled: k < len(row.Disabled) && row.Disabled[k], Hover: pointerOK && pointer.In(b)})
		}
	}
	for _, rr := range l.Rows {
		if !rr.Help.Empty() {
			drawHelpMark(canvas.SubImage(l.RowsViewport).(*image.RGBA), body, rr.Help, ink, pointerOK && pointer.In(rr.Help))
		}
	}
	dst = canvas
	if !l.Marker.Empty() {
		draw.Draw(dst, l.Marker.Intersect(l.Rect), &image.Uniform{C: ink.Title}, image.Point{}, draw.Src)
	}
	if g := s.Grid; g != nil {
		drawPopupGrid(dst, body, frames, g, l, ink, pointer, pointerOK, held)
	}
	if c := s.Canvas; c != nil && !l.Canvas.Empty() {
		over := l.canvasOver(pointer, pointerOK)
		drawCanvasSheet(dst, c, l, ink)
		clip := dst.SubImage(l.Canvas.Inset(1)).(*image.RGBA)
		for _, b := range l.canvasButtons(c, over, held) {
			drawPushButton(clip, body, b)
		}
		drawCanvasWords(dst, body, c, l, ink, over)
		drawCanvasLegend(dst, body, c, l, ink)
		if !l.CanvasBar.Empty() {
			drawVScrollBar(dst, frames, l.canvasBar(c).withPointer(pointer, pointerOK))
		}
	}
	if l.HasList && s.List.Text != nil {
		box := l.List
		box.HeadingFace = title
		drawListBox(dst, body, frames, box, s.List.Picker, s.List.Text, pointer, pointerOK)
	}
	for k, r := range l.ListHeads {
		for j, line := range l.ListHeadLines[k] {
			body.Draw(dst.SubImage(r).(*image.RGBA), line, r.Min.X, r.Min.Y+j*fontHeight(body, 15), ink.Title)
		}
	}
	for i, r := range l.Notes {
		bodyText(r, s.Notes.Lines[i], s.Notes.Ink)
	}
	for i, r := range l.Buttons {
		disabled := i < len(s.Buttons.Disabled) && s.Buttons.Disabled[i]
		drawPushButton(dst, body, pushButton{Press: held, Rect: r, Label: s.Buttons.Captions[i], Literal: true, LabelRaise: labelRaise(body), Disabled: disabled, Focus: s.Buttons.Focus != nil && *s.Buttons.Focus == i, Hover: pointerOK && pointer.In(r)})
	}
	if r := l.SwitchButton; !r.Empty() {
		drawPushButton(dst, body, pushButton{Press: held, Rect: r, Label: s.Buttons.Switch, Literal: true, LabelRaise: labelRaise(body), Hover: pointerOK && pointer.In(r)})
	}
}

// drawHelpMark draws a (?) mark in box r, lit while the pointer is over it.
func drawHelpMark(dst *image.RGBA, body *text.Font, r image.Rectangle, ink popupInk, hover bool) {
	ground := ink.Ground
	if hover {
		ground = ink.TitleGround
	}
	drawFrame(dst, panelFrame(r, ground, ink.Edge))
	if body != nil {
		body.Draw(dst.SubImage(r).(*image.RGBA), "?", r.Min.X+(r.Dx()-body.Advance("?"))/2, r.Min.Y+(r.Dy()-fontHeight(body, 15))/2, ink.Title)
	}
}

// foldIndent is how far a heading's label stands right of its fold mark:
// a tree row's step, none for a heading that does not open and close.
func foldIndent(row popupRow) int {
	if row.Fold == FoldNone {
		return 0
	}
	return listIndentStep
}

// rowFull is shown row rr's whole text, label then value, and whether the
// painter cuts any part of it: the label to its column, a value to its
// control, by the same widths the painter clips to.
func rowFull(title, body *text.Font, row popupRow, rr popupRowRect) (string, bool) {
	if body == nil || row.Edit != nil || row.Kind == popupMultiline {
		return "", false
	}
	full := strings.TrimSpace(row.Label + " " + row.Value)
	if row.Kind == popupHeading {
		f := title
		if f == nil {
			f = body
		}
		return full, f.Advance(row.Label) > rr.Rect.Dx()-4-foldIndent(row)
	}
	label := rr.Label
	if row.Kind == popupCard && !row.Struck {
		label.Min.X += 18
	}
	cut := row.Label != "" && body.Advance(row.Label) > label.Dx()-2
	room := -1
	switch c := rr.Control; {
	case c.Empty() || row.Value == "":
	case row.Kind == popupSwitch:
		room = c.Dx() - 12
	case row.Kind == popupChoice || row.Kind == popupCard:
		room = c.Dx() - min(c.Dx()/2, c.Dy()) - 2 - editTextX
	case row.Kind == popupEdit || row.Kind == popupPick:
		room = c.Dx() - 2 - editTextX
	default:
		room = c.Dx() - 2
	}
	if room >= 0 && body.Advance(row.Value) > room {
		cut = true
	}
	return full, cut
}

// popupTipAt is the tip of the shown row under p, false when the pointer is
// over no row or the row shows its text whole: a (?) mark's help, a cut
// row's whole text or the row's own Tip. Over the grid it is the whole
// upright column head; a cell's Tip is the editor's hover tip (tipAt).
func popupTipAt(bounds image.Rectangle, title, body *text.Font, spec popupSpec, l popupLayout, p image.Point, pointerOK bool) (popupTip, bool) {
	s := l.window(spec)
	if !pointerOK || body == nil {
		return popupTip{}, false
	}
	if p.In(l.TitleHelp) {
		return helpTip(bounds, body, l.TitleHelp, s.Title.Help), true
	}
	// The row drawn last stands on top: a row placed on a held card answers
	// over the place that holds the card.
	for _, rr := range slices.Backward(l.Rows) {
		if !p.In(rr.Rect) {
			continue
		}
		row := s.Body.Rows[rr.Index]
		if p.In(rr.Help) && p.In(l.RowsViewport) {
			return helpTip(bounds, body, rr.Help, row.Help), true
		}
		full, cut := rowFull(title, body, row, rr)
		width := rr.Rect.Dx()
		onButton := false
		if row.Kind == popupDirection {
			if !row.placed() {
				bounds = bounds.Intersect(l.RowsViewport)
			}
			for k, b := range rr.Buttons {
				if p.In(b) {
					full, cut, width, onButton = row.Choices[k].Label, true, min(max(width, popupTipW), bounds.Dx()), true
				}
			}
		}
		if row.PortraitTip != "" && row.Portrait != nil && p.In(rowPortraitBox(rr)) {
			// The portrait's own tip reads whole under the portrait.
			width = min(max(width, popupTipW), bounds.Dx())
			return layoutTip(bounds, body, rr.Rect, row.PortraitTip, nil, tipShape{Width: width}), true
		}
		if row.Tip != "" && row.Edit == nil && !onButton {
			// A row's own tip reads whole in a box at least popupTipW wide;
			// a pad's button names its choice instead.
			full, cut, width = row.Tip, true, min(max(width, popupTipW), bounds.Dx())
		}
		if !cut {
			return popupTip{}, false
		}
		how := tipShape{Width: width, Pull: row.Tip != "", Clip: row.Kind == popupDirection}
		return layoutTip(bounds, body, rr.Rect, full, nil, how), true
	}
	if tip, ok := canvasTipAt(bounds, body, s.Canvas, l, p); ok {
		return tip, true
	}
	g := s.Grid
	if g == nil {
		return popupTip{}, false
	}
	at, full := image.Rectangle{}, ""
	for c, r := range l.GridColHeads {
		if g.HeadH > 0 && p.In(r) {
			at, full = r, g.ColHeads[c]
		}
	}
	if full == "" {
		return popupTip{}, false
	}
	return layoutTip(bounds, body, at, full, nil, tipShape{Fit: true, Pull: true}), true
}

// rowPictureBoxes are where shown row rr's pictures stand: its portrait in
// the label's column at the control's height (empty without one), then its
// icons after its value, each square at the control's height, as many as
// the control holds.
func rowPictureBoxes(body *text.Font, rr popupRowRect, row popupRow) (portrait image.Rectangle, icons []image.Rectangle) {
	if rr.Control.Empty() {
		return image.Rectangle{}, nil
	}
	if row.Portrait != nil {
		portrait = rowPortraitBox(rr)
	}
	x := rr.Control.Min.X + 2
	if body != nil && row.Value != "" {
		x += body.Advance(row.Value) + popupPad
	}
	side := rr.Control.Dy()
	for range row.Icons {
		if x+side > rr.Control.Max.X {
			break
		}
		icons = append(icons, image.Rect(x, rr.Control.Min.Y, x+side, rr.Control.Max.Y))
		x += side + 2
	}
	return portrait, icons
}

// rowPortraitBox is where a row's portrait stands: the label's column at the
// control's height.
func rowPortraitBox(rr popupRowRect) image.Rectangle {
	return image.Rect(rr.Rect.Min.X+2, rr.Control.Min.Y, rr.Label.Max.X-2, rr.Control.Max.Y)
}

// drawRowPictures draws a row's portrait and icons at rowPictureBoxes, each
// by the editor's picture element.
func drawRowPictures(dst *image.RGBA, body *text.Font, l popupLayout, rr popupRowRect, row popupRow) {
	portrait, icons := rowPictureBoxes(body, rr, row)
	drawEditorPicture(dst, editorPicture{Pic: row.Portrait, Box: portrait, Clip: l.Rect})
	for k, box := range icons {
		drawEditorPicture(dst, editorPicture{Pic: row.Icons[k], Box: box, Clip: rr.Control.Intersect(l.Rect)})
	}
}

// headLines are the lines a flat column head s takes in a column width
// pixels wide: one, or two when s is wider and breaks at a space with the
// first line as long as fits.
func headLines(body *text.Font, s string, width int) []string {
	if body == nil || body.Advance(s) <= width {
		return []string{s}
	}
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == ' ' && body.Advance(s[:i]) <= width {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

// drawPopupGrid paints a grid's shown cells, each a push button holding its
// picture, sunk and framed while chosen and ticked while marked, and the
// grid's bar.
func drawPopupGrid(dst *image.RGBA, body *text.Font, frames []*image.RGBA, g *popupGrid, l popupLayout, ink popupInk, pointer image.Point, pointerOK bool, press ...*buttonPress) {
	var held *buttonPress
	if len(press) > 0 {
		held = press[0]
	}
	head := func(r image.Rectangle, s string, centre bool) {
		if body == nil || s == "" {
			return
		}
		lines, high := []string{s}, fontHeight(body, 15)
		if centre && r.Dy() >= 2*high {
			lines = headLines(body, s, r.Dx()-4)
		}
		y := r.Min.Y + (r.Dy()-len(lines)*high)/2
		for i, s := range lines {
			s = editorClip(body, s, r.Dx()-4)
			x := r.Min.X + 2
			if centre {
				x = r.Min.X + (r.Dx()-body.Advance(s))/2
			}
			body.Draw(dst.SubImage(r).(*image.RGBA), s, x, y+i*high, ink.Label)
		}
	}
	for k, r := range l.GridRowHeads {
		label := ""
		if len(l.GridHeadText) > 0 {
			label = l.GridHeadText[k]
		} else {
			label = g.RowHeads[l.GridTop+k]
		}
		head(r, label, false)
	}
	for c, r := range l.GridColHeads {
		if g.HeadH > 0 && !g.HeadFlat && body != nil {
			drawTextUp(dst, body, editorClip(body, g.ColHeads[c], r.Dy()-4), r, ink.Label)
			continue
		}
		head(r, g.ColHeads[c], true)
	}
	for i, c := range l.GridCells {
		cell := g.Cells[l.gridIndex(i)]
		if cell.Inert {
			drawFrame(dst, panelFrame(c, ink.TitleGround, ink.Edge))
			continue
		}
		label, raise := "", 0
		if cell.Icon == nil && cell.Label != "" {
			label, raise = cell.Label, labelRaise(body)
		}
		drawPushButton(dst, body, pushButton{Press: held, Rect: c, Label: label, Literal: true, LabelRaise: raise, Pressed: cell.Pressed, Inside: cell.Pressed, Hover: pointerOK && pointer.In(c)})
		if cell.Pressed {
			drawFrame(dst, panelFrame(c, ink.TitleGround, ink.Edge))
		}
		pic := c
		if g.CaptionH > 0 {
			pic.Max.Y -= g.CaptionH
			if body != nil && cell.Caption != "" {
				s := editorClip(body, cell.Caption, c.Dx()-4)
				body.Draw(dst.SubImage(c).(*image.RGBA), s, c.Min.X+(c.Dx()-body.Advance(s))/2, pic.Max.Y-2, ink.Label)
			}
		}
		box := pic.Inset(4)
		if cell.Pressed {
			box = box.Add(image.Pt(1, 1))
		}
		drawEditorPicture(dst, editorPicture{Pic: cell.Icon, Box: box, Clip: c.Inset(2)})
		if cell.Ticked {
			drawGridTick(dst, c, ink.Edge)
		}
	}
	if !l.GridBar.Empty() {
		bar := vScrollBar{Rect: l.GridBar, Pos: l.GridTop, Count: g.totalRows(l.GridCols) - l.GridRows + 1}
		drawVScrollBar(dst, frames, bar.withPointer(pointer, pointerOK))
	}
}

// drawTextUp draws s with the body font turned a quarter turn
// counter-clockwise inside r: it reads from bottom to top, starts 2 pixels
// above r's bottom and is centred across r. It is the one way the editor
// draws upright text. The line is drawn into its own picture in its own
// capture window, so no smoothed-text overlay redraws its glyphs unturned.
func drawTextUp(dst *image.RGBA, body *text.Font, s string, r image.Rectangle, ink color.RGBA) {
	if body == nil || s == "" {
		return
	}
	w, h := body.Advance(s), fontHeight(body, 15)
	if w <= 0 {
		return
	}
	line := image.NewRGBA(image.Rect(0, 0, w, h))
	text.Record(func() { body.Draw(line, s, 0, 0, ink) })
	x0, y0, clip := r.Min.X+(r.Dx()-h)/2, r.Max.Y-2-w, r.Intersect(dst.Bounds())
	for y := range h {
		for x := range w {
			c, at := line.RGBAAt(x, y), image.Pt(x0+y, y0+w-1-x)
			if c.A != 0 && at.In(clip) {
				dst.SetRGBA(at.X, at.Y, c)
			}
		}
	}
}

// drawGridTick draws a tick in the top-right corner of cell c.
func drawGridTick(dst *image.RGBA, c image.Rectangle, ink color.RGBA) {
	x, y := c.Max.X-13, c.Min.Y+6
	for i := 0; i < 10; i++ {
		py := y - i/2 + 2
		if i < 3 {
			py = y + i
		}
		for d := 0; d < 2; d++ {
			if p := image.Pt(x+i, py+d); p.In(dst.Bounds()) {
				dst.SetRGBA(p.X, p.Y, ink)
			}
		}
	}
}

// drawPopupGrip marks the bottom-right corner where a drag resizes the popup:
// four diagonal strokes, two pixels thick, inside the grip square.
func drawPopupGrip(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	for k := 1; k <= 4; k++ {
		for i := 0; i < 4*k-2; i++ {
			for d := 0; d < 2; d++ {
				p := image.Pt(r.Max.X-3-4*k+i+d+3, r.Max.Y-3-i)
				if p.In(dst.Bounds()) && p.X < r.Max.X-1 {
					dst.SetRGBA(p.X, p.Y, c)
				}
			}
		}
	}
}
