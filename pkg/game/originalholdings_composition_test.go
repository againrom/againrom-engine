package game

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestOriginalHoldings1108RearmPreservesSecondPairThenBooksAndPools(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 1, MapUnitID: 91, X: 5, Y: 6, HP: 20, MaxHP: 20,
			SecondBase: 13, SecondSpread: 7, KnownSpells: 1 << 6}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 1, Items: []uint16{0x0e06, 0x0e06, 0x0e06}}})
	if err != nil {
		t.Fatal(err)
	}
	// Cell triggers are restored before actors. The detached stock staging
	// world must retain that already-installed canonical overlay unchanged.
	tails := []sim.CellTail{{X: 8, Y: 9, Bytes: [6]byte{13, 1, 19, 61, 21, 63}}}
	if err := w.ImportOriginalCellTails(tails); err != nil {
		t.Fatal(err)
	}
	member := mapload.PartyMember{Hero: data.Hero{Body: 30, Reaction: 20, Mind: 10, Spirit: 5}}
	ms := &Mission{Map: &alm.Map{Width: 40, Height: 40}, World: w,
		Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{1: member}}}
	holdings := []sav.ActorHoldings{{MapUnitID: 91, Cell: 0x0605, HP: 7}}
	if err := restoreOriginalActorStock(ms, holdings, nil, &OriginalSaveResume{}); err != nil {
		t.Fatal(err)
	}
	if e := poolEntity(t, w, 91); e.MaxHP == 20 || e.MaxHP == 31 || e.KnownSpells != 1<<6 {
		t.Fatal("fixture did not separately exercise Rearm and the pre-import book")
	}
	var report OriginalSaveResume
	if err := restoreOriginalActors(ms, holdings,
		[]sav.ActorPools{{MapUnitID: 91, Cell: 0x0605, HP: 7, MaxHP: 31, Mana: 5, MaxMana: 23}},
		[]sav.ActorSpellbook{{MapUnitID: 91, RuntimeID: 1, Cell: 0x0605, HP: 7, HasSpellbook: true,
			SpellCount: 2, Spells: []sav.SavedSpell{{Slot: 1, ID: 1, Range: 7, ManaCost: 3}}}}, nil, &report, nil); err != nil {
		t.Fatal(err)
	}
	e := poolEntity(t, w, 91)
	assertPools(t, e, [4]int32{7, 31, 5, 23})
	if e.SecondBase != 13 || e.SecondSpread != 7 || e.KnownSpells != 1<<1 ||
		e.Book.State != sim.BookPresent || e.Book.Slots[0] != (sim.BookSpell{Range: 7, Defensive: 0, ManaCost: 3}) {
		t.Fatalf("shared handoff lost the second pair or source book: %+v", e)
	}
	pack, _ := w.CarriedStacks(e.ID)
	if len(pack) != 0 || report.Stocked != 1 || report.Books.Restored != 1 || report.PoolsRestored != 1 || w.Tick() != 0 ||
		!reflect.DeepEqual(w.CellTails(), tails) {
		t.Fatalf("shared handoff stock/count/tick: %+v %+v", pack, report)
	}
}

func TestOriginalHoldings1108BooksBothDoorsNewCastAndNativeContinuation(t *testing.T) {
	a := actorBookFixture(91, &poolFixtureSpell{id: 1, rangeByte: 7, cost: 3})
	b := actorBookFixture(92, &poolFixtureSpell{id: 1, rangeByte: 9, cost: 4})
	a.cell, b.cell, a.human, b.human = 0x0c0c, 0x0c0f, true, true
	a.holdings = &holdingFixture{items: []*holdingFixtureItem{{class: "Item", code: 0x0e06, count: 3, kind: 3, price: -50}}}
	b.holdings = &holdingFixture{}
	payload := poolFixtureSave(a, b)
	for _, door := range []string{"App", "diagnostic"} {
		t.Run(door, func(t *testing.T) {
			f := withCurrentMenuDefinitions(t, nonPartyBookFront1105(t))
			store := SaveStore{Dir: t.TempDir()}
			app := f.App("1108-composed-load")
			var w *sim.World
			if door == "diagnostic" {
				ms, report, err := loadOriginalMission(f, payload)
				if err != nil || report.Stocked != 2 || report.Books.Restored != 2 || report.PoolsRestored != 2 {
					t.Fatalf("diagnostic %+v %v", report, err)
				}
				w = ms.World
			} else {
				originals := t.TempDir()
				if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0600); err != nil {
					t.Fatal(err)
				}
				save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
				app.SetSaveSeams(save, list, load)
				groundAppLoad(t, app, list, "game9999.sav")
				w = f.live.world
			}
			caster, target := poolEntity(t, w, 91), poolEntity(t, w, 92)
			pack, _ := w.CarriedStacks(caster.ID)
			other, _ := w.CarriedStacks(target.ID)
			if len(pack) != 1 || pack[0].Count != 3 || pack[0].Price != -50 || len(other) != 0 ||
				caster.Book.Slots[0] != (sim.BookSpell{Range: 7, Defensive: 0, ManaCost: 3}) || w.Tick() != rawSavedSubTick1112(t, payload) {
				t.Fatalf("composed LOAD lost stock/book or advanced time: %+v %+v", pack, caster)
			}
			assertPools(t, caster, [4]int32{31, 31, 17, 50})
			if err := w.MoveCarried(caster.ID, target.ID, 0x0e06, 2); err != nil {
				t.Fatal(err)
			}
			if reason := w.BookSpellRefusal(caster.ID, target.ID, 1); reason != "" {
				t.Fatal(reason)
			}
			sim.Step(w, []sim.Command{{Kind: sim.KindCast, Entity: caster.ID, X: int32(target.ID), Y: 1}})
			if _, remaining, casting := w.CastingSpell(caster.ID); !casting || remaining == 0 {
				t.Fatal("new order did not enter saved-book wind-up")
			}
			var back *sim.World
			if door == "diagnostic" {
				raw, err := w.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				back = new(sim.World)
				mapload.BindSourceDerive(back)
				if err := back.UnmarshalBinary(raw); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessGameMenuAction("save"); err != nil {
					t.Fatal(err)
				}
				entries, err := store.List()
				if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
					t.Fatalf("ordinary SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
				}
				fresh := withCurrentMenuDefinitions(t, nonPartyBookFront1105(t))
				freshApp := fresh.App("1108-composed-fresh")
				save, list, load := fresh.SaveSeams(store, OriginalStore{}, nil)
				freshApp.SetSaveSeams(save, list, load)
				groundAppLoad(t, freshApp, list, entries[0].Name)
				back = fresh.live.world
			}
			if w.Hash() != back.Hash() {
				currentMenuWorldDiagnostics(t, w, back)
				t.Fatal("native LOAD changed stock or pending cast")
			}
			released := false
			for range 256 {
				left, right := sim.StepObserved(w, nil), sim.StepObserved(back, nil)
				if !reflect.DeepEqual(left, right) || w.Hash() != back.Hash() {
					t.Fatal("composed native continuation diverged")
				}
				for _, event := range left {
					if event.Caster == caster.ID && event.Spell == 1 {
						released = true
					}
				}
			}
			pack, _ = back.CarriedStacks(caster.ID)
			other, _ = back.CarriedStacks(target.ID)
			if !released || len(pack) != 1 || pack[0].Count != 1 || pack[0].Price != -50 ||
				len(other) != 1 || other[0].Count != 2 || other[0].Price != -50 {
				t.Fatal("new cast/transfer did not survive native continuation")
			}
		})
	}
}

