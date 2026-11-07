package mapedit

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"againrom/pkg/formats/alm"
	"againrom/pkg/scriptprofile"
)

// The parameter type codes of ALM-TRIG-046 a node field is offered for.
const (
	paramInt       = 1
	paramGroup     = 2
	paramPlayer    = 3
	paramUnit      = 4
	paramX         = 5
	paramY         = 6
	paramConst     = 7
	paramStructure = 9
)

// ScriptParam is one parameter slot of a new node: a type code and a value.
type ScriptParam struct{ Type, Value uint32 }

func scriptSection(name string, array int) *section {
	size := scriptArraySizes[array]
	sec := &section{name: name, typeID: alm.TypeScript,
		layout:   scriptLayout(array),
		length:   func(_ *Editor, b []byte) int { return fixedLength(b, size) },
		cellless: func([]byte) bool { return true },
		fields:   func(*Editor, []byte) []Field { return nil },
	}
	if array < 2 {
		sec.fields = func(_ *Editor, rec []byte) []Field { return nodeFields(rec) }
	} else {
		sec.fields = func(_ *Editor, rec []byte) []Field { return triggerFields() }
	}
	return sec
}

// scriptArraySizes is the record size of the three arrays of the type-7
// payload, in file order: actions, conditions, triggers.
var scriptArraySizes = [3]int{alm.ScriptNodeSize, alm.ScriptNodeSize, alm.ScriptTriggerSize}

// scriptLayout locates array a (0 actions, 1 conditions, 2 triggers) in the
// type-7 payload through the format's one walk of it. The three arrays must
// tile the payload exactly; otherwise no node edit is built over the bytes.
func scriptLayout(a int) func(*Editor, span) (arrayLayout, error) {
	return func(e *Editor, sp span) (arrayLayout, error) {
		p := e.data[sp.payloadOff : sp.payloadOff+sp.payloadSize]
		f, err := alm.FrameScript(p)
		var fe *alm.ScriptFrameError
		switch {
		case errors.As(err, &fe) && fe.Array < 0:
			return arrayLayout{}, fmt.Errorf("the type-7 payload is not three counted arrays: %d of %d bytes", fe.End, fe.Len)
		case errors.As(err, &fe) && fe.Short:
			return arrayLayout{}, fmt.Errorf("the type-7 payload ends before array %d", fe.Array)
		case errors.As(err, &fe):
			return arrayLayout{}, fmt.Errorf("type-7 array %d counts %d records past the payload", fe.Array, fe.Count)
		case err != nil:
			return arrayLayout{}, err
		}
		arr := f.Arrays()[a]
		return arrayLayout{countAt: sp.abs(arr.CountAt), first: arr.First, n: arr.N, after: len(p) - arr.End()}, nil
	}
}

// nodeFields offers identity, parameter kinds and typed values. Unedited
// bytes, including unknown parameter values and slot names, stay carried.
func nodeFields(rec []byte) []Field {
	out := []Field{{Key: "name", Label: "Name", Offset: 0, Width: alm.ScriptLabelLen, Text: &alm.ScriptLabelText},
		{Key: "opcode", Label: "Operation", Offset: alm.ScriptNodeOpcode, Width: 4, Min: 1, Max: math.MaxUint32},
		{Key: "id", Label: "Node ID", Offset: alm.ScriptNodeID, Width: 4, Min: 1, Max: int64(script.MaxID)}}
	for i := 0; i < alm.ScriptParams; i++ {
		t := binary.LittleEndian.Uint32(rec[alm.ScriptNodeTypes+4*i:])
		out = append(out, Field{Key: fmt.Sprintf("type%d", i), Label: fmt.Sprintf("Par%d kind", i), Offset: alm.ScriptNodeTypes + 4*i, Width: 4, Min: 0, Max: 9})
		f := Field{Key: fmt.Sprintf("value%d", i), Offset: alm.ScriptNodeValues + 4*i, Width: 4}
		switch t {
		case paramInt, paramConst:
			f.Label, f.Signed, f.Min, f.Max = "integer", true, math.MinInt32, math.MaxInt32
		case paramGroup:
			f.Label, f.Min, f.Max, f.Check = "group id", 0, math.MaxUint32, CheckScriptGroup
		case paramPlayer:
			f.Label, f.Min, f.Max, f.Check = "player slot", 1, 8, CheckScriptPlayer
		case paramUnit:
			f.Label, f.Min, f.Max, f.Check = "unit id", 1, math.MaxUint16, CheckScriptUnit
		case paramX:
			f.Label, f.Min, f.Max = "cell X", 0, 255
		case paramY:
			f.Label, f.Min, f.Max = "cell Y", 0, 255
		case paramStructure:
			f.Label, f.Min, f.Max, f.Check = "structure id", 1, math.MaxUint16, CheckScriptStructure
		case 8:
			f.Label, f.Min, f.Max = "item offset (code = offset + 3608)", 0, math.MaxUint16
		default:
			continue
		}
		f.Label = fmt.Sprintf("Par%d %s", i, f.Label)
		out = append(out, f)
	}
	return out
}

