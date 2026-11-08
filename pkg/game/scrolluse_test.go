package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

const (
	scrollUseOrdinary = 0x0e06
	scrollUseElven    = 0x0e23
)

// scrollUseTable is a second-game MagicItems table in the installed row
// shape: row 6 a Scroll and row 35 a SuperScroll of spell 1, shown as
// "Scroll" and "Elven Scroll", at powers 50 and 100.
func scrollUseTable() *mapload.Table {
	spells := make(dbCollection, 2)
	spells[1] = dbEntry{name: "Fire Arrow", params: make([]int32, 22)}
	spells[1].params[1], spells[1].params[2], spells[1].params[6], spells[1].params[8] = 19, 1, 7, 1
	spells[1].params[16], spells[1].params[17] = 4, 8
	rows := make(dbCollection, 36)
	rows[6] = dbEntry{name: "Scroll Fire Arrow", params: []int32{100}, strings: []string{"castSpell=Fire_Arrow:50"}}
	rows[35] = dbEntry{name: "SuperScroll Fire Arrow", params: []int32{1000}, strings: []string{"castSpell=Fire_Arrow:100"}}
	return &mapload.Table{Spells: spells, MagicItems: rows,
		Names: data.ItemNames{scrollUseOrdinary: "Scroll", scrollUseElven: "Elven Scroll"}}
}

func scrollUseCast(code uint16, price int32, power uint32) sim.ItemInstance {
	return sim.ItemInstance{Code: code, Kind: 4, Price: price, Effects: []sim.ItemEffect{{Kind: 41, Operand: 1 | power<<16}}}
}

// An Elven Scroll row (SuperScroll) builds the scroll kind with its row's
// spell and power, so its card is an ordinary scroll's card under its own
// name and at its own power (R2-ENGINE-709, R2-ENGINE-712). The ordinary
// Scroll row and the first game's row names build what they built before.
func TestAnElvenScrollRowBuildsAScrollWhoseCardNamesItsSpell(t *testing.T) {
	table := scrollUseTable()
	words := testSpellPopupWords()
	words.ItemSpellOf = "of"
	elven := mapload.ItemInstanceFromCode(scrollUseElven, table)
	if want := scrollUseCast(scrollUseElven, 1000, 100); !sim.ItemEqual(elven, want) || elven.Price != want.Price {
		t.Fatalf("Elven Scroll row built %+v, want %+v", elven, want)
	}
	if id, power, ok := sim.ScrollSpell(elven); !ok || id != 1 || power != 100 {
		t.Fatalf("Elven Scroll reads spell %d power %d ok %v, want spell 1 at 100", id, power, ok)
	}
	card := itemInstanceInfoLines(elven, table, words)
	if want := "Elven Scroll of Fire Arrow"; len(card) == 0 || card[0] != want {
		t.Fatalf("Elven Scroll card %q, want first line %q", card, want)
	}
	// The card's spell lines are the ordinary scroll's at power 100.
	same := itemInstanceInfoLines(scrollUseCast(scrollUseOrdinary, 100, 100), table, words)
	if !slices.Equal(card[1:], same[1:]) || !slices.Contains(card, "Range: 10") {
		t.Fatalf("Elven Scroll card %q, want the power-100 lines %q", card, same)
	}

	// Loss control: the ordinary row and its card are unchanged.
	ordinary := mapload.ItemInstanceFromCode(scrollUseOrdinary, table)
	if want := scrollUseCast(scrollUseOrdinary, 100, 50); !sim.ItemEqual(ordinary, want) || ordinary.Price != want.Price {
		t.Fatalf("Scroll row built %+v, want %+v", ordinary, want)
	}
	if got := itemInstanceInfoLines(ordinary, table, words); len(got) == 0 || got[0] != "Scroll of Fire Arrow" || !slices.Contains(got, "Range: 8") {
		t.Fatalf("Scroll card %q", got)
	}

	// Loss control: the first game's row names keep their kinds.
	first := dbCollection{{}, {name: "Scroll Fire", params: []int32{3000}, strings: []string{"1"}},
		{name: "Quest Tensers Scroll", params: []int32{-1}, strings: []string{"1"}},
		{name: "Potion Healing", params: []int32{50}}, {name: "Book Fire", params: []int32{1000}}}
	firstTable := &mapload.Table{Spells: table.Spells, MagicItems: first}
	for row, kind := range []uint8{1: 4, 2: 0, 3: 3, 4: 5} {
		if row == 0 {
			continue
		}
		got := mapload.ItemInstanceFromCode(uint16(0x0e00|row), firstTable)
		if got.Kind != kind || len(got.Effects) != 0 {
			t.Errorf("first-game row %d %q built kind %d effects %v, want kind %d and none", row, first[row].name, got.Kind, got.Effects, kind)
		}
	}
}

