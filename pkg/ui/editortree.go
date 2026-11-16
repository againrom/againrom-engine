package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// An editor window is a tree of elements. The root is the window; its
// children are the title, the tabs, the body of rows, the card, the grid,
// the canvas, the list, the notes, the pick bar and the button row, in
// that order, each present only when the window has it. The one layout
// pass (layoutWindow) reads the tree; popupSpec is the flat form the
// window builders still write, turned into a tree by specTree.

// uiKind is what an element of an editor window is.
type uiKind uint8

const (
	uiWindowKind uiKind = iota
	uiTitleKind
	uiTabsKind
	uiBodyKind
	uiCardKind
	uiGridKind
	uiCanvasKind
	uiListKind
	uiNotesKind
	uiPickKind
	uiButtonsKind
)

// uiNode is one element: its kind, its word (the caption it shows, when it
// has one), its children and the data its kind owns.
type uiNode struct {
	Kind     uiKind
	Word     string
	Children []uiNode
	Data     any
}

// uiPlace is the window's placement, kept on the root: the corner and width
// the store gives it, the bounds and room it lies in, the size the user
// dragged it to and the card a row drag holds.
type uiPlace struct {
	At       image.Point
	Width    int
	Bounds   image.Rectangle
	Room     image.Rectangle
	Slack    bool
	Short    bool
	Size     image.Point
	MinWidth int
	Fit      bool
	Drag     *popupRowDrag
}

// uiTitle is the title row: its words, the (?) mark's help and the close
// button's caption.
type uiTitle struct {
	Text, Help, Close string
}

// uiTabs is the tab row: its captions, the active tab and whether the
// active tab draws as a switch's chosen state.
type uiTabs struct {
	Captions []string
	Active   int
	Switch   bool
}

// uiBody is the rows: their columns and each column's first row, the label
// column's width, and the body's scroll state (the first row shown and
// how many rows show at most, and the first row a paged window shows).
type uiBody struct {
	Rows       []popupRow
	Columns    int
	ColumnTops []int
	LabelWidth int
	Top        int
	MaxRows    int
	Page       int
}

// uiCard is the character card: the subject, its font, the column beside
// the rows, its scale, whether it stands under the rows and the lines of
// a band under the rows that scroll with them.
type uiCard struct {
	Subject   *PanelSubject
	Font      *text.Font
	Aside     int
	Scale     int
	Below     bool
	BandLines int
}

// uiList is the list box under the rows: its model, its rows, the fewest
// rows a sized window keeps, its text, the column kept free beside it and
// its columns.
type uiList struct {
	Picker  *Picker
	Rows    int
	Least   int
	Text    listRowText
	Aside   int
	Columns []listColumn
}

// uiNotes is the note lines over the button row and their ink.
type uiNotes struct {
	Lines []string
	Ink   color.RGBA
}

// uiPick is the pick-on-map button's caption and whether it is armed.
type uiPick struct {
	Label string
	Armed bool
}

// uiButtons is the button row: its captions, which are off, the least
// button width, the focused button and the switch button at its left end.
type uiButtons struct {
	Captions []string
	Disabled []bool
	Least    int
	Focus    *int
	Switch   string
}

// uiWindow is a window's tree read into one value per element kind, the
// form the layout pass, the painter and the hit test read. An element the
// tree does not hold reads as its zero value.
type uiWindow struct {
	Place   uiPlace
	Title   uiTitle
	Tabs    uiTabs
	Body    uiBody
	Card    uiCard
	Grid    *popupGrid
	Canvas  *popupCanvas
	List    uiList
	Notes   uiNotes
	Pick    uiPick
	Buttons uiButtons
}

