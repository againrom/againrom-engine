package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func counterMission(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := missionFrontEnd(t)
	f.Town = NewTown(Campaign{})
	app := f.App("projectile counter")
	t.Cleanup(app.StopAudio)
	if err := app.OpenMission(f.MissionOpenerWith(10, nil)); err != nil {
		t.Fatal(err)
	}
	return f, app
}

func counterRawFreeIndex(t *testing.T, raw []byte) (uint32, bool) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	b := file.Store
	if len(b) < 24 || binary.LittleEndian.Uint32(b) != 0x31415926 {
		t.Fatal("counter witness has no framed registry")
	}
	nodes := uint64(binary.LittleEndian.Uint32(b[16:]))
	if 24+nodes*32 > uint64(len(b)) {
		t.Fatal("counter witness has a truncated node table")
	}
	find := func(first, count uint32, name string) []byte {
		if uint64(first)+uint64(count) > nodes {
			t.Fatal("counter witness child range exceeds node table")
		}
		for i := uint64(first); i < uint64(first)+uint64(count); i++ {
			n := b[24+32*i : 24+32*(i+1)]
			if string(bytes.TrimRight(n[16:32], "\x00")) == name {
				return n
			}
		}
		return nil
	}
	section := find(binary.LittleEndian.Uint32(b[4:]), binary.LittleEndian.Uint32(b[8:]), "Projectiles")
	if section == nil || binary.LittleEndian.Uint32(section[12:])&1 == 0 {
		return 0, false
	}
	leaf := find(binary.LittleEndian.Uint32(section[4:]), binary.LittleEndian.Uint32(section[8:]), "FreeIndex")
	if leaf == nil || binary.LittleEndian.Uint32(leaf[12:])&15 != 2 {
		return 0, false
	}
	return binary.LittleEndian.Uint32(leaf[4:]), true
}

func TestProcessProjectileCounterMissionLoadUsesItsCompleteStore(t *testing.T) {
	for _, present := range []bool{true, false} {
		t.Run(fmt.Sprint(present), func(t *testing.T) {
			f := restoredFacingFront(t)
			source := restoredFacingSource(t, f)
			app, _ := openOriginalSAVApp(t, f, source, "counter-running.sav")
			t.Cleanup(app.StopAudio)
			counterSeed(t, f.live.world, 500)
			old, hash := f.live, f.live.world.Hash()
			source = alterSAV(t, source, func(doc *sav.DocumentData) bool {
				for i, r := range doc.State.ValueRecords {
					if r.Path != "/Projectiles/FreeIndex" {
						continue
					}
					if present {
						doc.State.ValueRecords[i].Value.Int32 = 0x10049
					} else {
						doc.State.ValueRecords = slices.Delete(doc.State.ValueRecords, i, i+1)
					}
					return true
				}
				return false
			})
			wire, found := counterRawFreeIndex(t, source)
			if found != present || present && wire != 0x10049 {
				t.Fatal("controlled raw FreeIndex input", wire, found)
			}
			open, town, err := f.RestoreOriginal(source)
			if err != nil || town || f.live != old || f.live.world.Hash() != hash || f.projectiles.world != old.world || f.projectiles.value() != 500 {
				t.Fatal("prepared mission LOAD changed running state", town, err)
			}
			if err := app.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			want := uint16(0)
			if present {
				want = 73
			}
			input := kitProjectileStore(t, source)
			current := f.live.world.SavedProjectiles()
			if f.projectiles.world != f.live.world || f.projectiles.value() != want {
				t.Fatal("mission LOAD did not replace the process counter source")
			}
			if current.FreeIndex != want || !slices.Equal(current.IDs, input.IDs) || len(current.Items) != len(input.Items) {
				t.Fatalf("mission LOAD replaced counter or lost records: %+v, input %+v", current, input)
			}
			for i, p := range current.Items {
				if p != savProjectileToSaved(input.Items[i]) {
					t.Fatal("mission LOAD changed a saved record", p, input.Items[i])
				}
			}
			written := currentRuntimeSave(t, f)
			if value, ok := counterRawFreeIndex(t, written); !ok || value != uint32(want) {
				t.Fatalf("ordinary SAV raw FreeIndex=%d present=%v, want %d", value, ok, want)
			}
			cold := restoredFacingFront(t)
			loaded, _ := openOriginalSAVApp(t, cold, written, "counter-current.sav")
			t.Cleanup(loaded.StopAudio)
			if !reflect.DeepEqual(current, cold.live.world.SavedProjectiles()) {
				t.Fatal("ordinary SAV/cold LOAD changed complete current store")
			}
			counterRecord(t, cold.live.world, want)
			got := cold.live.world.SavedProjectiles()
			if len(got.Items) != len(current.Items)+1 || len(got.IDs) != len(current.IDs)+1 {
				t.Fatal("next record lost the loaded population")
			}
			for _, p := range current.Items {
				if !slices.Contains(got.Items, p) || !slices.Contains(got.IDs, p.ID) {
					t.Fatal("next record destroyed a loaded record or ID", p.ID)
				}
			}
		})
	}
}

