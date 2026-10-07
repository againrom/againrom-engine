package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// recoveredCap is the Mage's Low Hat that mission 10's sack at (38,64) holds.
const recoveredCap = 0xf625

// recoveredCapShop cold LOADs a town SAV into an App and walks into the
// merchant's room through the square and the merchant's conversation.
func recoveredCapShop(t *testing.T, raw []byte) (*FrontEnd, *ui.App, SaveStore, *townScreen) {
	t.Helper()
	f, app, store := cityPotionLoadApp(t, raw)
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	shop := f.TownScreen().(*townScreen)
	for n := 0; shop.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if shop.room != roomShop || !shop.hasCityShopTopology() {
		t.Fatalf("the merchant's room is not open on the city item graph: room %v", shop.room)
	}
	return f, app, store, shop
}

// recoveredCapShow turns the member picker until it shows id.
func recoveredCapShow(t *testing.T, app *ui.App, shop *townScreen, id string) {
	t.Helper()
	for n := 0; n < 8; n++ {
		if m := shop.shopPartyMember(shop.shopMemberIndex()); m != nil && m.ID == id {
			return
		}
		cityPotionPointer(t, app, "picker_next", 0, "press", "release")
	}
	t.Fatalf("the member picker never showed %s", id)
}

// recoveredCapCell is the pack strip cell that shows the cap.
func recoveredCapCell(t *testing.T, shop *townScreen) int {
	t.Helper()
	cell := currentTownPackCell(t, shop, recoveredCap)
	if shop.packBase != 0 || cell >= len(shop.ShopScreen().Pack) {
		t.Fatalf("the cap is strip cell %d past the first page", cell)
	}
	return cell
}

func recoveredCapTap(t *testing.T, app *ui.App, target string, index int) {
	t.Helper()
	cityPotionPointer(t, app, target, index, "press", "release")
}

// recoveredCapDrawn is what the shop's message strip draws of msg:
// drawShopMessage cuts the text to 307 pixels of the shop font.
func recoveredCapDrawn(f *FrontEnd, msg string) string {
	font := f.Font.Value()
	for font != nil && len(msg) > 1 {
		if w, _ := font.Measure(msg); w <= 307 {
			break
		}
		msg = msg[:len(msg)-1]
	}
	return msg
}

// TestReleaseTownDoubleClickWearsTheRecoveredCap is the tester's route in the
// town a mission return reaches, saved as SAV and loaded cold. The hero
// carries the Mage's Low Hat 0xf625 home from mission 10's sack at (38,64).
// In the merchant's room a click puts it on the trade table, the picker turns
// to the companion npc:22, and a click on the table puts it in her pack. A
// double click on it there wears it, as the drag onto her figure does
// (ITEM-USE-112, ITEM-WEAR-057, DIV-1465). The worn cap is the pack's own
// city object, and an F2 SAVE and cold LOAD keep both members' worn and pack
// objects and codes.
func TestReleaseTownDoubleClickWearsTheRecoveredCap(t *testing.T) {
	arrived := currentTown(t, func(f *FrontEnd, hero sim.EntityID) { equipmentReturnPick(t, f, hero, 38, 64) }, nil)
	t.Logf("the return arrives on the city item graph: %v; the cap is %q", arrived.Town.cityObjects != nil, decodeInstallText(itemName(data.ItemCode(recoveredCap), arrived.Table)))
	raw := currentTownSave(t, arrived)
	for _, route := range []string{"drag", "double click"} {
		t.Run(route, func(t *testing.T) {
			f, app, store, shop := recoveredCapShop(t, raw)
			recoveredCapShow(t, app, shop, "hero")
			recoveredCapTap(t, app, "pack", recoveredCapCell(t, shop))
			if table := f.Shop.Table(); len(table) != 1 || uint16(table[0].Code) != recoveredCap || !table[0].Mine {
				t.Fatalf("the hero's click staged %+v, want the cap alone", table)
			}
			recoveredCapShow(t, app, shop, "npc:22")
			recoveredCapTap(t, app, "table", 0)
			if _, pack := currentTownMember(t, f, "npc:22"); !slices.Contains(pack, recoveredCap) || len(f.Shop.Table()) != 0 {
				t.Fatalf("the companion's pack is %#x with %d table places, want the cap handed over", pack, len(f.Shop.Table()))
			}
			root, err := cityMutationParty(f.Town.cityObjects, "npc:22")
			if err != nil {
				t.Fatal(err)
			}
			cell := recoveredCapCell(t, shop)
			if cell > len(root.Pack) {
				t.Fatalf("the companion's pack roots %v do not reach strip cell %d", root.Pack, cell)
			}
			object := root.Pack[cell-1]
			slot, ok := EquipTarget(data.ItemCode(recoveredCap), f.Table)
			if !ok || object == 0 {
				t.Fatalf("the cap has slot %d (%v) and city object %d", slot, ok, object)
			}
			if route == "drag" {
				cityPotionPointer(t, app, "pack", cell, "press")
				cityPotionPointer(t, app, "doll_box", 0, "move", "release")
			} else {
				cityPotionPointer(t, app, "pack", cell, "press", "release", "press", "release")
			}
			msg := app.HeadlessMessage()
			worn, pack := currentTownMember(t, f, "npc:22")
			root, _ = cityMutationParty(f.Town.cityObjects, "npc:22")
			if worn[slot-1] != recoveredCap || slices.Contains(pack, recoveredCap) || len(f.Shop.Table()) != 0 || root.Worn[slot-1] != object {
				t.Fatalf("%s: message %q, drawn %q; slot %d holds %#x as object %d, pack %#x, %d table places; want the cap worn as object %d and nothing staged",
					route, msg, recoveredCapDrawn(f, msg), slot, worn[slot-1], root.Worn[slot-1], pack, len(f.Shop.Table()), object)
			}
			live := f.Town.cityObjects.Clone()
			cold, _, _ := cityPotionLoadApp(t, cityRosterF2Save(t, app, store, "cap"))
			if cold.Town == nil || cold.Town.cityObjects == nil {
				t.Fatal("the cold LOAD has no city item graph")
			}
			for _, id := range []string{"hero", "npc:22"} {
				a, errLive := cityMutationParty(live, id)
				b, errCold := cityMutationParty(cold.Town.cityObjects, id)
				if errLive != nil || errCold != nil {
					t.Fatal(id, errLive, errCold)
				}
				if !slices.Equal(a.Pack, b.Pack) || a.Worn != b.Worn {
					t.Fatalf("%s objects after SAVE and cold LOAD: pack %v worn %v, want pack %v worn %v", id, b.Pack, b.Worn, a.Pack, a.Worn)
				}
				wantWorn, wantPack := currentTownMember(t, f, id)
				gotWorn, gotPack := currentTownMember(t, cold, id)
				if gotWorn != wantWorn || !slices.Equal(gotPack, wantPack) {
					t.Fatalf("%s items after SAVE and cold LOAD: worn %#x pack %#x, want worn %#x pack %#x", id, gotWorn, gotPack, wantWorn, wantPack)
				}
			}
			t.Logf("%s: %q; the cap 0xf625 is worn in slot %d as city object %d through F2 SAV and cold LOAD", route, msg, slot, object)
		})
	}
}
