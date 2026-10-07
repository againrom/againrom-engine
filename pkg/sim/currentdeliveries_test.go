package sim

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"testing"
)

func ordinaryCurrentDelivery(t *testing.T, native *World, mutate func(*SavedSpellGraph)) (*World, CurrentDeliveryRestore) {
	t.Helper()
	rows := native.NativeSpellDeliverySaveStates()
	if len(rows) != 1 {
		t.Fatal("delivery fixture", rows)
	}
	r := rows[0]
	cold := new(World)
	if err := cold.UnmarshalBinary(mustMarshal(t, native)); err != nil {
		t.Fatal(err)
	}
	cold.deliveries, cold.effectOrder = nil, nil
	cold.tick, cold.hasSessionClock = uint64(uint32(cold.tick)), true
	child := SavedSpellNode{Value: SavedSpellEffect{Class: "PointEffect", SE41: 1, PE48: &r.Area.Payload}, Spell: r.Area.Spell, Target: r.Target, HasTarget: r.HasTarget}
	if !r.Attribution {
		child.Value.SE41 = 0
	}
	if r.AtCell {
		child.Value.Class, child.Value.PE48 = "AreaEffect", nil
		child.Value.AE44, child.Value.AE4C = &r.Area.Payload, r.Area.Remaining
		child.Value.AE48 = [4]byte{0, r.Area.Radius, r.Area.Direction << 5, r.Area.Stage}
	}
	g := &SavedSpellGraph{Roots: []uint32{1}, Nodes: []SavedSpellNode{child}}
	if !r.Released {
		g.Nodes = []SavedSpellNode{{Value: SavedSpellEffect{Class: "SpellTransport", ST4C: r.Remaining}, Primary: 2}, child}
	}
	cold.savedWorldEffects = &SavedWorldEffects{}
	if r.AtCell {
		id := uint32(len(g.Nodes))
		cold.savedWorldEffects.Areas = []SavedAreaDriver{{ID: id, Root: -1, Identity: 17, Key: r.Area.Key, Layer: 255, Mode: r.Area.Mode, Spell: r.Area.Spell}}
	}
	if mutate != nil {
		mutate(g)
	}
	cold.savedSpellGraph = g
	cold.refreshSavedSpellGraph()
	x, y := keyCell(r.Area.Key)
	return cold, CurrentDeliveryRestore{Node: 1, From: CellPoint{r.FromX, r.FromY}, Destination: CellPoint{x, y}, Policy: r.Policy}
}

