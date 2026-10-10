package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func pickupWorld1117(t *testing.T, distance int32) *World {
	t.Helper()
	hero := laFighter(0, 1, 17, 20, 20)
	hero.HP, hero.MaxHP, hero.ScanRange = 1000, 1000, 12
	enemy := laFighter(9, 2, 19, 20+distance, 20)
	enemy.HP, enemy.MaxHP = 1000, 1000
	w, err := NewLootWorld(1117, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{},
		[]Entity{hero, enemy}, nil, engRel(t, [3]uint32{1, 2, 1}, [3]uint32{2, 1, 2}),
		[]Sack{{X: 20, Y: 20, Gold: 17, Items: []uint16{0x101, 0x101}}})
	if err != nil {
		t.Fatal(err)
	}
	w.commandGroup([]int{0}, orderMove, cell{x: 20, y: 20})
	w.entities[0].TargetX, w.entities[0].TargetY, w.entities[0].HasTarget = 20, 20, true
	return w
}

func pickupTransfer1117(t *testing.T, w *World) {
	t.Helper()
	before := w.entities[0]
	if err := w.TakeSack(0, before.X, before.Y); err != nil {
		t.Fatal(err)
	}
	// The transfer primitive still does not own action state.
	if w.entities[0] != before {
		t.Fatal("TakeSack changed actor order/state")
	}
	if !w.CompleteSackPickup(0) {
		t.Fatal("gameplay completion refused live actor zero")
	}
	if e := w.entities[0]; e.ActorState != 2 || e.HasTarget || e.HasAttackTarget || e.GroupSpeed != 0 || len(w.routes[0]) != 0 {
		t.Fatalf("transfer retained stale order: %+v", e)
	}
	if order, _, ok := w.groupState(1, effectiveGroup(&w.entities[0])); !ok || order != 0 {
		t.Fatalf("pickup retained group order %d/%v", order, ok)
	}
}

func pickupRoundTrip1117(t *testing.T, w *World) *World {
	t.Helper()
	raw := mustMarshal(t, w)
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, mustMarshal(t, &back)) || w.Hash() != back.Hash() {
		t.Fatal("native round trip changed bytes/hash")
	}
	return &back
}

func TestPickup1117EveryPhaseCompletesBeforeNextAcquirePass(t *testing.T) {
	for phase := uint64(0); phase < 16; phase++ {
		for _, distance := range []int32{1, 2, 8} {
			t.Run(fmt.Sprintf("phase%d-distance%d", phase, distance), func(t *testing.T) {
				w := pickupWorld1117(t, distance)
				w.tick = phase
				// A remote guard post cannot leash acquire to the old location.
				w.entities[0].PostX, w.entities[0].PostY = 1, 1
				pickupTransfer1117(t, w)
				back := pickupRoundTrip1117(t, w)
				originalTransit := w.entities[0].Transit
				seenAttack := false
				for successor := 1; successor <= 34; successor++ {
					preTick := w.Tick()
					Step(w, nil)
					Step(back, nil)
					if w.Hash() != back.Hash() || !bytes.Equal(mustMarshal(t, w), mustMarshal(t, back)) {
						t.Fatalf("successor %d differs", successor)
					}
					e := w.entities[0]
					if successor == 1 {
						if e.ActorState != 0x0c || e.HasAttackTarget || e.HasTarget || e.Transit != originalTransit {
							t.Fatalf("completion step ran a target decision or lost state: %+v", e)
						}
						back = pickupRoundTrip1117(t, w)
					}
					if e.HasAttackTarget && !seenAttack {
						if preTick%16 != 6 || successor == 1 || distance != 1 || e.AttackTarget != 9 {
							t.Fatalf("first acquire at wrong tick/target: phase%d successor%d distance%d actor%+v", preTick, successor, distance, e)
						}
						seenAttack = true
					}
					if distance > 1 && (e.HasAttackTarget || e.HasTarget || e.X != 20 || e.Y != 20) {
						t.Fatalf("standing acquire chased beyond reach: %+v", e)
					}
				}
				if seenAttack != (distance == 1) {
					t.Fatalf("acquisition observed=%v for distance%d", seenAttack, distance)
				}
			})
		}
	}
}

