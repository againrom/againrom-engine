package game

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// currentTown is the player's route to a town reached by a mission return:
// chargen, mission 10 (m10), a mission SAVE in mission 20, a LOAD, the rest
// of mission 20 (m20) and its finish.
func currentTown(t *testing.T, m10, m20 func(f *FrontEnd, hero sim.EntityID)) *FrontEnd {
	t.Helper()
	f := loadedMission20(t, m10)
	if m20 != nil {
		m20(f, equipmentReturnHero(t, f))
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 20: %v", err)
	}
	return f
}

// currentTownShop opens the merchant's room on one party member.
func currentTownShop(f *FrontEnd, member string) *townScreen {
	s := f.TownScreen().(*townScreen)
	s.room, s.packBase = roomShop, 0
	for i, p := range s.shopParty() {
		if p.ID == member {
			s.shopMember = i
		}
	}
	return s
}

// currentTownPackCell is the shop strip's cell index of code in the open
// member's pack; the strip's first element is the money element.
func currentTownPackCell(t *testing.T, s *townScreen, code uint16) int {
	t.Helper()
	i := slices.IndexFunc(s.shopPackStacks(), func(st sim.ItemStack) bool { return st.Code == code })
	if i < 0 {
		t.Fatalf("the pack holds no %#x", code)
	}
	return i + 1
}

// currentTownSave makes the ordinary town SAVE, which must write SAV.
func currentTownSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("town SAVE: %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("town SAVE wrote %q, want an original-format SAV", name)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// currentTownReload is a cold LOAD of a town SAV in a fresh front end.
func currentTownReload(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	if _, town, err := load("town.sav"); err != nil || !town {
		t.Fatalf("town LOAD of the written SAV: town=%v err=%v", town, err)
	}
	return g
}

// memberItemCodes is a member's worn and pack item codes as the town shows
// them.
func memberItemCodes(p mapload.PartyMember) (worn [sim.EquipSlots]uint16, pack []uint16) {
	for i, item := range mapload.MemberItemEquipment(p, nil) {
		worn[i] = item.Code
	}
	for _, item := range mapload.MemberCarriedItems(p, nil) {
		pack = append(pack, item.Code)
	}
	return worn, pack
}

func currentTownMember(t *testing.T, f *FrontEnd, id string) (worn [sim.EquipSlots]uint16, pack []uint16) {
	t.Helper()
	for _, p := range f.Carried {
		if p.ID == id {
			return memberItemCodes(p)
		}
	}
	t.Fatalf("the town has no member %s", id)
	return worn, pack
}

// currentTownBuy buys one unit of code from the room's shelf.
func currentTownBuy(t *testing.T, s *townScreen, room int, code uint16) {
	t.Helper()
	s.chooseRoomShelf(room)
	i := slices.IndexFunc(s.sess.Shop.Shelf(s.shopShelf), func(item ShopItem) bool { return uint16(item.Code) == code })
	if i < 0 {
		t.Fatalf("the shelf has no %#x", code)
	}
	s.shopFromShelf(i, false)
	if action := s.shopBuy(); !strings.HasPrefix(action.Msg, "bought") {
		t.Fatalf("buy %#x: %q", code, action.Msg)
	}
}

// TestReleaseTownEquipAfterAReturnSavesAsSAV: after the return, the hero
// wears the two-handed sword 0x1106 from his pack in the merchant's room.
// The town SAV reloads with it held and the sword 0x103 in the pack, and the
// next action wears 0x103 again.
func TestReleaseTownEquipAfterAReturnSavesAsSAV(t *testing.T) {
	f := currentTown(t, func(f *FrontEnd, hero sim.EntityID) { equipmentReturnPick(t, f, hero, 20, 65) }, nil)
	s := currentTownShop(f, "hero")
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, 0x1106)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if worn, _ := currentTownMember(t, f, "hero"); worn[0] != 0x1106 {
		t.Fatalf("hero holds %#x after wearing 0x1106", worn[0])
	}
	g := currentTownReload(t, currentTownSave(t, f))
	worn, pack := currentTownMember(t, g, "hero")
	if worn[0] != 0x1106 || !slices.Contains(pack, 0x103) {
		t.Fatalf("reloaded hero weapon %#x, pack %#x; want 0x1106 held and 0x103 in the pack", worn[0], pack)
	}
	s = currentTownShop(g, "hero")
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, 0x103)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if worn, _ := currentTownMember(t, g, "hero"); worn[0] != 0x103 {
		t.Fatalf("next action: hero holds %#x after wearing 0x103", worn[0])
	}
}

