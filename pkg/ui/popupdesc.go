package ui

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
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
// the editor gives. Card stands the person's card and figure in their own
// column right of the field columns, as the unit window does. Sample is an
// example state in words and flags, for a mockup without a map.
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
	Card       bool              `json:"card,omitempty"`
	Tabs       []string          `json:"tabs,omitempty"`
	Rows       []popupDescRow    `json:"rows"`
	Buttons    []popupDescButton `json:"buttons,omitempty"`
	Sample     *popupScope       `json:"sample,omitempty"`
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
	Ink       string            `json:"ink,omitempty"`
	Column    int               `json:"column,omitempty"`
	Lines     int               `json:"lines,omitempty"`
	Wrap      bool              `json:"wrap,omitempty"`
	Field     string            `json:"field,omitempty"`
	Buttons   []popupDescButton `json:"buttons,omitempty"`
}

// popupDescButton is a button: its caption's word key, the row action it
// runs, the flag it needs and, for a window button, the flag that disables
// it.
type popupDescButton struct {
	Word string `json:"word"`
	Act  string `json:"act,omitempty"`
	When string `json:"when,omitempty"`
	Off  string `json:"off,omitempty"`
}

// popupKindNames names each popupRowKind in a description.
var popupKindNames = [...]string{
	popupText: "text", popupHeading: "heading", popupEdit: "edit", popupChoice: "choice", popupPick: "pick",
	popupLink: "link", popupSwitch: "switch", popupMultiline: "multiline", popupCard: "card", popupDirection: "direction",
}

func popupKindNamed(name string) (popupRowKind, bool) {
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
// field it edits (-1 none), its buttons' actions, and for a wrapped text
// row its ink role.
type popupMade struct {
	Row   popupRow
	Field int
	Acts  []string
	Wrap  bool
	Role  string
}

// build is the description as a popupSpec under the bindings b, every word
// key through word, and its rows with their fields and actions. It places
// nothing: layoutPopup lays the spec out.
func (d popupDesc) build(b popupBinds, word func(string) string) (popupSpec, []popupMade) {
	c := popupChain{b}
	s := popupSpec{Width: d.Width, LabelWidth: d.LabelWidth, MinWidth: d.MinWidth, ButtonW: d.ButtonW, Columns: d.Columns}
	if d.Card {
		// The card's column, at the card's width until a figure widens it;
		// the column shows once the window is given a person's card.
		s.CardAside = compactPanelW
	}
	if d.Title != "" {
		s.Title = word(d.Title)
		if len(d.TitleArgs) > 0 {
			args := make([]any, len(d.TitleArgs))
			for i, a := range d.TitleArgs {
				t := c.text(a, word)
				args[i] = t
				if n, err := strconv.Atoi(t); err == nil {
					args[i] = n
				}
			}
			s.Title = fmt.Sprintf(s.Title, args...)
		}
	}
	if d.TitleHelp != "" {
		s.TitleHelp = word(d.TitleHelp)
	}
	for _, t := range d.Tabs {
		s.Tabs = append(s.Tabs, word(t))
	}
	var made []popupMade
	c.rows(d.Rows, word, &made)
	for _, m := range made {
		s.Rows = append(s.Rows, m.Row)
	}
	for _, bt := range d.Buttons {
		if c.when(bt.When) {
			s.Buttons = append(s.Buttons, word(bt.Word))
			s.Disabled = append(s.Disabled, bt.Off != "" && c.when(bt.Off))
		}
	}
	return s, made
}

func (c popupChain) rows(rows []popupDescRow, word func(string) string, made *[]popupMade) {
	for _, r := range rows {
		if !c.when(r.When) {
			continue
		}
		if r.Each == "" {
			*made = append(*made, c.row(r, word))
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
		label = word(r.Label)
	} else if r.LabelBind != "" {
		label = c.text(r.LabelBind, word)
	}
	row := popupRow{Kind: kind, Label: strings.Repeat(" ", r.Indent) + label, Column: r.Column, Lines: r.Lines}
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
	}
	if r.Ink != "" || r.Wrap {
		// A wrapped text row takes its role's ink as the window's text rows do.
		row.Ink = inspectionRoleColor(r.Ink)
	}
	m := popupMade{Field: -1, Wrap: r.Wrap, Role: r.Ink}
	if r.Field != "" {
		m.Field = c.field(r.Field)
	}
	for _, bt := range r.Buttons {
		if c.when(bt.When) {
			row.Buttons = append(row.Buttons, word(bt.Word))
			m.Acts = append(m.Acts, bt.Act)
		}
	}
	m.Row = row
	return m
}

// parsePopupDesc reads a description and checks every row's kind.
func parsePopupDesc(data []byte) (popupDesc, error) {
	var d popupDesc
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return popupDesc{}, fmt.Errorf("window description: %w", err)
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
		case r.Each == "" && len(r.Rows) > 0:
			return fmt.Errorf("window description: a %s row holds rows", r.Kind)
		case r.Fold != "" && r.Kind != popupKindNames[popupHeading]:
			return fmt.Errorf("window description: a %s row folds; only a heading does", r.Kind)
		case r.Hint != "" && r.HintBind != "":
			return fmt.Errorf("window description: a %s row has a hint word and a hint binding", r.Kind)
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

// triggerWizardDesc is the trigger wizard's embedded description; a test
// reads every embedded description, so a broken one fails before a build.
var triggerWizardDesc = sync.OnceValue(func() popupDesc {
	d, err := screenDesc("triggerwizard")
	if err != nil {
		panic(err)
	}
	return d
})
