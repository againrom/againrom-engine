package sim

import "testing"

// referenced is an actor whose owner Reference scalar is a known Player key.
func referenced(e Entity, key uint32) Entity {
	e.NativeBasis.ScalarsPresent, e.NativeBasis.ScalarKnown = true, nativeScalarKnown
	e.NativeBasis.Scalars[ScalarReference] = key
	return e
}

func referenceOf(t *testing.T, w *World, id EntityID) (uint32, bool) {
	t.Helper()
	e, ok := w.Entity(id)
	if !ok {
		t.Fatalf("entity %d absent", id)
	}
	return e.NativeBasis.Scalars[ScalarReference], e.NativeBasis.ScalarIsKnown(ScalarReference)
}

// PARTY-JOIN-025 step 3 sets actor+0x14 to the new Player; UNIT-OWNER-009
// names actor+0x14 the owning Player. A joined actor carries the key its new
// Player's actors carry, not the key of the Player it left.
func TestHandOverWritesTheDestinationPlayerReference(t *testing.T) {
	t.Parallel()
	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Unit: 4, HasUnit: true, Player: 1, HasPlayer: true},
	}, 0)
	w := scriptWorld(t, s, []Entity{
		referenced(Entity{ID: 1, HP: 5, MaxHP: 5, Group: 2, Owner: 1}, 0x51000020),
		referenced(Entity{ID: 4, HP: 5, MaxHP: 5, Group: 7, Owner: 5}, 0x51000060),
		referenced(Entity{ID: 5, HP: 5, MaxHP: 5, Group: 7, Owner: 5}, 0x51000060),
	})
	runPass(t, w)
	if key, known := referenceOf(t, w, 4); key != 0x51000020 || !known {
		t.Fatalf("joined actor Reference %#x known %v, want the destination key 0x51000020", key, known)
	}
	if key, _ := referenceOf(t, w, 5); key != 0x51000060 {
		t.Fatalf("actor left behind changed Reference to %#x", key)
	}
}

// With no actor of the destination slot naming its Player, the World holds no
// key for it: the old key stops being a current value instead of naming the
// Player the actor left.
func TestHandOverToAPlayerWithNoKeyForgetsTheOldReference(t *testing.T) {
	t.Parallel()
	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Unit: 4, HasUnit: true, Player: 3, HasPlayer: true},
	}, 0)
	w := scriptWorld(t, s, []Entity{
		referenced(Entity{ID: 1, HP: 5, MaxHP: 5, Group: 2, Owner: 1}, 0x51000020),
		referenced(Entity{ID: 4, HP: 5, MaxHP: 5, Group: 7, Owner: 5}, 0x51000060),
	})
	runPass(t, w)
	if _, known := referenceOf(t, w, 4); known {
		t.Fatal("the Reference of the Player the actor left is still a known current value")
	}
}
