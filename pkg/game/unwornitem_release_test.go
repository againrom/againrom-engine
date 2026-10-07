package game

import (
	"testing"

	"againrom/pkg/sim"
)

// Taking off an item that raises the maximum health shrinks the pool and sends
// no damage message, so no hurt cue plays; the loss is not a blow.
func TestReleaseUnwearingAMaximumHealthItemIsNotDamage(t *testing.T) {
	f, app, id := wornItemOpen(t, sim.ItemEffect{Kind: 7, Operand: 30})
	before, _ := f.live.entity(id)
	wornItemWear(t, f, app, id)
	if e, _ := f.live.entity(id); e.MaxHP != before.MaxHP+30 || e.HP != before.HP+30 {
		t.Fatalf("worn: health %d/%d, want %d/%d", e.HP, e.MaxHP, before.HP+30, before.MaxHP+30)
	}
	f.live.enqueueUnequip(1)
	for n := 0; n < 4; n++ {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
	e, _ := f.live.entity(id)
	if e.HP != before.HP || e.MaxHP != before.MaxHP {
		t.Fatalf("unworn: health %d/%d, want %d/%d", e.HP, e.MaxHP, before.HP, before.MaxHP)
	}
	if n := f.live.blows[id]; n != 0 {
		t.Fatalf("unwearing sent %d damage messages", n)
	}
}
