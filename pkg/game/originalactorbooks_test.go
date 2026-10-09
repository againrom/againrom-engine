package game

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func actorBookFixture(id uint16, book ...*poolFixtureSpell) *poolFixtureActor {
	return &poolFixtureActor{mapID: id, cell: 0x0605 + id - 91, hp: 31, maxHP: 31, mana: 17, maxMana: 50, book: book}
}

func TestActorSpellbooks1105AllPlayersClassesSparseAndAliases(t *testing.T) {
	shared := &poolFixtureSpell{id: 1, rangeByte: 19, defensive: 2, cost: 65535}
	a := actorBookFixture(91, shared)
	b := actorBookFixture(92, &poolFixtureSpell{id: 1, rangeByte: 7, cost: 3}, nil, nil, nil, nil, &poolFixtureSpell{id: 6, rangeByte: 6, defensive: 255, cost: 32768})
	b.human = true
	c := actorBookFixture(93, shared)
	c.humanoid = true
	d, e := actorBookFixture(94), actorBookFixture(95, []*poolFixtureSpell{}...)
	// Explicit non-nil empty slice means present-empty in the literal writer.
	e.book = []*poolFixtureSpell{}
	first := &poolFixturePlayer{groups: [][]*poolFixtureActor{{nil, a, a}}}
	last := &poolFixturePlayer{groups: [][]*poolFixtureActor{{b, nil, c}, {d, e, a}}}
	payload := savedContainer(poolFixtureBody([]*poolFixturePlayer{nil, first, nil, last, first}, nil))
	f, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	books, err := f.ActorSpellbooks()
	if err != nil || len(books) != 5 {
		t.Fatalf("all Player books %+v %v", books, err)
	}
	for i, a := range []*poolFixtureActor{a, b, c, d, e} {
		got := books[i]
		if got.Off != a.off || got.MapUnitID != a.mapID || got.Cell != a.cell || got.RuntimeID == 0 || got.HP != 31 {
			t.Fatalf("actor identity %+v", got)
		}
	}
	if books[0].KnownSpells() != 1<<1 || books[1].KnownSpells() != 1<<1|1<<6 || books[1].SpellCount != 7 ||
		books[2].Spells[0] != books[0].Spells[0] || books[3].HasSpellbook || !books[4].HasSpellbook || books[4].KnownSpells() != 0 {
		t.Fatalf("sparse/presence/source records %+v", books)
	}
	if books[0].Spells[0].Range != 19 || books[0].Spells[0].ManaCost != 65535 || books[1].Spells[1].Defensive != 255 {
		t.Fatal("source parameters lost")
	}
	books[0].Spells[0].Range = 99
	if books[2].Spells[0].Range != 19 {
		t.Fatal("shared source projections alias mutable values")
	}
	party, _, err := f.PartyWalk()
	if err != nil || len(party) != 2 || party[0].Off != a.off || party[1].Off != a.off {
		t.Fatalf("raw first-Player provenance changed: %+v %v", party, err)
	}
	unique, err := f.Party()
	if err != nil || len(unique) != 1 || unique[0].Off != a.off || !reflect.DeepEqual(unique[0].Spells, party[0].Spells) {
		t.Fatalf("unique Party lost the saved book: %+v %v", unique, err)
	}
}

