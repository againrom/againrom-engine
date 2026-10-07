package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Weapons rows past eqSwordCode's. Greatsword fills both hands, as the
// shipped two-handed melee weapons do. Longbow is the shipped bow's shape:
// attack type 5, below the ranged attack types, and Hands 2. Flamer is the one
// shipped weapon in the ranged attack types, whose Hands cell is -1.
const (
	swGreatswordCode = uint16(0)<<12 | uint16(1)<<8 | uint16(0)<<5 | uint16(2)
	swLongbowCode    = uint16(0)<<12 | uint16(1)<<8 | uint16(0)<<5 | uint16(3)
	swFlamerCode     = uint16(0)<<12 | uint16(1)<<8 | uint16(0)<<5 | uint16(4)
)

// shieldWearTable is a parsed definition table whose weapons cover every way a
// weapon relates to a shield: a one-handed sword (Hands -1, as shipped), a
// Hands 2 melee weapon, a Hands 2 bow of attack type 5 and a ranged weapon of
// attack type 11 with no Hands value. One shield row, eqShieldCode.
func shieldWearTable(t *testing.T) *mapload.Table {
	t.Helper()
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes:    {eqScaleRow(1, 1, 1)},
			synth.DataBinMaterials: {eqScaleRow(1, 1, 1)},
			synth.DataBinWeapons: {
				eqWeaponRowHands("Sword", data.SkillBlade, 10, 20, 5, 3, 2, 7, 4, -1),
				eqWeaponRowHands("Greatsword", data.SkillBlade, 14, 26, 5, 3, 2, 9, 5, 2),
				eqWeaponRowHands("Longbow", 5, 6, 12, 8, 1, 6, 8, 5, 2),
				eqWeaponRowHands("Flamer", 11, 6, 12, 8, 1, 6, 8, 5, -1),
			},
			synth.DataBinShields: {eqShieldRow("Ward", 7, 3)},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("databin.Parse: %v", err)
	}
	return &mapload.Table{
		Shapes: f.Collection(databin.Shapes), Materials: f.Collection(databin.Materials),
		Weapons: f.Collection(databin.Weapons), Shields: f.Collection(databin.Shields), Armors: f.Collection(databin.Armors),
	}
}

func shieldWearShield() sim.ItemInstance {
	return sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 701,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}
}

func shieldWearWeapon(code uint16) sim.ItemInstance {
	return sim.ItemInstance{Code: code, Kind: 2, Price: 902}
}

// shieldWearWeapons names each weapon a wearer can hold beside the shield hand,
// and whether the shield leaves it in place.
var shieldWearWeapons = []struct {
	name string
	code uint16 // worn in slot 1; zero for none
	// moved is the slot a shield equip takes off in the same command.
	moved sim.EquipSlot
}{
	{"no weapon", 0, 0},
	{"one-handed weapon", eqSwordCode, 0},
	{"two-handed weapon", swGreatswordCode, 1},
	{"bow of attack type 5", swLongbowCode, 1},
	{"ranged weapon of attack type 11", swFlamerCode, 1},
}

func shieldWearWorld(t *testing.T, worn [sim.EquipSlots]sim.ItemInstance, pack ...sim.ItemInstance) *sim.World {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 20, MaxHP: 20}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, ItemInstances: pack, EquippedItems: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

func requireShieldWearInstance(t *testing.T, label string, got, want sim.ItemInstance) {
	t.Helper()
	sameEffects := len(got.Effects) == len(want.Effects) && (len(want.Effects) == 0 || reflect.DeepEqual(got.Effects, want.Effects))
	if !sim.ItemEqual(got, want) || got.Price != want.Price || !sameEffects {
		t.Fatalf("%s = %+v, want the complete instance %+v", label, got, want)
	}
}

func shieldWearWeaponData(t *testing.T, table *mapload.Table, code uint16) *data.Weapon {
	t.Helper()
	w, err := data.WeaponFromCode(data.ItemCode(code), table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("data.WeaponFromCode(%#x): %v", code, err)
	}
	return &w
}

