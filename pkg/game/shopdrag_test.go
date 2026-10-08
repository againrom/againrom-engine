package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// The shop's own doll moves (1005 round 2: `ITEM-CMD-007`, `ITEM-EQUIP-006`,
// `DIV-087`, `DIV-089`) driven directly, shoproom_test.go's own style over
// shopRoom and shopTable.

// shopWearableShelfIndex is a test helper: shopArmourPool combines the
// shield pool and the armour pool onto ONE shelf (shop.go's own doc), and A
// test that needs a particular wearable class searches the shelf rather than
// assuming an index; the pool deliberately mixes shields and armour.
// shopFigureArchive is a real *vfs.FS holding one base figure sheet, at the
// address memberFigure's own fallback resolves for shopRoom's single carried
// member (partyFigureDir, partyFigureFace, both zero on the fixture's
// mapload.PartyMember{}).
func shopFigureArchive(t *testing.T) *Archives {
	t.Helper()
	path := filepath.Join(t.TempDir(), GraphicsArchive)
	basePath := data.ItemFigureBasePath(partyFigureDir, partyFigureFace)
	archive := synth.Archive([]synth.File{{Path: basePath, Data: invBaseSheet()}})
	if err := os.WriteFile(path, archive, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open(%s): %v", path, err)
	}
	return &Archives{Containers: fsys}
}

// shopFigureArchiveWithWeaponLayer is shopFigureArchive plus one occupied
// slot's own figure layer, at code — item 4's own fixture:
// refreshShopDrag's weapon-fallback line (round-2 adversarial review) only
// moves a pixel if slot 1's OWN layer sheet is in the archive for the
// compositor to paint, which the base-only shopFigureArchive does not carry.
func shopFigureArchiveWithWeaponLayer(t *testing.T, code data.ItemCode) *Archives {
	t.Helper()
	path := filepath.Join(t.TempDir(), GraphicsArchive)
	basePath := data.ItemFigureBasePath(partyFigureDir, partyFigureFace)
	layerPath := data.ItemFigureLayerPath(partyFigureDir, code)
	archive := synth.Archive([]synth.File{
		{Path: basePath, Data: invBaseSheet()},
		{Path: layerPath, Data: invLayerSheetColored(0, 0, 0xff)},
	})
	if err := os.WriteFile(path, archive, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open(%s): %v", path, err)
	}
	return &Archives{Containers: fsys}
}

func shopWearableShelfIndex(t *testing.T, f *FrontEnd, shelf ShopShelf) int {
	t.Helper()
	items := f.Shop.Shelf(shelf)
	for i, it := range items {
		if slot, ok := EquipTarget(it.Code, f.Table); ok && slot != 2 {
			return i
		}
	}
	t.Fatal("the fixture drew no wearable item on this shelf")
	return -1
}

// TestShopEquipFromShelfBuysAndWearsInOneGesture is DIV-087's own act: the
// shelf item resolves a slot, the purse pays its price once, and the same
// gesture leaves it worn — never staged on the table at all.
func TestShopEquipFromShelfBuysAndWearsInOneGesture(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	item := f.Shop.Shelf(ShelfArmour)[idx]
	slot, _ := EquipTarget(item.Code, f.Table)
	// Units, not elements: the gesture buys and wears ONE unit, so a cell
	// holding more stays on the shelf with its count down by one
	// (`DIV-322`).
	before := shelfUnits(f.Shop, ShelfArmour)
	gold := f.Town.Gold()
	s.shelfBase = idx

	act := s.shopEquipFromShelf(0)
	if act.Msg == "" {
		t.Fatal("shopEquipFromShelf said nothing")
	}
	if got := s.shopWornSlots(0)[slot-1]; got != uint16(item.Code) {
		t.Errorf("worn[%d] = %d, want %d", slot-1, got, item.Code)
	}
	if got := f.Town.Gold(); got != gold-int(item.Price) {
		t.Errorf("gold = %d, want %d", got, gold-int(item.Price))
	}
	if got := shelfUnits(f.Shop, ShelfArmour); got != before-1 {
		t.Errorf("the shelf holds %d units, want %d", got, before-1)
	}
	if len(f.Shop.Table()) != 0 {
		t.Error("the shelf-to-doll gesture opened a table place")
	}
}

// TestShopEquipFromShelfRefusesWhenThePurseCannotPay is DIV-087's own
// refusal: the shelf, the worn slot and the purse all stay exactly what
// they were — the price and the wear rule are both asked before
// RemoveFromShelf is ever called.
func TestShopEquipFromShelfRefusesWhenThePurseCannotPay(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	item := f.Shop.Shelf(ShelfArmour)[idx]
	slot, _ := EquipTarget(item.Code, f.Table)
	f.Town.gold = int(item.Price) - 1
	before := len(f.Shop.Shelf(ShelfArmour))
	s.shelfBase = idx

	act := s.shopEquipFromShelf(0)
	if act.Msg != "" {
		t.Fatalf("shopEquipFromShelf = %+v, want no line for the purse refusal", act)
	}
	if s.shopWornSlots(0)[slot-1] != 0 {
		t.Error("the slot was worn despite the refusal")
	}
	if got := len(f.Shop.Shelf(ShelfArmour)); got != before {
		t.Errorf("the shelf holds %d, want %d unchanged", got, before)
	}
	if got := f.Town.Gold(); got != int(item.Price)-1 {
		t.Errorf("gold = %d, want the purse untouched", got)
	}
}