func transferCurrentDelivery(t *testing.T, native *World, mutate func(*SavedSpellGraph)) *World {
	t.Helper()
	cold, row := ordinaryCurrentDelivery(t, native, mutate)
	p := native.CurrentPolicy()
	if err := cold.RestoreCurrentClock(p.TickHigh, p.ClockKnown); err != nil {
		t.Fatal(err)
	}
	if err := cold.RestoreCurrentSpellDeliveries([]CurrentDeliveryRestore{row}); err != nil {
		t.Fatal(err)
	}
	if err := cold.RestoreCurrentContinuation(&p, nil, cold.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if cold.savedSpellGraph != nil || cold.savedWorldEffects != nil || len(cold.deliveries) != 1 {
		t.Fatal("ownership transfer retained temporary carriers or lost pending delivery")
	}
	requireSpellGraphBinary(t, cold)
	return cold
}

func equalCurrentDelivery(t *testing.T, native, cold *World, step int) {
	t.Helper()
	want, _ := json.Marshal(native.NativeSpellDeliverySaveStates())
	got, _ := json.Marshal(cold.NativeSpellDeliverySaveStates())
	if !bytes.Equal(want, got) || !reflect.DeepEqual(native.Entities(), cold.Entities()) || native.RandomState() != cold.RandomState() || !reflect.DeepEqual(native.ActiveEffects(), cold.ActiveEffects()) || !slices.Equal(native.CurrentWorldEffectOrder(), cold.CurrentWorldEffectOrder()) {
		t.Fatalf("delivery continuation at %d differs: %s / %s; HP %d / %d", step, want, got, native.entities[1].HP, cold.entities[1].HP)
	}
}

func TestCurrentDeliveryRetainsOrdinaryTimerTargetAndFutureImpact(t *testing.T) {
	for _, remaining := range []uint16{0, 1, 4, 65535} {
		for _, released := range []bool{false, true} {
			w := deliveryTestWorld(t, 1)
			w.spells[0].Distribution = 1
			w.queuePointDelivery(0, 1, w.spells[0], 0)
			w.deliveries[0].Remaining, w.deliveries[0].Released = remaining, released
			cold := transferCurrentDelivery(t, w, nil)
			if cold.spells[0].Distribution != 1 {
				t.Fatal("point delivery lost its installed distribution column")
			}
			equalCurrentDelivery(t, w, cold, 0)
			for step := 1; step <= 7; step++ {
				for _, world := range []*World{w, cold} {
					world.entities[0].HP = 0
					world.entities[1].X = 12
					world.stepWorldSpellEffects(nil)
				}
				equalCurrentDelivery(t, w, cold, step)
			}
			if w.entities[1].HP != 995 || cold.PendingSpellDeliveries() != 0 {
				t.Fatal("pending point did not hit exactly once")
			}
		}
	}
}

func TestCurrentDeliveryKeepsClockAboveOrdinaryTickWidth(t *testing.T) {
	w := deliveryTestWorld(t, 1)
	w.tick = 1<<32 | 17
	w.queuePointDelivery(0, 1, w.spells[0], 0)
	cold := transferCurrentDelivery(t, w, nil)
	for step := 0; step <= 10; step++ {
		if step != 0 {
			for _, world := range []*World{w, cold} {
				world.tick++
				world.stepWorldSpellEffects(nil)
			}
		}
		if cold.tick != w.tick || cold.hasSessionClock {
			t.Fatal("delivery clock was truncated or changed mode", cold.tick, w.tick)
		}
		equalCurrentDelivery(t, w, cold, step)
		if step == 1 {
			cold = transferCurrentDelivery(t, cold, nil)
		}
	}
	if w.entities[1].HP != 995 || cold.PendingSpellDeliveries() != 0 {
		t.Fatal("high-clock delivery did not hit exactly once")
	}
}

func TestCurrentDeliveryUsesOrdinaryMutationsWithoutChangingPolicy(t *testing.T) {
	w := deliveryTestWorld(t, 1)
	w.queuePointDelivery(0, 1, w.spells[0], 0)
	cold := transferCurrentDelivery(t, w, func(g *SavedSpellGraph) {
		g.Nodes[0].Value.ST4C = 1
		g.Nodes[1].Value.PE48.DirectDamage[19] = 19
	})
	w.stepWorldSpellEffects(nil)
	cold.stepWorldSpellEffects(nil)
	if cold.entities[1].HP != 1000 {
		t.Fatal("transport release applied its child early")
	}
	w.stepWorldSpellEffects(nil)
	cold.stepWorldSpellEffects(nil)
	if w.entities[1].HP != 1000 || cold.entities[1].HP != 981 {
		t.Fatal("ordinary counter/payload edits were overwritten by policy")
	}
	for _, stop := range []int{0, 1} {
		cold = transferCurrentDelivery(t, w, func(g *SavedSpellGraph) { g.Nodes[stop].Value.SE40 = 1 })
		for step := 0; step < 10; step++ {
			cold.stepWorldSpellEffects(nil)
		}
		if cold.entities[1].HP != 1000 || cold.PendingSpellDeliveries() != 0 {
			t.Fatal("ordinary stopped transport/child was ignored", stop)
		}
	}
}

func TestCurrentDeliveryAreaKeepsAimRadiusPayloadAndCloudLife(t *testing.T) {
	for _, spell := range []uint16{2, 3} {
		w := deliveryTestWorld(t, spell)
		r := &w.spells[0]
		r.Area, r.TargetsUnit, r.Distribution, r.Radius = true, false, distributionDiamond, 1
		if spell == 3 {
			r.AreaDuration = 1
		}
		if !w.landArea(*r, 0, 1, true, 1, 1, 5, 1, nil) {
			t.Fatal("area preparation")
		}
		cold := transferCurrentDelivery(t, w, nil)
		for step := 0; step < 28; step++ {
			for _, world := range []*World{w, cold} {
				world.entities[1].X = int32(5 + step%3)
				world.stepWorldSpellEffects(nil)
			}
			equalCurrentDelivery(t, w, cold, step)
			want, _ := w.NativeAreaSaveStates()
			got, _ := cold.NativeAreaSaveStates()
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("spell %d area life at %d differs: %v / %v", spell, step, want, got)
			}
		}
	}
	w := deliveryTestWorld(t, 2)
	r := w.spells[0]
	r.Area, r.TargetsUnit, r.Distribution, r.Radius = true, false, distributionDiamond, 0
	w.spells[0] = r
	w.landArea(r, 0, 1, true, 1, 1, 5, 1, nil)
	cold := transferCurrentDelivery(t, w, func(g *SavedSpellGraph) {
		g.Nodes[0].Value.ST4C = 1
		g.Nodes[1].Value.AE48[1] = 1
		g.Nodes[1].Value.AE44.DirectDamage[19] = 19
	})
	cold.entities[1].X = 6
	cold.stepWorldSpellEffects(nil)
	cold.stepWorldSpellEffects(nil)
	if cold.entities[1].HP != 981 {
		t.Fatal("ordinary area radius/payload was not authoritative", cold.entities[1].HP)
	}
}

