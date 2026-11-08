package game

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// weaponMemoryMission is one game's mission for a W test: member id wears
// held and carries other in its pack. table is the table W asks; reload is
// that game's cold LOAD of a written file.
type weaponMemoryMission struct {
	f      *FrontEnd
	id     sim.EntityID
	held   sim.ItemInstance
	other  sim.ItemInstance
	table  *mapload.Table
	second bool
	reload func(t *testing.T, raw []byte) *FrontEnd
	// take removes item from the pack the way the fixture's pack allows.
	take func(t *testing.T, mw *mapWorld, item sim.ItemInstance)
}

func firstWeaponMemoryMission(t *testing.T) weaponMemoryMission {
	t.Helper()
	f := currentArchiveFixture(t)
	id := f.live.mission.ids[0]
	table := shieldWearTable(t)
	held, other := shieldWearWeapon(eqSwordCode), shieldWearWeapon(swGreatswordCode)
	stock := sim.Stock{ID: id, ItemInstances: []sim.ItemInstance{shieldWearWeapon(swLongbowCode), other}}
	stock.EquippedItems[0] = held
	if !f.live.world.ReplaceStock(stock) {
		t.Fatal("the fixture member has no stock")
	}
	f.live.invParty.table = table
	return weaponMemoryMission{f: f, id: id, held: held, other: other, table: table,
		reload: func(t *testing.T, raw []byte) *FrontEnd {
			cold := openCurrentArchive(t, raw)
			cold.live.invParty.table = table
			return cold
		},
		take: func(t *testing.T, mw *mapWorld, item sim.ItemInstance) {
			items, _ := mw.world.CarriedItems(id)
			worn, _ := mw.world.EquippedItems(id)
			i := slices.IndexFunc(items, func(it sim.ItemInstance) bool { return sim.SameUnit(it, item, sim.UnitWorn) })
			if i < 0 || !mw.world.ReplaceStock(sim.Stock{ID: id, ItemInstances: slices.Delete(items, i, i+1), EquippedItems: worn}) {
				t.Fatalf("setup: %#x could not be taken from the pack", item.Code)
			}
		}}
}

func secondWeaponMemoryMission(t *testing.T) weaponMemoryMission {
	t.Helper()
	f, app, screen, _ := secondBagFront(t, secondBagSample())
	secondBagEnter(t, f, app, screen)
	// The bag fixture's table holds no row for its weapons; W asks this one.
	table := shieldWearTable(t)
	f.live.invParty.table = table
	return weaponMemoryMission{f: f, id: f.live.mission.ids[0], held: secondBagWornWeapon, other: secondBagWeapon, table: table, second: true,
		reload: func(t *testing.T, raw []byte) *FrontEnd {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "game0002.sav"), raw, 0o600); err != nil {
				t.Fatal(err)
			}
			cold, _ := secondMissionLoadOf(t, secondMissionContent{side: 128}, dir, "game0002.sav")
			if cold.live == nil {
				t.Fatal("the cold LOAD opened no mission")
			}
			cold.live.invParty.table = table
			return cold
		},
		take: func(t *testing.T, mw *mapWorld, item sim.ItemInstance) {
			id := f.live.mission.ids[0]
			e, _ := mw.world.Entity(id)
			idx := mw.packIndex(id, item)
			if idx < 0 {
				t.Fatalf("setup: the pack has no %#x", item.Code)
			}
			mw.pending = append(mw.pending, sim.DropCarried(id, sim.ItemSlot(idx), sim.CellPoint{X: e.X, Y: e.Y}))
		}}
}

var weaponMemoryGames = []struct {
	name string
	open func(t *testing.T) weaponMemoryMission
}{
	{"first game", firstWeaponMemoryMission},
	{"second game", secondWeaponMemoryMission},
}

// wear equips item from the pack through an ordinary order and runs its frames.
func (m weaponMemoryMission) wear(t *testing.T, mw *mapWorld, item sim.ItemInstance) {
	t.Helper()
	idx := mw.packIndex(m.id, item)
	if idx < 0 {
		t.Fatalf("the pack has no %#x: %v", item.Code, swapPackCodes(mw.world, m.id))
	}
	mw.pending = append(mw.pending, sim.Equip(m.id, sim.ItemSlot(idx), 1))
	swapFrames(mw, 3)
	if got, _ := swapHeld(t, mw.world, m.id); got != item.Code {
		t.Fatalf("setup: the hand holds %#x after the equip, want %#x", got, item.Code)
	}
}

func (m weaponMemoryMission) press(mw *mapWorld) {
	mw.swapWeapons([]uint32{uint32(m.id)})
	swapFrames(mw, 4)
}

