package sim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const spellGraphFormVersion byte = 97
const currentAreaFormVersion byte = 98

type currentAreaRecord struct {
	Index uint32
	Value CurrentAreaPayload
}

type spellContinuation struct {
	Graph *SavedSpellGraph
	Order []WorldEffectRef
	Areas []currentAreaRecord `json:",omitempty"`
}

func (w *World) appendSavedSpellGraph(b []byte) []byte {
	var areas []currentAreaRecord
	for i, e := range w.effects {
		if e.Current != nil {
			areas = append(areas, currentAreaRecord{uint32(i), *e.Current})
		}
	}
	if w.savedSpellGraph == nil && len(w.effectOrder) == 0 && len(areas) == 0 {
		return b
	}
	ws, version := beginWrapper(b), spellGraphFormVersion
	if len(areas) != 0 {
		version = currentAreaFormVersion
	}
	raw, _ := json.Marshal(spellContinuation{Graph: w.savedSpellGraph, Order: w.effectOrder, Areas: areas})
	return endWrapper(append(b, raw...), ws, version)
}

func (w *World) unmarshalSavedSpellGraph(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed saved spell graph section") }
	env, ok := readWrapper(data, spellGraphFormVersion)
	size := uint64(len(env.section()))
	if !ok || size == 0 || size > maxCarriedResumeBytes {
		return fail()
	}
	var continuation spellContinuation
	d := json.NewDecoder(bytes.NewReader(env.section()))
	d.DisallowUnknownFields()
	if d.Decode(&continuation) != nil || d.Decode(new(any)) != io.EOF {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	next.savedSpellGraph, next.effectOrder = continuation.Graph, continuation.Order
	if (data[0] == currentAreaFormVersion) != (len(continuation.Areas) > 0) {
		return fail()
	}
	for i, row := range continuation.Areas {
		if int(row.Index) >= len(next.effects) || i > 0 && continuation.Areas[i-1].Index >= row.Index || savedEffectClassFault(row.Value.Payload) != nil {
			return fail()
		}
		v := row.Value
		next.effects[row.Index].Current = &v
	}
	if err := next.worldEffectOrderFault(); err != nil {
		return err
	}
	if err := next.savedSpellGraphFault(true); err != nil {
		return err
	}
	if err := next.savedWorldEffectsFault(); err != nil {
		return err
	}
	canonical, _ := json.Marshal(continuation)
	if !bytes.Equal(canonical, env.section()) {
		return fail()
	}
	*w = next
	return nil
}