// specTree is the tree of a window the builders wrote as a popupSpec.
func specTree(s popupSpec) uiNode {
	root := uiNode{Kind: uiWindowKind, Data: uiPlace{
		At: s.At, Width: s.Width, Bounds: s.Bounds, Room: s.Room, Slack: s.Slack, Short: s.Short,
		Size: s.Size, MinWidth: s.MinWidth, Fit: s.Fit, Drag: s.Drag,
	}}
	add := func(n uiNode) { root.Children = append(root.Children, n) }
	add(uiNode{Kind: uiTitleKind, Word: s.Title, Data: uiTitle{Help: s.TitleHelp, Close: s.CloseLabel}})
	if len(s.Tabs) > 0 {
		add(uiNode{Kind: uiTabsKind, Data: uiTabs{Captions: s.Tabs, Active: s.ActiveTab, Switch: s.TabSwitch}})
	}
	add(uiNode{Kind: uiBodyKind, Data: uiBody{
		Rows: s.Rows, Columns: s.Columns, ColumnTops: s.ColumnTops, LabelWidth: s.LabelWidth,
		Top: s.Top, MaxRows: s.MaxRows, Page: s.Page,
	}})
	if s.Card != nil || s.CardFont != nil || s.CardAside != 0 || s.CardScale != 0 || s.CardBelow || s.BandLines != 0 {
		add(uiNode{Kind: uiCardKind, Data: uiCard{
			Subject: s.Card, Font: s.CardFont, Aside: s.CardAside, Scale: s.CardScale,
			Below: s.CardBelow, BandLines: s.BandLines,
		}})
	}
	if s.Grid != nil {
		add(uiNode{Kind: uiGridKind, Data: s.Grid})
	}
	if s.Canvas != nil {
		add(uiNode{Kind: uiCanvasKind, Data: s.Canvas})
	}
	if s.List != nil || s.ListRows != 0 || s.ListLeast != 0 || s.ListText != nil || s.ListAside != 0 || len(s.ListColumns) > 0 {
		add(uiNode{Kind: uiListKind, Data: uiList{
			Picker: s.List, Rows: s.ListRows, Least: s.ListLeast, Text: s.ListText,
			Aside: s.ListAside, Columns: s.ListColumns,
		}})
	}
	if len(s.Notes) > 0 || s.NoteInk != (color.RGBA{}) {
		add(uiNode{Kind: uiNotesKind, Data: uiNotes{Lines: s.Notes, Ink: s.NoteInk}})
	}
	if s.PickLabel != "" || s.PickArmed {
		add(uiNode{Kind: uiPickKind, Word: s.PickLabel, Data: uiPick{Armed: s.PickArmed}})
	}
	add(uiNode{Kind: uiButtonsKind, Data: uiButtons{
		Captions: s.Buttons, Disabled: s.Disabled, Least: s.ButtonW, Focus: s.ButtonFocus, Switch: s.SwitchButton,
	}})
	return root
}

// windowOf reads a window's tree into its uiWindow.
func windowOf(root uiNode) uiWindow {
	var w uiWindow
	w.Place, _ = root.Data.(uiPlace)
	for _, n := range root.Children {
		switch n.Kind {
		case uiTitleKind:
			w.Title, _ = n.Data.(uiTitle)
			w.Title.Text = n.Word
		case uiTabsKind:
			w.Tabs, _ = n.Data.(uiTabs)
		case uiBodyKind:
			w.Body, _ = n.Data.(uiBody)
		case uiCardKind:
			w.Card, _ = n.Data.(uiCard)
		case uiGridKind:
			w.Grid, _ = n.Data.(*popupGrid)
		case uiCanvasKind:
			w.Canvas, _ = n.Data.(*popupCanvas)
		case uiListKind:
			w.List, _ = n.Data.(uiList)
		case uiNotesKind:
			w.Notes, _ = n.Data.(uiNotes)
		case uiPickKind:
			w.Pick, _ = n.Data.(uiPick)
			w.Pick.Label = n.Word
		case uiButtonsKind:
			w.Buttons, _ = n.Data.(uiButtons)
		}
	}
	return w
}

// specWindow is the window a popupSpec describes, read through its tree.
func specWindow(s popupSpec) uiWindow { return windowOf(specTree(s)) }
