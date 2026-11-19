package ui

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"
)

// A new record is assembled from its kind's steps, kept as data (addKindForms):
// each step is one value the host offers choices and a default for and
// checks. The palettes place records through it, one record and one history
// entry per click, the one-click adds take every default, and the add_record
// and add_steps commands walk the same steps headless (newRecordOf).

// InspectionNewRecord is a new record as its steps have assembled it so far:
// its kind, the cell once a step chose one, and every other step's value in
// step order. A step that chooses several values gives one entry each. Clip,
// when set, is the paste brush's clip instead: what PasteRecords would place
// with the clip's origin on Cell.
type InspectionNewRecord struct {
	Kind    RecordKind
	Cell    image.Point
	HasCell bool
	Values  []InspectionFieldValue
	Clip    InspectionClip
	Area    image.Rectangle
}

// InspectionNewStep is what the host offers one step: its choices, none for
// a typed value, the value or text the step starts at, and Absent when the
// record as asked so far has no such value.
type InspectionNewStep struct {
	Choices   []InspectionChoice
	Groups    []InspectionChoiceGroup
	Pictures  func([]InspectionChoice) map[int64]*image.RGBA `json:"-"`
	ChoicePad []image.Point
	Value     int64
	Text      string
	Absent    bool
	Label     string
	Bounds    *InspectionStepBounds
	Templates map[int64]InspectionSpellTemplate
	// Palettes is the bar palette each choice of a structure kind step
	// lists in, by value; a value it lacks lists in its own kind's palette.
	Palettes map[int64]Palette
	// Shut are the choice values the list shows but does not take: its
	// headings.
	Shut []int64
}

type InspectionStepBounds struct{ Min, Max int64 }

// InspectionSpellTemplate gives a choice's Where constructor and link target.
// An empty target places the spell on the cell itself.
type InspectionSpellTemplate struct {
	Constructor int64
	Target      RecordKind
}

// InspectionNewRecords is implemented by a host that adds new records.
// NewRecordStep offers step key given the values before it; CheckNewRecord
// refuses the value r holds for key, with the reason; NewRecordPreview is a
// choice's picture, nil for none; AddRecord adds r as one history entry and
// changes nothing when it refuses.
type InspectionNewRecords interface {
	NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep
	CheckNewRecord(r InspectionNewRecord, key string) error
	NewRecordPreview(kind RecordKind, key string, value int64) *image.RGBA
	AddRecord(r InspectionNewRecord) (*InspectionDocument, error)
	// AddKinds are the record kinds the open install assembles, in no order.
	AddKinds() []RecordKind
}

// addControl is the kind of value a step takes.
type addControl uint8

const (
	addChoice addControl = iota // one of the host's choices
	addNumber                   // a whole number
	addName                     // text
	addCell                     // a cell on the map
	addMany                     // one or more of the host's choices
)

// addStep is one step: the host's key for its value, its label's word and
// the kind of its value. An optional step is left out when the host marks it
// Absent or a choice step is offered fewer than two choices.
type addStep struct {
	Key      string
	Word     string
	Control  addControl
	Optional bool
}

// addKindRow is one record kind a new record is assembled for: the kind the
// host adds, its name's word, the list views it lists in and its steps.
type addKindRow struct {
	Kind    RecordKind
	Word    string
	Filters []int
	Steps   []addStep
	// OneClick kinds are added by the list's Add button and the A key, at
	// once with every default; the list's window stays.
	OneClick bool
}

var (
	addOwner = addStep{Key: "owner", Word: "editor.add.owner", Control: addChoice}
	addAt    = addStep{Key: "cell", Word: "editor.object.cell", Control: addCell}
	addNamed = addStep{Key: "name", Word: "editor.add.name", Control: addName}
)

