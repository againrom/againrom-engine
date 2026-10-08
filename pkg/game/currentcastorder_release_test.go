package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// castOrderSession opens a cold front end, loads the one SAV in dir through
// the main-menu load window and pauses the mission.
func castOrderSession(t *testing.T, dir string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("cast order SAV")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal("main-menu LOAD:", err)
	}
	rows := app.HeadlessRows()
	if app.Screen() != ui.ScreenLoad || len(rows) != 1 {
		t.Fatalf("LOAD lists %v on %s, want one file", rows, app.Screen())
	}
	if err := app.HeadlessActivate(rows[0].Text); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatalf("LOAD refused the file: %v %s", err, app.HeadlessMessage())
	}
	if _, kind, up := f.LiveNotice(); up && kind == ui.NoticeSuccess {
		before := activePauseSampleNow(t, f)
		live, mission, state := f.live, f.live.mission, f.live.mission.state
		number, liveMission := mission.number, f.liveMission
		town, offered := f.Town, f.Offered
		townOpen, townDone, townGold := town.Open(), town.Done(number), town.Gold()
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal("Victory Continue through Escape:", err)
		}
		if app.Screen() != ui.ScreenMap || app.HeadlessNoticeOpen() || f.live != live || f.live.mission != mission || mission.state != state || mission.number != number || f.liveMission != liveMission || f.Town != town || f.Offered != offered || town.Open() != townOpen || town.Done(number) != townDone || town.Gold() != townGold {
			t.Fatal("Victory Continue changed mission/context or failed to close only the notice")
		}
		after := activePauseSampleNow(t, f)
		activePauseCompare(t, after, before)
		t.Logf("Victory Escape: mission=%d context unchanged; World/hash/queue/commanded/View equal; PlayerPaused=%v->%v PeriodUS=%d->%d Unpaced=%v->%v", number, before.View.PlayerPaused, after.View.PlayerPaused, before.View.PeriodUS, after.View.PeriodUS, before.View.Unpaced, after.View.Unpaced)
	}
	releasePauseMission(t, f, app)
	return f, app
}

// castOrderF2Save writes the current mission through F2 under name into a new
// directory and returns that directory and the file's bytes.
func castOrderF2Save(t *testing.T, f *FrontEnd, app *ui.App, name string) (string, []byte) {
	t.Helper()
	before := f.live.world.Hash()
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("F2 SAVE", err, app.Screen())
	}
	dir := t.TempDir()
	if err := app.HeadlessSaveEdit(dir, name, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != before {
		t.Fatal("F2 SAVE changed the current World")
	}
	raw, err := os.ReadFile(filepath.Join(dir, name+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	return dir, raw
}

// castOrderWire decodes the caster's action and order block from a SAV and
// checks what the original restores a cast from: action 0xd with order 8 and
// the target key at a unit, 0xe with order 9 at a cell, progress 2, and order
// byte +0x5c (AI-RETREAT-272, DIV-1491).
func castOrderWire(t *testing.T, raw []byte, caster, target sim.EntityID, atCell bool) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("action supplement", err)
	}
	r := castOrderRecord(t, doc, a, caster)
	order := savedRecordRawForTest(t, r, "U158")
	action, pending, flag, key := uint32(0xd), byte(8), byte(1), uint32(0)
	if atCell {
		action, pending, flag = 0xe, 9, 0
	} else {
		key = savedRecordValueForTest(t, castOrderRecord(t, doc, a, target), "Identity")
	}
	got := binary.LittleEndian.Uint32(savedRecordRawForTest(t, r, "U54"))
	aim := binary.LittleEndian.Uint32(order[0x28:])
	if got != action || order[8] != pending || order[9] != 2 || order[0x5c] != flag || aim != key {
		t.Fatalf("cast wire: action %#x order %d progress %d flag %d target %#x; want %#x %d 2 %d %#x",
			got, order[8], order[9], order[0x5c], aim, action, pending, flag, key)
	}
	return order[:0x60]
}

