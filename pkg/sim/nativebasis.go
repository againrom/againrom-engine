package sim

import (
	"cmp"
	"fmt"
	"slices"
)

const nativeBaseKnown = uint32(1<<24 - 1)

// Known bits identify current bytes. Unset bits may retain earlier history,
// but cannot supply an ordinary SAV field's current value.
type NativeActorBasis struct {
	BasePresent     bool
	BaseKnown       uint32
	Base            [24]byte
	ModifierPresent bool
	ModifierKnown   uint64
	Modifier        [64]byte
	BodyPresent     bool
	BodyKnown       bool
	Body            uint16
	AttackPresent   bool
	AttackKnown     uint32
	Attack          [24]byte
	DefencePresent  bool
	DefenceKnown    uint32
	Defence         [22]byte
	ScalarsPresent  bool
	ScalarKnown     uint32
	Scalars         [ScalarCount]uint32
	BlockPresent    bool
	BlockKnown      uint16
	Block           [10]byte
}

func (b NativeActorBasis) HasValues() bool {
	return b.hasCoreValues() || b.AttackPresent || b.DefencePresent || b.hasScalarValues()
}

func (b NativeActorBasis) hasCoreValues() bool {
	return b.BasePresent || b.ModifierPresent || b.BodyPresent
}

func (b NativeActorBasis) Validate() error {
	if b.BaseKnown & ^nativeBaseKnown != 0 || !b.BasePresent && (b.BaseKnown != 0 || b.Base != ([24]byte{})) {
		return fmt.Errorf("sim: invalid native actor base availability")
	}
	if !b.ModifierPresent && (b.ModifierKnown != 0 || b.Modifier != ([64]byte{})) {
		return fmt.Errorf("sim: absent native actor modifier has residue")
	}
	if !b.BodyPresent && (b.BodyKnown || b.Body != 0) {
		return fmt.Errorf("sim: absent native actor body has residue")
	}
	if b.AttackKnown & ^nativeBaseKnown != 0 || !b.AttackPresent && (b.AttackKnown != 0 || b.Attack != ([24]byte{})) ||
		b.DefenceKnown & ^uint32(1<<22-1) != 0 || !b.DefencePresent && (b.DefenceKnown != 0 || b.Defence != ([22]byte{})) {
		return fmt.Errorf("sim: invalid native live block availability")
	}
	return b.validateScalars()
}

func (b NativeActorBasis) DefenceByteKnown(index int) bool {
	return index >= 0 && index < len(b.Defence) && b.DefencePresent && b.DefenceKnown&(uint32(1)<<index) != 0
}

func (b NativeActorBasis) AttackByteKnown(index int) bool {
	return index >= 0 && index < len(b.Attack) && b.AttackPresent && b.AttackKnown&(uint32(1)<<index) != 0
}

func (b NativeActorBasis) WithBase(value [24]byte) NativeActorBasis {
	b.BasePresent, b.BaseKnown, b.Base = true, nativeBaseKnown, value
	return b
}

func (b NativeActorBasis) WithModifier(value [64]byte) NativeActorBasis {
	b.ModifierPresent, b.ModifierKnown, b.Modifier = true, ^uint64(0), value
	return b
}

func (b NativeActorBasis) WithBody(value uint16) NativeActorBasis {
	b.BodyPresent, b.BodyKnown, b.Body = true, true, value
	return b
}

func (b NativeActorBasis) BaseByteKnown(index int) bool {
	return index >= 0 && index < len(b.Base) && b.BasePresent && b.BaseKnown&(uint32(1)<<index) != 0
}

func (b NativeActorBasis) ModifierByteKnown(index int) bool {
	return index >= 0 && index < len(b.Modifier) && b.ModifierPresent && b.ModifierKnown&(uint64(1)<<index) != 0
}

type NativeActorBasisRecord struct {
	ID    EntityID
	Basis NativeActorBasis
}

