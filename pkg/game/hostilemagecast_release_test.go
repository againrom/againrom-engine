package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

// The owner's two mission SAVs with a computer mage group guarding a gate,
// read from AGAINROM_MAGE_CHASE_DIR (mage-anim.sav, magesmoving.sav) and never
// committed. The party flees past the mages; a mage whose cast starts while it
// is still crossing into a cell used to stand frozen mid-crossing, so the map
// drew its walk while its cast run played. The executor finishes a step in
// progress at the cell centre before it reads the cast order (AI-350, AI-352,
// MAGIC-239), and a cast plays the Attack run (MAGIC-CASTANIM-029).
func TestReleaseHostileMageCastRunNeverDrawsAWalk(t *testing.T) {
	dir := os.Getenv("AGAINROM_MAGE_CHASE_DIR")
	if dir == "" {
		t.Skip("set AGAINROM_MAGE_CHASE_DIR to the directory holding mage-anim.sav and magesmoving.sav")
	}
	for _, run := range []struct {
		save  string
		party []sim.EntityID
		moves [][2]int
		ticks int
	}{
		{"magesmoving.sav", []sim.EntityID{45, 46, 47}, [][2]int{{19, 62}}, 1600},
		{"mage-anim.sav", []sim.EntityID{45, 46, 47}, [][2]int{{20, 42}, {19, 62}}, 2000},
		{"mage-anim.sav", []sim.EntityID{45}, [][2]int{{20, 43}, {19, 62}}, 1600},
	} {
		raw, err := os.ReadFile(filepath.Join(dir, run.save))
		if err != nil {
			t.Fatal(err)
		}
		f := releaseFront(t)
		openOriginalSAVApp(t, f, raw, run.save)
		mw := f.live
		w := mw.world
		hero, ok := w.Entity(run.party[0])
		if !ok {
			t.Fatalf("%s: hero %d absent", run.save, run.party[0])
		}
		var casters []sim.EntityID
		for _, e := range w.Entities() {
			if e.Alive() && !e.OffMap && e.Owner != hero.Owner && e.KnownSpells != 0 {
				casters = append(casters, e.ID)
			}
		}
		if len(casters) == 0 {
			t.Fatalf("%s: no hostile book caster", run.save)
		}
		releases := make(map[sim.EntityID]int)
		closest := make(map[sim.EntityID]int32)
		for _, id := range casters {
			e, _ := w.Entity(id)
			closest[id] = max(e.X-hero.X, hero.X-e.X, e.Y-hero.Y, hero.Y-e.Y)
		}
		for _, id := range run.party {
			f.LiveOrder(uint32(id), run.moves[0][0], run.moves[0][1])
		}
		next, castTicks, hostileCasts := 1, 0, 0
		for n := 0; n < run.ticks; n++ {
			for _, ev := range f.LiveAdvanceCasts(1) {
				if c, ok := w.Entity(ev.Caster); ok && c.Owner != hero.Owner {
					hostileCasts++
					releases[ev.Caster]++
				}
			}
			mw.entityDraws()
			if h, _ := w.Entity(run.party[0]); !h.HasTarget && next < len(run.moves) {
				for _, id := range run.party {
					f.LiveOrder(uint32(id), run.moves[next][0], run.moves[next][1])
				}
				next++
			}
			for _, id := range casters {
				e, _ := w.Entity(id)
				h, _ := w.Entity(run.party[0])
				closest[id] = min(closest[id], max(e.X-h.X, h.X-e.X, e.Y-h.Y, h.Y-e.Y))
				if !mw.casting(id) {
					continue
				}
				castTicks++
				if e, _ := w.Entity(id); mw.drawnMoving[id] {
					t.Fatalf("%s tick %d: mage %d draws a walk while its cast run plays (at %d,%d, %d of %d crossing ticks owed)",
						run.save, w.Tick(), id, e.X, e.Y, e.Transit, e.TransitTotal)
				}
			}
		}
		if castTicks == 0 || hostileCasts == 0 {
			t.Fatalf("%s: the mages cast nothing (%d cast-run ticks, %d releases)", run.save, castTicks, hostileCasts)
		}
		t.Logf("%s party %v: %d cast-run ticks, %d hostile releases, none drawn walking", run.save, run.party, castTicks, hostileCasts)
		for _, id := range casters {
			t.Logf("%s party %v mage %d: %d releases, closest hero distance %d cells", run.save, run.party, id, releases[id], closest[id])
		}
	}
}
