package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentNativeBasisWire struct {
	Entity   sim.EntityID
	Base     []byte  `json:",omitempty"`
	Modifier []byte  `json:",omitempty"`
	Body     *uint16 `json:",omitempty"`
	Attack   []byte  `json:",omitempty"`
	Defence  []byte  `json:",omitempty"`
	Unbound  bool    `json:",omitempty"`
}

func (a currentActionData) MarshalJSON() ([]byte, error) {
	type transport currentActionData
	out := transport(a)
	unbound := make(map[sim.EntityID]bool)
	for _, wire := range a.NativeBasisWires {
		unbound[wire.Entity] = wire.Unbound
	}
	actors := func(rows []sim.ActorContinuation) ([]sim.ActorContinuation, error) {
		next := slices.Clone(rows)
		for i := range next {
			row := &next[i]
			if row.Current == nil || row.Current.NativeBasis == nil {
				continue
			}
			current := *row.Current
			basis, err := nativeScalarMetadata(*current.NativeBasis, unbound[row.Entity])
			if err != nil {
				return nil, err
			}
			current.NativeBasis, row.Current = &basis, &current
		}
		return next, nil
	}
	var err error
	out.Actions.Actors, err = actors(a.Actions.Actors)
	if err != nil {
		return nil, err
	}
	out.Held, err = actors(a.Held)
	if err != nil {
		return nil, err
	}
	out.RemovedNativeBases = slices.Clone(a.RemovedNativeBases)
	for i := range out.RemovedNativeBases {
		row := &out.RemovedNativeBases[i]
		row.Basis, err = nativeScalarMetadata(row.Basis, unbound[row.ID])
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(out)
}

func nativeBasisRecord(doc *sav.DocumentData, a *currentActionData, id sim.EntityID, removed bool) (*sav.DocumentRecordData, error) {
	var object uint16
	bindings := 0
	for _, b := range a.Bindings {
		if b.ID == id && !b.Structure {
			bindings++
			if bindings > 1 || b.Missing != (b.Object == 0) {
				return nil, fmt.Errorf("native basis has duplicate ordinary bindings")
			}
			object = b.Object
		}
	}
	if removed && bindings == 1 && object == 0 && pureCurrentTerminalActorValue(a.Values[id], id) {
		return nil, nil
	}
	if object == 0 || int(object) > len(doc.Objects) {
		return nil, fmt.Errorf("native basis has no ordinary actor")
	}
	r := &doc.Objects[object-1]
	if r.Class != "Human" && r.Class != "Humanoid" && r.Class != "Unit" {
		return nil, fmt.Errorf("native basis has an incompatible ordinary class")
	}
	if removed {
		identity, err := savedStructureValue(r, "Identity")
		if err != nil || identity == 0 || !slices.Contains(doc.DeadActors, object) {
			return nil, fmt.Errorf("removed native basis lacks an exact ordinary dead root")
		}
	}
	return r, nil
}

type currentNativeBasisValue struct {
	Entity  sim.EntityID
	Basis   *sim.NativeActorBasis
	Removed bool
}

func currentNativeBasisActors(a *currentActionData, includeHeld bool) ([]currentNativeBasisValue, error) {
	var rows []currentNativeBasisValue
	seen := map[sim.EntityID]bool{}
	populations := [][]sim.ActorContinuation{a.Actions.Actors}
	if includeHeld {
		populations = append(populations, a.Held)
	}
	for _, actors := range populations {
		for i := range actors {
			row := &actors[i]
			if row.Current == nil || row.Current.NativeBasis == nil {
				continue
			}
			if seen[row.Entity] {
				return nil, fmt.Errorf("native basis repeats an actor across current or held populations")
			}
			seen[row.Entity] = true
			rows = append(rows, currentNativeBasisValue{Entity: row.Entity, Basis: row.Current.NativeBasis})
		}
	}
	for i := range a.RemovedNativeBases {
		row := &a.RemovedNativeBases[i]
		if seen[row.ID] || i > 0 && row.ID <= a.RemovedNativeBases[i-1].ID || !row.Basis.HasValues() {
			return nil, fmt.Errorf("removed native basis repeats an actor or is empty")
		}
		seen[row.ID] = true
		rows = append(rows, currentNativeBasisValue{Entity: row.ID, Basis: &row.Basis, Removed: true})
	}
	return rows, nil
}

func captureCurrentNativeBasis(doc *sav.DocumentData, a *currentActionData) error {
	a.NativeHistoryVersion = 1
	for _, removed := range a.RemovedNativeBases {
		for i := range a.Held {
			row := &a.Held[i]
			if row.Entity == removed.ID && row.Current != nil {
				current := *row.Current
				current.NativeBasis = nil
				row.Current = &current
			}
		}
	}
	rows, err := currentNativeBasisActors(a, true)
	if err != nil {
		return err
	}
	a.NativeBasisWires = nil
	a.HeldNativeBasisWires = false
	for _, row := range a.Held {
		if row.Current != nil && row.Current.NativeBasis != nil {
			a.HeldNativeBasisWires = true
		}
	}
	for _, row := range rows {
		b := row.Basis
		r, err := nativeBasisRecord(doc, a, row.Entity, row.Removed)
		if err != nil {
			return err
		}
		v := currentNativeBasisWire{Entity: row.Entity}
		if r == nil {
			v.Unbound = true
			a.NativeBasisWires = append(a.NativeBasisWires, v)
			continue
		}
		if b.BaseKnown != 0 {
			raw, err := savedActorRaw(r, "U114", 24)
			if err != nil {
				return err
			}
			v.Base = bytes.Clone(raw)
		}
		if b.ModifierKnown != 0 {
			raw, err := savedActorRaw(r, "UD4", 64)
			if err != nil {
				return err
			}
			v.Modifier = bytes.Clone(raw)
		}
		if b.BodyKnown {
			body, err := savedStructureValue(r, "Body")
			if err != nil {
				return err
			}
			n := uint16(body)
			v.Body = &n
		}
		for _, block := range []struct {
			name  string
			known uint32
			dst   *[]byte
		}{{"UA6", b.AttackKnown, &v.Attack}, {"UBE", b.DefenceKnown, &v.Defence}} {
			if block.known != 0 {
				size := 24
				if block.name == "UBE" {
					size = 22
				}
				raw, err := savedActorRaw(r, block.name, size)
				if err != nil {
					return err
				}
				*block.dst = bytes.Clone(raw)
			}
		}
		if err := captureNativeScalars(b, r); err != nil {
			return err
		}
		a.NativeBasisWires = append(a.NativeBasisWires, v)
	}
	return nil
}

func matchCurrentNativeBasis(doc *sav.DocumentData, a *currentActionData) error {
	if len(a.NativeBasisWires) > 65535 || len(a.RemovedNativeBases) > 65535 {
		return fmt.Errorf("native basis wire population exceeds bound")
	}
	wires := make(map[sim.EntityID]currentNativeBasisWire, len(a.NativeBasisWires))
	for _, v := range a.NativeBasisWires {
		if _, ok := wires[v.Entity]; ok {
			return fmt.Errorf("native basis repeats a wire anchor")
		}
		wires[v.Entity] = v
	}
	rows, err := currentNativeBasisActors(a, a.HeldNativeBasisWires)
	if err != nil {
		return err
	}
	if a.HeldNativeBasisWires {
		held := false
		for _, row := range a.Held {
			held = held || row.Current != nil && row.Current.NativeBasis != nil
		}
		if !held {
			return fmt.Errorf("held native basis anchor marker has no held payload")
		}
	}
	for _, row := range rows {
		b := row.Basis
		if err := b.Validate(); err != nil {
			return err
		}
		if !b.HasValues() {
			return fmt.Errorf("empty native basis payload")
		}
		v, ok := wires[row.Entity]
		r, err := nativeBasisRecord(doc, a, row.Entity, row.Removed)
		if err != nil {
			return err
		}
		if r == nil {
			if !ok || !v.Unbound || len(v.Base) != 0 || len(v.Modifier) != 0 || len(v.Attack) != 0 || len(v.Defence) != 0 || v.Body != nil {
				return fmt.Errorf("removed native basis has conflicting missing ordinary anchors")
			}
			delete(wires, row.Entity)
			continue
		}
		if !ok || v.Unbound || (b.BaseKnown != 0) != (len(v.Base) == 24) || b.BaseKnown == 0 && len(v.Base) != 0 ||
			(b.ModifierKnown != 0) != (len(v.Modifier) == 64) || b.ModifierKnown == 0 && len(v.Modifier) != 0 || b.BodyKnown != (v.Body != nil) {
			return fmt.Errorf("native basis lacks exact ordinary wire anchors")
		}
		if b.BaseKnown != 0 {
			raw, err := savedActorRaw(r, "U114", 24)
			if err != nil {
				return err
			}
			for n := range b.Base {
				if b.BaseByteKnown(n) && raw[n] != v.Base[n] {
					b.Base[n] = raw[n]
				}
			}
		}
		if b.ModifierKnown != 0 {
			raw, err := savedActorRaw(r, "UD4", 64)
			if err != nil {
				return err
			}
			for n := range b.Modifier {
				if b.ModifierByteKnown(n) && raw[n] != v.Modifier[n] {
					b.Modifier[n] = raw[n]
				}
			}
		}
		if b.BodyKnown {
			body, err := savedStructureValue(r, "Body")
			if err != nil {
				return err
			}
			if uint16(body) != *v.Body {
				b.Body = uint16(body)
			}
		}
		for _, block := range []struct {
			name        string
			mask        uint32
			wire, value []byte
		}{{"UA6", b.AttackKnown, v.Attack, b.Attack[:]}, {"UBE", b.DefenceKnown, v.Defence, b.Defence[:]}} {
			if (block.mask != 0) != (len(block.wire) == len(block.value)) || block.mask == 0 && len(block.wire) != 0 {
				return fmt.Errorf("native live block lacks exact ordinary anchor")
			}
			if block.mask == 0 {
				continue
			}
			raw, err := savedActorRaw(r, block.name, len(block.value))
			if err != nil {
				return err
			}
			for n := range block.value {
				if block.mask&(uint32(1)<<n) != 0 && raw[n] != block.wire[n] {
					block.value[n] = raw[n]
				}
			}
		}
		if err := matchNativeScalars(b, r); err != nil {
			return err
		}
		delete(wires, row.Entity)
	}
	if len(wires) != 0 {
		return fmt.Errorf("native basis wire has no current payload")
	}
	return nil
}

// removedNativeBasisRecords resolves each removed actor's held basis to the
// record that carries it: its terminal binding, else its dead source root.
func removedNativeBasisRecords(state *SnapshotSAVDocument, w *sim.World) ([]*sav.DocumentRecordData, []sim.NativeActorBasis, error) {
	rows := w.RemovedNativeActorBases()
	if len(rows) == 0 {
		return nil, nil, nil
	}
	terminal, err := currentTerminalActorBindings(state, w)
	if err != nil {
		return nil, nil, err
	}
	var records []*sav.DocumentRecordData
	var bases []sim.NativeActorBasis
	for _, row := range rows {
		var record *sav.DocumentRecordData
		if object := terminal[row.ID]; object != 0 {
			record = &state.Document.Objects[object-1]
		} else {
			for _, dead := range w.OriginalDeadActors() {
				if dead.ID == row.ID {
					record, err = currentDeadSourceRoot(state, dead)
					if err != nil {
						return nil, nil, err
					}
					break
				}
			}
		}
		if record != nil {
			records, bases = append(records, record), append(bases, row.Basis)
		}
	}
	return records, bases, nil
}

func projectRemovedNativeBasis(state *SnapshotSAVDocument, w *sim.World) error {
	records, bases, err := removedNativeBasisRecords(state, w)
	if err != nil {
		return err
	}
	for i, record := range records {
		basis := bases[i]
		if err := projectNativeScalars(record, basis); err != nil {
			return err
		}
		if basis.BodyKnown {
			savedObjectSetValue(record, "Body", uint32(basis.Body))
		}
		for _, block := range []struct {
			name  string
			value []byte
			known uint64
		}{
			{"U114", basis.Base[:], uint64(basis.BaseKnown)}, {"UD4", basis.Modifier[:], basis.ModifierKnown},
			{"UA6", basis.Attack[:], uint64(basis.AttackKnown)}, {"UBE", basis.Defence[:], uint64(basis.DefenceKnown)},
		} {
			if block.known == 0 {
				continue
			}
			raw, err := savedActorRaw(record, block.name, len(block.value))
			if err != nil {
				return err
			}
			for i, value := range block.value {
				if block.known&(uint64(1)<<i) != 0 {
					raw[i] = value
				}
			}
		}
	}
	return nil
}