// A shield is worn on its own. One put on beside a weapon that fills both hands
// takes that weapon off into the pack in the same command; a one-handed weapon
// stays worn.
func TestLiveShieldEquipIsWornAloneAndTakesOffAWeaponFillingBothHands(t *testing.T) {
	table := shieldWearTable(t)
	shield := shieldWearShield()
	piece, err := data.ShieldFromCode(data.ItemCode(eqShieldCode), table.Shapes, table.Materials, table.Shields)
	if err != nil {
		t.Fatalf("ShieldFromCode: %v", err)
	}
	for _, tc := range shieldWearWeapons {
		t.Run(tc.name, func(t *testing.T) {
			var worn [sim.EquipSlots]sim.ItemInstance
			weapon := shieldWearWeapon(tc.code)
			if tc.code != 0 {
				worn[0] = weapon
			}
			w := shieldWearWorld(t, worn, shield)
			hero := eqHero()
			mw := equipMission(t, w, 7, hero, nil, table)

			mw.enqueueEquip(0)
			if want := []sim.Command{sim.EquipDisplacing(7, 0, 2, tc.moved)}; !reflect.DeepEqual(mw.pending, want) {
				t.Fatalf("pending = %+v, want %+v", mw.pending, want)
			}
			mw.tick()

			gotWorn, _ := w.EquippedItems(7)
			gotPack, _ := w.CarriedItems(7)
			requireShieldWearInstance(t, "slot 2", gotWorn[1], shield)
			var wantPack []sim.ItemInstance
			var stays *data.Weapon
			weaponStays := tc.code != 0 && tc.moved == 0
			switch {
			case weaponStays:
				requireShieldWearInstance(t, "slot 1", gotWorn[0], weapon)
				stays = shieldWearWeaponData(t, table, tc.code)
			case tc.code == 0:
				if !gotWorn[0].Empty() {
					t.Fatalf("slot 1 = %+v, want empty", gotWorn[0])
				}
			default:
				if !gotWorn[0].Empty() {
					t.Fatalf("slot 1 = %+v, want the weapon taken off", gotWorn[0])
				}
				wantPack = append(wantPack, weapon)
			}
			if len(gotPack) != len(wantPack) {
				t.Fatalf("pack = %+v, want %+v", gotPack, wantPack)
			}
			for i := range wantPack {
				requireShieldWearInstance(t, "pack", gotPack[i], wantPack[i])
			}

			want := hero.Recompute(data.Profile{}, data.Loadout{Weapon: stays, Mod: data.EquipMod{
				Defence: piece.Defence + 4, Absorption: piece.Absorption,
			}}).Combat
			entity, _ := mw.entity(7)
			if entity.Defence != want.Defence || entity.Absorption != want.Absorption {
				t.Fatalf("live Defence/Absorption = %d/%d, want the shield's %d/%d", entity.Defence, entity.Absorption, want.Defence, want.Absorption)
			}

			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			var back sim.World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}
			if back.Hash() != w.Hash() {
				t.Fatalf("round-trip hash = %#x, want %#x", back.Hash(), w.Hash())
			}
			backWorn, _ := back.EquippedItems(7)
			backPack, _ := back.CarriedItems(7)
			requireShieldWearInstance(t, "round-trip slot 2", backWorn[1], shield)
			if len(backPack) != len(wantPack) || backWorn[0].Empty() == weaponStays {
				t.Fatalf("round trip lost the worn state: worn=%+v pack=%+v", backWorn, backPack)
			}
		})
	}
}