func castOrderBook(w *sim.World, caster sim.EntityID) (sim.BookContinuation, bool) {
	for _, c := range w.Actions().Books {
		if c.Caster == caster {
			return c, true
		}
	}
	return sim.BookContinuation{}, false
}

// castOrderLockstep runs ordinary ticks on the uninterrupted session f and the
// cold-loaded session g until done reports true for f. Both must keep one
// World hash and finish on the same tick.
func castOrderLockstep(t *testing.T, f, g *FrontEnd, limit int, done func(*FrontEnd, []sim.CastEvent) bool) int {
	t.Helper()
	for tick := 1; tick <= limit; tick++ {
		var mine, cold []sim.CastEvent
		f.live.tickWithCastSink(func(ev []sim.CastEvent) { mine = append(mine, ev...) })
		g.live.tickWithCastSink(func(ev []sim.CastEvent) { cold = append(cold, ev...) })
		if f.live.world.Hash() != g.live.world.Hash() {
			logCurrentCarrierDiff(t, f.live.world, g.live.world)
			t.Fatalf("cold LOAD diverged from the uninterrupted session at tick %d", tick)
		}
		a, b := done(f, mine), done(g, cold)
		if a != b {
			t.Fatalf("tick %d: uninterrupted session done %v, cold LOAD %v", tick, a, b)
		}
		if a {
			return tick
		}
	}
	t.Fatalf("not done within %d ticks", limit)
	return 0
}

func castOrderName(f *FrontEnd, id sim.EntityID) string {
	for i, member := range f.live.mission.ids {
		if member == id {
			return f.live.mission.party[i].Name
		}
	}
	return ""
}

// castOrderAtUnit picks the party member whose book holds the restorative
// spell and the first wounded ally that spell admits and the screen shows.
func castOrderAtUnit(t *testing.T, app *ui.App, live *mapWorld) (caster, target sim.EntityID, spell uint32) {
	t.Helper()
	for _, r := range live.world.Spells() {
		if r.Restorative {
			spell = uint32(r.ID)
		}
	}
	for _, id := range live.mission.ids {
		if e, ok := live.entity(id); ok && e.Alive() && spell != 0 && e.KnownSpells&(1<<spell) != 0 && caster == 0 {
			caster = id
		}
	}
	castOrderSelect(t, app, live, caster)
	for _, id := range live.mission.ids {
		e, _ := live.entity(id)
		if id == caster || !e.Alive() || e.HP >= e.MaxHP || live.world.BookSpellRefusal(caster, id, spell) != "" {
			continue
		}
		if _, _, err := app.HeadlessEntityPoint(uint32(id)); err == nil {
			return caster, id, spell
		}
	}
	t.Fatalf("no visible wounded ally admits spell %d from %d", spell, caster)
	return
}

// castOrderAtCell picks the party member who knows all 28 spells and the first
// ground spell the rules admit at a free cell three away.
func castOrderAtCell(t *testing.T, app *ui.App, live *mapWorld) (caster sim.EntityID, cell sim.CellPoint, spell uint32) {
	t.Helper()
	for _, id := range live.mission.ids {
		if e, ok := live.entity(id); ok && e.Alive() && bits.OnesCount32(e.KnownSpells) == 28 {
			caster = id
		}
	}
	castOrderSelect(t, app, live, caster)
	me, _ := live.entity(caster)
	occupied := map[sim.CellPoint]bool{}
	for _, e := range live.world.Entities() {
		occupied[sim.CellPoint{X: e.X, Y: e.Y}] = true
	}
	for _, d := range []sim.CellPoint{{X: 3}, {Y: 3}, {X: -3}, {Y: -3}} {
		at := sim.CellPoint{X: me.X + d.X, Y: me.Y + d.Y}
		for s := uint32(1); s <= 28 && !occupied[at]; s++ {
			// 26 is Teleport, which moves the caster instead of landing.
			if s != 26 && live.world.BookSpellCellRefusal(caster, at.X, at.Y, s) == "" {
				return caster, at, s
			}
		}
	}
	t.Fatalf("no ground spell from %d is admissible at a free cell three away", caster)
	return
}

