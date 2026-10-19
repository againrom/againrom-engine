package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// heldWeaponSpell is what a written SAV holds for an actor's held Weapon: the
// first castSpell Effect's spell and power, and the WeaponSpell's S08 and key
// (zero when the Weapon carries none).
type heldWeaponSpell struct {
	effect, power, spell, key uint32
}

func savedHeldWeaponSpell(t *testing.T, raw []byte, actor sim.EntityID) heldWeaponSpell {
	t.Helper()
	doc := mustDecodeDocument(t, raw)
	r := savedActorRecord(t, &doc, actor)
	held, _ := savedObjectRefs(r, "HeldWeapon")
	if len(held) != 1 || held[0] == 0 || int(held[0]) > len(doc.Objects) || doc.Objects[held[0]-1].Class != "Weapon" {
		t.Fatalf("actor %d holds no Weapon record", actor)
	}
	weapon := &doc.Objects[held[0]-1]
	var got heldWeaponSpell
	effects, _ := savedObjectRefs(weapon, "Effects")
	for _, e := range effects {
		if e == 0 || int(e) > len(doc.Objects) {
			continue
		}
		if kind, _ := savedStructureValue(&doc.Objects[e-1], "E3C"); kind == 41 {
			operand, _ := savedStructureValue(&doc.Objects[e-1], "E40")
			got.effect, got.power = operand&0xff, operand>>16
			break
		}
	}
	if spell, _ := savedObjectRefs(weapon, "WeaponSpell"); len(spell) == 1 && spell[0] != 0 && int(spell[0]) <= len(doc.Objects) {
		got.spell, _ = savedStructureValue(&doc.Objects[spell[0]-1], "S08")
		got.key, _ = savedStructureValue(&doc.Objects[spell[0]-1], "This")
	}
	return got
}

// savedActorRecord is the record a SAV's action continuation binds to actor.
func savedActorRecord(t *testing.T, doc *sav.DocumentData, actor sim.EntityID) *sav.DocumentRecordData {
	t.Helper()
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		t.Fatalf("the save carries no action continuation: %v", err)
	}
	for _, row := range a.Bindings {
		if !row.Structure && !row.Missing && row.ID == actor && row.Object != 0 && int(row.Object) <= len(doc.Objects) {
			return &doc.Objects[row.Object-1]
		}
	}
	t.Fatalf("the save binds no record to actor %d", actor)
	return nil
}

// savedWeaponBorrow is the U64 key a SAV writes for an actor whose U68 borrows
// its held Weapon, or zero.
func savedWeaponBorrow(t *testing.T, raw []byte, actor sim.EntityID) uint32 {
	t.Helper()
	doc := mustDecodeDocument(t, raw)
	r := savedActorRecord(t, &doc, actor)
	borrow, _ := savedObjectRefs(r, "U68")
	held, _ := savedObjectRefs(r, "HeldWeapon")
	if len(borrow) != 1 || len(held) != 1 || borrow[0] == 0 || borrow[0] != held[0] {
		return 0
	}
	key, _ := savedStructureValue(r, "U64")
	return key
}

