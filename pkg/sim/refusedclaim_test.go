package sim

import (
	"encoding/binary"
	"testing"
)

// pocketGroup is a walker 1 sealed in a one-cell pocket at (2,4) in a
// Group at order whose members are sent to (20,4), with a hostile 3 in its
// own one-cell pocket at hostile: the far search finds no route, the static
// search's refusal. The walker sees scan cells and reaches reach.
func pocketGroup(t *testing.T, size int32, order uint8, hostile cell, scan, reach uint8) *World {
	t.Helper()
	b := Bounds{Width: size, Height: size}
	grid := make([]byte, size*size)
	for k := range grid {
		grid[k] = blockGround
	}
	for _, c := range []cell{{2, 4}, hostile, {20, 4}, {21, 4}} {
		grid[c.y*size+c.x] = 0
	}
	walker := engFighter(1, 2, 2, 4)
	walker.Reach, walker.ScanRange = reach, scan
	walker.HP, walker.MaxHP = 2000, 2000
	h := engFighter(3, 3, hostile.x, hostile.y)
	h.HP, h.MaxHP, h.ScanRange = 2000, 2000, 0
	w, err := NewRelatedWorld(1, b, ModeCanonical, Terrain{Block: grid}, []Entity{walker, h}, nil, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	wakeGroups(w)
	i := indexOfEntity(w.entities, 1)
	w.commandGroup([]int{i}, orderMove, cell{x: 20, y: 4})
	if order == orderMove {
		w.walkAtStateOne([]int{i})
		w.issueGroupDestination([]int{i}, cell{x: 20, y: 4})
		return w
	}
	g := w.savedGroupFor(1)
	g.AI[0x20] = order
	binary.LittleEndian.PutUint16(g.AI[10:], packedCell(20, 4))
	if order == orderRoam {
		binary.LittleEndian.PutUint16(g.AI[10:], packedCell(size/2, size/2))
	}
	e, o := &w.entities[i], w.savedOrder(1)
	e.ActorState, o.State = actorStateAcquire, uint32(actorStateAcquire)
	binary.LittleEndian.PutUint16(o.Raw[10:], packedCell(20, 4))
	return w
}

// claimRun is what one run shows of walker 1: the step of the first claim,
// of the first latch and of each Group pass, the steps it walked after the
// latch, how often it took a new victim, on how many passes after the claim
// it was sent a walk (pending kind 1), and how often its cell was claimed.
type claimRun struct {
	claim, latch                 int
	passes                       []int
	walks, picks, resent, claims int
}

func latched(o *SavedActorOrder) bool { return binary.LittleEndian.Uint32(o.Raw[0x50:]) != 0 }

// runClaim steps w for cycles script cycles; between runs after each step.
func runClaim(t *testing.T, w *World, cycles int, between func(*World)) claimRun {
	t.Helper()
	r := claimRun{claim: -1, latch: -1}
	victim, wasArrived := false, false
	for k := 0; k < cycles*scriptCycle; k++ {
		Step(w, nil)
		e, o := laEnt(t, w, 1), w.savedOrder(1)
		pass := w.ScriptPassJustRan()
		if pass {
			r.passes = append(r.passes, k)
		}
		if r.claim < 0 && w.walkClaimOf(1).Arrived {
			r.claim = k
			c := w.walkClaimOf(1)
			if c.Claim != packedCell(2, 4) || c.Target != packedCell(20, 4) {
				t.Fatalf("step %d: claim %#x target %#x; want the walker's cell (2,4) for (20,4)", k, c.Claim, c.Target)
			}
		}
		if r.claim >= 0 && r.latch < 0 && latched(o) {
			r.latch = k
		}
		if r.latch >= 0 && k > r.latch && e.HasTarget {
			r.walks++
		}
		if pass && r.claim >= 0 && k > r.claim && (e.HasTarget || o.Raw[8] == 1) {
			r.resent++
		}
		if arrived := w.walkClaimOf(1).Arrived; arrived && !wasArrived {
			r.claims++
		}
		wasArrived = w.walkClaimOf(1).Arrived
		if e.HasAttackTarget && !victim {
			r.picks++
		}
		victim = e.HasAttackTarget
		if between != nil {
			between(w)
			wasArrived = w.walkClaimOf(1).Arrived
		}
	}
	return r
}

// nextPass is the first Group pass after step k, or -1.
func (r claimRun) nextPass(k int) int {
	for _, p := range r.passes {
		if p > k {
			return p
		}
	}
	return -1
}

func dropArrival(w *World) {
	if c := w.walkClaimOf(1); c.Arrived {
		c.Arrived = false
		w.setWalkClaim(1, c)
	}
}

// A Move member refused by the static search is stood down at once with its
// cell claimed and the arrival flag raised, is latched by the next Group pass
// and not before, and then only acquires: it walks no more and strikes the
// hostile within reach (AI-682, AI-683). Loss control: with the flag lowered
// after every step the member is never latched and is sent again.
func TestAStaticRefusalLatchesOnTheNextGroupPass(t *testing.T) {
	t.Parallel()
	w := pocketGroup(t, 30, orderMove, cell{4, 4}, engSight, 3)
	r := runClaim(t, w, 4, nil)
	if r.claim < 0 || r.latch != r.nextPass(r.claim) || r.walks != 0 {
		t.Fatalf("claim step %d, latch step %d (next pass %d), walked %d steps after it; want the latch on the next pass, not before, and no walk",
			r.claim, r.latch, r.nextPass(r.claim), r.walks)
	}
	if c := w.walkClaimOf(1); c.Arrived || c.Claim != packedCell(2, 4) {
		t.Errorf("after the latch the claim is %#x arrived %v; want (2,4) kept and the flag lowered", c.Claim, c.Arrived)
	}
	if r.picks == 0 || laEnt(t, w, 3).HP >= 2000 {
		t.Errorf("picks %d, hostile health %d; want the latched member to strike the hostile within reach", r.picks, laEnt(t, w, 3).HP)
	}
	loss := runClaim(t, pocketGroup(t, 30, orderMove, cell{4, 4}, engSight, 3), 4, dropArrival)
	if loss.latch >= 0 || loss.claims < 2 {
		t.Errorf("loss control: latch step %d, %d refusals; want no latch and the walk sent and refused again", loss.latch, loss.claims)
	}
	t.Logf("claim step %d, latch step %d, passes %v; loss control %d refusals", r.claim, r.latch, r.passes[:3], loss.claims)
}

// The claim is the stand-down's state at the refusal: the order holds state
// 0xc, its post on the cell, pending kind 0 and no latch, and the member
// holds no victim (AI-682).
func TestAStaticRefusalStandsDownWithoutALatch(t *testing.T) {
	t.Parallel()
	w := pocketGroup(t, 30, orderMove, cell{4, 4}, engSight, 3)
	for k := 0; k < 2*scriptCycle; k++ {
		Step(w, nil)
		if !w.walkClaimOf(1).Arrived {
			continue
		}
		e, o := laEnt(t, w, 1), w.savedOrder(1)
		if o.State != uint32(actorStateAcquire) || !o.stoodDown || o.Raw[8] != 0 || latched(o) || e.HasAttackTarget || e.HasTarget || o.Raw[0] != 2 || o.Raw[1] != 4 {
			t.Fatalf("step %d: state %#x stood down %v kind %d latch %v victim %v walk %v post (%d,%d); want 0xc, kind 0, no latch, no victim, no walk, post (2,4)",
				k, o.State, o.stoodDown, o.Raw[8], latched(o), e.HasAttackTarget, e.HasTarget, o.Raw[0], o.Raw[1])
		}
		return
	}
	t.Fatal("the refusal never claimed the cell")
}

// A member refused by the near search keeps no claim: every order 4 pass
// sends it again and none latches it (AI-684). Loss control: a claim raised
// on its cell after every step latches it and the passes stop sending it.
func TestADynamicRefusalIsSentAgainOnEachGroupPass(t *testing.T) {
	t.Parallel()
	build := func() *World {
		w := jammedCorridor(t, 1, 25)
		wakeGroups(w)
		i := indexOfEntity(w.entities, 1)
		w.commandGroup([]int{i}, orderMove, cell{x: 20, y: 4})
		w.walkAtStateOne([]int{i})
		w.issueGroupDestination([]int{i}, cell{x: 20, y: 4})
		return w
	}
	run := func(w *World, between func(*World)) (stood, resent, passes int, latch bool) {
		stood = -1
		for k := 0; k < 8*scriptCycle; k++ {
			Step(w, nil)
			e, o := laEnt(t, w, 1), w.savedOrder(1)
			if stood < 0 && o.stoodDown {
				stood = k
			}
			if w.walkClaimOf(1).Arrived && between == nil {
				t.Fatalf("step %d: a near search's refusal raised the arrival flag", k)
			}
			latch = latch || latched(o)
			if w.ScriptPassJustRan() && stood >= 0 && k > stood {
				passes++
				if e.HasTarget && o.Raw[8] == 1 {
					resent++
				}
			}
			if between != nil {
				between(w)
			}
		}
		return
	}
	stood, resent, passes, latch := run(build(), nil)
	if stood < 0 || passes < 3 || resent != passes || latch {
		t.Fatalf("stand-down step %d, sent again on %d of %d passes, latch %v; want every pass to send it and no latch", stood, resent, passes, latch)
	}
	_, lossResent, lossPasses, lossLatch := run(build(), func(w *World) {
		e := laEnt(t, w, 1)
		if !e.HasTarget {
			w.setWalkClaim(1, walkClaim{Target: packedCell(20, 4), Claim: packedCell(e.X, e.Y), Arrived: true})
		}
	})
	if !lossLatch || lossResent == lossPasses {
		t.Errorf("loss control: latch %v, sent again on %d of %d passes; want a latch that stops the passes", lossLatch, lossResent, lossPasses)
	}
	t.Logf("stand-down step %d, sent again on %d of %d passes", stood, resent, passes)
}

// An order 2 member's walk refused by the static search takes the tail's
// pick at state 0xc; each later pass re-issues the walk, which ends on the
// claimed cell with no search, so no refusal and no further pick follows
// and nothing latches (AI-685). The hostile stands within reach but out of
// the Group's sight. Loss control: with the arrival flag lowered after every
// step each re-issued walk is searched and refused again.
func TestAnOrderTwoMemberIsSentAgainWithNoSearch(t *testing.T) {
	t.Parallel()
	r := runClaim(t, pocketGroup(t, 30, orderSwarm, cell{4, 4}, 1, 3), 6, nil)
	if r.claim < 0 || r.claims != 1 || r.picks != 1 || r.latch >= 0 || r.resent < 3 {
		t.Fatalf("claim step %d, %d refusals, picks %d, latch step %d, re-issued on %d passes; want one refusal, one pick, no latch and the walk re-issued",
			r.claim, r.claims, r.picks, r.latch, r.resent)
	}
	loss := runClaim(t, pocketGroup(t, 30, orderSwarm, cell{4, 4}, 1, 3), 6, dropArrival)
	if loss.claims <= 1 {
		t.Errorf("loss control: %d refusals; want each re-issued walk searched and refused", loss.claims)
	}
	t.Logf("claim step %d, one pick, re-issued on %d passes; loss control %d refusals", r.claim, r.resent, loss.claims)
}

// A Roam member whose walk the static search refuses is latched by the next
// Roam pass through the order 4 arm and then only acquires (AI-685, AI-662).
// Loss control: with the arrival flag lowered after every step it is never
// latched.
func TestARoamMemberLatchesAfterAStaticRefusal(t *testing.T) {
	t.Parallel()
	r := runClaim(t, pocketGroup(t, 60, orderRoam, cell{40, 40}, engSight, 1), 4, nil)
	if r.claim < 0 || r.latch != r.nextPass(r.claim) || r.walks != 0 {
		t.Fatalf("claim step %d, latch step %d (next pass %d), walked %d steps after it; want the latch on the next pass and no walk", r.claim, r.latch, r.nextPass(r.claim), r.walks)
	}
	loss := runClaim(t, pocketGroup(t, 60, orderRoam, cell{40, 40}, engSight, 1), 4, dropArrival)
	if loss.latch >= 0 {
		t.Errorf("loss control: latch step %d; want none", loss.latch)
	}
	t.Logf("claim step %d, latch step %d", r.claim, r.latch)
}

// The claim between the refusal and the latch survives the World's byte form
// and the Mover bytes a SAVE writes and a LOAD adopts (ApplyMover,
// adoptMover): each twin restored at the claim step latches on the same pass
// as the World. Loss control: a twin adopting Mover bytes whose arrival flag
// is cleared is never latched.
func TestTheClaimSurvivesTheByteFormAndTheMoverBytes(t *testing.T) {
	t.Parallel()
	want := runClaim(t, pocketGroup(t, 30, orderMove, cell{4, 4}, engSight, 3), 2, nil)
	restore := func(how string) claimRun {
		w := pocketGroup(t, 30, orderMove, cell{4, 4}, engSight, 3)
		for k := 0; k <= want.claim; k++ {
			Step(w, nil)
		}
		switch how {
		case "form":
			b, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			twin := &World{}
			if err := twin.UnmarshalBinary(b); err != nil {
				t.Fatal(err)
			}
			if twin.Hash() != w.Hash() || twin.walkClaimOf(1) != w.walkClaimOf(1) {
				t.Fatalf("the byte form changed the World or the claim: %+v, want %+v", twin.walkClaimOf(1), w.walkClaimOf(1))
			}
			w = twin
		case "mover", "loss":
			var m [180]byte
			if !w.ApplyMover(1, &m) || readWalkClaim(&m) != w.walkClaimOf(1) {
				t.Fatalf("the Mover bytes hold claim %+v, want %+v", readWalkClaim(&m), w.walkClaimOf(1))
			}
			if how == "loss" {
				m[0x90] = 0
			}
			w.setWalkClaim(1, walkClaim{})
			w.adoptMover(laEnt(t, w, 1), &m)
		}
		r := runClaim(t, w, 2, nil)
		r.latch += want.claim + 1
		return r
	}
	for _, how := range []string{"form", "mover"} {
		if got := restore(how); got.latch != want.latch {
			t.Errorf("%s: latch step %d, want %d as without the restore", how, got.latch, want.latch)
		}
	}
	if got := restore("loss"); got.latch == want.latch {
		t.Errorf("loss control: the twin without the arrival flag latched on step %d", got.latch)
	}
}

// A Move member walking to a sack holds the pick-up, which this build keeps
// inside the Group: its static refusal claims no cell and the Group pass
// latches nothing, so the pick-up keeps its own state. Loss control: the
// same refusal without the pick-up claims the cell and the pass latches it.
func TestAPickUpWalkClaimsNoCell(t *testing.T) {
	t.Parallel()
	for _, pickup := range []bool{true, false} {
		w := pocketGroup(t, 30, orderMove, cell{4, 4}, engSight, 3)
		i := indexOfEntity(w.entities, 1)
		if pickup {
			w.entities[i].PendingOrder = PendingOrder{Kind: PendingPickup, X: 20, Y: 4}
		}
		w.answerRefusedWalk(i, searchStatic)
		o := w.savedOrder(1)
		claimed, latch := w.walkClaimOf(1).Arrived, w.claimLatch(i, o)
		if claimed == pickup || latch == pickup {
			t.Errorf("pick-up %v: claimed %v, latched %v; want both %v", pickup, claimed, latch, !pickup)
		}
	}
}