// castOrderGroundClick clicks the first window pixel whose ground cell is at.
func castOrderGroundClick(t *testing.T, app *ui.App, at sim.CellPoint) {
	t.Helper()
	for py := 100; py < 560; py += 4 {
		for px := 160; px < 750; px += 4 {
			if x, y, err := app.HeadlessDropCell(px, py); err == nil && int32(x) == at.X && int32(y) == at.Y {
				for _, edge := range []string{"press", "release"} {
					if err := app.HeadlessPointer(edge, px, py); err != nil {
						t.Fatal(err)
					}
				}
				return
			}
		}
	}
	t.Fatalf("cell %v has no visible ground pixel", at)
}

func castOrderSelect(t *testing.T, app *ui.App, live *mapWorld, id sim.EntityID) {
	t.Helper()
	me, ok := live.entity(id)
	if !ok {
		t.Fatal("no caster")
	}
	inspectionCentre(live, int(me.X), int(me.Y))
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
}

// TestReleaseCastOrderSAVRestoresTheCast casts by pointer input from two
// corpus SAVs. At a unit: in game0018 (mission 40) the member whose book holds
// Heal casts it at the wounded ally it admits. At a cell: in game0021 (mission
// 10) the member who knows all 28 spells casts the first ground spell the
// rules admit three cells away. F2 SAVE follows while the cast charges, the
// order block is decoded, and the file is loaded cold. Ordinary ticks on both
// sessions run until the spell lands.
func TestReleaseCastOrderSAVRestoresTheCast(t *testing.T) {
	for _, tc := range []struct {
		name, file, sha string
		atCell          bool
	}{
		{"unit", "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b", false},
		{"cell", "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, raw := groundCorpusFile(t, tc.file, tc.sha)
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, filepath.Base(tc.file)), raw, 0o600); err != nil {
				t.Fatal(err)
			}
			f, app := castOrderSession(t, dir)
			live := f.live
			var caster, target sim.EntityID
			var cell sim.CellPoint
			var spell uint32
			if tc.atCell {
				caster, cell, spell = castOrderAtCell(t, app, live)
			} else {
				caster, target, spell = castOrderAtUnit(t, app, live)
			}
			if _, _, err := app.HeadlessSpellPoint(spell); err != nil {
				if err := app.HeadlessKey("book"); err != nil {
					t.Fatal(err)
				}
			}
			sx, sy, err := app.HeadlessSpellPoint(spell)
			if err != nil {
				t.Fatal(err)
			}
			for _, edge := range []string{"press", "release"} {
				if err := app.HeadlessPointer(edge, sx, sy); err != nil {
					t.Fatal(err)
				}
			}
			if tc.atCell {
				castOrderGroundClick(t, app, cell)
			} else {
				tx, ty, err := app.HeadlessEntityPoint(uint32(target))
				if err != nil {
					t.Fatal(err)
				}
				for _, edge := range []string{"press", "release"} {
					if err := app.HeadlessPointer(edge, tx, ty); err != nil {
						t.Fatal(err)
					}
				}
			}
			if len(live.pending) != 1 || (live.pending[0].Kind == sim.KindCast) == tc.atCell {
				t.Fatalf("pointer input queued %+v", live.pending)
			}
			c, ok := sim.BookContinuation{}, false
			for tick := 0; tick < 64 && c.Phase != 1; tick++ {
				live.tick()
				c, ok = castOrderBook(live.world, caster)
			}
			if !ok || c.Phase != 1 || c.Remaining < 2 || c.AtCell != tc.atCell || uint32(c.Spell) != spell || c.Target != target {
				t.Fatalf("cast is not charging at its input target: %+v", c)
			}
			savedDir, saved := castOrderF2Save(t, f, app, "cast "+tc.name)
			order := castOrderWire(t, saved, caster, target, tc.atCell)
			g, _ := castOrderSession(t, savedDir)
			if cold, _ := castOrderBook(g.live.world, caster); cold != c {
				t.Fatalf("cold LOAD cast %+v, saved %+v", cold, c)
			}
			tick := castOrderLockstep(t, f, g, 256, func(h *FrontEnd, events []sim.CastEvent) bool {
				for _, ev := range events {
					if ev.Caster != caster || uint32(ev.Spell) != spell || ev.AtCell != tc.atCell {
						continue
					}
					if tc.atCell && ev.ToX == cell.X && ev.ToY == cell.Y || !tc.atCell && ev.Target == target && ev.HealthRestored > 0 {
						return true
					}
				}
				return false
			})
			t.Logf("%s: %s casts spell %d at %q cell %v; F2 order block %x; landed %d ticks after cold LOAD",
				tc.name, castOrderName(f, caster), spell, castOrderName(f, target), cell, order, tick)
		})
	}
}

