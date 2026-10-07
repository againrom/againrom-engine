package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// flightSave reads one save of the owner's projectile corpus.
func flightSave(t *testing.T, name string) []byte {
	t.Helper()
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: projectile flight witness requires owner saves")
	}
	raw, err := os.ReadFile(filepath.Join(corpus, "2026-10-02", "projectiles-original-en", name))
	if err != nil {
		t.Skip("corpus file missing:", err)
	}
	return raw
}

// recordSet is a Projectiles store as the leaves each record carries, keyed by
// id, with the allocator and the id list order. The target's identity is a
// per-World number and is left out.
type recordSet struct {
	free   uint16
	ids    []uint16
	leaves map[uint16][16]int32
}

func recordSetOfStore(s sav.ProjectileStore) recordSet {
	r := recordSet{free: s.FreeIndex, ids: slices.Clone(s.IDs), leaves: map[uint16][16]int32{}}
	for _, p := range s.Items {
		r.leaves[p.ID] = shotLeaves(savedProjectileFromSav(p))
	}
	return r
}

func recordSetOfWorld(w *sim.World) recordSet {
	s := w.SavedProjectiles()
	r := recordSet{free: s.FreeIndex, ids: slices.Clone(s.IDs), leaves: map[uint16][16]int32{}}
	for _, p := range s.Items {
		r.leaves[p.ID] = shotLeaves(p)
	}
	return r
}

func (r recordSet) diff(o recordSet) error {
	if r.free != o.free || !slices.Equal(r.ids, o.ids) || len(r.leaves) != len(o.leaves) {
		return fmt.Errorf("allocator %d ids %v records %d, want allocator %d ids %v records %d",
			r.free, r.ids, len(r.leaves), o.free, o.ids, len(o.leaves))
	}
	for id, l := range r.leaves {
		if o.leaves[id] != l {
			return fmt.Errorf("record %d leaves %v, want %v", id, l, o.leaves[id])
		}
	}
	return nil
}

// flightTick is the World tick a corpus save was written on.
func flightTick(t *testing.T, raw []byte) uint64 {
	t.Helper()
	g := releaseFront(t)
	app, _ := openOriginalSAVApp(t, g, raw, "tick.sav")
	t.Cleanup(app.StopAudio)
	return g.live.world.Tick()
}

// flightRun loads raw and ticks the live world to tick want.
func flightRun(t *testing.T, raw []byte, want uint64) *mapWorld {
	t.Helper()
	g := releaseFront(t)
	app, _ := openOriginalSAVApp(t, g, raw, "flight.sav")
	t.Cleanup(app.StopAudio)
	mw := g.live
	for mw.world.Tick() < want {
		mw.tick()
	}
	return mw
}

// lateShots lists the records of r that the load-point set l does not hold and
// whose picture is a unit shot (1 to 12): shots the loaded World released
// itself.
func lateShots(r, l recordSet) []uint16 {
	var ids []uint16
	for id, leaves := range r.leaves {
		if _, held := l.leaves[id]; !held && leaves[3] >= 1 && leaves[3] <= 12 {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

// A loaded original save continues on the tick the original wrote its later
// saves: the live World's record set at each later save's tick equals that
// save's Projectiles leaf for leaf, shots the loaded World releases itself
// included. game0022 holds a rock, game0023 a rock and the burst, game0024 the
// burst and a bolt. One tick before or after, the set differs.
func TestReleaseOriginalProjectileFlightContinuesToLaterSaves(t *testing.T) {
	saves := map[string][]byte{}
	ticks := map[string]uint64{}
	for _, n := range []string{"game0022.sav", "game0023.sav", "game0024.sav"} {
		saves[n] = flightSave(t, n)
		ticks[n] = flightTick(t, saves[n])
	}
	for _, c := range [][2]string{
		{"game0022.sav", "game0023.sav"}, {"game0022.sav", "game0024.sav"}, {"game0023.sav", "game0024.sav"},
	} {
		want := recordSetOfStore(kitProjectileStore(t, saves[c[1]]))
		at := recordSetOfStore(kitProjectileStore(t, saves[c[0]]))
		late := lateShots(want, at)
		if len(late) == 0 {
			t.Fatalf("%s to %s: no shot was released in between", c[0], c[1])
		}
		got := recordSetOfWorld(flightRun(t, saves[c[0]], ticks[c[1]]).world)
		if err := got.diff(want); err != nil {
			t.Errorf("%s to tick %d (%s): %v", c[0], ticks[c[1]], c[1], err)
		}
		// Loss controls: a tick off, and a changed leaf, fail.
		for _, off := range []int{-1, 1} {
			other := recordSetOfWorld(flightRun(t, saves[c[0]], uint64(int(ticks[c[1]])+off)).world)
			if other.diff(want) == nil {
				t.Errorf("%s: the records %+d tick from %s passed the comparison", c[0], off, c[1])
			}
			for _, id := range late {
				if off == 1 && other.leaves[id] == want.leaves[id] {
					t.Errorf("%s: shot %d one tick after %s still equals the original", c[0], id, c[1])
				}
			}
		}
		changed := recordSet{free: want.free, ids: want.ids, leaves: map[uint16][16]int32{}}
		for id, l := range want.leaves {
			l[14]++
			changed.leaves[id] = l
		}
		if got.diff(changed) == nil {
			t.Errorf("%s: a changed leaf passed the comparison", c[1])
		}
	}
}

// The catapult's rider, run by the World loaded from game0022, hurts its target
// and queues the transport on tick 3098 and the burst exists on 3102 (SAV-1145;
// ANIM-115 gives the order). Between them the transport is the one pending
// delivery. A transport queued or a burst built a tick off moves a reading.
func TestReleaseSiegeRiderQueuesItsTransportWithTheBlowBeforeTheBurst(t *testing.T) {
	const target, hitTick, burstTick = 185, 3098, 3102
	g := releaseFront(t)
	app, _ := openOriginalSAVApp(t, g, flightSave(t, "game0022.sav"), "rider.sav")
	t.Cleanup(app.StopAudio)
	mw := g.live
	type reading struct {
		hp      int32
		pending int
		bursts  int
	}
	read := func() reading {
		e, _ := mw.entity(target)
		r := reading{hp: e.HP, pending: mw.world.PendingSpellDeliveries()}
		for _, p := range mw.world.SavedProjectiles().Items {
			if p.Picture == fireBallBurstPicture {
				r.bursts++
			}
		}
		return r
	}
	at := map[uint64]reading{}
	for mw.world.Tick() <= burstTick+1 {
		at[mw.world.Tick()] = read()
		mw.tick()
	}
	before, hit := at[hitTick-1], at[hitTick]
	if hit.hp >= before.hp || before.pending != 0 || hit.pending != 1 || hit.bursts != 0 {
		t.Errorf("tick %d: the rider's blow left %+v after %+v, want lower hit points and one pending delivery", hitTick, hit, before)
	}
	for tick := uint64(hitTick); tick < burstTick; tick++ {
		if r := at[tick]; r.pending != 1 || r.bursts != 0 || r.hp != hit.hp {
			t.Errorf("tick %d: %+v, want the transport pending, no burst and the hit points of the blow %d", tick, r, hit.hp)
		}
	}
	if r := at[burstTick]; r.pending != 0 || r.bursts != 1 || r.hp >= hit.hp {
		t.Errorf("tick %d: %+v, want the transport spent and the burst built with the blast", burstTick, r)
	}
}
