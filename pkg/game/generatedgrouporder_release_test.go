package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseMissionGroupOrdersSurviveCurrentSave(t *testing.T) {
	f := releaseFront(t)
	app := f.App("current mission group orders")
	res := ui.ChargenResult{Name: "Load walk", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}}
	if err := app.OpenMission(f.NewGameOpener(20, res)); err != nil {
		t.Fatal(err)
	}
	if f.live.mission.state.savedDocument != nil {
		t.Fatal("mission entry fabricated a saved document")
	}
	w := f.live.world
	check := func(w *sim.World) {
		seen := map[[2]uint32]bool{}
		self, other := 0, 0
		for _, e := range w.Entities() {
			pair := [2]uint32{e.Owner, e.Group}
			if seen[pair] {
				continue
			}
			seen[pair] = true
			want := uint8(1)
			if e.Owner == sim.SelfSlot {
				want = 3
				self++
			} else {
				other++
			}
			order, base, known := w.FrozenGroupAI(e.Owner, e.Group)
			if !known || order != want || base == 0 {
				t.Fatalf("group %v order=%d base=%d known=%t, want order%d", pair, order, base, known, want)
			}
		}
		if self == 0 || other == 0 {
			t.Fatal("fixture lacks both owner populations", self, other)
		}
	}
	check(w)
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	hash := w.Hash()
	constructedHash := worldHashWithConstructedCurrentSessionHead1115(t, w)
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil || w.Hash() != hash {
		t.Fatal("current SAVE changed state or failed", err)
	}
	cold := releaseFront(t)
	open, town, err := cold.RestoreOriginal(raw)
	if err == nil && !town {
		err = cold.App("current group cold LOAD").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("current cold LOAD", town, err)
	}
	check(cold.live.world)
	if cold.live.world.Hash() != hash && cold.live.world.Hash() != constructedHash {
		t.Fatal("current group SAVE/LOAD changed World", hash, cold.live.world.Hash())
	}
}
