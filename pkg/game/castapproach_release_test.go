package game

import (
	"encoding/binary"
	"math/bits"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// castApproachCell picks, for the member of live who knows all 28 spells, the
// first ground spell other than Teleport that admits a cell three off and is
// out of reach at a free cell shown on screen, six to twelve cells off.
func castApproachCell(t *testing.T, app *ui.App, live *mapWorld) (caster sim.EntityID, cell sim.CellPoint, spell uint32) {
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
	for n := int32(6); n <= 12; n++ {
		for _, d := range []sim.CellPoint{{X: n}, {Y: n}, {X: -n}, {Y: -n}} {
			at := sim.CellPoint{X: me.X + d.X, Y: me.Y + d.Y}
			if occupied[at] || !castApproachGround(app, at) {
				continue
			}
			for s := uint32(1); s <= 28; s++ {
				near := sim.CellPoint{X: me.X + d.X/n*3, Y: me.Y + d.Y/n*3}
				if s != 26 && live.world.BookSpellCellRefusal(caster, near.X, near.Y, s) == "" &&
					live.world.BookSpellCellRefusal(caster, at.X, at.Y, s) == "cell out of range" {
					return caster, at, s
				}
			}
		}
	}
	t.Fatalf("no ground spell from %d is out of reach at a visible free cell", caster)
	return
}

func castApproachGround(app *ui.App, at sim.CellPoint) bool {
	for py := 100; py < 560; py += 4 {
		for px := 160; px < 750; px += 4 {
			if x, y, err := app.HeadlessDropCell(px, py); err == nil && int32(x) == at.X && int32(y) == at.Y {
				return true
			}
		}
	}
	return false
}

// TestReleasePlayerCastApproachSaveColdLoad casts by pointer input from the
// game0021 corpus SAV (mission 10) at a ground cell beyond the spell's reach.
// The caster keeps the order and walks; F2 SAVE follows while it walks, the
// order block carries the cell order at progress 0 without the creature
// retention flag, and the file loads cold. Both sessions keep one hash until
// the spell lands on the clicked cell.
func TestReleasePlayerCastApproachSaveColdLoad(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0021.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := castOrderSession(t, dir)
	live := f.live
	caster, cell, spell := castApproachCell(t, app, live)
	start, _ := live.entity(caster)
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
	castOrderGroundClick(t, app, cell)
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindCastAt {
		t.Fatalf("pointer input queued %+v", live.pending)
	}
	for tick := 0; tick < 3; tick++ {
		live.tick()
	}
	c, ok := castOrderBook(live.world, caster)
	if !ok || c.Phase != sim.BookPhaseApproach || !c.AtCell || uint32(c.Spell) != spell || c.X != cell.X || c.Y != cell.Y {
		t.Fatalf("the out-of-reach cast is not walking to its cell: %+v %v", c, ok)
	}
	savedDir, saved := castOrderF2Save(t, f, app, "cast approach")
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal("action supplement", err)
	}
	record := castOrderRecord(t, doc, actions, caster)
	order := savedRecordRawForTest(t, record, "U158")
	state := binary.LittleEndian.Uint32(savedRecordRawForTest(t, record, "U54"))
	if order[8] != 9 || order[9] != 0 || order[0x60] != 0 {
		t.Fatalf("the walking cast's order block reads kind %d progress %d flag %d, want 9, 0 and 0", order[8], order[9], order[0x60])
	}
	g, _ := castOrderSession(t, savedDir)
	if cold, _ := castOrderBook(g.live.world, caster); cold != c {
		t.Fatalf("cold LOAD cast %+v, saved %+v", cold, c)
	}
	tick := castOrderLockstep(t, f, g, 600, func(h *FrontEnd, events []sim.CastEvent) bool {
		for _, ev := range events {
			if ev.Caster == caster && uint32(ev.Spell) == spell && ev.AtCell && ev.ToX == cell.X && ev.ToY == cell.Y {
				return true
			}
		}
		return false
	})
	end, _ := f.live.entity(caster)
	t.Logf("%s walks from (%d,%d) to (%d,%d) and casts spell %d at %v %d ticks after cold LOAD; saved action %#x, order block %x",
		castOrderName(f, caster), start.X, start.Y, end.X, end.Y, spell, cell, tick, state, order[:0x60])
}
