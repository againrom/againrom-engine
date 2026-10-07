package sim

import (
	"encoding/binary"
	"strings"
	"testing"
)

func engagement1163World(t *testing.T) *World {
	w := savedIncomingEscortWorld(t)
	o := w.savedOrder(10)
	o.State, o.EscortTarget, o.EscortBound = 3, 0, false
	binary.LittleEndian.PutUint32(o.Raw[0x0c:], 1002)
	w.entities[0].Reach, w.entities[0].AttackCharge = 4, 10
	return w
}

func TestEngagement1163SourceIdentityAndColdContinuation(t *testing.T) {
	w := engagement1163World(t)
	if issue := w.savedPrimaryIssue(&w.savedGroups.Groups[0]); issue != "" {
		t.Fatal(issue)
	}
	w.savedActorDispatch(0)
	e := &w.entities[0]
	if !e.HasAttackTarget || e.AttackTarget != 30 || e.ActorState != 3 || w.savedOrder(10).Raw[8] != 5 || w.savedOrder(10).Raw[0x14] != 4 {
		t.Fatal("source key1002 did not engage native actor30", *e)
	}
	// A repeated AI decision must not restart an already charging attack.
	e.AttackPhase, e.AttackCountdown = AttackCharging, 7
	w.savedActorDispatch(0)
	if e.AttackPhase != AttackCharging || e.AttackCountdown != 7 {
		t.Fatal("same source target restarted the attack cycle")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for range 8 {
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("cold native engagement diverged")
		}
	}
}

func TestEngagement1163BadSourceKeysRefuseWholePrimary(t *testing.T) {
	for _, mutation := range []string{"zero", "native-id", "missing", "collision", "self", "stage"} {
		t.Run(mutation, func(t *testing.T) {
			w := engagement1163World(t)
			w.savedOrder(20).State = 1 // This earlier member must not move.
			o := w.savedOrder(10)
			switch mutation {
			case "zero":
				binary.LittleEndian.PutUint32(o.Raw[0x0c:], 0)
			case "native-id":
				binary.LittleEndian.PutUint32(o.Raw[0x0c:], 30)
			case "missing":
				binary.LittleEndian.PutUint32(o.Raw[0x0c:], 0xdeadbeef)
			case "collision":
				w.entities[1].SourceBinding.Identity = 1002
			case "self":
				binary.LittleEndian.PutUint32(o.Raw[0x0c:], 1000)
			case "stage":
				o.RepairStage = 2
			}
			before := w.Hash()
			issue := w.savedPrimaryIssue(&w.savedGroups.Groups[0])
			w.savedGroupPass(nil)
			if !strings.Contains(issue, "engagement") || before != w.Hash() || w.entities[1].HasTarget {
				t.Fatal("bad target guessed or partially dispatched", issue)
			}
		})
	}
}

func TestEngagement1163NativeZeroIsARealTarget(t *testing.T) {
	w := mustWorld(t, 113, Bounds{Width: 32, Height: 32}, []Entity{
		{ID: 0, Owner: 2, X: 10, Y: 8, HP: 100, MaxHP: 100, SourceBinding: SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 0x3000}},
		{ID: 10, Owner: 1, X: 8, Y: 8, HP: 100, MaxHP: 100, Reach: 4},
	})
	g := SavedGroup{ID: 71, Selector: 19, Members: []SavedGroupMember{{Archive: 2, Entity: 10, Bound: true}}}
	g.AI[0x45] = 1
	o := SavedActorOrder{Entity: 10, State: 3}
	binary.LittleEndian.PutUint32(o.Raw[0x0c:], 0x3000)
	if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
		t.Fatal(err)
	}
	w.savedActorDispatch(1)
	if !w.entities[1].HasAttackTarget || w.entities[1].AttackTarget != 0 {
		t.Fatal("native entity zero treated as missing target")
	}
}

