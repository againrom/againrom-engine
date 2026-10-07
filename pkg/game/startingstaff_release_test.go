package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseFreshMageStaffUsesCompleteEffectPrice(t *testing.T) {
	f := releaseFront(t)
	res := ui.ChargenResult{
		Name:    "Initial staff price",
		Choices: []int{0, 1, 0},
		Stats:   []int{25, 25, 25, 25},
	}
	literal, ok := StartingWeaponName(true, data.SkillBlade)
	if !ok {
		t.Fatal("mage starting weapon literal unavailable")
	}
	_, effects, rejected, err := mapload.ParseItemCell(literal, f.Table)
	if err != nil || rejected != 0 || len(effects) != 1 || effects[0].Kind != 41 {
		t.Fatalf("starting literal effects = %+v rejected=%d err=%v", effects, rejected, err)
	}
	party := f.ChargenParty(res)
	if len(party) != 1 || party[0].Weapon == nil || party[0].Carry != nil ||
		party[0].WornItems[0].Code != 0 {
		t.Fatalf("fresh chargen did not reach the code-only weapon seam: %+v", party)
	}
	base := mapload.ItemInstanceFromCode(uint16(party[0].Weapon.Code), f.Table)
	complete := base.Clone()
	complete.Effects = append([]sim.ItemEffect(nil), effects...)
	complete.Price = 1659
	if base.Code != 0x810d || base.Price != 167 || effects[0].Operand != 0x000a0001 ||
		f.Table.Spells.EntryParams(1)[20] != 50 {
		t.Fatalf("installed initial staff inputs changed: base=%+v effects=%+v", base, effects)
	}
	if complete.Price <= base.Price {
		t.Fatalf("installed cast control has no positive surcharge: base=%d complete=%d", base.Price, complete.Price)
	}
	fresh := mapload.MemberItemEquipment(party[0], f.Table)[0]
	if fresh.Code != complete.Code || fresh.Kind != complete.Kind ||
		fresh.Price != complete.Price || !reflect.DeepEqual(fresh.Effects, complete.Effects) {
		t.Fatalf("fresh staff = %+v, complete current constructor = %+v", fresh, complete)
	}
	loaded := party[0]
	loaded.WornItems[0] = complete.Clone()
	loaded.WornItems[0].Price = base.Price
	before := loaded.WornItems[0].Clone()
	got := mapload.MemberItemEquipment(loaded, f.Table)[0]
	if !reflect.DeepEqual(got, before) || !reflect.DeepEqual(loaded.WornItems[0], before) {
		t.Fatalf("canonical loaded staff changed: got=%+v input=%+v want=%+v", got, loaded.WornItems[0], before)
	}

	ordinary := party[0]
	ordinary.Weapon = nil
	plain := mapload.MemberItemEquipment(ordinary, f.Table)[0]
	if plain.Code != base.Code || plain.Price != base.Price || len(plain.Effects) != 0 {
		t.Fatalf("ordinary code-only item = %+v, want base %+v", plain, base)
	}

	legacy := party[0]
	legacy.Carry = &mapload.Carry{}
	legacy.Carry.Equipped[0] = base.Code
	legacyItem := mapload.MemberItemEquipment(legacy, f.Table)[0]
	if !reflect.DeepEqual(legacyItem, sim.PlainItem(base.Code)) {
		t.Fatalf("carried legacy instance was reconstructed: %+v", legacyItem)
	}
}

