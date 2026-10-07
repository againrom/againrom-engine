package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseNativeCityItemConstructorFieldsAndResources(t *testing.T) {
	f := releaseFront(t)
	// Fixed staff code selects non-default shape1/material8. The same
	// installed constructor also resolves the first shield and chargen armor.
	codes := []uint16{0x812d, 0x0201}
	member := f.ChargenParty(ui.ChargenResult{Name: "Item Fields", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}})[0]
	for slot, item := range mapload.MemberItemEquipment(member, f.Table) {
		if slot > 1 && item.Code != 0 {
			codes = append(codes, item.Code)
		}
	}
	classes := map[string]bool{}
	for _, code := range codes {
		item := mapload.SourceConstructedItem(mapload.ItemInstanceFromCode(code, f.Table), f.Table)
		obj, err := nativeCityItemObject(item, 123, 456, f.Table)
		if err != nil {
			t.Fatal(err)
		}
		classes[obj.Class] = true
		fields := obj.Item.Fields
		// Read the installed table operands directly. Neither expected field
		// index nor width is obtained from the native-city writer helper.
		shape, material, kind, row := int(code>>5&7), int(code>>12), int(code>>8&15), int(code&31)
		product := f.Table.Materials.EntryDoubles(material)[8] * f.Table.Shapes.EntryDoubles(shape)[8]
		want48 := uint16(int32(math.Trunc(product)))
		if fields[5] != byte(shape) || fields[6] != byte(material) || binary.LittleEndian.Uint16(fields[7:9]) != want48 {
			t.Fatalf("code%04x Item+45/+46/+48=%x, want %d/%d/%d from %g", code, fields[5:9], shape, material, want48, product)
		}
		if obj.Item.Token[16] != byte(row) {
			t.Fatalf("code%04x definition row=%d, want%d", code, obj.Item.Token[16], row)
		}
		// ITEM-APPEAR-023 rebuilds from the separate constructor bytes. Use
		// that tuple to address the real icon without ItemCode path helpers.
		addr := fmt.Sprintf("graphics/inventory/%02d%02d%d%02d.16a", fields[6], kind, fields[5], row)
		icon, err := loadItemIcon(f.Archives.Containers, addr)
		if err != nil || icon == nil || icon.Bounds().Empty() {
			t.Fatalf("code%04x constructor tuple cannot resolve %s: %v", code, addr, err)
		}
		t.Logf("code%04x class=%s shape=%d material=%d F48=%d resource=%s", code, obj.Class, fields[5], fields[6], want48, addr)
	}
	for _, class := range []string{"Weapon", "Shield", "Armor"} {
		if !classes[class] {
			t.Fatalf("real constructor population missed %s", class)
		}
	}
}

func TestReleaseNativeCitySavedWeaponOwnsSpell(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Spell Fields", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	f.addChapterCompanions(f.Town.Chapter())
	hero := f.Carried[0]
	weapon := mapload.MemberItemEquipment(hero, f.Table)[0]
	var id uint8
	for _, effect := range weapon.Effects {
		if effect.Kind == 41 {
			id = uint8(effect.Operand)
			break
		}
	}
	if id == 0 || weapon.Code == 0 {
		t.Fatal("generated mage lacks its real spell weapon")
	}
	p := f.Table.Spells.EntryParams(int(id))
	want := []byte{id, byte(p[6]), 0, byte(p[1]), byte(p[1] >> 8)}
	if p[18] == 1 {
		want[2] = 1
	}
	store := SaveStore{Dir: t.TempDir()}
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ordinary mage native writer: %v", err)
	}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("ordinary mage SAVE=%q err=%v, want native SAV", name, err)
	}
	payload, err := os.ReadFile(filepath.Join(store.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	prov, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	doc := prov.Data()
	matched := false
	for _, obj := range doc.Objects {
		if obj.Unit == nil || obj.Unit.Reference74 == 0 {
			continue
		}
		item := doc.Objects[obj.Unit.Reference74-1].Item
		if item == nil || binary.LittleEndian.Uint16(item.Fields[:2]) != weapon.Code {
			continue
		}
		if item.WeaponExtra == 0 {
			t.Fatal("saved real mage Weapon lacks its owned Spell")
		}
		spell := doc.Objects[item.WeaponExtra-1].Spell
		if spell == nil || !bytes.Equal(spell.Fields[:5], want) {
			t.Fatalf("saved owned Spell=%+v, want literal row fields%x", spell, want)
		}
		for _, ref := range obj.Unit.Spells {
			if ref == item.WeaponExtra {
				t.Fatal("weapon Spell aliases a spellbook object")
			}
		}
		if len(item.Effects) != len(weapon.Effects) {
			t.Fatalf("saved weapon effects=%v, want%d", item.Effects, len(weapon.Effects))
		}
		for i, ref := range item.Effects {
			v := doc.Objects[ref-1].Effect
			if v == nil || v.Fields[0] != weapon.Effects[i].Kind || v.Fields[1] != weapon.Effects[i].Mode ||
				binary.LittleEndian.Uint32(v.Fields[2:6]) != weapon.Effects[i].Operand {
				t.Fatalf("saved ordered Effect%d=%+v", i, v)
			}
		}
		matched = true
	}
	if !matched {
		t.Fatalf("written SAV has no generated weapon code%04x", weapon.Code)
	}
	// The separate equip/unequip boundary is also exercised against actual
	// installed operands: carrying the same fresh item preserves its Effects.
	member := mapload.PartyMember{CarriedItems: []sim.ItemInstance{weapon}}
	var unit sav.CityUnitData
	seq := 0
	objects, err := nativeCityAttachItems(nil, &unit, member, f.Table, 123, &seq)
	if err != nil {
		t.Fatal(err)
	}
	if item := objects[unit.Container[0]-1].Item; item.WeaponExtra != 0 || len(item.Effects) != len(weapon.Effects) {
		t.Fatalf("unequipped real weapon Spell/effects=%d/%v", item.WeaponExtra, item.Effects)
	}
}
