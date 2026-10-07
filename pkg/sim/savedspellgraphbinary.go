package sim

import (
	"bytes"
	"encoding/binary"
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
	base := b[0]
	b[0] = spellGraphFormVersion
	if len(areas) != 0 {
		b[0] = currentAreaFormVersion
	}
	raw, _ := json.Marshal(spellContinuation{Graph: w.savedSpellGraph, Order: w.effectOrder, Areas: areas})
	b = append(b, raw...)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(raw)))
	return append(b, base, 'S', 'P', 'G', '1')
}

func (w *World) unmarshalSavedSpellGraph(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed saved spell graph section") }
	if len(data) < headerLen+10 || !bytes.Equal(data[len(data)-4:], []byte("SPG1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	if baseVersion != formatVersion && baseVersion != autoHealingFormVersion {
		return fail()
	}
	size := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if size == 0 || size > maxCarriedResumeBytes || size > uint64(len(data)-9-headerLen) {
		return fail()
	}
	start := len(data) - 9 - int(size)
	var continuation spellContinuation
	d := json.NewDecoder(bytes.NewReader(data[start : len(data)-9]))
	d.DisallowUnknownFields()
	if d.Decode(&continuation) != nil || d.Decode(new(any)) != io.EOF {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
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
	if !bytes.Equal(canonical, data[start:len(data)-9]) {
		return fail()
	}
	*w = next
	return nil
}
