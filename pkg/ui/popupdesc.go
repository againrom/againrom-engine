package ui

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// screenFiles holds the editor windows kept as descriptions.
//
//go:embed screens/*.json
var screenFiles embed.FS

// popupDesc is one editor window as data: the popupSpec it builds, with a
// word key for every caption and a binding name for every value the map or
// the editor gives. Fit widens the label column to its labels or sets a
// label over its field, as the plain view does. Card stands the person's
// card and figure in their own column right of the field columns, as the
// unit window does. Sample is an example state in words and flags, for a
// mockup without a map.
type popupDesc struct {
	Window     string            `json:"window"`
	Title      string            `json:"title,omitempty"`
	TitleArgs  []string          `json:"titleArgs,omitempty"`
	TitleHelp  string            `json:"titleHelp,omitempty"`
	Width      int               `json:"width,omitempty"`
	LabelWidth int               `json:"labelWidth,omitempty"`
	MinWidth   int               `json:"minWidth,omitempty"`
	ButtonW    int               `json:"buttonWidth,omitempty"`
	Columns    int               `json:"columns,omitempty"`
	Fit        bool              `json:"fit,omitempty"`
	Card       bool              `json:"card,omitempty"`
	Tabs       []string          `json:"tabs,omitempty"`
	Rows       []popupDescRow    `json:"rows"`
	Buttons    []popupDescButton `json:"buttons,omitempty"`
	Sample     *popupScope       `json:"sample,omitempty"`
	Preview    *popupPreview     `json:"preview,omitempty"`
	// TitleHelpMore are word keys whose words follow the title help, each
	// after a space: the keys line every prompt states once.
	TitleHelpMore []string `json:"titleHelpMore,omitempty"`
	// Question makes the window the question element: the editor's one
	// question builder lays it out from its title, help, rows, list, grid,
	// notes and buttons.
	Question bool `json:"question,omitempty"`
	// Notes are the window's state lines, written in the ink role NoteInk.
	Notes   []popupDescNote `json:"notes,omitempty"`
	NoteInk string          `json:"noteInk,omitempty"`
	// List is a list of rows the window picks one of; Grid a grid of cells.
	List *popupDescList `json:"list,omitempty"`
	Grid *popupDescGrid `json:"grid,omitempty"`
	// Tab is the shown tab of Tabs, from 0.
	Tab int `json:"tab,omitempty"`
	// Canvas is a sheet the editor draws below the grid (the spell book),
	// shown in a mockup as the picture its Picture names.
	Canvas *popupDescCanvas `json:"canvas,omitempty"`
	// Switch is the check at the button row's left end: its word, marked
	// while the flag On names is set.
	Switch *popupDescSwitch `json:"switch,omitempty"`
	Panels *popupDescPanels `json:"panels,omitempty"`
}

type popupPreview struct {
	Width       int  `json:"width,omitempty"`
	Top         int  `json:"top,omitempty"`
	HoverRow    *int `json:"hoverRow,omitempty"`
	HoverList   bool `json:"hoverList,omitempty"`
	HoverColumn int  `json:"hoverColumn,omitempty"`
}

// popupDescCanvas is a canvas shown as one picture.
type popupDescCanvas struct {
	Picture string `json:"picture"`
}

type popupDescSwitch struct {
	Word string `json:"word"`
	On   string `json:"on,omitempty"`
}

// popupDescNote is a note line: a word with its verbs filled, or without a
// word the text of binding Bind; once, or with Each once for every item of
// a list binding.
type popupDescNote struct {
	popupDescButton
	Each string `json:"each,omitempty"`
	Bind string `json:"bind,omitempty"`
}