// TestShopEquipFromPackWearsAtNoCharge is spec bullet 1's own "no purchase"
// arm: the item is already the player's, so wearing it from the pack moves
// no coin — DIV-087's own charge applies to the shelf pair alone.
func TestShopEquipFromPackWearsAtNoCharge(t *testing.T) {
	f, s := shopRoom(t, nil)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	item := f.Shop.Shelf(ShelfArmour)[idx]
	slot, _ := EquipTarget(item.Code, f.Table)
	f.Carried[0].Carry.Items = []uint16{uint16(item.Code)}
	gold := f.Town.Gold()

	act := s.shopEquipFromPack(1) // strip index 1: index 0 is the money element
	if act.Msg != "worn" {
		t.Fatalf("shopEquipFromPack = %+v, want \"worn\"", act)
	}
	if got := s.shopWornSlots(0)[slot-1]; got != uint16(item.Code) {
		t.Errorf("worn[%d] = %d, want %d", slot-1, got, item.Code)
	}
	if got := f.Town.Gold(); got != gold {
		t.Errorf("gold changed from %d to %d wearing a pack item", gold, got)
	}
	if got := len(f.Carried[0].Carry.Items); got != 0 {
		t.Errorf("the pack still holds %d units after wearing its only one", got)
	}
}

// TestShopEquipFromPackRefusesTheWrongClass is shopUsable's own refusal
// (0162), restated for the drag: a row admitting the mage alone refuses the
// default party member, who is a fighter, and leaves the pack and the worn
// slot untouched.
func TestShopEquipFromPackRefusesTheWrongClass(t *testing.T) {
	f, s := shopRoom(t, nil)
	table := shopTable()
	table.Armors = shopCollection{{}, {
		name: "mage only", price: 25, slot: 5, suit: 2, // bit 1 alone: mage, not fighter
		masks: [data.ShopShapes]uint16{1 << 0},
	}}
	f.Table = table
	code := data.ComposeItemCode(0, 5, 0, 1) // material 0, shape 0: masks[0]'s own bit
	f.Carried[0].Carry.Items = []uint16{uint16(code)}

	act := s.shopEquipFromPack(1)
	if act.Msg != "" {
		t.Fatalf("shopEquipFromPack = %+v, want no line for the class refusal", act)
	}
	if s.shopWornSlots(0)[4] != 0 {
		t.Error("the slot was worn despite the class refusal")
	}
	if len(f.Carried[0].Carry.Items) != 1 {
		t.Error("the pack lost its item despite the refusal")
	}
}

// TestShopEquipDisplacesTheOldWornItemBackToThePack is equip.go's own swap
// (`World.equip`) restated over the party record: wearing a second item into
// an already-occupied slot returns the first to the pack rather than
// discarding it. Both codes are drawn from the shelf's own wearable entries,
// which the "helm" row's fixed slot column makes land on the same slot
// regardless of which shape or material either draw carries.
func TestShopEquipDisplacesTheOldWornItemBackToThePack(t *testing.T) {
	f, s := shopRoom(t, nil)
	items := f.Shop.Shelf(ShelfArmour)
	bySlot := make(map[int][]data.ItemCode)
	for _, it := range items {
		n, ok := EquipTarget(it.Code, f.Table)
		if !ok {
			continue
		}
		found := false
		for _, c := range bySlot[n] {
			if c == it.Code {
				found = true
				break
			}
		}
		if !found {
			bySlot[n] = append(bySlot[n], it.Code)
		}
	}
	var codes []data.ItemCode
	var slot int
	for n, candidates := range bySlot {
		if len(candidates) >= 2 {
			slot, codes = n, candidates[:2]
			break
		}
	}
	if len(codes) < 2 {
		t.Fatal("the fixture drew fewer than two distinct wearable codes for one slot")
	}
	first, second := codes[0], codes[1]
	f.Carried[0].Carry.Items = []uint16{uint16(first), uint16(second)}

	s.shopEquipFromPack(1) // wears first
	s.shopEquipFromPack(1) // pack now holds only second, at strip index 1

	if got := s.shopWornSlots(0)[slot-1]; got != uint16(second) {
		t.Errorf("worn[%d] = %d, want %d", slot-1, got, second)
	}
	items2 := f.Carried[0].Carry.Items
	if len(items2) != 1 || items2[0] != uint16(first) {
		t.Errorf("pack = %v, want exactly the displaced %d", items2, first)
	}
}

func TestShopRangedEquipDisplacesShield(t *testing.T) {
	shield := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 701,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}
	bow := sim.ItemInstance{Code: eqBowCode, Kind: 2, Price: 902,
		Effects: []sim.ItemEffect{{Kind: 17, Operand: 3}}}
	t.Run("ranged weapon displaces shield", func(t *testing.T) {
		f, s := shopRoom(t, nil)
		f.Table = eqDefsTable(t)
		carry := f.Carried[0].Carry
		oldSword := sim.ItemInstance{Code: eqSwordCode, Kind: 2, Price: 411}
		carry.ItemInstances = []sim.ItemInstance{bow.Clone()}
		carry.EquippedItems[0], carry.EquippedItems[1] = oldSword.Clone(), shield.Clone()
		carry.Equipped[0], carry.Equipped[1] = oldSword.Code, shield.Code
		s.composeShopFaces()
		before := s.ShopScreen().Character.Subject.Combat
		act := s.shopEquipFromPack(1)
		if act.Msg != "worn" {
			t.Fatalf("shopEquipFromPack = %+v, want worn", act)
		}
		gotWorn := s.shopWornItemSlots(0)
		if gotWorn == nil || !sim.ItemEqual(gotWorn[0], bow) || !gotWorn[1].Empty() {
			t.Fatalf("worn = %+v, want bow and no shield", gotWorn)
		}
		pack := s.shopPackItemInstances()
		if len(pack) != 2 || !sim.ItemEqual(pack[0], oldSword) || !sim.ItemEqual(pack[1], shield) {
			t.Fatalf("pack = %+v, want displaced sword and shield", pack)
		}
		after := s.ShopScreen().Character.Subject.Combat
		if after.Defence >= before.Defence {
			t.Errorf("shop card Defence = %d after removing shield, want below %d", after.Defence, before.Defence)
		}
	})
}