// scrollUseMission is a mission of either game whose party is one fighter
// (7, no mana, no book) carrying item, beside an enemy (8) four cells off.
func scrollUseMission(t *testing.T, second bool, item sim.ItemInstance) *mapWorld {
	t.Helper()
	spells := []sim.SpellRule{{ID: 1, ManaCost: 5, School: 1, MaxRange: 5, DamageMin: 10, DamageMax: 10, TargetsUnit: true, Damaging: true}}
	var script *sim.Script
	if second {
		sim.AssignSecondGameArms(spells)
		s, err := sim.NewROM2Script(nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		script = s
	}
	fighter := sim.Entity{ID: 7, Owner: sim.SelfSlot, X: 3, Y: 3, HP: 100, MaxHP: 100, TypeID: sim.HumanTypeID, TokenSize: 1}
	enemy := sim.Entity{ID: 8, Owner: 2, X: 7, Y: 3, HP: 100, MaxHP: 100, TokenSize: 1}
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{fighter, enemy}, script, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{item}}}, spells)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.SpellGame() == rules.SpellGameSecond; got != second {
		t.Fatalf("world plays the second game %v, want %v", got, second)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Hero: eqHero()}}, Start: mapload.Start{IDs: []sim.EntityID{7}}}
	return openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
}

// A fighter reads a scroll by the mage's own action in both games: the pack
// use arms the item cast and the map release (the viewer's item-cast sink)
// orders it; the fighter pays no mana and the scroll's own spell and power
// land (ITEM-USE-112; the second game by owner direction). The Elven Scroll
// case is the second game's SuperScroll row.
func TestAFighterReadsAScrollThroughThePackUse(t *testing.T) {
	table := scrollUseTable()
	for _, tc := range []struct {
		name   string
		second bool
		item   sim.ItemInstance
	}{
		{"first game", false, scrollUseCast(0x0e10, 300, 50)},
		{"second game", true, mapload.ItemInstanceFromCode(scrollUseOrdinary, table)},
		{"second game Elven Scroll", true, mapload.ItemInstanceFromCode(scrollUseElven, table)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := scrollUseMission(t, tc.second, tc.item)
			if !mw.view.SelectUnit(7) {
				t.Fatal("the fighter cannot be selected")
			}
			mw.enqueueEquip(0)
			if len(mw.pending) != 0 {
				t.Fatalf("the pack use queued %+v, want the map arm alone", mw.pending)
			}
			mw.useScroll(7, 0, fmt.Sprintf("%#v", tc.item), 8, 0, 0, false)
			if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindUseScroll {
				t.Fatalf("the release queued %+v, want one scroll use", mw.pending)
			}
			for n := 0; n < 120; n++ {
				mw.tick()
				if items, _ := mw.world.CarriedItems(7); len(items) == 0 && len(mw.world.ScrollCasts()) == 0 {
					break
				}
			}
			items, _ := mw.world.CarriedItems(7)
			fighter, enemy := releaseEntity(t, mw, 7), releaseEntity(t, mw, 8)
			if len(items) != 0 || len(mw.world.ScrollCasts()) != 0 || enemy.HP >= 100 || fighter.Mana != 0 {
				t.Fatalf("pack %+v casts %+v enemy health %d fighter mana %d", items, mw.world.ScrollCasts(), enemy.HP, fighter.Mana)
			}
		})
	}
}