// popupDescList is a list: a row for every item of list Each, its text the
// word Word with the verbs Args fill, or without a word the text of binding
// Bind, after the picture its Icon binding names; the item whose Chosen
// flag is set is the selection. Rows is how many rows show. Aside is the
// picture beside the list.
type popupDescList struct {
	Each    string                `json:"each"`
	Word    string                `json:"word,omitempty"`
	Bind    string                `json:"bind,omitempty"`
	Args    []string              `json:"args,omitempty"`
	Icon    string                `json:"icon,omitempty"`
	Chosen  string                `json:"chosen,omitempty"`
	Inert   string                `json:"inert,omitempty"`
	Rows    int                   `json:"rows,omitempty"`
	Aside   *popupDescAside       `json:"aside,omitempty"`
	Columns []popupDescListColumn `json:"columns,omitempty"`
}

// popupDescAside is the picture element beside a list: the picture its
// Picture binding names, in a column Width pixels wide right of the list,
// as high as the list.
type popupDescAside struct {
	Picture string `json:"picture"`
	Width   int    `json:"width"`
}

// popupDescListColumn binds one cell per item and names its column's word.
type popupDescListColumn struct {
	Head  string   `json:"head"`
	Share int      `json:"share"`
	Bind  string   `json:"bind,omitempty"`
	Word  string   `json:"word,omitempty"`
	Args  []string `json:"args,omitempty"`
}

// popupDescGrid is the one grid form: a cell for every item of list Each.
// Without Column each item is one cell, the grid's own popupDescGridCell,
// and the cells flow into the columns the width holds, Rows of them shown
// (zero every row that fits), or one row headed by RowHeads' word when it
// is given. With Column each item is a column headed by
// its ColHead binding, the cells of Column top to bottom, each grid row
// headed by the word of the same place in RowHeads. Side is a cell's side,
// zero gridCellSide; High the height of a row whose cells show a picture.
type popupDescGrid struct {
	popupDescGridCell
	Each     string              `json:"each"`
	Rows     int                 `json:"rows,omitempty"`
	ColHead  string              `json:"colHead,omitempty"`
	RowHeads []string            `json:"rowHeads,omitempty"`
	Column   []popupDescGridCell `json:"column,omitempty"`
	Side     int                 `json:"side,omitempty"`
	High     int                 `json:"high,omitempty"`
}

// popupDescGridCell is one cell of an item: the picture its Picture binding
// names, the text its Label binding gives when no picture shows, its Tip
// binding's text on hover; pressed while the Chosen flag is set, ticked
// while the Ticked flag is, taking no press while the Inert flag is.
type popupDescGridCell struct {
	Picture string `json:"picture,omitempty"`
	Label   string `json:"label,omitempty"`
	Tip     string `json:"tip,omitempty"`
	Chosen  string `json:"chosen,omitempty"`
	Ticked  string `json:"ticked,omitempty"`
	Inert   string `json:"inert,omitempty"`
}

// popupDescRow is one row, or with Each a block repeated for every item of
// a list binding. When names the flag the row needs ("!name": needs it
// unset). Label is a word key; LabelBind, Value and HintBind name bindings.
// Wrap marks a text row the window wraps to its width. Fold makes a heading
// a block that opens and closes: open while the flag it names is set.
type popupDescRow struct {
	Kind      string            `json:"kind,omitempty"`
	When      string            `json:"when,omitempty"`
	Each      string            `json:"each,omitempty"`
	Rows      []popupDescRow    `json:"rows,omitempty"`
	Label     string            `json:"label,omitempty"`
	LabelBind string            `json:"labelBind,omitempty"`
	Indent    int               `json:"indent,omitempty"`
	Fold      string            `json:"fold,omitempty"`
	Value     string            `json:"value,omitempty"`
	Hint      string            `json:"hint,omitempty"`
	HintBind  string            `json:"hintBind,omitempty"`
	Help      string            `json:"help,omitempty"`
	HelpBind  string            `json:"helpBind,omitempty"`
	Ink       string            `json:"ink,omitempty"`
	Column    int               `json:"column,omitempty"`
	Lines     int               `json:"lines,omitempty"`
	Wrap      bool              `json:"wrap,omitempty"`
	Field     string            `json:"field,omitempty"`
	Buttons   []popupDescButton `json:"buttons,omitempty"`
	// Focus marks the row the keyboard is in.
	Focus bool `json:"focus,omitempty"`
	// Picture names the binding of the picture a text row shows in its
	// label column, Lines rows high (three when unset); Icons names the
	// binding of the pictures drawn after the value. A binding's text is
	// picture names separated by spaces, which the editor resolves.
	Picture string `json:"picture,omitempty"`
	Icons   string `json:"icons,omitempty"`
	// LabelArgs fill the label word's verbs from bindings, as titleArgs do.
	LabelArgs []string `json:"labelArgs,omitempty"`
	// Struck names the flag that shows a card struck out.
	Struck string `json:"struck,omitempty"`
	// Whole makes a multiline row as tall as its text wrapped to its width.
	Whole bool `json:"whole,omitempty"`
}

