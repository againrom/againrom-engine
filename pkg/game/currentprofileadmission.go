package game

import (
	"fmt"

	"againrom/pkg/sim"
)

// Arithmetic policy has no ordinary actor field. Resolve its existing action
// binding before importing ordinary profiles and their final pool values.
func currentProfileBases(state *SnapshotSAVDocument) (map[uint32]sim.CurrentProfileBasis, error) {
	if state == nil || state.Document == nil {
		return nil, nil
	}
	a, err := readCurrentActions(state.Document)
	if err != nil || a == nil {
		return nil, err
	}
	bindings := make(map[sim.EntityID]currentActionBinding)
	for _, b := range a.Bindings {
		if b.Structure {
			continue
		}
		if _, duplicate := bindings[b.ID]; duplicate {
			return nil, fmt.Errorf("current profile has repeated actor binding %d", b.ID)
		}
		bindings[b.ID] = b
	}
	bases := make(map[uint32]sim.CurrentProfileBasis)
	seen := make(map[sim.EntityID]bool)
	for _, rows := range [][]sim.ActorContinuation{a.Actions.Actors, a.Held} {
		for _, row := range rows {
			if seen[row.Entity] {
				return nil, fmt.Errorf("current profile has repeated actor %d", row.Entity)
			}
			seen[row.Entity] = true
			if row.ProfileBasis == nil {
				continue
			}
			if *row.ProfileBasis > sim.ProfileNativeRetired {
				return nil, fmt.Errorf("current profile has invalid arithmetic policy %d", *row.ProfileBasis)
			}
			b, ok := bindings[row.Entity]
			if !ok || b.Missing || b.Object == 0 || int(b.Object) > len(state.Document.Objects) {
				return nil, fmt.Errorf("current profile actor %d has no ordinary binding", row.Entity)
			}
			record := &state.Document.Objects[b.Object-1]
			if record.Class != "Unit" && record.Class != "Human" && record.Class != "Humanoid" {
				return nil, fmt.Errorf("current profile actor %d binds a non-actor", row.Entity)
			}
			key, err := savedStructureValue(record, "Identity")
			if err != nil || key == 0 {
				return nil, fmt.Errorf("current profile actor %d has no ordinary identity", row.Entity)
			}
			if _, duplicate := bases[key]; duplicate {
				return nil, fmt.Errorf("current profile has repeated ordinary identity %d", key)
			}
			bases[key] = *row.ProfileBasis
		}
	}
	return bases, nil
}
