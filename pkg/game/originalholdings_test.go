package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Synthetic archive controls, not claims about a recorded ROM1 session.
type holdingFixtureItem struct {
	class        string
	code, count  uint16
	kind, state  uint8
	price        int32
	weight       int16
	effects      []sim.ItemEffect
	row, ownKind uint8
	attack       [24]byte
	defence      [22]byte
}
type holdingFixture struct {
	humanoid       bool
	weapon, shield *holdingFixtureItem
	worn           [12]*holdingFixtureItem
	items          []*holdingFixtureItem
	insertIndex    *uint32
	accumulator    *int32
}

func TestOriginalHoldings1108ExactGraphRolesAndOwnership(t *testing.T) {
	weapon := &holdingFixtureItem{class: "Weapon", code: 0x810e, count: 1, kind: 2, price: -981,
		effects: []sim.ItemEffect{{Kind: 41, Operand: 327681}, {Kind: 12, Mode: 1, Operand: 0xffff0003}, {Kind: 12, Mode: 1, Operand: 0xffff0003}}}
	boots := &holdingFixtureItem{class: "Armor", code: 0x0c01, count: 1, kind: 1, price: 7}
	potion := &holdingFixtureItem{class: "Item", code: 0x0e06, count: 3, kind: 3, price: 50}
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, holdings: &holdingFixture{weapon: weapon, items: []*holdingFixtureItem{potion}}}
	b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 9, maxHP: 41, holdings: &holdingFixture{humanoid: true}}
	b.holdings.worn[11] = boots
	players := []*poolFixturePlayer{nil, {}, {groups: [][]*poolFixtureActor{{a, nil}, {a, b}}}}
	read := func() ([]sav.ActorHoldings, error) {
		f, err := sav.Open(savedContainer(poolFixtureBody(players, nil)))
		if err != nil {
			return nil, err
		}
		return f.ActorHoldings()
	}
	got, err := read()
	if err != nil || len(got) != 2 {
		t.Fatalf("graph: %+v %v", got, err)
	}
	if got[0].HeldWeapon.Price != -981 || got[0].HeldWeapon.Kind != 2 || len(got[0].HeldWeapon.Effects) != 3 || got[0].Items[0].Stack != 3 || got[1].Worn[11].Code != 0x0c01 || got[1].Worn[0] != nil {
		t.Fatalf("projection changed values/roles: %+v", got)
	}
	// A repeated whole actor above is one owner; sharing its item with
	// another actor or a distinct held/inventory role is not another copy.
	b.holdings.weapon = weapon
	if rows, err := read(); err != nil || len(rows) != 2 || !reflect.DeepEqual(rows[0].HeldWeapon, rows[1].HeldWeapon) {
		t.Fatalf("cross-owner alias values: %v", err)
	}
	b.holdings.weapon = nil
	a.holdings.items = append(a.holdings.items, weapon)
	if rows, err := read(); err != nil || len(rows) != 2 || len(rows[0].Items) != 2 || !reflect.DeepEqual(*rows[0].HeldWeapon, rows[0].Items[1]) {
		t.Fatalf("worn/carried alias values: %v", err)
	}
	a.holdings.items = []*holdingFixtureItem{nil}
	if rows, err := read(); err != nil || len(rows) != 2 || len(rows[0].Items) != 1 || !reflect.DeepEqual(rows[0].Items[0], sav.Piece{}) {
		t.Fatalf("explicit null inventory slot: %v", err)
	}
}

func holdingsMission(t *testing.T) *Mission {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 1, MapUnitID: 91, X: 5, Y: 6, HP: 20, MaxHP: 20}, {ID: 2, MapUnitID: 92, X: 6, Y: 6, HP: 20, MaxHP: 20}},
		nil, sim.Relations{}, nil, []sim.Stock{{ID: 1, Items: []uint16{0x0e06}}, {ID: 2, Items: []uint16{0x0e07}}})
	if err != nil {
		t.Fatal(err)
	}
	return &Mission{Map: &alm.Map{Width: 40, Height: 40}, World: w}
}

