package sim

import "testing"

// A cast's activity stamp, for a book and a scroll cast at a unit behind the
// caster, which turns 32 ticks first. The second game stores the tick its
// cast routine runs: a book cast on the tick it pays, after the turn, and a
// scroll cast on its wind-up's first tick, the turn's end; its tripled
// restoration returns 81 ticks later. The first game stores, on the order's
// tick, the run end of turn, wind-up and recovery.
//
// The stamp after the step that starts the turn is the turn's own end in the
// second game and the cast's run end in the first. The test then erases the
// stamp while the caster turns, so the first stamp after the turn is the
// cast's. Fails while the second game stamps at the order.
func TestCastStampByGame(t *testing.T) {
	const unset = 1 << 30
	for _, tc := range []struct {
		second, scroll bool
		// turnEnd is the stamp after the step that starts the turn: the
		// book order and the turn come at tick 0, the scroll's at tick 1.
		turnEnd uint32
		// castAt is the tick of the first stamp after the turn; 0 is none.
		castAt uint64
	}{
		{second: true, turnEnd: 32, castAt: 32},
		{second: true, scroll: true, turnEnd: 33, castAt: 33},
		// Turn 32, wind-up 8 and recovery 5 from the order's tick.
		{turnEnd: 45},
		{scroll: true, turnEnd: 46},
	} {
		w, order := stampWorld(t, tc.second, tc.scroll)
		turned, turnEnd, castAt, end := false, uint32(0), uint64(0), uint32(0)
		for range 120 {
			tick := w.tick
			Step(w, order)
			order = nil
			e := &w.entities[0]
			if e.Turning() {
				if !turned {
					turned, turnEnd = true, e.ActionClock.End
				}
				e.ActionClock.End = unset
				continue
			}
			if turned && castAt == 0 && e.ActionClock.End != unset {
				castAt, end = tick, e.ActionClock.End
			}
		}
		if !turned || turnEnd != tc.turnEnd || castAt != tc.castAt {
			t.Errorf("second %v scroll %v: turned %v, stamp after the turn's start %d, cast stamp at tick %d; want %d, at tick %d",
				tc.second, tc.scroll, turned, turnEnd, castAt, tc.turnEnd, tc.castAt)
			continue
		}
		if castAt == 0 {
			continue
		}
		e := w.entities[0]
		if end != uint32(castAt) || e.regenerationRate(castAt+81) != 3 || e.regenerationRate(castAt+80) != 1 {
			t.Errorf("second %v scroll %v: cast stamp end %d at tick %d, rate %d at +81, %d at +80",
				tc.second, tc.scroll, end, castAt, e.regenerationRate(castAt+81), e.regenerationRate(castAt+80))
		}
	}
}

// stampWorld is a mage at (8,8) facing away from a hostile unit at (8,10),
// ordered to cast Fire Arrow at it from its book or from a scroll.
func stampWorld(t *testing.T, second, scroll bool) (*World, []Command) {
	t.Helper()
	mage := spMage(1, 8, 8, 30, 100, 100, 1<<1)
	mage.Owner, mage.Speed, mage.AttackCharge, mage.AttackRelax = SelfSlot, 30, 8, 4
	mage.RotationSpeed, mage.Facing, mage.DesiredFacing, mage.ManaRegenPeriod = 4, 0, 0, 50
	victim := spEnt(2, 8, 10)
	victim.Owner, victim.HP, victim.MaxHP = 2, 500, 500
	rel := engRel(t, [3]uint32{1, 2, relationHostile}, [3]uint32{2, 1, relationHostile})
	arrow := hlArrow()
	arrow.Delivery, arrow.EffectSpeed = 2, 200
	stock := []Stock{{ID: 1, ItemInstances: []ItemInstance{{Code: 0x0e06, Kind: 4, Price: 100, Effects: []ItemEffect{{Kind: 41, Operand: uint32(arrow.ID) | 50<<16}}}}}}
	w, err := NewStockedSpelledWorld(7, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{mage, victim}, nil, rel, nil, stock, []SpellRule{arrow})
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	w.entities[0].ActionClock = ActionClock{Known: true, End: 1 << 30}
	if scroll {
		return w, []Command{UseScroll(1, 0, 2)}
	}
	return w, []Command{Cast(1, 2, 1)}
}

// A flight transport's wait: the second game measures every flight row by the
// Euclidean distance over its speed, the first game only the rows that carry
// speedFlight. Lightning keeps its fixed 10 in both.
func TestFlightWaitByGame(t *testing.T) {
	arrow := SpellRule{ID: 1, Delivery: 2, EffectSpeed: 200}
	lightning := SpellRule{ID: 10, Arm: 13, Delivery: 2, EffectSpeed: 60}
	for _, tc := range []struct {
		name          string
		script        *Script
		diagonal, row uint16
	}{
		// 7 cells on both axes: isqrt(2 * 1792^2) = 2534, / 200 = 12.
		{"second", &Script{dialect: ScriptROM2}, 12, 8},
		// The first game keeps 7 Chebyshev cells: 1792 / 200 = 8.
		{"first", nil, 8, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := casterWorldOf(t, false, tc.script)
			at := func(r SpellRule, x, y int32) uint16 {
				return w.spellDeliveryRemaining(spellDelivery{Rule: r, FromX: 10, FromY: 10, X: x, Y: y})
			}
			if got := at(arrow, 17, 17); got != tc.diagonal {
				t.Fatalf("diagonal wait %d, want %d", got, tc.diagonal)
			}
			if got := at(arrow, 17, 10); got != tc.row {
				t.Fatalf("straight wait %d, want %d", got, tc.row)
			}
			if got := at(lightning, 17, 17); got != 10 {
				t.Fatalf("Lightning wait %d, want 10", got)
			}
		})
	}
}

// A tower's reach is the square of side 2R + 1 around its record cell, and the
// spell's Max Range does not cut it (R2-ENGINE-1560).
func TestBuildingCasterReachIsTheRecordSquare(t *testing.T) {
	for _, tc := range []struct {
		x, y int32
		cast bool
	}{{13, 13, true}, {7, 13, true}, {14, 13, false}, {13, 6, false}} {
		w := casterWorld(t, false)
		w.spells[0].MaxRange = 1
		w.entities[0].X, w.entities[0].Y = tc.x, tc.y
		w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
		w.buildingCasterPass()
		if got := len(w.ScriptCasts()) == 1; got != tc.cast {
			t.Fatalf("unit at %d,%d: cast %v, want %v", tc.x, tc.y, got, tc.cast)
		}
	}
}

// The mana restoration effect moves a mage's modifier and leaves a unit with
// no mana pool unchanged (R2-ENGINE-1564).
func TestManaRestorationEffectReachesOnlyAMage(t *testing.T) {
	for _, maxMana := range []int32{0, 100} {
		_, landed, ok := effectLanding(Entity{MaxMana: maxMana}, EffectManaRegeneration, 50)
		want := int32(0)
		if maxMana > 0 {
			want = 50
		}
		if !ok || landed != want {
			t.Fatalf("max mana %d: landed %d ok %v, want %d", maxMana, landed, ok, want)
		}
	}
}