// popupDescButton is a button: its caption's word key, the row action it
// runs, the flag it needs and, for a window button, the flag that disables
// it.
type popupDescButton struct {
	Word string `json:"word"`
	Act  string `json:"act,omitempty"`
	When string `json:"when,omitempty"`
	Off  string `json:"off,omitempty"`
	// Args fill the caption's verbs from bindings, as titleArgs do.
	Args []string `json:"args,omitempty"`
	// Armed names the flag that draws a row button pressed in.
	Armed string `json:"armed,omitempty"`
	// Tip is the word key of what a row button does, its hover tip.
	Tip string `json:"tip,omitempty"`
}

// popupKindNames names each popupRowKind in a description.
var popupKindNames = [...]string{
	popupText: "text", popupHeading: "heading", popupEdit: "edit", popupChoice: "choice", popupPick: "pick",
	popupLink: "link", popupSwitch: "switch", popupMultiline: "multiline", popupCard: "card", popupDirection: "direction",
}

// popupFileKind names the row that picks a file or a folder: its value is
// the chosen name, never typed, and its button opens the system's file
// dialog. It is laid out as a pick row.
const popupFileKind = "file"

func popupKindNamed(name string) (popupRowKind, bool) {
	if name == popupFileKind {
		return popupPick, true
	}
	for k, n := range popupKindNames {
		if n == name {
			return popupRowKind(k), true
		}
	}
	return 0, false
}

// popupBinds answers a description's bindings: whether a flag condition
// holds, a value's text, a value's word key, an editor field and a list.
// The second result is false for a name it does not know.
type popupBinds interface {
	when(cond string) (bool, bool)
	value(name string) (string, bool)
	key(name string) (string, bool)
	field(name string) (int, bool)
	list(name string) ([]popupBinds, bool)
}

// popupScope is a set of bindings: the editor's state for a window, or a
// description's sample. Fields is the editor's and never written.
type popupScope struct {
	Values map[string]string       `json:"values,omitempty"`
	Words  map[string]string       `json:"words,omitempty"`
	Flags  []string                `json:"flags,omitempty"`
	Lists  map[string][]popupScope `json:"lists,omitempty"`
	Fields map[string]int          `json:"-"`
}

func (s *popupScope) when(cond string) (bool, bool) {
	name, unset := strings.CutPrefix(cond, "!")
	for _, f := range s.Flags {
		if f == name {
			return !unset, true
		}
	}
	return false, false
}

func (s *popupScope) value(name string) (string, bool) { v, ok := s.Values[name]; return v, ok }
func (s *popupScope) key(name string) (string, bool)   { v, ok := s.Words[name]; return v, ok }
func (s *popupScope) field(name string) (int, bool)    { v, ok := s.Fields[name]; return v, ok }

func (s *popupScope) list(name string) ([]popupBinds, bool) {
	items, ok := s.Lists[name]
	out := make([]popupBinds, len(items))
	for i := range items {
		out[i] = &items[i]
	}
	return out, ok
}

func (s *popupScope) flag(name string) { s.Flags = append(s.Flags, name) }

// popupChain is the bindings in reach of a row, innermost last: a name is
// looked up from the row's list item outwards.
type popupChain []popupBinds

