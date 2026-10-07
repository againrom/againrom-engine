package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

// The corpus's one nonempty Projectiles store.
const carriedCastSource = "2026-08-15/game0018.sav"
const carriedCastSourceHash = "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b"

func carriedCastLabel(edition string) string {
	if edition == "RU" {
		return "9612 Стрела v2 RU"
	}
	return "9611 стрела v2 EN"
}

func carriedCastFile(edition string) string {
	if edition == "RU" {
		return "game9612.sav"
	}
	return "game9611.sav"
}

// carriedSpellChain returns every record reachable from the world effect roots
// in first-reach order, with each reference replaced by its position in that
// order, so two documents compare without their archive numbering.
func carriedSpellChain(doc sav.DocumentData) []sav.DocumentRecordData {
	position := map[uint16]uint16{}
	var order []uint16
	var visit func(uint16)
	visit = func(index uint16) {
		if index == 0 || position[index] != 0 {
			return
		}
		order = append(order, index)
		position[index] = uint16(len(order))
		for _, slot := range doc.Objects[index-1].RefSlots {
			for _, child := range slot.Objects {
				visit(child)
			}
		}
	}
	for _, root := range doc.World.Effects {
		visit(root)
	}
	out := make([]sav.DocumentRecordData, len(order))
	for i, index := range order {
		r := doc.Objects[index-1]
		r.RefSlots = nil
		for _, slot := range doc.Objects[index-1].RefSlots {
			mapped := sav.DocumentRefsData{Name: slot.Name}
			for _, child := range slot.Objects {
				mapped.Objects = append(mapped.Objects, position[child])
			}
			r.RefSlots = append(r.RefSlots, mapped)
		}
		out[i] = r
	}
	return out
}

// carriedOrderValues is the attack order word set of one actor record.
func carriedOrderValues(t *testing.T, r sav.DocumentRecordData) map[string]uint32 {
	t.Helper()
	out := map[string]uint32{}
	for _, name := range []string{"U5C", "U6C", "U136"} {
		v, err := savedStructureValue(&r, name)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = v
	}
	raw, err := savedMotionRaw(&r, "U58", 4)
	if err != nil {
		t.Fatal(err)
	}
	out["U58"] = uint32(raw[0])
	return out
}

// carriedCastSave loads the source through the original LOAD door and saves it
// with the ordinary F2 SAVE.
func carriedCastSave(t *testing.T) (source, saved []byte, edition string) {
	t.Helper()
	_, source = groundCorpusFile(t, carriedCastSource, carriedCastSourceHash)
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, source, "source.sav")
	t.Cleanup(app.StopAudio)
	if f.live == nil || f.live.world == nil || f.liveMission != 40 {
		t.Fatalf("source LOAD opened mission %d", f.liveMission)
	}
	edition = kitOwnerEdition(f)
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	saved = cityRosterF2Save(t, app, store, carriedCastLabel(edition))
	return source, saved, edition
}

func checkCarriedCastSave(t *testing.T, source, saved []byte) {
	t.Helper()
	want, err := sav.DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	got, err := sav.DecodeDocumentData(saved)
	if err != nil {
		t.Fatal(err)
	}
	wantChain, gotChain := carriedSpellChain(want), carriedSpellChain(got)
	if len(wantChain) != 3 || wantChain[0].Class != "SpellTransport" || wantChain[1].Class != "PointEffect" || wantChain[2].Class != "Effect_DirectDamage" {
		t.Fatalf("the source holds an unexpected effect graph: %d records", len(wantChain))
	}
	if !reflect.DeepEqual(wantChain, gotChain) {
		t.Fatalf("SAVE changed the SpellTransport graph:\nsource %+v\nsaved  %+v", wantChain, gotChain)
	}
	if len(want.World.Effects) != len(got.World.Effects) {
		t.Fatalf("world effect roots %d, source %d", len(got.World.Effects), len(want.World.Effects))
	}
	// Each actor that is ordered to attack keeps its order words.
	for i, r := range want.Objects {
		if r.Class != "Human" && r.Class != "Unit" {
			continue
		}
		if target, _ := savedStructureValue(&r, "U5C"); target == 0 {
			continue
		}
		key, _ := savedStructureValue(&r, "Identity")
		for _, g := range got.Objects {
			if gk, _ := savedStructureValue(&g, "Identity"); gk == key && g.Class == r.Class {
				if w, o := carriedOrderValues(t, r), carriedOrderValues(t, g); !reflect.DeepEqual(w, o) {
					t.Errorf("actor object %d identity %d order words changed: source %v saved %v", i+1, key, w, o)
				}
			}
		}
	}
}

// TestReleaseCarriedSpellTransportSurvivesSave pins the SpellTransport graph
// and the attack order words of the original source through the ordinary SAVE.
func TestReleaseCarriedSpellTransportSurvivesSave(t *testing.T) {
	source, saved, _ := carriedCastSave(t)
	checkCarriedCastSave(t, source, saved)
}

// TestKitProjectileV2Build writes the v2 kit file into AGAINROM_OWNER_KIT_OUT,
// or into a temporary directory when it is unset.
func TestKitProjectileV2Build(t *testing.T) {
	source, saved, edition := carriedCastSave(t)
	checkCarriedCastSave(t, source, saved)
	kitOwnerWrite(t, kitOwnerOutput(t, edition), carriedCastFile(edition), saved)
	loaded := strings.Join(kitOwnerReceive(t, saved), "; ")
	for _, need := range []string{"projectile records 1, projectile drivers 1", "after: projectile records 0, projectile drivers 0"} {
		if !strings.Contains(loaded, need) {
			t.Fatalf("cold LOAD of the v2 file lacks %q: %s", need, loaded)
		}
	}
	t.Logf("BUILD %s label %q bytes %d", edition, carriedCastLabel(edition), len(saved))
}
