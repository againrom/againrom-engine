package game

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The host of the editor's Add window. Each kind of new record is a row of
// newRecordKinds: the existing host call that adds it, the values that call
// takes, and the record kind every other value is a field of. AddRecord runs
// the call and then the fields as SetFields does, on one model and as one
// history entry.

// newRecordKind is how one kind of new record is added: the record kind the
// values add does not take are fields of ("" for none), the keys add needs,
// the keys it also takes, and whether the record stands on a cell.
type newRecordKind struct {
	edit     string
	required []string
	optional []string
	spatial  bool
	add      func(s *mapInspectionEdits, r ui.InspectionNewRecord) error
}

var newRecordKinds = map[string]newRecordKind{
	"Unit": {edit: "Unit", required: []string{"class"}, optional: []string{"tier"}, spatial: true,
		add: func(s *mapInspectionEdits, r ui.InspectionNewRecord) error {
			_, err := s.AddUnitTier(r.Cell, recordValue(r, "class"), recordValue(r, "tier"))
			return err
		}},
	"Structure": {edit: "Structure", required: []string{"kind"}, spatial: true,
		add: func(s *mapInspectionEdits, r ui.InspectionNewRecord) error {
			_, err := s.AddStructure(r.Cell, recordValue(r, "kind"))
			return err
		}},
	// A sack is added with its first item; each other item is the
	// sack's "additem" field, as the sack's popup adds one.
	"Sack": {edit: "Sack", required: []string{"item"}, spatial: true,
		add: func(s *mapInspectionEdits, r ui.InspectionNewRecord) error {
			items := recordValues(r, "item")
			if _, err := s.AddSack(r.Cell, items[0]); err != nil {
				return err
			}
			for _, item := range items[1:] {
				if _, err := s.SetFields("Sack", s.batch.Count(mapedit.KindLoot)-1, []ui.InspectionFieldValue{{Key: "additem", Value: item}}); err != nil {
					return err
				}
			}
			return nil
		}},
	"Player": {edit: "Player", add: func(s *mapInspectionEdits, r ui.InspectionNewRecord) error {
		_, err := s.AddPlayer()
		return err
	}},
	"Trigger":   {edit: "Trigger", add: addScriptOf("Trigger")},
	"Action":    {edit: "Action", add: addScriptOf("Action")},
	"Condition": {edit: "Condition", add: addScriptOf("Condition")},
}

