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
// views. The detailed view (the profile's "technical") is every field as
// the host lists it. The basic view (the profile's "plain") shows fields,
// built by the same row builder, in the order and the blocks of the record
// kind's plain table, under plain words; a field the table leaves out
// stands in the detailed view only. A unit's basic view is its character
// card drawn larger, each stat the card draws an input at its place. The
// switch changes what shows, never a value: both views edit the one draft
// and Apply writes it through applyObject.

// plainShow is how the plain view presents a table row.
type plainShow uint8

const (
	plainField plainShow = iota // the field's own control
	plainItem                   // the field's control showing the item's name and picture
	plainCode                   // a line stating the field's number; the control stands elsewhere
	plainBits                   // the field's control, then its word and known bits
	plainName                   // the field's control, or a line with the card's name when the host gives none
	plainUses                   // the script nodes that name the record
	plainPanel                  // the field's input on the card, at the place the card draws its panel field
)

// plainBlock is one block of a plain view: the word of its heading, none
// for the block the window opens with; fold marks a block that opens and
// closes, closed at first; groups keeps the fields' group headings in it;
// card marks the block whose rows stand on the character card, drawn
// cardScale times its size under the rows.
type plainBlock struct {
	id, word     string
	fold, groups bool
	card         bool
}

// cardScale is the size of the card a card block stands on, in times the
// character panel's own: a line of the panel then holds the editor's text
// field in the body font.
const cardScale = 2

// plainRow is one row of a record kind's plain table. Key names the field:
// a "#" in it stands for a number, a final "*" for any rest; an empty key is
// a row no field fills. Word is the plain label's word id, "" for the
// field's own label; a "%d" in it takes the key's number. Panel is the
// card's field a plainPanel row edits. Rows sort by block, then order, then
// the key's number, then the host's field order.
type plainRow struct {
	key, word string
	order     int
	block     string
	show      plainShow
	panel     PanelField
}

// plainView is a record kind's plain table: its blocks in window order, its
// rows, and the block a field no row names stands in; with rest "" such a
// field stands in the detailed view only.
type plainView struct {
	blocks []plainBlock
	rows   []plainRow
	rest   string
}

