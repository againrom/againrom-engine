package sim

import (
	"fmt"
	"reflect"
	"testing"
)

func fireballReachWorld(t *testing.T, dx, dy int32, observer, ground byte) *World {
	t.Helper()
	rule := SpellRule{ID: 2, ManaCost: 5, School: 1, MaxRange: 10, Area: true,
		Distribution: 2, Radius: 1, Damaging: true, DamageMin: 10, DamageMax: 10}
	caster := spMage(1, 32, 32, 30, 100, 100, 1<<2)
	caster.Owner, caster.ScanRange, caster.Skill[1] = 1, 6, 60
	caster.AttackCharge, caster.AttackRelax = 8, 4
	caster.Book.State = BookPresent
	caster.Book.Slots[1].ManaCost = 5
	RefreshBook(Rules{}, &caster, []SpellRule{rule})
	ally := spEnt(2, 32+dx, 31+dy)
	ally.Owner, ally.ScanRange = 1, 12
	enemy := spEnt(3, 32+dx, 32+dy)
	enemy.Owner = 2
	heights := make([]byte, 64*64)
	for i := range heights {
		heights[i] = ground
	}
	heights[32*64+32] = observer
	w, err := NewStockedSpelledWorld(91, Bounds{Width: 64, Height: 64}, ModeCanonical,
		Terrain{Height: heights}, []Entity{caster, ally, enemy}, nil, acEnemies(t), nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	for i := range w.groups {
		w.groups[i].order = orderStandGround
	}
	return w
}

func TestFireballPointUsesBookRangeAcrossTerrainHeights(t *testing.T) {
	for _, altitude := range []struct {
		name             string
		observer, ground byte
	}{{"flat", 64, 64}, {"above", 96, 32}, {"below", 32, 96}} {
		for _, p := range [][2]int32{{12, 0}, {12, 12}, {13, 0}, {13, 13}} {
			t.Run(fmt.Sprintf("%s/%d,%d", altitude.name, p[0], p[1]), func(t *testing.T) {
				w := fireballReachWorld(t, p[0], p[1], altitude.observer, altitude.ground)
				if got := SpellCharacteristicsFor(Rules{}, w.entities[0], w.spells[0]).Range; got != 12 {
					t.Fatalf("Fireball range=%d, want12", got)
				}
				before := w.Hash()
				refusal := w.BookSpellCellRefusal(1, 32+p[0], 32+p[1], 2)
				if w.Hash() != before {
					t.Fatal("Range observation mutated the world")
				}
				want := ""
				if p[0] > 12 {
					want = "cell out of range"
				}
				if refusal != want {
					t.Fatalf("refusal=%q, want%q", refusal, want)
				}
				events := spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 32 + p[0], Y: 32 + p[1], Spell: 2})
				if p[0] <= 12 {
					if len(events) != 1 || w.entities[0].Mana != 95 || w.entities[2].HP >= 100 {
						t.Fatalf("Full-range cast: events=%v mana=%d HP=%d", events, w.entities[0].Mana, w.entities[2].HP)
					}
				} else if len(events) != 0 || w.entities[0].Mana != 100 || w.entities[2].HP != 100 {
					t.Fatal("Out-of-range cast released or paid")
				}
				if w.entities[0].X != 32 || w.entities[0].Y != 32 {
					t.Fatal("Cast moved the caster")
				}
			})
		}
	}
}

func TestFireballBeyondPersonalSightSurvivesNativeCheckpoint(t *testing.T) {
	w := fireballReachWorld(t, 12, 0, 32, 96)
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, X: 44, Y: 32, Spell: 2}})
	if len(w.bookCasts) != 1 || w.entities[2].HP != 100 {
		t.Fatal("Full-range cast was not admitted with a pending windup")
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	back := new(World)
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	releases := 0
	for tick := 0; tick < 64; tick++ {
		want := StepReported(w, nil)
		got := StepReported(back, nil)
		if !reflect.DeepEqual(got, want) || w.Hash() != back.Hash() {
			t.Fatalf("Native continuation differs at tick%d", tick)
		}
		releases += len(want.Casts)
	}
	if releases != 1 || w.entities[2].HP >= 100 {
		t.Fatalf("Resumed cast: releases=%d targetHP=%d", releases, w.entities[2].HP)
	}
}
