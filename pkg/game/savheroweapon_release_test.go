package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// heroWeaponWords is the hero's to-hit word and damage base and spread bytes
// in UD4, and own weight U8E, from a written SAV.
type heroWeaponWords struct{ toHit, base, spread, own int32 }

func savedHeroWeaponWords(t *testing.T, raw []byte, hero sim.EntityID) heroWeaponWords {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatalf("the save carries no action continuation: %v", err)
	}
	var got []heroWeaponWords
	for _, row := range a.Bindings {
		if row.Structure || row.Missing || row.ID != hero || int(row.Object) > len(doc.Objects) {
			continue
		}
		r := &doc.Objects[row.Object-1]
		ud4, err := savedActorRaw(r, "UD4", 64)
		if r.Class != "Human" || err != nil {
			continue
		}
		own, _ := savedStructureValue(r, "U8E")
		got = append(got, heroWeaponWords{int32(int16(binary.LittleEndian.Uint16(ud4[0x12:]))), int32(ud4[0x20]), int32(ud4[0x21]), int32(int16(own))})
	}
	if len(got) != 1 {
		t.Fatalf("the save holds %d Human records bound to the hero, want one", len(got))
	}
	return got[0]
}

// wornWeaponWords is what the worn weapon adds to the modifier block under the
// original's equip arithmetic: its to-hit word and damage pair, plus to-hit
// Effects (kind 12) and damage-base Effects (kind 49) (HERO-EQUIP-017,
// ITEM-EFFKEY-147). The zero item gives the bare hand's zero words.
func wornWeaponWords(item sim.ItemInstance, own int32) heroWeaponWords {
	q := item.SourceEquipment
	w := heroWeaponWords{int32(int16(binary.LittleEndian.Uint16(q.Attack[:]))), int32(q.Attack[14]), int32(q.Attack[15]), own}
	for _, e := range item.Effects {
		switch e.Kind {
		case 12:
			w.toHit += int32(e.Operand)
		case 49:
			w.base += int32(e.Operand)
		}
	}
	return w
}

// weaponSwap is a campaign hero on mission 20 wearing a sword that carries
// to-hit and damage Effects and holding a second weapon as a map or an
// original SAV holds it: its code, without constructed operands.
type weaponSwap struct {
	f                       *FrontEnd
	sword, second           sim.ItemInstance
	swordWords, secondWords heroWeaponWords
	bareWords               heroWeaponWords
}