// The drawn starting weapon stands for a worn weapon until it is used. A shield
// put on over one that fills both hands leaves it in the pack, as a worn one
// would, and the hero is then bare.
func TestLiveShieldEquipOverAStartingWeaponFillingBothHandsLeavesItInThePack(t *testing.T) {
	table := shieldWearTable(t)
	shield := shieldWearShield()
	for _, code := range []uint16{swGreatswordCode, swLongbowCode} {
		start := shieldWearWeaponData(t, table, code)
		w := shieldWearWorld(t, [sim.EquipSlots]sim.ItemInstance{}, shield)
		mw := equipMission(t, w, 7, eqHero(), start, table)

		mw.enqueueEquip(0)
		if want := []sim.Command{sim.EquipDisplacing(7, 0, 2, 0)}; !reflect.DeepEqual(mw.pending, want) {
			t.Fatalf("%#x: pending = %+v, want the shield alone", code, mw.pending)
		}
		mw.tick()

		worn, _ := w.EquippedItems(7)
		pack, _ := w.CarriedItems(7)
		requireShieldWearInstance(t, "slot 2", worn[1], shield)
		if !worn[0].Empty() || len(pack) != 1 || pack[0].Code != uint16(start.Code) {
			t.Fatalf("%#x: slot 1 = %#x, pack holds %d items %+v, want the starting weapon %#x alone in the pack",
				code, worn[0].Code, len(pack), pack, uint16(start.Code))
		}
		if mw.currentWeaponFallbackActive() {
			t.Fatalf("%#x: the drawn starting weapon is still standing in for a worn one", code)
		}
	}
}

// A restored world that holds a shield and no weapon keeps the shield worn,
// complete.
func TestOpeningRestoredShieldOnlyWorldKeepsTheCompleteShieldWorn(t *testing.T) {
	table := shieldWearTable(t)
	shield := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 722,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 9}}}
	var worn [sim.EquipSlots]sim.ItemInstance
	worn[1] = shield.Clone()
	w := shieldWearWorld(t, worn)
	equipMission(t, w, 7, eqHero(), nil, table)
	gotWorn, _ := w.EquippedItems(7)
	gotPack, _ := w.CarriedItems(7)
	requireShieldWearInstance(t, "slot 2", gotWorn[1], shield)
	if len(gotPack) != 0 {
		t.Fatalf("pack = %+v, want empty", gotPack)
	}
}

// A shield beside a weapon that fills both hands cannot be worn together; a
// world restored with that pair keeps the weapon and puts the shield in the
// pack, complete.
func TestOpeningRestoredShieldBesideAWeaponFillingBothHandsMovesTheShieldToPack(t *testing.T) {
	table := shieldWearTable(t)
	shield := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 722,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 9}}}
	weapon := shieldWearWeapon(swLongbowCode)
	var worn [sim.EquipSlots]sim.ItemInstance
	worn[0], worn[1] = weapon, shield.Clone()
	w := shieldWearWorld(t, worn)
	equipMission(t, w, 7, eqHero(), nil, table)
	gotWorn, _ := w.EquippedItems(7)
	gotPack, _ := w.CarriedItems(7)
	requireShieldWearInstance(t, "slot 1", gotWorn[0], weapon)
	if !gotWorn[1].Empty() || len(gotPack) != 1 {
		t.Fatalf("worn=%+v pack=%+v, want the bow worn and the shield packed", gotWorn, gotPack)
	}
	requireShieldWearInstance(t, "pack", gotPack[0], shield)
}

// shieldWearShop is the town shop with one member who wears weapon, or nothing
// when weapon is zero, and carries the shield.
func shieldWearShop(t *testing.T, weapon uint16) (*FrontEnd, *townScreen) {
	t.Helper()
	f, s := shopRoom(t, nil)
	f.Table = shieldWearTable(t)
	carry := f.Carried[0].Carry
	carry.ItemInstances = []sim.ItemInstance{shieldWearShield()}
	carry.Items = []uint16{eqShieldCode}
	if weapon != 0 {
		carry.EquippedItems[0] = shieldWearWeapon(weapon)
		carry.Equipped[0] = weapon
	}
	return f, s
}