func TestPickup1117ReplacementOrdersWinBeforeCompletion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cmd   Command
		state uint8
	}{
		{"move", Command{Kind: KindMoveTo, Entity: 0, X: 25, Y: 20}, actorStateGuard},
		{"group-move", Command{Kind: KindGroupMoveTo, Entity: 0, X: 25, Y: 20}, actorStateGuard},
		{"stop", Command{Kind: KindGroupStance, Entity: 0, X: int32(OrderGuard)}, actorStateGuard},
		{"stand", Command{Kind: KindGroupStance, Entity: 0, X: int32(OrderStandGround)}, actorStateGuard},
		{"patrol", Command{Kind: KindGroupPatrolTo, Entity: 0, X: 25, Y: 20}, actorStatePatrol},
		{"retreat", Command{Kind: KindGroupRetreat, Entity: 0, Player: 1}, actorStateRetreat},
		{"defend", Command{Kind: KindGroupDefend, Entity: 0, X: 9}, actorStateDefend},
		// 1141: a player attack order now carries the law's own engage state
		// (`AI-CMD-054`, `actor+0x50 = 3` at `L00014`) instead of leaving the
		// unit at guard. The replacement still wins over completion, which is
		// what this case asserts; the state it lands in is the corrected one.
		{"attack", Command{Kind: KindAttack, Entity: 0, X: 9}, actorStateEngage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := pickupWorld1117(t, 2)
			pickupTransfer1117(t, w)
			back := pickupRoundTrip1117(t, w)
			Step(w, []Command{tc.cmd})
			Step(back, []Command{tc.cmd})
			if w.entities[0].ActorState != tc.state || w.Hash() != back.Hash() {
				t.Fatalf("replacement lost to completion: state=%d want%d", w.entities[0].ActorState, tc.state)
			}
			for n := 0; n < 33; n++ {
				pickupRoundTrip1117(t, w)
				Step(w, nil)
				Step(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatalf("successor %d differs", n)
				}
			}
		})
	}
}

func TestPickup1117BookAndScrollReplacePendingCompletion(t *testing.T) {
	for _, kind := range []uint8{KindCast, KindCastAt, KindUseScroll, KindUseScrollAt} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			w := scrollWorld1090(t, 2, 2)
			atCell := kind == KindCastAt || kind == KindUseScrollAt
			if atCell {
				w.spells[0].TargetsUnit = false
				w.spells[0].Area = true
			}
			w.entities[0].Mana, w.entities[0].MaxMana = 1000, 1000
			w.entities[0].ScanRange = 8
			w.entities[0].KnownSpells = 1 << 1
			w.sacks = []Sack{{X: 1, Y: 1, Gold: 3}}
			if err := w.TakeSack(1, 1, 1); err != nil {
				t.Fatal(err)
			}
			if !w.CompleteSackPickup(1) {
				t.Fatal("marker refused")
			}
			back := pickupRoundTrip1117(t, w)
			cmd := Command{Kind: kind, Entity: 1, X: 2, Y: 1}
			if kind == KindCastAt {
				cmd.Spell = 1
			}
			Step(w, []Command{cmd})
			Step(back, []Command{cmd})
			want := actorStateGuard
			if kind == KindCast || kind == KindCastAt {
				want = actorStateAcquire
			}
			if w.entities[0].ActorState != want || !w.actorCastBusy(0) {
				t.Fatalf("admitted cast did not replace completion: state%d casts%v scrolls%v", w.entities[0].ActorState, w.bookCasts, w.scrollCasts)
			}
			for n := 0; n < 33; n++ {
				pickupRoundTrip1117(t, w)
				if w.Hash() != back.Hash() {
					t.Fatalf("successor%d differs", n)
				}
				Step(w, nil)
				Step(back, nil)
			}
		})
	}
}

func TestPickup1117DeathOffMapAndAbsentActor(t *testing.T) {
	w := pickupWorld1117(t, 8)
	before := mustMarshal(t, w)
	if w.CompleteSackPickup(999) || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("absent actor changed world")
	}
	pickupTransfer1117(t, w)
	w.takeOffMap(0)
	back := pickupRoundTrip1117(t, w)
	for n := 0; n < 33; n++ {
		Step(w, nil)
		Step(back, nil)
		if w.entities[0].ActorState != 2 || w.Hash() != back.Hash() {
			t.Fatal("off-map pending order advanced or diverged")
		}
	}
	before = mustMarshal(t, w)
	if w.CompleteSackPickup(0) || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("off-map admission changed world")
	}
	if !w.returnToMap(0) || !back.returnToMap(0) {
		t.Fatal("return failed")
	}
	Step(w, nil)
	Step(back, nil)
	if w.entities[0].ActorState != 0x0c || w.Hash() != back.Hash() {
		t.Fatal("returned actor did not complete")
	}
	w = pickupWorld1117(t, 8)
	pickupTransfer1117(t, w)
	Step(w, []Command{{Kind: KindKill, Entity: 0}})
	if w.entities[0].ActorState != actorStateGuard || w.entities[0].Alive() {
		t.Fatal("death retained pickup state")
	}
	before = mustMarshal(t, w)
	if w.CompleteSackPickup(0) || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("dead admission changed world")
	}
	pickupRoundTrip1117(t, w)
}

