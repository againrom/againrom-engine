package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func itemWeightEntity(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatal("missing entity", id)
	return sim.Entity{}
}

func TestReleaseOriginalItemWeightTransferAndNativeReload(t *testing.T) {
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-02/game0002.sav", "b1ce079cc2b3f1bc101862c1dcf2afd8237421458b3e2474c4447efe5df2e761")
	src, e := sav.Open(raw)
	if e != nil {
		t.Fatal(e)
	}
	chars, e := src.Party()
	if e != nil {
		t.Fatal(e)
	}
	var witch sav.Character
	for _, c := range chars {
		if c.MapUnitID == 21 {
			witch = c
		}
	}
	if witch.Off != 248 || len(witch.Items) != 1 || witch.Items[0].Weight != 1 || witch.Items[0].Stack != 3 {
		t.Fatal("natural source changed", witch)
	}
	f.SetDeterministicFrames(true)
	app := f.App("item-weight-source")
	initialSave, initialList, initialLoad := agsSaveSeams(f, SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(initialSave, initialList, initialLoad)
	groundAppLoad(t, app, initialList, filepath.Base(path))
	w := f.live.world
	var id sim.EntityID
	for _, a := range w.Entities() {
		if a.MapUnitID == 21 {
			id = a.ID
		}
	}
	if id == 0 {
		t.Fatal("Witch not rebound")
	}
	stock, _ := w.CarriedStacks(id)
	if len(stock) != 1 || stock[0].Weight != 1 || !stock[0].WeightPresent || stock[0].Count != 3 {
		t.Fatal("source weight missing", stock)
	}
	receiver := f.live.mission.ids[0]
	if receiver == id {
		receiver = f.live.mission.ids[1]
	}
	if got := itemWeightEntity(t, w, receiver).Load; got != 53 {
		t.Fatal("receiver before transfer", got)
	}
	if e = w.MoveCarried(id, receiver, 0x0e06, 2); e != nil {
		t.Fatal(e)
	}
	if got := itemWeightEntity(t, w, receiver).Load; got != 54 {
		t.Fatal("39+(29+2)/2 must be54, got", got)
	}
	pair := []uint32{itemWeightEntity(t, w, id).SourceBinding.RuntimeID, itemWeightEntity(t, w, receiver).SourceBinding.RuntimeID}
	want := itemWeightSeam(w, pair)
	store, name, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(path)})
	if got := itemWeightSeam(w, pair); got != want {
		t.Fatal("SAVE changed live weights", want, got)
	}
	out, e := sav.Open(written)
	if e != nil {
		t.Fatal(e)
	}
	saved, e := out.Party()
	if e != nil {
		t.Fatal(e)
	}
	for _, runtime := range pair {
		var live []string
		stacks, _ := w.CarriedStacks(world1170Entity(t, f, runtime).ID)
		for _, st := range stacks {
			live = append(live, fmt.Sprintf("%04x/%d/%d", st.Code, st.Count, st.Weight))
		}
		var doc []string
		for _, c := range saved {
			if c.RuntimeID == runtime {
				for _, piece := range c.Items {
					doc = append(doc, fmt.Sprintf("%04x/%d/%d", piece.Code, piece.Stack, piece.Weight))
				}
			}
		}
		slices.Sort(live)
		slices.Sort(doc)
		if len(live) == 0 || !slices.Equal(live, doc) {
			t.Fatal("written pack differs from live", runtime, live, doc)
		}
	}
	cold := loadLocalLegacySave(t, store, name)
	for tick := 0; ; tick++ {
		if got := itemWeightSeam(cold.live.world, pair); got != itemWeightSeam(f.live.world, pair) {
			t.Fatal("restored weights differ", tick, itemWeightSeam(f.live.world, pair), got)
		}
		if tick == 16 {
			break
		}
		f.live.tick()
		cold.live.tick()
	}
	lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
		for i := range doc.Objects {
			code, _ := savedStructureValue(&doc.Objects[i], "F40")
			count, _ := savedStructureValue(&doc.Objects[i], "F42")
			if savedItemClass(doc.Objects[i].Class) && code == 0x0e06 && count == 2 {
				savedObjectSetValue(&doc.Objects[i], "F4A", 9)
				return true
			}
		}
		return false
	})
	if itemWeightSeam(lost.live.world, pair) == want {
		t.Fatal("loss control: a SAV with the moved stack's weight altered still matches")
	}
	t.Logf("natural SAV F4A1 count3, transfer2 receiver53->54; menu SAVE %s, fresh LOAD and 16 ticks keep %s", name, want)
}

