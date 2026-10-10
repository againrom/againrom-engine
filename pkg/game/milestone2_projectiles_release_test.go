package game

import (
	"againrom/pkg/sim"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
	"encoding/binary"
)

func TestReleaseMilestone2Projectiles1157(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectile1157Read(file.Store)
	if err != nil {
		t.Fatal(err)
	}
	// SAV-PROJCORP-430 anchors this one nonempty document, not a general
	// relation between the allocator and the greatest live ID.
	if !want.present || want.free != 267 || !slices.Equal(want.ids, []uint16{266}) || len(want.items) != 1 || len(want.values) != 18 {
		t.Fatalf("nonempty raw release subject changed: %+v", want)
	}
	f := releaseFront(t)
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	drivers := ms.World.SavedWorldEffectDrivers()
	if drivers == nil || len(drivers.Projectiles) != 1 || drivers.Projectiles[0].ID != 266 || drivers.Projectiles[0].Phases != 4 || !drivers.Projectiles[0].HasTarget {
		t.Fatal("authentic driver/phase binding lost", drivers)
	}
	d := drivers.Projectiles[0]
	joined := false
	for _, dead := range ms.World.OriginalDeadActors() {
		if dead.ID == d.Target {
			joined = dead.Source.ArchiveIndex == 152 && dead.Source.MapUnitID == 92 && dead.Source.Identity == 0x02c24500 && dead.Current.Stage == 3 && dead.Current.HP == -57 && dead.ID != 157 && dead.ID != 152
		}
	}
	if !joined {
		t.Fatal("runtime157/archive152/MapID92/native identity conflated", d)
	}
	expected := projectile1162Next(want, map[uint16]sim.SavedProjectileDriver{266: d}, ms.World)
	if p := expected.items[0].fields; p[0] != 19465 || p[1] != 28086 || p[13] != 3 || p[5] != 1 || p[14] != 2 {
		t.Fatal("independent first-point arithmetic", p)
	}
	sim.Step(ms.World, nil)
	if diff := expected.worldDifferences(ms.World.SavedProjectiles()); len(diff) > 0 {
		t.Fatal("authentic first transition", diff)
	}
	snapshot, err := snapshotSavedDocument(ms)
	if err != nil {
		t.Fatal(err)
	}
	if diff := expected.documentDifferences(snapshot); len(diff) > 0 {
		t.Fatal("changed Document", diff)
	}
	for _, name := range []string{"stale-coordinate", "stale-phase", "lost-ids", "missing-bindings", "wrong-target"} {
		bad, err := cloneSavedDocument(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		switch name {
		case "stale-coordinate", "stale-phase":
			key := "/Prj266/x"
			if name == "stale-phase" {
				key = "/Prj266/actionphase"
			}
			for i := range bad.Document.State.ValueRecords {
				r := &bad.Document.State.ValueRecords[i]
				if r.Path == key {
					r.Value.Int32--
				}
			}
		case "lost-ids":
			for i := range bad.Document.State.ValueRecords {
				if bad.Document.State.ValueRecords[i].Path == "/Projectiles/IDs" {
					bad.Document.State.ValueRecords[i].Value.Bytes = nil
				}
			}
		case "missing-bindings":
			bad.WorldEffects = nil
		case "wrong-target":
			for _, index := range bad.Document.DeadActors {
				r := &bad.Document.Objects[index-1]
				identity, _ := savedStructureValue(r, "Identity")
				if identity == 0x02c24500 {
					savedObjectSetValue(r, "RuntimeID", 152)
					break
				}
			}
		}
		if restoreSavedDocument(&Mission{World: ms.World}, bad) == nil {
			t.Fatal("native loss escaped", name)
		}
	}
	t.Log(fmt.Sprintf("Prj266 first point19465,28086 phase1/action3/segments2; target runtime157 -> late-dead archive152 -> reserved native%d; counter, coordinate, binding and target controls RED", d.Target))
	projectileSAVApp(t, raw)
	projectile1162MovedCorpseApp(t, raw)
	projectile1162RemovedCorpseApp(t, raw)
}

// A controlled move separates the current native corpse from both its source
// Token and the projectile's saved actionx/y. No SourceBinding is fabricated.
func projectile1162MovedCorpseApp(t *testing.T, raw []byte) {
	t.Helper()
	f := releaseFront(t)
	app, path := openOriginalSAVApp(t, f, raw, "moving-corpse.sav")
	d := f.live.world.SavedWorldEffectDrivers().Projectiles[0]
	corpse, exists := entityIn(f.live.world.Entities(), d.Target)
	if !exists || corpse.SourceBinding.Class != 0 || corpse.X != 77 || corpse.Y != 110 {
		t.Fatal("expected materialized corpse with independent native identity", exists, corpse.SourceBinding.Class, corpse.X, corpse.Y)
	}
	checkDraw := func(front *FrontEnd, x, y, phase int, present bool) {
		t.Helper()
		draws := front.live.savedProjectileDraws()
		if !present {
			if len(draws) != 0 {
				t.Fatal("completed source projectile still drawn")
			}
			return
		}
		if len(draws) != 1 || draws[0].Sheet == nil || draws[0].Pos != image.Pt(x, y) || draws[0].Frame != 3*4+phase || !draws[0].Mirror {
			t.Fatalf("installed projectile sheet/current position/frame: count%d expected(%d,%d)/%d mirrored, ANIM-PROJ-026's (5-8)&0xf = 13 halved", len(draws), x, y, 3*4+phase)
		}
	}
	check := func(front *FrontEnd, x, y, phase, action, segments, tx, ty int32) {
		t.Helper()
		items := front.live.world.SavedProjectiles().Items
		if len(items) != 1 {
			t.Fatal("projectile disappeared early")
		}
		p := items[0]
		if p.X != x || p.Y != y || p.Phase != phase || p.ActionPhase != action || p.ActionSegments != segments || p.ActionX != tx || p.ActionY != ty {
			t.Fatal("current target step", p)
		}
		checkDraw(front, int(x), int(y), int(phase), true)
		if _, _, err := front.Snapshot(true); err != nil {
			t.Fatal("changed current Document", err)
		}
	}
	checkDraw(f, 19278, 27985, 1, true)
	placeCorpse := func(front *FrontEnd, x, y, wantX, wantY int32) {
		t.Helper()
		target := front.live.world.SavedWorldEffectDrivers().Projectiles[0].Target
		if err := front.live.world.HeadlessPlace(target, x, y); err != nil {
			t.Fatal(err)
		}
		if c, _ := entityIn(front.live.world.Entities(), target); c.X != wantX || c.Y != wantY {
			t.Fatal("corpse placement cell", c.X, c.Y)
		}
	}
	placeCorpse(f, 82, 111, 81, 111)
	f.live.tick()
	check(f, 19806, 28171, 1, 3, 2, 20864, 28544)
	store, name, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(path)})
	if got := projectileSAVFields(t, written); got[0] != 19806 || got[1] != 28171 || got[10] != 20864 || got[11] != 28544 {
		t.Fatal("written moved-corpse projectile", got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh := loadLocalLegacySave(t, store, name)
	check(fresh, 19806, 28171, 1, 3, 2, 20864, 28544)
	if projectileSeam(f) != projectileSeam(fresh) {
		t.Fatal("moved-corpse SAV LOAD differs", projectileSeam(f), projectileSeam(fresh))
	}
	lost := loadAlteredSAV(t, written, projectileSAVAlter("/Prj266/actionx"))
	if projectileSeam(lost) == projectileSeam(fresh) {
		t.Fatal("loss control: a SAV with Prj266/actionx altered still matches")
	}
	for _, front := range []*FrontEnd{f, fresh} {
		placeCorpse(front, 83, 111, 84, 113)
		front.live.tick()
		check(front, 20719, 28613, 2, 4, 1, 21632, 29056)
		front.live.tick()
		check(front, 21632, 29056, 2, 5, 0, 21632, 29056)
		front.live.tick()
		checkDraw(front, 0, 0, 0, false)
		if len(front.live.world.SavedProjectiles().Items) != 0 {
			t.Fatal("terminal projectile retained")
		}
	}
	t.Log("moved corpse: placed81,111 ->84,113; current aim20864,28544 ->21632,29056; steps19806,28171 ->20719,28613 ->21632,29056; installed phase frames13,14,14 mirrored then removal across ordinary SAVE/fresh LOAD")
}

// Private controlled derivative: preserve the source graph and identities, set
// its corpse at the existing final HP threshold, and give the projectile enough
// local ticks to observe removal. No modified source byte is committed/written
// to the owner's corpus, and the original missing-target policy is not inferred.
func projectile1162RemovedCorpseApp(t *testing.T, raw []byte) {
	t.Helper()
	f, app, path, p := removedCorpseProjectileApp(t, raw)
	store, name, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(path)})
	if got := projectileSAVFields(t, written); got[9] != 157 || got[10] != p.ActionX || got[11] != p.ActionY {
		t.Fatal("written detached projectile", got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh := loadLocalLegacySave(t, store, name)
	if projectileSeam(f) != projectileSeam(fresh) {
		t.Fatal("removed-target SAV LOAD differs", projectileSeam(f), projectileSeam(fresh))
	}
	lost := loadAlteredSAV(t, written, projectileSAVAlter("/Prj266/actiontarget"))
	if projectileSeam(lost) == projectileSeam(fresh) {
		t.Fatal("loss control: a SAV with Prj266/actiontarget altered still matches")
	}
	for i := int32(0); i <= p.ActionSegments; i++ {
		f.live.tick()
		fresh.live.tick()
		if projectileSeam(f) != projectileSeam(fresh) {
			t.Fatal("SAV removed-target continuation", i, projectileSeam(f), projectileSeam(fresh))
		}
		for _, front := range []*FrontEnd{f, fresh} {
			if _, _, err := front.Snapshot(true); err != nil {
				t.Fatal("removed target current Document", err)
			}
			// A cast in this mission builds records of its own (ANIM-147);
			// the source record is read by its id.
			if item, ok := savedProjectileByID(front.live.world, p.ID); ok {
				if item.ActionX != p.ActionX || item.ActionY != p.ActionY || item.ActionTarget != 157 {
					t.Fatal("detached native target re-aimed or lost source key")
				}
			}
		}
	}
	if _, ok := savedProjectileByID(f.live.world, p.ID); ok {
		t.Fatal("removed-target projectile did not complete")
	}
	t.Log("native removal policy: captured latest current corpse point before removal, retained source target157, ordinary SAVE/fresh LOAD then local completion")
}

func projectileSAVApp(t *testing.T, raw []byte) {
	t.Helper()
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectile1157Read(source.Store)
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	ms, report, err := loadOriginalMission(f, raw)
	if err != nil || report.ProjectilesApplied != want.present {
		t.Fatal("direct resume/presence", report.ProjectilesApplied, want.present, err)
	}
	if diff := want.worldDifferences(ms.World.SavedProjectiles()); len(diff) != 0 {
		t.Fatal("initial World", diff)
	}
	if diff := want.documentDifferences(ms.savedDocument); len(diff) != 0 {
		t.Fatal("initial Document before Snapshot", diff)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "projectiles.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	app := f.App("Projectile acceptance")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	if app.Screen() != ui.ScreenMenu {
		t.Fatal("original title LOAD did not start at title")
	}
	groundAppLoad(t, app, list, "projectiles.sav")
	projectileRecordsCheck(t, want, f)
	first := f.live
	groundAppLoad(t, app, list, "projectiles.sav")
	if first == f.live {
		t.Fatal("mission-menu original LOAD did not replace the driver")
	}
	projectileRecordsCheck(t, want, f)
	bindings := map[uint16]sim.SavedProjectileDriver{}
	if drivers := f.live.world.SavedWorldEffectDrivers(); drivers != nil {
		for _, d := range drivers.Projectiles {
			bindings[d.ID] = d
		}
	}
	if len(bindings) > 0 {
		want = projectile1162Next(want, bindings, f.live.world)
		f.live.tick()
		projectileRecordsCheck(t, want, f)
	}
	store, name, written := menuSAVE(t, f, app, OriginalStore{Dir: dir})
	out, err := sav.Open(written)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := projectile1157Read(out.Store); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("written Projectiles differ from the stepped oracle: %v | %+v | %+v", err, got, want)
	}
	if err := os.Remove(filepath.Join(dir, "projectiles.sav")); err != nil {
		t.Fatal(err)
	}
	fresh := loadLocalLegacySave(t, store, name)
	projectileRecordsCheck(t, want, fresh)
	// The source oracle describes the store until a cast builds a record of
	// its own (ANIM-147); from then the two continuations answer each other.
	checked := 0
	for i := range 20 {
		before := f.live.world.Tick()
		freshBefore := fresh.live.world.Tick()
		want = projectile1162Next(want, bindings, f.live.world)
		f.live.tick()
		fresh.live.tick()
		if built := projectileBuiltSince(f.live.world, want.free); built != nil || checked < i {
			if built != nil && built.Picture%2 != 0 {
				t.Fatalf("step %d built %+v, want an even cast picture", i, *built)
			}
		} else {
			projectileRecordsCheck(t, want, f)
			projectileRecordsCheck(t, want, fresh)
			checked++
		}
		if f.live.world.Tick() != before+1 || fresh.live.world.Tick() != freshBefore+1 {
			t.Fatalf("driver did not advance exactly one tick at step %d", i)
		}
		if a, b := projectileSeam(f), projectileSeam(fresh); a != b {
			t.Fatalf("SAV continuation differs at step %d: %s | %s", i, a, b)
		}
	}
	lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
		for i := range doc.State.ValueRecords {
			if doc.State.ValueRecords[i].Path == "/Prj266/x" {
				doc.State.ValueRecords[i].Value.Int32--
				return true
			}
		}
		return false
	})
	if projectileSeam(lost) == projectileSeam(f) {
		t.Fatal("loss control: a SAV with Prj266/x altered still matches")
	}
	t.Logf("Projectile allocator, %d ordered IDs, %d distinct items and %d raw leaves pass both original LOAD doors, known changed step, menu SAV SAVE, source-free fresh LOAD and %d independently checked continuation ticks", len(want.ids), len(want.items), len(want.values), checked)
}