func TestPickup1117RejectsMalformedPendingWithoutChangingReceiver(t *testing.T) {
	w := pickupWorld1117(t, 8)
	pickupTransfer1117(t, w)
	good := mustMarshal(t, w)
	// Header and the three one-byte cell planes precede unchanged entity+100.
	o := headerLen + 3*64*64
	if good[o+100] != 2 {
		t.Fatal("fixture does not address pending state byte")
	}
	for _, tc := range []struct {
		name string
		edit func([]byte)
	}{
		// 7 rather than 3: `AI-STATE-043` measures 3 as a WRITTEN state and
		// 1141 gave this build its constant, so 3 decodes. 7 is one of the
		// bytes that claim finds unwritten and on the switch's default arm.
		{"unknown-state", func(b []byte) { b[o+100] = 7 }},
		{"movement", func(b []byte) {
			binary.LittleEndian.PutUint32(b[o+12:], 21)
			binary.LittleEndian.PutUint32(b[o+16:], 20)
			b[o+24] = 1
		}},
		{"attack", func(b []byte) { binary.LittleEndian.PutUint32(b[o+44:], 9); b[o+48] = 1 }},
		{"group-speed", func(b []byte) { b[o+43] = 1 }},
		{"patrol", func(b []byte) { binary.LittleEndian.PutUint32(b[o+101:], 1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Clone(good)
			tc.edit(bad)
			if err := w.UnmarshalBinary(bad); err == nil {
				t.Fatal("accepted malformed pending state")
			}
			if !bytes.Equal(good, mustMarshal(t, w)) {
				t.Fatal("failed decode changed receiver")
			}
		})
	}
	for _, order := range []uint8{orderGuard, orderMove, orderSwarm2} {
		bad := pickupRoundTrip1117(t, w)
		for i := range bad.groups {
			if bad.groups[i].owner == 1 && bad.groups[i].group == effectiveGroup(&bad.entities[0]) {
				bad.groups[i].order = order
			}
		}
		if err := w.UnmarshalBinary(mustMarshal(t, bad)); err == nil || !strings.Contains(err.Error(), "pickup completion") {
			t.Fatalf("group order%d: wanted pending-group refusal, got %v", order, err)
		}
		if !bytes.Equal(good, mustMarshal(t, w)) {
			t.Fatal("bad group changed receiver")
		}
	}
	dead := pickupRoundTrip1117(t, w)
	Step(dead, []Command{{Kind: KindKill, Entity: 0}})
	deadForm := mustMarshal(t, dead)
	deadForm[o+100] = 2
	if err := w.UnmarshalBinary(deadForm); err == nil {
		t.Fatal("accepted pending pickup on dead actor")
	}
	if !bytes.Equal(good, mustMarshal(t, w)) {
		t.Fatal("bad death state changed receiver")
	}
}

func TestPickup1117PreservesCommittedCrossingAndScriptReplacement(t *testing.T) {
	w := pickupWorld1117(t, 8)
	w.entities[0].Transit, w.entities[0].TransitTotal = 7, 9
	pickupTransfer1117(t, w)
	if w.entities[0].Transit != 7 || w.entities[0].TransitTotal != 9 {
		t.Fatal("completion marker repurposed crossing")
	}
	back := pickupRoundTrip1117(t, w)
	Step(w, nil)
	Step(back, nil)
	if w.entities[0].Transit != 6 || w.entities[0].TransitTotal != 9 || w.entities[0].ActorState != 0xc || w.Hash() != back.Hash() {
		t.Fatal("completion changed real crossing duration")
	}
	w = pickupWorld1117(t, 8)
	pickupTransfer1117(t, w)
	w.cmdGroupCommandedMove(effectiveGroup(&w.entities[0]), orderMove, 25, 20)
	Step(w, nil)
	if w.entities[0].ActorState != actorStateGuard {
		t.Fatal("script replacement did not cancel pending completion")
	}
	pickupRoundTrip1117(t, w)
}
