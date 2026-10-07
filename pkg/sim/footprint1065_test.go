package sim

import (
	"bytes"
	"testing"
)

func TestFootprintAdmissionReadsEveryTerrainAndOccupancyCell(t *testing.T) {
	const width = 12
	grid := make([]byte, width*width)
	grid[4*width+4] = blockGround
	mover := Entity{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, TokenSize: 2}
	blocker := Entity{ID: 2, X: 7, Y: 7, HP: 100, MaxHP: 100, TokenSize: 1}
	w := mustWorldGrid(t, 1065, Bounds{Width: width, Height: width}, ModeCanonical, grid,
		[]Entity{mover, blocker})
	s := newRouteScratch(w)

	// The anchor (3,3) is clear, but the footprint's far corner (4,4) is not.
	if w.terrainOpenFootprint(w.entities[0], 3, 3) {
		t.Fatal("a 2x2 mover was admitted with its far terrain cell blocked")
	}
	if w.open(s, terrainRelation, 0, 3, 3) {
		t.Fatal("the terrain route relation admitted a partially blocked footprint")
	}
	// The anchor (6,6) is free, but its far corner is held by entity 2.
	if w.enterable(s, 0, 6, 6) {
		t.Fatal("a 2x2 mover was admitted with its far occupancy cell held")
	}
	// Moving one cell keeps half the old footprint. Those self-held cells must
	// be subtracted rather than mistaken for another actor.
	if !w.enterable(s, 0, 2, 1) {
		t.Fatal("a 2x2 mover could not advance across cells shared with its old footprint")
	}
}

func TestFootprintMoveReplacesTheWholeOccupancyAtomically(t *testing.T) {
	b := Bounds{Width: 8, Height: 8}
	w := mustWorld(t, 1065, b, []Entity{{
		ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, TokenSize: 2,
	}})
	s := newRouteScratch(w)
	e := w.entities[0]
	from, to := cell{x: 1, y: 1}, cell{x: 2, y: 1}
	s.moved(w, e, from, to)
	w.entities[0].X, w.entities[0].Y = to.x, to.y

	for y := int32(0); y < b.Height; y++ {
		for x := int32(0); x < b.Width; x++ {
			want := int32(0)
			if x >= 2 && x <= 3 && y >= 1 && y <= 2 {
				want = 1
			}
			i, _ := w.cellIndex(x, y)
			if got := s.at(DomainGround.layer(), i); got != want {
				t.Fatalf("occupancy after 2x2 move at (%d,%d) = %d, want %d", x, y, got, want)
			}
		}
	}
}

func TestStepPublishesANewLargeFootprintBeforeTheNextActorResolves(t *testing.T) {
	w := mustWorld(t, 1065, Bounds{Width: 10, Height: 10}, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, TokenSize: 2},
		{ID: 2, X: 4, Y: 2, HP: 100, MaxHP: 100, TokenSize: 1},
	})
	Step(w, []Command{
		{Kind: KindMoveTo, Entity: 1, X: 2, Y: 1},
		{Kind: KindMoveTo, Entity: 2, X: 3, Y: 2},
	})

	big, small := spAt(t, w, 1), spAt(t, w, 2)
	if big.X != 2 || big.Y != 1 {
		t.Fatalf("2x2 actor moved to (%d,%d), want (2,1)", big.X, big.Y)
	}
	if small.X == 3 && small.Y == 2 {
		t.Fatal("later actor entered the first actor's newly published non-anchor footprint cell")
	}
}

func TestRestingFlyerPublishesItsWholeFootprint(t *testing.T) {
	w := mustWorld(t, 1065, Bounds{Width: 10, Height: 10}, []Entity{
		{ID: 1, X: 0, Y: 1, HP: 100, MaxHP: 100, TokenSize: 2, Domain: DomainAir},
		{ID: 2, X: 3, Y: 2, HP: 100, MaxHP: 100, TokenSize: 1, Domain: DomainAir},
	})
	Step(w, []Command{
		{Kind: KindMoveTo, Entity: 1, X: 1, Y: 1},
		{Kind: KindMoveTo, Entity: 2, X: 2, Y: 2},
	})

	big, small := spAt(t, w, 1), spAt(t, w, 2)
	if big.X != 1 || big.Y != 1 || big.HasTarget {
		t.Fatalf("2x2 flyer did not rest at (1,1): %+v", big)
	}
	if small.X == 2 && small.Y == 2 {
		t.Fatal("later flyer rested on a non-anchor cell of the first flyer's new footprint")
	}
}

