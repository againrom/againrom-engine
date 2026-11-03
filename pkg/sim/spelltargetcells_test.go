package sim

import "testing"

func TestCreatureBitEightSkipsTheSlotDraw(t *testing.T) {
	w := secondTeleportCreature(t, 9, 5)
	w.entities[1].OffMap = true
	before := w.rng
	w.creatureEngageCast(0, 2)
	if w.rng != before || len(w.engageDrew) != 0 {
		t.Fatal("bit-8 reset consumed a slot draw")
	}
	w = secondTeleportCreature(t, 9, 5)
	before = w.rng
	w.creatureEngageCast(0, 2)
	if w.rng == before || len(w.engageDrew) != 1 {
		t.Fatal("placed victim did not consume its carried slot draw")
	}
}

func TestCreatureTeleportTestsDynamicPlaneWithItsKindMask(t *testing.T) {
	for _, tc := range []struct {
		name   string
		domain Domain
		mask   byte
	}{
		{"kind1", DomainGround, 0x41}, {"kind2", DomainGhost, 0x44}, {"kind3", DomainAir, 0x82},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for bit := byte(1); bit != 0; bit <<= 1 {
				w := secondTeleportCreature(t, 9, 5)
				w.entities[0].Domain = tc.domain
				w.savedCellPlanes = &SavedCellPlanes{}
				w.savedCellPlanes.Dynamic[cellKey(8, 4)] = bit
				x, y, ok := w.creatureScanCell(w.entities[0], w.entities[1])
				wantX := int32(8)
				if bit&tc.mask != 0 {
					wantX = 9
				}
				if !ok || x != wantX || y != 4 {
					t.Errorf("P2 bit %02x mask %02x: %d,%d %v; want %d,4", bit, tc.mask, x, y, ok, wantX)
				}
			}
			w := secondTeleportCreature(t, 9, 5)
			w.entities[0].Domain, w.entities[0].TokenSize = tc.domain, 2
			w.savedCellPlanes = &SavedCellPlanes{}
			w.savedCellPlanes.Dynamic[cellKey(9, 4)] = tc.mask
			w.savedCellPlanes.Dynamic[cellKey(10, 4)] = tc.mask
			if x, y, ok := w.creatureScanCell(w.entities[0], w.entities[1]); !ok || x != 8 || y != 5 {
				t.Fatalf("full-footprint row-major scan: %d,%d %v; want 8,5", x, y, ok)
			}
		})
	}
}

func TestCreatureAcidStreamUsesCentresAndPackedEdgeCell(t *testing.T) {
	w := secondCreatureWorld(t, secondSlotRows(), [CreatureSpellSlots]CreatureSpell{}, 0, Spellbook{State: BookPresent})
	w.entities[0].TokenSize = 3
	for _, tc := range []struct{ vx, vy, x, y int32 }{
		{6, 1, 5, 4}, {10, 2, 6, 4}, {10, 6, 6, 5}, {10, 10, 6, 6},
		{6, 10, 5, 6}, {2, 10, 4, 6}, {1, 6, 4, 5}, {2, 2, 4, 4},
		{8, 7, 6, 6},
	} {
		w.entities[1].X, w.entities[1].Y = tc.vx, tc.vy
		if got := creatureOperandsOf(w, 9); !got.ok || got.x != tc.x || got.y != tc.y {
			t.Errorf("size 3 toward %d,%d: %v; want %d,%d", tc.vx, tc.vy, got, tc.x, tc.y)
		}
	}
	w.entities[0].X, w.entities[0].Y, w.entities[0].TokenSize = 0, 0, 1
	w.entities[1].X, w.entities[1].Y = 0, 0
	if got := creatureOperandsOf(w, 9); !got.ok || got.x != 255 || got.y != 255 {
		t.Fatalf("packed northwest edge: %v; want 255,255", got)
	}
}

func TestCreatureTeleportMaskIncludesNativeOccupantLayers(t *testing.T) {
	for _, casterDomain := range []Domain{DomainGround, DomainGhost, DomainAir} {
		for _, occupantDomain := range []Domain{DomainGround, DomainAir} {
			w := secondTeleportCreature(t, 9, 5)
			w.entities[0].Domain = casterDomain
			occupant := spEnt(3, 8, 4)
			occupant.Domain = occupantDomain
			w.entities = append(w.entities, occupant)
			wantX := int32(8)
			if (casterDomain == DomainAir) == (occupantDomain == DomainAir) {
				wantX = 9
			}
			if x, y, ok := w.creatureScanCell(w.entities[0], w.entities[1]); !ok || x != wantX || y != 4 {
				t.Fatalf("caster %d occupant %d: %d,%d %v; want %d,4", casterDomain, occupantDomain, x, y, ok, wantX)
			}
		}
	}
}

