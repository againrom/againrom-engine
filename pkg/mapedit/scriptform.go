package mapedit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/scriptprofile"
)

// The Againrom script record in the model (DIV-4599). On a map carrying it,
// KindTrigger is the record's trigger array, so counting, reading, placing,
// deleting, reordering and the field table reach those triggers through the
// one section machinery. ExtendScript and FoldScript move a map's triggers
// between the type-7 record and this one as one edit each.

// recordTriggers is KindTrigger's row on a map carrying the record: the
// record's head counts the triggers, and the record's one trigger decoder
// measures each and holds it to the record's rules.
var recordTriggers = &section{name: "trigger", typeID: alm.ScriptRecordTypeID,
	layout:   recordTriggerLayout,
	length:   recordTriggerLength,
	cellless: func([]byte) bool { return true },
	fields:   func(_ *Editor, rec []byte) []Field { return recordTriggerFields(rec) },
}

func recordTriggerLayout(e *Editor, sp span) (arrayLayout, error) {
	if sp.payloadSize < alm.ScriptRecordHeadSize {
		return arrayLayout{}, fmt.Errorf("the script record is %d bytes, under its %d-byte head", sp.payloadSize, alm.ScriptRecordHeadSize)
	}
	countAt := sp.abs(alm.ScriptRecordTriggerCount)
	n := int64(e.u32(countAt))
	if limit := int64(scriptprofile.Extended.Triggers); n > limit {
		return arrayLayout{}, fmt.Errorf("the script record counts %d triggers, past %d", n, limit)
	}
	return arrayLayout{countAt: countAt, first: alm.ScriptRecordHeadSize, n: int(n)}, nil
}

func recordTriggerLength(_ *Editor, b []byte) int {
	_, n, err := alm.DecodeScriptRecordTrigger(0, b)
	if err != nil {
		return -1
	}
	return n
}

// recordTriggerFields is a record trigger's field table: the name, each
// condition's group, ids and code, each action id and the fire-once word.
// The slots are the ones the trigger holds; SetTriggerSlots changes their
// number.
func recordTriggerFields(rec []byte) []Field {
	conds := int(binary.LittleEndian.Uint32(rec[alm.ScriptRecordCounts:]))
	acts := int(binary.LittleEndian.Uint32(rec[alm.ScriptRecordCounts+4:]))
	maxID := int64(scriptprofile.Extended.MaxID)
	out := []Field{{Key: "name", Label: "Name", Offset: 0, Width: alm.ScriptLabelLen, Text: &alm.ScriptLabelText}}
	for i := range conds {
		at := alm.ScriptRecordTriggerHead + alm.ScriptRecordConditionSize*i
		out = append(out, Field{Key: scriptprofile.GroupKey(i), Label: fmt.Sprintf("Condition%d group", i+1), Offset: at, Width: 4, Max: int64(scriptprofile.ExtendedConditions - 1)},
			Field{Key: scriptprofile.LeftKey(i), Label: fmt.Sprintf("Condition%d left ID", i+1), Offset: at + 4, Width: 4, Min: 1, Max: maxID},
			Field{Key: scriptprofile.RightKey(i), Label: fmt.Sprintf("Condition%d right ID", i+1), Offset: at + 8, Width: 4, Min: 1, Max: maxID},
			Field{Key: scriptprofile.CmpKey(i), Label: fmt.Sprintf("Condition%d comparison", i+1), Offset: at + 12, Width: 4, Max: int64(scriptprofile.MaxCompare)})
	}
	base := alm.ScriptRecordTriggerHead + alm.ScriptRecordConditionSize*conds
	for i := range acts {
		out = append(out, Field{Key: scriptprofile.ActionKey(i), Label: fmt.Sprintf("Action%d ID", i+1), Offset: base + alm.ScriptRecordActionSize*i, Width: 4, Min: 1, Max: maxID})
	}
	return append(out, Field{Key: "once", Label: "Fire once", Offset: alm.ScriptRecordOnce, Width: 4, Max: 1})
}

// Extended reports that the map carries the Againrom script record.
func (e *Editor) Extended() bool { return e.locate().carries(alm.ScriptRecordTypeID) }

// scriptRoom refuses a write that adds checks conditions and triggers
// triggers past the map's profile, with a *ScriptLimitError.
func (e *Editor) scriptRoom(checks, triggers int) error {
	p := scriptprofile.Of(e.Extended())
	if n := e.Count(KindCondition); checks > 0 && p.Registers > 0 && n+checks > p.Registers {
		return &ScriptLimitError{Limit: LimitChecks, Max: int64(p.Registers), Got: int64(n)}
	}
	if n := e.Count(KindTrigger); n+triggers > p.Triggers {
		return &ScriptLimitError{Limit: LimitTriggers, Max: int64(p.Triggers), Got: int64(n)}
	}
	return nil
}