func (c popupChain) when(cond string) bool {
	if cond == "" {
		return true
	}
	for i := len(c) - 1; i >= 0; i-- {
		if on, ok := c[i].when(cond); ok {
			return on
		}
	}
	return strings.HasPrefix(cond, "!")
}

// text is the binding's text, or its word key's word.
func (c popupChain) text(name string, word func(string) string) string {
	for i := len(c) - 1; i >= 0; i-- {
		if v, ok := c[i].value(name); ok {
			return v
		}
		if k, ok := c[i].key(name); ok {
			return word(k)
		}
	}
	return ""
}

func (c popupChain) field(name string) int {
	for i := len(c) - 1; i >= 0; i-- {
		if f, ok := c[i].field(name); ok {
			return f
		}
	}
	return -1
}

func (c popupChain) list(name string) []popupBinds {
	for i := len(c) - 1; i >= 0; i-- {
		if items, ok := c[i].list(name); ok {
			return items
		}
	}
	return nil
}

// popupMade is one built row with what the editor keeps beside it: the
// field it edits (-1 none) and its buttons' actions. Picture and Icons name the pictures the
// editor puts in its label column and after its value.
type popupMade struct {
	Row     popupRow
	Field   int
	Acts    []string
	Wrap    bool
	Picture string
	Icons   []string
}

// build is the description as a popupSpec under the bindings b, every word
// key through word, and its rows with their fields and actions. It places
// nothing: layoutPopup lays the spec out.
func (d popupDesc) build(b popupBinds, word func(string) string) (popupSpec, []popupMade) {
	c := popupChain{b}
	s := popupSpec{Width: d.Width, LabelWidth: d.LabelWidth, MinWidth: d.MinWidth, ButtonW: d.ButtonW, Columns: d.Columns, Fit: d.Fit}
	if d.Card {
		// The card's column, at the card's width until a figure widens it;
		// the column shows once the window is given a person's card.
		s.CardAside = compactPanelW
	}
	if d.Title != "" {
		s.Title = c.format(d.Title, d.TitleArgs, word)
	}
	if d.TitleHelp != "" {
		s.TitleHelp = word(d.TitleHelp)
	}
	for _, k := range d.TitleHelpMore {
		s.TitleHelp = strings.TrimSpace(s.TitleHelp + " " + word(k))
	}
	for _, t := range d.Tabs {
		s.Tabs = append(s.Tabs, word(t))
	}
	if len(s.Tabs) > 0 {
		s.ActiveTab, s.TabSwitch = min(d.Tab, len(s.Tabs)-1), true
	}
	var made []popupMade
	c.rows(d.Rows, word, &made)
	for _, m := range made {
		s.Rows = append(s.Rows, m.Row)
	}
	if d.Switch != nil {
		s.SwitchButton = markWord(d.Switch.On != "" && c.when(d.Switch.On), word(d.Switch.Word))
	}
	for _, bt := range d.Buttons {
		if c.when(bt.When) {
			s.Buttons = append(s.Buttons, c.format(bt.Word, bt.Args, word))
			s.Disabled = append(s.Disabled, bt.Off != "" && c.when(bt.Off))
		}
	}
	for _, n := range d.Notes {
		items := []popupChain{c}
		if n.Each != "" {
			items = nil
			for _, item := range c.list(n.Each) {
				items = append(items, append(c[:len(c):len(c)], item))
			}
		}
		for _, in := range items {
			switch {
			case !in.when(n.When):
			case n.Word == "":
				s.Notes = append(s.Notes, in.text(n.Bind, word))
			default:
				s.Notes = append(s.Notes, in.format(n.Word, n.Args, word))
			}
		}
	}
	if d.NoteInk != "" {
		s.NoteInk = inspectionRoleColor(d.NoteInk)
	}
	if d.List != nil {
		s.List, s.ListRows, s.ListAside, _ = c.listBox(d.List, word, nil)
		for _, col := range d.List.Columns {
			s.ListColumns = append(s.ListColumns, listColumn{Head: word(col.Head), Share: col.Share})
		}
	}
	s.Grid, _ = c.grid(d.Grid, word)
	return s, made
}

