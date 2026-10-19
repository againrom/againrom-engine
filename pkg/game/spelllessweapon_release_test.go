package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// wornGapLoad loads AGAINROM_WORN_SPELL_GAP_SAVE, a first-game SAV in which a
// party mage wears a spell weapon whose record holds no WeaponSpell,
// from a temp copy, and returns that mage.
func wornGapLoad(t *testing.T) (*FrontEnd, sim.EntityID) {
	t.Helper()
	src := os.Getenv("AGAINROM_WORN_SPELL_GAP_SAVE")
	if src == "" {
		t.Skip("AGAINROM_WORN_SPELL_GAP_SAVE is not set: the witness needs a SAV whose party mage wears a spell weapon without its Spell")
	}
	if releaseFront(t).Archives.Game() != base.GameROM1 {
		t.Skip("the witness SAV is a first-game save")
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "gap.sav")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := loadAreaContinuation(t, path)
	for _, id := range f.live.mission.ids {
		e, ok := f.live.world.Entity(id)
		worn, _ := f.live.world.EquippedItems(id)
		_, _, cast := worn[0].CastSpell()
		if ok && e.Alive() && e.MaxMana > 0 && cast && !worn[0].SourceEquipment.Spell.Present {
			return f, id
		}
	}
	t.Fatal("no living party mage wears a spell weapon without its Spell")
	return nil, 0
}

type gapAttack struct {
	casting, charging bool
	shots             []sim.SavedProjectile
}

// gapStrike orders mage to attack its nearest hostile and runs until the
// first physical wind-up has run 60 ticks or a cast starts, recording the
// projectile records built meanwhile.
func gapStrike(t *testing.T, f *FrontEnd, mage sim.EntityID) gapAttack {
	t.Helper()
	victim := nearestHostile(t, f, mage)
	seen := map[uint16]bool{}
	for _, p := range f.live.world.SavedProjectiles().Items {
		seen[p.ID] = true
	}
	var got gapAttack
	f.live.strike(uint32(mage), uint32(victim))
	after := -1
	for k := 0; k < 900 && (after < 0 || k < after+60); k++ {
		f.live.tick()
		e, _ := f.live.world.Entity(mage)
		if e.HasAttackTarget && e.AttackPhase == sim.AttackCasting {
			got.casting = true
			return got
		}
		if e.HasAttackTarget && e.AttackPhase == sim.AttackCharging && after < 0 {
			got.charging, after = true, k
		}
		for _, p := range f.live.world.SavedProjectiles().Items {
			if !seen[p.ID] {
				seen[p.ID] = true
				got.shots = append(got.shots, p)
			}
		}
	}
	return got
}

// TestReleaseALoadedSpelllessStaffAttacksWithoutACast: on the owner's case
// (Ferion's worn Prismatic Spray staff holds no Spell) the first attack after
// LOAD is no cast: a physical wind-up and a shot record with no spell and the
// class Projectile's picture (MAGIC-379). A SAVE without a re-equip writes
// the staff without its Spell (ITEM-187). Taken off, the staff holds no Spell
// in the pack and in a SAVE; worn again it holds a new one and the attack is
// its cast, the loss control.
func TestReleaseALoadedSpelllessStaffAttacksWithoutACast(t *testing.T) {
	f, mage := wornGapLoad(t)
	worn, _ := f.live.world.EquippedItems(mage)
	staff := worn[0]
	spell, power, _ := staff.CastSpell()
	raw, stage, err := writerCensusSave(f, true, t.TempDir())
	if err != nil {
		t.Fatal(stage, err)
	}
	if got := savedHeldWeaponSpell(t, raw, mage); got.spell != 0 || got.key != 0 || got.effect != uint32(uint8(spell)) {
		t.Fatalf("SAVE after LOAD wrote the held Weapon as %+v, want Effect %d and no Spell", got, spell)
	}
	e, _ := f.live.world.Entity(mage)
	picture, _, _ := f.live.classShot(f.live.spellClientClass(e.ID, e.Class))
	got := gapStrike(t, f, mage)
	if got.casting || !got.charging {
		t.Fatalf("the first attack after LOAD: %+v, want a physical wind-up", got)
	}
	shot := slices.IndexFunc(got.shots, func(p sim.SavedProjectile) bool { return p.ActionSpell == 0 && p.Picture == int32(picture) })
	if picture == 0 || shot < 0 {
		t.Fatalf("no shot with picture %d and no spell among %+v", picture, got.shots)
	}
	t.Logf("mage %d: Effect spell %d power %d; shot picture %d", mage, spell, power, picture)

	f.live.switchInventorySubject(uint32(mage))
	f.live.enqueueUnequip(0)
	for range 4 {
		f.live.tick()
	}
	stacks, _ := f.live.world.CarriedStacks(mage)
	index := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == staff.Code && sim.ItemEqual(s.Instance(), staff) })
	if index < 0 || stacks[index].SourceEquipment.Spell.Present {
		t.Fatalf("the staff taken off is not in the pack without a Spell: index %d", index)
	}
	raw, stage, err = writerCensusSave(f, true, t.TempDir())
	if err != nil {
		t.Fatal(stage, err)
	}
	path := filepath.Join(t.TempDir(), "off.sav")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold := loadAreaContinuation(t, path)
	carried, _ := cold.live.world.CarriedStacks(mage)
	if k := slices.IndexFunc(carried, func(s sim.ItemStack) bool { return s.Code == staff.Code }); k < 0 || carried[k].SourceEquipment.Spell.Present {
		t.Fatalf("the cold LOAD holds the staff taken off at %d with a Spell", k)
	}
	if again, _ := equipStaff(t, f, mage, index); again != spell {
		t.Fatalf("worn again the staff names spell %d", again)
	}
	worn, _ = f.live.world.EquippedItems(mage)
	if s := worn[0].SourceEquipment.Spell; !s.Present || uint16(s.ID) != uint16(uint8(spell)) {
		t.Fatalf("worn again the staff holds Spell %+v", s)
	}
	if control := gapStrike(t, f, mage); !control.casting {
		t.Fatalf("control: worn again the attack is %+v, want the staff's cast", control)
	}
}