func (m weaponMemoryMission) save(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// rewriteActions edits the actions leaf of a written file, the engine
// document's own where the game writes its original's records.
func (m weaponMemoryMission) rewriteActions(t *testing.T, raw []byte, edit func(*currentActionData)) []byte {
	t.Helper()
	engine := raw
	if m.second {
		var err error
		if engine, err = sav.EngineDocument(raw); err != nil {
			t.Fatal(err)
		}
	}
	doc, err := sav.DecodeDocumentData(engine)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatalf("no actions leaf: %v", err)
	}
	edit(a)
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, b); err != nil {
		t.Fatal(err)
	}
	if engine, err = sav.EncodeDocumentData(doc); err != nil {
		t.Fatal(err)
	}
	if !m.second {
		return engine
	}
	native, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if native, err = sav.StoreEngineDocument(native, engine); err != nil {
		t.Fatal(err)
	}
	out, err := sav.EncodeDocumentData(native)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// withoutMemory is the file with no memory block, as an original's file has
// none.
func (m weaponMemoryMission) withoutMemory(t *testing.T, raw []byte) []byte {
	t.Helper()
	return m.rewriteActions(t, raw, func(a *currentActionData) { delete(a.Residue, "weaponMemory") })
}

// A member that swapped weapons before a SAVE takes the previous weapon back
// after the LOAD, an enchanted one among the second game's, and the pair
// keeps swapping through a further SAVE and LOAD. The first press, before any
// change of weapon, is the control: with no memory W queues nothing.
func TestWeaponSwapMemoryComesBackThroughSaveAndLoad(t *testing.T) {
	for _, game := range weaponMemoryGames {
		t.Run(game.name, func(t *testing.T) {
			m := game.open(t)
			mw := m.f.live
			swapFrames(mw, 1)
			mw.swapWeapons([]uint32{uint32(m.id)})
			if len(mw.pending) != 0 {
				t.Fatalf("control: W with no memory queued %+v", mw.pending)
			}
			m.wear(t, mw, m.other)

			cold := m.reload(t, m.save(t, m.f))
			if got, _ := swapHeld(t, cold.live.world, m.id); got != m.other.Code {
				t.Fatalf("after the LOAD the hand holds %#x, want %#x", got, m.other.Code)
			}
			m.press(cold.live)
			if got, _ := swapHeld(t, cold.live.world, m.id); got != m.held.Code {
				t.Fatalf("W after the LOAD: the hand holds %#x, want the previous weapon %#x", got, m.held.Code)
			}

			again := m.reload(t, m.save(t, cold))
			m.press(again.live)
			if got, _ := swapHeld(t, again.live.world, m.id); got != m.other.Code {
				t.Fatalf("W after a second LOAD: the hand holds %#x, want %#x", got, m.other.Code)
			}
		})
	}
}

// A file with no memory block, as an original program's has none, loads with
// no memory: W queues nothing and the hand keeps what it holds. The World is
// the same with the block and without it.
func TestWeaponSwapMemoryAbsentBlockLoadsEmpty(t *testing.T) {
	for _, game := range weaponMemoryGames {
		t.Run(game.name, func(t *testing.T) {
			m := game.open(t)
			mw := m.f.live
			swapFrames(mw, 1)
			m.wear(t, mw, m.other)
			raw := m.save(t, m.f)

			bare := m.reload(t, m.withoutMemory(t, raw))
			if _, kept := bare.live.weaponSwap.previous[m.id]; kept {
				t.Fatal("a file with no memory block loaded a memory")
			}
			bare.live.swapWeapons([]uint32{uint32(m.id)})
			if len(bare.live.pending) != 0 {
				t.Errorf("W queued %+v", bare.live.pending)
			}
			swapFrames(bare.live, 3)
			if got, _ := swapHeld(t, bare.live.world, m.id); got != m.other.Code {
				t.Errorf("the hand holds %#x, want %#x", got, m.other.Code)
			}

			full := m.reload(t, raw)
			if _, kept := full.live.weaponSwap.previous[m.id]; !kept {
				t.Fatal("control: the file with its block loaded no memory")
			}
			full.live.pending = nil
			if a, b := m.reload(t, m.withoutMemory(t, raw)).live.world.Hash(), full.live.world.Hash(); a != b {
				t.Errorf("the World hashes %#x without the block and %#x with it", a, b)
			}
		})
	}
}

// A remembered weapon gone from the pack before the SAVE writes no memory for
// it: after the LOAD W does nothing.
func TestWeaponSwapMemoryWeaponGoneBeforeSaveDoesNothing(t *testing.T) {
	for _, game := range weaponMemoryGames {
		t.Run(game.name, func(t *testing.T) {
			m := game.open(t)
			mw := m.f.live
			swapFrames(mw, 1)
			m.wear(t, mw, m.other)
			m.take(t, mw, m.held)
			swapFrames(mw, 2)
			if mw.packIndex(m.id, m.held) >= 0 {
				t.Fatal("setup: the previous weapon is still in the pack")
			}

			cold := m.reload(t, m.save(t, m.f))
			cold.live.swapWeapons([]uint32{uint32(m.id)})
			if len(cold.live.pending) != 0 {
				t.Errorf("W queued %+v", cold.live.pending)
			}
			swapFrames(cold.live, 3)
			if got, _ := swapHeld(t, cold.live.world, m.id); got != m.other.Code {
				t.Errorf("the hand holds %#x, want %#x", got, m.other.Code)
			}
		})
	}
}