func TestActorSpellbooks1105MalformedLatePlayerIsAtomic(t *testing.T) {
	for _, kind := range []string{"zero id", "id29", "slot", "class", "flag", "count", "truncated"} {
		t.Run(kind, func(t *testing.T) {
			good := actorBookFixture(91, &poolFixtureSpell{id: 1, rangeByte: 7, cost: 3})
			spell := &poolFixtureSpell{id: 1, rangeByte: 9, cost: 4}
			bad := actorBookFixture(92, spell)
			switch kind {
			case "zero id":
				spell.id = 0
			case "id29":
				spell.id = 29
			case "slot":
				spell.id = 6
			case "class":
				spell.actorRef = true
			}
			body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{good}}}, {groups: [][]*poolFixtureActor{{bad}}}}, nil)
			switch kind {
			case "flag":
				body[bad.bookOff] = 2
			case "count":
				binary.LittleEndian.PutUint32(body[bad.bookOff+5:], 65537)
			case "truncated":
				body = body[:spell.off+4]
			}
			file := &sav.File{Body: body, Head: sav.Head{End: 75, PlayerCount: 2}}
			if partial, err := file.ActorSpellbooks(); !errors.Is(err, sav.ErrSpellbook) || len(partial) != 0 {
				t.Fatalf("partial malformed projection: %+v %v", partial, err)
			}
			f := poolFixtureFront(t, 91, 92)
			app := f.App("book-refusal")
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			payload := savedContainer(body)
			before, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			oldLive, oldHash := f.live, f.live.world.Hash()
			if _, _, err := f.RestoreOriginal(payload); !errors.Is(err, sav.ErrSpellbook) {
				t.Fatal("LOAD accepted late malformed book", err)
			}
			if _, _, err := loadOriginalMission(f, payload); !errors.Is(err, sav.ErrSpellbook) {
				t.Fatal("diagnostic LOAD accepted late malformed book", err)
			}
			after, _, err := f.Snapshot(true)
			if err != nil || f.live != oldLive || f.live.world.Hash() != oldHash || !reflect.DeepEqual(before, after) {
				t.Fatal("malformed LOAD changed old snapshot/hash")
			}
			// Syntactically intact but semantically invalid books appear in the
			// real picker, then refuse in its App LOAD callback.
			if kind == "flag" || kind == "count" || kind == "truncated" {
				return
			}
			originals := t.TempDir()
			if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			if err := headlessOpenLoad(app); err != nil {
				t.Fatal(err)
			}
			rows := list()
			if len(rows) != 1 {
				t.Fatalf("malformed book picker rows %+v", rows)
			}
			before, _, _ = f.Snapshot(true)
			if err := app.HeadlessActivate(rows[0].Label); err != nil {
				t.Fatal(err)
			}
			after, _, err = f.Snapshot(true)
			if err != nil || app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" || f.live != oldLive || f.live.world.Hash() != oldHash || !reflect.DeepEqual(before, after) {
				t.Fatal("App accepted malformed book or changed active game")
			}
		})
	}
}

func TestOriginalActorSpellbooks1105JoinBoundaries(t *testing.T) {
	entities := []sim.Entity{{ID: 0, MapUnitID: 91, X: 5, Y: 6, HP: 30, MaxHP: 30}, {ID: 1, MapUnitID: 92, X: 6, Y: 6, HP: 30, MaxHP: 30}, {ID: 2, MapUnitID: 93, X: 7, Y: 6, HP: 30, MaxHP: 30, OffMap: true}, {ID: 3, MapUnitID: 94, X: 8, Y: 6, HP: 0, MaxHP: 30}}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{Map: &alm.Map{Width: 40, Height: 40}, World: w, Party: []mapload.PartyMember{{Saved: &mapload.Saved{MapUnitID: 92}}}}
	valid := sav.ActorSpellbook{RuntimeID: 10, MapUnitID: 91, Cell: 0x0605, HP: 30, HasSpellbook: true, Spells: []sav.SavedSpell{{Slot: 1, ID: 1, Range: 7, Defensive: 2, ManaCost: 65535}}}
	sources := []sav.ActorSpellbook{valid}
	for _, mutate := range []func(*sav.ActorSpellbook){
		func(a *sav.ActorSpellbook) { a.MapUnitID = 92 }, func(a *sav.ActorSpellbook) { a.MapUnitID = 999 },
		func(a *sav.ActorSpellbook) { a.RuntimeID = 0 }, func(a *sav.ActorSpellbook) { a.MapUnitID = 0 },
		func(a *sav.ActorSpellbook) { a.Cell = 0xffff }, func(a *sav.ActorSpellbook) { a.HP = 0 }, func(a *sav.ActorSpellbook) { a.HP = -1 }, func(a *sav.ActorSpellbook) { a.Stage = 3 },
		func(a *sav.ActorSpellbook) { a.MapUnitID = 93 }, func(a *sav.ActorSpellbook) { a.MapUnitID = 94 },
	} {
		a := valid
		mutate(&a)
		sources = append(sources, a)
	}
	var report OriginalSaveResume
	if err := applyOriginalActorSpellbooks(ms, sources, &report); err != nil {
		t.Fatal(err)
	}
	if report.Books.Restored != 1 || report.Books.Party != 1 || report.Books.Excluded != 6 || report.Books.Unmatched != 3 {
		t.Fatalf("counts %+v", report.Books)
	}
	before := w.Hash()
	if err := applyOriginalActorSpellbooks(ms, []sav.ActorSpellbook{valid, valid}, &OriginalSaveResume{}); err == nil || !strings.Contains(err.Error(), "ambiguous") || before != w.Hash() {
		t.Fatalf("source ambiguity %v", err)
	}
	entities[1].MapUnitID = 91
	w, err = sim.NewWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	ms.World, ms.Party = w, nil
	before = w.Hash()
	if err := applyOriginalActorSpellbooks(ms, []sav.ActorSpellbook{valid}, &OriginalSaveResume{}); err == nil || !strings.Contains(err.Error(), "ambiguous") || before != w.Hash() {
		t.Fatalf("target ambiguity %v", err)
	}
}