func (w *World) RemovedNativeActorBases() []NativeActorBasisRecord {
	out := slices.Clone(w.removedNativeBases)
	for i := range out {
		out[i].Basis = w.nativeRemovedBasisNow(out[i])
	}
	return out
}

func (w *World) hasRemovedNativeBasisOwner(id EntityID) bool {
	for _, row := range w.currentTerminalActors {
		if row.ID == id {
			return true
		}
	}
	for _, row := range w.originalDead {
		if row.ID == id && row.terminal.Stage != 0 {
			return true
		}
	}
	return false
}

func (w *World) retainRemovedNativeBasis(e Entity) {
	if e.ActorLoad.Source.Class != 0 || !e.NativeBasis.HasValues() || !w.hasRemovedNativeBasisOwner(e.ID) {
		return
	}
	for i, row := range w.removedNativeBases {
		if row.ID == e.ID {
			w.removedNativeBases[i].Basis = w.nativeBasisNow(e)
			return
		}
	}
	w.removedNativeBases = append(w.removedNativeBases, NativeActorBasisRecord{ID: e.ID, Basis: w.nativeBasisNow(e)})
	w.sortRemovedNativeBases()
}

func (w *World) sortRemovedNativeBases() {
	slices.SortFunc(w.removedNativeBases, func(a, b NativeActorBasisRecord) int { return cmp.Compare(a.ID, b.ID) })
}

func (w *World) RestoreNativeActorBases(records []NativeActorBasisRecord) error {
	if len(records) > 65535 {
		return fmt.Errorf("sim: too many native actor basis records")
	}
	var previous EntityID
	owners := make(map[EntityID]bool, len(w.removedNativeBases)+len(records))
	for _, row := range w.removedNativeBases {
		owners[row.ID] = true
	}
	for _, e := range w.entities {
		if e.NativeBasis.HasValues() {
			owners[e.ID] = true
		}
	}
	for n, r := range records {
		index := indexOfEntity(w.entities, r.ID)
		if n > 0 && r.ID <= previous || !r.Basis.HasValues() ||
			index >= 0 && w.entities[index].ActorLoad.Source.Class != 0 || index < 0 && !w.hasRemovedNativeBasisOwner(r.ID) {
			return fmt.Errorf("sim: invalid native actor basis membership")
		}
		if err := r.Basis.Validate(); err != nil {
			return err
		}
		previous = r.ID
		owners[r.ID] = true
	}
	if len(owners) > 65535 {
		return fmt.Errorf("sim: too many native actor bases")
	}
	for _, r := range records {
		if index := indexOfEntity(w.entities, r.ID); index >= 0 {
			w.entities[index].NativeBasis = r.Basis
			continue
		}
		found := false
		for i, row := range w.removedNativeBases {
			if row.ID == r.ID {
				w.removedNativeBases[i] = r
				found = true
				break
			}
		}
		if !found {
			w.removedNativeBases = append(w.removedNativeBases, r)
		}
	}
	w.sortRemovedNativeBases()
	return nil
}

func (w *World) nativeBasisFault() error {
	count := len(w.removedNativeBases)
	for i, row := range w.removedNativeBases {
		if i > 0 && row.ID <= w.removedNativeBases[i-1].ID || indexOfEntity(w.entities, row.ID) >= 0 ||
			!w.hasRemovedNativeBasisOwner(row.ID) || !row.Basis.HasValues() {
			return fmt.Errorf("sim: invalid removed native actor basis owner")
		}
		if err := row.Basis.Validate(); err != nil {
			return err
		}
	}
	for _, e := range w.entities {
		if err := e.NativeBasis.Validate(); err != nil {
			return err
		}
		if e.NativeBasis.HasValues() && e.ActorLoad.Source.Class != 0 {
			return fmt.Errorf("sim: native actor basis has a source-backed owner")
		}
		if e.NativeBasis.HasValues() {
			count++
		}
	}
	if count > 65535 {
		return fmt.Errorf("sim: too many native actor bases")
	}
	return nil
}
