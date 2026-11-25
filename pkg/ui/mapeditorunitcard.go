package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
)

// The unit window is one window in two modes. The detailed mode shows every
// section a unit has data for, each in its column: who the unit is with its
// card, the doll, the equipment and the bag. The normal mode is the same
// window less the parts unitModes names. The card is drawn cardScale times
// its size and edits each value it states in place: an input stands where the
// card draws the value. The switch changes what shows, never a value: both
// modes edit the one draft and Apply writes it through applyObject.

// unitMode is what one mode of the unit window leaves out of the detailed
// window: whole sections and single fields.
type unitMode struct {
	sections, keys []string
}

// unitModes are the unit window's detailed mode, the profile's "technical"
// view, and its normal mode, the profile's "plain" view.
var unitModes = [2]unitMode{
	{},
	{sections: []string{"equipment", "inventory"}, keys: []string{"face", "sex"}},
}

// leaves reports that the mode does not show field f.
func (m unitMode) leaves(f InspectionField) bool {
	return slices.Contains(m.sections, f.Section) || slices.Contains(m.keys, f.Key)
}

// unitModeShown is the mode the profile chose for the unit window.
func (e *MapEditor) unitModeShown() unitMode {
	if e.props.plain {
		return unitModes[1]
	}
	return unitModes[0]
}

// unitColumns are the unit window's sections in column order. The card
// stands under the identity section and edits the cardSection's fields; the
// doll section is the unit's figure.
var unitColumns = []string{"identity", "doll", "equipment", "inventory"}

// unitSectionsShown are the sections of unitColumns the shown unit window
// has data for in mode m: identity always, the doll for a person's card, a
// section of fields while the mode shows one of them.
func (e *MapEditor) unitSectionsShown(m unitMode, card *PanelSubject) []string {
	r := e.doc.Records[e.selected]
	var out []string
	for _, section := range unitColumns {
		has := section == "identity" || section == "doll" && unitDoll(card)
		if section == "doll" {
			has = has || slices.ContainsFunc(e.obj.unit.fields, func(f InspectionField) bool { return f.Key == "speaker" && f.Picture != nil })
		}
		has = has || slices.ContainsFunc(e.obj.fields, func(f InspectionField) bool {
			return f.Section == section && !hiddenField(r, f) && !m.leaves(f)
		})
		if has {
			out = append(out, section)
		}
	}
	return out
}

// unitSectionOrder is the order the window adds its sections' rows in: the
// column order, the doll after the section it shares a column with.
func unitSectionOrder(shown []string, columns int) []string {
	at := slices.Index(shown, "doll")
	if at < 0 || columns >= len(shown) || at+1 >= len(shown) {
		return shown
	}
	out := slices.Clone(shown)
	out[at], out[at+1] = out[at+1], out[at]
	return out
}

// unitDoll reports that a unit of card has a doll: a person's card.
func unitDoll(card *PanelSubject) bool {
	return card != nil && card.Char.Band == CharacterBandPerson
}

// unitColumn is the column of section in a window of columns columns over
// the sections shown: the card's fields and a field of no section stand in
// the identity column. A window with fewer columns than sections stands the
// doll over the section after it, and a section past the last column in the
// last.
func unitColumn(section string, shown []string, columns int) int {
	if at := slices.Index(shown, "doll"); at >= 0 && columns < len(shown) {
		shown = slices.Delete(slices.Clone(shown), at, at+1)
		if section == "doll" {
			section = shown[min(at, len(shown)-1)]
		}
	}
	k := slices.Index(shown, section)
	if section == cardSection || k < 0 {
		k = 0
	}
	return min(k, max(1, columns)-1)
}

// unitColumnWidth is the width of one column of the unit window: the least
// column width, or wider when the card or the figure needs it.
func (e *MapEditor) unitColumnWidth(s popupSpec) int {
	w := personColumnWidth
	if s.Card != nil {
		w = max(w, compactPanelW*cardScale+2*popupPad)
	}
	if unitDoll(s.Card) {
		w = max(w, figureAside(e.unitFigureSize())+2*popupPad)
	}
	return w
}