func TestOriginalHoldings1108LatePoolRefusalKeepsStockBookAndLiveSession(t *testing.T) {
	f := withCurrentMenuDefinitions(t, nonPartyBookFront1105(t), 5)
	app := f.App("1108-composed-refusal")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	a := actorBookFixture(91, &poolFixtureSpell{id: 1, rangeByte: 7, cost: 3})
	b := actorBookFixture(92, &poolFixtureSpell{id: 1, rangeByte: 9, cost: 4})
	a.holdings = &holdingFixture{items: []*holdingFixtureItem{{class: "Item", code: 0x0e06, count: 3, kind: 3, price: -50}}}
	b.holdings = &holdingFixture{}
	a.profile, b.profile = literalProfile1107(), literalProfile1107()
	// Prove the same holdings/books reach a complete handoff before changing
	// only the later pool word. No parser error substitutes for staging rollback.
	ms, report, err := loadOriginalMission(f, poolFixtureSave(a, b))
	if err != nil || ms == nil || report.Stocked != 2 || report.Books.Restored != 2 || report.ProfilesRestored != 2 || report.PoolsRestored != 2 {
		t.Fatalf("valid control: %+v %v", report, err)
	}
	assertCurrent1107(t, poolEntity(t, ms.World, 91))
	assertCurrent1107(t, poolEntity(t, ms.World, 92))
	b.mana = b.maxMana + 1
	payload := poolFixtureSave(a, b)
	// A refused LOAD returns no mission and no report; the pool word refuses it.
	ms, _, err = loadOriginalMission(f, payload)
	if err == nil || ms != nil || !strings.Contains(err.Error(), "pools") {
		t.Fatalf("late pool refusal: %v", err)
	}
	f.Offered = 77
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	oldLive, oldTown, oldHash := f.live, f.Town, f.live.world.Hash()
	if _, _, err := f.RestoreOriginal(payload); err == nil || !strings.Contains(err.Error(), "pools") {
		t.Fatal("FrontEnd accepted invalid late pool", err)
	}
	originals := t.TempDir()
	if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
	app.SetSaveSeams(save, list, load)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	rows := list()
	if len(rows) != 1 {
		t.Fatalf("LOAD rows: %+v", rows)
	}
	if err := app.HeadlessActivate(rows[0].Label); err != nil {
		t.Fatal(err)
	}
	after, _, err := f.Snapshot(true)
	if err != nil || app.Screen() != ui.ScreenLoad || !strings.Contains(app.HeadlessMessage(), "pools") ||
		f.live != oldLive || f.Town != oldTown || f.live.world.Hash() != oldHash || f.Offered != 77 || !reflect.DeepEqual(before, after) {
		t.Fatal("late pool refusal changed old stock/book/session", err)
	}
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	fresh := withCurrentMenuDefinitions(t, nonPartyBookFront1105(t), 5)
	open, town, err := fresh.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("post-refusal current SAV LOAD", town, err)
	}
	if err := fresh.App("post-refusal current SAV").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if fresh.live.world.Hash() != oldHash || fresh.Offered != f.Offered {
		t.Fatal("post-refusal SAVE changed the previous game")
	}
	for range 20 {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("post-refusal continuation changed")
		}
	}
}