func openWeaponSwap(t *testing.T) weaponSwap {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	sword := mapload.SourceConstructedItem(sim.PlainItem(0x6164), f.Table)
	sword.Effects = []sim.ItemEffect{{Kind: 12, Operand: 4}, {Kind: 49, Operand: 3}}
	second := mapload.SourceConstructedItem(sim.PlainItem(0x104), f.Table)
	if sword.SourceEquipment.Class != sim.SourceWeapon || second.SourceEquipment.Class != sim.SourceWeapon || !sword.WeightPresent || !second.WeightPresent {
		t.Fatal("the installed table does not construct both weapons")
	}
	party := f.ChargenParty(ui.ChargenResult{Name: "swap", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	party[0].WornItems[0], party[0].Worn[0] = sword, sword.Code
	plain := sim.PlainItem(second.Code)
	party[0].Carry = &mapload.Carry{Equipped: party[0].Worn, EquippedItems: party[0].WornItems, Items: []uint16{plain.Code}, ItemInstances: []sim.ItemInstance{plain}}
	rest := int32(0)
	for slot, item := range party[0].WornItems {
		if slot != 0 && item.Code != 0 {
			rest += int32(mapload.SourceConstructedItem(item, f.Table).Weight)
		}
	}
	app := f.App("weapon swap")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	s := weaponSwap{f: f, sword: sword, second: second,
		swordWords: wornWeaponWords(sword, rest+int32(sword.Weight)), secondWords: wornWeaponWords(second, rest+int32(second.Weight)),
		bareWords: wornWeaponWords(sim.ItemInstance{}, rest)}
	if s.swordWords.toHit == s.secondWords.toHit && s.swordWords.base == s.secondWords.base {
		t.Fatal("control: the two weapons do not discriminate")
	}
	return s
}

func (s weaponSwap) save(t *testing.T, f *FrontEnd, label string, want heroWeaponWords) []byte {
	t.Helper()
	raw, stage, err := writerCensusSave(f, true, t.TempDir())
	if err != nil {
		t.Fatal(label, stage, err)
	}
	if got := savedHeroWeaponWords(t, raw, f.live.mission.ids[0]); got != want {
		t.Fatalf("%s: SAVE wrote %+v, want %+v", label, got, want)
	}
	return raw
}

// swap equips the carried weapon with the code, or removes the weapon for
// code 0, through the inventory route, and runs four ticks.
func (s weaponSwap) swap(t *testing.T, f *FrontEnd, code uint16) {
	t.Helper()
	hero := f.live.mission.ids[0]
	f.live.switchInventorySubject(uint32(hero))
	if code == 0 {
		f.live.enqueueUnequip(0)
	} else {
		stacks, _ := f.live.world.CarriedStacks(hero)
		index := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == code })
		if index < 0 {
			t.Fatalf("the hero carries no %#x", code)
		}
		f.live.enqueueEquip(index)
	}
	for range 4 {
		f.live.tick()
	}
	if worn, _ := f.live.world.Equipped(hero); worn[0] != code {
		t.Fatalf("slot 1 holds %#x after the swap, want %#x", worn[0], code)
	}
}

// TestReleaseHeroWeaponSwapWritesTheWornWeapon: each equip and removal on the
// inventory route, then a SAVE, writes the modifier words of the weapon then
// worn and the worn weight as own weight (DIV-3358, DIV-3359); a LOAD of that
// SAVE keeps the words, and the next swap on the loaded game writes the
// sword's words again. The unchanged SAVE writes the sword's words.
func TestReleaseHeroWeaponSwapWritesTheWornWeapon(t *testing.T) {
	s := openWeaponSwap(t)
	s.save(t, s.f, "unchanged", s.swordWords)
	s.swap(t, s.f, s.second.Code)
	raw := s.save(t, s.f, "second weapon", s.secondWords)
	s.swap(t, s.f, 0)
	s.save(t, s.f, "removal", s.bareWords)

	path := filepath.Join(t.TempDir(), "swap.sav")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold := loadAreaContinuation(t, path)
	s.save(t, cold, "LOAD of the second weapon", s.secondWords)
	s.swap(t, cold, s.sword.Code)
	s.save(t, cold, "sword after LOAD", s.swordWords)
}

// TestReleaseUnknownWeaponArmFormsFromTheWornSet: a hero whose weapon-arm
// modifier bytes are unknown, as a record the engine cannot attribute holds
// them, writes the worn set's words after the next swap and after the
// removal, not the retained bytes (DIV-3359).
func TestReleaseUnknownWeaponArmFormsFromTheWornSet(t *testing.T) {
	s := openWeaponSwap(t)
	hero := s.f.live.mission.ids[0]
	e, _ := s.f.live.world.Entity(hero)
	basis := e.NativeBasis
	if !basis.ModifierPresent {
		t.Fatal("the hero carries no native modifier block")
	}
	binary.LittleEndian.PutUint16(basis.Modifier[0x12:], 40)
	basis.Modifier[0x20], basis.Modifier[0x21] = 50, 60
	for _, n := range []int{0x12, 0x13, 0x20, 0x21} {
		basis.ModifierKnown &^= uint64(1) << n
	}
	if err := s.f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: hero, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
	s.swap(t, s.f, s.second.Code)
	s.save(t, s.f, "second weapon", s.secondWords)
	s.swap(t, s.f, 0)
	s.save(t, s.f, "removal", s.bareWords)
}