func mustDecodeDocument(t *testing.T, raw []byte) sav.DocumentData {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// nearestHostile is the living actor hostile to actor nearest to it.
func nearestHostile(t *testing.T, f *FrontEnd, actor sim.EntityID) sim.EntityID {
	t.Helper()
	e, _ := f.live.world.Entity(actor)
	var best sim.EntityID
	bestDistance := int64(-1)
	for _, other := range f.live.world.Entities() {
		if !other.Alive() || other.OffMap || !f.live.world.Relations().Hostile(e.Owner, other.Owner) {
			continue
		}
		dx, dy := int64(other.X-e.X), int64(other.Y-e.Y)
		if d := dx*dx + dy*dy; bestDistance < 0 || d < bestDistance {
			best, bestDistance = other.ID, d
		}
	}
	if best == 0 {
		t.Fatal("no hostile actor stands on the map")
	}
	return best
}

// staffSpellLoad loads AGAINROM_STAFF_SPELL_SAVE, an original SAV in which a
// party mage carries a spell weapon whose record holds no WeaponSpell, from a
// temp copy, and returns that mage and the carried stack's index.
func staffSpellLoad(t *testing.T) (*FrontEnd, sim.EntityID, int) {
	t.Helper()
	src := os.Getenv("AGAINROM_STAFF_SPELL_SAVE")
	if src == "" {
		t.Skip("AGAINROM_STAFF_SPELL_SAVE is not set: the witness needs an original SAV whose party mage carries a spell weapon without its Spell")
	}
	if releaseFront(t).Archives.Game() != base.GameROM1 {
		t.Skip("the witness SAV is a first-game save")
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "staff.sav")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := loadAreaContinuation(t, path)
	for _, id := range f.live.mission.ids {
		e, ok := f.live.world.Entity(id)
		if !ok || !e.Alive() || e.MaxMana <= 0 {
			continue
		}
		stacks, _ := f.live.world.CarriedStacks(id)
		index := slices.IndexFunc(stacks, func(s sim.ItemStack) bool {
			_, _, cast := s.Instance().CastSpell()
			return cast && s.SourceEquipment.Class == sim.SourceWeapon && !s.SourceEquipment.Spell.Present
		})
		if index >= 0 {
			return f, id, index
		}
	}
	t.Fatal("no living party mage carries a spell weapon without its Spell")
	return nil, 0, 0
}

// staffSpellMission opens mission 20 with a generated mage who holds no
// weapon and carries a staff whose castSpell Effect names Prismatic Spray at
// power 20, as a plain item: the form an item has in a session this engine
// started.
func staffSpellMission(t *testing.T) (*FrontEnd, sim.EntityID, int) {
	t.Helper()
	f := releaseFront(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the first game's SAVE builds the Spell; the second game writes the World's")
	}
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Staff spell", Choices: []int{0, 1, 0}, Stats: []int{30, 30, 40, 40}})
	if party[0].Worn[0] == 0 {
		t.Fatal("the generated mage holds no staff")
	}
	plain := sim.PlainItem(party[0].Worn[0])
	plain.Effects = []sim.ItemEffect{{Kind: 41, Operand: 14 | 20<<16}}
	party[0].WornItems[0], party[0].Worn[0] = sim.ItemInstance{}, 0
	party[0].Carry = &mapload.Carry{Equipped: party[0].Worn, EquippedItems: party[0].WornItems, Items: []uint16{plain.Code}, ItemInstances: []sim.ItemInstance{plain}}
	a := f.App("staff spell")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	for k := 0; k < 16 && f.live.mission.open; k++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	mage := f.live.mission.ids[0]
	stacks, _ := f.live.world.CarriedStacks(mage)
	index := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == plain.Code && len(s.Effects) == 1 })
	if index < 0 {
		t.Fatal("the mage does not carry the staff")
	}
	return f, mage, index
}

// equipStaff wears the carried staff at index through the inventory route,
// runs four ticks and returns its castSpell spell and power.
func equipStaff(t *testing.T, f *FrontEnd, mage sim.EntityID, index int) (uint16, int32) {
	t.Helper()
	stacks, _ := f.live.world.CarriedStacks(mage)
	staff := stacks[index].Instance()
	spell, power, _ := staff.CastSpell()
	f.live.switchInventorySubject(uint32(mage))
	f.live.enqueueEquip(index)
	for range 4 {
		f.live.tick()
	}
	if worn, _ := f.live.world.Equipped(mage); worn[0] != staff.Code {
		t.Fatalf("slot 1 holds %#x after the equip, want %#x", worn[0], staff.Code)
	}
	return spell, power
}

