package alm

// The Againrom script record: the extended machine's triggers, carried in an
// .almx map as one record the original steps over (DIV-4599).
//
// The record's typeId is ScriptRecordTypeID. The original's record switch has
// cases 0..9 and steps over every id at or above TypeIDCount without an error
// (ALM-REQ-055); the second game's dispatcher has cases 0..12 (R2-ASSET-021),
// so the id stays clear of 10..12 and of everything near them. The type-7
// record still carries the action and condition nodes; the triggers that bind
// them move here, and a map carrying this record carries no type-7 trigger.

// Every word is a little-endian uint32. The record is the head, then the
// triggers back to back, and it ends exactly after the last one:
//
//	head     +0x00 version (1), +0x04 variable count, +0x08 trigger count
//	trigger  +0x00 name, 64 bytes, Windows-1251, NUL-padded
//	         +0x40 fire-once word, 0 or 1
//	         +0x44 condition count, +0x48 action count
//	         +0x4c the conditions, 16 bytes each: group, left condition id,
//	               right condition id, comparison code
//	         then the action ids, 4 bytes each

// A trigger's conditions are numbered into groups: the first is group 0 and
// each next one is in its predecessor's group or the one after it. A trigger
// holds when any group holds, and a group holds when all of its conditions do.

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/scriptprofile"
)

// The record's numbers.
const (
	ScriptRecordTypeID        = 100
	ScriptRecordVersion       = 1
	ScriptRecordHeadSize      = 12
	ScriptRecordVariables     = 4 // the head's variable count word
	ScriptRecordTriggerCount  = 8 // the head's trigger count word
	ScriptRecordTriggerHead   = 76
	ScriptRecordConditionSize = 16
	ScriptRecordActionSize    = 4
	ScriptRecordOnce          = 0x40
	ScriptRecordCounts        = 0x44
)

// ScriptRecord is the record decoded: the variable count and the triggers.
// A trigger's Group holds one group number per condition.
type ScriptRecord struct {
	Variables int
	Triggers  []ScriptTrigger
}

// ScriptRecordError is a record the extended machine cannot hold or the bytes
// do not frame. Trigger is the trigger the refusal is in, -1 for the head or
// the map; Index is the condition or action, -1 for neither. Field names
// what is refused; Got is its value and Max the bound it passes. A record
// cut short is Field "length" with Max the bytes needed and Got the bytes
// left.
type ScriptRecordError struct {
	Trigger, Index int
	Field          string
	Max, Got       int64
}

func (e *ScriptRecordError) Error() string {
	where := "the script record"
	if e.Trigger >= 0 {
		where = fmt.Sprintf("script record trigger %d", e.Trigger)
		if e.Index >= 0 {
			where += fmt.Sprintf(" entry %d", e.Index)
		}
	}
	switch e.Field {
	case "length":
		return fmt.Sprintf("alm: %s needs %d byte(s), %d remain", where, e.Max, e.Got)
	case "trailing":
		return fmt.Sprintf("alm: %s is followed by %d byte(s)", where, e.Got)
	}
	return fmt.Sprintf("alm: %s: %s is %d, the extended machine holds %d", where, e.Field, e.Got, e.Max)
}

func recordRefusal(trigger, index int, field string, max, got int64) error {
	return &ScriptRecordError{Trigger: trigger, Index: index, Field: field, Max: max, Got: got}
}

// DecodeScriptRecord is the one decoder of the record's payload. A value past
// the extended profile's limits or a payload the counts do not tile is a
// *ScriptRecordError; each span is multiplied out in int64 and compared with
// the bytes left before anything is allocated.
func DecodeScriptRecord(payload []byte) (ScriptRecord, error) {
	limits := scriptprofile.Extended
	le := binary.LittleEndian
	total := int64(len(payload))
	if total < ScriptRecordHeadSize {
		return ScriptRecord{}, recordRefusal(-1, -1, "length", ScriptRecordHeadSize, total)
	}
	if v := le.Uint32(payload); v != ScriptRecordVersion {
		return ScriptRecord{}, recordRefusal(-1, -1, "version", ScriptRecordVersion, int64(v))
	}
	vars, n := int64(le.Uint32(payload[4:])), int64(le.Uint32(payload[8:]))
	if vars > int64(limits.Variables) {
		return ScriptRecord{}, recordRefusal(-1, -1, "variables", int64(limits.Variables), vars)
	}
	if n > int64(limits.Triggers) {
		return ScriptRecord{}, recordRefusal(-1, -1, "triggers", int64(limits.Triggers), n)
	}
	at := int64(ScriptRecordHeadSize)
	if need := n * ScriptRecordTriggerHead; need > total-at {
		return ScriptRecord{}, recordRefusal(-1, -1, "length", need, total-at)
	}
	out := ScriptRecord{Variables: int(vars), Triggers: make([]ScriptTrigger, n)}
	for i := range out.Triggers {
		t, size, err := DecodeScriptRecordTrigger(i, payload[at:])
		if err != nil {
			return ScriptRecord{}, err
		}
		out.Triggers[i] = t
		at += int64(size)
	}
	if at != total {
		return ScriptRecord{}, recordRefusal(-1, -1, "trailing", 0, total-at)
	}
	return out, nil
}