func counterSeed(t *testing.T, w *sim.World, value uint16) {
	t.Helper()
	store := w.SavedProjectiles()
	store.FreeIndex = value
	w.SetSavedProjectiles(store)
}

func counterRecord(t *testing.T, w *sim.World, want uint16) {
	t.Helper()
	id, ok := w.ReleaseAreaBurst(sim.AreaBurst{CellX: 8, CellY: 8, Picture: 17, Segments: 16, Phases: 8})
	if !ok || id != want {
		t.Fatalf("record id=%d produced=%v, want %d; store=%+v", id, ok, want, w.SavedProjectiles())
	}
}

func TestProcessProjectileCounterContinuesAcrossFreshMissionEntries(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "new game reset"}[reset], func(t *testing.T) {
			f, app := counterMission(t)
			counterSeed(t, f.live.world, 65534)
			counterRecord(t, f.live.world, 65534)
			if reset {
				f.resetSessionForNewGame()
			}
			if err := app.OpenMission(f.MissionOpenerWith(10, nil)); err != nil {
				t.Fatal(err)
			}
			if store := f.live.world.SavedProjectiles(); store.FreeIndex != 65535 || len(store.Items) != 0 || len(store.IDs) != 0 {
				t.Fatalf("fresh mission discarded the process counter or retained old records: %+v", store)
			}
			counterRecord(t, f.live.world, 65535)
			counterRecord(t, f.live.world, 0)
			if got := f.live.world.SavedProjectiles().FreeIndex; got != 1 {
				t.Fatalf("wrapped counter=%d, want 1", got)
			}
			independent, _ := counterMission(t)
			if got := independent.live.world.SavedProjectiles().FreeIndex; got != 0 {
				t.Fatalf("another FrontEnd inherited counter %d", got)
			}
		})
	}
}

func TestProcessProjectileCounterCommitsLatestValueAfterDeferredPreparation(t *testing.T) {
	f, app := counterMission(t)
	counterSeed(t, f.live.world, 37)
	old := f.live
	store, hash := old.world.SavedProjectiles(), old.world.Hash()
	before, err := old.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.prepareNewGameWith(20, 1, nil); err == nil {
		t.Fatal("missing mission opened")
	}
	if _, _, err := f.RestoreOriginal([]byte("bad SAV")); err == nil {
		t.Fatal("malformed LOAD succeeded")
	}
	open, err := f.prepareNewGameWith(10, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := old.world.MarshalBinary()
	if err != nil || f.live != old || f.projectiles.world != old.world || f.projectiles.value() != 37 || old.world.Hash() != hash || !reflect.DeepEqual(old.world.SavedProjectiles(), store) || !bytes.Equal(before, after) {
		t.Fatal("failed or deferred candidate changed the running state", err)
	}
	counterRecord(t, old.world, 37)
	counterRecord(t, old.world, 38)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.live == old || f.live.world.SavedProjectiles().FreeIndex != 39 {
		t.Fatalf("activation did not use the latest running counter: %+v", f.live.world.SavedProjectiles())
	}
	counterRecord(t, f.live.world, 39)
}

func TestProcessProjectileCounterDetachesWhenItsDriverEnds(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(fmt.Sprint(reset), func(t *testing.T) {
			f, app := counterMission(t)
			old := f.live.world
			counterSeed(t, old, 123)
			if reset {
				f.resetSessionForNewGame()
			} else {
				f.endLive()
			}
			if f.projectiles.world != nil || f.projectiles.retained != 123 {
				t.Fatalf("ended driver retained its World or lost its counter: %+v", f.projectiles)
			}
			counterSeed(t, old, 999)
			if err := app.OpenMission(f.MissionOpenerWith(10, nil)); err != nil {
				t.Fatal(err)
			}
			if f.live.world.SavedProjectiles().FreeIndex != 123 || f.projectiles.world != f.live.world {
				t.Fatal("new driver did not start from the detached scalar")
			}
		})
	}
}