func TestEngagement1163RefusedCastCannotAuthorLatentAttack(t *testing.T) {
	for _, script := range []bool{false, true} {
		w := engagement1163World(t)
		e := &w.entities[0]
		e.ActorState, e.CastWait = actorStateAcquire, 200
		o := w.savedOrder(10)
		o.State, o.Authored = uint32(actorStateAcquire), true
		if script {
			w.cmdGroupAttack(ScriptInstant{Group: 19, HasGroup: true, Unit: 30, HasUnit: true})
			if e.ActorState == 3 || o.State == 3 || e.HasAttackTarget || e.CastWait == 0 {
				t.Fatal("refused attack authored latent engagement", script, e.ActorState, o.State, e.HasAttackTarget, e.CastWait)
			}
			continue
		}
		// The player's own command is taken beside the recovery, typed in the
		// saved order and not latent, and the recovery is left running.
		Step(w, []Command{{Kind: KindAttack, Entity: 10, X: 30}})
		e, o = &w.entities[0], w.savedOrder(10)
		if e.ActorState != 3 || o.State != 3 || !o.Authored || !e.HasAttackTarget || e.AttackTarget != 30 || e.CastWait == 0 {
			t.Fatal("player attack beside a recovery was not taken as a typed engagement", e.ActorState, o.State, e.HasAttackTarget, e.CastWait)
		}
	}
}

func TestEngagement1163PlayerVetoAndAcceptedOrder(t *testing.T) {
	for _, veto := range []bool{false, true} {
		w := engagement1163World(t)
		w.entities[0].Reach = 1
		if veto {
			w.entities[2].Domain = DomainAir
		}
		if w.targetVetoed(0, 2) != veto {
			t.Fatal("fixture does not discriminate preference veto")
		}
		Step(w, []Command{{Kind: KindAttack, Entity: 10, X: 30}})
		e, o := w.entities[0], w.savedOrder(10)
		if veto {
			if e.ActorState != actorStateAcquire || o.State != uint32(actorStateAcquire) || e.HasAttackTarget || e.PostX != 8 || e.PostY != 8 {
				t.Fatal("veto did not acquire at the current post", e, *o)
			}
		} else if e.ActorState != 3 || o.State != 3 || !o.Authored || e.AttackTarget != 30 || !e.HasAttackTarget {
			t.Fatal("accepted player attack lost typed command", e, *o)
		}
	}
}

func TestEngagement1163StructureTargetNeverBecomesAnActorKey(t *testing.T) {
	w := engagement1163World(t)
	// Actor30 and structure30 deliberately share the numeric ID. The old
	// incoming actor order also retains a valid source key for actor30.
	w.structures = []Structure{{ID: 30, Col: 25, Row: 24, Field42: 100, MaxHealth: 100, Width: 1, Height: 1, Attach: 1}}
	w.rebuildStructureSlots()
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 10, X: 30}})
	for range 2 {
		w.savedActorDispatch(0)
		e := w.entities[0]
		if !e.HasAttackTarget || e.AttackTarget != 30 || e.AttackTargetKind != AttackTargetStructure || !w.savedOrder(10).Authored {
			t.Fatal("typed structure target reinterpreted as actor", e.AttackTargetKind)
		}
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if cold.entities[0].AttackTargetKind != AttackTargetStructure {
		t.Fatal("cold LOAD lost structure target kind")
	}
}

func TestEngagement1163PlayerVetoReplacesPriorAction(t *testing.T) {
	for _, prior := range []string{"move", "attack"} {
		t.Run(prior, func(t *testing.T) {
			w := engagement1163World(t)
			w.entities[0].Reach = 1
			w.entities[1].Owner = 2
			w.entities[2].Domain = DomainAir
			if prior == "move" {
				Step(w, []Command{{Kind: KindMoveTo, Entity: 10, X: 16, Y: 12}})
				if !w.entities[0].HasTarget {
					t.Fatal("prior player move was not admitted")
				}
			} else {
				Step(w, []Command{{Kind: KindAttack, Entity: 10, X: 20}})
				if !w.entities[0].HasAttackTarget || w.entities[0].AttackPhase != AttackCharging {
					t.Fatal("prior player attack did not start charging")
				}
			}
			if !w.targetVetoed(0, 2) {
				t.Fatal("fixture did not reach target preference veto")
			}
			Step(w, []Command{{Kind: KindAttack, Entity: 10, X: 30}})
			e, order := w.entities[0], w.savedOrder(10)
			if e.ActorState != actorStateAcquire || order.State != uint32(actorStateAcquire) || !order.Authored || e.HasTarget || len(w.routes[0]) != 0 || e.Stall != 0 || e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
				t.Fatalf("veto left prior %s active: state=%d move=%v route=%v attack=%v phase=%d count=%d order=%d", prior, e.ActorState, e.HasTarget, w.routes[0], e.HasAttackTarget, e.AttackPhase, e.AttackCountdown, order.State)
			}
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			for range 8 {
				Step(w, nil)
				Step(&cold, nil)
				if w.Hash() != cold.Hash() {
					t.Fatal("native LOAD changed veto continuation")
				}
			}
		})
	}
}
