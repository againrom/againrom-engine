package ui

import (
	"fmt"
	"image"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// The object window shows a unit, a structure or a sack in one of two
// views. The technical view is every field as the host lists it. The plain
// view shows the same fields, built by the same row builder, in the order
// and the blocks of the record kind's plain table, under plain words. The
// switch changes what shows, never a value: both views edit the one draft
// and Apply writes it through applyObject.

// plainShow is how the plain view presents a table row.
type plainShow uint8

const (
	plainField      plainShow = iota // the field's own control
	plainItem                        // the field's control showing the item's name and picture
	plainCode                        // a line stating the field's number; the control stands elsewhere
	plainBits                        // the field's control, then its word and known bits
	plainName                        // a line with the unit's name from its card
	plainExperience                  // a line with the unit's experience from its card
	plainUses                        // the script nodes that name the record
)

// plainBlock is one block of a plain view: the word of its heading, none
// for the block the window opens with; fold marks a block that opens and
// closes, closed at first; groups keeps the fields' group headings in it.
type plainBlock struct {
	id, word     string
	fold, groups bool
}

// plainRow is one row of a record kind's plain table. Key names the field:
// a "#" in it stands for a number, a final "*" for any rest; an empty key is
// a row no field fills. Word is the plain label's word id, "" for the
// field's own label; a "%d" in it takes the key's number. Rows sort by
// block, then order, then the key's number, then the host's field order.
type plainRow struct {
	key, word string
	order     int
	block     string
	show      plainShow
}

// plainView is a record kind's plain table: its blocks in window order, its
// rows, and the block a field no row names stands in.
type plainView struct {
	blocks []plainBlock
	rows   []plainRow
	rest   string
}

var plainViews = map[RecordKind]plainView{
	KindUnit: {
		blocks: []plainBlock{
			{id: "main"},
			{id: "look", word: "editor.plain.look"},
			{id: "equipment", word: "editor.plain.equipment"},
			{id: "bag", word: "editor.plain.bag"},
			{id: "stats", word: "editor.plain.stats", groups: true},
			{id: "raw", word: "editor.plain.raw", fold: true},
		},
		rows: []plainRow{
			{key: "", word: "editor.plain.name", order: 1, block: "main", show: plainName},
			{key: "owner", word: "editor.plain.owner", order: 2, block: "main"},
			{key: "class", word: "editor.plain.kind", order: 3, block: "main"},
			{key: "tier", word: "editor.plain.tier", order: 4, block: "main"},
			{key: "", word: "editor.plain.experience", order: 5, block: "main", show: plainExperience},
			{key: "hp", word: "editor.plain.life", order: 6, block: "main"},
			{key: "mana", word: "editor.plain.mana", order: 7, block: "main"},
			{key: "group", word: "editor.plain.group", order: 8, block: "main"},
			{key: "face", order: 10, block: "look"},
			{key: "sex", order: 11, block: "look"},
			{key: "npc", order: 12, block: "look"},
			{key: "npcflag", order: 13, block: "look"},
			{key: "gear", order: 20, block: "equipment"},
			{key: "wear#", order: 21, block: "equipment", show: plainItem},
			{key: "wearmaterial#", order: 21, block: "equipment"},
			{key: "weartier#", order: 21, block: "equipment"},
			{key: "wearrecipe#", order: 21, block: "equipment"},
			{key: "wearrecipe#.*", order: 21, block: "equipment"},
			{key: "carry#", order: 30, block: "bag", show: plainItem},
			{key: "carryrecipe#", order: 30, block: "bag"},
			{key: "carryrecipe#.*", order: 30, block: "bag"},
			{key: "carryadd", order: 31, block: "bag"},
			{key: "holdings", order: 32, block: "bag"},
			{key: "body", order: 40, block: "stats"},
			{key: "mind", order: 40, block: "stats"},
			{key: "spirit", order: 40, block: "stats"},
			{key: "reaction", order: 40, block: "stats"},
			{key: "defence", order: 40, block: "stats"},
			{key: "absorption", order: 40, block: "stats"},
			{key: "skill.*", order: 40, block: "stats"},
			{key: "protection.*", order: 40, block: "stats"},
			{key: "resolved", order: 41, block: "stats"},
			{key: "overrides", order: 42, block: "stats"},
			{key: "definition", order: 50, block: "raw"},
		},
		rest: "raw",
	},
	KindStructure: {
		blocks: []plainBlock{
			{id: "main"},
			{id: "uses"},
			{id: "placement", word: "editor.plain.placement", fold: true},
			{id: "raw", word: "editor.plain.raw", fold: true},
		},
		rows: []plainRow{
			{key: "kind", word: "editor.plain.type", order: 1, block: "main"},
			{key: "owner", word: "editor.plain.owner", order: 2, block: "main"},
			{key: "identity", word: "editor.plain.target", order: 3, block: "main"},
			{key: "width", word: "editor.plain.width", order: 4, block: "main"},
			{key: "height", word: "editor.plain.height", order: 5, block: "main"},
			{key: "", order: 10, block: "uses", show: plainUses},
			{key: "placement", word: "editor.plain.word", order: 20, block: "placement", show: plainBits},
		},
		rest: "raw",
	},
	KindSack: {
		blocks: []plainBlock{
			{id: "main"},
			{id: "details", word: "editor.plain.details", fold: true},
		},
		rows: []plainRow{
			{key: "gold", word: "editor.plain.gold", order: 1, block: "main"},
			{key: "item#", word: "editor.plain.item", order: 2, block: "main", show: plainItem},
			{key: "additem", word: "editor.plain.additem", order: 3, block: "main"},
			{key: "item#", word: "editor.plain.code", order: 10, block: "details", show: plainCode},
			{key: "recipe#", word: "editor.plain.recipe", order: 10, block: "details"},
			{key: "use#", order: 10, block: "details"},
		},
		rest: "details",
	},
}

// plainMatch reports that key fits pattern, and the number a "#" took.
func plainMatch(pattern, key string) (int, bool) {
	if pattern == "" {
		return 0, false
	}
	head, tail, numbered := strings.Cut(pattern, "#")
	if !numbered {
		if rest, ok := strings.CutSuffix(pattern, "*"); ok {
			return 0, strings.HasPrefix(key, rest)
		}
		return 0, key == pattern
	}
	digits, ok := strings.CutPrefix(key, head)
	if !ok {
		return 0, false
	}
	end := 0
	for end < len(digits) && digits[end] >= '0' && digits[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(digits[:end])
	if err != nil {
		return 0, false
	}
	rest := digits[end:]
	if prefix, ok := strings.CutSuffix(tail, "*"); ok {
		return n, strings.HasPrefix(rest, prefix)
	}
	return n, rest == tail
}

// plainEntry is one row the plain view shows: table row row over field
// field, -1 for a row no field fills, with the key's number.
type plainEntry struct {
	row, field, number int
	block              string
	rest               bool
}

// plainEntries are the rows of v over fields in window order: each table
// row over each field it names, and each field no row edits in the rest
// block, after the table's rows.
func plainEntries(v plainView, fields []InspectionField) []plainEntry {
	var out []plainEntry
	edited := map[int]bool{}
	for r, row := range v.rows {
		if row.key == "" {
			out = append(out, plainEntry{row: r, field: -1, block: row.block})
			continue
		}
		for i, f := range fields {
			n, ok := plainMatch(row.key, f.Key)
			if !ok {
				continue
			}
			if row.show != plainCode {
				if edited[i] {
					continue
				}
				edited[i] = true
			}
			out = append(out, plainEntry{row: r, field: i, number: n, block: row.block})
		}
	}
	for i := range fields {
		if !edited[i] {
			out = append(out, plainEntry{row: -1, field: i, block: v.rest, rest: true})
		}
	}
	block := func(id string) int {
		return slices.IndexFunc(v.blocks, func(b plainBlock) bool { return b.id == id })
	}
	order := func(p plainEntry) int {
		if p.rest {
			return int(^uint(0) >> 1)
		}
		return v.rows[p.row].order
	}
	sort.SliceStable(out, func(a, b int) bool {
		x, y := out[a], out[b]
		if block(x.block) != block(y.block) {
			return block(x.block) < block(y.block)
		}
		if order(x) != order(y) {
			return order(x) < order(y)
		}
		if x.number != y.number {
			return x.number < y.number
		}
		return x.field < y.field
	})
	return out
}

// propertyView is the object window's view: plain or technical, and the
// plain view's blocks the map maker opened. The zero view is technical, the
// window as it was before the plain view; the editor profile starts plain.
type propertyView struct {
	plain bool
	open  map[string]bool
}

// plainTable is the plain table of the shown record's window, false when
// the window shows no plain view: the technical view is chosen, or the
// window is not a unit's, a structure's or a sack's own.
func (e *MapEditor) plainTable() (plainView, bool) {
	if !e.props.plain {
		return plainView{}, false
	}
	return e.plainOffer()
}

// plainOffer is the plain table of the shown record's window, whichever
// view is chosen: the window that shows the switch.
func (e *MapEditor) plainOffer() (plainView, bool) {
	if e.modWindow || e.messagePage != 0 || e.doc == nil || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return plainView{}, false
	}
	if _, ok := e.scriptPopup(); ok {
		return plainView{}, false
	}
	v, ok := plainViews[e.doc.Records[e.selected].Kind]
	return v, ok
}

// switchView turns the object window to the other view and keeps the
// choice in the editor profile. The draft stays as typed.
func (e *MapEditor) switchView() {
	e.props.plain = !e.props.plain
	e.obj.open, e.obj.focus = -1, -1
	e.objectTop = 0
	e.savePanels()
}

// viewSwitchButton puts the offered view switch in the window's title row,
// left of its close button: the button row and the rows keep their room
// and their places in both views.
func (e *MapEditor) viewSwitchButton(s *popupSpec) {
	if e.obj.viewOffer {
		s.TitleButton = e.viewCaption()
	}
}

// ViewSwitchRect is the window rectangle of the object window's view
// switch, when the window shows one.
func (e *MapEditor) ViewSwitchRect() (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok || s.TitleButton == "" {
		return image.Rectangle{}, false
	}
	return layoutPopup(s, e.font, e.bodyFont()).TitleButton, true
}

// PlainView reports that the object window shows its plain view.
func (e *MapEditor) PlainView() bool {
	_, ok := e.plainTable()
	return ok
}

// PlainFieldBlocks names, for each field of the shown object window, the
// block of the plain view whose table row edits it; a field the table does
// not name maps to "" and stands in the table's rest block. It is nil when
// the window has no plain view.
func (e *MapEditor) PlainFieldBlocks() map[string]string {
	e.objectSpec()
	v, ok := e.plainOffer()
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, p := range plainEntries(v, e.obj.fields) {
		if p.field < 0 {
			continue
		}
		key := e.obj.fields[p.field].Key
		if p.rest {
			out[key] = ""
		} else if v.rows[p.row].show != plainCode {
			out[key] = p.block
		}
	}
	return out
}

// viewCaption is the switch button's caption: the view shown.
func (e *MapEditor) viewCaption() string {
	view := e.word("editor.view.technical")
	if e.props.plain {
		view = e.word("editor.view.plain")
	}
	return fmt.Sprintf(e.word("editor.view.switch"), view)
}

// blockKey names a block of the shown record kind's plain view.
func (e *MapEditor) blockKey(id string) string {
	return e.doc.Records[e.selected].Kind.String() + "/" + id
}

// blockOpen reports that block b of the shown window's plain view shows its
// rows.
func (e *MapEditor) blockOpen(b plainBlock) bool {
	return !b.fold || e.props.open[e.blockKey(b.id)]
}

// toggleBlock opens or closes a block of the plain view.
func (e *MapEditor) toggleBlock(id string) {
	if e.props.open == nil {
		e.props.open = map[string]bool{}
	}
	k := e.blockKey(id)
	e.props.open[k] = !e.props.open[k]
	e.obj.open = -1
}

// plainFieldRows adds the plain view's rows of the shown record: each
// block's heading, or the row that opens and closes it, then the rows of an
// open block in table order. field adds field i's rows under label and
// reports the row that edits it, -1 for none.
func (e *MapEditor) plainFieldRows(s *popupSpec, links *[]int, v plainView, field func(i int, label string, heads bool) int, textWidth int) {
	o := &e.obj
	o.rowBlock = map[int]string{}
	native := e.nativeUnitWindow()
	entries := plainEntries(v, o.fields)
	shown := ""
	for _, p := range entries {
		at := slices.IndexFunc(v.blocks, func(b plainBlock) bool { return b.id == p.block })
		if at < 0 {
			continue
		}
		b := v.blocks[at]
		if p.field >= 0 && native && o.fields[p.field].Parent != "" && o.fields[p.field].Parent != o.detail {
			continue
		}
		if p.block != shown {
			shown = p.block
			switch {
			case b.fold:
				o.rowBlock[len(s.Rows)] = b.id
				s.Rows, *links = append(s.Rows, blockRow(e.word(b.word), e.blockOpen(b))), append(*links, 0)
			case b.word != "":
				s.Rows, *links = append(s.Rows, popupRow{Kind: popupHeading, Label: e.word(b.word)}), append(*links, 0)
			}
		}
		if !e.blockOpen(b) {
			continue
		}
		row := plainRow{block: p.block}
		if !p.rest {
			row = v.rows[p.row]
		}
		label := ""
		if row.word != "" {
			label = e.word(row.word)
			if strings.Contains(label, "%d") {
				label = fmt.Sprintf(label, p.number)
			}
		}
		text := func(value string) {
			s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: label, Value: value}), append(*links, 0)
		}
		switch row.show {
		case plainName:
			if s.Card != nil && s.Card.Name != "" {
				text(s.Card.Name)
			}
		case plainExperience:
			if s.Card != nil && s.Card.Char.Known && !s.Card.Char.ExperienceWithheld {
				text(strconv.Itoa(s.Card.Char.Experience))
			}
		case plainUses:
			if e.hasUsedBy() {
				e.scriptLinkSection(s, links, e.word("editor.usedby.title"), e.usedByRows())
				o.usesShown = true
			}
		case plainCode:
			text(strings.TrimSpace(o.buf[p.field]))
		default:
			// A field outside every group takes no heading of its own.
			n := field(p.field, label, b.groups && o.fields[p.field].Group != "")
			if n < 0 {
				continue
			}
			if row.show == plainItem {
				e.plainItemRow(&s.Rows[n], p.field)
			}
			if row.show == plainBits {
				for _, line := range e.placementLines(p.field) {
					for _, part := range editorTextLines(e.bodyFont(), line, textWidth) {
						s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: part}), append(*links, 0)
					}
				}
			}
		}
	}
}