// checkRecordTrigger holds one trigger to the extended profile's value rules:
// group numbering, condition and action ids from 1 to the id cap, codes in
// the alphabet, the fire-once word 0 or 1.
func checkRecordTrigger(i int, t ScriptTrigger) error {
	limits := scriptprofile.Extended
	if n := t.Pairs(); n > limits.Pairs || len(t.Group) > n {
		return recordRefusal(i, -1, "conditions", int64(limits.Pairs), int64(max(n, len(t.Group))))
	}
	if len(t.Acts) > limits.Actions {
		return recordRefusal(i, -1, "actions", int64(limits.Actions), int64(len(t.Acts)))
	}
	if t.Once > 1 {
		return recordRefusal(i, -1, "fire-once word", 1, int64(t.Once))
	}
	group := uint32(0)
	for k := range t.Pairs() {
		g := slot(t.Group, k)
		if k == 0 && g != 0 || k > 0 && g != group && g != group+1 {
			return recordRefusal(i, k, "group", int64(group)+min(int64(k), 1), int64(g))
		}
		group = g
		left, right, cmp := t.Pair(k)
		for _, id := range [2]uint32{left, right} {
			if id == 0 || id > limits.MaxID {
				return recordRefusal(i, k, "condition id", int64(limits.MaxID), int64(id))
			}
		}
		if cmp > limits.MaxCompare {
			return recordRefusal(i, k, "comparison code", int64(limits.MaxCompare), int64(cmp))
		}
	}
	for k, id := range t.Acts {
		if id == 0 || id > limits.MaxID {
			return recordRefusal(i, k, "action id", int64(limits.MaxID), int64(id))
		}
	}
	return nil
}

