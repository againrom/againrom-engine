package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestSourceActor1110AppChangedQuotientAndFreshNativeAction(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		f := currentPoolFixtureFront(t, 91, 92)
		app := f.App("source derive action")
		if fromMap {
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
		}
		sum := int32(17)
		a := &poolFixtureActor{human: true, mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, loadWords: &[4]int16{37, 7, 603, 301}, holdings: &holdingFixture{accumulator: &sum, items: []*holdingFixtureItem{{class: "Item", code: 0xe01, count: 3, kind: 3, weight: 5}}}}
		b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 10, maxHP: 100, loadWords: &[4]int16{19, 2, 0, 301}, holdings: &holdingFixture{}}
		originals := t.TempDir()
		if err := os.WriteFile(filepath.Join(originals, "game1110.sav"), poolFixtureSave(a, b), 0600); err != nil {
			t.Fatal(err)
		}
		store := SaveStore{Dir: t.TempDir()}
		save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
		app.SetSaveSeams(save, list, load)
		groundAppLoad(t, app, list, "game1110.sav")
		one, two := poolEntity(t, f.live.world, 91), poolEntity(t, f.live.world, 92)
		if one.Load != 603 || one.Speed != 37 || one.MaxHP != 100 {
			t.Fatal("LOAD derived current state", one.CurrentActorLoad())
		}
		if err := f.live.world.MoveCarried(one.ID, two.ID, 0xe01, 1); err != nil {
			t.Fatal(err)
		}
		one = poolEntity(t, f.live.world, 91)
		// Independent source stats: Body30, Reaction20, Mind10, Spirit5,
		// XP0 and zero modifiers. HPmax=trunc(60*(1+1.1^30/100))=70.
		if one.Load != 13 || one.Capacity != 301 || one.Speed != 16 || one.HP != 10 || one.MaxHP != 70 || one.ToHit != 4 || one.Defence != 6 || one.ActorLoad.Source.Class != 2 {
			t.Fatal("App transfer did not publish whole source sheet", one)
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		entries, err := store.List()
		if err != nil || len(entries) != 1 {
			t.Fatal(entries, err, app.HeadlessMessage())
		}
		fresh := currentPoolFixtureFront(t, 91, 92)
		freshApp := fresh.App("fresh source action")
		fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
		freshApp.SetSaveSeams(fs, fl, ff)
		groundAppLoad(t, freshApp, fl, localOriginalSaveToken(entries[0].Name))
		if fresh.live.world.Hash() != f.live.world.Hash() {
			currentMenuWorldDiagnostics(t, f.live.world, fresh.live.world)
			t.Fatal("fresh native LOAD changed source")
		}
		for _, w := range []*sim.World{f.live.world, fresh.live.world} {
			if err := w.MoveCarried(one.ID, two.ID, 0xe01, 1); err != nil {
				t.Fatal(err)
			}
		}
		if fresh.live.world.Hash() != f.live.world.Hash() || poolEntity(t, fresh.live.world, 91).Load != 10 {
			t.Fatal("next native source action differs")
		}
	}
}

func TestActorLoad1110CityCurrentCardAndDistinctStagingReturn(t *testing.T) {
	f := city1102Front(t, false)
	check := func(load, accumulator int32) {
		t.Helper()
		member := trainingPartyMember(t, f, "hero")
		s := member.Carry.LiveLoad
		panel := partyPanelSubject(member, f.Table)
		if s == nil || s.Load != load || s.Inventory.Accumulator != accumulator || s.Capacity != 411 || s.Inventory.OwnWeight != 40 ||
			panel.Weight != int(load) || panel.Speed != 19 || mapload.PartyLoad(member, f.Table) != load {
			t.Fatalf("city current state/card differs: %+v %+v", s, panel)
		}
	}
	// city1099Fixture writes current load90, own40, cap411, speed19; the
	// city1102 item constructor writes five units weighing7 and accumulator35.
	// The literal current90 is intentionally not the refreshed57.
	check(90, 35)
	screen := f.TownScreen().(*townScreen)
	if action := screen.shopUnequipDoll(12); action.Msg != "" {
		t.Fatal("empty worn slot was not a no-op", action)
	}
	check(90, 35)
	city1102Take(t, f, 1)
	check(90, 28) // source-to-tray changes the container, not the current word
	screen.shopClear()
	check(90, 35) // bulk return is not individual return's helper call
	reloadTrainingCity(t, f)
	check(90, 35)
	city1102Take(t, f, 1)
	check(90, 28)
	f.TownScreen().(*townScreen).shopOffTable(0, false)
	check(57, 35) // individual return refreshes own40 + trunc(35/2)
}

