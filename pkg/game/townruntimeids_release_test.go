package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

func townHumanTokens(t *testing.T, raw []byte) (ids, masks []uint32) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[uint32]bool{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if !runtimePlaceable(r.Class) {
			continue
		}
		id, _ := savedStructureValue(r, "RuntimeID")
		if r.Class != "Human" {
			seen[id] = id != 0
			continue
		}
		mask, _ := savedStructureValue(r, "T18")
		if id == 0 || id > 65535 || seen[id] {
			t.Errorf("Human runtime id %#x is outside the original's id space or repeated", id)
		}
		seen[id] = true
		ids, masks = append(ids, id), append(masks, mask)
	}
	return ids, masks
}

func setTownHumanTokens(t *testing.T, raw []byte, ids []uint32, mask uint32) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for i := range doc.Objects {
		if r := &doc.Objects[i]; r.Class == "Human" {
			savedObjectSetValue(r, "RuntimeID", ids[n])
			savedObjectSetValue(r, "T18", mask)
			n++
		}
	}
	if n != len(ids) {
		t.Fatalf("town holds %d Humans, want %d", n, len(ids))
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestReleaseTownSaveHumanRuntimeIDs(t *testing.T) {
	wide := setTownHumanTokens(t, currentTownSave(t, currentTown(t, nil, nil)), []uint32{0x21000089, 0x051000B7}, 0)
	g := currentTownReload(t, wide)
	repaired := currentTownSave(t, g)
	ids, masks := townHumanTokens(t, repaired)
	for i := range ids {
		if masks[i] != 2 {
			t.Errorf("Human %d publication mask %d, want 2", i, masks[i])
		}
	}
	h := currentTownReload(t, repaired)
	app := h.App("town runtime ids")
	app.Layout(1024, 768)
	if err := app.OpenMission(h.MissionOpener(h.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	h.SetDeterministicFrames(true)
	if len(h.live.mission.ids) != len(ids) {
		t.Fatalf("mission holds %d heroes, the town %d", len(h.live.mission.ids), len(ids))
	}
	for _, hero := range h.live.mission.ids {
		start, _ := h.live.entity(hero)
		moved := false
		for _, d := range [][2]int{{3, 0}, {-3, 0}, {0, 3}, {0, -3}} {
			h.LiveOrder(uint32(hero), int(start.X)+d[0], int(start.Y)+d[1])
			h.LiveAdvance(40)
			if now, ok := h.live.entity(hero); ok && (now.X != start.X || now.Y != start.Y) {
				moved = true
				break
			}
		}
		if !moved {
			t.Errorf("hero %d did not move after the repaired town", hero)
		}
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := h.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	mission, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	townHumanTokens(t, mission)

	_, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "city.sav"), source, 0o644); err != nil {
		t.Fatal(err)
	}
	o := releaseFront(t)
	_, _, load := o.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	if _, town, err := load("city.sav"); err != nil || !town {
		t.Fatalf("LOAD of the original town: town=%v err=%v", town, err)
	}
	before, _ := townHumanTokens(t, source)
	after, masks := townHumanTokens(t, currentTownSave(t, o))
	if len(before) != len(after) || masks[0] != 2 || masks[1] != 2 || before[0] != after[0] || before[1] != after[1] {
		t.Fatalf("original town Human ids %v -> %v, masks %v", before, after, masks)
	}
}
