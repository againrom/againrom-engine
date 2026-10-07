package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func roodDocumentActor(t *testing.T, doc sav.DocumentData) (uint16, uint32) {
	t.Helper()
	var object uint16
	var identity uint32
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		unit, _ := savedStructureValue(r, "T08")
		runtime, _ := savedStructureValue(r, "RuntimeID")
		if unit != 443 || runtime != 258 {
			continue
		}
		if object != 0 {
			t.Fatal("Rood has repeated Human records")
		}
		object = uint16(i + 1)
		identity, _ = savedStructureValue(r, "Identity")
	}
	if object == 0 || identity == 0 {
		t.Fatal("Rood's bound Human record is absent")
	}
	return object, identity
}

func roodRootKeys(t *testing.T, doc sav.DocumentData) []uint32 {
	t.Helper()
	keys := make([]uint32, 0, len(doc.DeadActors))
	for _, object := range doc.DeadActors {
		if object == 0 || int(object) > len(doc.Objects) {
			t.Fatal("invalid DeadActors reference", object)
		}
		key, err := savedStructureValue(&doc.Objects[object-1], "Identity")
		if err != nil || key == 0 {
			t.Fatal("DeadActors identity", object, err)
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func roodGroupRecord(t *testing.T, doc sav.DocumentData) sav.DocumentRecordData {
	t.Helper()
	for _, object := range doc.Players {
		p := &doc.Objects[object-1]
		slot, _ := savedStructureValue(p, "Slot")
		if slot == 5 {
			if len(p.Groups) == 0 {
				t.Fatal("Rood's Player has no Group 0")
			}
			return p.Groups[0]
		}
	}
	t.Fatal("Rood's Player is absent")
	return sav.DocumentRecordData{}
}

func requireRoodPlacement(t *testing.T, doc sav.DocumentData, hp int16, stage uint32, rooted bool) {
	t.Helper()
	object, identity := roodDocumentActor(t, doc)
	r := &doc.Objects[object-1]
	for name, want := range map[string]uint32{
		"Health": uint32(uint16(hp)), "HealthMax": 86, "Stage": stage,
		"U6C": 0, "RuntimeID": 258, "T08": 443,
	} {
		got, err := savedStructureValue(r, name)
		if err != nil || got != want {
			t.Fatalf("Rood %s=%d err=%v, want %d", name, got, err, want)
		}
	}
	inRoot := slices.Contains(roodRootKeys(t, doc), identity)
	if inRoot != rooted {
		t.Fatalf("Rood DeadActors=%v, want %v", inRoot, rooted)
	}
	groupCount := 0
	for _, player := range doc.Players {
		p := &doc.Objects[player-1]
		slot, _ := savedStructureValue(p, "Slot")
		if slot != 5 {
			continue
		}
		for i := range p.Groups {
			members, ok := savedObjectRefs(&p.Groups[i], "Actors")
			if ok && slices.Contains(members, object) {
				if i != 0 {
					t.Fatalf("Rood moved to Player 4 Group %d", i)
				}
				groupCount++
			}
		}
	}
	if !rooted && groupCount != 1 {
		t.Fatalf("Rood Player 4 Group 0 references=%d, rooted=%v", groupCount, rooted)
	}
	if rooted && groupCount != 0 {
		t.Fatalf("torn-down Rood remains in Player 4 Group: %d references", groupCount)
	}
}

func loadRoodMission(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("Rood original LOAD", town, err)
	}
	if err := f.App("Rood root boundary").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func saveRoodMission(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatal("Rood ordinary SAVE", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestReleaseZeroHealthDeadRootBoundary(t *testing.T) {
	path := os.Getenv("AGAINROM_ROOD_SAV")
	if path == "" {
		t.Skip("no AGAINROM_ROOD_SAV: exact original Rood SAV is required")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%X", sha256.Sum256(source)); got != "28E705DA3AE24A6537082C110D3C99A6715E79B2E0F1B005B2EAD8F553CD2DC1" {
		t.Fatalf("original Rood SAV hash=%s", got)
	}
	sourceDoc, err := sav.DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	requireRoodPlacement(t, sourceDoc, 0, 1, false)
	t.Run("zero_source", func(t *testing.T) {
		f := loadRoodMission(t, source)
		rood := worldEntityByRuntimeID(t, f, 258)
		if rood.HP != 0 || rood.Decay != sim.DecayFallen {
			t.Fatal("loaded Rood state", rood.HP, rood.Decay)
		}
		written := saveRoodMission(t, f)
		out, err := sav.DecodeDocumentData(written)
		if err != nil {
			t.Fatal(err)
		}
		requireRoodPlacement(t, out, 0, 1, false)
		if got, want := roodRootKeys(t, out), roodRootKeys(t, sourceDoc); !slices.Equal(got, want) {
			t.Fatalf("DeadActors root identities=%v, want %v", got, want)
		}
		sourceObject, _ := roodDocumentActor(t, sourceDoc)
		outObject, _ := roodDocumentActor(t, out)
		if !reflect.DeepEqual(out.Objects[outObject-1], sourceDoc.Objects[sourceObject-1]) || !reflect.DeepEqual(roodGroupRecord(t, out), roodGroupRecord(t, sourceDoc)) {
			t.Fatal("zero-health SAVE changed Rood's complete Human record or Group 0")
		}
		cold := loadRoodMission(t, written)
		for i := 0; i < 256; i++ {
			cold.live.tick()
		}
		rood = worldEntityByRuntimeID(t, cold, 258)
		if rood.HP != 0 || rood.Decay != sim.DecayFallen {
			t.Fatal("cold continuation changed zero-health Rood", rood.HP, rood.Decay)
		}
		back, err := sav.DecodeDocumentData(saveRoodMission(t, cold))
		if err != nil {
			t.Fatal(err)
		}
		requireRoodPlacement(t, back, 0, 1, false)
	})
	t.Run("loaded_stale_root", func(t *testing.T) {
		contaminated := alterSAV(t, source, func(doc *sav.DocumentData) bool {
			object, _ := roodDocumentActor(t, *doc)
			doc.DeadActors = append(doc.DeadActors, object)
			return true
		})
		f := loadRoodMission(t, contaminated)
		first := saveRoodMission(t, f)
		written, err := sav.DecodeDocumentData(first)
		if err != nil {
			t.Fatal(err)
		}
		requireRoodPlacement(t, written, 0, 1, false)
		if got, want := roodRootKeys(t, written), roodRootKeys(t, sourceDoc); !slices.Equal(got, want) {
			t.Fatalf("stale root survived ordinary SAVE: got %v, want %v", got, want)
		}
		cold := loadRoodMission(t, first)
		for i := 0; i < 256; i++ {
			cold.live.tick()
		}
		rood := worldEntityByRuntimeID(t, cold, 258)
		if rood.HP != 0 || rood.Decay != sim.DecayFallen {
			t.Fatal("cold continuation changed Rood", rood.HP, rood.Decay)
		}
		back, err := sav.DecodeDocumentData(saveRoodMission(t, cold))
		if err != nil {
			t.Fatal(err)
		}
		requireRoodPlacement(t, back, 0, 1, false)
	})
	for _, hp := range []int16{-1, -9} {
		t.Run(fmt.Sprintf("stage_one_%d", hp), func(t *testing.T) {
			input := alterSAV(t, source, func(doc *sav.DocumentData) bool {
				object, _ := roodDocumentActor(t, *doc)
				savedObjectSetValue(&doc.Objects[object-1], "Health", uint32(uint16(hp)))
				return true
			})
			f := loadRoodMission(t, input)
			rood := worldEntityByRuntimeID(t, f, 258)
			if rood.HP != int32(hp) || rood.Decay != sim.DecayFallen {
				t.Fatal("loaded shallow body", rood.HP, rood.Decay)
			}
			out, err := sav.DecodeDocumentData(saveRoodMission(t, f))
			if err != nil {
				t.Fatal(err)
			}
			requireRoodPlacement(t, out, hp, 1, false)
			if hp != -9 {
				return
			}
			for i := 0; i < 256 && rood.Decay < sim.DecayBones; i++ {
				f.live.tick()
				rood = worldEntityByRuntimeID(t, f, 258)
			}
			if rood.Decay < sim.DecayBones {
				t.Fatal("negative-health Rood did not reach teardown", rood.HP, rood.Decay)
			}
			out, err = sav.DecodeDocumentData(saveRoodMission(t, f))
			if err != nil {
				t.Fatal(err)
			}
			requireRoodPlacement(t, out, int16(rood.HP), uint32(rood.Decay), true)
			cold := loadRoodMission(t, saveRoodMission(t, f))
			back, err := sav.DecodeDocumentData(saveRoodMission(t, cold))
			if err != nil {
				t.Fatal(err)
			}
			requireRoodPlacement(t, back, int16(rood.HP), uint32(rood.Decay), true)
		})
	}
}
