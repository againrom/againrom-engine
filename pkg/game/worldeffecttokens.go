package game

import (
	"encoding/json"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentWorldEffectToken struct {
	Object uint16
	Value  *sim.SavedObjectToken `json:",omitempty"`
	Wire   *sim.SavedObjectToken `json:",omitempty"`
}

func captureCurrentWorldEffectTokens(state *SnapshotSAVDocument, w *sim.World, a *currentActionData) error {
	d := w.SavedWorldEffectDrivers()
	if d == nil || len(d.Tokens) == 0 {
		return nil
	}
	if state.WorldEffects == nil || len(state.WorldEffects.TokenObjects) != len(d.Tokens) {
		return fmt.Errorf("world effect Token population differs")
	}
	for i, token := range d.Tokens {
		v := token
		a.WorldEffectTokens = append(a.WorldEffectTokens, currentWorldEffectToken{Object: state.WorldEffects.TokenObjects[i], Value: &v})
	}
	return nil
}

func finalizeCurrentWorldEffectTokens(doc *sav.DocumentData) error {
	a, err := readCurrentActions(doc)
	if err != nil || a == nil || len(a.WorldEffectTokens) == 0 {
		return err
	}
	for i := range a.WorldEffectTokens {
		r := &a.WorldEffectTokens[i]
		if r.Object == 0 {
			continue
		}
		if int(r.Object) > len(doc.Objects) || r.Value == nil {
			return fmt.Errorf("world effect Token lacks its current record")
		}
		wire, err := savedItemToken(&doc.Objects[r.Object-1])
		if err != nil {
			return err
		}
		if wire == *r.Value {
			r.Value, r.Wire = nil, nil
		} else {
			r.Wire = &wire
		}
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, raw)
}

func restoreCurrentWorldEffectTokens(ms *Mission, a *currentActionData) error {
	if len(a.WorldEffectTokens) == 0 {
		if d := ms.World.SavedWorldEffectDrivers(); d != nil {
			d.Tokens = nil
			if err := ms.World.ImportOriginalWorldEffectDrivers(d); err != nil {
				return err
			}
		}
		if ms.savedDocument.WorldEffects != nil {
			ms.savedDocument.WorldEffects.TokenObjects = nil
		}
		return nil
	}
	if len(a.WorldEffectTokens) > 65534 {
		return fmt.Errorf("invalid world effect Token population")
	}
	if ms.savedDocument.WorldEffects == nil {
		ms.savedDocument.WorldEffects = &SnapshotSAVWorldEffects{Version: 1}
	}
	d := ms.World.SavedWorldEffectDrivers()
	if d == nil {
		d = &sim.SavedWorldEffects{}
	}
	d.Tokens = nil
	var objects []uint16
	seen := map[uint16]bool{}
	family, err := worldEffectRecords(ms.savedDocument.Document)
	if err != nil {
		return err
	}
	allowed := map[uint16]bool{}
	for _, object := range family {
		allowed[object] = true
	}
	for _, row := range a.WorldEffectTokens {
		var token sim.SavedObjectToken
		if row.Object == 0 {
			if row.Value == nil || row.Wire != nil {
				return fmt.Errorf("absent world effect Token has no value")
			}
			token = *row.Value
		} else {
			if int(row.Object) > len(ms.savedDocument.Document.Objects) || !allowed[row.Object] || seen[row.Object] || (row.Value == nil) != (row.Wire == nil) {
				return fmt.Errorf("invalid world effect Token binding")
			}
			var err error
			token, err = savedItemToken(&ms.savedDocument.Document.Objects[row.Object-1])
			if err != nil {
				return err
			}
			if row.Wire != nil && token == *row.Wire {
				token = *row.Value
			}
			seen[row.Object] = true
		}
		d.Tokens = append(d.Tokens, token)
		objects = append(objects, row.Object)
	}
	if err := ms.World.ImportOriginalWorldEffectDrivers(d); err != nil {
		return err
	}
	ms.savedDocument.WorldEffects.TokenObjects = objects
	return nil
}