// The town shop wears a shield from the pack on its own, and takes a weapon
// that fills both hands off into the pack.
func TestShopShieldIsWornAloneAndTakesOffAWeaponFillingBothHands(t *testing.T) {
	shield := shieldWearShield()
	for _, tc := range shieldWearWeapons {
		t.Run(tc.name, func(t *testing.T) {
			_, s := shieldWearShop(t, tc.code)
			if act := s.shopEquipFromPack(1); act.Msg != "worn" {
				t.Fatalf("shopEquipFromPack = %+v, want worn", act)
			}
			worn := s.shopWornItemSlots(0)
			pack := s.shopPackItemInstances()
			if worn == nil {
				t.Fatal("the member shows no worn slots")
			}
			requireShieldWearInstance(t, "slot 2", worn[1], shield)
			switch {
			case tc.code == 0:
				if !worn[0].Empty() || len(pack) != 0 {
					t.Fatalf("worn=%+v pack=%+v, want the shield alone", worn, pack)
				}
			case tc.moved == 0:
				requireShieldWearInstance(t, "slot 1", worn[0], shieldWearWeapon(tc.code))
				if len(pack) != 0 {
					t.Fatalf("pack = %+v, want empty", pack)
				}
			default:
				if !worn[0].Empty() || len(pack) != 1 {
					t.Fatalf("worn=%+v pack=%+v, want the weapon in the pack", worn, pack)
				}
				requireShieldWearInstance(t, "pack", pack[0], shieldWearWeapon(tc.code))
			}
		})
	}
}

// The shop draws the starting weapon as a stand-in until it is used. A shield
// over one that fills both hands leaves it, made real, in the pack.
func TestShopShieldOverAStartingWeaponFillingBothHandsLeavesItInThePack(t *testing.T) {
	shield := shieldWearShield()
	_, s := shieldWearShop(t, 0)
	member := s.shopPartyMember(0)
	start := shieldWearWeaponData(t, s.in.Table, swGreatswordCode)
	member.Weapon = start

	if act := s.shopEquipFromPack(1); act.Msg != "worn" {
		t.Fatalf("shopEquipFromPack = %+v, want worn", act)
	}
	worn := s.shopWornItemSlots(0)
	pack := s.shopPackItemInstances()
	if worn == nil || !worn[0].Empty() || !member.WeaponMaterialized || len(pack) != 1 || pack[0].Code != uint16(start.Code) {
		t.Fatalf("materialized=%v pack holds %d items %+v, want the starting weapon %#x alone in the pack and slot 1 empty",
			member.WeaponMaterialized, len(pack), pack, uint16(start.Code))
	}
	requireShieldWearInstance(t, "slot 2", worn[1], shield)
}

// A member with no pack cannot put a weapon anywhere, so a shield that would
// take the weapon off is refused. A shield that takes nothing off is not.
func TestShopShieldNeedsAPackOnlyToTakeAWeaponOff(t *testing.T) {
	for _, tc := range shieldWearWeapons {
		t.Run(tc.name, func(t *testing.T) {
			_, s := shieldWearShop(t, tc.code)
			s.shopPartyMember(0).Carry.LiveLoad = &sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true}}
			slot, ok, act := s.shopWear(data.ItemCode(eqShieldCode))
			if want := tc.moved == 0; ok != want {
				t.Fatalf("shopWear ok = %v (%+v), want %v", ok, act, want)
			}
			if ok && slot != 2 {
				t.Fatalf("shopWear slot = %d, want 2", slot)
			}
			if !ok && act.Msg != "no pack" {
				t.Fatalf("shopWear refused with %q, want no pack", act.Msg)
			}
		})
	}
}