func TestOriginalHoldings1108BatchRefusalsAreAtomic(t *testing.T) {
	good := sav.ActorHoldings{MapUnitID: 91, Cell: 0x0605, HP: 7}
	for _, kind := range []string{"source collision", "dead source collision", "target collision", "zero code", "zero count", "equipped stack", "wrong slot", "slot collision", "armor held overlap", "held class", "aggregate expansion"} {
		t.Run(kind, func(t *testing.T) {
			ms := holdingsMission(t)
			bad := sav.ActorHoldings{MapUnitID: 92, Cell: 0x0606, HP: 9, Items: []sav.Piece{{Class: "Item", Code: 0x0e06, Stack: 2, Kind: 3, Price: 50}}}
			switch kind {
			case "source collision":
				bad.MapUnitID = 91
			case "dead source collision":
				bad.MapUnitID = 91
				bad.HP = 0
			case "target collision":
				entities := ms.World.Entities()
				entities[1].MapUnitID = 91
				w, err := sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, entities)
				if err != nil {
					t.Fatal(err)
				}
				ms.World = w
			case "zero code":
				bad.Items[0].Code = 0
			case "zero count":
				bad.Items[0].Stack = 0
			case "equipped stack":
				bad.HeldWeapon = &sav.Piece{Class: "Weapon", Code: 0x0101, Stack: 2}
			case "wrong slot":
				bad.Worn[11] = &sav.Piece{Class: "Armor", Code: 0x0701, Stack: 1}
			case "slot collision":
				bad.HeldWeapon = &sav.Piece{Class: "Weapon", Code: 0x0101, Stack: 1}
				bad.Worn[0] = &sav.Piece{Class: "Armor", Code: 0x0101, Stack: 1}
			case "armor held overlap":
				bad.Worn[0] = &sav.Piece{Class: "Armor", Code: 0x0101, Stack: 1}
			case "held class":
				bad.HeldWeapon = &sav.Piece{Class: "Armor", Code: 0x0101, Stack: 1}
			case "aggregate expansion":
				bad.Items = nil
				for range 17 {
					bad.Items = append(bad.Items, sav.Piece{Class: "Item", Code: 0x0e06, Stack: 65535})
				}
			}
			before := ms.World.Hash()
			report := OriginalSaveResume{}
			if err := restoreOriginalActorStock(ms, []sav.ActorHoldings{good, bad}, nil, &report); err == nil || ms.World.Hash() != before || report.Stocked != 0 {
				t.Fatalf("refusal=%v hash changed=%t report=%+v", err, ms.World.Hash() != before, report)
			}
		})
	}
}

// DIV-748
func TestRestoreOriginalActorStockRestoresDespiteUnsupportedItemEffectAndCountsIt(t *testing.T) {
	ms := holdingsMission(t)
	source := []sav.ActorHoldings{{MapUnitID: 91, Cell: 0x0605, HP: 7, Items: []sav.Piece{
		{Class: "Item", Code: 0x0e06, Stack: 1, Kind: 3, Price: 50, UnsupportedEffectStates: []uint8{1}},
	}}}
	report := OriginalSaveResume{}
	if err := restoreOriginalActorStock(ms, source, nil, &report); err != nil {
		t.Fatalf("a holdings batch with one unsupported item Effect was refused: %v", err)
	}
	if report.Stocked != 1 {
		t.Fatalf("Stocked = %d, want 1", report.Stocked)
	}
	if report.UnsupportedItemEffects != 1 {
		t.Fatalf("UnsupportedItemEffects = %d, want 1", report.UnsupportedItemEffects)
	}
}

