package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/sim"
)

const ownerDarknessSave = "C:/Users/Eugen/againrom/review/hotfix-dark-light/owner-dark-game0464.sav"

// darknessCells counts the cells the current effects paint with spell and the
// cells the window's lighting plane darkens.
func darknessCells(f *FrontEnd, spell uint16) (painted, darkened int) {
	effects := f.live.world.CellEffects()
	for _, e := range effects {
		if e.Spell == spell {
			painted += len(e.Cells)
		}
	}
	mw := &mapWorld{world: f.live.world}
	for _, c := range spellLightingCells(mw.cellEffectsByArm(), f.live.world.Bounds()) {
		if c.Terrain == darknessBrightness {
			darkened++
		}
	}
	return painted, darkened
}

func loadOwnerDarknessSave(t *testing.T) (*FrontEnd, SaveStore) {
	t.Helper()
	if m, err := base.Detect(os.Getenv("AGAINROM_ASSETS"), nil); err == nil && m.Profile.HasFirstData() {
		t.Skip("the owner's Darkness save is a second-game save; the root is a first-game install")
	}
	raw, err := os.ReadFile(ownerDarknessSave)
	if err != nil {
		t.Skip("the owner's Darkness save is absent")
	}
	store := SaveStore{Dir: t.TempDir()}
	const name = "game0464.sav"
	if err := os.WriteFile(filepath.Join(store.Dir, name), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return deadPatrolLoad(t, store, name), store
}

// A second-game SAV whose profile places no player formation block takes the
// held-document LOAD. Its saved Darkness area keeps its cells, the lighting
// plane darkens them, and a SAVE of the loaded game LOADs to the same effects.
func TestReleaseSecondHeldDocumentDarknessAreaSurvivesLoad(t *testing.T) {
	f, store := loadOwnerDarknessSave(t)
	const darkness, want = 14, 57
	if painted, darkened := darknessCells(f, darkness); painted != want || darkened != want {
		t.Fatalf("LOAD holds %d Darkness cells and darkens %d, want %d", painted, darkened, want)
	}
	if refusal := f.live.mission.state.OriginalSaveReport().WorldEffectsRefusal; refusal != "" {
		t.Fatalf("saved world effects left unbound: %s", refusal)
	}
	before := f.live.world.CellEffects()
	// The mission is already won; answer its panel with Continue so a SAVE is admitted.
	f.live.mission.open, f.live.mission.delayedVictory, f.live.mission.victoryTaken = false, true, false
	name, _ := deadPatrolSave(t, f, store)
	g := deadPatrolLoad(t, store, name)
	if !reflect.DeepEqual(g.live.world.CellEffects(), before) {
		t.Fatal("SAVE and LOAD of the loaded game changed the cell effects")
	}
}

// A Darkness cast after the held-document LOAD lays its own cells beside the
// loaded area, and the cast area ends with its spell.
func TestReleaseSecondHeldDocumentDarknessCastAfterLoad(t *testing.T) {
	f, _ := loadOwnerDarknessSave(t)
	const darkness = 14
	w := f.live.world
	var anchor sim.Entity
	for _, e := range w.Entities() {
		if e.Owner == sim.SelfSlot && e.Alive() {
			anchor = e
			break
		}
	}
	mage := sim.Entity{HP: 100, MaxHP: 100, Owner: sim.SelfSlot, TokenSize: 1, Mind: 50, Mana: 500, MaxMana: 500,
		KnownSpells: 1 << darkness, AttackCharge: 4, AttackRelax: 2, Book: sim.Spellbook{State: sim.BookPresent}}
	mage.Book.Slots[darkness-1] = sim.BookSpell{Range: 10, ManaCost: 1}
	id, err := w.CheatSummon(mage, nil, [sim.EquipSlots]sim.ItemInstance{}, anchor.X, anchor.Y)
	if err != nil {
		t.Fatal(err)
	}
	count := func() (areas, cells int) {
		for _, e := range w.CellEffects() {
			if e.Spell == darkness {
				areas, cells = areas+1, cells+len(e.Cells)
			}
		}
		return
	}
	known, loaded := count()
	caster, _ := w.Entity(id)
	f.live.attackOrCast(uint32(id), 0, darkness, int(caster.X)+3, int(caster.Y), true)
	for i := 0; ; i++ {
		if areas, _ := count(); areas > known {
			break
		}
		if i > 256 {
			t.Fatalf("mage %d never laid a Darkness area", id)
		}
		f.live.tick()
	}
	if _, cells := count(); cells <= loaded {
		t.Fatalf("the cast holds no cells beyond the loaded %d", loaded)
	}
	for i := 0; ; i++ {
		if areas, cells := count(); areas == known && cells == loaded {
			break
		}
		if i > 4096 {
			t.Fatalf("the cast Darkness area outlives its spell after %d ticks", i)
		}
		f.live.tick()
	}
}