func TestStoneCurseBlankTagRecomputesUnitOnStartRenewalAndEnd(t *testing.T) {
	w := curseWorld(t, 24, 40)
	e := &w.entities[1]
	e.ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 1, HasOwner: true, ManaReservePercent: 75}}
	e.MaxMana = 80
	e.Capacity = 100
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
	for _, phase := range []string{"start", "renewal", "end"} {
		e = &w.entities[1]
		e.ActorLoad.Source.ManaFloor = 1
		if phase == "end" {
			w.attached[0].Remaining = 1
			w.stepAttachedEffects()
		} else if !w.attachEffect(2, 1, w.spells[0], EffectNone, 0, 3, EffectDuration) {
			t.Fatalf("%s: no attachment", phase)
		}
		if got := w.entities[1].ActorLoad.Source.ManaFloor; got != 60 {
			t.Errorf("%s: mana floor %d, want 60", phase, got)
		}
	}
}

func TestStoneCurseBlankTagCallsTheSharedSourceDeriveAtBothRenewalEnds(t *testing.T) {
	w := curseWorld(t, 24, 0)
	w.entities[1].ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 2}}
	w.entities[1].Capacity = 100
	calls := 0
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { calls++; return s, nil })
	for _, want := range []int{2, 5} {
		if !w.attachEffect(2, 1, w.spells[0], EffectNone, 0, 3, EffectDuration) || calls != want {
			t.Fatalf("start/renewal calls %d, want %d", calls, want)
		}
	}
	w.attached[0].Remaining = 1
	w.stepAttachedEffects()
	if calls != 6 || len(w.attached) != 0 {
		t.Fatalf("end calls %d, effects %v", calls, w.attached)
	}
}

func TestBlizzardFirstPulseWaitsForTheNextAreaPass(t *testing.T) {
	w := secondWorld(t, 7, Bounds{Width: 40, Height: 40}, []SpellRule{blizzardRow(10, 30)}, blizzardCaster(20, 12))
	before := w.rng
	if !w.landArea(w.spells[0], 50, 1, true, 20, 12, 20, 20, nil) {
		t.Fatal("no area")
	}
	if w.rng != before || len(w.effects[0].Cells) != 0 {
		t.Fatal("insertion already attempted a pulse")
	}
	states, err := w.NativeAreaSaveStates()
	if err != nil || len(states) != 1 || states[0].Stage != 0 || states[0].Remaining != 0 {
		t.Fatalf("before first update: %+v, %v", states, err)
	}
	cold := cloneWorld(t, w)
	for update := 1; update <= 94; update++ {
		r := w.rng
		if (update-1)%3 == 0 {
			r.uniform(5)
			r.uniform(5)
		}
		w.decayCellEffects(nil)
		cold.decayCellEffects(nil)
		if w.rng != r || w.Hash() != cold.Hash() {
			t.Fatalf("update %d: rng %+v want %+v cold %+v; hashes %x/%x; effects %+v/%+v", update, w.rng, r, cold.rng, w.Hash(), cold.Hash(), w.effects, cold.effects)
		}
	}
	if len(w.effects) != 0 {
		t.Fatal("area survived attempt 32")
	}
}

func TestBlizzardStruckCellWalksGroundAirThenStructure(t *testing.T) {
	caster := blizzardCaster(20, 12)
	ground, air := spEnt(2, 20, 20), spEnt(3, 20, 20)
	ground.TokenSize, air.Domain = 3, DomainAir
	building := Structure{ID: 1, Col: 20, Row: 20, Width: 1, Height: 1, MaxHealth: 100, Field42: 100, Attach: 1}
	w := blizzardWorld(t, 7, false, blizzardRow(10, 30), []Structure{building}, caster, ground, air)
	r := w.rng
	groundLoss := int32(10 + r.uniform(20))
	airLoss := int32(10 + r.uniform(20))
	buildingLoss := int32(10 + r.uniform(20) - 5)
	w.applyAreaCells(cellEffect{Spell: 7, Mode: areaModeRing}, w.spells[0], []uint16{cellKey(20, 20)})
	if w.entities[1].HP != 100-groundLoss || w.entities[2].HP != 100-airLoss || int32(w.structures[0].Field42) != 100-buildingLoss || w.rng != r {
		t.Fatalf("occupants: hp %d/%d building %d; expected losses %d/%d/%d", w.entities[1].HP, w.entities[2].HP, w.structures[0].Field42, groundLoss, airLoss, buildingLoss)
	}
}

func TestBlizzardCurrentAreaRestoresEveryCutIncludingBeforeFirstPulse(t *testing.T) {
	w := secondWorld(t, 7, Bounds{Width: 40, Height: 40}, []SpellRule{blizzardRow(10, 30)}, blizzardCaster(20, 12))
	if !w.landArea(w.spells[0], 50, 1, true, 20, 12, 20, 20, nil) {
		t.Fatal("no area")
	}
	for update := 0; update < 94; update++ {
		cold := currentAreaOrdinaryWorld(t, w, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("cut %d changed current state", update)
		}
		w.stepWorldSpellEffects(nil)
		cold.stepWorldSpellEffects(nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("cut %d changed the next area update", update)
		}
	}
}
