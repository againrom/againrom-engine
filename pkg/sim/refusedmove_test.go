package sim

import "testing"

// refusedMove is the jammed corridor with the walker under a Move to a cell
// past the jam and the hostile ordered onto it: every walk the Move Group
// sends is refused at the jam while the hostile strikes the walker.
func refusedMove(t *testing.T, reach uint8) *World {
	t.Helper()
	w := jammedCorridor(t, reach, 4)
	wakeGroups(w)
	i, hi := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 3)
	w.entities[hi].HP, w.entities[hi].MaxHP = 2000, 2000
	w.entities[i].HP, w.entities[i].MaxHP = 2000, 2000
	w.commandGroup([]int{i}, orderMove, cell{x: 20, y: 4})
	w.walkAtStateOne([]int{i})
	w.issueGroupDestination([]int{i}, cell{x: 20, y: 4})
	w.orderAttack(hi, 1)
	return w
}

// refusedMoveRun is what one run of the fixture shows: the tick of the
// first stand-down, of the walker's pick and of its first blow on the
// hostile, each -1 when it never came, and the stand-downs counted.
type refusedMoveRun struct {
	standDown, pick, blow, standDowns int
}

// runRefusedMove steps w for twenty script cycles. between runs after each
// step; it may replace the World a later step uses.
func runRefusedMove(t *testing.T, w *World, between func(tick int, w *World) *World) refusedMoveRun {
	t.Helper()
	r := refusedMoveRun{standDown: -1, pick: -1, blow: -1}
	stood := false
	for k := 0; k < 20*scriptCycle; k++ {
		Step(w, nil)
		e, h := laEnt(t, w, 1), laEnt(t, w, 3)
		o := w.savedOrder(1)
		now := o != nil && o.stoodDown && o.State == uint32(actorStateAcquire)
		if now && !stood {
			r.standDowns++
			if r.standDown < 0 {
				r.standDown = k
				if e.HasAttackTarget {
					t.Fatalf("tick %d: the first refusal took victim %d", k, e.AttackTarget)
				}
				if e.ActorState != actorStateAcquire || e.PostX != e.X || e.PostY != e.Y || o.Raw[0] != uint8(e.X) || o.Raw[1] != uint8(e.Y) || o.Raw[8] != 0 {
					t.Fatalf("tick %d: stand-down state %#x post (%d,%d) order post (%d,%d) kind %d; want 0xc on (%d,%d), kind 0",
						k, e.ActorState, e.PostX, e.PostY, o.Raw[0], o.Raw[1], o.Raw[8], e.X, e.Y)
				}
			}
		}
		stood = now
		if r.pick < 0 && e.HasAttackTarget && e.AttackTarget == 3 {
			r.pick = k
		}
		if r.blow < 0 && h.HP < 2000 {
			r.blow = k
		}
		if between != nil {
			w = between(k, w)
		}
	}
	return r
}

// A Move member's first refused walk stands it down to state 0xc with its
// post on the jam and takes no pick; the Move arm sends it again, and its
// next refusal takes the hostile within reach and strikes it (AI-649). Loss
// control: with the stand-down dropped after every step, each refusal is a
// first one and the walker never takes a victim.
func TestARefusedMoveMemberTakesThePickOnALaterRefusal(t *testing.T) {
	t.Parallel()
	for _, reach := range []uint8{1, 3} {
		r := runRefusedMove(t, refusedMove(t, reach), nil)
		if r.standDown < 0 || r.pick <= r.standDown || r.blow < r.pick {
			t.Errorf("reach %d: stand-down tick %d, pick tick %d, first blow tick %d; want a pick after the stand-down and a blow after the pick", reach, r.standDown, r.pick, r.blow)
		}
		loss := runRefusedMove(t, refusedMove(t, reach), func(_ int, w *World) *World {
			if o := w.savedOrder(1); o != nil {
				o.stoodDown = false
			}
			return w
		})
		if loss.pick >= 0 || loss.standDowns < 2 {
			t.Errorf("reach %d loss control: pick tick %d after %d stand-downs; want no pick after at least two", reach, loss.pick, loss.standDowns)
		}
		t.Logf("reach %d: stand-down tick %d, pick tick %d, first blow tick %d; loss control %d stand-downs, no pick", reach, r.standDown, r.pick, r.blow, loss.standDowns)
	}
}

