package game

import (
	"againrom/pkg/sim"
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func projectileRecordsCheck(t *testing.T, want projectile1157Source, f *FrontEnd) {
	t.Helper()
	if f.live == nil {
		t.Fatal("missing mission driver")
	}
	if diff := want.worldDifferences(f.live.world.SavedProjectiles()); len(diff) != 0 {
		t.Fatalf("raw source vs World: %v", diff)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if diff := want.documentDifferences(snapshot.SavedDocument); len(diff) != 0 {
		t.Fatalf("raw source vs retained Document: %v", diff)
	}
}

func projectile1157App(t *testing.T, raw []byte, front func() *FrontEnd) {
	t.Helper()
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectile1157Read(source.Store)
	if err != nil {
		t.Fatal(err)
	}
	f := front()
	f.SetDeterministicFrames(true)
	// Test the initial retained Document before any Snapshot projection.
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
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("Projectile acceptance")
	app.Layout(1024, 768)
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: dir}, nil)
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
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatal("menu SAVE using the explicit AGS codec", entries, err)
	}
	if err := os.Remove(filepath.Join(dir, "projectiles.sav")); err != nil {
		t.Fatal(err)
	}
	fresh := front()
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("Fresh Projectile LOAD")
	app2.Layout(1024, 768)
	s, l, ld := nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(s, l, ld)
	groundAppLoad(t, app2, l, entries[0].Name)
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("fresh native LOAD changed World hash")
	}
	t.Log("fresh native LOAD has equal World hash; checking retained Projectile records separately")
	projectileRecordsCheck(t, want, fresh)
	for i := range 20 {
		before := f.live.world.Tick()
		freshBefore := fresh.live.world.Tick()
		want = projectile1162Next(want, bindings, f.live.world)
		f.live.tick()
		fresh.live.tick()
		projectileRecordsCheck(t, want, f)
		projectileRecordsCheck(t, want, fresh)
		if f.live.world.Tick() != before+1 || fresh.live.world.Tick() != freshBefore+1 {
			t.Fatalf("driver did not advance exactly one tick at step %d", i)
		}
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("native continuation differs at step %d", i)
		}
	}
	projectileRecordsCheck(t, want, f)
	projectileRecordsCheck(t, want, fresh)
	t.Logf("Projectile allocator, %d ordered IDs, %d distinct items and %d raw leaves pass both original LOAD doors, known changed step, menu SAVE, source-free fresh LOAD and20 independently checked continuation ticks", len(want.ids), len(want.items), len(want.values))
}

// Independent current-state oracle: saved source leaves plus frozen import
// parameters; it never reads the production projectile after stepping it.
func projectile1162Next(src projectile1157Source, bindings map[uint16]sim.SavedProjectileDriver, w *sim.World) projectile1157Source {
	out := src
	out.dirs = maps.Clone(src.dirs)
	out.values = maps.Clone(src.values)
	out.ids = slices.Clone(src.ids)
	out.items = nil
	for _, item := range src.items {
		p := item.fields
		d, armed := bindings[item.id]
		root := fmt.Sprintf("/Prj%d", item.id)
		if armed && p[14] == 0 {
			out.ids = slices.DeleteFunc(out.ids, func(id uint16) bool { return id == item.id })
			delete(out.dirs, root)
			for path := range out.values {
				if strings.HasPrefix(path, root+"/") {
					delete(out.values, path)
				}
			}
			continue
		}
		if armed {
			if d.HasTarget {
				for _, e := range w.Entities() {
					if e.ID != d.Target {
						continue
					}
					x, y := e.X*256+128, e.Y*256+128
					if fx, fy, ok := w.ActorFinePosition(e.ID); ok {
						x, y = e.X*256+int32(fx), e.Y*256+int32(fy)
					} else if e.Stride.Present && e.Transit > 0 {
						paid := int32(e.TransitTotal - e.Transit)
						x, y = e.Stride.FromX*256+128+paid*int32(e.Stride.StepX), e.Stride.FromY*256+128+paid*int32(e.Stride.StepY)
					}
					p[10], p[11] = x, y
					break
				}
			}
			p[0] += (p[10] - p[0]) / p[14]
			p[1] += (p[11] - p[1]) / p[14]
			p[2] += (p[12] - p[2]) / p[14]
			p[13]++
			p[5] = 0
			if d.Phases > 0 {
				p[5] = (p[13] / 2) % int32(d.Phases)
			}
			p[6] = p[7]
			p[14]--
		}
		item.fields = p
		out.items = append(out.items, item)
		for i, name := range projectile1157Names {
			key := root + "/" + name
			r := out.values[key]
			r.word = uint32(p[i])
			out.values[key] = r
		}
	}
	if !slices.Equal(out.ids, src.ids) {
		v := out.values["/Projectiles/IDs"]
		v.kind, v.word = 6, 0
		v.data = make([]byte, 4*len(out.ids))
		for i, id := range out.ids {
			binary.LittleEndian.PutUint32(v.data[i*4:], uint32(id))
		}
		out.values["/Projectiles/IDs"] = v
	}
	return out
}
