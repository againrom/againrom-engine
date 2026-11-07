package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// secondAttackRecords is the record index and command target of each Unit
// and Human record of doc under order 3.
func secondAttackRecords(doc *sav.DocumentData) map[int]uint32 {
	out := map[int]uint32{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		order, command := secondRecordRaw(r, "U50"), secondRecordRaw(r, "URaw184")
		if (r.Class == "Unit" || r.Class == "Human") && len(order) == 4 && len(command) == 184 && binary.LittleEndian.Uint32(order) == 3 {
			out[i] = binary.LittleEndian.Uint32(command[0x0c:])
		}
	}
	return out
}

// A second-game mission SAVE writes the hero's attack order with its
// target's record key at command +0x0c: the original's planner reads that
// target before any null test (R2-ENGINE-936). A file whose records alone
// hold the hero under order 3 with target 0 loads, and the next SAVE
// writes no record under order 3 with target 0. Fails while the second
// game's command block is written with +0x0c zero.
func TestSecondMissionSaveWritesTheAttackTarget(t *testing.T) {
	c := secondGuardMission()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	w := f.live.world
	var hero, knight sim.EntityID
	heroFound, knightFound := false, false
	for _, e := range w.Entities() {
		if e.Owner == sim.SelfSlot && !heroFound {
			hero, heroFound = e.ID, true
		}
		if e.MapUnitID == 7 {
			knight, knightFound = e.ID, true
		}
	}
	if !heroFound || !knightFound {
		t.Fatal("the mission holds no hero or no knight 7")
	}
	sim.Step(w, []sim.Command{sim.Attack(hero, knight)})
	raw := secondPageSave(t, f, app)
	doc := checkSecondMissionFile(t, raw, 10)
	key, err := savedStructureValue(secondKnightRecord(t, &doc, 7), "Identity")
	if err != nil || key == 0 {
		t.Fatalf("knight 7's record key %#x: %v", key, err)
	}
	attacks := secondAttackRecords(&doc)
	if len(attacks) != 1 {
		t.Fatalf("%d records under order 3, want the hero's: %v", len(attacks), attacks)
	}
	at := -1
	for i, target := range attacks {
		if at = i; target != key {
			t.Fatalf("the hero's attack order names %#x, want knight 7's key %#x", target, key)
		}
	}

	bareDoc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	bareDoc.State.ValueRecords = slices.DeleteFunc(slices.Clone(bareDoc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
		return strings.EqualFold(r.Path, sav.NativeFormPath)
	})
	binary.LittleEndian.PutUint32(secondRecordRaw(&bareDoc.Objects[at], "URaw184")[0x0c:], 0)
	bare, err := sav.EncodeDocumentData(bareDoc)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), bare, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, loadedApp := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if loaded.live == nil {
		t.Fatal("the records' order 3 with target 0 does not LOAD")
	}
	resaved := checkSecondMissionFile(t, secondPageSave(t, loaded, loadedApp), 10)
	for i, target := range secondAttackRecords(&resaved) {
		if target == 0 {
			t.Fatalf("the resave's record %d holds order 3 with target 0", i+1)
		}
	}
	name := bareDoc.Objects[at].Texts
	for i := range resaved.Objects {
		r := &resaved.Objects[i]
		if r.Class != bareDoc.Objects[at].Class || !slices.Equal(r.Texts, name) {
			continue
		}
		if order := binary.LittleEndian.Uint32(secondRecordRaw(r, "U50")); order == 3 {
			t.Fatalf("the resaved hero holds order 3 with target %#x", binary.LittleEndian.Uint32(secondRecordRaw(r, "URaw184")[0x0c:]))
		}
		return
	}
	t.Fatal("the resave holds no record of the hero")
}
