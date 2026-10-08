package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestRetiredActorEffectMaskProjectionUsesCurrentAbsence(t *testing.T) {
	_, ms := actorEffectFixture(t)
	sim.Step(ms.World, nil)
	sim.Step(ms.World, nil)
	state, err := snapshotSavedDocument(ms)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms.World.ActiveEffects()) != 0 || len(state.ActorEffects.Rows) != 1 || state.ActorEffects.Rows[0].ObjectIndex != 0 {
		t.Fatal("fixture did not naturally retire its attachment")
	}
	row := state.ActorEffects.Rows[0]
	var actor *sav.DocumentRecordData
	for _, binding := range state.Actors {
		if binding.EntityID == row.Entity {
			actor = &state.Document.Objects[binding.ObjectIndex-1]
		}
	}
	if actor == nil {
		t.Fatal("retired attachment lost its actor")
	}
	const otherBit = uint32(1) << 27
	// A known native scalar can replay the loaded mask before attachments.
	savedObjectSetValue(actor, "U144", otherBit|uint32(1)<<row.Spell)
	if validateSavedActorEffectsWorld(state, ms.World) == nil {
		t.Fatal("stale retired mask was accepted")
	}
	for range 2 {
		if err := projectSavedActorEffects(state, ms.World); err != nil {
			t.Fatal("current absence did not replace the retired bit", err)
		}
		for _, binding := range state.Actors {
			if binding.EntityID == row.Entity {
				actor = &state.Document.Objects[binding.ObjectIndex-1]
			}
		}
		mask, err := savedStructureValue(actor, "U144")
		refs, _ := savedObjectRefs(actor, "Effects")
		if err != nil || mask != otherBit || len(refs) != 0 || state.ActorEffects.Rows[0].ObjectIndex != 0 {
			t.Fatal("retired mask projection changed an independent bit or attachment", mask, refs, state.ActorEffects.Rows, err)
		}
	}
	savedObjectSetValue(actor, "U144", otherBit|uint32(1)<<row.Spell)
	if validateSavedActorEffectsWorld(state, ms.World) == nil {
		t.Fatal("post-projection retired mask loss was accepted")
	}
}
