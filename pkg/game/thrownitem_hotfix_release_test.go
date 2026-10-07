package game

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func thrownSackItems(w *sim.World) map[string][]uint16 {
	out := map[string][]uint16{}
	for _, s := range w.Sacks() {
		items := append([]uint16(nil), s.Items...)
		sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
		out[fmt.Sprintf("%d,%d", s.X, s.Y)] = items
	}
	return out
}

func thrownHeld(w *sim.World, hero sim.EntityID) int {
	n := 0
	pack, _ := w.CarriedStacks(hero)
	for _, st := range pack {
		n += int(st.Count)
	}
	worn, _ := w.EquippedItems(hero)
	for _, v := range worn {
		if !v.Empty() {
			n++
		}
	}
	for _, s := range w.Sacks() {
		n += len(s.Items)
	}
	return n
}

func thrownLivingAt(w *sim.World, x, y int32) bool {
	for _, e := range w.Entities() {
		if e.Alive() && !e.OffMap && e.X == x && e.Y == y {
			return true
		}
	}
	return false
}

// A thrown item lies in a sack the player can see. A sack on the thrower's own
// cell is drawn beneath the thrower, so a throw beyond the 5x5 window, onto the
// thrower, or onto another unit used to leave nothing visible. The sack now lies
// on a free neighbour of the thrower; SAVE and a cold LOAD keep it, and the
// ordinary pickup order returns the items.
func TestReleaseThrownItemLiesBesideTheThrowerAndSurvivesSaveLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("thrown item")
	if err := app.OpenMission(f.NewGameOpener(20, ui.ChargenResult{Name: "Thrown item", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	mw := f.live
	hero := mw.mission.ids[0]
	at, _ := mw.entity(hero)
	total := thrownHeld(mw.world, hero)
	if total < 3 {
		t.Fatalf("the hero holds %d items, want at least 3 to throw", total)
	}
	throw := func(label string, worn bool, index int, x, y int32) {
		t.Helper()
		before := len(mw.world.Sacks())
		mw.enqueueDrop(worn, index, x, y)
		mw.tick()
		if got := thrownHeld(mw.world, hero); got != total {
			t.Fatalf("%s: items held or on the ground %d, want %d", label, got, total)
		}
		for _, s := range mw.world.Sacks() {
			if s.X == at.X && s.Y == at.Y {
				t.Fatalf("%s: a sack lies on the thrower's own cell %d,%d", label, s.X, s.Y)
			}
			if thrownLivingAt(mw.world, s.X, s.Y) {
				t.Fatalf("%s: a sack lies beneath a living unit at %d,%d", label, s.X, s.Y)
			}
		}
		if len(mw.world.Sacks()) < before {
			t.Fatalf("%s: a sack vanished", label)
		}
	}
	startSacks := len(mw.world.Sacks())
	throw("far throw of a pack item", false, 0, at.X+5, at.Y)
	if got := len(mw.world.Sacks()); got != startSacks+1 {
		t.Fatalf("a throw beyond the window made %d new sacks, want 1", got-startSacks)
	}
	south := groundAt(mw.world.Sacks(), at.X, at.Y+1)
	if south == nil || len(south.Items) != 1 {
		t.Fatalf("the far throw did not land on the south neighbour %d,%d: %+v", at.X, at.Y+1, mw.world.Sacks())
	}
	throw("worn item released on the thrower", true, 0, at.X, at.Y)
	throw("second worn item thrown off the map", true, 1, 1<<20, 1<<20)
	for i := 2; i < sim.EquipSlots; i++ {
		throw(fmt.Sprintf("worn slot %d", i+1), true, i, at.X, at.Y)
	}
	if held, _ := mw.world.CarriedStacks(hero); len(held) != 0 {
		t.Fatalf("the hero still carries %+v", held)
	}
	for _, s := range mw.world.Sacks() {
		if s.X == at.X && s.Y == at.Y {
			t.Fatal("a thrown item lies under the thrower")
		}
	}
	want := thrownSackItems(mw.world)

	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal("SAVE after the throws", err)
	}
	cold, _ := loadSAVWindow(t, store, name)
	if got := thrownSackItems(cold.live.world); !reflect.DeepEqual(got, want) {
		t.Fatalf("cold LOAD changed the ground:\nwant %v\n got %v", want, got)
	}
	ch := cold.live.mission.ids[0]
	for tick := 0; tick < 20; tick++ {
		cold.live.tick()
	}
	if got := thrownHeld(cold.live.world, ch); got != total {
		t.Fatalf("after LOAD and 20 ticks the items held or on the ground are %d, want %d", got, total)
	}
	again, _, _ := cold.SaveSeams(store, OriginalStore{}, nil)
	if _, err := again(true); err != nil {
		t.Fatal("SAVE after LOAD", err)
	}

	// The sack beside the thrower is reachable through the ordinary pickup order.
	target := south
	cold.live.orderPickup(ch, target.X, target.Y)
	for tick := 0; tick < 400 && cold.live.pickup.set; tick++ {
		cold.live.tick()
	}
	if cold.live.pickup.set {
		t.Fatal("the hero never took the sack beside the thrower")
	}
	if left := groundAt(cold.live.world.Sacks(), target.X, target.Y); left != nil {
		t.Fatalf("the sack still lies at %d,%d: %+v", target.X, target.Y, left)
	}
	if got := thrownHeld(cold.live.world, ch); got != total {
		t.Fatalf("after the pickup the items held or on the ground are %d, want %d", got, total)
	}
}