// TestReleaseTownStaffTakenOffHoldsNoSpell: a companion who joins in the
// first town wears his staff with its Spell. Taken off through the shop doll,
// before and after a town SAVE and LOAD, the staff holds no Spell in the pack,
// and the town SAVE writes it without one (ITEM-187). Worn again, the town
// SAVE writes it with the Spell, the control.
func TestReleaseTownStaffTakenOffHoldsNoSpell(t *testing.T) {
	f, i := townCompanionStaff(t, false)
	staff := f.Carried[i].WornItems[0]
	if !staff.SourceEquipment.Spell.Present || mapload.HasSourceActor(f.Carried[i]) {
		t.Fatalf("control: the companion's staff holds %+v", staff.SourceEquipment.Spell)
	}
	member := f.Carried[i].CompanionNPC
	packStaff := func(f *FrontEnd) (int, int, sim.ItemStack) {
		t.Helper()
		for k, p := range f.Carried {
			if p.CompanionNPC != member {
				continue
			}
			for at, s := range cityMemberStacks(p, f.Table) {
				if s.Code == staff.Code {
					return k, at, s
				}
			}
			return k, -1, sim.ItemStack{}
		}
		t.Fatal("the companion left the party")
		return 0, -1, sim.ItemStack{}
	}
	takeOff := func(f *FrontEnd) *FrontEnd {
		t.Helper()
		k, _, _ := packStaff(f)
		screen := f.TownScreen().(*townScreen)
		screen.shopMember = k
		if a := screen.shopUnequipDoll(1); a.Msg != "off, into the pack" {
			t.Fatalf("unequip: %q", a.Msg)
		}
		if _, _, s := packStaff(f); s.Code != staff.Code || s.SourceEquipment.Spell.Present {
			t.Fatalf("the staff taken off is %+v", s.SourceEquipment)
		}
		raw, stage, err := writerCensusSave(f, false, t.TempDir())
		if err != nil {
			t.Fatal(stage, err)
		}
		g := releaseFront(t)
		if _, town, err := g.RestoreOriginal(raw); err != nil || !town {
			t.Fatalf("town LOAD: town=%t err=%v", town, err)
		}
		if _, _, s := packStaff(g); s.Code != staff.Code || s.SourceEquipment.Spell.Present {
			t.Fatalf("the town SAVE and LOAD hold the staff taken off as %+v", s.SourceEquipment)
		}
		return g
	}
	g := takeOff(f)
	screen := g.TownScreen().(*townScreen)
	if !screen.hasCityShopTopology() {
		t.Fatal("the loaded town has no city topology")
	}
	k, at, _ := packStaff(g)
	screen.shopMember = k
	if a := screen.shopEquipFromPack(at + 1 - screen.packBase); a.Msg != "worn" {
		t.Fatalf("wear: %q", a.Msg)
	}
	raw, stage, err := writerCensusSave(g, false, t.TempDir())
	if err != nil {
		t.Fatal(stage, err)
	}
	h := releaseFront(t)
	if _, town, err := h.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("town LOAD: town=%t err=%v", town, err)
	}
	for _, p := range h.Carried {
		if p.CompanionNPC == member && !p.WornItems[0].SourceEquipment.Spell.Present {
			t.Fatalf("control: worn again the staff is saved as %+v", p.WornItems[0].SourceEquipment)
		}
	}
	takeOff(h)
}