// The literal archive writer supplies words independently of SAV projection,
// item tables and load derivation. These are synthetic controls, not a claim
// that an original runtime produced or consumed this sequence.
func TestActorLoad1110BothAppDoorsKeepCurrentWordsAndOrderedNextTransfer(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		t.Run(map[bool]string{false: "main menu", true: "mission menu"}[fromMap], func(t *testing.T) {
			index, sourceSum, targetSum := uint32(1), int32(17), int32(-6)
			item := func(code, count uint16, weight int16) *holdingFixtureItem {
				return &holdingFixtureItem{class: "Item", code: code, count: count, kind: 3, weight: weight}
			}
			a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100,
				loadWords: &[4]int16{18, 7, 53, 301}, holdings: &holdingFixture{accumulator: &sourceSum,
					items: []*holdingFixtureItem{item(0x0e01, 2, 5), item(0x0e02, 1, 4), item(0x0e01, 1, 5)}}}
			b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 10, maxHP: 100,
				loadWords: &[4]int16{19, 2, 0, 301}, holdings: &holdingFixture{insertIndex: &index, accumulator: &targetSum,
					items: []*holdingFixtureItem{item(0x0e02, 1, 4), item(0x0e03, 1, 3)}}}
			f := currentPoolFixtureFront(t, 91, 92)
			app := f.App("actor load literals")
			if fromMap {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			originals := t.TempDir()
			payload := poolFixtureSave(a, b)
			if err := os.WriteFile(filepath.Join(originals, "game1110.sav"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game1110.sav")
			first, second := poolEntity(t, f.live.world, 91), poolEntity(t, f.live.world, 92)
			if f.live.world.Tick() != rawSavedSubTick1112(t, payload) || first.Load != 53 || second.Load != 0 || first.Capacity != 301 ||
				first.ActorLoad.OwnWeight != 7 || first.ActorLoad.Accumulator != 17 || first.ActorLoad.InsertIndex != 10000 ||
				second.ActorLoad.Accumulator != -6 || second.ActorLoad.InsertIndex != 1 {
				t.Fatalf("first read changed literal current words: %+v %+v", first.CurrentActorLoad(), second.CurrentActorLoad())
			}
			if v, ok := first.RetainedHumanSpeed(); !ok || v != 18 {
				t.Fatal("source current speed lost", v, ok)
			}
			stacks, _ := f.live.world.CarriedStacks(first.ID)
			if len(stacks) != 3 || stacks[0].Count != 2 || stacks[2].Count != 1 {
				t.Fatal("LOAD merged source element boundaries", stacks)
			}
			if err := f.live.world.MoveCarried(first.ID, second.ID, 0x0e01, 1); err != nil {
				t.Fatal(err)
			}
			first, second = poolEntity(t, f.live.world, 91), poolEntity(t, f.live.world, 92)
			if first.ActorLoad.Accumulator != 12 || first.Load != 13 || second.ActorLoad.Accumulator != -1 || second.Load != 2 {
				t.Fatal("next transfer did not update both signed accumulators/current words", first.CurrentActorLoad(), second.CurrentActorLoad())
			}
			for _, e := range []sim.Entity{first, second} {
				if _, ok := e.RetainedHumanSpeed(); !ok {
					t.Fatal("equal quotient retired source speed", e.ID)
				}
			}
			stacks, _ = f.live.world.CarriedStacks(second.ID)
			if len(stacks) != 3 || stacks[0].Code != 0x0e02 || stacks[1].Code != 0x0e01 || stacks[2].Code != 0x0e03 {
				t.Fatal("destination ignored saved insertion index", stacks)
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 {
				t.Fatal("native save", entries, err)
			}
			fresh := currentPoolFixtureFront(t, 91, 92)
			freshApp := fresh.App("fresh native actor load")
			fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(fs, fl, ff)
			groundAppLoad(t, freshApp, fl, entries[0].Name)
			if fresh.live.world.Hash() != f.live.world.Hash() || !reflect.DeepEqual(poolEntity(t, fresh.live.world, 91).CurrentActorLoad(), first.CurrentActorLoad()) {
				t.Fatal("native fresh LOAD changed current state")
			}
		})
	}
}