// listBox is the description's list under the bindings: its picker with
// each row's picture from picture (none when nil), the rows it shows, the
// column its picture element keeps beside it, and the name of the picture
// beside it, which the editor resolves.
func (c popupChain) listBox(l *popupDescList, word func(string) string, picture func(string) *image.RGBA) (*Picker, int, int, string) {
	var rows []PickerRow
	chosen := 0
	for i, item := range c.list(l.Each) {
		in := append(c[:len(c):len(c)], item)
		row := PickerRow{Text: in.text(l.Bind, word), Choosable: l.Inert == "" || !in.when(l.Inert)}
		if l.Word != "" {
			row.Text = in.format(l.Word, l.Args, word)
		}
		if len(l.Columns) > 0 {
			var cells []string
			for _, col := range l.Columns {
				cell := in.text(col.Bind, word)
				if col.Word != "" {
					cell = in.format(col.Word, col.Args, word)
				}
				cells = append(cells, cell)
			}
			row.Text = strings.Join(cells, "\t")
		}
		if l.Icon != "" {
			row.IconCell = true
			if picture != nil {
				row.Icon = picture(in.text(l.Icon, func(string) string { return "" }))
			}
		}
		rows = append(rows, row)
		if l.Chosen != "" && in.when(l.Chosen) {
			chosen = i
		}
	}
	shown := len(rows)
	if l.Rows > 0 {
		shown = min(l.Rows, len(rows))
	}
	m := NewPicker(rows).SetWindow(shown)
	m.Select(chosen)
	aside, name := 0, ""
	if l.Aside != nil {
		aside, name = l.Aside.Width+popupPad, c.text(l.Aside.Picture, func(string) string { return "" })
	}
	return m, shown, aside, name
}

// grid is the description's grid under the bindings, nil for none, and the
// name of each cell's picture in the cells' order, which the editor
// resolves ("" for none). A grid of columns ticks its chosen cells; its
// heads' sizes are the fonts' and left unset.
func (c popupChain) grid(g *popupDescGrid, word func(string) string) (*popupGrid, []string) {
	if g == nil {
		return nil, nil
	}
	items := c.list(g.Each)
	out := &popupGrid{Rows: g.Rows, Side: g.Side}
	column := []popupDescGridCell{g.popupDescGridCell}
	if len(g.Column) > 0 {
		column = g.Column
		out = &popupGrid{Rows: len(column), Cols: len(items), Side: g.Side, HeadFlat: true}
	} else if len(g.RowHeads) > 0 {
		out.Cols = len(items)
	}
	for _, h := range g.RowHeads {
		out.RowHeads = append(out.RowHeads, word(h))
	}
	var names []string
	for k, cell := range column {
		for _, item := range items {
			in := append(c[:len(c):len(c)], item)
			if len(g.Column) > 0 && k == 0 && g.ColHead != "" {
				out.ColHeads = append(out.ColHeads, in.text(g.ColHead, word))
			}
			on := cell.Chosen != "" && in.when(cell.Chosen)
			ticked := on && len(g.Column) > 0 || cell.Ticked != "" && in.when(cell.Ticked)
			made := popupGridCell{Pressed: on, Ticked: ticked, Inert: cell.Inert != "" && in.when(cell.Inert)}
			if cell.Tip != "" {
				made.Tip = in.text(cell.Tip, word)
			}
			name := ""
			if cell.Picture != "" {
				name = in.text(cell.Picture, func(string) string { return "" })
			}
			if name == "" && cell.Label != "" {
				made.Label = in.text(cell.Label, word)
			}
			out.Cells = append(out.Cells, made)
			names = append(names, name)
		}
	}
	return out, names
}