// EncodeScriptRecord is the one encoder of the record's payload. It refuses,
// with a *ScriptRecordError, everything DecodeScriptRecord refuses, so what it
// writes decodes back to r. A name the 64-byte field cannot hold is an error.
func EncodeScriptRecord(r ScriptRecord) ([]byte, error) {
	limits := scriptprofile.Extended
	if r.Variables < 0 || r.Variables > limits.Variables {
		return nil, recordRefusal(-1, -1, "variables", int64(limits.Variables), int64(r.Variables))
	}
	if len(r.Triggers) > limits.Triggers {
		return nil, recordRefusal(-1, -1, "triggers", int64(limits.Triggers), int64(len(r.Triggers)))
	}
	le := binary.LittleEndian
	out := make([]byte, ScriptRecordHeadSize, ScriptRecordHeadSize+ScriptRecordTriggerHead*len(r.Triggers))
	le.PutUint32(out, ScriptRecordVersion)
	le.PutUint32(out[4:], uint32(r.Variables))
	le.PutUint32(out[8:], uint32(len(r.Triggers)))
	for i, t := range r.Triggers {
		b, err := EncodeScriptRecordTrigger(i, t)
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
	return out, nil
}

// DecodeScriptRecordTrigger is the one decoder of a trigger of the record:
// trigger i at the start of b, and the bytes it takes. Its counts are bounded
// by the extended profile and by the bytes left before anything is allocated,
// and its values are held to the record's rules; a refusal is a
// *ScriptRecordError.
func DecodeScriptRecordTrigger(i int, b []byte) (ScriptTrigger, int, error) {
	limits := scriptprofile.Extended
	le := binary.LittleEndian
	total := int64(len(b))
	if total < ScriptRecordTriggerHead {
		return ScriptTrigger{}, 0, recordRefusal(i, -1, "length", ScriptRecordTriggerHead, total)
	}
	head := b[:ScriptRecordTriggerHead]
	once := int64(le.Uint32(head[ScriptRecordOnce:]))
	conds := int64(le.Uint32(head[ScriptRecordCounts:]))
	acts := int64(le.Uint32(head[ScriptRecordCounts+4:]))
	if once > 1 {
		return ScriptTrigger{}, 0, recordRefusal(i, -1, "fire-once word", 1, once)
	}
	if conds > int64(limits.Pairs) {
		return ScriptTrigger{}, 0, recordRefusal(i, -1, "conditions", int64(limits.Pairs), conds)
	}
	if acts > int64(limits.Actions) {
		return ScriptTrigger{}, 0, recordRefusal(i, -1, "actions", int64(limits.Actions), acts)
	}
	at := int64(ScriptRecordTriggerHead)
	if need := conds*ScriptRecordConditionSize + acts*ScriptRecordActionSize; need > total-at {
		return ScriptTrigger{}, 0, recordRefusal(i, -1, "length", need, total-at)
	}
	words := make([]uint32, 4*conds+acts)
	t := ScriptTrigger{
		Name:  ScriptLabelText.Decode(head[:ScriptLabelLen]),
		Group: words[0*conds : 1*conds : 1*conds],
		Left:  words[1*conds : 2*conds : 2*conds],
		Right: words[2*conds : 3*conds : 3*conds],
		Cmp:   words[3*conds : 4*conds : 4*conds],
		Acts:  words[4*conds:],
		Once:  uint32(once),
	}
	for k := range conds {
		rec := b[at+k*ScriptRecordConditionSize:]
		t.Group[k], t.Left[k], t.Right[k], t.Cmp[k] = le.Uint32(rec), le.Uint32(rec[4:]), le.Uint32(rec[8:]), le.Uint32(rec[12:])
	}
	at += conds * ScriptRecordConditionSize
	for k := range acts {
		t.Acts[k] = le.Uint32(b[at+k*ScriptRecordActionSize:])
	}
	at += acts * ScriptRecordActionSize
	if err := checkRecordTrigger(i, t); err != nil {
		return ScriptTrigger{}, 0, err
	}
	return t, int(at), nil
}

// EncodeScriptRecordTrigger is the one encoder of a trigger of the record:
// trigger i's bytes, refused with a *ScriptRecordError on everything
// DecodeScriptRecordTrigger refuses. A name the 64-byte field cannot hold is
// an error.
func EncodeScriptRecordTrigger(i int, t ScriptTrigger) ([]byte, error) {
	if err := checkRecordTrigger(i, t); err != nil {
		return nil, err
	}
	name, err := ScriptLabelText.Encode(t.Name)
	if err != nil {
		return nil, fmt.Errorf("alm: script record trigger %d: %w", i, err)
	}
	le := binary.LittleEndian
	out := make([]byte, ScriptRecordTriggerHead, ScriptRecordTriggerHead+ScriptRecordConditionSize*t.Pairs()+ScriptRecordActionSize*len(t.Acts))
	copy(out, name)
	le.PutUint32(out[ScriptRecordOnce:], t.Once)
	le.PutUint32(out[ScriptRecordCounts:], uint32(t.Pairs()))
	le.PutUint32(out[ScriptRecordCounts+4:], uint32(len(t.Acts)))
	for k := range t.Pairs() {
		left, right, cmp := t.Pair(k)
		out = le.AppendUint32(out, slot(t.Group, k))
		out = le.AppendUint32(out, left)
		out = le.AppendUint32(out, right)
		out = le.AppendUint32(out, cmp)
	}
	for _, id := range t.Acts {
		out = le.AppendUint32(out, id)
	}
	return out, nil
}

// AppendScriptRecord returns stream with r encoded as one more record after
// its last one, the file header's record count raised by one. stream must be
// a map Open accepts and carry no script record.
func AppendScriptRecord(stream []byte, r ScriptRecord) ([]byte, error) {
	m, err := Open(stream)
	if err != nil {
		return nil, err
	}
	if m.HasScriptRecord() {
		return nil, recordRefusal(-1, -1, "records", 1, 2)
	}
	framed, err := FrameScriptRecord(r)
	if err != nil {
		return nil, err
	}
	return appendRecord(stream, m, framed), nil
}

// appendRecord is the one writer that adds a framed Againrom record to
// stream, opened as m: after its last record, before any trailer, with the
// file header's record count raised by one.
func appendRecord(stream []byte, m *Map, framed []byte) []byte {
	end := FileHeaderSize
	for _, rec := range m.Records {
		end += RecordHeaderSize + int(rec.PayloadSize)
	}
	out := make([]byte, 0, len(stream)+len(framed))
	out = append(out, stream[:end]...)
	binary.LittleEndian.PutUint32(out[FileRecordCount:], m.RecordCount+1)
	out = append(out, framed...)
	return append(out, stream[end:]...)
}

// FrameScriptRecord is r as one record of the stream: its 20-byte record
// header, then the payload EncodeScriptRecord writes. A writer that adds the
// record after a stream's last one raises the file header's record count.
func FrameScriptRecord(r ScriptRecord) ([]byte, error) {
	payload, err := EncodeScriptRecord(r)
	if err != nil {
		return nil, err
	}
	return frameRecord(ScriptRecordTypeID, payload), nil
}

// frameRecord is the one framer of the Againrom record family: the 20-byte
// record header of typeID, then payload.
func frameRecord(typeID uint32, payload []byte) []byte {
	le := binary.LittleEndian
	out := make([]byte, RecordHeaderSize, RecordHeaderSize+len(payload))
	le.PutUint32(out[RecordTag:], recordTag)
	le.PutUint32(out[RecordHeaderLength:], recordHdrLen)
	le.PutUint32(out[RecordPayloadSize:], uint32(len(payload)))
	le.PutUint32(out[RecordTypeID:], typeID)
	return append(out, payload...)
}

// HasScriptRecord reports whether the map carries the Againrom script record.
func (m *Map) HasScriptRecord() bool { return m.scriptRecords > 0 }