func TestCurrentDeliveryTransferPreservesOtherAliasesAndRejectsAmbiguousOwnershipAtomically(t *testing.T) {
	w := deliveryTestWorld(t, 1)
	w.queuePointDelivery(0, 1, w.spells[0], 0)
	cold, row := ordinaryCurrentDelivery(t, w, nil)
	g := cold.savedSpellGraph
	g.Nodes = append(g.Nodes, SavedSpellNode{Value: SavedSpellEffect{Class: "SpellTransport", ST4C: 3}, Primary: 4}, g.Nodes[1], SavedSpellNode{Value: SavedSpellEffect{Class: "PointEffect"}, Retired: true})
	g.Roots = []uint32{3, 1, 3}
	cold.refreshSavedSpellGraph()
	before := cold.Hash()
	if err := cold.RestoreCurrentSpellDeliveries([]CurrentDeliveryRestore{row, row}); err == nil || cold.Hash() != before {
		t.Fatal("duplicate ownership was not rejected atomically", err)
	}
	if err := cold.RestoreCurrentSpellDeliveries([]CurrentDeliveryRestore{row}); err != nil {
		t.Fatal(err)
	}
	g = cold.savedSpellGraph
	if len(g.Nodes) != 3 || !g.Nodes[2].Retired || g.Nodes[0].Primary != 2 || !slices.Equal(g.Roots, []uint32{1, 1}) || !slices.Equal(cold.CurrentWorldEffectOrder(), []WorldEffectRef{{EffectSavedGraph, 1}, {EffectNativeDelivery, 0}, {EffectSavedGraph, 1}}) {
		t.Fatal("ownership transfer changed unrelated aliases, history or order")
	}
	requireSpellGraphBinary(t, cold)
	cold, row = ordinaryCurrentDelivery(t, w, func(g *SavedSpellGraph) { g.Roots = []uint32{1, 2} })
	before = cold.Hash()
	if err := cold.RestoreCurrentSpellDeliveries([]CurrentDeliveryRestore{row}); err == nil || cold.Hash() != before {
		t.Fatal("a separately rooted child was silently consumed", err)
	}
	b, err := json.Marshal(row.Policy)
	if err != nil || bytes.Contains(b, []byte("Rule")) || bytes.Contains(b, []byte("Payload")) || bytes.Contains(b, []byte("Remaining")) || bytes.Contains(b, []byte("Damage")) {
		t.Fatal("ordinary state leaked into delivery policy", string(b), err)
	}
}

func sacrificeDeliveryWorld(t *testing.T, hasCaster bool) *World {
	t.Helper()
	w := deliveryTestWorld(t, 4)
	r := SpellRule{ID: 4, Area: true, Distribution: distributionStaged, School: 1, Delivery: 2, EffectSpeed: 256}
	var err error
	w, err = NewSpelledWorld(17, Bounds{32, 32}, ModeCanonical, nil, w.entities, nil, []SpellRule{r})
	if err != nil {
		t.Fatal(err)
	}
	w.relations.turnHostile(SelfSlot, 2)
	e := cellEffect{Key: 0x1010, Spell: 4, Direction: areaDirection(15, 15)}
	first, second := w.ringStageCells(e, 0), w.ringStageCells(e, 1)
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("sacrifice ring fixture")
	}
	w.entities[1].X, w.entities[1].Y = keyCell(first[0])
	other := w.entities[1]
	other.ID = 3
	other.X, other.Y = keyCell(second[0])
	w.entities = append(w.entities, other)
	w, err = NewSpelledWorld(17, Bounds{32, 32}, ModeCanonical, nil, w.entities, nil, []SpellRule{r})
	if err != nil {
		t.Fatal(err)
	}
	w.relations.turnHostile(SelfSlot, 2)
	caster := EntityID(0)
	if hasCaster {
		caster = 1
	}
	if !w.landArea(r, 0, caster, hasCaster, 1, 1, 16, 16, nil) {
		t.Fatal("sacrifice queue")
	}
	w.deliveries[0].Remaining = 1
	return w
}