func TestReleaseInitialMageStaffPriceReachesCurrentSAV(t *testing.T) {
	f := releaseFront(t)
	res := ui.ChargenResult{Name: "Initial staff SAV", Choices: []int{0, 1, 0}, Stats: []int{25, 25, 25, 25}}
	party := f.ChargenParty(res)
	if len(party) != 1 || party[0].Weapon == nil {
		t.Fatal("no fresh mage")
	}
	literal, _ := StartingWeaponName(true, data.SkillBlade)
	_, effects, rejected, err := mapload.ParseItemCell(literal, f.Table)
	if err != nil || rejected != 0 || len(effects) != 1 {
		t.Fatalf("literal parse: %+v %d %v", effects, rejected, err)
	}
	want := mapload.ItemInstanceFromCode(uint16(party[0].Weapon.Code), f.Table)
	want.Effects = append([]sim.ItemEffect(nil), effects...)
	want.Price = 1659
	if want.Code != 0x810d || effects[0].Operand != 0x000a0001 || f.Table.Spells.EntryParams(1)[20] != 50 {
		t.Fatalf("installed initial staff inputs changed: %+v", want)
	}

	app := f.App("initial staff SAV")
	if err := app.OpenMission(f.NewGameOpener(10, res)); err != nil {
		t.Fatal(err)
	}
	hero := equipmentReturnHero(t, f)
	worn, ok := f.live.world.EquippedItems(hero)
	if !ok || worn[0].Code != want.Code || worn[0].Price != want.Price ||
		!reflect.DeepEqual(worn[0].Effects, want.Effects) {
		t.Fatalf("initial live staff = %+v ok=%v, want complete price/effect %+v", worn[0], ok, want)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("ordinary SAVE = %q err=%v", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	staff, err := startingStaffRecord(doc, res.Name)
	if err != nil {
		t.Fatal(err)
	}
	if uint16(startingStaffScalar(staff, "F40")) != want.Code ||
		int32(startingStaffScalar(staff, "T1C")) != want.Price {
		t.Fatalf("SAV held staff = %+v, want code/price %+v", staff.Values, want)
	}
	refs := startingStaffRefs(staff, "Effects")
	if len(refs) != 1 || refs[0] == 0 || int(refs[0]) > len(doc.Objects) {
		t.Fatalf("SAV effects=%v", refs)
	}
	effect := doc.Objects[refs[0]-1]
	if effect.Class != "Effect" || startingStaffScalar(effect, "E3C") != 41 ||
		startingStaffScalar(effect, "E3D") != uint32(want.Effects[0].Mode) ||
		startingStaffScalar(effect, "E40") != want.Effects[0].Operand {
		t.Fatalf("SAV cast effect = %+v, want %+v", effect, want.Effects[0])
	}

	g := releaseFront(t)
	_, _, load := g.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("cold LOAD: town=%v err=%v", town, err)
	}
	if err := g.App("initial staff loaded").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	restored, ok := g.live.world.EquippedItems(equipmentReturnHero(t, g))
	if !ok || restored[0].Code != want.Code || restored[0].Price != want.Price ||
		!reflect.DeepEqual(restored[0].Effects, want.Effects) {
		t.Fatalf("cold restored staff = %+v ok=%v, want %+v", restored[0], ok, want)
	}
}

func startingStaffScalar(r sav.DocumentRecordData, key string) uint32 {
	for _, v := range r.Values {
		if v.Name == key {
			return v.Value
		}
	}
	return 0
}
func startingStaffRefs(r sav.DocumentRecordData, key string) []uint16 {
	for _, v := range r.RefSlots {
		if v.Name == key {
			return v.Objects
		}
	}
	return nil
}
func startingStaffRecord(doc sav.DocumentData, heroName string) (sav.DocumentRecordData, error) {
	found := false
	var result sav.DocumentRecordData
	for _, actor := range doc.Objects {
		if actor.Class != "Human" {
			continue
		}
		named := false
		for _, text := range actor.Texts {
			if text.Name == "Name" && text.Value == heroName {
				named = true
			}
		}
		if !named {
			continue
		}
		refs := startingStaffRefs(actor, "HeldWeapon")
		if found || len(refs) != 1 || refs[0] == 0 || int(refs[0]) > len(doc.Objects) {
			return result, fmt.Errorf("missing or ambiguous initial Human held weapon")
		}
		result, found = doc.Objects[refs[0]-1], true
	}
	if !found || result.Class != "Weapon" {
		return result, fmt.Errorf("initial Human's Weapon not found")
	}
	return result, nil
}
