package game

import (
	"againrom/pkg/sim"
	"fmt"
	"testing"
)

func TestScrollDisplayArmCannotSpendAChangedSlot(t *testing.T) {
	item := sim.ItemInstance{Code: 0xe10, Kind: 4, Effects: []sim.ItemEffect{{Kind: 41, Operand: 1 | 40<<16}}}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 1, Owner: sim.SelfSlot, X: 2, Y: 2, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 1, ItemInstances: []sim.ItemInstance{item}}})
	if err != nil {
		t.Fatal(err)
	}
	mw := &mapWorld{world: w, commanded: make(map[sim.EntityID]bool)}
	changed := item.Clone()
	changed.Effects[0].Operand += 1 << 16
	mw.useScroll(1, 0, fmt.Sprintf("%#v", changed), 1, 2, 2, false)
	if len(mw.pending) != 0 {
		t.Fatal("stale display identity issued a use")
	}
	mw.useScroll(1, 0, fmt.Sprintf("%#v", item), 1, 2, 2, false)
	if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindUseScroll {
		t.Fatal("unchanged display identity refused")
	}
}