// NewTrigger is t as one trigger of the map's form: the 184-byte record
// NewScriptTrigger builds, or on an extended map the bytes of the next
// trigger of the Againrom record.
func (e *Editor) NewTrigger(t alm.ScriptTrigger) ([]byte, error) {
	return e.newTrigger(t, nil)
}

// newTrigger is NewTrigger where on an ordinary map a slot word the same
// slot of trigger record like holds stands unchecked; like nil checks every
// word.
func (e *Editor) newTrigger(t alm.ScriptTrigger, like []byte) ([]byte, error) {
	if e.Extended() {
		return alm.EncodeScriptRecordTrigger(e.Count(KindTrigger), t)
	}
	if like == nil {
		return NewScriptTrigger(t)
	}
	return newScriptTrigger(t, heldIn(like))
}

// setRecordTrigger writes t over record trigger i, whose bytes are rec,
// keeping its name.
func (e *Editor) setRecordTrigger(i int, rec []byte, t alm.ScriptTrigger) error {
	old, _, err := alm.DecodeScriptRecordTrigger(i, rec)
	if err != nil {
		return err
	}
	t.Name = old.Name
	next, err := alm.EncodeScriptRecordTrigger(i, t)
	if err != nil || bytes.Equal(next, rec) {
		return err
	}
	w, err := e.walkIndex(KindTrigger, "rewrite", i)
	if err != nil {
		return err
	}
	at, l := w.at(i)
	ed := edit{replace(at, e.data[at:at+l], next)}
	if len(next) != l {
		ed = append(ed, e.sizeWordPart(w.span, len(next)-l))
	}
	e.apply(ed)
	return nil
}

// ScriptVariables is an extended map's variable count; 0 on an original map.
func (e *Editor) ScriptVariables() int {
	sp, ok := e.locate().at(alm.ScriptRecordTypeID)
	if !ok || sp.payloadSize < alm.ScriptRecordHeadSize {
		return 0
	}
	return int(e.u32(sp.abs(alm.ScriptRecordVariables)))
}

// SetScriptVariables sets an extended map's variable count to n as one edit.
func (e *Editor) SetScriptVariables(n int) error {
	sp, err := e.locate().optional("set the variable count", alm.ScriptRecordTypeID)
	if err != nil {
		return err
	}
	if limit := scriptprofile.Extended.Variables; n < 0 || n > limit {
		return &ScriptLimitError{Limit: LimitVariables, Max: int64(limit), Got: int64(n - 1)}
	}
	at := sp.abs(alm.ScriptRecordVariables)
	if old := int(e.u32(at)); old != n {
		e.apply(edit{e.wordPart(at, n-old)})
	}
	return nil
}

// cmpNotEqual is the comparison code "differs" (TRIG-CMP-006): a condition
// compared with itself under it never holds.
const cmpNotEqual = 1

// scriptForms is what moving triggers between the forms reads of the
// nodes: the action ids that name drop cells only, which run nothing on
// either machine (TRIG-DROP-013), and a condition id to stand in a pair
// that only one of the forms needs.
type scriptForms struct {
	dropOnly map[uint32]bool
	anyCheck uint32
}

func scriptFormsOf(sc alm.Script) scriptForms {
	f := scriptForms{dropOnly: map[uint32]bool{}, anyCheck: 1}
	other := map[uint32]bool{}
	for _, n := range sc.Actions {
		if n.Opcode == dropOpcode {
			f.dropOnly[n.ID] = true
		} else {
			other[n.ID] = true
		}
	}
	for id := range other {
		delete(f.dropOnly, id)
	}
	for _, n := range sc.Conditions {
		if n.ID != 0 && n.ID <= scriptprofile.MaxID && (f.anyCheck == 1 || n.ID < f.anyCheck) {
			f.anyCheck = n.ID
		}
	}
	return f
}

// runs reports that acts names an action that runs.
func (f scriptForms) runs(acts []uint32) bool {
	return slices.ContainsFunc(acts, func(id uint32) bool { return id != 0 && !f.dropOnly[id] })
}

