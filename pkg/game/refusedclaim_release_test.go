package game

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

// claimRow is one human member's walk as a tick leaves it.
type claimRow struct {
	x, y          int32
	state, latch  uint32
	pending       uint8
	walk, victim  bool
	arrived       bool
	claim, target uint16
}

func (r claimRow) String() string {
	return fmt.Sprintf("(%d,%d) state %#x kind %d latch %d walk %v victim %v claim %#x for %#x arrived %v",
		r.x, r.y, r.state, r.pending, r.latch, r.walk, r.victim, r.claim, r.target, r.arrived)
}

func claimRowOf(t *testing.T, w *sim.World, id sim.EntityID) claimRow {
	t.Helper()
	e, ok := w.Entity(id)
	if !ok {
		t.Fatalf("the World holds no actor %d", id)
	}
	r := claimRow{x: e.X, y: e.Y, walk: e.HasTarget, victim: e.HasAttackTarget}
	_, orders, _ := w.SavedGroups()
	for _, o := range orders {
		if o.Entity == id {
			r.state, r.pending, r.latch = o.State, o.Raw[8], binary.LittleEndian.Uint32(o.Raw[0x50:])
		}
	}
	motions, _, _, _ := w.SavedActorMotions()
	for _, m := range motions {
		if m.Entity == id {
			r.target, r.claim = binary.LittleEndian.Uint16(m.Mover[0x74:]), binary.LittleEndian.Uint16(m.Mover[0x76:])
			r.arrived = binary.LittleEndian.Uint32(m.Mover[0x90:]) != 0
		}
	}
	return r
}

// The two human members of game0013 (mission 10) ordered as one Group to
// (44,32), the cell AI-682's run sent them to. The far search settles each
// on a cell short of it, where the Move latches it on arrival. The same
// order again finds each member on the cell its search settles on: the
// static search's refusal. A refused member stands down at once with its
// cell claimed, is latched on the next Group pass and walks no more in the
// 300 ticks recorded (AI-682, AI-683). A SAVE and cold LOAD between the
// refusal and the latch restore the same World, and the loaded twin keeps
// step with it. Each member's changes are logged.
func TestReleaseARefusedGroupMoveLatchesOnTheNextGroupPass(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-14/game0013.sav", "b211b9ad621a2cec38ff5a1d1f3f632542a58e1e377c8f562b25ba48ca2aa5ea")
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	opener, town, err := f.RestoreOriginal(raw)
	if err != nil || town || opener == nil {
		t.Fatal("LOAD", err, town)
	}
	app := f.App("refused Group Move")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	live := f.live
	var members []sim.EntityID
	for _, e := range live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.Alive() && !e.OffMap {
			members = append(members, e.ID)
		}
	}
	if len(members) != 2 {
		t.Fatalf("game0013 holds %d living human members, want Naira and her ally", len(members))
	}
	t.Logf("members %v, the party's character %d", members, live.mission.ids[0])
	order := func() {
		for _, id := range members {
			live.enqueue(uint32(id), 44, 32)
		}
	}
	order()
	arrived := 0
	for n := 1; n <= 600 && arrived < len(members); n++ {
		live.tick()
		arrived = 0
		for _, id := range members {
			if r := claimRowOf(t, live.world, id); r.latch == 1 && !r.walk {
				arrived++
			}
		}
		if arrived == len(members) {
			for _, id := range members {
				t.Logf("first order: member %d latched at tick %d on %v", id, n, claimRowOf(t, live.world, id))
			}
		}
	}
	if arrived != len(members) {
		t.Fatal("the first order did not latch both members on arrival")
	}
	order()
	a := &unreachableArena{front: f, live: live}
	rows := map[sim.EntityID]claimRow{}
	claimed, latched := map[sim.EntityID]int{}, map[sim.EntityID]int{}
	var passes []int
	var cold *FrontEnd
	var log strings.Builder
	for n := 1; n <= 300; n++ {
		live.tick()
		if live.world.ScriptPassJustRan() {
			passes = append(passes, n)
		}
		reload := false
		for _, id := range members {
			r := claimRowOf(t, live.world, id)
			if r != rows[id] {
				fmt.Fprintf(&log, "tick %d member %d: %v\n", n, id, r)
			}
			rows[id] = r
			if _, ok := claimed[id]; !ok && r.arrived && r.state == 0xc && r.latch == 0 {
				claimed[id] = n
				reload = reload || cold == nil
			}
			if _, ok := latched[id]; !ok && claimed[id] > 0 && r.latch == 1 {
				latched[id] = n
			}
			if latched[id] > 0 && n > latched[id] && r.walk {
				t.Errorf("tick %d: member %d walks after its latch at tick %d", n, id, latched[id])
			}
		}
		if cold != nil {
			cold.live.tick()
			assertEntitiesEqual(t, live.world, cold.live.world, members, "successor of the claimed member")
		}
		if reload {
			cold = a.reloadCold(t)
			assertCurrentWorldEqual(t, live.world, cold.live.world, "LOAD of the claimed member")
		}
	}
	t.Logf("Group passes %v\n%s", passes[:4], log.String())
	if len(claimed) == 0 {
		t.Fatal("no member's walk was refused by the static search")
	}
	for id, k := range claimed {
		next := -1
		for _, p := range passes {
			if p > k {
				next = p
				break
			}
		}
		if latched[id] != next {
			t.Errorf("member %d: claimed at tick %d, latched at tick %d; want the next Group pass, tick %d", id, k, latched[id], next)
		}
	}
	t.Logf("claimed %v latched %v", claimed, latched)
}