// projectileBuiltSince is a record whose id the counter gave after free, or nil.
func projectileBuiltSince(w *sim.World, free uint16) *sim.SavedProjectile {
	for _, p := range w.SavedProjectiles().Items {
		if p.ID >= free {
			return &p
		}
	}
	return nil
}

func projectileSeam(f *FrontEnd) string {
	out := fmt.Sprintf("%+v", f.live.world.SavedProjectiles())
	for _, d := range f.live.savedProjectileDraws() {
		out += fmt.Sprintf(" draw%v/%d/%t", d.Pos, d.Frame, d.Sheet != nil)
	}
	return out
}

func projectileSAVFields(t *testing.T, written []byte) [16]int32 {
	t.Helper()
	out, err := sav.Open(written)
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectile1157Read(out.Store)
	if err != nil || len(got.items) != 1 {
		t.Fatal("written Projectiles", err, got)
	}
	return got.items[0].fields
}

func projectileSAVAlter(path string) func(*sav.DocumentData) bool {
	return func(doc *sav.DocumentData) bool {
		for i := range doc.State.ValueRecords {
			if doc.State.ValueRecords[i].Path == path {
				doc.State.ValueRecords[i].Value.Int32 += 256
				return true
			}
		}
		return false
	}
}

func removedCorpseProjectileApp(t *testing.T, raw []byte) (*FrontEnd, *ui.App, string, sim.SavedProjectile) {
	t.Helper()
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	actors, _, _, err := readActorRoots(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range actors {
		if a.loc.ArchiveIndex == 152 {
			state := a.loc.StateOff + 1 + int(source.Body[a.loc.StateOff])
			source.Body[state+46] = 4
			binary.LittleEndian.PutUint16(source.Body[state+16:], 64936) // signed -600
			found = true
		}
	}
	if !found {
		t.Fatal("target raw actor missing")
	}
	r, err := reg.Parse(source.Store)
	if err != nil {
		t.Fatal(err)
	}
	source.Store, err = reg.SetInt(source.Store, r, "Prj266", "actionsegments", 40)
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app, path := openOriginalSAVApp(t, f, source.Marshal(), "removing-corpse.sav")
	d := f.live.world.SavedWorldEffectDrivers().Projectiles[0]
	removed := false
	var last sim.Entity
	for tick := 0; tick < 33; tick++ {
		if err := f.live.world.HeadlessPlace(d.Target, 82+int32(tick%2), 111); err != nil {
			t.Fatal(err)
		}
		last, _ = entityIn(f.live.world.Entities(), d.Target)
		f.live.tick()
		if f.live.world.SavedWorldEffectDrivers().Projectiles[0].TargetDetached {
			removed = true
			break
		}
	}
	if !removed {
		t.Fatal("controlled final corpse never removed")
	}
	if _, exists := entityIn(f.live.world.Entities(), d.Target); exists {
		t.Fatal("detached marker left a current entity")
	}
	p := f.live.world.SavedProjectiles().Items[0]
	if p.ActionX != last.X*256+128 || p.ActionY != last.Y*256+128 || p.ActionTarget != 157 || p.ActionSegments <= 0 {
		t.Fatal("removal lost latest position or source key", p)
	}
	return f, app, path, p
}