func TestPursuingLargeFlyerDoesNotRestOnAnotherFlyer(t *testing.T) {
	attacker := cbEnt(1, 0, 2)
	attacker.Domain, attacker.TokenSize = DomainAir, 2
	attacker.AttackCharge, attacker.AttackRelax = 1, 0
	attacker.DamageBase, attacker.AlwaysHits = 10, true
	resting := cbEnt(2, 2, 2)
	resting.Domain = DomainAir
	victim := cbEnt(3, 3, 2)
	w := mustWorld(t, 1065, Bounds{Width: 12, Height: 8}, []Entity{attacker, resting, victim})

	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 3}})
	if got := spAt(t, w, 1); got.X != 1 || got.Y != 2 || !got.HasTarget {
		t.Fatalf("pursuit tick 1 left attacker at (%d,%d), target=%v", got.X, got.Y, got.HasTarget)
	}
	Step(w, nil)
	if got := spAt(t, w, 1); got.X != 2 || got.Y != 2 || !got.HasTarget || counted(got) {
		t.Fatalf("pursuit tick 2 left attacker at (%d,%d), target=%v counted=%v; want a moving crossing",
			got.X, got.Y, got.HasTarget, counted(got))
	}

	// At this point the attacker is in range but its 2x2 resting footprint
	// would cover entity 2. The stop attempt must refuse without ending pursuit,
	// allowing the ordinary mover to close one more cell and rest clear of it.
	Step(w, nil)
	got, held := spAt(t, w, 1), spAt(t, w, 2)
	if got.X != 3 || got.Y != 2 || got.HasTarget || !counted(got) {
		t.Fatalf("refused rest stranded attacker at (%d,%d), target=%v counted=%v; want resting at (3,2)",
			got.X, got.Y, got.HasTarget, counted(got))
	}
	if !got.HasAttackTarget || got.AttackTarget != 3 {
		t.Fatalf("refused rest cleared pursuit %v/%d", got.HasAttackTarget, got.AttackTarget)
	}
	if counted(held) && footprintsOverlap(got.X, got.Y, got.TokenSize, held.X, held.Y, held.TokenSize) {
		t.Fatalf("resting attacker footprint at (%d,%d) overlaps resting flyer at (%d,%d)",
			got.X, got.Y, held.X, held.Y)
	}
	if victim := spAt(t, w, 3); victim.HP >= 100 {
		t.Fatalf("continued pursuit reached the victim but did not attack: HP=%d", victim.HP)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary corrected pursuit: %v", err)
	}
	wantHash := w.Hash()
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary corrected pursuit: %v", err)
	}
	if gotHash := back.Hash(); gotHash != wantHash {
		t.Fatalf("corrected pursuit hash after load = %016x, want %016x", gotHash, wantHash)
	}
	loaded, loadedHeld := spAt(t, &back, 1), spAt(t, &back, 2)
	if counted(loaded) && counted(loadedHeld) &&
		footprintsOverlap(loaded.X, loaded.Y, loaded.TokenSize, loadedHeld.X, loadedHeld.Y, loadedHeld.TokenSize) {
		t.Fatal("corrected pursuit round-tripped a same-layer resting overlap")
	}
}

func TestLargeFlyerPursuitRemovesItsWholeRestingFootprint(t *testing.T) {
	attacker := cbEnt(1, 1, 1)
	attacker.Domain, attacker.TokenSize = DomainAir, 2
	w := mustWorld(t, 1065, Bounds{Width: 8, Height: 8}, []Entity{attacker, cbEnt(2, 6, 1)})
	s := newRouteScratch(w)
	w.walkTo(s, 0, 6, 1)

	for y := int32(1); y <= 2; y++ {
		for x := int32(1); x <= 2; x++ {
			i, _ := w.cellIndex(x, y)
			if got := s.at(DomainAir.layer(), i); got != 0 {
				t.Fatalf("moving 2x2 flyer left resting occupancy %d at (%d,%d)", got, x, y)
			}
		}
	}
}

func TestTeleportPaysForAnOccupiedOrPartiallyBlockedFootprint(t *testing.T) {
	teleport := SpellRule{ID: 26, ManaCost: 20, School: 5, MaxRange: 16}
	for _, tc := range []struct {
		name     string
		block    bool
		occupant bool
	}{
		{name: "occupied far cell", occupant: true},
		{name: "blocked far cell", block: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := effectMage(1, 1, 1, 1<<26)
			caster.TokenSize = 2
			ents := []Entity{caster}
			if tc.occupant {
				ents = append(ents, Entity{ID: 2, X: 6, Y: 6, HP: 100, MaxHP: 100, TokenSize: 1})
			}
			grid := make([]byte, 16*16)
			if tc.block {
				grid[6*16+6] = blockGround
			}
			w, err := NewStockedSpelledWorld(1065, Bounds{Width: 16, Height: 16}, ModeCanonical,
				Terrain{Block: grid}, ents, nil, Relations{}, nil, nil, []SpellRule{teleport})
			if err != nil {
				t.Fatalf("NewStockedSpelledWorld: %v", err)
			}
			if r := w.BookSpellCellRefusal(1, 5, 5, 26); r != "" {
				t.Fatalf("blocked relocation must not refuse cast admission: %q", r)
			}
			before := spAt(t, w, 1)
			spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 5, Y: 5, Spell: 26})
			after := spAt(t, w, 1)
			if after.X != before.X || after.Y != before.Y || after.Mana != before.Mana-20 || after.CastWait == 0 {
				t.Fatalf("refused Teleport changed caster from (%d,%d), mana %d to (%d,%d), mana %d, recovery %d",
					before.X, before.Y, before.Mana, after.X, after.Y, after.Mana, after.CastWait)
			}
		})
	}
}