// cardPastRoom reports that the rows of the card's column, down to the end
// of the card's held place, are higher than room at pitch a line: the
// window's first rows could not show the card whole.
func cardPastRoom(rows []popupRow, pitch, room int) bool {
	lines := map[int]int{}
	for _, r := range rows {
		if r.placed() {
			continue
		}
		lines[r.Column] += max(1, r.Lines)
		if r.Hold == holdCard && lines[r.Column]*pitch > room {
			return true
		}
	}
	return false
}

// cardScale is the size of the unit window's card, in times the character
// panel's own: a line of the panel then holds the editor's text field in the
// body font.
const cardScale = 2

// cardInput is a field the card edits in place: the panel field the card
// draws its value at, on the pair's second number when second is set.
// whole stretches the input across the card's line.
type cardInput struct {
	key    string
	panel  PanelField
	second bool
	whole  bool
}

// unitCard is every field the unit window's card edits.
var unitCard = []cardInput{
	{key: "name", panel: PanelFieldName, whole: true},
	{key: "hp", panel: PanelFieldHealth},
	{key: "hpmax", panel: PanelFieldHealth, second: true},
	{key: "mana", panel: PanelFieldManaCard},
	{key: "manamax", panel: PanelFieldManaCard, second: true},
	{key: "body", panel: PanelFieldBody},
	{key: "reaction", panel: PanelFieldReaction},
	{key: "mind", panel: PanelFieldMind},
	{key: "spirit", panel: PanelFieldSpirit},
	{key: "absorption", panel: PanelFieldAbsorption},
	{key: "defence", panel: PanelFieldDefence},
	{key: "sight", panel: PanelFieldSight},
	{key: "speed", panel: PanelFieldMoveSpeed},
	{key: "skill.blade", panel: PanelFieldSkillBlade},
	{key: "skill.axe", panel: PanelFieldSkillAxe},
	{key: "skill.bludgeon", panel: PanelFieldSkillBludgeon},
	{key: "skill.pike", panel: PanelFieldSkillPike},
	{key: "skill.shooting", panel: PanelFieldSkillShooting},
	{key: "protection.fire", panel: PanelFieldProtFire},
	{key: "protection.water", panel: PanelFieldProtWater},
	{key: "protection.air", panel: PanelFieldProtAir},
	{key: "protection.earth", panel: PanelFieldProtEarth},
	{key: "protection.astral", panel: PanelFieldProtAstral},
}

// cardFields are the fields of the shown window the card edits, by unitCard
// entry, -1 for an entry no editable field fills.
func (e *MapEditor) cardFields() []int {
	out := make([]int, len(unitCard))
	for k, c := range unitCard {
		out[k] = slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == c.key && !f.ReadOnly })
	}
	return out
}

// onCard reports that field i stands on the card, as an input where the
// card draws it or as a row under the card.
func (e *MapEditor) onCard(i int) bool {
	return e.obj.fields[i].Section == cardSection || slices.Contains(e.cardFields(), i)
}

// heldCardRows adds the card block at the end of column column: the row that
// holds the card, or in a band window the band under the rows that holds the
// card and the figure; an input where the card draws each value it edits,
// then each card field the card does not draw as a row of its own.
func (e *MapEditor) heldCardRows(s *popupSpec, links *[]int, field func(i int, label string, heads bool) int, textWidth, column int) {
	pitch := fontHeight(e.bodyFont(), 15) + popupRowExtra
	s.CardScale = cardScale
	card := cardSize(specWindow(*s)).Y
	if e.obj.band {
		s.CardBelow = true
		s.BandLines = (max(card, e.unitFigureSize().Y) + 2*popupPad + pitch - 1) / pitch
	} else {
		s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Hold: holdCard, Lines: max(1, (card+pitch-1)/pitch), Column: column}), append(*links, 0)
	}
	start := len(s.Rows)
	e.cardRows(s, links, field, textWidth)
	r := e.doc.Records[e.selected]
	for i, f := range e.obj.fields {
		if e.onCard(i) && e.fieldRow(i) < 0 && !hiddenField(r, f) && (f.Parent == "" || f.Parent == e.obj.detail) {
			field(i, "", true)
		}
	}
	for k := start; k < len(s.Rows); k++ {
		s.Rows[k].Column = column
	}
}