func nonPartyBookFront1105(t *testing.T, ids ...uint16) *FrontEnd {
	if len(ids) == 0 {
		ids = []uint16{91, 92}
	}
	f := spellbookFront1096(t)
	f.Table.Humans.(dbCollection)[1].name = "NPC_fixture"
	row := f.Table.Humans.(dbCollection)[1].params
	row[0], row[1], row[2], row[3], row[8], row[16], row[19] = 30, 20, 30, 30, 10, 7, 4
	mapBytes := poolFixtureMap(ids...)
	for off := 20; off+20 <= len(mapBytes); {
		size := int(binary.LittleEndian.Uint32(mapBytes[off+8:]))
		if binary.LittleEndian.Uint32(mapBytes[off+12:]) == 6 {
			for i := range ids {
				binary.LittleEndian.PutUint16(mapBytes[off+20+i*70+8:], 7)
			}
			break
		}
		off += 20 + size
	}
	f.Archives = poolFixtureFrontMap(t, mapBytes).Archives
	for _, row := range f.Table.Spells.(dbCollection)[1:] {
		row.params[1], row.params[6], row.params[18] = 99, 1, 1
	}
	return f
}

func TestOriginalActorSpellbooksLoadClearsTemplateAndConstructsSourceOnlyActor(t *testing.T) {
	shared := &poolFixtureSpell{id: 1, rangeByte: 19, defensive: 255, cost: 32768}
	a, b, c := actorBookFixture(91, shared), actorBookFixture(92, shared), actorBookFixture(93)
	b.humanoid = true
	d, missing := actorBookFixture(94), actorBookFixture(999, shared)
	missing.cell = 0x0c0c
	d.human = true
	d.book = []*poolFixtureSpell{}
	first := &poolFixturePlayer{groups: [][]*poolFixtureActor{{nil, a, a}}}
	payload := savedContainer(poolFixtureBody([]*poolFixturePlayer{nil, first, nil, {groups: [][]*poolFixtureActor{{b, c, d, missing}}}, first}, nil))
	for _, diagnostic := range []bool{false, true} {
		f := nonPartyBookFront1105(t, 91, 92, 93, 94, 95)
		f.Table.Units = actorRegistryTable().Units
		var w *sim.World
		if diagnostic {
			ms, r, err := loadOriginalMission(f, payload)
			if err != nil || r.Books.Restored != 5 || r.Books.Absent != 1 || r.Books.Empty != 1 || r.Books.Unmatched != 0 {
				t.Fatalf("presence report %+v %v", r.Books, err)
			}
			w = ms.World
		} else {
			openSpellbookSave1096(t, f, payload)
			w = f.live.world
		}
		for _, id := range []uint16{91, 92} {
			e := poolEntity(t, w, id)
			if e.KnownSpells != 1<<1 || e.Book.Slots[0] != (sim.BookSpell{Range: 19, Defensive: 255, ManaCost: 32768}) {
				t.Fatal("temporary Unit/Humanoid book lost")
			}
		}
		for id, state := range map[uint16]sim.BookState{93: sim.BookAbsent, 94: sim.BookPresent} {
			e := poolEntity(t, w, id)
			if e.KnownSpells != 0 || e.Book != (sim.Spellbook{State: state}) {
				t.Fatal("template membership survived absent/empty book")
			}
		}
		if e := poolEntity(t, w, 95); e.Book.State != sim.BookLegacy || e.KnownSpells != 1<<1 {
			t.Fatal("unmatched map actor changed")
		}
		if e := poolEntity(t, w, 999); e.ID < 5 || e.SourceBinding.Class != 1 || e.KnownSpells != 1<<1 || e.Book.Slots[0] != (sim.BookSpell{Range: 19, Defensive: 255, ManaCost: 32768}) {
			t.Fatal("source-only actor or its saved book was dropped")
		}
	}
}

