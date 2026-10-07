package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The character sheet's own protection rows, while a Protection effect stands.
//
// TO CONFIRM IT WITNESSES THE FIX, delete the Protection overlay loop in
// entityDraws (pkg/game/world.go) — that is the tree exactly as it was at
// 5cee2da — and rerun: the second assertion reddens with the base value.

const (
	shW, shH   = 16, 16
	shSeed     = 0x1001
	shCaster   = sim.EntityID(1)
	shBaseFire = 20
)

// shWorld is one mage who knows a Protection row, and the target beside him.
func shWorld(t *testing.T, sched [][]sim.Command) *mapWorld {
	t.Helper()
	// A Protection row in the shipped shape: power/2 magnitude, duration mode,
	// its own SpellDuration. The caster's Mind and skill decide the power.
	rule := sim.SpellRule{ID: 5, ManaCost: 1, School: 1, MaxRange: 6, TargetsUnit: true,
		Defensive: true, SpellDuration: 4, EffectKind: sim.EffectProtectionFire,
		EffectMode: sim.EffectDuration}
	caster := sim.Entity{ID: shCaster, X: 3, Y: 3, HP: 100, MaxHP: 100,
		Mana: 100, MaxMana: 100, Mind: 60, KnownSpells: 1 << 5, TokenSize: 1,
		Protection: [5]int32{shBaseFire, 0, 0, 0, 0},
		Resistance: [5]uint8{40, 3, 20, 60, 80}}
	w, err := sim.NewStockedSpelledWorld(shSeed, sim.Bounds{Width: shW, Height: shH},
		sim.ModeCanonical, sim.Terrain{}, []sim.Entity{caster}, nil, sim.Relations{},
		nil, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	v, err := ui.NewViewer("sheet", terrain.Grid{
		Width: shW, Height: shH, Tiles: make([]uint16, shW*shH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	mw := newMapWorld(w, sched, deathBundle(), v)
	// The character the load computed, exactly as a mission start would leave
	// it: the base protections and nothing about any effect.
	mw.chars = map[sim.EntityID]ui.UnitCharacter{
		shCaster: {Known: true, Band: ui.CharacterBandPerson, Mage: true,
			Protection: [5]int{shBaseFire, 0, 0, 0, 0},
			Resistance: [5]int{1, 1, 1, 1, 1}},
	}
	return mw
}

func TestTheSheetsWeaponResistanceRowsStateTheLiveEntityValues(t *testing.T) {
	mw := shWorld(t, nil)
	want := [5]int{40, 3, 20, 60, 80}
	for _, d := range mw.entityDraws() {
		if d.ID == uint32(shCaster) {
			if d.Char.Resistance != want {
				t.Fatalf("sheet weapon resistance = %v, want live entity values %v", d.Char.Resistance, want)
			}
			return
		}
	}
	t.Fatal("the push carries no entry for the caster")
}

func TestTheSheetsProtectionRowStatesTheLiveValueWhileAnEffectStands(t *testing.T) {
	// The mage protects himself on the first tick.
	mw := shWorld(t, [][]sim.Command{{
		{Kind: sim.KindCast, Entity: shCaster, X: int32(shCaster), Y: 5},
	}})

	sheetFire := func() int {
		t.Helper()
		for _, d := range mw.entityDraws() {
			if d.ID == uint32(shCaster) {
				return d.Char.Protection[0]
			}
		}
		t.Fatal("the push carries no entry for the caster")
		return 0
	}
	if got := sheetFire(); got != shBaseFire {
		t.Fatalf("before the cast the sheet states fire protection %d, want the base %d", got, shBaseFire)
	}

	// Run until the cast's wind-up releases and the effect is attached.
	var attached int32
	for i := 0; i < 200 && attached == 0; i++ {
		mw.tick()
		for _, e := range mw.world.Entities() {
			if e.ID == shCaster {
				attached = e.Protection[0]
			}
		}
		if attached == shBaseFire {
			attached = 0
		}
	}
	if attached <= shBaseFire {
		t.Fatalf("setup: the cast never raised the caster's own fire protection (it is %d) — "+
			"the assertion below would witness nothing", attached)
	}
	if got := sheetFire(); got != int(attached) {
		t.Fatalf("with a Protection effect standing the sheet states fire protection %d, "+
			"want the entity's own %d", got, attached)
	}

	// And it comes back down when the effect expires.
	for i := 0; i < 4000; i++ {
		mw.tick()
		done := true
		for _, e := range mw.world.Entities() {
			if e.ID == shCaster && e.Protection[0] != shBaseFire {
				done = false
			}
		}
		if done {
			break
		}
	}
	if got := sheetFire(); got != shBaseFire {
		t.Fatalf("after the effect expired the sheet states fire protection %d, want the base %d",
			got, shBaseFire)
	}
}