// staffSpellSaves SAVEs the equipped mage and checks the held Weapon's
// Effect and Spell. A cold LOAD holds the Spell, and the mage's next attack
// on a hostile is a cast of that spell at that power. A SAVE in that cast
// borrows the Weapon in U68 and names its Spell in U64, and a cold LOAD of it
// continues the cast to its end.
func staffSpellSaves(t *testing.T, f *FrontEnd, mage sim.EntityID, spell uint16, power int32) {
	t.Helper()
	raw, stage, err := writerCensusSave(f, true, t.TempDir())
	if err != nil {
		t.Fatal(stage, err)
	}
	want := heldWeaponSpell{effect: uint32(uint8(spell)), power: uint32(uint16(power)), spell: uint32(uint8(spell))}
	got := savedHeldWeaponSpell(t, raw, mage)
	if got.effect != want.effect || got.power != want.power || got.spell != want.spell || got.key == 0 {
		t.Fatalf("SAVE wrote the held Weapon as %+v, want Effect and Spell %d at power %d", got, spell, power)
	}
	path := filepath.Join(t.TempDir(), "equipped.sav")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold := loadAreaContinuation(t, path)
	worn, _ := cold.live.world.EquippedItems(mage)
	e, _ := cold.live.world.Entity(mage)
	if !worn[0].SourceEquipment.Spell.Present || uint16(worn[0].SourceEquipment.Spell.ID) != uint16(uint8(spell)) || e.WeaponSpell != spell || e.WeaponSpellLevel != power {
		t.Fatalf("cold LOAD holds Spell %+v and casts %d at %d, want %d at %d", worn[0].SourceEquipment.Spell, e.WeaponSpell, e.WeaponSpellLevel, spell, power)
	}

	victim := nearestHostile(t, cold, mage)
	cold.live.strike(uint32(mage), uint32(victim))
	casting := false
	for range 600 {
		cold.live.tick()
		if e, _ := cold.live.world.Entity(mage); e.HasAttackTarget && e.AttackTarget == victim && e.AttackPhase == sim.AttackCasting {
			casting = true
			break
		}
	}
	if !casting {
		t.Fatal("the mage's attack after LOAD never became a cast")
	}
	raw, stage, err = writerCensusSave(cold, true, t.TempDir())
	if err != nil {
		t.Fatal(stage, err)
	}
	if bad := castsWithoutSource(mustDecodeDocument(t, raw)); len(bad) != 0 {
		t.Fatalf("SAVE in the weapon cast: %v", bad)
	}
	borrow := savedWeaponBorrow(t, raw, mage)
	if held := savedHeldWeaponSpell(t, raw, mage); borrow == 0 || borrow != held.key || held.spell != want.spell {
		t.Fatalf("SAVE in the cast writes U64 %#x, want the held Weapon's Spell %#x (S08 %d)", borrow, held.key, held.spell)
	}
	path = filepath.Join(t.TempDir(), "casting.sav")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	again := loadAreaContinuation(t, path)
	if e, _ := again.live.world.Entity(mage); !e.HasAttackTarget || e.AttackTarget != victim || e.AttackPhase != sim.AttackCasting || e.WeaponSpell != spell {
		t.Fatalf("cold LOAD of the cast holds %+v", e)
	}
	for range 120 {
		again.live.tick()
		if e, _ := again.live.world.Entity(mage); e.AttackPhase != sim.AttackCasting {
			return
		}
	}
	t.Fatal("the loaded cast never completed")
}

