package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

func strideActor1115(t *testing.T, f *FrontEnd) sim.Entity {
	t.Helper()
	return newGroupActors1115(t, f.live.world)[newGroupA]
}

func strideEast1115() sim.NativeStride {
	// Literal accepted movement, independent of the capture implementation:
	// flat ground, source speed 100, rate clamp 63, east by one cell.
	return sim.NativeStride{Present: true, FromX: 15, FromY: 16, ToX: 16, ToY: 16,
		Rate: 63, StepX: 63, StepY: 0, Direction: 2}
}

func strideStartEast(t *testing.T, f *FrontEnd) {
	t.Helper()
	e := strideActor1115(t, f)
	if e.X != 15 || e.Y != 16 || e.Speed != 100 || e.Facing != 64 || e.Transit != 0 || e.Stride.Present {
		t.Fatalf("literal source movement differs: %+v", e)
	}
	// This pre-existing public rate query is a fixture check, not the new
	// stride capture being used as its own expected answer.
	rate, total, adjacent, ok := f.live.world.StepRate(e.ID, 16, 16)
	if !ok || !adjacent || rate != 63 || total != 5 {
		t.Fatalf("literal east rate = %d/%d adjacent=%t ok=%t, want 63/5", rate, total, adjacent, ok)
	}
	f.live.enqueue(uint32(e.ID), 20, 16)
	f.live.tick()
	e = strideActor1115(t, f)
	if e.X != 16 || e.Y != 16 || e.Transit != 4 || e.TransitTotal != 5 || e.Facing != 64 || e.Stride != strideEast1115() {
		t.Fatalf("accepted east crossing was not captured: %+v", e)
	}
}

// Both original LOAD doors reach ordinary map input, native menu SAVE and a
// fresh App's native LOAD. This tests native stride provenance and continuation;
// it does not export Position, mover or cell state into original-format SAV.
func TestSavedStride1115BothDoorsMenuSaveAndContinuation(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("from-map-%t/replace-mid-crossing-%t", fromMap, replace), func(t *testing.T) {
				f := newGroupFront(t, -1)
				app := f.App("native stride source")
				if fromMap {
					if err := app.OpenMission(f.MissionOpener(10)); err != nil {
						t.Fatal(err)
					}
				}
				originals, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
				path := filepath.Join(originals, "game1115.sav")
				raw := newGroupLiteral(t, f, false)
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
				app.SetSaveSeams(save, list, load)
				groundAppLoad(t, app, list, "game1115.sav")
				clear(raw)
				// Remove only this test's synthetic input. Native LOAD must not
				// reconstruct the accepted movement from an original source.
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				strideStartEast(t, f)
				if replace {
					e := strideActor1115(t, f)
					f.live.enqueue(uint32(e.ID), 20, 12) // northeast, unlike the active east crossing
					f.live.tick()
					e = strideActor1115(t, f)
					if e.TargetX != 20 || e.TargetY != 12 || e.Transit != 3 || e.TransitTotal != 5 || e.Stride != strideEast1115() {
						t.Fatalf("replacement command rewrote the accepted crossing: %+v", e)
					}
				}
				before := groupDocumentSnapshot(t, f)
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessGameMenuAction("save"); err != nil {
					t.Fatal(err)
				}
				entries, err := store.List()
				if err != nil || len(entries) != 1 || !strings.HasSuffix(entries[0].Name, ".sav") {
					t.Fatal("ordinary native menu SAVE", entries, err)
				}
				fresh := newGroupFront(t, -1)
				freshApp := fresh.App("native stride fresh load")
				fs, fl, fn := fresh.SaveSeams(store, OriginalStore{}, nil)
				freshApp.SetSaveSeams(fs, fl, fn)
				groundAppLoad(t, freshApp, fl, entries[0].Name)
				after := groupDocumentSnapshot(t, fresh)
				if !bytes.Equal(before.World, after.World) || f.live.world.Hash() != fresh.live.world.Hash() || strideActor1115(t, fresh).Stride != strideEast1115() {
					t.Fatal("current SAV/fresh App LOAD changed World or stride")
				}
				owed := strideActor1115(t, f).Transit
				independentFacing, northeast := false, false
				for tick := 1; tick <= 20; tick++ {
					f.live.tick()
					fresh.live.tick()
					e, loaded := strideActor1115(t, f), strideActor1115(t, fresh)
					if f.live.world.Hash() != fresh.live.world.Hash() || !reflect.DeepEqual(e, loaded) {
						t.Fatalf("native continuation diverged at tick %d", tick)
					}
					if tick <= int(owed) && (e.X != 16 || e.Y != 16 || e.Transit != owed-uint16(tick) || e.Stride != strideEast1115()) {
						t.Fatalf("accepted crossing changed while paying tick %d: %+v", tick, e)
					}
					if e.Stride == strideEast1115() && e.Facing != 64 {
						if !replace || e.Facing != 32 || e.Transit != 0 {
							t.Fatalf("unexpected independent facing at tick %d: %+v", tick, e)
						}
						independentFacing = true
					}
					if replace && e.Stride.Present && e.Stride != strideEast1115() {
						want := sim.NativeStride{Present: true, FromX: e.X - 1, FromY: e.Y + 1,
							ToX: e.X, ToY: e.Y, Rate: 63, StepX: 44, StepY: -44, Direction: 1}
						if e.Stride != want || e.TransitTotal != 6 {
							t.Fatalf("replacement's accepted diagonal at tick %d = %+v, want %+v", tick, e.Stride, want)
						}
						northeast = true
					}
				}
				if replace && (!independentFacing || !northeast) {
					t.Fatalf("continuation missed facing-independent old stride or new diagonal: facing=%t diagonal=%t", independentFacing, northeast)
				}
			})
		}
	}
}