// TestReleasePackChangeAndTownSaleAfterAReturnSaveAsSAV: the hero carries
// the sword 0x1106 in from mission 10, after the LOAD picks up the armour
// 0xf72d at (24,39) into his pack, and after the return sells 0x1106. The
// town SAV reloads with that pack and purse, and the next action buys a
// sword.
func TestReleasePackChangeAndTownSaleAfterAReturnSaveAsSAV(t *testing.T) {
	f := currentTown(t, func(f *FrontEnd, hero sim.EntityID) { equipmentReturnPick(t, f, hero, 20, 65) },
		func(f *FrontEnd, hero sim.EntityID) { equipmentReturnPick(t, f, hero, 24, 39) })
	s := currentTownShop(f, "hero")
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, 0x1106)})
	before := f.Town.Gold()
	if action := s.shopSell(); !strings.HasPrefix(action.Msg, "he pays") || f.Town.Gold() <= before {
		t.Fatalf("sell 0x1106: %q", action.Msg)
	}
	gold := f.Town.Gold()
	g := currentTownReload(t, currentTownSave(t, f))
	_, pack := currentTownMember(t, g, "hero")
	if slices.Contains(pack, 0x1106) || !slices.Contains(pack, 0xf72d) || g.Town.Gold() != gold {
		t.Fatalf("reloaded hero pack %#x, gold %d; want 0xf72d without 0x1106 and gold %d", pack, g.Town.Gold(), gold)
	}
	currentTownBuy(t, currentTownShop(g, "hero"), roomWeapons, 0x1102)
	if _, pack := currentTownMember(t, g, "hero"); !slices.Contains(pack, 0x1102) {
		t.Fatalf("next action: the bought 0x1102 is not in the pack %#x", pack)
	}
}

// TestReleaseCompanionTownUnequipAfterAReturnSavesAsSAV: after the return,
// the companion takes off her armour 0xf82b into her pack. The town SAV
// reloads with the slot empty and the armour carried, and the next action
// wears it again.
func TestReleaseCompanionTownUnequipAfterAReturnSavesAsSAV(t *testing.T) {
	f := currentTown(t, nil, nil)
	s := currentTownShop(f, "npc:22")
	if action := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlDoll, Index: 7}); action.Msg != "off, into the pack" {
		t.Fatalf("take off 0xf82b: %q", action.Msg)
	}
	g := currentTownReload(t, currentTownSave(t, f))
	worn, pack := currentTownMember(t, g, "npc:22")
	if worn[7] != 0 || !slices.Contains(pack, 0xf82b) {
		t.Fatalf("reloaded companion slot 8 %#x, pack %#x; want empty and 0xf82b carried", worn[7], pack)
	}
	s = currentTownShop(g, "npc:22")
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, 0xf82b)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if worn, _ := currentTownMember(t, g, "npc:22"); worn[7] != 0xf82b {
		t.Fatalf("next action: companion slot 8 holds %#x after wearing 0xf82b", worn[7])
	}
}

// TestReleaseTownPurchaseAfterAReturnSavesAsSAV: after the return the hero
// buys the sword 0x1102. The town SAV reloads with it in his pack and the
// purse it left, and the next action wears it.
func TestReleaseTownPurchaseAfterAReturnSavesAsSAV(t *testing.T) {
	f := currentTown(t, nil, nil)
	currentTownBuy(t, currentTownShop(f, "hero"), roomWeapons, 0x1102)
	gold := f.Town.Gold()
	g := currentTownReload(t, currentTownSave(t, f))
	if _, pack := currentTownMember(t, g, "hero"); !slices.Contains(pack, 0x1102) || g.Town.Gold() != gold {
		t.Fatalf("reloaded hero pack %#x, gold %d; want 0x1102 and gold %d", pack, g.Town.Gold(), gold)
	}
	s := currentTownShop(g, "hero")
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, 0x1102)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if worn, _ := currentTownMember(t, g, "hero"); worn[0] != 0x1102 {
		t.Fatalf("next action: hero holds %#x after wearing 0x1102", worn[0])
	}
}