// cardRows stands the card's inputs on the window's card. Each cell the card
// draws with a value becomes a row at the cell's place, read from the card's
// own line report: the input of the field unitCard names for the cell's panel
// field, built by field, or else a line stating the value the map does not
// store.
func (e *MapEditor) cardRows(s *popupSpec, links *[]int, field func(i int, label string, heads bool) int, textWidth int) {
	o := &e.obj
	if s.Card == nil {
		return
	}
	font, body := s.CardFont, e.bodyFont()
	if font == nil {
		font = body
	}
	if font == nil {
		return
	}
	edits, maxEdits, inputs := map[PanelField]int{}, map[PanelField]int{}, map[int]cardInput{}
	for k, i := range e.cardFields() {
		if i < 0 {
			continue
		}
		c := unitCard[k]
		inputs[i] = c
		if c.second {
			maxEdits[c.panel] = i
		} else {
			edits[c.panel] = i
		}
	}
	layout := CompactPanelLayout(nil)
	lineH := font.Height() + layout.Gap
	// An input holds four digits of the window's own font, or the digits
	// of its field's largest value when that has more.
	holds := func(i int) int {
		if body == nil {
			return 0
		}
		return body.Advance(strings.Repeat("0", max(4, o.fields[i].ValueWidth))) + 2*editTextX
	}
	// A stat whose stored value differs from the card's value, the one the
	// game uses, is stated in words beside the card.
	var differ []string
	// input stands field i's row on box, where the card states value; it
	// reports false when the field has no row.
	input := func(i int, label, value string, box image.Rectangle, start, limit int) bool {
		n := field(i, "", false)
		if n < 0 {
			return false
		}
		// The card states the value; the row is its input, the field's
		// notes its tip.
		tip := []string{e.editorHostText(o.fields[i].Label)}
		if o.fields[i].Inherit && !inputs[i].whole {
			tip = append(tip, e.groupNote(o.fields[i].Group))
		}
		var meaning []string
		if h := s.Rows[n].Help; h != "" {
			tip, meaning = append(tip, h), append(meaning, h)
		}
		for _, r := range s.Rows[n+1:] {
			tip = append(tip, r.Label)
			meaning = append(meaning, r.Label)
		}
		s.Rows, *links = s.Rows[:n+1], (*links)[:n+1]
		row := &s.Rows[n]
		if inputs[i].whole {
			box.Min.X, box.Max.X = start, limit
		}
		least := holds(i)
		box.Min.X = max(min(box.Min.X, limit-least), start)
		box.Max.X = min(max(box.Max.X, box.Min.X+least), limit)
		box.Min.Y++
		box.Max.Y--
		row.Place, row.Buttons, row.Tip = box, nil, strings.Join(slices.DeleteFunc(tip, func(t string) bool { return t == "" }), " ")
		if stored := strings.TrimSpace(row.Value); stored != "" && !inputs[i].whole && !sameNumber(stored, value) {
			name := strings.TrimSpace(label)
			if name == "" {
				name = e.editorHostText(o.fields[i].Label)
			}
			why := strings.Join(meaning, " ")
			if why == "" {
				why = fmt.Sprintf(e.word("editor.plain.derived"), value)
			}
			differ = append(differ, name+": "+why)
		}
		return true
	}
	cell := func(f PanelField, label, value string, pen, start, end, y int) {
		if value == "" {
			return
		}
		// A pair such as health's hp/max has an input on each number when
		// the map stores both.
		number, rest, pair := strings.Cut(value, "/")
		box := image.Rectangle{Min: image.Pt(pen, y).Mul(cardScale), Max: image.Pt(pen+font.Advance(number), y+lineH).Mul(cardScale)}
		if !pair {
			box.Max.X = (pen + font.Advance(value)) * cardScale
		}
		limit := end * cardScale
		if pair {
			limit = min(limit, box.Max.X)
		}
		// The second input stands after the "/" and may reach the card's
		// edge when the cell ends there; when that room cannot hold it, it
		// takes the room and the first input ends before it.
		second := image.Rectangle{Min: image.Pt(pen+font.Advance(number+"/"), y).Mul(cardScale), Max: image.Pt(pen+font.Advance(value), y+lineH).Mul(cardScale)}
		reach := end * cardScale
		if end == layout.Size.X-layout.Pad.X {
			reach = layout.Size.X * cardScale
		}
		secondMax, hasMax := maxEdits[f]
		hasMax = hasMax && pair
		secondStart := second.Min.X
		if hasMax {
			secondStart = min(secondStart, reach-holds(secondMax))
			limit = min(limit, secondStart)
		}
		if i, ok := edits[f]; ok && input(i, label, number, box, start*cardScale, limit) {
			if hasMax {
				input(secondMax, "", rest, second, secondStart, reach)
			}
			return
		}
		s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: label, Value: value, Place: box,
			Tip: fmt.Sprintf(e.word("editor.plain.fixed"), strings.TrimSpace(label+" "+value))}), append(*links, 0)
	}
	// The SPELLCASTER caption the card draws opens the unit's spell book: a
	// row over the caption, the field that holds the book its field.
	book := slices.IndexFunc(o.fields, func(f InspectionField) bool { return f.Book != nil })
	caption := func(f PanelField, label string, x, y int) {
		if f != PanelFieldSpellcaster || book < 0 || label == "" {
			return
		}
		w, _ := font.Measure(label)
		o.rowField[len(s.Rows)] = book
		s.Rows, *links = append(s.Rows, popupRow{Kind: popupLink, Over: []PanelField{f}, Label: label, Tip: e.editorHostText(o.fields[book].Label),
			Place: image.Rectangle{Min: image.Pt(x, y).Mul(cardScale), Max: image.Pt(x+w, y+lineH).Mul(cardScale)}}), append(*links, 0)
	}
	for _, r := range CharacterPanelReport(layout, font, *s.Card) {
		caption(r.Field, r.Label, r.At.X+r.Indent, r.At.Y)
		if r.Right {
			caption(r.RightField, r.RightLabel, r.At.X+r.RightX, r.At.Y)
		}
		left := r.At.X + r.Indent
		labelEnd := left
		if r.Label != "" {
			width, _ := font.Measure(r.Label)
			labelEnd += width + layout.LabelGap
		}
		end := layout.Size.X - layout.Pad.X
		if r.Right {
			end = r.At.X + r.RightX - max(1, layout.ColumnGap/2)
		}
		if r.Field == PanelFieldName {
			// The name's input takes the card's whole line.
			labelEnd = layout.Pad.X
		}
		cell(r.Field, r.Label, r.Value, left+r.ValueX, labelEnd, end, r.At.Y)
		if r.Right {
			right := r.At.X + r.RightX
			if r.RightLabel != "" {
				width, _ := font.Measure(r.RightLabel)
				right += width + layout.LabelGap
			}
			cell(r.RightField, r.RightLabel, r.RightValue, r.At.X+r.RightX+r.RightValueX, right, layout.Size.X-layout.Pad.X, r.At.Y)
		}
	}
	for _, line := range differ {
		for _, part := range editorTextLines(body, line, textWidth) {
			s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: part}), append(*links, 0)
		}
	}
}

// UnitFieldPlaces names, for each field the shown unit window has a row for
// in the shown mode, where the row stands: "card" for an input on the card,
// else the field's section. It is nil when no unit window shows.
func (e *MapEditor) UnitFieldPlaces() map[string]string {
	s, _, _ := e.objectSpec()
	if !e.unitWindowShown() {
		return nil
	}
	out := map[string]string{}
	for row, i := range e.obj.rowField {
		key := e.obj.fields[i].Key
		switch {
		case s.Rows[row].placed():
			out[key] = "card"
		case out[key] == "":
			out[key] = e.obj.fields[i].Section
		}
	}
	return out
}
