package game

import (
	"errors"
	"fmt"
	"image"
	"math"
	"slices"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The host of palette placement and add_record. Each kind is a row of
// newRecordKinds: the existing host call that adds it, the values that call
// takes, and the record kind every other value is a field of. AddRecord runs
// the call and then the fields as SetFields does, on one model and as one
// history entry.

// newRecordKind is how one kind of new record is added: the record kind the
// values add does not take are fields of ("" for none), the keys add needs,
// the keys it also takes, and whether the record stands on a cell.
type newRecordKind struct {
	edit     ui.RecordKind
	required []string
	optional []string
	spatial  bool
	add      func(s *mapInspectionEdits, r ui.InspectionNewRecord) (int, error)
}

var newRecordKinds = map[ui.RecordKind]newRecordKind{
	ui.KindEnchantment: {edit: ui.KindEnchantment, required: []string{"template"}, optional: []string{"link"}, spatial: true,
		add: (*mapInspectionEdits).addTrap},
	ui.KindUnit: {edit: ui.KindUnit, required: []string{"identity"}, spatial: true,
		add: func(s *mapInspectionEdits, r ui.InspectionNewRecord) (int, error) {
			_, err := s.AddUnitIdentity(r.Cell, recordValue(r, "identity"))
			return s.newIndex(s.batch, ui.KindUnit), err
		}},
	ui.KindStructure: {edit: ui.KindStructure, required: []string{"kind"}, spatial: true,
		add: func(s *mapInspectionEdits, r ui.InspectionNewRecord) (int, error) {
			_, err := s.AddStructure(r.Cell, recordValue(r, "kind"))
			return s.newIndex(s.batch, ui.KindStructure), err
		}},
	// A sack is added with its first item; each other item is the
	// sack's "additem" field, as the sack's popup adds one.
	ui.KindSack: {edit: ui.KindSack, required: []string{"item"}, spatial: true,
		add: func(s *mapInspectionEdits, r ui.InspectionNewRecord) (int, error) {
			items := recordValues(r, "item")
			if _, err := s.AddSack(r.Cell, items[0]); err != nil {
				return -1, err
			}
			for _, item := range items[1:] {
				if _, err := s.SetFields(ui.KindSack, s.batch.Count(mapedit.KindLoot)-1, []ui.InspectionFieldValue{{Key: "additem", Value: item}}); err != nil {
					return -1, err
				}
			}
			return s.newIndex(s.batch, ui.KindSack), nil
		}},
	ui.KindPlayer: {edit: ui.KindPlayer, add: func(s *mapInspectionEdits, r ui.InspectionNewRecord) (int, error) {
		_, err := s.AddPlayer()
		return s.newIndex(s.batch, ui.KindPlayer), err
	}},
	ui.KindTrigger:   {edit: ui.KindTrigger, add: addScriptOf(ui.KindTrigger)},
	ui.KindAction:    {edit: ui.KindAction, add: addScriptOf(ui.KindAction)},
	ui.KindCondition: {edit: ui.KindCondition, add: addScriptOf(ui.KindCondition)},
}

// addTrap is the add call of a new trap: addCellSpell on a cell no other
// type-9 record holds.
func (s *mapInspectionEdits) addTrap(r ui.InspectionNewRecord) (int, error) {
	if kind, ok := spellTemplateKind(s.batch, recordValue(r, "template")); ok {
		if err := trapCellFree(s.batch, r.Cell, kind, -1); err != nil {
			return -1, err
		}
	}
	return s.addCellSpell(r)
}

// addCellSpell is the add call of a new type-9 record on s.batch: a copy of
// the record or the constructor r's template names, on r's cell, bound to
// r's link.
func (s *mapInspectionEdits) addCellSpell(r ui.InspectionNewRecord) (int, error) {
	v := recordValue(r, "template")
	kind, ok := spellTemplateKind(s.batch, v)
	if !ok {
		return -1, wordErr("editor.err.spelltemplate")
	}
	if err := s.admitCell(s.batch, r.Cell, "spell cell", ui.KindEnchantment, admitAdd); err != nil {
		return -1, err
	}
	link := recordValue(r, "link")
	if link < 0 || link > math.MaxUint32 {
		return -1, wordErr("editor.err.objectid")
	}
	if err := spellObjectAllowed(s.batch, kind, uint32(link)); err != nil {
		return -1, err
	}
	return s.batch.AddSpellFrom(int(v), uint32(r.Cell.X), uint32(r.Cell.Y), uint32(link))
}

func addScriptOf(kind ui.RecordKind) func(*mapInspectionEdits, ui.InspectionNewRecord) (int, error) {
	return func(s *mapInspectionEdits, _ ui.InspectionNewRecord) (int, error) {
		_, err := s.AddScript(kind)
		return s.newIndex(s.batch, kind), err
	}
}

// takes reports a key the kind's add call takes rather than a field.
func (k newRecordKind) takes(key string) bool {
	return slices.Contains(k.required, key) || slices.Contains(k.optional, key)
}

// complete reports that r holds every value the kind's add call needs.
func (k newRecordKind) complete(r ui.InspectionNewRecord) bool {
	if k.spatial && !r.HasCell {
		return false
	}
	for _, key := range k.required {
		if len(recordValues(r, key)) == 0 {
			return false
		}
	}
	return true
}

func recordValues(r ui.InspectionNewRecord, key string) []int64 {
	var out []int64
	for _, v := range r.Values {
		if v.Key == key && !v.Text {
			out = append(out, v.Value)
		}
	}
	return out
}

// recordValue is r's value of key, 0 when it holds none.
func recordValue(r ui.InspectionNewRecord, key string) int64 {
	if v := recordValues(r, key); len(v) > 0 {
		return v[0]
	}
	return 0
}

func newRecordKindOf(kind ui.RecordKind) (newRecordKind, error) {
	k, ok := newRecordKinds[kind]
	if !ok {
		return newRecordKind{}, fmt.Errorf("the editor adds no %s record", strings.ToLower(kind.String()))
	}
	return k, nil
}

// AddRecord adds the new record r as one edit: its kind's add call, then
// every value the call does not take as a field of the new record.
func (s *mapInspectionEdits) AddRecord(r ui.InspectionNewRecord) (_ *ui.InspectionDocument, said error) {
	if row := s.row(); !row.adds.has(r.Kind) || !row.manyValues && (len(r.Values) > 1 && r.Kind != ui.KindSack && r.Kind != ui.KindEnchantment || len(recordValues(r, "item")) > 1) {
		return nil, s.refuse()
	}
	defer s.worded(&said)
	if r.Kind == ui.KindMessage {
		return s.NewMessage(ui.InspectionMessageOwner{Kind: ui.KindMessage})
	}
	k, err := newRecordKindOf(r.Kind)
	if err != nil {
		return nil, err
	}
	if !k.complete(r) {
		return nil, fmt.Errorf("the new %s lacks a value", strings.ToLower(r.Kind.String()))
	}
	if r.Kind == ui.KindEnchantment {
		if err := s.checkNewValue(r, "spell"); err != nil {
			return nil, err
		}
	}
	for _, v := range r.Values {
		if err := s.checkNewValue(r, v.Key); err != nil {
			return nil, err
		}
	}
	index := -1
	next, err := s.change(func(m *mapedit.Editor) (bool, error) {
		if !r.Area.Empty() {
			return s.addTrapArea(m, k, r, &index)
		}
		var err error
		index, err = s.addOn(m, k, r)
		return err == nil, err
	})
	if next != nil {
		next.AddedRecord = ui.RecordRef{Kind: k.edit, Index: index}
	}
	return next, err
}

// addOn runs r's calls on m and folds them into one history entry.
func (s *mapInspectionEdits) addOn(m *mapedit.Editor, k newRecordKind, r ui.InspectionNewRecord) (int, error) {
	mark := m.Mark()
	s.batch = m
	defer func() { s.batch = nil }()
	index, err := k.add(s, r)
	if err != nil {
		return -1, err
	}
	if k.edit != "" {
		var fields []ui.InspectionFieldValue
		for _, v := range r.Values {
			if !k.takes(v.Key) {
				fields = append(fields, v)
			}
		}
		if len(fields) > 0 {
			if _, err := s.SetFields(k.edit, index, fields); err != nil {
				return -1, err
			}
		}
	}
	m.Squash(mark)
	return index, nil
}

// newIndex is the index of the record the kind's add call appended to m.
func (s *mapInspectionEdits) newIndex(m *mapedit.Editor, kind ui.RecordKind) int {
	ek, err := editKind(kind)
	if err != nil {
		return -1
	}
	return m.Count(ek) - 1
}

// trialAdd runs r's calls on a copy of the model, which it then drops, and
// returns the copy and the new record's index. When checkPlacement refuses
// the new record's footprint, the copy holding it comes back with the
// refusal.
func (s *mapInspectionEdits) trialAdd(k newRecordKind, r ui.InspectionNewRecord) (*mapedit.Editor, int, error) {
	m := s.model.Clone()
	index, err := s.addOn(m, k, r)
	var refusal placementRefusal
	if errors.As(err, &refusal) {
		return m, -1, err
	}
	if err != nil {
		return nil, -1, err
	}
	return m, index, nil
}

// CheckNewRecord refuses the value r holds for key: its own range and
// reference, and, once r holds every value its add call needs, the whole
// record as AddRecord would add it.
func (s *mapInspectionEdits) CheckNewRecord(r ui.InspectionNewRecord, key string) (said error) {
	if !s.row().adds.has(r.Kind) {
		return s.refuse()
	}
	defer s.worded(&said)
	k, err := newRecordKindOf(r.Kind)
	if err != nil {
		return err
	}
	if err := s.checkNewValue(r, key); err != nil {
		return err
	}
	if !k.complete(r) {
		return nil
	}
	if !r.Area.Empty() {
		return s.checkTrapArea(r)
	}
	_, _, err = s.trialAdd(k, r)
	return err
}

// checkNewValue is the range and reference check of r's values of key.
func (s *mapInspectionEdits) checkNewValue(r ui.InspectionNewRecord, key string) error {
	if r.Kind == ui.KindEnchantment && key == "spell" && recordValue(r, key) == 0 {
		return wordErr("editor.err.spellnone")
	}
	view, err := s.model.Map()
	if err != nil {
		return err
	}
	if s.row().playableRing {
		if err := playableAreaError(view, r.Kind, admitAdd); err != nil {
			return err
		}
	}
	for _, v := range recordValues(r, key) {
		switch key {
		case "identity":
			if _, ok := s.identityOf(v); r.Kind == ui.KindUnit && !ok {
				return wordErr("editor.err.identity")
			}
		case "owner":
			if v < 1 || v > math.MaxUint32 {
				return wordErr("editor.err.owner", v)
			}
			if err := rosterSlotError(view, uint32(v)); err != nil {
				return err
			}
		case "kind":
			if err := s.row().structureKind(s.inspector, v); err != nil {
				return err
			}
		case "item":
			if v <= 0 || v > math.MaxUint16 {
				return wordErr("editor.err.itemcode", v)
			}
			if err := s.inspector.checkItem(v); err != nil {
				return err
			}
		}
	}
	if key == "cell" && r.HasCell {
		return s.admitCell(s.model, r.Cell, "cell", r.Kind, admitAdd)
	}
	return nil
}

// NewRecordStep offers step key of the new record r: the choices it may
// take, given the values before it, and the value it starts at. A field of
// the new record starts at the value its kind's bare add call writes.
func (s *mapInspectionEdits) NewRecordStep(r ui.InspectionNewRecord, key string) ui.InspectionNewStep {
	if steps := s.row().newSteps; steps != nil && !steps[key] {
		return ui.InspectionNewStep{Absent: true}
	}
	var out ui.InspectionNewStep
	first := func() {
		if len(out.Choices) > 0 {
			out.Value = out.Choices[0].Value
		}
	}
	switch key {
	case "link":
		if view, err := s.model.Map(); err == nil {
			i := recordValue(r, "template")
			if i >= 0 && i < int64(len(view.Enchantments)) {
				rec := view.Enchantments[i]
				if rec.A == 4 || rec.A == 8 {
					out.Value = int64(uint32(rec.B) | uint32(rec.C)<<16)
				}
			} else if kind, ok := spellTemplateKindIn(view, i); ok {
				out.Value = firstSpellObject(view, kind)
			}
		}
	case "template":
		view, err := s.model.Map()
		if err != nil {
			return out
		}
		return s.cellSpellPalette(view)
	case "castkind", "range", "power":
		// The steps are those of the record the chosen spell makes.
		var sets []mapedit.FieldSet
		if spell := recordValues(r, "spell"); len(spell) > 0 {
			sets = append(sets, mapedit.FieldSet{Key: "spell", Value: spell[0]})
		}
		for _, f := range s.model.SpellTemplateFields(int(recordValue(r, "template")), sets...) {
			if f.Key != key {
				continue
			}
			out.Value, out.Text = s.newFieldDefault(r, key)
			out.Label = s.inspector.inspectionUTF8(s.inspector.word(f.Label))
			out.Bounds = &ui.InspectionStepBounds{Min: f.Min, Max: f.Max}
			if key == "castkind" {
				out.Choices = s.RecordChoices(ui.KindEnchantment, -1, key)
			}
			return out
		}
		out.Absent = true
	case "identity":
		out.Choices, out.Shut = s.inspector.identityChoices(s.mission())
		out.Groups = s.inspector.identityGroups(out.Choices)
		for _, c := range out.Choices {
			if !slices.Contains(out.Shut, c.Value) {
				out.Value = c.Value
				break
			}
		}
	case "kind":
		out.Palettes, _ = structurePalettes(s.inspector.structures, s.inspector.table, s.inspector.editorTables())
		for _, c := range s.RecordChoices(ui.KindStructure, -1, "kind") {
			if out.Palettes[c.Value] == ui.PaletteNone && s.inspector.catalogPalettes() {
				// A list of the catalog leaves no row out.
				out.Palettes[c.Value] = ui.PaletteOwn
			}
			if out.Palettes[c.Value] != ui.PaletteNone {
				out.Choices = append(out.Choices, c)
			}
		}
		first()
	case "item":
		out = s.fieldChoiceOffer(fieldChoiceAsk{kind: ui.KindSack, index: -1, key: "item"})
	case "cell":
	default:
		out.Value, out.Text = s.newFieldDefault(r, key)
		switch key {
		case "spell":
			out.Choices, out.Pictures = s.RecordChoices(ui.KindEnchantment, -1, key), s.spellPictures
		case "owner":
			out.Choices = s.RecordChoices(r.Kind, -1, "owner")
		case "group":
			out.Value = s.newUnitGroup(r)
		case "facing":
			offer := s.fieldChoiceOffer(fieldChoiceAsk{kind: ui.KindUnit, index: -1, key: key})
			out.Choices, out.ChoicePad = offer.Choices, offer.ChoicePad
		case "control":
			out.Choices = []ui.InspectionChoice{{Value: 0, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.add.control.human"))},
				{Value: 1, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.add.control.computer"))}}
		case "once":
			out.Choices = []ui.InspectionChoice{{Value: 1, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.trigger.once.yes"))},
				{Value: 0, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.trigger.once.no"))}}
		}
	}
	return out
}

// newFieldDefault is the value field key of a new record of r's kind takes
// from its bare add call, made with r's values for that call, on the first
// playable cell before a cell is chosen.
func (s *mapInspectionEdits) newFieldDefault(r ui.InspectionNewRecord, key string) (int64, string) {
	k, err := newRecordKindOf(r.Kind)
	if err != nil || k.edit == "" {
		return 0, ""
	}
	bare := ui.InspectionNewRecord{Kind: r.Kind, Cell: r.Cell, HasCell: true}
	if view, err := s.model.Map(); err == nil && !r.HasCell {
		bare.Cell = mapload.Playable(view).Min
	}
	for _, v := range r.Values {
		if k.takes(v.Key) {
			bare.Values = append(bare.Values, v)
		}
	}
	if !k.complete(bare) {
		return 0, ""
	}
	// A footprint refusal still leaves the bare record on the copy, whose
	// fields are the defaults.
	m, index, err := s.trialAdd(k, bare)
	var refusal placementRefusal
	if errors.As(err, &refusal) {
		index, err = s.newIndex(m, k.edit), nil
	}
	if err != nil {
		return 0, ""
	}
	ek, _ := editKind(k.edit)
	fields, err := m.Fields(ek, index)
	if err != nil {
		return 0, ""
	}
	for _, f := range fields {
		if f.Key != key {
			continue
		}
		if f.Text != nil {
			text, _ := m.TextFieldValue(ek, index, key)
			return 0, text
		}
		v, _ := m.FieldValue(ek, index, key)
		return v, ""
	}
	return 0, ""
}

// newUnitGroup is the group the record builder starts a new unit
// in, given the values the record holds before the step (identity and
// owner): the lowest group no unit of the map carries, so that a new unit
// is a script group of its own until it is given another.
func (s *mapInspectionEdits) newUnitGroup(r ui.InspectionNewRecord) int64 {
	return s.freeGroup()
}

// freeGroup is the lowest group number no unit of the map carries, whoever
// owns it: a script names a group by its number alone.
func (s *mapInspectionEdits) freeGroup() int64 {
	view, err := s.model.Map()
	if err != nil {
		return 1
	}
	used := map[int64]bool{}
	for _, u := range view.AuthoredUnits {
		used[int64(u.GroupID)] = true
	}
	n := int64(1)
	for used[n] {
		n++
	}
	return n
}

const sharedStructurePicture = "beeh"

// NewRecordPreview is a unit class's standing sprite, a structure kind's
// information picture (map art for the grid's shared picture), or an object.
func (s *mapInspectionEdits) NewRecordPreview(kind ui.RecordKind, key string, value int64) *image.RGBA {
	x := s.inspector
	switch key {
	case "object":
		if value <= 0 || value > 0xff {
			return nil
		}
		return s.BrushObjectIcon(int(value))
	case "identity":
		return s.identityPicture(value)
	case "identity-grid":
		return s.identityPicturesOf([]ui.InspectionChoice{{Value: value}}, true)[value]
	case "class":
		if value < math.MinInt16 || value > math.MaxInt16 {
			return nil
		}
		return s.unitPreview(x.unitClass(int16(value)), 0)
	case "kind", "kind-grid":
		if x.structures == nil || value <= 0 || value >= int64(len(x.structures.Classes)) {
			return nil
		}
		if c := x.structures.Classes[value]; c != nil {
			_, inTable := mapload.StructureMaxHealth(x.table, byte(value))
			palette, _ := editorPalette(structurePaletteInput(c, inTable, x.editorTables()))
			if key == "kind-grid" && (structureGridMapPalettes[palette] || c.Picture == sharedStructurePicture) {
				w, h := max(3, c.TileWidth), max(2, c.TileHeight)
				g := terrain.Grid{Width: w, Height: h, Structures: []terrain.StructureRecord{{Key: uint32(value), VariableWidth: w, VariableHeight: h}}}
				places, _, _ := terrain.StructurePlacements(g, x.structures, nil, 0)
				var bounds image.Rectangle
				for _, p := range places {
					bounds = bounds.Union(p.Rect())
				}
				if bounds.Empty() {
					if c.PreviewFrame != nil {
						return c.PreviewFrame.RGBA()
					}
					return nil
				}
				pic := image.NewRGBA(image.Rectangle{Max: bounds.Size()})
				for _, p := range places {
					at := p.TopLeft.Sub(bounds.Min)
					terrain.BlitStatic(pic, p.Frame, at.X, at.Y)
				}
				return pic
			}
			return c.Portrait
		}
	}
	return nil
}

// unitPreview is class sheet c's first frame at tier in the direction a
// placed unit faces by default, as the map draws it; nil for a sheet with
// none.
func (s *mapInspectionEdits) unitPreview(c *terrain.UnitClass, tier int) *image.RGBA {
	frames := c.TierFrames(tier)
	if len(frames) == 0 {
		return nil
	}
	idx, _ := terrain.SelectUnitFrame(c.Anim, len(frames), false, 0, 0, 0)
	return frames[idx].RGBA()
}

var _ ui.InspectionNewRecords = (*mapInspectionEdits)(nil)

// AddKinds are the record kinds the open install's tables assemble.
func (s *mapInspectionEdits) AddKinds() []ui.RecordKind {
	return s.inspector.editorTables().addKinds
}