func TestShopShieldMaterializesCompatibleStartingWeaponFallback(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Table = eqDefsTable(t)
	member := &f.Carried[0]
	weapon := eqSword(t, f.Table)
	member.Weapon = weapon
	shield := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 701,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}
	member.Carry.ItemInstances = []sim.ItemInstance{shield.Clone()}

	if act := s.shopEquipFromPack(1); act.Msg != "worn" {
		t.Fatalf("shopEquipFromPack = %+v, want worn", act)
	}
	worn := s.shopWornItemSlots(0)
	if worn == nil || worn[0].Code != uint16(weapon.Code) || !sim.ItemEqual(worn[1], shield) ||
		worn[1].Price != shield.Price || len(s.shopPackItemInstances()) != 0 || !member.WeaponMaterialized {
		t.Fatalf("fallback shield result member=%+v worn=%+v pack=%+v", *member, worn, s.shopPackItemInstances())
	}
}

// TestShopUnequipDollTakesItOffIntoThePack is spec bullet 2's own tap:
// "pressing a worn item on the doll and releasing without moving takes it
// off, into the pack" — restated for the shop's own ShopControlDoll arm.
func TestShopUnequipDollTakesItOffIntoThePack(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(code, f.Table)
	s.shelfBase = idx
	s.shopEquipFromShelf(0)

	act := click(s, ui.ShopControlDoll, slot-1)
	if act.Msg == "" {
		t.Fatal("ShopClick on the doll said nothing")
	}
	if s.shopWornSlots(0)[slot-1] != 0 {
		t.Error("the slot is still worn after the tap")
	}
	items := f.Carried[0].Carry.Items
	if len(items) != 1 || items[0] != uint16(code) {
		t.Errorf("pack = %v, want exactly the taken-off %d", items, code)
	}
}

// TestShopUnequipDollRefusesAnEmptySlot is unequipFromWorn's own totality
// (world.go), restated for the shop: a slot with no code moves nothing.
func TestShopUnequipDollRefusesAnEmptySlot(t *testing.T) {
	f, s := shopRoom(t, nil)

	act := s.shopUnequipDoll(1)
	if act.Msg != "" {
		t.Errorf("shopUnequipDoll on an empty slot said %q, want silence", act.Msg)
	}
	if len(f.Carried[0].Carry.Items) != 0 {
		t.Error("an empty-slot unequip put something in the pack")
	}
}

// TestShopUnequipToTableStagesTheItemForSale is spec bullet 2's own "onto a
// shop grid" release for a doll origin (`DIV-089`): the item leaves the
// worn slot and lands on the table, Mine and at its own shop price, exactly
// as shopFromPack's own put would land it.
func TestShopUnequipToTableStagesTheItemForSale(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(code, f.Table)
	s.shelfBase = idx
	s.shopEquipFromShelf(0)
	price := s.shopItemPrice(code)

	act := s.shopUnequipToTable(slot)
	if act.Msg != "on the table" {
		t.Fatalf("shopUnequipToTable = %+v, want \"on the table\"", act)
	}
	if s.shopWornSlots(0)[slot-1] != 0 {
		t.Error("the slot is still worn after the table move")
	}
	table := f.Shop.Table()
	if len(table) != 1 || !table[0].Mine || table[0].Code != code || table[0].Price != price || table[0].Count != 1 {
		t.Errorf("table = %+v, want one player place of %d at %d", table, code, price)
	}
}

// TestShopUnequipToTableRefusesWhenTheTableIsFull is PutOnTable's own
// refusal (shop.go), restated for the doll origin: a full table leaves the
// slot worn rather than dropping the item on the floor of a screen that
// keeps no ground at all.
func TestShopUnequipToTableRefusesWhenTheTableIsFull(t *testing.T) {
	f, s := shopRoom(t, nil)
	for i := int32(1); i <= ShopTablePlaces; i++ {
		if !f.Shop.PutOnTable(ShopItem{Code: data.ItemCode(i), Price: 1, Count: 1}) {
			t.Fatalf("place %d was refused filling the table", i)
		}
	}
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(code, f.Table)
	s.shelfBase = idx
	s.shopEquipFromShelf(0)

	act := s.shopUnequipToTable(slot)
	if act.Msg != "" {
		t.Fatalf("shopUnequipToTable = %+v, want no line for the full-table refusal", act)
	}
	if s.shopWornSlots(0)[slot-1] != uint16(code) {
		t.Error("the slot was cleared despite the refusal")
	}
	if len(f.Shop.Table()) != ShopTablePlaces {
		t.Errorf("the table holds %d, want %d unchanged", len(f.Shop.Table()), ShopTablePlaces)
	}
}

