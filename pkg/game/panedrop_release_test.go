package game

import (
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// A pack item dragged onto the character pane is worn while the pane shows the
// figure and while it shows the statistics card (`TOWN-348`). The item first
// leaves the doll by a tap, so the pack holds something this hero wears, and
// is dragged back with the pointer edges a player sends. The pane is switched
// with the paperdoll key.
func TestReleasePackItemDraggedOntoThePaneIsWornInBothModes(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("pack item dragged onto the pane")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	id := live.mission.ids[0]
	settle := func(frames int) {
		t.Helper()
		for n := 0; n < frames; n++ {
			if err := stepConsumableApp(app); err != nil {
				t.Fatal(err)
			}
		}
	}
	pointer := func(x, y int, edges ...string) {
		t.Helper()
		for _, edge := range edges {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
	}
	statistics := func() bool {
		t.Helper()
		_, on, err := app.HeadlessCharacterPane()
		if err != nil {
			t.Fatal(err)
		}
		return on
	}
	showStatistics := func(want bool) {
		t.Helper()
		if statistics() != want {
			if err := app.HeadlessKey("doll"); err != nil {
				t.Fatal(err)
			}
		}
		if statistics() != want {
			t.Fatalf("the pane shows statistics = %v after the paperdoll key, want %v", statistics(), want)
		}
	}
	carries := func(code uint16) int {
		stacks, _ := live.world.CarriedStacks(id)
		return slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == code })
	}

	settle(32)
	e, _ := live.entity(id)
	live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	settle(1)
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	worn, _ := live.world.Equipped(id)
	slot := 0
	for s := sim.EquipSlots; s >= 1 && slot == 0; s-- {
		if _, _, err := app.HeadlessDollSlotPoint(s); err == nil && worn[s-1] != 0 {
			slot = s
		}
	}
	if slot == 0 {
		t.Fatalf("no worn slot of %#x answers a doll pixel", worn)
	}
	code := worn[slot-1]
	t.Logf("slot %d wears %#x", slot, code)

	for _, mode := range []bool{false, true} {
		label := map[bool]string{false: "figure", true: "statistics"}[mode]
		showStatistics(false)
		sx, sy, err := app.HeadlessDollSlotPoint(slot)
		if err != nil {
			t.Fatal(err)
		}
		pointer(sx, sy, "press", "release")
		settle(3)
		if now, _ := live.world.Equipped(id); now[slot-1] != 0 || carries(code) < 0 {
			t.Fatalf("%s: a tap on slot %d left %#x worn and the pack at %d, want it carried", label, slot, now[slot-1], carries(code))
		}

		showStatistics(mode)
		cx, cy, err := app.HeadlessPackCellPoint(carries(code))
		if err != nil {
			t.Fatal(err)
		}
		px, py, err := app.HeadlessDollBoxPoint()
		if err != nil {
			t.Fatal(err)
		}
		pointer(cx, cy, "press")
		pointer(px, py, "move", "release")
		settle(3)
		now, _ := live.world.Equipped(id)
		if now[slot-1] != code || carries(code) >= 0 {
			t.Fatalf("%s: after the drag slot %d wears %#x and the pack index is %d, want %#x worn and none carried",
				label, slot, now[slot-1], carries(code), code)
		}
		if statistics() != mode {
			t.Fatalf("%s: the drop changed the pane's mode to statistics = %v", label, statistics())
		}
	}
}