// TestReleaseOwnerCastSaveRestoresHealAtDanath loads the owner's engine SAV in
// which Reniesta charges Heal at Danath, writes it again with F2 and loads the
// new file cold. The new file's order block names Danath and the unit branch;
// both sessions then run ordinary ticks until the cast ends.
func TestReleaseOwnerCastSaveRestoresHealAtDanath(t *testing.T) {
	path := os.Getenv("AGAINROM_CAST_CRASH_SAV")
	if path == "" {
		t.Skip("no AGAINROM_CAST_CRASH_SAV: the owner's crashing game0001 is an owner input")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "ad3a71eac52191582a0bb1ad27592a3bc7029c422ba9af337558f1476fb3cf02" {
		t.Fatalf("owner save changed: %s", got)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := castOrderSession(t, dir)
	books := f.live.world.Actions().Books
	if len(books) != 1 || books[0].Spell != 6 || books[0].AtCell || books[0].Phase != 1 {
		t.Fatalf("owner save does not hold one charging Heal at a unit: %+v", books)
	}
	c := books[0]
	var danath sim.EntityID
	for i, p := range f.live.mission.party {
		if p.Name == "Danath" {
			danath = f.live.mission.ids[i]
		}
	}
	if danath == 0 || c.Target != danath {
		t.Fatalf("Heal targets %d, Danath is %d", c.Target, danath)
	}
	savedDir, out := castOrderF2Save(t, f, app, "9401 cast flag")
	order := castOrderWire(t, out, c.Caster, danath, false)
	if kit := os.Getenv("AGAINROM_CAST_FLAG_KIT"); kit != "" {
		if err := os.WriteFile(filepath.Join(kit, "game9401.sav"), out, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if label, err := OriginalSaveLabel(out, f.textSelector()); err != nil || label != "9401 cast flag - mission 41" {
		t.Fatalf("SAV label %q %v", label, err)
	}
	g, _ := castOrderSession(t, savedDir)
	if cold, _ := castOrderBook(g.live.world, c.Caster); cold != c {
		t.Fatalf("cold LOAD cast %+v, saved %+v", cold, c)
	}
	d, _ := f.live.entity(danath)
	var outcome []sim.CastEvent
	tick := castOrderLockstep(t, f, g, 64, func(h *FrontEnd, events []sim.CastEvent) bool {
		for _, ev := range events {
			if ev.Caster == c.Caster && h == f {
				outcome = append(outcome, ev)
			}
		}
		now, ok := castOrderBook(h.live.world, c.Caster)
		if ok && (now.Target != danath || now.AtCell || now.Spell != 6) {
			t.Fatalf("cast changed while charging: %+v", now)
		}
		return !ok
	})
	after, _ := f.live.entity(danath)
	t.Logf("Heal ended %d ticks after cold LOAD; Danath health %d/%d before, %d/%d after; caster events %+v; F2 order block %x",
		tick, d.HP, d.MaxHP, after.HP, after.MaxHP, outcome, order)
}