// TestShopDragDispatchesTheFourRecognisedPairs is ShopDrag's own switch: the
// two equip pairs and the two doll-origin pairs each reach their own
// method, and every other pair — including a surface paired with itself —
// is a no-op.
func TestShopDragDispatchesTheFourRecognisedPairs(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(code, f.Table)
	s.shelfBase = idx

	act := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if act.Msg != "" || s.shopWornSlots(0)[slot-1] == uint16(code) {
		t.Fatalf("shelf-to-doll drag = %+v, worn[%d] = %d, want the item back on its shelf and nothing worn", act, slot-1, s.shopWornSlots(0)[slot-1])
	}
	s.shopWearInto(slot, uint16(code))

	act = s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1}, ui.ShopControl{Kind: ui.ShopControlPackCell})
	if act.Msg == "" || s.shopWornSlots(0)[slot-1] != 0 {
		t.Fatalf("doll-to-pack drag = %+v, worn[%d] = %d, want the slot cleared", act, slot-1, s.shopWornSlots(0)[slot-1])
	}

	s.shopWearInto(slot, uint16(code))
	beforeTable := len(f.Shop.Table())
	act = s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1}, ui.ShopControl{Kind: ui.ShopControlShelfCell})
	if act.Msg == "" || s.shopWornSlots(0)[slot-1] != 0 || len(f.Shop.Table()) != beforeTable+1 {
		t.Fatalf("doll-to-shelf drag = %+v, worn[%d] = %d, table = %d, want the slot cleared and staged on the table (not the pack)",
			act, slot-1, s.shopWornSlots(0)[slot-1], len(f.Shop.Table()))
	}

	// A pair naming no recognised combination is silent and moves nothing.
	if act := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell}, ui.ShopControl{Kind: ui.ShopControlPackCell}); act.Msg != "" {
		t.Errorf("pack-to-pack drag = %+v, want a no-op", act)
	}
	if act := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlDoll}, ui.ShopControl{Kind: ui.ShopControlDoll}); act.Msg != "" {
		t.Errorf("doll-to-doll drag = %+v, want a no-op", act)
	}
}

// A shelf object released on the doll stays on its shelf (DIV-1667).
func TestShopShelfObjectReleasedOnTheDollStaysOnItsShelf(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	s.shelfBase = idx
	shelf, gold := f.Shop.Shelf(ShelfArmour), f.Town.Gold()
	slot, _ := EquipTarget(shelf[idx].Code, f.Table)
	worn := s.shopWornSlots(0)[slot-1]

	if act := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlDoll}); act.Msg != "" {
		t.Fatalf("shelf-to-doll drag = %+v, want a no-op", act)
	}
	if !reflect.DeepEqual(f.Shop.Shelf(ShelfArmour), shelf) || f.Town.Gold() != gold || len(f.Shop.Table()) != 0 || s.shopWornSlots(0)[slot-1] != worn {
		t.Fatal("a shelf object released on the doll changed the shelf, purse, table or worn set")
	}
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlTableCell})
	if len(f.Shop.Table()) != 1 {
		t.Fatalf("control: shelf-to-table left %d table places, want 1", len(f.Shop.Table()))
	}
}

// TestShopDragDispatchesTheFourTablePairs is "the five-place grid"'s own
// dispatch witness (round-2 adversarial review, `DIV-091`): the table joined
// the drag surfaces at this review's own landing, and each of its four new
// pairs reaches the SAME model function a plain click on that cell already
// runs — shopClickShelfCell, shopFromPack and shopOffTable — never a
// second copy of the trade logic. The destination named in a table-origin
// drag does not choose shopOffTable's own home: the item's own history
// (Mine) does, which is why both TableCell destinations below are driven and
// both land on the SAME place their own Mine flag names.
func TestShopDragDispatchesTheFourTablePairs(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	s.shelfBase = idx

	// shopFromShelf's own success arm returns an empty TownAction (shoproom.go:
	// only its refusal carries a message), so the table's own new content is
	// the success signal here, not act.Msg.
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlTableCell})
	table := f.Shop.Table()
	if len(table) != 1 || table[0].Code != code || table[0].Mine {
		t.Fatalf("shelf-to-table drag left table = %+v, want the shelf's own item staged, not Mine", table)
	}

	act := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlPackCell})
	if act.Msg != "back on his shelf" || len(f.Shop.Table()) != 0 {
		t.Fatalf("table-to-pack drag (a shelf item) = %+v, table = %+v, want it returned to the shelf", act, f.Shop.Table())
	}

	f.Carried[0].Carry.Items = []uint16{uint16(code)}
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlTableCell})
	table = f.Shop.Table()
	if len(table) != 1 || !table[0].Mine || table[0].Code != code {
		t.Fatalf("pack-to-table drag left table = %+v, want the pack's own item staged, Mine", table)
	}

	act = s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlShelfCell})
	items := f.Carried[0].Carry.Items
	if act.Msg != "back in your pack" || len(f.Shop.Table()) != 0 || len(items) != 1 || items[0] != uint16(code) {
		t.Fatalf("table-to-shelf drag (a pack item) = %+v, table = %+v, pack = %v, want it returned to the pack", act, f.Shop.Table(), items)
	}
}