// format is word key's word with its verbs filled from the bindings args,
// a binding whose text is a whole number filling it as a number.
func (c popupChain) format(key string, args []string, word func(string) string) string {
	if len(args) == 0 {
		return word(key)
	}
	values, texts := make([]any, len(args)), make([]any, len(args))
	for i, a := range args {
		t := c.text(a, word)
		values[i], texts[i] = t, t
		if n, err := strconv.Atoi(t); err == nil {
			values[i] = n
		}
	}
	// A number a %s verb takes stays its text.
	if out := fmt.Sprintf(word(key), values...); !strings.Contains(out, "%!") {
		return out
	}
	return fmt.Sprintf(word(key), texts...)
}

func (c popupChain) rows(rows []popupDescRow, word func(string) string, made *[]popupMade) {
	for _, r := range rows {
		if !c.when(r.When) {
			continue
		}
		if r.Each == "" {
			// A card holds the rows under it: its Span counts those made.
			head := len(*made)
			*made = append(*made, c.row(r, word))
			c.rows(r.Rows, word, made)
			(*made)[head].Row.Span = len(*made) - head - 1
			continue
		}
		for _, item := range c.list(r.Each) {
			append(c[:len(c):len(c)], item).rows(r.Rows, word, made)
		}
	}
}

func (c popupChain) row(r popupDescRow, word func(string) string) popupMade {
	kind, _ := popupKindNamed(r.Kind)
	label := ""
	if r.Label != "" {
		label = c.format(r.Label, r.LabelArgs, word)
	} else if r.LabelBind != "" {
		label = c.text(r.LabelBind, word)
	}
	row := popupRow{Kind: kind, Label: strings.Repeat(" ", r.Indent) + label, Column: r.Column, Lines: r.Lines, Focus: r.Focus}
	if r.Value != "" {
		row.Value = c.text(r.Value, word)
	}
	if r.Hint != "" {
		row.Hint = word(r.Hint)
	} else if r.HintBind != "" {
		row.Hint = c.text(r.HintBind, word)
	}
	if r.Fold != "" {
		row.Fold = blockRow("", c.when(r.Fold)).Fold
	}
	if r.Help != "" {
		row.Help = word(r.Help)
	} else if r.HelpBind != "" {
		row.Help = c.text(r.HelpBind, word)
	}
	if r.Ink != "" || r.Wrap {
		// A wrapped text row takes its role's ink as the window's text rows do.
		row.Ink = inspectionRoleColor(r.Ink)
	}
	row.Struck, row.Whole = r.Struck != "" && c.when(r.Struck), r.Whole
	m := popupMade{Field: -1, Wrap: r.Wrap}
	if r.Field != "" {
		m.Field = c.field(r.Field)
	}
	for _, bt := range r.Buttons {
		if c.when(bt.When) {
			tip := ""
			if bt.Tip != "" {
				tip = word(bt.Tip)
			}
			row.Buttons = append(row.Buttons, c.format(bt.Word, bt.Args, word))
			row.Armed = append(row.Armed, bt.Armed != "" && c.when(bt.Armed))
			row.ButtonTips = append(row.ButtonTips, tip)
			m.Acts = append(m.Acts, bt.Act)
		}
	}
	if !slices.Contains(row.Armed, true) {
		row.Armed = nil
	}
	if !slices.ContainsFunc(row.ButtonTips, func(t string) bool { return t != "" }) {
		row.ButtonTips = nil
	}
	if r.Kind == popupFileKind {
		// The file row's button and its empty value say the same on every
		// page that picks a file; a row may name its own.
		if len(row.Buttons) == 0 {
			row.Buttons, m.Acts = []string{word("editor.pick.choose")}, append(m.Acts, popupFileKind)
		}
		if row.Hint == "" {
			row.Hint = word("editor.pick.none")
		}
	}
	if r.Picture != "" {
		m.Picture = c.text(r.Picture, word)
		if row.Lines == 0 {
			row.Lines = 3
		}
	}
	if r.Icons != "" {
		m.Icons = lineWords(c.text(r.Icons, word))
	}
	m.Row = row
	return m
}

