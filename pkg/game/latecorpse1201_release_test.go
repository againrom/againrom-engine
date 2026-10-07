package game

import (
	"fmt"
	"image"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseLateCorpseConstructor1201(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2027-09-07/game0032.sav", "7375f0c08c8361b2fa32d20564802acba688d5e5a445ab5ca635c609632499ed")
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	app := f.App("late corpse constructor")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)

	var virtual sim.OriginalDeadRecord
	found := 0
	for _, r := range f.live.world.OriginalDeadActors() {
		if r.Source.MapUnitID == 0 {
			virtual = r
			found++
		}
	}
	if found != 1 {
		t.Fatalf("game0032.sav MapUnitID-0 dead records = %d, want exactly 1 (this fixture's own hired-mercenary death)", found)
	}
	if virtual.Current.Stage < 2 || virtual.Current.Stage > 4 {
		t.Fatalf("virtual record Current.Stage = %d, want the pre-terminal decay range [2,4] this story draws (SAV-DEADLOAD-128)", virtual.Current.Stage)
	}

	// SAV-DEADLOAD-125: the dead-list clock does not require a living entity.
	for _, e := range f.live.world.Entities() {
		if e.ID == virtual.ID {
			t.Fatalf("virtual dead actor %d must never be a live entity (SAV-DEADLOAD-125/132)", virtual.ID)
		}
	}

	wantCell := image.Point{X: int(virtual.Current.Cell & 255), Y: int(virtual.Current.Cell >> 8)}
	drawn := 0
	for _, d := range f.live.entityDraws() {
		if d.ID != uint32(virtual.ID) {
			continue
		}
		drawn++
		if d.Cell != wantCell {
			t.Errorf("draw Cell = %v, want %v (the record's own Current.Cell)", d.Cell, wantCell)
		}
		if d.Life != ui.LifeDead {
			t.Errorf("draw Life = %d, want ui.LifeDead", d.Life)
		}
		if d.HP != int(virtual.Current.HP) {
			t.Errorf("draw HP = %d, want %d", d.HP, virtual.Current.HP)
		}
		if !d.Untargetable {
			t.Error("draw Untargetable = false, want true: no live entity backs this id, ordinary combat must not select it")
		}
		if d.Art == nil || d.Frame == nil {
			t.Error("draw has nil Art/Frame: this is the pre-story failure mode, the unresolved square, for a record this fixture's own T0E resolves")
		}
	}
	if drawn != 1 {
		t.Fatalf("entityDraws() produced %d draws for the MapUnitID-0 record, want exactly 1 -- before this story it was 0 (the defect this story fixes)", drawn)
	}
	t.Logf("MapUnitID-0 dead actor %d (stage %d, HP %d, cell %v) now draws once, with resolved Art/Frame, matching every ordinary corpse at the same depth", virtual.ID, virtual.Current.Stage, virtual.Current.HP, wantCell)
}