// TestShopEquipFromTableWearsAMinePlaceForFree is counterexample 3's own
// witness (round-2 adversarial review): the table-to-doll pair, previously
// unrecognised by ShopDrag, wears a player-owned table place at no charge
// — shopOffTable's own free return to the pack, restated for a wear.
func TestShopEquipFromTableWearsAMinePlaceForFree(t *testing.T) {
	f, s := shopRoom(t, nil)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	item := f.Shop.Shelf(ShelfArmour)[idx]
	slot, _ := EquipTarget(item.Code, f.Table)
	f.Carried[0].Carry.Items = []uint16{uint16(item.Code)}
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlTableCell})
	if len(f.Shop.Table()) != 1 || !f.Shop.Table()[0].Mine {
		t.Fatalf("fixture did not stage a Mine table place: table = %+v", f.Shop.Table())
	}
	gold := f.Town.Gold()

	act := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if act.Msg != "worn" {
		t.Fatalf("table-to-doll drag (Mine) = %+v, want \"worn\"", act)
	}
	if got := s.shopWornSlots(0)[slot-1]; got != uint16(item.Code) {
		t.Errorf("worn[%d] = %d, want %d", slot-1, got, item.Code)
	}
	if len(f.Shop.Table()) != 0 {
		t.Errorf("table still holds %d places after the wear", len(f.Shop.Table()))
	}
	if got := f.Town.Gold(); got != gold {
		t.Errorf("gold changed from %d to %d wearing back the player's own item", gold, got)
	}
}

// A merchant-owned table place released on the doll stays on the table (DIV-1668).
func TestShopTableNonMinePlaceReleasedOnTheDollReturnsToTheTable(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	item := f.Shop.Shelf(ShelfArmour)[idx]
	slot, _ := EquipTarget(item.Code, f.Table)
	s.shelfBase = idx
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlTableCell})
	if len(f.Shop.Table()) != 1 || f.Shop.Table()[0].Mine {
		t.Fatalf("fixture did not stage a non-Mine table place: table = %+v", f.Shop.Table())
	}
	gold := f.Town.Gold()
	worn := s.shopWornSlots(0)[slot-1]

	act := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if act.Msg != "" {
		t.Fatalf("table-to-doll drag (not Mine) = %+v, want a no-op", act)
	}
	if got := s.shopWornSlots(0)[slot-1]; got != worn {
		t.Errorf("worn[%d] = %d, want %d unchanged", slot-1, got, worn)
	}
	if got := f.Town.Gold(); got != gold {
		t.Errorf("gold = %d, want %d unchanged", got, gold)
	}
	if len(f.Shop.Table()) != 1 {
		t.Errorf("table holds %d places, want the staged place kept", len(f.Shop.Table()))
	}
}

// TestShopDragPackToShelfStagesOnTheTable is counterexample 5's own witness
// (round-2 adversarial review, second pass, `ITEM-CMD-007`): the
// pack-to-shelf pair, previously unrecognised, joins pack-to-table and
// stages the pack item for sale — the same act a pack-to-table drag
// already runs, not a second, unpriced one.
func TestShopDragPackToShelfStagesOnTheTable(t *testing.T) {
	f, s := shopRoom(t, nil)
	code := invWornCode(1)
	f.Carried[0].Carry.Items = []uint16{uint16(code)}
	price := s.shopItemPrice(code)

	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlShelfCell})
	table := f.Shop.Table()
	if len(table) != 1 || !table[0].Mine || table[0].Code != code || table[0].Price != price {
		t.Fatalf("pack-to-shelf drag left table = %+v, want one Mine place of %d at %d", table, code, price)
	}
	if got := f.Carried[0].Carry.Items; len(got) != 0 {
		t.Errorf("pack still holds %v after the drag staged its only unit", got)
	}
}

func TestShopDragShelfToPackStagesOnTheTable(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	s.shelfBase = idx

	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlPackCell})
	table := f.Shop.Table()
	if len(table) != 1 || table[0].Mine || table[0].Code != code {
		t.Fatalf("shelf-to-pack drag left table = %+v, want one non-Mine place of %d", table, code)
	}
	if got := f.Carried[0].Carry.Items; len(got) != 0 {
		t.Errorf("pack holds %v after a shelf-to-pack drag, want nothing inserted unpriced", got)
	}
}

// TestShopSuppressDollCachesAndRecomposesOnChange is refreshShopDrag's own
// guard (world.go's refreshDollDrag mirrored): the same slot twice costs one
// compare and no second composition, a different slot recomposes, and 0
// clears the override.
func TestShopSuppressDollCachesAndRecomposesOnChange(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Archives = shopFigureArchive(t)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(code, f.Table)
	s.shelfBase = idx
	s.shopEquipFromShelf(0)

	s.ShopSuppressDoll(slot)
	fig1, mask1 := s.shopSuppressFigure, s.shopSuppressMask
	if fig1 == nil || mask1 == nil {
		t.Fatalf("ShopSuppressDoll(%d) composed no figure or no mask", slot)
	}
	s.ShopSuppressDoll(slot)
	if s.shopSuppressFigure != fig1 {
		t.Error("the same slot recomposed a new figure instead of keeping the cached one")
	}

	s.ShopSuppressDoll(0)
	if s.shopSuppressSlot != 0 || s.shopSuppressFigure != nil || s.shopSuppressMask != nil {
		t.Error("ShopSuppressDoll(0) did not clear the override")
	}
}

