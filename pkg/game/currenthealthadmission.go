package game

import (
	"fmt"

	"againrom/pkg/sim"
)

type currentAdmissionHealth struct {
	HP, MaxHP         int32
	WireHP, WireMaxHP uint16
}

// Current HP words can look dead before their width operands are restored.
// Resolve only the two admission fields, by the existing ordinary identity,
// before living/dying classification. The registry carries this plan through
// stock, pools and lifecycle imports; final decoding uses the same wire basis.
func currentAdmissionHealthByIdentity(state *SnapshotSAVDocument) (map[uint32]currentAdmissionHealth, error) {
	if state == nil || state.Document == nil {
		return nil, nil
	}
	a, err := readCurrentActions(state.Document)
	if err != nil || a == nil || a.Values == nil {
		return nil, err
	}
	bindings := make(map[sim.EntityID]currentActionBinding)
	for _, b := range a.Bindings {
		if b.Structure {
			continue
		}
		if _, duplicate := bindings[b.ID]; duplicate {
			return nil, fmt.Errorf("current health admission has repeated actor binding %d", b.ID)
		}
		bindings[b.ID] = b
	}
	result := make(map[uint32]currentAdmissionHealth)
	seen := make(map[sim.EntityID]bool)
	for _, rows := range [][]sim.ActorContinuation{a.Actions.Actors, a.Held} {
		for _, row := range rows {
			if seen[row.Entity] {
				return nil, fmt.Errorf("current health admission has repeated actor %d", row.Entity)
			}
			seen[row.Entity] = true
			v, present := a.Values[row.Entity]
			if !present {
				continue // The complete continuation validates its required population.
			}
			b, present := bindings[row.Entity]
			if !present || b.Missing || b.Object == 0 || int(b.Object) > len(state.Document.Objects) {
				return nil, fmt.Errorf("current health admission actor %d has no ordinary binding", row.Entity)
			}
			record := &state.Document.Objects[b.Object-1]
			if record.Class != "Unit" && record.Class != "Human" && record.Class != "Humanoid" {
				return nil, fmt.Errorf("current health admission actor %d binds a non-actor", row.Entity)
			}
			key, err := savedStructureValue(record, "Identity")
			if err != nil || key == 0 {
				return nil, fmt.Errorf("current health admission actor %d has no ordinary identity", row.Entity)
			}
			if _, duplicate := result[key]; duplicate {
				return nil, fmt.Errorf("current health admission has repeated ordinary identity %d", key)
			}
			hp, hpErr := savedStructureValue(record, "Health")
			maximum, maxErr := savedStructureValue(record, "HealthMax")
			if hpErr != nil || maxErr != nil || hp > 65535 || maximum > 65535 {
				return nil, fmt.Errorf("current health admission has invalid ordinary words")
			}
			health, maximumHealth, err := sim.CurrentActorHealthFromWords(uint16(hp), uint16(maximum), v.Widths)
			if err != nil {
				return nil, err
			}
			result[key] = currentAdmissionHealth{HP: health, MaxHP: maximumHealth, WireHP: uint16(hp), WireMaxHP: uint16(maximum)}
		}
	}
	return result, nil
}