// The stand-down survives the World's byte form and an import that carries
// only the order's state word, as an original SAV LOAD does: a twin restored
// at the stand-down tick either way takes its pick and strikes on the same
// ticks. Loss control: the imported twin whose state word is not 0xc never
// picks.
func TestTheStandDownSurvivesTheByteFormAndAStateWordImport(t *testing.T) {
	t.Parallel()
	want := runRefusedMove(t, refusedMove(t, 3), nil)
	restore := func(stateWord bool, form bool) func(int, *World) *World {
		return func(k int, w *World) *World {
			if k != want.standDown {
				return w
			}
			b, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			twin := &World{}
			if err := twin.UnmarshalBinary(b); err != nil {
				t.Fatal(err)
			}
			if form {
				if twin.Hash() != w.Hash() {
					t.Fatal("the byte form changed the World")
				}
				return twin
			}
			groups, orders, _ := twin.SavedGroups()
			plain := make([]SavedActorOrder, len(orders))
			for n, o := range orders {
				plain[n] = SavedActorOrder{Entity: o.Entity, State: o.State, Authored: o.Authored, RepairStage: o.RepairStage,
					EscortTarget: o.EscortTarget, EscortBound: o.EscortBound, Raw: o.Raw, Patrol: o.Patrol}
				if o.Entity == 1 && !stateWord {
					plain[n].State = uint32(actorStateGuard)
				}
			}
			if err := twin.ImportOriginalGroups(groups, plain); err != nil {
				t.Fatal(err)
			}
			return twin
		}
	}
	for _, tc := range []struct {
		name            string
		stateWord, form bool
		pick            bool
	}{
		{"byte form", true, true, true},
		{"state word import", true, false, true},
		{"loss control: guard state word", false, false, false},
	} {
		got := runRefusedMove(t, refusedMove(t, 3), restore(tc.stateWord, tc.form))
		if tc.pick && (got.pick != want.pick || got.blow != want.blow) {
			t.Errorf("%s: pick tick %d, first blow tick %d; want %d and %d as without the restore", tc.name, got.pick, got.blow, want.pick, want.blow)
		}
		if !tc.pick && got.pick == want.pick {
			t.Errorf("%s: pick tick %d equals the restored run's", tc.name, got.pick)
		}
	}
}

// The setters put a member back at state 1: after a stand-down a new Move
// and a script Move (sub-command 4) each make the next refusal a first one
// again, and a Swarm 2 member stands down as a Move member does (AI-648).
func TestTheMoveSettersClearTheStandDown(t *testing.T) {
	t.Parallel()
	w := refusedMove(t, 3)
	i := indexOfEntity(w.entities, 1)
	w.standDownRefusedWalk(i)
	if w.refusedWalkArm(i) != walkArmPick {
		t.Fatal("a stood-down Move member does not take the pick")
	}
	w.commandGroup([]int{i}, orderMove, cell{x: 20, y: 4})
	w.walkAtStateOne([]int{i})
	if w.refusedWalkArm(i) != walkArmStandDown {
		t.Fatal("a new Move left the stand-down")
	}
	w.standDownRefusedWalk(i)
	w.cmdSavedGroupOrder(ScriptInstant{HasGroup: true, Group: w.savedGroupFor(1).Selector, Args: [scriptParams]int32{4, 20, 4}})
	if g := w.savedGroupFor(1); g.AI[0x20] != orderMove {
		t.Fatalf("fixture: the script Move left order %d", g.AI[0x20])
	}
	if w.refusedWalkArm(i) != walkArmStandDown {
		t.Fatal("the script setter left the stand-down")
	}
	w.commandGroup([]int{i}, orderSwarm2, cell{x: 20, y: 4})
	w.walkAtStateOne([]int{i})
	if w.refusedWalkArm(i) != walkArmStandDown {
		t.Fatal("a Swarm 2 member does not stand down")
	}
	w.standDownRefusedWalk(i)
	if w.refusedWalkArm(i) != walkArmPick {
		t.Fatal("a stood-down Swarm 2 member does not take the pick")
	}
}