// TestRefreshShopDragAppliesTheSameWeaponFallbackAsComposeShopFaces is item
// 4's own witness (round-2 adversarial review): a member whose Worn[0] is
// empty but who carries a Weapon — rosterTemplate's own shape
// (pkg/mapload/spawn.go) — composes slot 1's own layer into the SUPPRESSED
// figure exactly as composeShopFaces composes it into the ordinary one.
// Suppressing a DIFFERENT slot (never 1 itself) exercises the fallback
// without the suppression's own eq.SetCode(want, 0) undoing it on the same
// slot it just filled.
func TestRefreshShopDragAppliesTheSameWeaponFallbackAsComposeShopFaces(t *testing.T) {
	f, s := shopRoom(t, nil)
	weaponCode := invWornCode(1)
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}
	f.Archives = shopFigureArchiveWithWeaponLayer(t, weaponCode)

	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	shelfCode := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(shelfCode, f.Table)
	if slot == 1 {
		t.Fatal("the fixture drew a wearable landing on slot 1 itself; this test needs a DIFFERENT slot suppressed to isolate the fallback")
	}
	s.shelfBase = idx
	s.shopEquipFromShelf(0)

	s.ShopSuppressDoll(slot)
	if s.shopSuppressMask == nil {
		t.Fatal("ShopSuppressDoll composed no mask")
	}
	found := false
	for _, b := range s.shopSuppressMask.Slot {
		if b == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("the suppressed figure carries no slot-1 pixel; refreshShopDrag dropped the Weapon fallback composeShopFaces applies")
	}
}

// TestShopEquippedCodeAppliesTheSameWeaponFallbackAsComposeShopFaces is
// counterexample 1's own read-side witness (round-2 adversarial review): a
// member whose Worn/Carry.Equipped slot 1 is empty but who carries a Weapon
// — rosterTemplate's own shape (pkg/mapload/spawn.go) — answers
// shopEquippedCode for slot 1 exactly as composeShopFaces already draws the
// figure and mask for it, and ShopScreen's own SlotInfo line follows. Before
// the fix, shopEquippedCode(i,1) answered (0,true) over this member and
// SlotInfo[0] stayed empty despite the doll drawing a weapon.
func TestShopEquippedCodeAppliesTheSameWeaponFallbackAsComposeShopFaces(t *testing.T) {
	f, s := shopRoom(t, nil)
	weaponCode := invWornCode(1)
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}

	code, ok := s.shopEquippedCode(0, 1)
	if !ok || code != weaponCode {
		t.Fatalf("shopEquippedCode(0,1) = %v, %v, want %v, true", code, ok, weaponCode)
	}
	v := s.ShopScreen()
	if len(v.SlotInfo[0]) == 0 {
		t.Error("ShopScreen SlotInfo[0] is empty over a fallback-worn slot 1")
	}
}

// TestShopWeaponFallbackCodeRefusesWhenThePackAlreadyHoldsTheCode is
// counterexample 2 (round-2 adversarial review, third pass):
// PartyMember.WeaponMaterialized (round-2 adversarial review, fifth pass) is
// what survives CarryParty/OwnParty across a mid-mission unequip, not a live
// scan of the pack: a member who took the starting weapon off during the
// mission returns to the shop with the bit already true, the state a real
// mission-side occupation (rearm's own tick, world.go) leaves behind.
// shopWeaponFallbackCode must answer ok=false here: the member has already
// materialized the weapon, so nothing should stand in for slot 1, regardless
// of what the pack currently holds.
func TestShopWeaponFallbackCodeRefusesWhenThePackAlreadyHoldsTheCode(t *testing.T) {
	weaponCode := invWornCode(1)
	f, s := shopRoom(t, []uint16{uint16(weaponCode)})
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}
	f.Carried[0].WeaponMaterialized = true
	if code, ok := s.shopWeaponFallbackCode(0); ok {
		t.Fatalf("shopWeaponFallbackCode = %d, true, want ok=false: the member already materialized this weapon", code)
	}
}

func TestShopFallbackCannotAppendASecondWeaponToThePack(t *testing.T) {
	weaponCode := invWornCode(1)
	f, s := shopRoom(t, []uint16{uint16(weaponCode)})
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}
	f.Carried[0].WeaponMaterialized = true

	if act := s.shopUnequipDoll(1); act.Msg != "" {
		t.Fatalf("shopUnequipDoll(1) = %+v, want a no-op for an already-materialized weapon", act)
	}
	if got := f.Carried[0].Carry.Items; len(got) != 1 || got[0] != uint16(weaponCode) {
		t.Fatalf("pack = %v, want one copy of %d", got, weaponCode)
	}
}

func TestShopFallbackCannotStageASecondSellableWeapon(t *testing.T) {
	weaponCode := invWornCode(1)
	f, s := shopRoom(t, []uint16{uint16(weaponCode)})
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}
	f.Carried[0].WeaponMaterialized = true

	if act := s.shopUnequipToTable(1); act.Msg != "" {
		t.Fatalf("shopUnequipToTable(1) = %+v, want a no-op for an already-materialized weapon", act)
	}
	if got := len(f.Shop.Table()); got != 0 {
		t.Fatalf("table places = %d, want 0: no sellable copy may be created", got)
	}
	if got := f.Carried[0].Carry.Items; len(got) != 1 || got[0] != uint16(weaponCode) {
		t.Fatalf("pack = %v, want one copy of %d", got, weaponCode)
	}
}