// triggerFields is the trigger record's field table: the name, each slot of
// the record and the fire-once flag. SetTriggerSlots writes through it.
func triggerFields() []Field {
	out := []Field{{Key: "name", Label: "Name", Offset: 0, Width: alm.ScriptLabelLen, Text: &alm.ScriptLabelText}}
	maxID := int64(script.MaxID)
	for i := 0; i < alm.ScriptPairs; i++ {
		out = append(out, Field{Key: scriptprofile.LeftKey(i), Label: fmt.Sprintf("Pair%d left condition ID (0 empty)", i+1), Offset: alm.ScriptTriggerPairs + 8*i, Width: 4, Max: maxID},
			Field{Key: scriptprofile.RightKey(i), Label: fmt.Sprintf("Pair%d right condition ID", i+1), Offset: alm.ScriptTriggerPairs + 8*i + 4, Width: 4, Max: maxID},
			Field{Key: scriptprofile.CmpKey(i), Label: fmt.Sprintf("Pair%d comparison", i+1), Offset: alm.ScriptTriggerCompare + 4*i, Width: 4, Max: int64(script.MaxCompare)})
	}
	for i := 0; i < alm.ScriptActionRefs; i++ {
		out = append(out, Field{Key: scriptprofile.ActionKey(i), Label: fmt.Sprintf("Action%d ID (0 empty)", i+1), Offset: alm.ScriptTriggerActions + 4*i, Width: 4, Max: maxID})
	}
	return append(out, Field{Key: "once", Label: "Fire once", Offset: alm.ScriptTriggerOnce, Width: 4, Max: 1})
}

func (e *Editor) Reorder(k Kind, from, to int) error {
	w, err := e.walkIndex(k, "reorder", from)
	if err != nil {
		return err
	}
	if to < 0 || to >= w.n() {
		return fmt.Errorf("mapedit: no %s %d", sectionName(k), to)
	}
	if to == from {
		return nil
	}
	lo, hi := min(from, to), max(from, to)
	start, _ := w.at(lo)
	end, l := w.at(hi)
	end += l
	var next []byte
	relocated := map[int]int{}
	for i := lo; i <= hi; i++ {
		j := i
		if from < to {
			if i == to {
				j = from
			} else {
				j = i + 1
			}
		} else {
			if i == to {
				j = from
			} else {
				j = i - 1
			}
		}
		at, n := w.at(j)
		relocated[at] = start + len(next)
		next = append(next, e.data[at:at+n]...)
	}
	p := replace(start, e.data[start:end], next)
	p.relocated = relocated
	e.apply(edit{p})
	return nil
}