// Taking the weapon off, into the pack or onto the table, leaves the shield
// worn.
func TestShopWeaponRemovalLeavesTheShieldWorn(t *testing.T) {
	for _, toTable := range []bool{false, true} {
		name := "to pack"
		if toTable {
			name = "to table"
		}
		t.Run(name, func(t *testing.T) {
			f, s := shopRoom(t, nil)
			f.Table = eqDefsTable(t)
			weapon := sim.ItemInstance{Code: eqSwordCode, Kind: 2, Price: 411,
				Effects: []sim.ItemEffect{{Kind: 12, Operand: 3}}}
			shield := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 722,
				Effects: []sim.ItemEffect{{Kind: 15, Operand: 9}}}
			carry := f.Carried[0].Carry
			carry.EquippedItems[0], carry.EquippedItems[1] = weapon.Clone(), shield.Clone()
			carry.Equipped[0], carry.Equipped[1] = weapon.Code, shield.Code

			if toTable {
				if act := s.shopUnequipToTable(1); act.Msg != "on the table" {
					t.Fatalf("shopUnequipToTable = %+v", act)
				}
				table := f.Shop.Table()
				if len(table) != 1 || table[0].Code != data.ItemCode(weapon.Code) || table[0].Price != weapon.Price {
					t.Fatalf("table = %+v, want complete weapon", table)
				}
			} else if act := s.shopUnequipDoll(1); act.Msg != "off, into the pack" {
				t.Fatalf("shopUnequipDoll = %+v", act)
			}

			worn := s.shopWornItemSlots(0)
			pack := s.shopPackItemInstances()
			wantPack := 1
			if toTable {
				wantPack = 0
			}
			if worn == nil || !worn[0].Empty() || len(pack) != wantPack {
				t.Fatalf("weapon removal left worn=%+v pack=%+v, want no weapon and %d in the pack", worn, pack, wantPack)
			}
			requireShieldWearInstance(t, "slot 2", worn[1], shield)
		})
	}
}

// In a city whose items are objects with identities, a shield that takes a
// weapon off leaves both objects where the player put them; a one-handed weapon
// then joins the shield and leaves it worn when it comes off. The pair survives
// SAVE and LOAD.
func TestCityShopShieldTakesAWeaponOffAndBothNodesSurviveSAV(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Table = shieldWearTable(t)
	f.Town.open = true
	f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	twoHanded, shield, sword := shieldWearWeapon(swGreatswordCode), shieldWearShield(), shieldWearWeapon(eqSwordCode)
	cityShopSetTestPack(t, f, 0, sim.StackItem(twoHanded, 1), sim.StackItem(shield, 1), sim.StackItem(sword, 1))
	cityShopGraph(t, f)
	ids := slices.Clone(cityShopRoots(t, f, 0).Pack)
	if len(ids) != 3 || ids[0] == ids[1] || ids[1] == ids[2] || ids[0] == ids[2] {
		t.Fatalf("pack nodes = %v, want three distinct", ids)
	}
	nodes := func(label string, f *FrontEnd, weapon, shielded sim.SavedObjectID, pack ...sim.SavedObjectID) {
		t.Helper()
		root := cityShopRoots(t, f, 0)
		if root.Worn[0] != weapon || root.Worn[1] != shielded || !slices.Equal(root.Pack, pack) {
			t.Fatalf("%s: worn nodes %v pack nodes %v, want %d and %d worn with pack %v", label, root.Worn[:2], root.Pack, weapon, shielded, pack)
		}
	}
	wear := func(label string, cell int) {
		t.Helper()
		if act := screen.shopEquipFromPack(cell); act.Msg != "worn" {
			t.Fatalf("%s = %+v, want worn", label, act)
		}
	}
	wear("two-handed weapon", 1)
	wear("shield over the two-handed weapon", 1)
	nodes("shield worn, weapon packed", f, 0, ids[1], ids[2], ids[0])
	worn := mapload.MemberItemEquipment(f.Carried[0], f.Table)
	if !worn[0].Empty() || worn[1].Code != shield.Code {
		t.Fatalf("party record wears %#x and %#x, want the shield alone", worn[0].Code, worn[1].Code)
	}
	cold := cityShopColdLoad(t, f)
	nodes("after SAVE and LOAD", cold, 0, ids[1], ids[2], ids[0])

	wear("one-handed weapon beside the shield", 1)
	nodes("one-handed weapon beside the shield", f, ids[2], ids[1], ids[0])
	if act := screen.shopUnequipDoll(1); act.Msg != "off, into the pack" {
		t.Fatalf("weapon removal = %+v", act)
	}
	nodes("weapon removed", f, 0, ids[1], ids[0], ids[2])
}