// The fixture seeds Carry.Items with the starting weapon's own code directly
// (bypassing shopUnequipDoll/shopUnequipToTable), so it also sets
// WeaponMaterialized true to match the only reachable state that combination
// stands for: the fallback already resolved, the weapon sitting in the pack
// exactly as an actual unequip would have left it (round-2 adversarial
// review, fifth pass). Leaving WeaponMaterialized false here would put the
// fixture in a state shopUnequipDoll/shopUnequipToTable never produce —
// Carry.Items holding the code while the fallback is still armed — and
// shopWearInto would read the still-armed fallback as a second, distinct
// "old" item and push a duplicate back into the pack.
func TestShopEquippingThePackedStartingWeaponDoesNotDisplaceADuplicate(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomWeapons)
	items := f.Shop.Shelf(ShelfWeapons)
	idx := -1
	for i, it := range items {
		if slot, ok := EquipTarget(it.Code, f.Table); ok && slot == 1 {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("fixture has no slot-1 weapon")
	}
	weaponCode := items[idx].Code
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}
	f.Carried[0].Carry.Items = []uint16{uint16(weaponCode)}
	f.Carried[0].WeaponMaterialized = true

	if act := s.shopEquipFromPack(1); act.Msg != "worn" {
		t.Fatalf("shopEquipFromPack(1) = %+v, want worn", act)
	}
	if got := s.shopWornSlots(0)[0]; got != uint16(weaponCode) {
		t.Fatalf("slot 1 = %d, want %d", got, weaponCode)
	}
	if got := len(f.Carried[0].Carry.Items); got != 0 {
		t.Fatalf("pack count = %d, want 0: the equipped unit must not remain as a displaced duplicate", got)
	}
}

// TestShopEquippedCodeDoesNotDuplicateAPostMissionUnequippedWeapon restates
// counterexample 2 through shopEquippedCode, the same door
// TestShopEquippedCodeAppliesTheSameWeaponFallbackAsComposeShopFaces above
// proves the fallback fires through: for a member whose starting weapon was
// already materialized (a mission-side unequip, WeaponMaterialized already
// true), shopEquippedCode(0,1) must answer 0 (the array's own empty cell, ok
// staying true because n=1 is a well-formed query — the same contract
// TestShopEquippedCodeAppliesTheSameWeaponFallbackAsComposeShopFaces
// exercises for the case where the fallback SHOULD fire), not the fallback
// code a second time.
func TestShopEquippedCodeDoesNotDuplicateAPostMissionUnequippedWeapon(t *testing.T) {
	weaponCode := invWornCode(1)
	f, s := shopRoom(t, []uint16{uint16(weaponCode)})
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}
	f.Carried[0].WeaponMaterialized = true

	code, ok := s.shopEquippedCode(0, 1)
	if !ok || code != 0 {
		t.Fatalf("shopEquippedCode(0,1) = %v, %v, want 0, true: the member already materialized this weapon, not the fallback", code, ok)
	}
}

// TestShopUnequipDollTakesOffAFallbackWeaponAndClearsIt is counterexample
// 1's own write-side witness for shopUnequipDoll: a member whose slot 1
// reads through the Weapon fallback (the raw array's own cell empty) can
// take it off the same as any ordinarily-worn item.
// member.WeaponMaterialized (round-2 adversarial review, fifth pass) is what
// the write sets true — the persisted history bit, not a clearing of
// member.Weapon itself, which stays as the permanent record of the starting
// weapon — so a second unequip on the same slot correctly finds nothing
// left to take off.
func TestShopUnequipDollTakesOffAFallbackWeaponAndClearsIt(t *testing.T) {
	f, s := shopRoom(t, nil)
	weaponCode := invWornCode(1)
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}

	act := s.shopUnequipDoll(1)
	if act.Msg != "off, into the pack" {
		t.Fatalf("shopUnequipDoll(1) over a fallback weapon = %+v, want the ordinary success message", act)
	}
	if !f.Carried[0].WeaponMaterialized {
		t.Error("member.WeaponMaterialized still false after the fallback weapon was taken off")
	}
	items := f.Carried[0].Carry.Items
	if len(items) != 1 || items[0] != uint16(weaponCode) {
		t.Errorf("pack = %v, want exactly the taken-off %d", items, weaponCode)
	}

	act2 := s.shopUnequipDoll(1)
	if act2.Msg != "" {
		t.Errorf("a second unequip on the now-empty slot said %q, want silence", act2.Msg)
	}
}

// TestShopUnequipToTableTakesOffAFallbackWeapon restates the counterexample
// for the doll-to-table release (shopUnequipToTable): the fallback weapon
// lands on the table at its own shop price, and member.WeaponMaterialized is
// set the same way shopUnequipDoll sets it.
func TestShopUnequipToTableTakesOffAFallbackWeapon(t *testing.T) {
	f, s := shopRoom(t, nil)
	weaponCode := invWornCode(1)
	f.Carried[0].Weapon = &data.Weapon{Code: weaponCode}
	price := s.shopItemPrice(weaponCode)

	act := s.shopUnequipToTable(1)
	if act.Msg != "on the table" {
		t.Fatalf("shopUnequipToTable(1) over a fallback weapon = %+v, want \"on the table\"", act)
	}
	if !f.Carried[0].WeaponMaterialized {
		t.Error("member.WeaponMaterialized still false after the fallback weapon moved to the table")
	}
	table := f.Shop.Table()
	if len(table) != 1 || !table[0].Mine || table[0].Code != weaponCode || table[0].Price != price {
		t.Errorf("table = %+v, want one player place of %d at %d", table, weaponCode, price)
	}
}

