package game

import (
	"testing"

	"againrom/pkg/sim"
)

// A script-undying unit at health 0 or below is living to every decision
// outside the simulation that asks whether a unit lives (DIV-4009): it can
// voice an acknowledgement, its local Teleport reveals the fog, and the
// headless scenario reads it as alive. A plain unit at the same health is a
// body to each of them.
func TestUndyingUnitIsLivingOutsideTheSimulation(t *testing.T) {
	undyingHero := heroMan
	undyingHero.hp, undyingHero.undying = -5, true
	fallenHero := heroMan
	fallenHero.hp = -5
	mw := voiceFixture(t, undyingHero, fallenHero)
	const undying, fallen = 1, 2

	script := &scriptedDraw{}
	script.set(0)
	if e, _, ok := mw.speakerOf([]uint32{undying}, script.draw); !ok || e.ID != undying {
		t.Errorf("the undying hero at -5 voices nothing: speaker %d (%v)", e.ID, ok)
	}
	script.set(0)
	if _, _, ok := mw.speakerOf([]uint32{fallen}, script.draw); ok {
		t.Error("the fallen hero at -5 voices an acknowledgement")
	}

	for _, tc := range []struct {
		id   sim.EntityID
		want bool
	}{{undying, true}, {fallen, false}} {
		e, _ := mw.entity(tc.id)
		moved := []sim.Relocation{{Caster: tc.id, Owner: sim.SelfSlot, FromX: 9, FromY: 9, ToX: e.X, ToY: e.Y}}
		if got := mw.localTeleportArrived(moved); got != tc.want {
			t.Errorf("Teleport of entity %d at -5 reveals the fog %v, want %v", tc.id, got, tc.want)
		}
	}

	p := &PlayWorld{World: mw.world}
	yes, no := true, false
	for _, tc := range []struct {
		ref   string
		alive bool
	}{{"e1", true}, {"e2", false}} {
		dead := !tc.alive
		if held, err := (&HeadlessUntil{Unit: tc.ref, Dead: &dead}).holds(p); err != nil || !held {
			t.Errorf("%s: until dead=%v holds %v (%v)", tc.ref, dead, held, err)
		}
		want := &no
		if tc.alive {
			want = &yes
		}
		if err := (&HeadlessUnitAssertion{Alive: want}).check(p, tc.ref); err != nil {
			t.Errorf("%s: %v", tc.ref, err)
		}
	}
	if alive, fell := headlessCounts(p, map[sim.EntityID]bool{undying: true, fallen: true}); alive != 1 || fell != 1 {
		t.Errorf("headless counts alive %d fallen %d, want 1 and 1", alive, fell)
	}
}