// TestReleaseWornSpellWeaponSavesItsSpell: a generated mage on mission 20
// who holds no weapon carries a plain staff without a Spell, the control,
// and equips it through the inventory route. The worn Weapon holds a Spell
// naming its castSpell Effect's spell (ITEM-SPELLMOVE-132), and SAVE writes
// it beside the Effect's power. The cold LOAD, the cast after it, and the
// SAVE and LOAD inside that cast follow (staffSpellSaves).
func TestReleaseWornSpellWeaponSavesItsSpell(t *testing.T) {
	f, mage, index := staffSpellMission(t)
	stacks, _ := f.live.world.CarriedStacks(mage)
	if stacks[index].SourceEquipment.Spell.Present || stacks[index].SourceEquipment.Class != 0 {
		t.Fatalf("control: the carried staff already holds %+v", stacks[index].SourceEquipment)
	}
	spell, power := equipStaff(t, f, mage, index)
	worn, _ := f.live.world.EquippedItems(mage)
	if got := worn[0].SourceEquipment.Spell; !got.Present || uint16(got.ID) != uint16(uint8(spell)) {
		t.Fatalf("the equip left the worn staff with Spell %+v", got)
	}
	staffSpellSaves(t, f, mage, spell, power)
}

// TestReleaseOriginalCarriedSpellWeaponSavesItsSpell: a party mage of an
// original SAV equips a carried spell weapon whose record holds no Spell. The
// original actor's equip builds the Spell (ITEM-SPELLMOVE-132); SAVE, the
// cold LOAD, the cast, and the SAVE and LOAD inside it keep it.
func TestReleaseOriginalCarriedSpellWeaponSavesItsSpell(t *testing.T) {
	f, mage, index := staffSpellLoad(t)
	spell, power := equipStaff(t, f, mage, index)
	staffSpellSaves(t, f, mage, spell, power)
}

// townCompanionStaff arrives in the first town with a generated fighter, takes
// the chapter's companions, and returns the first companion who wears a spell
// weapon. stored selects the second game's rule for the join.
func townCompanionStaff(t *testing.T, stored bool) (*FrontEnd, int) {
	t.Helper()
	f := releaseFront(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the first game's town join builds the Spell; the second game keeps the row's")
	}
	reachabilityWalkArrive(t, f, "Staff companion")
	func() {
		firstGameFiles.storedWeaponSpell = stored
		defer func() { firstGameFiles.storedWeaponSpell = false }()
		f.addChapterCompanions(f.Town.Chapter())
	}()
	for i, p := range f.Carried {
		if _, _, ok := p.WornItems[0].CastSpell(); ok && p.CompanionNPC != 0 {
			return f, i
		}
	}
	t.Fatal("no companion of the first town wears a spell weapon")
	return nil, 0
}

// TestReleaseTownCompanionWearsItsStaffSpell: a companion the town constructs
// from his row wears his staff with the Spell the template equip builds from
// its castSpell Effect and the Spells row (ITEM-203), and the town's SAVE and
// cold LOAD keep that Weapon as held (ITEM-212). With the build switched off
// the joined staff holds none.
func TestReleaseTownCompanionWearsItsStaffSpell(t *testing.T) {
	control, i := townCompanionStaff(t, true)
	if control.Carried[i].WornItems[0].SourceEquipment.Spell.Present {
		t.Fatal("control: without the join build the companion's staff still holds a Spell")
	}
	f, i := townCompanionStaff(t, false)
	staff := f.Carried[i].WornItems[0]
	id, _, _ := staff.CastSpell()
	rule := mapload.SpellRules(f.Table)[uint8(id)-1]
	spell := staff.SourceEquipment.Spell
	if !spell.Present || spell.ID != uint8(id) || spell.Range != rule.MaxRange || int32(spell.ManaCost) != rule.ManaCost {
		t.Fatalf("the companion's staff holds Spell %+v for spell %d", spell, id)
	}
	raw, stage, err := writerCensusSave(f, false, t.TempDir())
	if err != nil {
		t.Fatal(stage, err)
	}
	g := releaseFront(t)
	if _, town, err := g.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("town LOAD: town=%t err=%v", town, err)
	}
	for _, p := range g.Carried {
		if p.CompanionNPC == f.Carried[i].CompanionNPC {
			if p.WornItems[0].SourceEquipment != staff.SourceEquipment {
				t.Fatalf("the town SAVE and LOAD changed the staff to %+v", p.WornItems[0].SourceEquipment)
			}
			return
		}
	}
	t.Fatal("the town LOAD lost the companion")
}
