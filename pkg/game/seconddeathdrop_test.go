package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	secondDropServerID = 9101
	secondDropUnitID   = 11
	// secondDropMask is the Dragon rows' mask (R2-ENGINE-942): magic,
	// weapon mode 8 and bit 29.
	secondDropMask = 0x2c0fffff
)

// secondDropMission is mission 10 with a dragon-like creature of Player 2
// near the fallback drop: type 71, gold at chance 100 and its item chance
// 100 over 0..100000 with the Dragon mask.
func secondDropMission() secondMissionContent {
	p := make([]int32, 62)
	for i := range p {
		p[i] = -1
	}
	p[4], p[29], p[30], p[55] = 20, 71, 4, secondDropServerID
	p[38], p[39], p[40] = 100, 300, 200
	p[41], p[42], p[43], p[44] = 100, 0, 100000, secondDropMask
	return secondMissionContent{side: 128, players: 2,
		creatures: map[int]dbEntry{2: {name: "Dragon", params: p}},
		units: []alm.Unit{{X: 64<<8 | 0x80, Y: 64<<8 | 0x80, ClassID: 9, ClassSubID: 1,
			ServerID: secondDropServerID, Owner: 2, UnitID: secondDropUnitID, GroupID: 6}}}
}

// secondDropTables give f's table the shop collections the draw reads:
// scroll rows 6 and 35 and the six potions at price 10, and no equipment.
func secondDropTables(f *FrontEnd) {
	rows := sleepCollection{secondShopTestCollection{}, 128}
	for _, i := range []int{6, 35, 69, 70, 71, 72, 73, 74} {
		rows.secondShopTestCollection[i] = []int32{10}
	}
	f.Table.Shapes, f.Table.Materials, f.Table.Armors, f.Table.Shields = rows, rows, rows, rows
	f.Table.Weapons, f.Table.MagicItems, f.Table.Magic = rows, rows, rows
	if spells, ok := f.Table.Spells.(dbCollection); ok {
		f.Table.Spells = secondDropSpells{spells}
	}
	if _, ok := secondShopTables(f.Table); !ok {
		panic("the drop tables are not the shop's collections")
	}
}

// secondDropSpells is the fixture's spell rows as the shop reads them.
type secondDropSpells struct{ dbCollection }

func (secondDropSpells) EntryRaw(int) []byte        { return nil }
func (secondDropSpells) EntryDoubles(int) []float64 { return nil }

// secondDropKill kills the dragon of f's mission, ticks until its body has
// dropped, and answers its one sack.
func secondDropKill(t *testing.T, f *FrontEnd) sim.Sack {
	t.Helper()
	w := f.live.world
	dragon := secondDropCreature(t, w)
	if dragon.DeathItem != (sim.DeathItemRoll{Chance: 100, Min: 0, Max: 100000, Mask: secondDropMask}) {
		t.Fatalf("the dragon holds the roll %+v", dragon.DeathItem)
	}
	if err := w.HeadlessDamage(dragon.ID, dragon.HP+20); err != nil {
		t.Fatal(err)
	}
	for range 60 {
		f.live.tick()
	}
	sacks := w.Sacks()
	if len(sacks) != 1 || sacks[0].X != 64 || sacks[0].Y != 64 || sacks[0].Gold < 300 || sacks[0].Gold > 500 ||
		len(sacks[0].ItemInstances) != 1 {
		t.Fatalf("sacks %+v, want one at 64/64 with 300..500 gold and one item", sacks)
	}
	if part := sacks[0].ItemInstances[0].Code >> 8 & 0xf; part != 0xe {
		t.Fatalf("the dropped item %+v is not a book, scroll or potion", sacks[0].ItemInstances[0])
	}
	return sacks[0]
}

func secondDropCreature(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.MapUnitID == secondDropUnitID {
			return e
		}
	}
	t.Fatal("the World holds no dragon")
	return sim.Entity{}
}

// secondDropLoad SAVEs f's mission from the game menu into dir and LOADs it
// into a cold front with the drop tables.
func secondDropLoad(t *testing.T, app *ui.App, dir string) *FrontEnd {
	t.Helper()
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := secondSaveFiles(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("mission SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	cold, _ := secondMissionLoadOf(t, secondDropMission(), dir, entries[0].Name)
	if cold.live == nil {
		t.Fatal("the cold LOAD opened no mission")
	}
	secondDropTables(cold)
	return cold
}

// A second-game dragon's death leaves its gold and one rolled book, scroll
// or potion in a sack at its cell (R2-ENGINE-940). A mission SAVE after the
// drop LOADs with the same sack and item, and the hero's next pick-up takes
// the item into his pack.
func TestSecondDeathDropCrossesSaveAndIsPickedUp(t *testing.T) {
	f, app, screen, dir := secondBagFrontOf(t, nil, secondDropMission())
	secondDropTables(f)
	secondBagEnter(t, f, app, screen)
	sack := secondDropKill(t, f)
	cold := secondDropLoad(t, app, dir)
	w := cold.live.world
	if got := w.Sacks(); len(got) != 1 || got[0].X != sack.X || got[0].Y != sack.Y || got[0].Gold != sack.Gold ||
		len(got[0].ItemInstances) != 1 || got[0].ItemInstances[0].Code != sack.ItemInstances[0].Code {
		t.Fatalf("LOAD sacks %+v, want %+v", got, sack)
	}
	id := cold.live.mission.ids[0]
	cold.live.pending = append(cold.live.pending, sim.PickUp(id, sim.CellPoint{X: sack.X, Y: sack.Y}))
	for range 3000 {
		cold.live.tick()
		if len(w.Sacks()) == 0 {
			break
		}
	}
	stacks, _ := w.CarriedStacks(id)
	held := false
	for _, s := range stacks {
		held = held || s.Code == sack.ItemInstances[0].Code
	}
	if len(w.Sacks()) != 0 || !held {
		hero, _ := w.Entity(id)
		t.Fatalf("after the pick-up: sacks %+v, pack %+v, hero at %d/%d", w.Sacks(), stacks, hero.X, hero.Y)
	}
}

// A living dragon keeps its roll across a mission SAVE and LOAD, and its
// death there drops the item.
func TestSecondDeathRollSurvivesLoad(t *testing.T) {
	f, app, screen, dir := secondBagFrontOf(t, nil, secondDropMission())
	secondDropTables(f)
	secondBagEnter(t, f, app, screen)
	cold := secondDropLoad(t, app, dir)
	secondDropKill(t, cold)
}