// NewScriptNode builds one node record: a zero record with the label, opcode,
// id and the parameter slots set. At most ten parameters.
func NewScriptNode(label string, opcode, id uint32, params ...ScriptParam) ([]byte, error) {
	if len(params) > alm.ScriptParams {
		return nil, fmt.Errorf("mapedit: a node holds %d parameters, not %d", alm.ScriptParams, len(params))
	}
	if len(label) >= alm.ScriptLabelLen {
		return nil, fmt.Errorf("mapedit: a node label is under %d bytes", alm.ScriptLabelLen)
	}
	rec := make([]byte, alm.ScriptNodeSize)
	copy(rec, label)
	binary.LittleEndian.PutUint32(rec[alm.ScriptNodeOpcode:], opcode)
	binary.LittleEndian.PutUint32(rec[alm.ScriptNodeID:], id)
	for i, p := range params {
		binary.LittleEndian.PutUint32(rec[alm.ScriptNodeValues+4*i:], p.Value)
		binary.LittleEndian.PutUint32(rec[alm.ScriptNodeTypes+4*i:], p.Type)
	}
	return rec, nil
}

// ScriptLimit names a limit of the original trigger record.
type ScriptLimit int

const (
	LimitPairs   ScriptLimit = iota // condition pairs per trigger
	LimitActions                    // action ids per trigger
	LimitCompare                    // the highest comparison code
	LimitID                         // the highest node id a slot names
)

// ScriptLimitError is a trigger value the original record cannot hold:
// Limit is the limit passed, Max its value and Got what the value asks.
type ScriptLimitError struct {
	Limit    ScriptLimit
	Max, Got int64
}

func (e *ScriptLimitError) Error() string {
	switch e.Limit {
	case LimitPairs:
		return fmt.Sprintf("mapedit: a trigger record holds %d condition pairs, not %d", e.Max, e.Got)
	case LimitActions:
		return fmt.Sprintf("mapedit: a trigger record holds %d actions, not %d", e.Max, e.Got)
	case LimitCompare:
		return fmt.Sprintf("mapedit: a comparison code is 0..%d, not %d", e.Max, e.Got)
	}
	return fmt.Sprintf("mapedit: a trigger slot names a node id up to %d, not %d", e.Max, e.Got)
}

// pairSides is the words of one pair in the field table: left, right, code.
const pairSides = 3

// triggerWords is t as the record's slot words beside their fields: each
// pair's left, right and code, then each action, every slot of the record,
// the ones t does not spell empty. A value the record cannot hold is a *ScriptLimitError; kept
// reports the slot words to leave unchecked, the ones the record already
// holds.
func triggerWords(t alm.ScriptTrigger, kept func(f Field, v uint32) bool) ([]Field, []uint32, error) {
	if t.Pairs() > script.Pairs {
		return nil, nil, &ScriptLimitError{Limit: LimitPairs, Max: int64(script.Pairs), Got: int64(t.Pairs())}
	}
	if len(t.Acts) > script.Actions {
		return nil, nil, &ScriptLimitError{Limit: LimitActions, Max: int64(script.Actions), Got: int64(len(t.Acts))}
	}
	fields := triggerFields()
	slots := fields[1 : len(fields)-1]
	words := make([]uint32, 0, len(slots))
	for p := 0; p < alm.ScriptPairs; p++ {
		left, right, cmp := t.Pair(p)
		words = append(words, left, right, cmp)
	}
	for a := 0; a < alm.ScriptActionRefs; a++ {
		words = append(words, t.Act(a))
	}
	for i, v := range words {
		if kept != nil && kept(slots[i], v) {
			continue
		}
		limit, most := LimitID, script.MaxID
		if i < pairSides*alm.ScriptPairs && i%pairSides == pairSides-1 {
			limit, most = LimitCompare, script.MaxCompare
		}
		if v > most {
			return nil, nil, &ScriptLimitError{Limit: limit, Max: int64(most), Got: int64(v)}
		}
	}
	return slots, words, nil
}

