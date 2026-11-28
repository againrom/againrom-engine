package sim

import (
	"encoding/binary"
	"testing"
)

func preTableEscortWorld(t *testing.T, second bool, sub int32, ents ...Entity) *World {
	t.Helper()
	w := asWorld(t, second, ents...)
	w.cmdGroupOrder(ScriptInstant{
		Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{sub, 3},
	})
	return w
}

func TestTerminalEscortSubjectStandsDownAndAcquiresInBothGames(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, tc := range []struct {
			name string
			sub  int32
			want uint8
		}{
			{"defend", subCommandDefend, actorStateDefend},
			{"follow", subCommandFollow, actorStateFollow},
		} {
			game := "rom1"
			if second {
				game = "rom2"
			}
			t.Run(game+"/"+tc.name+"/dying_keeps_escort", func(t *testing.T) {
				subject := laFighter(1, SelfSlot, 7, 10, 10)
				escort := laFighter(2, SelfSlot, 7, 20, 10)
				w := preTableEscortWorld(t, second, tc.sub, subject, escort)
				w.entities[0].setCurrentHealth(-1)
				esPass(w)
				e := laEnt(t, w, 2)
				if e.ActorState != tc.want || !e.HasEscortTarget || e.EscortTarget != 1 {
					t.Fatalf("dying subject ended escort early: %+v", e)
				}
			})
			t.Run(game+"/"+tc.name+"/terminal_stands_down_and_acquires", func(t *testing.T) {
				subject := laFighter(1, SelfSlot, 7, 10, 10)
				escort := laFighter(2, SelfSlot, 7, 20, 10)
				foe := laFighter(3, asSide, 8, 21, 10)
				w := preTableEscortWorld(t, second, tc.sub, subject, escort, foe)
				o := w.savedOrder(2)
				o.Raw[9] = 1
				w.entities[0].setCurrentHealth(-10)
				esPass(w)
				e := laEnt(t, w, 2)
				o = w.savedOrder(2)
				if e.ActorState != actorStateAcquire || e.HasEscortTarget || e.EscortTarget != 0 || e.EscortRange != 0 {
					t.Fatalf("terminal subject did not stand down escort: %+v", e)
				}
				if e.PostX != e.X || e.PostY != e.Y {
					t.Fatalf("post (%d,%d), want own cell (%d,%d)", e.PostX, e.PostY, e.X, e.Y)
				}
				if o.State != uint32(actorStateAcquire) || binary.LittleEndian.Uint32(o.Raw[0x50:]) != 1 || o.Raw[9] != 1 {
					t.Fatalf("saved order after stand-down: state=%d complete=%d progress=%d", o.State, binary.LittleEndian.Uint32(o.Raw[0x50:]), o.Raw[9])
				}
				if !e.HasAttackTarget || e.AttackTarget != 3 {
					t.Fatalf("escort did not acquire adjacent hostile in the same pass: %+v", e)
				}
			})
		}
	}
}

func TestTerminalEngageTargetStandsDownAndAcquiresInBothGames(t *testing.T) {
	for _, second := range []bool{false, true} {
		game := "rom1"
		if second {
			game = "rom2"
		}
		t.Run(game+"/terminal_victim_stands_down_and_acquires", func(t *testing.T) {
			attacker := laFighter(1, SelfSlot, 7, 10, 10)
			victim := laFighter(2, asSide, 8, 12, 10)
			foe := laFighter(3, asSide, 8, 11, 10)
			attacker.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 1}
			victim.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 2, Identity: 2}
			foe.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 3, Identity: 3}
			w := asWorld(t, second, attacker, victim, foe)
			w.entities[0].ActorState = actorStateEngage
			w.entities[1].setCurrentHealth(-10)
			o := w.ensureSavedOrder(0)
			o.State = uint32(actorStateEngage)
			o.Authored = false
			o.Raw[8], o.Raw[9] = 5, 1
			binary.LittleEndian.PutUint32(o.Raw[0x0c:], 2)
			w.savedEngage(0, o)
			e := w.entities[0]
			if e.ActorState != actorStateAcquire || o.State != uint32(actorStateAcquire) {
				t.Fatalf("state %d order %d, want acquire", e.ActorState, o.State)
			}
			if e.PostX != e.X || e.PostY != e.Y || binary.LittleEndian.Uint32(o.Raw[0x50:]) != 1 || o.Raw[9] != 1 {
				t.Fatalf("stand-down fields: post=(%d,%d) complete=%d progress=%d", e.PostX, e.PostY, binary.LittleEndian.Uint32(o.Raw[0x50:]), o.Raw[9])
			}
			if !e.HasAttackTarget || e.AttackTarget != 3 {
				t.Fatalf("attacker did not acquire adjacent hostile in the same pass: %+v", e)
			}
		})
	}
}

func TestTerminalEscortStandDownReplayHashAndRoundTrip(t *testing.T) {
	for _, second := range []bool{false, true} {
		game := "rom1"
		if second {
			game = "rom2"
		}
		t.Run(game, func(t *testing.T) {
			run := func() (*World, bool) {
				subject := laFighter(1, SelfSlot, 7, 10, 10)
				subject.HP = 1
				killer := laFighter(4, asSide, 9, 11, 10)
				killer.DamageBase = 25
				escort := laFighter(2, SelfSlot, 7, 20, 10)
				foe := laFighter(3, asSide, 8, 21, 10)
				w := preTableEscortWorld(t, second, subCommandDefend, subject, escort, foe, killer)
				acquired := false
				for range 60 {
					Step(w, nil)
					e := laEnt(t, w, 2)
					if e.ActorState == actorStateAcquire && !e.HasEscortTarget && e.HasAttackTarget && e.AttackTarget == 3 {
						acquired = true
					}
				}
				return w, acquired
			}
			w1, ok1 := run()
			w2, ok2 := run()
			if !ok1 || !ok2 {
				t.Fatalf("escort did not stand down and acquire foe during replay: ok1=%v ok2=%v", ok1, ok2)
			}
			if w1.Hash() != w2.Hash() {
				t.Fatalf("replay hash mismatch: %x vs %x", w1.Hash(), w2.Hash())
			}
			raw, err := w1.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}
			if cold.Hash() != w1.Hash() {
				t.Fatalf("round-trip hash mismatch: %x vs %x", cold.Hash(), w1.Hash())
			}
			for range 16 {
				Step(w1, nil)
				Step(&cold, nil)
			}
			if cold.Hash() != w1.Hash() {
				t.Fatalf("post-reload replay hash mismatch: %x vs %x", cold.Hash(), w1.Hash())
			}
		})
	}
}
