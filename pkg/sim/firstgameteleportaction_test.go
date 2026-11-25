package sim

import (
	"bytes"
	"testing"
)

func teleportActionWorld(t *testing.T, script *Script, terrain Terrain, ents ...Entity) *World {
	t.Helper()
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, terrain, ents, script, engRel(t))
	if err != nil {
		t.Fatal(err)
	}
	wakeGroups(w)
	return w
}

// ScriptInstantTeleport moves any living unit standing on Source (player,
// ally, enemy, monster, ground or air) to Target through the trap teleport's
// one move, stands it there, and reports the Relocation event. When Target is
// taken, the unit takes the nearest free cell; an empty Source or a dead body
// moves nothing.
func TestFirstGameTeleportActionMovesAnyUnitAndFindsNearestFreeCell(t *testing.T) {
	t.Run("player ally enemy monster and air units", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			owner  uint32
			domain Domain
		}{
			{"player", SelfSlot, DomainGround},
			{"ally", 2, DomainGround},
			{"enemy", 3, DomainGround},
			{"monster", 4, DomainGround},
			{"air", 4, DomainAir},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s, err := NewExtendedScript(100, []ScriptCheck{constCheck(0, 1)}, []ScriptInstant{
					{Op: ScriptInstantTeleport, Args: [scriptParams]int32{10, 10, 30, 40}},
				}, []ScriptTrigger{
					{Instants: acts(), Latch: 0, Conditions: []ScriptPair{pair(0, 0, ScriptCmpEQ)}, Actions: []int32{0}},
				})
				if err != nil {
					t.Fatal(err)
				}
				u := Entity{ID: 1, Owner: tc.owner, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: tc.domain}
				w := teleportActionWorld(t, s, Terrain{}, u)
				var relocs []Relocation
				for range 32 {
					rep := StepReported(w, nil)
					relocs = append(relocs, rep.Relocations...)
				}
				e, _ := w.Entity(1)
				if e.X != 30 || e.Y != 40 {
					t.Fatalf("%s unit at (%d,%d), want (30,40)", tc.name, e.X, e.Y)
				}
				if e.HasTarget || e.ActorState != actorStateAcquire || e.PostX != 30 || e.PostY != 40 {
					t.Fatalf("after teleport: target %t state %#x post (%d,%d)", e.HasTarget, e.ActorState, e.PostX, e.PostY)
				}
				if len(relocs) != 1 || relocs[0] != (Relocation{Caster: 1, Owner: tc.owner, FromX: 10, FromY: 10, ToX: 30, ToY: 40}) {
					t.Fatalf("relocations %+v, want one relocation (10,10)->(30,40)", relocs)
				}
			})
		}
	})

	t.Run("taken target gives nearest free cell", func(t *testing.T) {
		s, err := NewExtendedScript(100, []ScriptCheck{constCheck(0, 1)}, []ScriptInstant{
			{Op: ScriptInstantTeleport, Args: [scriptParams]int32{10, 10, 30, 40}},
		}, []ScriptTrigger{
			{Instants: acts(), Latch: 0, Conditions: []ScriptPair{pair(0, 0, ScriptCmpEQ)}, Actions: []int32{0}},
		})
		if err != nil {
			t.Fatal(err)
		}
		blocker := Entity{ID: 1, Owner: 2, HP: 50, MaxHP: 50, X: 30, Y: 40, Domain: DomainGround}
		mover := Entity{ID: 2, Owner: SelfSlot, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: DomainGround}
		w := teleportActionWorld(t, s, Terrain{}, blocker, mover)
		var relocs []Relocation
		for range 32 {
			rep := StepReported(w, nil)
			relocs = append(relocs, rep.Relocations...)
		}
		e, _ := w.Entity(2)
		dx, dy := e.X-30, e.Y-40
		if dx*dx+dy*dy != 1 {
			t.Fatalf("mover at (%d,%d), want adjacent free cell to (30,40)", e.X, e.Y)
		}
		if len(relocs) != 1 || relocs[0].Caster != 2 || relocs[0].ToX != e.X || relocs[0].ToY != e.Y {
			t.Fatalf("relocations %+v, want one to (%d,%d)", relocs, e.X, e.Y)
		}
	})

	t.Run("empty source or dead body moves nothing", func(t *testing.T) {
		s, err := NewExtendedScript(100, []ScriptCheck{constCheck(0, 1)}, []ScriptInstant{
			{Op: ScriptInstantTeleport, Args: [scriptParams]int32{10, 10, 30, 40}},
		}, []ScriptTrigger{
			{Instants: acts(), Latch: 0, Conditions: []ScriptPair{pair(0, 0, ScriptCmpEQ)}, Actions: []int32{0}},
		})
		if err != nil {
			t.Fatal(err)
		}
		corpse := Entity{ID: 1, Owner: SelfSlot, HP: -5, MaxHP: 50, X: 10, Y: 10, Domain: DomainGround}
		bystander := Entity{ID: 2, Owner: SelfSlot, HP: 50, MaxHP: 50, X: 12, Y: 10, Domain: DomainGround}
		w := teleportActionWorld(t, s, Terrain{}, corpse, bystander)
		for range 32 {
			if rep := StepReported(w, nil); len(rep.Relocations) != 0 {
				t.Fatalf("unexpected relocations %+v", rep.Relocations)
			}
		}
		if e, _ := w.Entity(1); e.X != 10 || e.Y != 10 {
			t.Fatalf("corpse moved to (%d,%d)", e.X, e.Y)
		}
		if e, _ := w.Entity(2); e.X != 12 || e.Y != 10 {
			t.Fatalf("bystander moved to (%d,%d)", e.X, e.Y)
		}
	})
}