func itemWeightSeam(w *sim.World, runtimes []uint32) string {
	out := ""
	for _, runtime := range runtimes {
		for _, e := range w.Entities() {
			if e.SourceBinding.RuntimeID != runtime {
				continue
			}
			out += fmt.Sprintf("%d load%d:", runtime, e.Load)
			stacks, _ := w.CarriedStacks(e.ID)
			for _, st := range stacks {
				out += fmt.Sprintf(" %04x/%d/%d/%t", st.Code, st.Count, st.Weight, st.WeightPresent)
			}
			out += ";"
		}
	}
	return out
}

// The artifact is a genuine old-producer envelope outside Git. Its optional
// path makes the default no-install gate independent of owner files.
func TestInstanceWeightFrozenCityV3(t *testing.T) {
	root := os.Getenv("AGAINROM_WEIGHT_OLD_CITY")
	if root == "" {
		t.Skip("AGAINROM_WEIGHT_OLD_CITY is unset: no frozen predecessor city artifact")
	}
	path := filepath.Join(root, "trained.ags")
	raw, e := os.ReadFile(path)
	if e != nil {
		// Named but absent is a missing subject, not a gate failure, and the
		// reason deliberately omits the variable so a supplied-root run counts
		// it as lacking a subject instead of as a still-gated skip.
		t.Skip("frozen predecessor city artifact is absent")
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "60082e1537dad3ca76c4e84531c5380db71d4323ed3803dbb10414cf83819b76" {
		t.Fatal("old city artifact changed")
	}
	s, label, e := DecodeSave(raw)
	if e != nil {
		t.Fatal(e)
	}
	f := releaseFront(t)
	loadSecondPhysicalApp1104(t, f, path)
	loaded, _, e := f.Snapshot(false)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(s.Party, loaded.Party) || !reflect.DeepEqual(s.OriginalCity, loaded.OriginalCity) {
		t.Fatal("old city native state or source baseline changed")
	}
	exported, e := f.ExportOriginalSave(loaded, label)
	// This frozen AGS predates score history.
	if loaded.Fame == nil || loaded.Fame.Known || e != nil || len(exported) == 0 {
		t.Fatal("old city score-history export changed", e)
	}
	if _, town, e := releaseFront(t).RestoreOriginal(exported); e != nil || !town {
		t.Fatal("old city score-history export does not cold-load", town, e)
	}
	ags, e := EncodeSave(loaded, label)
	if e != nil {
		t.Fatal(e)
	}
	again, _, e := DecodeSave(ags)
	if e != nil || !reflect.DeepEqual(loaded, again) {
		t.Fatal("old city AGS recovery changed", e)
	}
	want, e := os.ReadFile(filepath.Join(root, "trained.sav"))
	if e != nil {
		t.Fatal(e)
	}
	newFront := releaseFront(t)
	if _, town, e := newFront.RestoreOriginal(want); e != nil || !town {
		t.Fatal("new source city LOAD", town, e)
	}
	for _, p := range newFront.Carried {
		for _, i := range p.WornItems {
			if i.Code != 0 && !i.WeightPresent {
				t.Fatal("new city source weight absent")
			}
		}
	}
	t.Log("genuine city-v3 App LOAD/AGS retains party+baseline; absent score history now exports and cold-loads (DIV-1319); preserved historical SAV still imports source weights")
}