func TestOriginalHoldings1108AllPlayerAppLoadSaveFreshLoad(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	weapon := &holdingFixtureItem{class: "Weapon", code: 0x0101, count: 1, kind: 2, price: -200, effects: []sim.ItemEffect{{Kind: 41, Operand: 327681}}}
	potion := &holdingFixtureItem{class: "Item", code: 0x0e06, count: 3, kind: 3, price: 50, effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00064}}}
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, holdings: &holdingFixture{weapon: weapon, items: []*holdingFixtureItem{potion}}}
	b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 9, maxHP: 41, holdings: &holdingFixture{}}
	payload := poolFixtureSave(a, b)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	app := f.App("1108-synthetic-holdings")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9999.sav")
	e := poolEntity(t, f.live.world, 91)
	worn, _ := f.live.world.EquippedItems(e.ID)
	carried, _ := f.live.world.CarriedStacks(e.ID)
	if worn[0].Price != -200 || len(carried) != 1 || carried[0].Count != 3 || e.WeaponSpellSource != sim.WeaponSpellItem || e.WeaponSpell != 1 {
		t.Fatalf("all-Player stock/cache: %+v %+v %+v", worn, carried, e)
	}
	if err := f.live.world.MoveCarried(e.ID, poolEntity(t, f.live.world, 92).ID, 0x0e06, 2); err != nil {
		t.Fatal(err)
	}
	hash := f.live.world.Hash()
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("SAVE: %v %v: %s", entries, err, app.HeadlessMessage())
	}
	fresh := currentPoolFixtureFront(t, 91, 92)
	app2 := fresh.App("1108-fresh-native")
	s2, l2, load2 := fresh.SaveSeams(store, OriginalStore{}, nil)
	app2.SetSaveSeams(s2, l2, load2)
	groundAppLoad(t, app2, l2, entries[0].Name)
	if fresh.live.world.Hash() != hash {
		t.Fatal("fresh native LOAD changed canonical stock")
	}
	diagnostic, report, err := ResumeOriginalSave(f.Archives.Containers, payload, nil, mapload.DifficultyNormal, nil, nil)
	if err != nil || report.Stocked != 2 {
		t.Fatalf("diagnostic: %+v %v", report, err)
	}
	dw, _ := diagnostic.World.EquippedItems(poolEntity(t, diagnostic.World, 91).ID)
	if !reflect.DeepEqual(dw, worn) {
		t.Fatal("LOAD doors disagree")
	}
	// A holdings item's own unsupported Effect state no longer refuses the
	// LOAD: it restores the item, with that one Effect simply not applied, the
	// same per-item disclosure ground loot uses.
	a.holdings.items[0].state = 8
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), poolFixtureSave(a, b), 0600); err != nil {
		t.Fatal(err)
	}
	groundAppLoad(t, app, list, "game9999.sav")
	e = poolEntity(t, f.live.world, 91)
	carried, _ = f.live.world.CarriedStacks(e.ID)
	if len(carried) != 1 || carried[0].Code != 0x0e06 || carried[0].Count != 3 || len(carried[0].Effects) != 0 {
		t.Fatalf("the unsupported-effect item's own supported state was lost or fabricated: %+v", carried)
	}
}

func TestOriginalHoldings1108ExclusionsNeverGuessOrTouchParty(t *testing.T) {
	ms := holdingsMission(t)
	ms.Party = []mapload.PartyMember{{Saved: &mapload.Saved{MapUnitID: 92}}}
	worn, _ := ms.World.EquippedItems(2)
	carried, _ := ms.World.CarriedStacks(2)
	source := []sav.ActorHoldings{
		{MapUnitID: 91, Cell: 0x0605, HP: 7},
		{MapUnitID: 92, Cell: 0x0606, HP: 9},
		{MapUnitID: 93, Cell: 0x0605, HP: 9}, // same cell as 91 is not identity
		{MapUnitID: 94, Cell: 0x0605, HP: 0},
		{MapUnitID: 95, Cell: 0x0605, HP: 65535},
		{MapUnitID: 96, Cell: 0x0605, HP: 9, Stage: 3},
		{MapUnitID: 97, Cell: 0xffff, HP: 9},
		{MapUnitID: 0, Cell: 0x0605, HP: 9},
	}
	var r OriginalSaveResume
	if err := restoreOriginalActorStock(ms, source, nil, &r); err != nil {
		t.Fatal(err)
	}
	if r.Stocked != 1 || r.StockParty != 1 || r.StockDead != 3 || r.StockOffMap != 1 || r.StockUnbound != 1 || r.StockUnmatched != 1 {
		t.Fatalf("counters: %+v", r)
	}
	afterWorn, _ := ms.World.EquippedItems(2)
	afterPack, _ := ms.World.CarriedStacks(2)
	if !reflect.DeepEqual(afterWorn, worn) || !reflect.DeepEqual(afterPack, carried) {
		t.Fatal("persistent party changed")
	}
	pack, _ := ms.World.CarriedStacks(1)
	if len(pack) != 0 {
		t.Fatal("empty source did not clear starter stock")
	}
}

func TestRestoreOriginalActorStockStagingCarriesSavedProjectiles(t *testing.T) {
	ms := holdingsMission(t)
	if err := ms.World.ImportOriginalProjectiles(sim.SavedProjectiles{
		FreeIndex: 267, IDs: []uint16{266}, Items: []sim.SavedProjectile{{ID: 266, X: 5, ActionSpell: 42}},
	}); err != nil {
		t.Fatalf("ImportOriginalProjectiles: %v", err)
	}
	var r OriginalSaveResume
	if err := restoreOriginalActorStock(ms, nil, nil, &r); err != nil {
		t.Fatal(err)
	}
	got := ms.World.SavedProjectiles()
	if got.FreeIndex != 267 || len(got.Items) != 1 || got.Items[0].X != 5 || got.Items[0].ActionSpell != 42 {
		t.Fatalf("saved projectiles did not survive restoreOriginalActorStock's own staging: %+v", got)
	}
}
