package sim

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/scriptprofile"
)

// A mod's global variables are Againrom's own script state (DIV-5108): the
// original has none. The World holds the values of the globals its mod
// declares, seated at the mission's entry; a script reads, sets and adds
// them through checks and actions of their own. An id the World does not
// hold reads 0 and takes no write.

// ScriptGlobal is one global variable a World holds: its id and its value.
type ScriptGlobal struct {
	ID, Value int32
}

// scriptGlobalFormVersion carries the globals a World holds.
const scriptGlobalFormVersion byte = 144

const scriptGlobalRecordLen = 8

// validScriptGlobals refuses a list whose ids are outside the global range
// or not strictly ascending.
func validScriptGlobals(list []ScriptGlobal) error {
	prior := int32(scriptprofile.FirstGlobal - 1)
	for _, g := range list {
		if g.ID < scriptprofile.FirstGlobal || g.ID > scriptprofile.LastGlobal {
			return fmt.Errorf("sim: global %d is outside %d..%d", g.ID, scriptprofile.FirstGlobal, scriptprofile.LastGlobal)
		}
		if g.ID <= prior {
			return fmt.Errorf("sim: global %d stands twice or out of order", g.ID)
		}
		prior = g.ID
	}
	return nil
}

// SeatScriptGlobals makes list the globals w holds, sorted by id; nil holds
// none. An id outside the global range or named twice is refused and leaves
// w unchanged.
func (w *World) SeatScriptGlobals(list []ScriptGlobal) error {
	sorted := slices.Clone(list)
	slices.SortStableFunc(sorted, func(a, b ScriptGlobal) int { return int(a.ID) - int(b.ID) })
	if err := validScriptGlobals(sorted); err != nil {
		return err
	}
	if len(sorted) == 0 {
		sorted = nil
	}
	w.globals = sorted
	return nil
}

// ScriptGlobals is a copy of the globals w holds, by id.
func (w *World) ScriptGlobals() []ScriptGlobal { return slices.Clone(w.globals) }

// globalIndex is the index of global id in w's list, or -1.
func (w *World) globalIndex(id int32) int {
	i, ok := slices.BinarySearchFunc(w.globals, id, func(g ScriptGlobal, id int32) int { return int(g.ID) - int(id) })
	if !ok {
		return -1
	}
	return i
}

// globalAt is the value of global id; 0 for an id w does not hold.
func (w *World) globalAt(id int32) int32 {
	if i := w.globalIndex(id); i >= 0 {
		return w.globals[i].Value
	}
	return 0
}

// setGlobal writes v into global id; an id w does not hold takes no write.
// The list is copied first, so a World copy that shares it keeps its values.
func (w *World) setGlobal(id, v int32) {
	i := w.globalIndex(id)
	if i < 0 {
		return
	}
	globals := slices.Clone(w.globals)
	globals[i].Value = v
	w.globals = globals
}

// appendScriptGlobals is the outermost wrapper: the globals w holds, written
// only when it holds one, so a World without globals keeps its bytes.
func (w *World) appendScriptGlobals(b []byte) []byte {
	if len(w.globals) == 0 {
		return b
	}
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for _, g := range w.globals {
		b = le.AppendUint32(b, uint32(g.ID))
		b = le.AppendUint32(b, uint32(g.Value))
	}
	return endWrapper(b, ws, scriptGlobalFormVersion)
}

func (w *World) unmarshalScriptGlobals(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed script-global section") }
	env, ok := readWrapper(data, scriptGlobalFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%scriptGlobalRecordLen != 0 || len(s)/scriptGlobalRecordLen > scriptprofile.LastGlobal {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	le := binary.LittleEndian
	globals := make([]ScriptGlobal, 0, len(s)/scriptGlobalRecordLen)
	for at := 0; at < len(s); at += scriptGlobalRecordLen {
		globals = append(globals, ScriptGlobal{ID: int32(le.Uint32(s[at:])), Value: int32(le.Uint32(s[at+4:]))})
	}
	if validScriptGlobals(globals) != nil {
		return fail()
	}
	next.globals = globals
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