func TestReleaseLateCorpseNativeReloadKeepsBody(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	path, _ := groundCorpusFile(t, "2027-09-07/game0032.sav", "7375f0c08c8361b2fa32d20564802acba688d5e5a445ab5ca635c609632499ed")
	app := f.App("SAV reload keeps the virtual body")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))

	var virtual uint32
	for _, r := range f.live.world.OriginalDeadActors() {
		if r.Source.MapUnitID == 0 {
			virtual = r.Current.RuntimeID
		}
	}
	if virtual == 0 {
		t.Fatal("game0032.sav's own MapUnitID-0 dead record missing")
	}
	if got := lateCorpseSeam(f, virtual); !strings.Contains(got, "draws1") {
		t.Fatal("first load must draw the virtual body once with resolved art", got)
	}

	var victim sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID == 0 && e.SourceBinding.Class != 0 && e.Alive() && !slices.Contains(f.live.mission.guarded, e.ID) {
			victim = e
			break
		}
	}
	if victim.SourceBinding.Identity == 0 {
		t.Fatal("game0032.sav's own unguarded source-bound, unbound-map-unit actor missing")
	}
	headlessDamage(t, f.live.world, victim.ID, victim.HP+20)
	for i := 0; i < 300; i++ {
		f.live.tick()
		victim, _ = f.live.entity(victim.ID)
		if victim.Decay >= 2 {
			break
		}
	}
	if victim.Decay < 2 {
		t.Fatal("death did not reach the guarded late-corpse decay stage")
	}
	runtimes := []uint32{virtual, victim.SourceBinding.RuntimeID}
	want := lateCorpseSeam(f, runtimes...)
	store, name, written := menuSAVE(t, f, app, OriginalStore{})
	out, err := sav.Open(written)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := out.DeadActors()
	if err != nil {
		t.Fatal(err)
	}
	written1 := 0
	for _, d := range dead {
		for _, r := range f.live.world.OriginalDeadActors() {
			if d.RuntimeID == virtual && r.Current.RuntimeID == virtual {
				written1++
				if d.MapUnitID != 0 || d.Stage != r.Current.Stage || d.HP != r.Current.HP || d.Cell != r.Current.Cell {
					t.Fatalf("written body differs from live: %+v %+v", d, r.Current)
				}
			}
		}
	}
	if written1 != 1 {
		t.Fatal("written MapUnitID-0 bodies", written1)
	}
	second := 0
	for _, d := range dead {
		if d.RuntimeID == runtimes[1] {
			second++
			if d.MapUnitID != 0 || int32(d.Stage) != int32(victim.Decay) || int32(d.HP) != int32(victim.HP) || d.Cell != uint16(victim.X)|uint16(victim.Y)<<8 {
				t.Fatalf("written second corpse differs from live: %+v", d)
			}
		}
	}
	if second != 1 {
		t.Fatal("written second MapUnitID-0 corpses", second)
	}
	fresh := loadLocalLegacySave(t, store, name)
	if got := lateCorpseSeam(fresh, virtual); !strings.HasPrefix(got, fmt.Sprintf("dead%d mu0 ", virtual)) || !strings.HasSuffix(got, " draws1;") {
		t.Fatal("fresh LOAD must draw the virtual body exactly once beside the second corpse", got)
	}
	for tick := 0; ; tick++ {
		if got, live := lateCorpseSeam(fresh, runtimes...), lateCorpseSeam(f, runtimes...); got != live {
			t.Fatalf("tick%d restored corpses differ: %s | %s", tick, live, got)
		}
		if tick == 16 {
			break
		}
		f.live.tick()
		fresh.live.tick()
	}
	lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
		for i := range doc.Objects {
			if id, _ := savedStructureValue(&doc.Objects[i], "RuntimeID"); id == virtual && doc.Objects[i].Class == "Human" {
				return savedStructureSetValue(&doc.Objects[i], "Stage", 3) == nil
			}
		}
		return false
	})
	if lateCorpseSeam(lost, runtimes...) == want {
		t.Fatal("loss control: a SAV with the body's decay stage altered still matches")
	}
	t.Logf("menu SAVE %s, fresh LOAD and 16 ticks keep %s", name, want)
}

func lateCorpseSeam(f *FrontEnd, runtimes ...uint32) string {
	draws := func(id uint32) int {
		n := 0
		for _, d := range f.live.entityDraws() {
			if d.ID == id && d.Art != nil && d.Frame != nil {
				n++
			}
		}
		return n
	}
	out := ""
	for _, runtime := range runtimes {
		for _, r := range f.live.world.OriginalDeadActors() {
			if r.Current.RuntimeID == runtime {
				out += fmt.Sprintf("dead%d mu%d %+v draws%d;", runtime, r.Source.MapUnitID, r.Current, draws(uint32(r.ID)))
			}
		}
		for _, e := range f.live.world.Entities() {
			if e.SourceBinding.RuntimeID == runtime {
				out += fmt.Sprintf("actor%d decay%d hp%d at%d,%d alive%t draws%d;", runtime, e.Decay, e.HP, e.X, e.Y, e.Alive(), draws(uint32(e.ID)))
			}
		}
	}
	return out
}