// A repeating trigger (Once: false) with an existing condition moves a first
// unit stepping on Source, round-trips through MarshalBinary/UnmarshalBinary
// during play, and moves a second unit stepping on Source later to Target or
// the nearest free cell when the first still stands there.
func TestFirstGameTeleportActionRepeatsAndRoundTripsSave(t *testing.T) {
	s, err := NewExtendedScript(100, []ScriptCheck{
		{Op: ScriptCheckNearest, Register: 0, Player: SelfSlot, HasPlayer: true, Args: [scriptParams]int32{10, 10}},
		constCheck(1, 0),
	}, []ScriptInstant{
		{Op: ScriptInstantTeleport, Args: [scriptParams]int32{10, 10, 30, 40}},
	}, []ScriptTrigger{
		{Instants: acts(), Once: false, Latch: 0, Conditions: []ScriptPair{pair(0, 1, ScriptCmpEQ)}, Actions: []int32{0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	u1 := engFighterOne(1, SelfSlot, 8, 10)
	u1.Speed = 16
	u2 := engFighterOne(2, SelfSlot, 6, 10)
	u2.Speed = 16
	w := teleportActionWorld(t, s, Terrain{}, u1, u2)

	cmds := []Command{MoveTo(1, CellPoint{X: 10, Y: 10})}
	for range 200 {
		Step(w, cmds)
		cmds = nil
		if e, _ := w.Entity(1); e.X == 30 && e.Y == 40 {
			break
		}
	}
	if e, _ := w.Entity(1); e.X != 30 || e.Y != 40 {
		t.Fatalf("first unit at (%d,%d), want (30,40)", e.X, e.Y)
	}

	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	again, err := loaded.MarshalBinary()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatalf("world save round-trip differed: %v", err)
	}

	cmds = []Command{MoveTo(2, CellPoint{X: 10, Y: 10})}
	moved := false
	for range 240 {
		rep := StepReported(&loaded, cmds)
		cmds = nil
		for _, r := range rep.Relocations {
			if r.Caster == 2 {
				moved = true
			}
		}
		if moved {
			break
		}
	}
	if !moved {
		t.Fatal("second unit never teleported after reload")
	}
	e2, _ := loaded.Entity(2)
	if e2.X != 30 || e2.Y != 39 {
		t.Fatalf("second unit at (%d,%d), want the free cell (30,39)", e2.X, e2.Y)
	}
}