// extended is type-7 trigger t as the record holds it, firing as the
// original fires it: a pair with an empty side and an empty action slot
// go; a trigger the original never fires (first left id 0, or a used pair
// whose code is past the alphabet) keeps one condition that never holds,
// unless no action of it runs.
func (f scriptForms) extended(t alm.ScriptTrigger) (alm.ScriptTrigger, error) {
	x := alm.ScriptTrigger{Name: t.Name}
	if t.Once != 0 {
		x.Once = 1
	}
	maxID := scriptprofile.Extended.MaxID
	for _, id := range t.Acts {
		if id > maxID {
			return x, &ScriptLimitError{Limit: LimitID, Max: int64(maxID), Got: int64(id)}
		}
		if id != 0 {
			x.Acts = append(x.Acts, id)
		}
	}
	first, _, _ := t.Pair(0)
	fires := first != 0
	for k := range t.Pairs() {
		left, right, cmp := t.Pair(k)
		switch {
		case left == 0 || right == 0:
			continue
		case cmp > scriptprofile.MaxCompare:
			fires = false
			continue
		case left > maxID || right > maxID:
			return x, &ScriptLimitError{Limit: LimitID, Max: int64(maxID), Got: int64(max(left, right))}
		}
		x.Left, x.Right, x.Cmp = append(x.Left, left), append(x.Right, right), append(x.Cmp, cmp)
	}
	if !fires {
		x.Left, x.Right, x.Cmp = nil, nil, nil
		if f.runs(x.Acts) {
			x.Left, x.Right, x.Cmp = []uint32{f.anyCheck}, []uint32{f.anyCheck}, []uint32{cmpNotEqual}
		}
	}
	return x, nil
}

// ExtendScript moves the map's type-7 triggers into the Againrom script
// record as one edit, in map order, each as extended reads it, with the
// hundred registers as the record's first variables; the type-7 record
// keeps the nodes. The record goes after the stream's last record, as
// alm.AppendScriptRecord puts it.
func (e *Editor) ExtendScript() error {
	if e.Extended() {
		return fmt.Errorf("mapedit: the map's script is already extended")
	}
	w, err := e.walkSection(KindTrigger, "extend the script")
	if err != nil {
		return err
	}
	m, err := e.Map()
	if err != nil {
		return err
	}
	sc, err := m.Script()
	if err != nil {
		return err
	}
	forms := scriptFormsOf(sc)
	r := alm.ScriptRecord{Variables: scriptprofile.FirstRegisters}
	for i, t := range sc.Triggers {
		x, err := forms.extended(t)
		if err != nil {
			return fmt.Errorf("mapedit: cannot extend trigger %d: %w", i, err)
		}
		r.Triggers = append(r.Triggers, x)
	}
	framed, err := alm.FrameScriptRecord(r)
	if err != nil {
		return err
	}
	last := w.frame.physical[len(w.frame.physical)-1]
	ed := edit{replace(last.payloadOff+last.payloadSize, nil, framed), e.wordPart(alm.FileRecordCount, 1)}
	if start, end := w.span.abs(w.starts[0]), w.span.abs(w.starts[w.n()]); end > start {
		ed = append(ed, replace(start, e.data[start:end], nil), e.wordPart(w.countAt, -w.n()), e.sizeWordPart(w.span, start-end))
	}
	e.apply(ed)
	return nil
}

// FoldScript moves an extended map's triggers back into the type-7 record
// as one edit, each written by NewScriptTrigger, and removes the Againrom
// record, so the original opens the map. A script the original record
// cannot hold is a *ScriptLimitError and nothing is written.
func (e *Editor) FoldScript() error {
	f := e.locate()
	sp, err := f.optional("fold the script", alm.ScriptRecordTypeID)
	if err != nil {
		return err
	}
	r, err := alm.DecodeScriptRecord(e.data[sp.payloadOff : sp.payloadOff+sp.payloadSize])
	if err != nil {
		return err
	}
	first := scriptprofile.First
	if r.Variables > first.Registers {
		return &ScriptLimitError{Limit: LimitVariables, Max: int64(first.Registers), Got: int64(r.Variables - 1)}
	}
	if len(r.Triggers) > first.Triggers {
		return &ScriptLimitError{Limit: LimitTriggers, Max: int64(first.Triggers), Got: int64(len(r.Triggers))}
	}
	if n := e.Count(KindCondition); n > first.Registers {
		return &ScriptLimitError{Limit: LimitChecks, Max: int64(first.Registers), Got: int64(n)}
	}
	w, err := e.walkRow(f, sections[KindTrigger], "fold the script")
	if err != nil {
		return err
	}
	var recs []byte
	for _, t := range r.Triggers {
		b, err := NewScriptTrigger(t)
		if err != nil {
			return err
		}
		recs = append(recs, b...)
	}
	at, frameEnd := w.span.abs(w.starts[w.n()]), sp.payloadOff+sp.payloadSize
	ed := edit{e.wordPart(w.countAt, len(r.Triggers)), e.sizeWordPart(w.span, len(recs)), e.wordPart(alm.FileRecordCount, -1)}
	if at == sp.headerOff {
		ed = append(ed, replace(at, e.data[at:frameEnd], recs))
	} else {
		ed = append(ed, replace(at, nil, recs), replace(sp.headerOff, e.data[sp.headerOff:frameEnd], nil))
	}
	e.apply(ed)
	return nil
}
