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

// secondSightMission is the guard mission over an eight-record roster whose
// rows plant stances off the diagonal.
func secondSightMission() secondMissionContent {
	c := secondGuardMission()
	c.players = 8
	c.stances = map[[2]int]uint16{{0, 1}: 1, {1, 0}: 1, {2, 4}: 2, {6, 3}: 1, {4, 0}: 1}
	return c
}

// secondCommandWord is the word every original-made Unit and Human command
// block stores at +0x16 (R2-SESSION-339).
const secondCommandWord = 65500

// checkSecondSessionSight: the file's 70x70 session matrix holds the World's
// relation byte in every cell, ORed with 0x10 on the diagonal of each of
// the file's eight Player slots, and zero outside the World's 50x50.
func checkSecondSessionSight(t *testing.T, name string, raw []byte, rel sim.Relations) {
	t.Helper()
	doc := checkSecondMissionFile(t, raw, 10)
	var slots []int
	for _, p := range doc.Players {
		slot, _ := savedStructureValue(&doc.Objects[p-1], "Slot")
		slots = append(slots, int(slot))
	}
	slices.Sort(slots)
	if !slices.Equal(slots, []int{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Fatalf("%s: the file's Players hold slots %v, want 1..8", name, slots)
	}
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	for a := range 70 {
		for b := range 70 {
			got, err := f.Diplomacy(a, b)
			if err != nil {
				t.Fatal(err)
			}
			want := rel.Byte(uint32(a), uint32(b))
			if a == b && slices.Contains(slots, a) {
				want |= 0x10
			}
			if got != want {
				t.Fatalf("%s: M[%d][%d] is %#02x, want %#02x", name, a, b, got, want)
			}
		}
	}
}

// checkSecondCommandWords: every Unit and Human record's command block holds
// the command word at +0x16, the post at +0 and zero in every other byte.
func checkSecondCommandWords(t *testing.T, name string, raw []byte) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	humans := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" {
			continue
		}
		block := secondRecordRaw(r, "URaw184")
		if len(block) != 184 {
			t.Fatalf("%s: %s record %d holds a %d-byte command block", name, r.Class, i+1, len(block))
		}
		if w := binary.LittleEndian.Uint16(block[0x16:]); w != secondCommandWord {
			t.Fatalf("%s: %s record %d writes command word %d, want %d", name, r.Class, i+1, w, secondCommandWord)
		}
		for at, v := range block {
			if at >= 2 && (at < 0x16 || at > 0x17) && v != 0 {
				t.Fatalf("%s: %s record %d command block byte %#x is %#x", name, r.Class, i+1, at, v)
			}
		}
		if r.Class == "Human" {
			humans++
		}
	}
	if humans < 3 {
		t.Fatalf("%s: %d Human records, want the party and two knights", name, humans)
	}
}

// A second-game mission SAVE writes, in its session matrix, bit 0x10 of
// M[Q][P] wherever the World's fog lets Player P see by Player Q's bodies:
// each Player's own diagonal (R2-ENGINE-826, R2-SESSION-339). Every other
// cell is the World's relation byte. Every Unit and Human command block
// writes word +0x16 65500, the value of every original-made record. The
// engine's LOAD and the LOAD of the records alone, the latter with each
// diagonal planted at 2 as an older Againrom file holds it, both SAVE the
// same again. Fails while the matrix diagonal is 2 and +0x16 is 0.
func TestSecondMissionSaveWritesOwnSightAndCommandWord(t *testing.T) {
	c := secondSightMission()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	rel := f.live.world.Relations()
	for at, w := range c.stances {
		if got := rel.Byte(uint32(at[0]+1), uint32(at[1]+1)); got != byte(w) {
			t.Fatalf("the World holds stance %d at [%d][%d], want the map's %d", got, at[0]+1, at[1]+1, w)
		}
	}
	raw := secondPageSave(t, f, app)
	checkSecondSessionSight(t, "SAVE", raw, rel)
	checkSecondCommandWords(t, "SAVE", raw)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold, coldApp := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if cold.live == nil {
		t.Fatal("the file does not LOAD")
	}
	again := secondPageSave(t, cold, coldApp)
	checkSecondSessionSight(t, "SAVE after LOAD", again, cold.live.world.Relations())
	checkSecondCommandWords(t, "SAVE after LOAD", again)

	// The records alone: their LOAD keeps the diagonal they hold, the file's
	// 0x12 or a planted 2, and its SAVE writes 0x12 from either.
	for _, diagonal := range []byte{0x12, 2} {
		bareDoc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		bareDoc.State.ValueRecords = slices.DeleteFunc(slices.Clone(bareDoc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
			return strings.EqualFold(r.Path, sav.NativeFormPath)
		})
		for i := 1; i <= 8; i++ {
			bareDoc.World.Session.Diplomacy[i][i] = diagonal
		}
		bare, err := sav.EncodeDocumentData(bareDoc)
		if err != nil {
			t.Fatal(err)
		}
		bareDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(bareDir, "game0001.sav"), bare, 0o600); err != nil {
			t.Fatal(err)
		}
		loaded, loadedApp := secondMissionLoadOf(t, c, bareDir, "game0001.sav")
		if loaded.live == nil {
			t.Fatal("the file's records alone do not LOAD")
		}
		loadedRel := loaded.live.world.Relations()
		for i := uint32(1); i <= 8; i++ {
			if got := loadedRel.Byte(i, i); got != diagonal {
				t.Fatalf("the records' LOAD holds M[%d][%d] %#02x, want the file's %#02x", i, i, got, diagonal)
			}
		}
		resaved := secondPageSave(t, loaded, loadedApp)
		checkSecondSessionSight(t, "SAVE after the records' LOAD", resaved, loadedRel)
		checkSecondCommandWords(t, "SAVE after the records' LOAD", resaved)
	}
}
