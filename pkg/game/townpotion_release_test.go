package game

import (
	"encoding/binary"
	"encoding/json"
	"math/rand"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func townPotionActor(t *testing.T, raw []byte, memberID string, ticks int) (sim.Entity, []sim.ActiveEffect) {
	t.Helper()
	f, app, _ := cityLoadApp(t, raw)
	if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	if ticks > 0 {
		f.LiveAdvance(ticks)
	}
	for i, p := range f.live.mission.party {
		if p.ID == memberID {
			e, _ := f.live.entity(f.live.mission.ids[i])
			return e, f.live.world.ActiveEffects()
		}
	}
	t.Fatal("next mission omitted the member")
	return sim.Entity{}, nil
}

// townPotionOldRecord adds the timer and applied modifier an earlier build wrote.
func townPotionOldRecord(t *testing.T, raw []byte, memberID string) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("current city policy", err)
	}
	var object uint16
	for i, p := range a.Party {
		if string(p.ID) != memberID || p.City == nil {
			continue
		}
		a.Party[i].City.Potion = &sim.ActiveEffect{Target: 1, Kind: sim.EffectHealthRegeneration, Mode: sim.EffectDuration, Magnitude: 100, Remaining: 960}
		for _, binding := range a.Bindings {
			if !binding.Structure && !binding.Missing && binding.ID == p.Entity {
				object = binding.Object
			}
		}
	}
	if object == 0 {
		t.Fatal("member has no ordinary actor binding")
	}
	modifier, err := savedActorRaw(&doc.Objects[object-1], "UD4", 64)
	if err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint16(modifier[10:], binary.LittleEndian.Uint16(modifier[10:])+100)
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestReleaseTownRegenerationPotionLeavesNoEffect(t *testing.T) {
	lineage := releaseFront(t)
	lineage.Carried = lineage.NextParty()
	lineage.arriveInTown()
	control := currentTownSave(t, lineage)
	f, app, store := cityGroupsColdApp(t, control)
	t.Cleanup(app.StopAudio)
	shop := kitOwnerShop(t, f, app)
	shop.shopMember = 0
	member := shop.shopPartyMember(shop.shopMemberIndex())
	before := mapload.CloneParty([]mapload.PartyMember{*member})[0]
	var item ShopItem
	found := false
	for _, s := range shopPotionStock(f.Table, rand.New(rand.NewSource(1))) {
		if f.Table.MagicItems.EntryName(int(s.Code&0xff)) == "Potion Health Regeneration" {
			item, found = s, true
		}
	}
	if !found {
		t.Fatal("installed shop does not sell the regeneration potion")
	}
	f.Shop.shelves[ShelfBooks] = []ShopItem{item}
	f.Town.gold = 100000
	cityShopPointer(t, app, "shelf_pick", 3, "press", "release")
	gold := f.Town.Gold()
	packAt := releaseShopBuyToPack(t, app, f, uint16(item.Code))
	if err := app.HeadlessPointer("press", packAt.X, packAt.Y); err != nil {
		t.Fatal(err)
	}
	cityShopPointer(t, app, "doll_box", 0, "move", "release")
	member = shop.shopPartyMember(shop.shopMemberIndex())
	for _, st := range shop.shopPackStacks() {
		if st.Code == uint16(item.Code) {
			t.Fatal("the potion is still in the pack")
		}
	}
	if f.Town.Gold() != gold-int(item.Price) {
		t.Fatalf("gold %d -> %d, want one price %d", gold, f.Town.Gold(), item.Price)
	}
	townPotionUnchanged(t, before, *member, f.Table)
	raw := cityRosterF2Save(t, app, store, "drunk")
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if a, err := readCurrentActions(&doc); err != nil || a == nil {
		t.Fatal("current city policy", err)
	} else {
		for _, p := range a.Party {
			if p.City != nil && p.City.Potion != nil {
				t.Fatalf("city SAVE wrote a potion timer for %s: %+v", p.ID, p.City.Potion)
			}
		}
	}
	want, _ := townPotionActor(t, control, before.ID, 1)
	for name, saved := range map[string][]byte{"drunk": raw, "earlier build record": townPotionOldRecord(t, raw, before.ID)} {
		for _, ticks := range []int{0, 1} {
			got, effects := townPotionActor(t, saved, before.ID, ticks)
			for _, e := range effects {
				if e.Spell == 0 {
					t.Fatalf("%s tick %d: the next mission holds a potion effect %+v", name, ticks, e)
				}
			}
			if ticks == 1 && (got.HealthRegeneration != want.HealthRegeneration || got.ManaRegeneration != want.ManaRegeneration ||
				got.HP != want.HP || got.Mana != want.Mana || got.NativeBasis != want.NativeBasis) {
				t.Fatalf("%s: the next mission's actor differs from the undrunk control:\ngot  %+v\nwant %+v", name, got, want)
			}
		}
	}
}
