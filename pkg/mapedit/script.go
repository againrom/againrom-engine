package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"

	"againrom/pkg/formats/alm"
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

// scriptLayout locates array a (0 actions, 1 conditions, 2 triggers) in the type-7
// payload. The three arrays must tile the payload exactly, as the engine's
// own walk requires; otherwise no node edit is built over the bytes.
func scriptLayout(a int) func(*Editor, span) (arrayLayout, error) {
	return func(e *Editor, sp span) (arrayLayout, error) {
		p := e.data[sp.payloadOff : sp.payloadOff+sp.payloadSize]
		sizes := scriptArraySizes
		var lay arrayLayout
		off := 0
		for k, size := range sizes {
			if off+alm.CountWordSize > len(p) {
				return lay, fmt.Errorf("the type-7 payload ends before array %d", k)
			}
			n := uint64(binary.LittleEndian.Uint32(p[off:]))
			if n > uint64(len(p)-off-alm.CountWordSize)/uint64(size) {
				return lay, fmt.Errorf("type-7 array %d counts %d records past the payload", k, n)
			}
			start := off + alm.CountWordSize
			if k == a {
				lay = arrayLayout{countAt: sp.abs(off), first: start, n: int(n), after: len(p) - (start + int(n)*size)}
			}
			off = start + int(n)*size
		}
		if off != len(p) {
			return lay, fmt.Errorf("the type-7 payload is not three counted arrays: %d of %d bytes", off, len(p))
		}
		return lay, nil
	}
}

// nodeFields offers identity, parameter kinds and typed values. Unedited
// bytes, including unknown parameter values and slot names, stay carried.
func nodeFields(rec []byte) []Field {
	out := []Field{{Key: "name", Label: "Name", Offset: 0, Width: alm.ScriptLabelLen, Text: &alm.ScriptLabelText},
		{Key: "opcode", Label: "Operation", Offset: alm.ScriptNodeOpcode, Width: 4, Min: 1, Max: math.MaxUint32},
		{Key: "id", Label: "Node ID", Offset: alm.ScriptNodeID, Width: 4, Min: 1, Max: scriptMaxID}}
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

func triggerFields() []Field {
	out := []Field{{Key: "name", Label: "Name", Offset: 0, Width: alm.ScriptLabelLen, Text: &alm.ScriptLabelText}}
	for i := 0; i < 3; i++ {
		out = append(out, Field{Key: fmt.Sprintf("left%d", i), Label: fmt.Sprintf("Pair%d left condition ID (0 empty)", i+1), Offset: alm.ScriptTriggerPairs + 8*i, Width: 4, Max: scriptMaxID},
			Field{Key: fmt.Sprintf("right%d", i), Label: fmt.Sprintf("Pair%d right condition ID", i+1), Offset: alm.ScriptTriggerPairs + 8*i + 4, Width: 4, Max: scriptMaxID},
			Field{Key: fmt.Sprintf("cmp%d", i), Label: fmt.Sprintf("Pair%d comparison", i+1), Offset: alm.ScriptTriggerCompare + 4*i, Width: 4, Max: 5})
	}
	for i := 0; i < 4; i++ {
		out = append(out, Field{Key: fmt.Sprintf("action%d", i), Label: fmt.Sprintf("Action%d ID (0 empty)", i+1), Offset: alm.ScriptTriggerActions + 4*i, Width: 4, Max: scriptMaxID})
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
		next = append(next, e.data[at:at+n]...)
	}
	e.apply(edit{replace(start, e.data[start:end], next)})
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
	if e.Count(KindTrigger)+1 > scriptTriggers {
		return 0, fmt.Errorf("mapedit: the script already holds %d of %d triggers", e.Count(KindTrigger), scriptTriggers)
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
	trigger := make([]byte, alm.ScriptTriggerSize)
	copy(trigger, dropTrigger)
	binary.LittleEndian.PutUint32(trigger[alm.ScriptTriggerActions:], id)
	binary.LittleEndian.PutUint32(trigger[alm.ScriptTriggerOnce:], 1)
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