// The first game's pair with a shield: the shield went to the pack with the
// two-handed weapon and comes back with the sword after the LOAD.
func TestWeaponSwapMemoryBringsTheShieldBackAfterLoad(t *testing.T) {
	m := firstWeaponMemoryMission(t)
	mw := m.f.live
	shield := shieldWearShield()
	stock := sim.Stock{ID: m.id, ItemInstances: []sim.ItemInstance{m.other}}
	stock.EquippedItems[0], stock.EquippedItems[1] = m.held, shield
	if !mw.world.ReplaceStock(stock) {
		t.Fatal("the fixture member has no stock")
	}
	swapFrames(mw, 1)
	idx := mw.packIndex(m.id, m.other)
	mw.pending = append(mw.pending, sim.EquipDisplacing(m.id, sim.ItemSlot(idx), 1, 2))
	swapFrames(mw, 3)
	if weapon, held := swapHeld(t, mw.world, m.id); weapon != m.other.Code || held != 0 {
		t.Fatalf("setup: the hands hold %#x and %#x", weapon, held)
	}

	cold := m.reload(t, m.save(t, m.f))
	m.press(cold.live)
	weapon, held := swapHeld(t, cold.live.world, m.id)
	if weapon != m.held.Code || held != shield.Code {
		t.Errorf("W after the LOAD: the hands hold %#x and %#x, want %#x and %#x", weapon, held, m.held.Code, shield.Code)
	}
}

// A damaged memory row refuses the LOAD with a bounded error.
func TestWeaponSwapMemoryRefusesAnInvalidRow(t *testing.T) {
	m := firstWeaponMemoryMission(t)
	mw := m.f.live
	swapFrames(mw, 1)
	m.wear(t, mw, m.other)
	raw := m.save(t, m.f)
	if !bytes.Contains(raw, []byte("weaponMemory")) {
		t.Fatal("the file holds no memory block")
	}
	bad := m.rewriteActions(t, raw, func(a *currentActionData) {
		a.Residue["weaponMemory"] = json.RawMessage(`[{"Unit":1,"Weapon":257,"WeaponAt":-3}]`)
	})
	doc, err := sav.DecodeDocumentData(bad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readCurrentActions(&doc); err == nil || !strings.Contains(err.Error(), "weapon") {
		t.Errorf("an invalid memory row answered %v", err)
	}
}

// A previous shield still worn when the SAVE is taken is written as worn and
// read back from slot 2; a previous weapon at a later pack element is read
// back from that element. The rows are the memory's whole SAV form.
func TestWeaponSwapMemoryRowsFindWornShieldAndLaterElement(t *testing.T) {
	table := shieldWearTable(t)
	var worn [2][sim.EquipSlots]sim.ItemInstance
	worn[0][0], worn[0][1] = shieldWearWeapon(eqSwordCode), shieldWearShield()
	w := swapWorld(t, worn, [2][]sim.ItemInstance{{shieldWearWeapon(swLongbowCode), shieldWearWeapon(swGreatswordCode)}, nil})
	mw := swapMission(t, w, table)
	mw.weaponSwap.previous = map[sim.EntityID]heldPair{7: {weapon: shieldWearWeapon(swGreatswordCode), shield: shieldWearShield()}}
	rows := mw.weaponMemoryRows()
	if len(rows) != 1 || rows[0].WeaponAt != 1 || rows[0].ShieldAt != shieldWornAt || rows[0].Shield != eqShieldCode {
		t.Fatalf("rows %+v", rows)
	}
	cold := swapMission(t, w, table)
	cold.restoreWeaponMemory(rows, func(id sim.EntityID) (sim.EntityID, bool) { return id, true })
	got := cold.weaponSwap.previous[7]
	if !sim.SameUnit(got.weapon, shieldWearWeapon(swGreatswordCode), sim.UnitWorn) || !sim.SameUnit(got.shield, shieldWearShield(), sim.UnitWorn) {
		t.Errorf("restored pair %+v", got)
	}
	// A place that no longer holds the code keeps no row.
	stale := []weaponMemoryRow{{Unit: 7, Weapon: swFlamerCode, WeaponAt: 1}}
	empty := swapMission(t, w, table)
	empty.restoreWeaponMemory(stale, func(id sim.EntityID) (sim.EntityID, bool) { return id, true })
	if _, kept := empty.weaponSwap.previous[7]; kept {
		t.Error("a row whose place holds another code was restored")
	}
}
