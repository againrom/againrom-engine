package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"
)

// The type-7 payload is three counted arrays: actions and conditions of 796
// bytes, then triggers of 184 (ALM-TRIG-044). A node is an opcode, an id and
// ten parameter slots, each a value and a type code (ALM-TRIG-045,
// ALM-TRIG-046). Action and condition nodes are two rows of the section
// table; their settable fields are the typed parameter slots of the record.
const (
	scriptTypeID      = 7
	scriptNodeSize    = 796
	scriptTriggerSize = 184
	scriptCountSize   = 4

	nodeLabelLen = 64
	nodeOpcode   = 0x40
	nodeID       = 0x44
	nodeValues   = 0x4c
	nodeTypes    = 0x74
	nodeSlots    = 10
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
	sec := &section{name: name, typeID: scriptTypeID,
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
var scriptArraySizes = [3]int{scriptNodeSize, scriptNodeSize, scriptTriggerSize}

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
			if off+scriptCountSize > len(p) {
				return lay, fmt.Errorf("the type-7 payload ends before array %d", k)
			}
			n := uint64(binary.LittleEndian.Uint32(p[off:]))
			if n > uint64(len(p)-off-scriptCountSize)/uint64(size) {
				return lay, fmt.Errorf("type-7 array %d counts %d records past the payload", k, n)
			}
			start := off + scriptCountSize
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
	out := []Field{{Key: "name", Label: "Name", Offset: 0, Width: 64, Text: true},
		{Key: "opcode", Label: "Operation", Offset: nodeOpcode, Width: 4, Min: 1, Max: math.MaxUint32},
		{Key: "id", Label: "Node ID", Offset: nodeID, Width: 4, Min: 1, Max: scriptMaxID}}
	for i := 0; i < nodeSlots; i++ {
		t := binary.LittleEndian.Uint32(rec[nodeTypes+4*i:])
		out = append(out, Field{Key: fmt.Sprintf("type%d", i), Label: fmt.Sprintf("Par%d kind", i), Offset: nodeTypes + 4*i, Width: 4, Min: 0, Max: 9})
		f := Field{Key: fmt.Sprintf("value%d", i), Offset: nodeValues + 4*i, Width: 4}
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
	out := []Field{{Key: "name", Label: "Name", Offset: 0, Width: 64, Text: true}}
	for i := 0; i < 3; i++ {
		out = append(out, Field{Key: fmt.Sprintf("left%d", i), Label: fmt.Sprintf("Pair%d left condition ID (0 empty)", i+1), Offset: triggerPairs + 8*i, Width: 4, Max: scriptMaxID},
			Field{Key: fmt.Sprintf("right%d", i), Label: fmt.Sprintf("Pair%d right condition ID", i+1), Offset: triggerPairs + 8*i + 4, Width: 4, Max: scriptMaxID},
			Field{Key: fmt.Sprintf("cmp%d", i), Label: fmt.Sprintf("Pair%d comparison", i+1), Offset: triggerCompare + 4*i, Width: 4, Max: 5})
	}
	for i := 0; i < 4; i++ {
		out = append(out, Field{Key: fmt.Sprintf("action%d", i), Label: fmt.Sprintf("Action%d ID (0 empty)", i+1), Offset: triggerActions + 4*i, Width: 4, Max: scriptMaxID})
	}
	return append(out, Field{Key: "once", Label: "Fire once", Offset: triggerOnce, Width: 4, Max: 1})
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
	if len(params) > nodeSlots {
		return nil, fmt.Errorf("mapedit: a node holds %d parameters, not %d", nodeSlots, len(params))
	}
	if len(label) >= nodeLabelLen {
		return nil, fmt.Errorf("mapedit: a node label is under %d bytes", nodeLabelLen)
	}
	rec := make([]byte, scriptNodeSize)
	copy(rec, label)
	binary.LittleEndian.PutUint32(rec[nodeOpcode:], opcode)
	binary.LittleEndian.PutUint32(rec[nodeID:], id)
	for i, p := range params {
		binary.LittleEndian.PutUint32(rec[nodeValues+4*i:], p.Value)
		binary.LittleEndian.PutUint32(rec[nodeTypes+4*i:], p.Type)
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
		if id := e.u32(at + nodeID); id >= next {
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
		if e.u32(at+nodeOpcode) == dropOpcode {
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