// InspectionItemLook is implemented by edits whose host names and pictures
// an item code: the name the item-name builder gives it and the picture the
// inventory shows, nil for none.
type InspectionItemLook interface {
	ItemLook(code int64) (name string, icon *image.RGBA)
}

// plainItemRow shows an item field's row as the item's name and picture;
// an empty or inherited slot keeps its choice's words.
func (e *MapEditor) plainItemRow(row *popupRow, i int) {
	host, ok := e.doc.Edits.(InspectionItemLook)
	code, err := strconv.ParseInt(strings.TrimSpace(e.obj.buf[i]), 10, 64)
	if !ok || err != nil || code <= 0 {
		return
	}
	name, icon := host.ItemLook(code)
	if name != "" {
		row.Value = name
	}
	if icon != nil {
		row.Icons = []*image.RGBA{icon}
	}
}

// placementLines state a structure's placement word as typed: its number in
// hex with the bits it sets, and what the building reads from it (ALM-128):
// a building whose word is 0 starts with no health, any other word keeps
// its type's health; a shop, types 34 and 35, does not read it so.
func (e *MapEditor) placementLines(i int) []string {
	o := &e.obj
	v, err := strconv.ParseInt(strings.TrimSpace(o.buf[i]), 10, 64)
	if err != nil {
		return nil
	}
	word := uint16(v)
	var bits []string
	for b := 0; b < 16; b++ {
		if word&(1<<b) != 0 {
			bits = append(bits, strconv.Itoa(b))
		}
	}
	set := e.word("editor.plain.bits.none")
	if len(bits) > 0 {
		set = strings.Join(bits, ", ")
	}
	out := []string{fmt.Sprintf(e.word("editor.plain.bits"), word, set)}
	kind := int64(-1)
	for k, f := range o.fields {
		if f.Key == "kind" {
			kind, _ = strconv.ParseInt(strings.TrimSpace(o.buf[k]), 10, 64)
		}
	}
	switch {
	case kind == 34 || kind == 35:
		out = append(out, e.word("editor.plain.placement.shop"))
	case word == 0:
		out = append(out, e.word("editor.plain.placement.zero"))
	default:
		out = append(out, e.word("editor.plain.placement.health"))
	}
	return out
}