// addKindForms is the form of every kind a new record can be assembled for,
// as data; the host's AddKinds says which of them the open install offers.
var addKindForms = []addKindRow{
	{Kind: KindEnchantment, Word: "editor.cast.add", Filters: []int{5, editorCellSpellsFilter}, Steps: []addStep{
		{Key: "template", Word: "editor.cast.template", Control: addChoice},
		{Key: "link", Word: "editor.cast.link", Control: addNumber}, addAt,
		{Key: "spell", Word: "editor.cast.spell", Control: addChoice},
		{Key: "power", Word: "editor.cast.powerstep", Control: addNumber},
		{Key: "castkind", Word: "editor.cast.kind", Control: addChoice, Optional: true},
		{Key: "range", Word: "editor.cast.range", Control: addNumber, Optional: true}}},
	{Kind: KindUnit, Word: "editor.add.unit", Filters: []int{1}, Steps: []addStep{
		{Key: "identity", Word: "editor.unit.field.identity", Control: addChoice},
		addOwner, {Key: "group", Word: "editor.unit.field.group", Control: addNumber},
		{Key: "facing", Word: "editor.unit.field.facing", Control: addChoice, Optional: true}, addAt}},
	{Kind: KindStructure, Word: "editor.add.structure", Filters: []int{2}, Steps: []addStep{
		{Key: "kind", Word: "editor.add.kind", Control: addChoice}, addOwner, addAt}},
	{Kind: KindSack, Word: "editor.add.sack", Filters: []int{3}, Steps: []addStep{
		addAt, {Key: "item", Word: "editor.add.items", Control: addMany}}},
	{Kind: KindPlayer, Word: "editor.add.player", Filters: []int{editorPlayersFilter}, Steps: []addStep{
		addNamed, {Key: "colour", Word: "editor.add.colour", Control: addNumber},
		{Key: "control", Word: "editor.add.control", Control: addChoice}}, OneClick: true},
	{Kind: KindTrigger, Word: "editor.add.trigger", Steps: []addStep{
		addNamed, {Key: "once", Word: "editor.trigger.once", Control: addChoice}}},
	{Kind: KindMessage, Word: "editor.message.new"},
	{Kind: KindAction, Word: "editor.add.action"},
	{Kind: KindCondition, Word: "editor.add.condition"},
}

// addKindRows are the forms of the kinds the open install's host offers, in
// the forms' order; none without a host.
func (e *MapEditor) addKindRows() []addKindRow {
	host, ok := e.addHost()
	if !ok {
		return nil
	}
	offered := host.AddKinds()
	var out []addKindRow
	for _, k := range addKindForms {
		if slices.Contains(offered, k.Kind) {
			out = append(out, k)
		}
	}
	return out
}

// offersKind reports that the open install's host assembles kind.
func (e *MapEditor) offersKind(kind RecordKind) bool {
	host, ok := e.addHost()
	return ok && slices.Contains(host.AddKinds(), kind)
}

// addKindOf is kind's form.
func addKindOf(kind RecordKind) (addKindRow, bool) {
	for _, k := range addKindForms {
		if k.Kind == kind {
			return k, true
		}
	}
	return addKindRow{}, false
}

// addKindFor is the kind the list view filter lists.
func (e *MapEditor) addKindFor(filter int) (addKindRow, bool) {
	for _, k := range e.addKindRows() {
		for _, f := range k.Filters {
			if f == filter {
				return k, true
			}
		}
	}
	return addKindRow{}, false
}

// addRun is one new record's assembly: its kind, its steps, each step's
// text, each many step's values and the steps the host leaves out. A
// palette's placing keeps one as the placing prompt (promptPlace): placed
// counts the records it placed, note is its last count or refusal, stroke
// is a press held over the map, and brush marks a placing the object
// brush's grid began, which Esc ends back in that brush.
type addRun struct {
	kind   RecordKind
	steps  []addStep
	texts  []string
	many   [][]int64
	skip   []bool
	note   string
	placed int
	stroke *placeStroke
	brush  bool
}

func (e *MapEditor) addHost() (InspectionNewRecords, bool) {
	if !e.editable() || e.catalog {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionNewRecords)
	return h, ok
}

func (r *addRun) setSteps(steps []addStep) {
	r.steps = append([]addStep(nil), steps...)
	n := len(r.steps)
	r.texts, r.many, r.skip = make([]string, n), make([][]int64, n), make([]bool, n)
}

// addRunning is the palette's placing run, ok false while none is armed.
func (e *MapEditor) addRunning() (*addRun, bool) {
	r := e.prompt.add
	return r, e.prompt.kind == promptPlace && r != nil
}

// addRecord is the record of every step before upTo the host did not leave
// out.
func (e *MapEditor) addRecord(r *addRun, upTo int) InspectionNewRecord {
	out := InspectionNewRecord{Kind: r.kind}
	for i := 0; i < upTo && i < len(r.steps); i++ {
		st, text := r.steps[i], strings.TrimSpace(r.texts[i])
		if r.skip[i] {
			continue
		}
		switch st.Control {
		case addCell:
			if c, ok := parseCell(text); ok {
				out.Cell, out.HasCell = c, true
			}
		case addName:
			out.Values = append(out.Values, InspectionFieldValue{Key: st.Key, Text: true, TextValue: r.texts[i]})
		case addMany:
			for _, v := range r.many[i] {
				out.Values = append(out.Values, InspectionFieldValue{Key: st.Key, Value: v})
			}
		default:
			if v, err := strconv.ParseInt(text, 10, 64); err == nil {
				out.Values = append(out.Values, InspectionFieldValue{Key: st.Key, Value: v})
			}
		}
	}
	return out
}