func addScriptOf(kind string) func(*mapInspectionEdits, ui.InspectionNewRecord) error {
	return func(s *mapInspectionEdits, _ ui.InspectionNewRecord) error {
		_, err := s.AddScript(kind)
		return err
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

func newRecordKindOf(kind string) (newRecordKind, error) {
	k, ok := newRecordKinds[kind]
	if !ok {
		return newRecordKind{}, fmt.Errorf("the editor adds no %s record", strings.ToLower(kind))
	}
	return k, nil
}

// AddRecord adds the new record r as one edit: its kind's add call, then
// every value the call does not take as a field of the new record.
func (s *mapInspectionEdits) AddRecord(r ui.InspectionNewRecord) (*ui.InspectionDocument, error) {
	if r.Kind == "Message" {
		return s.NewMessage("Message", 0, "")
	}
	k, err := newRecordKindOf(r.Kind)
	if err != nil {
		return nil, err
	}
	if !k.complete(r) {
		return nil, fmt.Errorf("the new %s lacks a value", strings.ToLower(r.Kind))
	}
	for _, v := range r.Values {
		if err := s.checkNewValue(r, v.Key); err != nil {
			return nil, err
		}
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		return true, s.addOn(m, k, r)
	})
}

// addOn runs r's calls on m and folds them into one history entry.
func (s *mapInspectionEdits) addOn(m *mapedit.Editor, k newRecordKind, r ui.InspectionNewRecord) error {
	mark := m.Mark()
	s.batch = m
	defer func() { s.batch = nil }()
	if err := k.add(s, r); err != nil {
		return err
	}
	if k.edit != "" {
		var fields []ui.InspectionFieldValue
		for _, v := range r.Values {
			if !k.takes(v.Key) {
				fields = append(fields, v)
			}
		}
		if len(fields) > 0 {
			if _, err := s.SetFields(k.edit, s.newIndex(m, k), fields); err != nil {
				return err
			}
		}
	}
	m.Squash(mark)
	return nil
}

// newIndex is the index of the record the kind's add call appended to m.
func (s *mapInspectionEdits) newIndex(m *mapedit.Editor, k newRecordKind) int {
	ek, err := editKind(k.edit)
	if err != nil {
		return -1
	}
	return m.Count(ek) - 1
}

// trialAdd runs r's calls on a copy of the model, which it then drops, and
// returns the copy and the new record's index.
func (s *mapInspectionEdits) trialAdd(k newRecordKind, r ui.InspectionNewRecord) (*mapedit.Editor, int, error) {
	m := s.model.Clone()
	if err := s.addOn(m, k, r); err != nil {
		return nil, -1, err
	}
	return m, s.newIndex(m, k), nil
}

// CheckNewRecord refuses the value r holds for key: its own range and
// reference, and, once r holds every value its add call needs, the whole
// record as AddRecord would add it.
func (s *mapInspectionEdits) CheckNewRecord(r ui.InspectionNewRecord, key string) error {
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
	_, _, err = s.trialAdd(k, r)
	return err
}

// checkNewValue is the range and reference check of r's values of key.
func (s *mapInspectionEdits) checkNewValue(r ui.InspectionNewRecord, key string) error {
	view, err := s.model.Map()
	if err != nil {
		return err
	}
	for _, v := range recordValues(r, key) {
		switch key {
		case "class":
			if v < math.MinInt16 || v > math.MaxInt16 {
				return fmt.Errorf("class key %d is outside %d..%d", v, math.MinInt16, math.MaxInt16)
			}
			if _, err := s.inspector.secondaryKey(int16(v)); err != nil {
				return err
			}
		case "tier":
			class := recordValue(r, "class")
			if v < 0 || v > math.MaxUint16 || class < math.MinInt16 || class > math.MaxInt16 {
				return fmt.Errorf("tier %d is outside 0..%d", v, math.MaxUint16)
			}
			if v != 0 {
				if err := s.inspector.tierError(alm.Unit{ClassID: int16(class), ClassSubID: uint16(v)}); err != nil {
					return err
				}
			}
		case "owner":
			if v < 1 || v > math.MaxUint32 {
				return fmt.Errorf("owner slot %d is not in the map's roster", v)
			}
			if err := rosterSlotError(view, uint32(v)); err != nil {
				return err
			}
		case "kind":
			if err := s.inspector.checkStructureKind(v); err != nil {
				return err
			}
		case "item":
			if v <= 0 || v > math.MaxUint16 {
				return fmt.Errorf("item code %d is outside 1..65535", v)
			}
			if err := s.inspector.checkItem(v); err != nil {
				return err
			}
		}
	}
	if key == "cell" && r.HasCell {
		return insideMap(s.model, r.Cell, "cell")
	}
	return nil
}

// NewRecordStep offers step key of the new record r: the choices it may
// take, given the values before it, and the value it starts at. A field of
// the new record starts at the value its kind's bare add call writes.
func (s *mapInspectionEdits) NewRecordStep(r ui.InspectionNewRecord, key string) ui.InspectionNewStep {
	var out ui.InspectionNewStep
	first := func() {
		if len(out.Choices) > 0 {
			out.Value = out.Choices[0].Value
		}
	}
	switch key {
	case "class":
		out.Choices = s.RecordChoices("Unit", -1, "class")
		first()
	case "tier":
		out.Choices, out.Value = s.UnitTiers(recordValue(r, "class"))
	case "kind":
		out.Choices = s.RecordChoices("Structure", -1, "kind")
		first()
	case "item":
		out.Choices = s.RecordChoices("Sack", -1, "item")
	case "cell":
	default:
		out.Value, out.Text = s.newFieldDefault(r, key)
		switch key {
		case "owner":
			out.Choices = s.RecordChoices(r.Kind, -1, "owner")
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
// from its bare add call, made with r's values for that call, on cell 0,0
// before a cell is chosen.
func (s *mapInspectionEdits) newFieldDefault(r ui.InspectionNewRecord, key string) (int64, string) {
	k, err := newRecordKindOf(r.Kind)
	if err != nil || k.edit == "" {
		return 0, ""
	}
	bare := ui.InspectionNewRecord{Kind: r.Kind, Cell: r.Cell, HasCell: true}
	for _, v := range r.Values {
		if k.takes(v.Key) {
			bare.Values = append(bare.Values, v)
		}
	}
	if !k.complete(bare) {
		return 0, ""
	}
	m, index, err := s.trialAdd(k, bare)
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

// NewRecordPreview is a unit class's standing sprite or a structure kind's
// information picture.
func (s *mapInspectionEdits) NewRecordPreview(kind, key string, value int64) *image.RGBA {
	x := s.inspector
	switch key {
	case "class":
		if x.units == nil || value < math.MinInt16 || value > math.MaxInt16 {
			return nil
		}
		c := x.units.Classes[int32(value)]
		frames := c.TierFrames(0)
		if c == nil || len(frames) == 0 {
			return nil
		}
		idx, _ := terrain.SelectUnitFrame(c.Anim, len(frames), false, 0, 0, 0)
		return frames[idx].RGBA()
	case "kind":
		if x.structures == nil || value <= 0 || value >= int64(len(x.structures.Classes)) {
			return nil
		}
		if c := x.structures.Classes[value]; c != nil {
			return c.Portrait
		}
	}
	return nil
}

var _ ui.InspectionNewRecords = (*mapInspectionEdits)(nil)