// NewScriptTrigger builds one trigger record from t: a zero record with the
// name, the pairs, their codes, the actions and the fire-once word set. It is
// the one encoder of a trigger record; a value the record cannot hold (more
// pairs or actions than it has slots, a code above the alphabet, an id above
// the cap) is a *ScriptLimitError and builds nothing.
func NewScriptTrigger(t alm.ScriptTrigger) ([]byte, error) {
	if len(t.Name) >= alm.ScriptLabelLen {
		return nil, fmt.Errorf("mapedit: a trigger name is under %d bytes", alm.ScriptLabelLen)
	}
	slots, words, err := triggerWords(t, nil)
	if err != nil {
		return nil, err
	}
	rec := make([]byte, alm.ScriptTriggerSize)
	copy(rec, t.Name)
	for i, f := range slots {
		binary.LittleEndian.PutUint32(rec[f.Offset:], words[i])
	}
	binary.LittleEndian.PutUint32(rec[alm.ScriptTriggerOnce:], t.Once)
	return rec, nil
}

// SetTriggerSlots writes t's pairs, codes, actions and fire-once word over
// trigger i as one edit, through the trigger's field table, writing only
// the words that change; its name and the bytes no field names stay. A word
// the record already holds is kept unchecked, so a shipped code outside the
// alphabet survives a write of the other slots; a changed word the record
// cannot hold is a *ScriptLimitError and nothing is written.
func (e *Editor) SetTriggerSlots(i int, t alm.ScriptTrigger) error {
	rec, err := e.Record(KindTrigger, i)
	if err != nil {
		return err
	}
	held := func(f Field, v uint32) bool { return binary.LittleEndian.Uint32(rec[f.Offset:]) == v }
	slots, words, err := triggerWords(t, held)
	if err != nil {
		return err
	}
	var sets []FieldSet
	for k, f := range slots {
		if !held(f, words[k]) {
			sets = append(sets, FieldSet{Key: f.Key, Value: int64(words[k])})
		}
	}
	if binary.LittleEndian.Uint32(rec[alm.ScriptTriggerOnce:]) != t.Once {
		sets = append(sets, FieldSet{Key: "once", Value: int64(t.Once)})
	}
	if len(sets) == 0 {
		return nil
	}
	return e.SetFields(KindTrigger, i, sets...)
}

// NextScriptID is one above the highest id among the records of node array k,
// the id a new node takes (ids are unique within an array, ALM-TRIG-045).
func (e *Editor) NextScriptID(k Kind) (uint32, error) {
	w, err := e.walkSection(k, "number a "+sectionName(k))
	if err != nil {
		return 0, err
	}
	next := uint32(1)
	for i := 0; i < w.n(); i++ {
		at, _ := w.at(i)
		if id := e.u32(at + alm.ScriptNodeID); id >= next {
			if id == math.MaxUint32 {
				return 0, fmt.Errorf("mapedit: node ID space is exhausted")
			}
			next = id + 1
		}
	}
	return next, nil
}

// dropOpcode is the action node whose first two values are a drop cell
// (MISSION-DROP-002).
const dropOpcode = 0x10002

// SetDropCell sets the cell of the map's first drop action (value 0 is X,
// value 1 is Y) as one edit, or appends a drop action when the map has none.
func (e *Editor) SetDropCell(x, y uint32) error {
	w, err := e.walkSection(KindAction, "set the drop cell")
	if err != nil {
		return err
	}
	for i := 0; i < w.n(); i++ {
		at, _ := w.at(i)
		if e.u32(at+alm.ScriptNodeOpcode) == dropOpcode {
			return e.SetFields(KindAction, i, FieldSet{"value0", int64(x)}, FieldSet{"value1", int64(y)})
		}
	}
	if x > 255 || y > 255 {
		return fmt.Errorf("mapedit: a drop cell %d,%d is outside 0..255", x, y)
	}
	id, err := e.NextScriptID(KindAction)
	if err != nil {
		return err
	}
	rec, err := NewScriptNode("Drop location", dropOpcode, id, ScriptParam{paramX, x}, ScriptParam{paramY, y})
	if err != nil {
		return err
	}
	_, err = e.Place(KindAction, rec)
	return err
}