// addPlacement is the record a click on cell places, and the placement
// preview's: the placing run with the palette's values (placeValues),
// completed by addAccepted. A structure the Ruined switch marks takes the
// ruined placement.
func (e *MapEditor) addPlacement(cell image.Point) InspectionNewRecord {
	r, ok := e.addRunning()
	host, hok := e.addHost()
	if !ok || !hok {
		return InspectionNewRecord{}
	}
	run := *r
	run.texts, run.skip = slices.Clone(r.texts), slices.Clone(r.skip)
	run.many = make([][]int64, len(r.many))
	for i, m := range r.many {
		run.many[i] = slices.Clone(m)
	}
	e.placeValues(&run)
	out := e.addAccepted(host, &run, cell)
	if !run.brush && run.kind == KindStructure && e.place.ruined {
		out.Values = append(out.Values, InspectionFieldValue{Key: "placement", Value: 0})
	}
	return out
}

// addAccepted completes run, every step without a value taking the host's
// default, then assembles it with addRecord: the cell is the cell step's
// answer, a step the host leaves out stays out, and a step with no answer
// takes its default (addDefault; a many step with no value the first
// choice).
func (e *MapEditor) addAccepted(host InspectionNewRecords, run *addRun, cell image.Point) InspectionNewRecord {
	for i, st := range run.steps {
		if st.Control == addCell {
			run.texts[i] = fmt.Sprintf("%d,%d", cell.X, cell.Y)
			continue
		}
		var offer InspectionNewStep
		offer, run.skip[i] = e.addOffer(host, run, i)
		if run.skip[i] {
			run.texts[i] = ""
			continue
		}
		switch st.Control {
		case addName:
			run.texts[i] = addDefault(st, offer)
		case addMany:
			if len(run.many[i]) == 0 && len(offer.Choices) > 0 {
				run.many[i] = []int64{offer.Choices[0].Value}
			}
		default:
			if _, err := strconv.ParseInt(strings.TrimSpace(run.texts[i]), 10, 64); err != nil {
				run.texts[i] = addDefault(st, offer)
			}
		}
	}
	return e.addRecord(run, len(run.steps))
}

func parseCell(s string) (image.Point, bool) {
	a, b, ok := strings.Cut(s, ",")
	x, ex := strconv.Atoi(strings.TrimSpace(a))
	y, ey := strconv.Atoi(strings.TrimSpace(b))
	return image.Pt(x, y), ok && ex == nil && ey == nil
}

// addOffer is step i of r: the host's offer for the record of the kept steps
// before it, and whether the step is left out. Placing and add_record both
// walk through it.
func (e *MapEditor) addOffer(host InspectionNewRecords, r *addRun, i int) (InspectionNewStep, bool) {
	st := r.steps[i]
	offer := host.NewRecordStep(e.addRecord(r, i), st.Key)
	return offer, st.Optional && (offer.Absent || st.Control != addNumber && len(offer.Choices) < 2)
}

// addDefault is the text a step starts at: the offer's value, its text for a
// name, nothing for a cell or a many step.
func addDefault(st addStep, offer InspectionNewStep) string {
	switch st.Control {
	case addCell, addMany:
		return ""
	case addName:
		return offer.Text
	}
	return strconv.FormatInt(offer.Value, 10)
}

// addPicked places the record of a click at p on cell. A spell placed on a
// unit or structure takes the number of the one under p as its link; a click
// on none places nothing and names what the palette places on.
func (e *MapEditor) addPlacementAt(p, cell image.Point) (InspectionNewRecord, bool) {
	record := e.addPlacement(cell)
	if record.Kind != KindEnchantment {
		return record, true
	}
	target := e.castTemplateOffer().Templates[e.castTemplate()].Target
	if target == "" {
		return record, true
	}
	hits, _ := e.recordHitsAt(p, target)
	if len(hits) == 0 {
		return record, false
	}
	if i := slices.IndexFunc(record.Values, func(v InspectionFieldValue) bool { return v.Key == "link" }); i >= 0 {
		record.Values[i].Value = e.doc.Records[hits[0]].SpellLink
	}
	return record, true
}

func (e *MapEditor) addPicked(p, cell image.Point) {
	r, ok := e.addRunning()
	if !ok {
		return
	}
	record, ok := e.addPlacementAt(p, cell)
	if !ok {
		r.note = e.castWhere().Label
		e.keepPick(pickPlace)
		return
	}
	ch, err := e.commandAddRecord(record)
	if err != nil {
		r.note = e.editorHostText(err.Error())
		e.armPick(pickPlace)
		return
	}
	if ch.doc != nil {
		e.placedNote(r)
		e.keepPick(pickPlace)
	}
}

// paletteHost is the host the palettes list from: the map's editing host, or
// for a map with none the document's list-only Palettes.
func (e *MapEditor) paletteHost() (InspectionNewRecords, bool) {
	if h, ok := e.addHost(); ok {
		return h, true
	}
	if e.doc != nil && e.doc.Edits == nil && !e.catalog && e.doc.Palettes != nil {
		return e.doc.Palettes, true
	}
	return nil, false
}