// parsePopupDesc reads a description and checks every row's kind.
func parsePopupDesc(data []byte) (popupDesc, error) {
	var input struct {
		popupDesc
		Hover *struct {
			Row    int `json:"row"`
			Column int `json:"column,omitempty"`
		} `json:"hover,omitempty"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return popupDesc{}, fmt.Errorf("window description: %w", err)
	}
	d := input.popupDesc
	if err := d.Panels.check(); err != nil {
		return popupDesc{}, err
	}
	if h := input.Hover; h != nil {
		if d.Preview == nil {
			d.Preview = &popupPreview{}
		}
		if d.Preview.HoverRow == nil {
			d.Preview.HoverRow, d.Preview.HoverList, d.Preview.HoverColumn = &h.Row, true, h.Column
		}
	}
	return d, checkPopupRows(d.Rows)
}

func checkPopupRows(rows []popupDescRow) error {
	for _, r := range rows {
		switch _, ok := popupKindNamed(r.Kind); {
		case r.Each != "" && r.Kind != "":
			return fmt.Errorf("window description: a repeated block %q has a kind", r.Each)
		case r.Each == "" && !ok:
			return fmt.Errorf("window description: unknown row kind %q", r.Kind)
		case r.Each == "" && len(r.Rows) > 0 && r.Kind != popupKindNames[popupCard]:
			return fmt.Errorf("window description: a %s row holds rows", r.Kind)
		case r.Fold != "" && r.Kind != popupKindNames[popupHeading]:
			return fmt.Errorf("window description: a %s row folds; only a heading does", r.Kind)
		case r.Hint != "" && r.HintBind != "":
			return fmt.Errorf("window description: a %s row has a hint word and a hint binding", r.Kind)
		case r.Picture != "" && r.Kind != popupKindNames[popupText] && r.Kind != popupKindNames[popupMultiline]:
			return fmt.Errorf("window description: a %s row holds a picture; only a text or multiline row does", r.Kind)
		case r.Whole && r.Kind != popupKindNames[popupMultiline]:
			return fmt.Errorf("window description: a %s row is whole; only a multiline row is", r.Kind)
		}
		if err := checkPopupRows(r.Rows); err != nil {
			return err
		}
	}
	return nil
}

func (d popupDesc) encode() []byte {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		panic(err) // a description holds only strings, numbers and lists
	}
	return append(b, '\n')
}

func screenDesc(name string) (popupDesc, error) {
	data, err := screenFiles.ReadFile("screens/" + name + ".json")
	if err != nil {
		return popupDesc{}, fmt.Errorf("window description %q: %w", name, err)
	}
	return parsePopupDesc(data)
}

var triggerWizardDesc = embeddedPopupDesc("triggerwizard")
var triggerDetailsDesc = embeddedPopupDesc("triggerdetails")

func embeddedPopupDesc(name string) func() popupDesc {
	return sync.OnceValue(func() popupDesc {
		d, err := screenDesc(name)
		if err != nil {
			panic(err)
		}
		return d
	})
}

// madeRowLines is rows with each Wrap text row cut into the lines the one
// line breaker gives for its column, one text row each; the first line
// keeps the row's buttons.
func madeRowLines(rows []popupRow, made []popupMade, box lineBox) ([]popupRow, []popupMade) {
	var outRows []popupRow
	var outMade []popupMade
	for k, r := range rows {
		if !made[k].Wrap {
			outRows, outMade = append(outRows, r), append(outMade, made[k])
			continue
		}
		label := strings.TrimLeft(r.Label, " ")
		indent := r.Label[:len(r.Label)-len(label)]
		room := box
		if box.Measure != nil {
			room.Width -= box.Measure(indent)
		}
		for i, line := range editorRule.plain(room, label) {
			row, md := r, made[k]
			row.Label, md.Wrap = indent+line, false
			if i > 0 {
				row.Buttons, md.Acts = nil, nil
			}
			md.Row = row
			outRows, outMade = append(outRows, row), append(outMade, md)
		}
	}
	return outRows, outMade
}