// TestShopWearIntoDisplacesAFallbackWeaponRatherThanLosingIt is
// counterexample 1's own data-loss witness: wearing a NEW item into slot 1
// while the OLD slot-1 content exists only through the Weapon fallback (the
// raw array reading empty) must still displace the old weapon to the pack,
// not silently drop it — shopWearInto's own "old" used to come from the
// raw array alone, which read 0 here and so never grew the pack.
// member.Weapon itself stays as the permanent starting-weapon record
// (round-2 adversarial review, fifth pass); WeaponMaterialized is the field
// the wear sets.
func TestShopWearIntoDisplacesAFallbackWeaponRatherThanLosingIt(t *testing.T) {
	f, s := shopRoom(t, nil)
	oldCode := invWornCode(1)
	f.Carried[0].Weapon = &data.Weapon{Code: oldCode}

	click(s, ui.ShopControlShelfPick, roomWeapons)
	items := f.Shop.Shelf(ShelfWeapons)
	idx := -1
	for i, it := range items {
		if slot, ok := EquipTarget(it.Code, f.Table); ok && slot == 1 {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("the fixture drew no slot-1 wearable on the weapon shelf")
	}
	newCode := items[idx].Code
	s.shelfBase = idx
	s.shopEquipFromShelf(0)

	if !f.Carried[0].WeaponMaterialized {
		t.Error("member.WeaponMaterialized still false after a new weapon was worn into the fallback slot")
	}
	if got := s.shopWornSlots(0)[0]; got != uint16(newCode) {
		t.Errorf("worn[0] = %d, want the newly worn %d", got, newCode)
	}
	packItems := f.Carried[0].Carry.Items
	if len(packItems) != 1 || packItems[0] != uint16(oldCode) {
		t.Errorf("pack = %v, want exactly the displaced old weapon %d", packItems, oldCode)
	}
}

// TestShopScreenNeverSubstitutesAnotherMembersSuppressedFigure is item 3's
// own witness (round-2 adversarial review): shopStepMember does NOT call
// composeShopFaces (its own doc, and world.go's dollSubject precedent this
// guard restates), so a suppression armed for the shown member and a picker
// step to a DIFFERENT one is constructible directly against the model, even
// though the two gestures share one physical button and no input path this
// build drives can produce the sequence. Without the shopSuppressMember
// guard, member 1's screen would show member 0's stale suppressed figure —
// `DIV-085`'s own disagreement between a hit test and a drawn pixel,
// restated for two different party members instead of two pixels of one.
func TestShopScreenNeverSubstitutesAnotherMembersSuppressedFigure(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Carried = append(f.Carried, mapload.PartyMember{Carry: &mapload.Carry{}})
	f.Archives = shopFigureArchive(t)
	s.composeShopFaces()
	member1Figure := s.shopFigure(1)
	if member1Figure == nil {
		t.Fatal("member 1's own ordinary figure composed nothing")
	}

	s.ShopSuppressDoll(1) // shown member is still 0 here
	if s.shopSuppressFigure == nil {
		t.Fatal("ShopSuppressDoll composed no figure")
	}

	s.shopStepMember(1) // moves the picker to member 1, without touching the suppression cache

	v := s.ShopScreen()
	if v.Figure != member1Figure {
		t.Errorf("ShopScreen for member 1 = %p, want member 1's own ordinary figure %p, not member 0's stale suppressed one", v.Figure, member1Figure)
	}
	if v.Figure == s.shopSuppressFigure {
		t.Error("ShopScreen substituted member 0's suppressed figure for member 1")
	}
}

// TestShopScreenShowsTheSuppressedFigureOverTheOrdinaryOne is ShopScreen's
// own substitution: while a slot is suppressed for the shown member, the
// view carries the suppressed figure and mask, world.go's own dollSubject
// restated for the shop.
func TestShopScreenShowsTheSuppressedFigureOverTheOrdinaryOne(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Archives = shopFigureArchive(t)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(code, f.Table)
	s.shelfBase = idx
	s.shopEquipFromShelf(0)
	ordinaryView := s.ShopScreen()
	ordinary := ordinaryView.Figure

	s.ShopSuppressDoll(slot)
	v := s.ShopScreen()
	if v.Figure == ordinary || v.Figure != s.shopSuppressFigure {
		t.Error("ShopScreen did not substitute the suppressed figure")
	}
	if v.SlotMask != s.shopSuppressMask {
		t.Error("ShopScreen did not substitute the suppressed mask alongside the figure")
	}
	if v.Character.Figure != v.Figure {
		t.Error("the shared character pane kept the dressed figure while the shop doll preview was undressed")
	}
	if !reflect.DeepEqual(v.Character.Subject, ordinaryView.Character.Subject) {
		t.Fatal("drag preview recomputed character statistics before the item reached a destination")
	}
	if item, ok := s.shopEquippedItem(0, slot); !ok || item.Empty() {
		t.Fatal("drag preview removed the canonical equipped item before the drop committed")
	}

	s.ShopSuppressDoll(0)
	restored := s.ShopScreen()
	if restored.Figure != ordinary || !reflect.DeepEqual(restored.Character.Subject, ordinaryView.Character.Subject) {
		t.Fatal("cancelling the drag did not restore the exact ordinary doll and unchanged statistics")
	}
	s.ShopSuppressDoll(slot)
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1},
		ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1})
	if item, ok := s.shopEquippedItem(0, slot); ok && !item.Empty() {
		t.Fatal("successful doll-to-pack drop left the item canonically equipped")
	}
}

func TestDollToShopGridRevealsTheTradeTableBeforeCommitting(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Archives = shopFigureArchive(t)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(code, f.Table)
	s.shelfBase = idx
	s.shopEquipFromShelf(0)
	s.shopBook = true

	act := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1},
		ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0})
	if act.Msg != "on the table" || s.shopBook || len(f.Shop.Table()) != 1 || !f.Shop.Table()[0].Mine {
		t.Fatalf("doll stage = %+v book %v table %+v", act, s.shopBook, f.Shop.Table())
	}
	if item, ok := s.shopEquippedItem(0, slot); ok && !item.Empty() {
		t.Fatal("doll stage revealed the table but did not commit the unequip")
	}
}
