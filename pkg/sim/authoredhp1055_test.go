package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func TestHealBoundaryIncludesFinishableBodiesButNotMinusTenOrBuffs(t *testing.T) {
	heal := hlHeal()
	buff := SpellRule{ID: 5, Defensive: true, EffectKind: EffectProtectionFire, EffectMagnitude: 5}
	for _, tc := range []struct {
		hp   int32
		rule SpellRule
		want bool
	}{{0, heal, true}, {-1, heal, true}, {-9, heal, true}, {-10, heal, false}, {0, buff, false}, {-9, buff, false}} {
		target := spEnt(2, 3, 3)
		target.HP = tc.hp
		if got := spellTargetable(target, tc.rule); got != tc.want {
			t.Errorf("HP %d rule %d targetable = %v, want %v", tc.hp, tc.rule.ID, got, tc.want)
		}
	}

	noHealth := spEnt(2, 3, 3)
	noHealth.HP, noHealth.MaxHP = 0, 0
	if spellTargetable(noHealth, heal) {
		t.Fatal("Heal admitted a target with no health system")
	}
}

func TestAuthoredBodyRevivesOnceWithoutDroppingEquipmentAndRoundTrips(t *testing.T) {
	heal := hlHeal()
	caster := spMage(1, 2, 2, 30, 50, 50, 1<<heal.ID)
	body := spEnt(2, 3, 3)
	body.HP, body.Defence = 0, 40
	PrepareAuthoredBody(&body)
	PrepareAuthoredBody(&body)
	if body.Defence != 20 {
		t.Fatalf("authored body defence after duplicate preparation = %d, want one death-time half to 20", body.Defence)
	}

	var worn [EquipSlots]uint16
	worn[0], worn[4] = 0x0201, 0x0801
	stock := []Stock{{ID: body.ID, Items: []uint16{0x0101}, Equipped: worn}}
	w, err := NewStockedSpelledWorld(0x1055, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, []Entity{caster, body}, nil, Relations{}, nil, stock, []SpellRule{heal})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	initial := cbAt(t, w, body.ID)
	if initial.HP != 0 || initial.MaxHP != 100 || initial.Decay != DecayFallen || initial.Dwell == 0 || initial.Defence != 20 {
		t.Fatalf("initial authored body = HP %d/%d decay %d dwell %d defence %d",
			initial.HP, initial.MaxHP, initial.Decay, initial.Dwell, initial.Defence)
	}
	if len(w.Sacks()) != 0 {
		t.Fatal("authored starting equipment was dropped into a sack")
	}

	initialForm, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary initial: %v", err)
	}
	initialHash := w.Hash()
	var resumed World
	if err := resumed.UnmarshalBinary(initialForm); err != nil {
		t.Fatalf("UnmarshalBinary initial: %v", err)
	}
	if resumed.Hash() != initialHash || !bytes.Equal(initialForm, mustWorldForm1055(t, &resumed)) {
		t.Fatalf("initial save/hash drifted: %016x/%016x", initialHash, resumed.Hash())
	}
	if !reflect.DeepEqual(resumed.Stock(), w.Stock()) {
		t.Fatalf("initial stock drifted: %+v -> %+v", w.Stock(), resumed.Stock())
	}

	// Reconstructing the same already-normalised body is a session operation,
	// not a second authored death. Defence and equipment must not move again.
	rebuilt, err := NewStockedSpelledWorld(0x1055, resumed.Bounds(), ModeCanonical,
		Terrain{}, resumed.Entities(), nil, Relations{}, resumed.Sacks(), resumed.Stock(), []SpellRule{heal})
	if err != nil {
		t.Fatalf("rebuild initial body: %v", err)
	}
	if got := cbAt(t, rebuilt, body.ID).Defence; got != 20 {
		t.Fatalf("double construction halved defence again to %d", got)
	}

	beforeStock := resumed.Stock()
	if !resumed.ordinaryEffect(0, 1, heal, 0) {
		t.Fatal("Heal refused an authored HP-0 body")
	}
	revived := cbAt(t, &resumed, body.ID)
	if revived.HP <= 0 || revived.Decay != DecayNone || revived.Dwell != 0 || revived.Defence != 40 {
		t.Fatalf("revived body = HP %d decay %d dwell %d defence %d, want alive/0/0/40",
			revived.HP, revived.Decay, revived.Dwell, revived.Defence)
	}
	if len(resumed.Sacks()) != 0 || !reflect.DeepEqual(resumed.Stock(), beforeStock) {
		t.Fatalf("revival moved equipment: stock %+v -> %+v, sacks %+v",
			beforeStock, resumed.Stock(), resumed.Sacks())
	}
	if !resumed.ordinaryEffect(0, 1, heal, 0) {
		t.Fatal("second useful Heal was refused")
	}
	if got := cbAt(t, &resumed, body.ID).Defence; got != 40 {
		t.Fatalf("second Heal changed restored defence to %d", got)
	}

	healedForm := mustWorldForm1055(t, &resumed)
	healedHash := resumed.Hash()
	var healed World
	if err := healed.UnmarshalBinary(healedForm); err != nil {
		t.Fatalf("UnmarshalBinary healed: %v", err)
	}
	if healed.Hash() != healedHash || !bytes.Equal(healedForm, mustWorldForm1055(t, &healed)) {
		t.Fatalf("healed save/hash drifted: %016x/%016x", healedHash, healed.Hash())
	}
}