func TestUnitTargetTeleportPaysForTheOccupiedTarget(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<26)
	caster.TokenSize = 2
	beacon := Entity{ID: 2, X: 5, Y: 5, HP: 100, MaxHP: 100, TokenSize: 1}
	w := hlWorld(t, 1065, Relations{}, []SpellRule{{ID: 26, ManaCost: 20, School: 5, MaxRange: 16}},
		caster, beacon)

	if r := w.BookSpellRefusal(1, 2, 26); r != "" {
		t.Fatalf("unit-form cast must admit a blocked relocation: %q", r)
	}
	before := spAt(t, w, 1)
	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 26})
	after := spAt(t, w, 1)
	if after.X != before.X || after.Y != before.Y || after.Mana != before.Mana-20 || after.CastWait == 0 {
		t.Fatalf("refused unit-form Teleport changed caster from (%d,%d), mana %d to (%d,%d), mana %d, recovery %d",
			before.X, before.Y, before.Mana, after.X, after.Y, after.Mana, after.CastWait)
	}
}

func TestTeleportReleaseRechecksFootprintOccupancyWithoutRefund(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<26)
	caster.TokenSize = 2
	other := Entity{ID: 2, X: 12, Y: 12, HP: 100, MaxHP: 100, TokenSize: 1}
	w := hlWorld(t, 1065, Relations{}, []SpellRule{{ID: 26, ManaCost: 20, School: 5, MaxRange: 16}},
		caster, other)

	Step(w, []Command{{Kind: KindCastAt, Entity: 1, X: 5, Y: 5, Spell: 26}})
	if len(w.bookCasts) != 1 {
		t.Fatalf("admission queued %d casts, want one wind-up", len(w.bookCasts))
	}
	// Another same-layer actor enters the destination's far footprint cell
	// during the wind-up. Only relocation is refused; the cast still costs mana.
	w.entities[1].X, w.entities[1].Y = 6, 6
	finishCellCast(w)
	got := spAt(t, w, 1)
	if got.X != 1 || got.Y != 1 || got.Mana != 80 || got.CastWait == 0 {
		t.Fatalf("occupied-at-release cast left caster at (%d,%d), mana %d, recovery %d",
			got.X, got.Y, got.Mana, got.CastWait)
	}
}

func TestScriptPlacementUsesTheWholeFootprint(t *testing.T) {
	returning := Entity{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, TokenSize: 2, OffMap: true}
	other := Entity{ID: 2, X: 6, Y: 6, HP: 100, MaxHP: 100, TokenSize: 1}
	w := mustWorld(t, 1065, Bounds{Width: 12, Height: 12}, []Entity{returning, other})
	if w.placeFree(0, 5, 5) {
		t.Fatal("script placement admitted a 2x2 actor over an occupied far cell")
	}
}

func TestSameVersionRouteWithABlockedNonAnchorLoadsAndKeepsItsRoute(t *testing.T) {
	b := Bounds{Width: 8, Height: 8}
	grid := make([]byte, b.Width*b.Height)
	grid[2*b.Width+3] = blockGround // covered by old route anchor (2,1), not the anchor itself
	e := Entity{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, TokenSize: 2,
		Domain: DomainGround, HasTarget: true, TargetX: 4, TargetY: 1}
	legacy := mustWorldGrid(t, 1065, b, ModeCanonical, grid, []Entity{e})
	legacy.routes[0] = []cell{{x: 2, y: 1}, {x: 3, y: 1}, {x: 4, y: 1}}
	form, err := legacy.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary legacy same-version route: %v", err)
	}

	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatalf("old same-version route no longer loads: %v", err)
	}
	again, err := loaded.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary loaded legacy route: %v", err)
	}
	if !bytes.Equal(again, form) {
		t.Fatal("loading changed the accepted same-version route before its first use")
	}

	for tick := 0; tick < 63; tick++ {
		got := loaded.entities[0]
		if entityCoversCell(got, 3, 2) {
			t.Fatalf("tick %d: large actor crossed the blocked non-anchor cell from (%d,%d)", tick+1, got.X, got.Y)
		}
		if got.X == 4 && got.Y == 1 {
			return
		}
		Step(&loaded, nil)
	}
	got := loaded.entities[0]
	t.Fatalf("large actor did not arrive: ended at (%d,%d), route %v", got.X, got.Y, loaded.routes[0])
}
