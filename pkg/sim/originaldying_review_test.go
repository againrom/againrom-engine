package sim

import "testing"

func TestOriginalDying1144ZeroTimerDoesNotAdvanceStage(t *testing.T) {
	w := sourceBindingWorld1111(t)
	w.entities[0].HP, w.entities[0].Decay, w.entities[0].Dwell = -10, DecayFallen, 0
	w.entities[0].ActorLoad = ActorLoad{}
	w.carried[0] = []ItemStack{{Code: 0x101, Count: 1}}
	err := w.ImportOriginalDyingActors([]OriginalDyingActor{{ID: 7, HP: -10, Timer: 0}})
	t.Logf("result err=%v HP=%d stage=%d timer=%d sacks=%d carried=%d", err, w.entities[0].HP, w.entities[0].Decay, w.entities[0].Dwell, len(w.sacks), len(w.carried[0]))
	if err != nil || w.entities[0].Decay != DecayFallen || len(w.sacks) != 0 || len(w.carried[0]) != 1 {
		t.Fatal("import advanced the source stage or replayed terminal loot before a tick")
	}
}

func TestOriginalDying1144LateAdmissionRefusalKeepsSavedGroups(t *testing.T) {
	w := sourceBindingWorld1111(t)
	if err := w.ImportSavedGroups([]SavedGroup{{ID: 1, Members: []SavedGroupMember{{Archive: 4, Entity: 7, Bound: true}}}}, nil); err != nil {
		t.Fatal(err)
	}
	e := w.entities[0]
	e.HP, e.Decay, e.Dwell, e.TokenSize = 0, DecayFallen, 3, 1
	late := e
	late.ID = 8 // duplicate archive identity passes the first guard, fails wire validation
	before := w.Hash()
	err := w.ImportOriginalLivingActors([]OriginalLivingActor{{Entity: e}, {Entity: late, New: true}})
	t.Logf("err=%v hash-before=%x after=%x members=%d", err, before, w.Hash(), len(w.savedGroups.Groups[0].Members))
	if err == nil || before != w.Hash() {
		t.Fatal("refused admission changed the original world")
	}
}