var plainViews = map[RecordKind]plainView{
	KindUnit: {
		// The name, the player and the group, then who the unit is, which
		// gives the card's lower part its template, and whether a dialogue
		// shows it. Every stat the card draws stands on it; the map keys
		// behind who it is fold away; equipment, the bag, the look, the
		// resolved values, the overrides and the map file fields stand in
		// the detailed view only.
		blocks: []plainBlock{
			{id: "main"},
			{id: "keys", word: "editor.plain.keys", fold: true},
			{id: "card", card: true},
		},
		rows: []plainRow{
			{key: "name", word: "editor.plain.name", order: 1, block: "main", show: plainName},
			{key: "owner", word: "editor.plain.player", order: 2, block: "main"},
			{key: "group", word: "editor.plain.group", order: 3, block: "main"},
			{key: "identity", word: "editor.plain.who", order: 4, block: "main"},
			{key: "facing", word: "editor.plain.facing", order: 5, block: "main"},
			{key: "speaker", order: 6, block: "main"},
			{key: "keys", order: 7, block: "keys"},
			{key: "hp", block: "card", show: plainPanel, panel: PanelFieldHealth},
			{key: "mana", block: "card", show: plainPanel, panel: PanelFieldManaCard},
			{key: "body", block: "card", show: plainPanel, panel: PanelFieldBody},
			{key: "reaction", block: "card", show: plainPanel, panel: PanelFieldReaction},
			{key: "mind", block: "card", show: plainPanel, panel: PanelFieldMind},
			{key: "spirit", block: "card", show: plainPanel, panel: PanelFieldSpirit},
			{key: "absorption", block: "card", show: plainPanel, panel: PanelFieldAbsorption},
			{key: "defence", block: "card", show: plainPanel, panel: PanelFieldDefence},
			{key: "sight", block: "card", show: plainPanel, panel: PanelFieldSight},
			{key: "speed", block: "card", show: plainPanel, panel: PanelFieldMoveSpeed},
			{key: "skill.blade", block: "card", show: plainPanel, panel: PanelFieldSkillBlade},
			{key: "skill.axe", block: "card", show: plainPanel, panel: PanelFieldSkillAxe},
			{key: "skill.bludgeon", block: "card", show: plainPanel, panel: PanelFieldSkillBludgeon},
			{key: "skill.pike", block: "card", show: plainPanel, panel: PanelFieldSkillPike},
			{key: "skill.shooting", block: "card", show: plainPanel, panel: PanelFieldSkillShooting},
			{key: "protection.fire", block: "card", show: plainPanel, panel: PanelFieldProtFire},
			{key: "protection.water", block: "card", show: plainPanel, panel: PanelFieldProtWater},
			{key: "protection.air", block: "card", show: plainPanel, panel: PanelFieldProtAir},
			{key: "protection.earth", block: "card", show: plainPanel, panel: PanelFieldProtEarth},
			{key: "protection.astral", block: "card", show: plainPanel, panel: PanelFieldProtAstral},
		},
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
	KindSack:  sackPlainView,
	KindStock: sackPlainView,
}

// sackPlainView is the plain table of a sack's window: a sack on the
// ground's, and a unit's own sack's, which its unit's bag opens.
var sackPlainView = plainView{
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
		{key: "recipe#.*", order: 10, block: "details"},
		{key: "use#", order: 10, block: "details"},
	},
	rest: "details",
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
		if row.key == "" || row.show == plainName && !slices.ContainsFunc(fields, func(f InspectionField) bool { return f.Key == row.key }) {
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
// not name maps to "" and stands in the table's rest block, or in the
// detailed view only when the table has none. A card field maps to its
// block while the basic view shows its input on the card, to "" when the
// card does not draw it. It is nil when the window has no plain view.
func (e *MapEditor) PlainFieldBlocks() map[string]string {
	e.objectSpec()
	v, ok := e.plainOffer()
	if !ok {
		return nil
	}
	out := map[string]string{}
	r := e.doc.Records[e.selected]
	for _, p := range plainEntries(v, e.obj.fields) {
		if p.field < 0 || sackField(r, e.obj.fields[p.field]) {
			continue
		}
		key := e.obj.fields[p.field].Key
		switch {
		case p.rest:
			out[key] = ""
		case v.rows[p.row].show == plainPanel:
			out[key] = ""
			if e.props.plain && e.fieldRow(p.field) >= 0 {
				out[key] = p.block
			}
		case v.rows[p.row].show != plainCode:
			out[key] = p.block
		}
	}
	return out
}

// viewCaption is the switch button's caption: the view shown.
func (e *MapEditor) viewCaption() string {
	view := e.word("editor.view.detailed")
	if e.props.plain {
		view = e.word("editor.view.basic")
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
	record := e.doc.Records[e.selected]
	shown := ""
	var card []plainEntry
	for _, p := range entries {
		at := slices.IndexFunc(v.blocks, func(b plainBlock) bool { return b.id == p.block })
		if at < 0 || p.field >= 0 && sackField(record, o.fields[p.field]) {
			continue
		}
		b := v.blocks[at]
		if p.field >= 0 && native && o.fields[p.field].Parent != "" && o.fields[p.field].Parent != o.detail {
			continue
		}
		if b.card {
			card = append(card, p)
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
		show := row.show
		if show == plainName && p.field >= 0 {
			show = plainField
		}
		switch show {
		case plainName:
			if s.Card != nil && s.Card.Name != "" {
				text(s.Card.Name)
				s.Rows[len(s.Rows)-1].Tip = fmt.Sprintf(e.word("editor.plain.fixed"), label+" "+s.Card.Name)
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
	if slices.ContainsFunc(v.blocks, func(b plainBlock) bool { return b.card }) {
		e.plainCardRows(s, links, v, card, field, textWidth)
	}
}

// plainCardRows stands the card block's rows on the window's card, drawn
// cardScale times its size under the rows. Each cell the card draws with a
// value becomes a row at the cell's place, read from the card's own line
// report: the input of the field whose table row names the cell's panel
// field, built by field, or else a line stating the value the map does not
// store. A card field the card does not draw has no row here.
func (e *MapEditor) plainCardRows(s *popupSpec, links *[]int, v plainView, entries []plainEntry, field func(i int, label string, heads bool) int, textWidth int) {
	o := &e.obj
	if s.Card == nil {
		return
	}
	s.CardScale, s.CardBelow = cardScale, true
	// The window's own script uses stand in the detailed view.
	o.usesShown = true
	font, body := s.CardFont, e.bodyFont()
	if font == nil {
		font = body
	}
	if font == nil {
		return
	}
	edits := map[PanelField]int{}
	for _, p := range entries {
		if p.field >= 0 && !p.rest && v.rows[p.row].show == plainPanel && !o.fields[p.field].ReadOnly {
			edits[v.rows[p.row].panel] = p.field
		}
	}
	layout := CompactPanelLayout(nil)
	lineH := font.Height() + layout.Gap
	// An input holds four digits of the window's own font.
	least := 0
	if body != nil {
		least = body.Advance("0000") + 2*editTextX
	}
	// A stat whose stored value differs from the card's value, the one the
	// game uses, is stated in words beside the card.
	var differ []string
	cell := func(f PanelField, label, value string, pen, start, y int) {
		if value == "" {
			return
		}
		// A pair such as health's hp/max keeps its max part on the card;
		// the input stands on the first number.
		number, _, pair := strings.Cut(value, "/")
		box := image.Rectangle{Min: image.Pt(pen, y).Mul(cardScale), Max: image.Pt(pen+font.Advance(number), y+lineH).Mul(cardScale)}
		if !pair {
			box.Max.X = (pen + font.Advance(value)) * cardScale
		}
		if i, ok := edits[f]; ok {
			if n := field(i, "", false); n >= 0 {
				// The card states the value; the row is its input, the
				// field's notes its tip.
				tip := []string{e.editorHostText(o.fields[i].Label)}
				if o.fields[i].Inherit {
					tip = append(tip, e.groupNote(o.fields[i].Group))
				}
				var meaning []string
				for _, r := range s.Rows[n+1:] {
					tip = append(tip, r.Label)
					meaning = append(meaning, r.Label)
				}
				s.Rows, *links = s.Rows[:n+1], (*links)[:n+1]
				row := &s.Rows[n]
				box.Min.X = max(min(box.Min.X, box.Max.X-least), start*cardScale)
				row.Place, row.Buttons, row.Tip = box, nil, strings.Join(slices.DeleteFunc(tip, func(t string) bool { return t == "" }), " ")
				if stored := strings.TrimSpace(row.Value); stored != "" && !sameNumber(stored, number) {
					name := strings.TrimSpace(label)
					if name == "" {
						name = e.editorHostText(o.fields[i].Label)
					}
					why := strings.Join(meaning, " ")
					if why == "" {
						why = fmt.Sprintf(e.word("editor.plain.derived"), number)
					}
					differ = append(differ, name+": "+why)
				}
				return
			}
		}
		s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: label, Value: value, Place: box,
			Tip: fmt.Sprintf(e.word("editor.plain.fixed"), strings.TrimSpace(label+" "+value))}), append(*links, 0)
	}
	for _, r := range CharacterPanelReport(layout, font, *s.Card) {
		left := r.At.X + r.Indent
		labelEnd := left
		if r.Label != "" {
			labelEnd += font.Advance(r.Label) + 1
		}
		cell(r.Field, r.Label, r.Value, left+r.ValueX, labelEnd, r.At.Y)
		if r.Right {
			right := r.At.X + r.RightX
			if r.RightLabel != "" {
				right += font.Advance(r.RightLabel) + 1
			}
			cell(r.RightField, r.RightLabel, r.RightValue, r.At.X+r.RightX+r.RightValueX, right, r.At.Y)
		}
	}
	for _, line := range differ {
		for _, part := range editorTextLines(body, line, textWidth) {
			s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: part}), append(*links, 0)
		}
	}
}

// sameNumber reports that a stored value and the card's text of it name one
// number: the card writes sight with one decimal.
func sameNumber(stored, shown string) bool {
	a, errA := strconv.ParseFloat(stored, 64)
	b, errB := strconv.ParseFloat(shown, 64)
	if errA != nil || errB != nil {
		return stored == shown
	}
	return a == b
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
