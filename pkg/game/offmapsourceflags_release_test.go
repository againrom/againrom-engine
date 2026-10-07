package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func sourceFlagsWireValue(t *testing.T, raw []byte, identity uint32) uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		key, _ := savedStructureValue(r, "Identity")
		if key == identity {
			value, err := savedStructureValue(r, "U4C")
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
	}
	t.Fatalf("SAV has no source actor identity %#x", identity)
	return 0
}

func loadSourceFlagsMission(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("original mission LOAD: town=%t error=%v", town, err)
	}
	if err := f.App("source flags round trip").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func saveSourceFlagsMission(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("current SAV: name=%q error=%v", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestReleaseOffMapSourceFlagsSurviveCurrentSaveLoad(t *testing.T) {
	root := os.Getenv("AGAINROM_SAVE_CORPUS")
	if os.Getenv("AGAINROM_ASSETS") == "" || root == "" {
		t.Skip("AGAINROM_ASSETS and AGAINROM_SAVE_CORPUS required")
	}
	for _, tc := range []struct{ name, sha string }{
		{"game9999.sav", "8DA6BEC860289AC804563D5A55DF0E201D2403138ED6C55B3A32B43073F1CD09"},
		{"oldsaves7/game9999.sav", "29AEA649C78893C296B09FBD6F2F3B3ECCC3346168D058C58B2502751C89F2CA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, "2026-09-27", filepath.FromSlash(tc.name)))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%X", sha256.Sum256(raw)); got != tc.sha {
				t.Fatalf("owner SAV SHA256=%s, want %s", got, tc.sha)
			}
			f := loadSourceFlagsMission(t, raw)
			initial, initialFound := f.live.world.Entity(66)
			if !initialFound || initial.MapUnitID != 106 {
				t.Fatalf("owner input changed actor placement: %+v found=%t", initial.SourceBinding, initialFound)
			}
			f.LiveAdvance(30)
			party := make(map[sim.EntityID]bool)
			for _, id := range f.live.mission.ids {
				party[id] = true
			}
			killed := 0
			for _, e := range f.live.world.Entities() {
				if killed == 5 {
					break
				}
				if party[e.ID] || e.Owner == sim.SelfSlot || e.HP <= 0 || e.Decay != sim.DecayNone {
					continue
				}
				f.LiveKill(uint32(e.ID))
				killed++
			}
			f.LiveAdvance(2400)
			played, playedFound := f.live.world.Entity(initial.ID)
			if !playedFound || played.SourceBinding.Identity != initial.SourceBinding.Identity {
				t.Fatal("played mission lost the actor's source identity")
			}
			written := saveSourceFlagsMission(t, f)
			sourceWire := sourceFlagsWireValue(t, raw, initial.SourceBinding.Identity)
			writtenWire := sourceFlagsWireValue(t, written, initial.SourceBinding.Identity)
			if sourceWire != 0x06 || initial.SourceBinding.ClassFlags != 0x06 || initial.OffMap ||
				played.SourceBinding.ClassFlags != 0x06 || !played.OffMap || writtenWire != 0x0e {
				t.Fatalf("actor 66 source U4C=%#x initial flags=%#x off-map=%t; played flags=%#x off-map=%t; current SAV U4C=%#x", sourceWire, initial.SourceBinding.ClassFlags, initial.OffMap, played.SourceBinding.ClassFlags, played.OffMap, writtenWire)
			}
			g := loadSourceFlagsMission(t, written)
			compare := func(phase string) {
				before := make(map[uint32]sim.Entity)
				for _, e := range f.live.world.Entities() {
					if e.SourceBinding.Identity != 0 {
						before[e.SourceBinding.Identity] = e
					}
				}
				seen, foundTarget := 0, false
				for _, after := range g.live.world.Entities() {
					prior, ok := before[after.SourceBinding.Identity]
					if !ok {
						continue
					}
					seen++
					if after.ID == initial.ID {
						foundTarget = true
					}
					if after.SourceBinding.ClassFlags != prior.SourceBinding.ClassFlags || after.OffMap != prior.OffMap {
						t.Errorf("%s actor id=%d identity=%#x class=%d map=%d HP=%d decay=%d flags %#x -> %#x, off-map %t -> %t", phase, after.ID, after.SourceBinding.Identity, after.SourceBinding.Class, after.MapUnitID, prior.HP, prior.Decay, prior.SourceBinding.ClassFlags, after.SourceBinding.ClassFlags, prior.OffMap, after.OffMap)
					}
				}
				if seen == 0 || !foundTarget {
					t.Fatalf("%s did not compare off-map actor 66 among source actors", phase)
				}
			}
			compare("cold LOAD")
			f.LiveAdvance(120)
			g.LiveAdvance(120)
			compare("next 120 ticks")
		})
	}
}