// SetScriptNode rewrites node i of array k (actions or conditions) as one
// edit: its opcode and parameter slots become opcode and params, the slots
// past params are cleared, and its label becomes label when label is not
// empty. Its id and the slot names stay.
func (e *Editor) SetScriptNode(k Kind, i int, label string, opcode uint32, params ...ScriptParam) error {
	if k != KindAction && k != KindCondition {
		return fmt.Errorf("mapedit: %s holds no script node", sectionName(k))
	}
	if len(params) > alm.ScriptParams {
		return fmt.Errorf("mapedit: a node holds %d parameters, not %d", alm.ScriptParams, len(params))
	}
	if len(label) >= alm.ScriptLabelLen {
		return fmt.Errorf("mapedit: a node label is under %d bytes", alm.ScriptLabelLen)
	}
	w, err := e.walkIndex(k, "rewrite", i)
	if err != nil {
		return err
	}
	at, l := w.at(i)
	rec := e.data[at : at+l]
	next := cloneBytes(rec)
	if label != "" {
		clear(next[:alm.ScriptLabelLen])
		copy(next, label)
	}
	binary.LittleEndian.PutUint32(next[alm.ScriptNodeOpcode:], opcode)
	for s := 0; s < alm.ScriptParams; s++ {
		var p ScriptParam
		if s < len(params) {
			p = params[s]
		}
		binary.LittleEndian.PutUint32(next[alm.ScriptNodeValues+4*s:], p.Value)
		binary.LittleEndian.PutUint32(next[alm.ScriptNodeTypes+4*s:], p.Type)
	}
	if string(next) == string(rec) {
		return nil
	}
	e.apply(edit{replace(at, rec, next)})
	return nil
}

// The drop zone as the shipped maps write it: an action named "Drop
// Location" holding the cell as an X and a Y slot, and a trigger with no
// condition pair whose first action names it, firing once (scenario 10's
// "Deploy Location"). The builder collects the action whatever trigger names
// it (TRIG-DROP-013); the trigger never runs for want of a pair.
const (
	dropLabel   = "Drop Location"
	dropTrigger = "Deploy Location"
)

// AddDropZone appends a drop action at x, y and its trigger as one edit, and
// returns the new zone's index among the drop actions.
func (e *Editor) AddDropZone(x, y uint32) (int, error) {
	if x > 255 || y > 255 {
		return 0, fmt.Errorf("mapedit: a drop cell %d,%d is outside 0..255", x, y)
	}
	if e.Count(KindTrigger)+1 > script.Triggers {
		return 0, fmt.Errorf("mapedit: the script already holds %d of %d triggers", e.Count(KindTrigger), script.Triggers)
	}
	zones, err := e.DropZones()
	if err != nil {
		return 0, err
	}
	id, err := e.NextScriptID(KindAction)
	if err != nil {
		return 0, err
	}
	action, err := NewScriptNode(dropLabel, dropOpcode, id, ScriptParam{paramX, x}, ScriptParam{paramY, y})
	if err != nil {
		return 0, err
	}
	trigger, err := NewScriptTrigger(alm.ScriptTrigger{Name: dropTrigger, Acts: []uint32{id}, Once: 1})
	if err != nil {
		return 0, err
	}
	if _, err := e.PlaceAll([]Placement{{KindAction, action}, {KindTrigger, trigger}}); err != nil {
		return 0, err
	}
	return len(zones), nil
}

// DropZones are the indices, among the actions, of the drop actions in file
// order: zone z is action DropZones()[z].
func (e *Editor) DropZones() ([]int, error) {
	w, err := e.walkSection(KindAction, "list the drop zones")
	if err != nil {
		return nil, err
	}
	var out []int
	for i := 0; i < w.n(); i++ {
		at, _ := w.at(i)
		if e.u32(at+alm.ScriptNodeOpcode) == dropOpcode {
			out = append(out, i)
		}
	}
	return out, nil
}