func TestProcessProjectileCounterSeedRetainsCandidateRecords(t *testing.T) {
	f, _ := counterMission(t)
	counterRecord(t, f.live.world, 0)
	want := f.live.world.SavedProjectiles()
	want.FreeIndex = 92
	seedProjectileCounter(f.live.world, 92)
	if !reflect.DeepEqual(f.live.world.SavedProjectiles(), want) {
		t.Fatal("counter input replaced candidate records or IDs")
	}
}

func TestProcessProjectileCounterTownPresenceAndAcceptanceBoundary(t *testing.T) {
	city, newFront := spellbookCitySource(t)
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprint(explicit), func(t *testing.T) {
			file, err := sav.Open(city)
			if err != nil {
				t.Fatal(err)
			}
			registry, ok := file.StateStore()
			if !ok {
				t.Fatal("city input lacks its state store")
			}
			var copyNode func(*reg.Node) synth.RegNode
			copyNode = func(n *reg.Node) synth.RegNode {
				out := synth.RegNode{Name: n.Name, Kind: n.Kind, Str: n.Str, Int: n.Int, Float: n.Float, Ints: n.Ints}
				for _, child := range n.Children {
					out.Children = append(out.Children, copyNode(child))
				}
				return out
			}
			var roots []synth.RegNode
			for _, n := range registry.Root.Children {
				roots = append(roots, copyNode(n))
			}
			leaves := []synth.RegNode{{Name: "IDs", Kind: 6}}
			if explicit {
				leaves = append(leaves, synth.RegNode{Name: "FreeIndex", Kind: 2, Int: 91})
			}
			roots = append(roots, synth.RegNode{Name: "Projectiles", Kind: 1, Children: leaves})
			file.Store = synth.Reg(17, roots)
			raw := file.Marshal()
			parsed, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			if _, anyLeaf, err := parsed.Projectiles(); err != nil || !anyLeaf {
				t.Fatal("projectile reader did not recognize the controlled input", anyLeaf, err)
			}
			store, _ := parsed.StateStore()
			if value, present := store.GetInt("Projectiles", "FreeIndex"); present != explicit || explicit && value != 91 {
				t.Fatal("FreeIndex presence was conflated with IDs presence", value, present)
			}
			f, _ := counterMission(t)
			prototype := newFront()
			f.Table, f.Campaign = prototype.Table, prototype.Campaign
			counterSeed(t, f.live.world, 123)
			old, hash := f.live, f.live.world.Hash()
			if _, _, err := f.RestoreOriginal(raw); err == nil {
				t.Fatal("town with an extra Projectiles section passed the current strict city boundary")
			}
			if f.live != old || f.live.world.Hash() != hash || f.projectiles.value() != 123 || f.projectiles.world != old.world {
				t.Fatal("refused town input changed the running counter or World")
			}
			if _, town, err := f.RestoreOriginal(city); err != nil || !town {
				t.Fatal("ordinary town input refused", town, err)
			}
			if f.live != nil || f.projectiles.world != nil || f.projectiles.value() != 123 {
				t.Fatal("accepted town LOAD did not retain and detach the process counter")
			}
		})
	}
}
