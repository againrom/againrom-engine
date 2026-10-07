package sim

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func currentEffectWidthWorld(t *testing.T, wire uint16) *World {
	t.Helper()
	w := originalAttachmentWorld(t)
	if err := w.ImportOriginalAttachedEffects([]ActiveEffect{{Target: 1, Spell: 20, Kind: EffectAbsorption, Mode: EffectDuration, Magnitude: int32(int16(wire)), Remaining: 64}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func currentEffectWidthValues(w *World, rows ...EffectNumericResidue) map[EntityID]ActorValues {
	values := map[EntityID]ActorValues{}
	for _, e := range w.Entities() {
		v := e.Values()
		if e.ID == 1 {
			v.EffectWidths = rows
		}
		values[e.ID] = v
	}
	return values
}

func TestCurrentEffectWidthContinuationBoundsAndOrdinaryPrecedence(t *testing.T) {
	for _, magnitude := range []int32{40000, -40000, -1 << 31, 1<<31 - 1} {
		t.Run(fmt.Sprint(magnitude), func(t *testing.T) {
			wire := uint16(magnitude)
			row := EffectNumericResidue{Spell: 20, Wire: wire, Lift: int64(magnitude) - int64(int16(wire))}
			for _, edit := range []bool{false, true} {
				input := wire
				if edit {
					input = 7
				}
				w := currentEffectWidthWorld(t, input)
				values := currentEffectWidthValues(w, row)
				encoded, err := json.Marshal(values)
				if err != nil || strings.Contains(string(encoded), "EffectWidths") {
					t.Fatal("transient widths duplicated the serialized leaf", err)
				}
				if err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil); err != nil {
					t.Fatal(err)
				}
				want := magnitude
				if edit {
					want = 7
				}
				if got := w.ActiveEffects()[0].Magnitude; got != want {
					t.Fatal("wrong current magnitude", got, want)
				}
			}
		})
	}
}

func TestCurrentEffectWidthContinuationRejectsMalformedAndLateFailureAtomically(t *testing.T) {
	good := EffectNumericResidue{Spell: 20, Wire: 40000, Lift: 65536}
	for _, name := range []string{"zero", "non-width", "overflow", "huge", "absent spell", "duplicate", "late action"} {
		t.Run(name, func(t *testing.T) {
			w := currentEffectWidthWorld(t, good.Wire)
			row, actions := good, w.Actions()
			switch name {
			case "zero":
				row.Lift = 0
			case "non-width":
				row.Lift++
			case "overflow":
				row.Wire, row.Lift = 1, 1<<31
			case "huge":
				row.Lift = -1 << 63
			case "absent spell":
				row.Spell = 21
			case "late action":
				actions.Actors = nil
			}
			rows := []EffectNumericResidue{row}
			if name == "duplicate" {
				rows = append(rows, row)
			}
			before := w.Hash()
			if err := w.RestoreCurrentContinuation(nil, currentEffectWidthValues(w, rows...), actions, nil); err == nil || w.Hash() != before {
				t.Fatal("invalid continuation changed the World", err)
			}
		})
	}
}