func TestZeroHealthIsStableUnderDecayAndNotAVIPDeath(t *testing.T) {
	body := spEnt(1, 3, 3)
	body.HP, body.DyingTime = 0, 3
	PrepareAuthoredBody(&body)
	w := spWorld(t, 0x1056, nil, body)
	for n := 0; n < 4*decayCycle; n++ {
		Step(w, nil)
	}
	got := cbAt(t, w, body.ID)
	if got.HP != 0 || got.Decay != DecayFallen || got.Dwell != 0 {
		t.Fatalf("HP-0 fixed point moved to HP %d decay %d dwell %d", got.HP, got.Decay, got.Dwell)
	}
	if scriptVIPDead(got) {
		t.Fatal("an authored HP-0 body counted as a completed VIP death")
	}
}

func TestDwellExpiredAuthoredBodyBlocksAMoverThroughRestoration(t *testing.T) {
	heal := hlHeal()
	for _, tc := range []struct {
		name    string
		restore func(*World) bool
	}{
		{"Heal", func(w *World) bool { return w.ordinaryEffect(0, 2, heal, 0) }},
		{"script health write", func(w *World) bool {
			w.setUnitProperty(2, propertyHealth, 1)
			return true
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := spMage(1, 4, 0, 30, 50, 50, 1<<heal.ID)
			mover := spEnt(2, 1, 0)
			body := spEnt(3, 2, 0)
			body.HP, body.DyingTime = 0, 1
			PrepareAuthoredBody(&body)

			w, err := NewSpelledWorld(0x1055, Bounds{Width: 5, Height: 1}, ModeCanonical,
				nil, []Entity{caster, mover, body}, nil, []SpellRule{heal})
			if err != nil {
				t.Fatalf("NewSpelledWorld: %v", err)
			}

			Step(w, nil)
			if got := occEntity(t, w, body.ID); got.HP != 0 || got.Decay != DecayFallen || got.Dwell != 0 {
				t.Fatalf("authored body after dwell = HP %d decay %d dwell %d, want 0/%d/0",
					got.HP, got.Decay, got.Dwell, DecayFallen)
			} else if !cellRecordHolds(got) {
				t.Fatal("dwell-expired restorable body left its cell-record actor slot")
			}

			Step(w, []Command{{Entity: mover.ID, X: body.X, Y: body.Y}})
			if !tc.restore(w) {
				t.Fatalf("%s refused the authored body", tc.name)
			}

			gotMover := occEntity(t, w, mover.ID)
			gotBody := occEntity(t, w, body.ID)
			if !gotBody.Alive() || gotBody.Decay != DecayNone || gotBody.Dwell != 0 {
				t.Fatalf("%s left body at HP %d decay %d dwell %d",
					tc.name, gotBody.HP, gotBody.Decay, gotBody.Dwell)
			}
			if gotMover.X == gotBody.X && gotMover.Y == gotBody.Y {
				t.Fatalf("%s restored body %d over living mover %d at (%d,%d)",
					tc.name, gotBody.ID, gotMover.ID, gotBody.X, gotBody.Y)
			}
			if gotMover.X != 1 || gotMover.Y != 0 {
				t.Fatalf("mover entered the restorable body's cell before %s: (%d,%d)",
					tc.name, gotMover.X, gotMover.Y)
			}
		})
	}
}

func mustWorldForm1055(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}