func TestOriginalActorSpellbooks1105BothDoorsNewCastAndNativeWindup(t *testing.T) {
	a := actorBookFixture(91, &poolFixtureSpell{id: 1, rangeByte: 7, cost: 3})
	b := actorBookFixture(92, &poolFixtureSpell{id: 1, rangeByte: 9, defensive: 2, cost: 65535})
	a.cell, b.cell, b.human = 0x0c0c, 0x0c0f, true
	payload := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}, {groups: [][]*poolFixtureActor{{b}}}}, nil))
	for _, diagnostic := range []bool{false, true} {
		f := withCurrentMenuDefinitions(t, nonPartyBookFront1105(t))
		store := SaveStore{Dir: t.TempDir()}
		app := f.App("nonparty-books")
		save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
		app.SetSaveSeams(save, list, load)
		var w *sim.World
		if diagnostic {
			ms, report, err := loadOriginalMission(f, payload)
			if err != nil || report.Books.Restored != 2 || report.Books.Spells != 2 {
				t.Fatalf("diagnostic %+v %v", report.Books, err)
			}
			w = ms.World
		} else {
			originals := t.TempDir()
			if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			save, list, load = f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game9999.sav")
			w = f.live.world
		}
		caster, target := poolEntity(t, w, 91), poolEntity(t, w, 92)
		if caster.KnownSpells != 1<<1 || caster.Book.Slots[0] != (sim.BookSpell{Range: 7, Defensive: 0, ManaCost: 3}) || target.Book.Slots[0] != (sim.BookSpell{Range: 9, Defensive: 2, ManaCost: 65535}) || caster.Book.State != sim.BookPresent {
			t.Fatal("source values lost at LOAD")
		}
		if reason := w.BookSpellRefusal(caster.ID, target.ID, 1); reason != "" {
			t.Fatal("saved range/cost refused", reason)
		}
		events := sim.StepObserved(w, []sim.Command{{Kind: sim.KindCast, Entity: caster.ID, X: int32(target.ID), Y: 1}})
		if spell, remaining, casting := w.CastingSpell(caster.ID); len(events) != 0 || !casting || spell != 1 || remaining == 0 {
			t.Fatal("new cast did not enter wind-up")
		}
		var back *sim.World
		if diagnostic {
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
				t.Fatalf("ordinary native SAVE %+v %v: %s", entries, err, app.HeadlessMessage())
			}
			fresh := withCurrentMenuDefinitions(t, nonPartyBookFront1105(t))
			freshApp := fresh.App("fresh-nonparty")
			xsave, xlist, xload := fresh.SaveSeams(store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(xsave, xlist, xload)
			groundAppLoad(t, freshApp, xlist, entries[0].Name)
			back = fresh.live.world
		}
		if w.Hash() != back.Hash() {
			currentMenuWorldDiagnostics(t, w, back)
			t.Fatal("native load changed wind-up")
		}
		released := false
		for range 256 {
			left, right := sim.StepObserved(w, nil), sim.StepObserved(back, nil)
			if !reflect.DeepEqual(left, right) || w.Hash() != back.Hash() {
				t.Fatal("native events/hash diverged")
			}
			for _, event := range left {
				if event.Caster == caster.ID && event.Spell == 1 {
					released = true
				}
			}
			if released {
				break
			}
		}
		if !released || poolEntity(t, w, 91).Mana != 14 || poolEntity(t, w, 92).HP >= 31 {
			t.Fatal("new book cast ignored saved parameters")
		}
		if poolEntity(t, w, 92).Book != target.Book {
			t.Fatal("cast changed other actor same-ID instance")
		}
		// Raw Defensive2 is not canonicalized to1; signed -1 costs credit
		// one mana on this separate, newly ordered book cast.
		if reason := w.BookSpellRefusal(target.ID, caster.ID, 1); reason != "" {
			t.Fatal(reason)
		}
		controlBytes, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var control sim.World
		if err := control.UnmarshalBinary(controlBytes); err != nil {
			t.Fatal(err)
		}
		sim.Step(w, []sim.Command{{Kind: sim.KindCast, Entity: target.ID, X: int32(caster.ID), Y: 1}})
		sim.Step(&control, nil)
		released = false
		for range 256 {
			sim.Step(&control, nil)
			for _, event := range sim.StepObserved(w, nil) {
				if event.Caster == target.ID && event.Spell == 1 {
					released = true
				}
			}
			if released {
				break
			}
		}
		if !released || poolEntity(t, w, 92).Mana != poolEntity(t, &control, 92).Mana+1 {
			t.Fatalf("saved signed cost lost: released %t mana %d tick %d", released, poolEntity(t, w, 92).Mana, w.Tick())
		}
	}
}