func TestCurrentDeliverySacrificeConstructsAtImpactAndKeepsLaterStages(t *testing.T) {
	for _, hasCaster := range []bool{true, false} {
		w := sacrificeDeliveryWorld(t, hasCaster)
		cold := transferCurrentDelivery(t, w, nil)
		for _, world := range []*World{w, cold} {
			// These pools differ from both queue time and SAVE time.
			world.entities[0].HP, world.entities[0].Mana = 80, 20
		}
		for step := 0; step < 12; step++ {
			w.stepWorldSpellEffects(nil)
			cold.stepWorldSpellEffects(nil)
			equalCurrentDelivery(t, w, cold, step)
			want, _ := w.NativeAreaSaveStates()
			got, _ := cold.NativeAreaSaveStates()
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("sacrifice stage %d differs: %v / %v", step, want, got)
			}
			if step == 1 && hasCaster {
				if w.entities[1].HP != 900 || w.entities[0].HP != 1 || w.entities[0].Mana != 0 {
					t.Fatal("impact did not construct and spend once")
				}
				// Saving after impact retains its prepared payload; it must not
				// spend again or reconstruct from a subsequently changed caster.
				cold = currentAreaOrdinaryWorld(t, cold, nil)
				w.entities[0].HP, cold.entities[0].HP = 70, 70
			}
		}
		if hasCaster && (w.entities[2].HP != 900 || w.entities[0].HP != 70) {
			t.Fatal("later ring stage re-derived or spent again")
		}
		if !hasCaster && (w.entities[1].HP != 1000 || w.entities[0].HP != 80) {
			t.Fatal("missing caster gained a construction")
		}
	}
}

func TestCurrentDeliverySacrificeHonorsPreparedOrdinaryPayloadAndStop(t *testing.T) {
	for _, stop := range []bool{false, true} {
		w := sacrificeDeliveryWorld(t, true)
		cold := transferCurrentDelivery(t, w, func(g *SavedSpellGraph) {
			p := SavedEffect{Class: "Effect_DirectDamage", E0C: 4}
			p.DirectDamage[19], p.DirectDamage[21] = 17, 1
			g.Nodes[1].Value.AE44 = &p
			if stop {
				g.Nodes[1].Value.SE40 = 1
			}
		})
		cold.entities[0].HP, cold.entities[0].Mana = 80, 20
		for step := 0; step < 6; step++ {
			cold.stepWorldSpellEffects(nil)
		}
		if stop {
			if cold.entities[1].HP != 1000 || cold.entities[0].HP != 80 || cold.entities[0].Mana != 20 {
				t.Fatal("stopped ordinary sacrifice spent or hit")
			}
		} else if cold.entities[1].HP != 983 || cold.entities[2].HP != 983 || cold.entities[0].HP != 1 || cold.entities[0].Mana != 0 {
			t.Fatal("ordinary prepared damage lost to old native construction", cold.entities)
		}
	}
}

func TestCurrentDeliverySacrificeIgnoresUnconstructedTableDamage(t *testing.T) {
	for _, hasCaster := range []bool{false, true} {
		w := sacrificeDeliveryWorld(t, hasCaster)
		for _, r := range []*SpellRule{&w.spells[0], &w.deliveries[0].Rule} {
			r.Damaging, r.DamageMin, r.DamageMax = true, 7, 9
		}
		w.deliveries[0].Power = 37
		cold := transferCurrentDelivery(t, w, nil)
		for _, world := range []*World{w, cold} {
			world.entities[0].HP, world.entities[0].Mana = 80, 20
		}
		for step := 0; step < 8; step++ {
			w.stepWorldSpellEffects(nil)
			cold.stepWorldSpellEffects(nil)
			equalCurrentDelivery(t, w, cold, step)
		}
		if hasCaster && w.entities[1].HP > 900 {
			t.Fatal("impact reused the table damage")
		}
	}
}
