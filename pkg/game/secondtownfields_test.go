package game

import (
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// townFieldsSave is a town SAVE whose Humans row 30 holds a skill 10 and a
// skill 20, the campaign having kept the Player block in raw.
func townFieldsSave(t *testing.T, raw []byte) []byte {
	t.Helper()
	return identityTownSaveOf(t, func(tbl *mapload.Table, c *secondCampaign) {
		row := tbl.Humans.(dbCollection)[30].params
		row[11], row[12] = 10, 20
		if raw != nil {
			c.kept = &secondTailKept{Pairs: slices.Clone(secondNewPairs), PlayerRaw: slices.Clone(raw)}
		}
	})
}

func townFieldsDoc(t *testing.T, file []byte) *sav.DocumentData {
	t.Helper()
	doc, err := sav.DecodeDocumentData(file)
	if err != nil {
		t.Fatal(err)
	}
	return &doc
}

// A second-game town SAVE writes each member's +0x1c from his Humans row (the
// sum of the experience of its skills, trunc((1.1^s-1)*1000), times 0.01:
// 1593 + 5727 = 7320, so 73), an Item's +0xe as 0, the Player's +0xa48 as the
// sum of its kill-grade bytes and the selection as the hero's +4.
func TestSecondTownSaveWritesTheClaimedFields(t *testing.T) {
	doc := townFieldsDoc(t, townFieldsSave(t, nil))
	var companions, pieces int
	for i := range doc.Objects {
		r := &doc.Objects[i]
		switch r.Class {
		case "Human":
			if id, _ := savedStructureValue(r, "U148"); id&0xffff == 22 {
				if got, _ := savedStructureValue(r, "T1C"); got != 73 {
					t.Fatalf("the companion's +0x1c is %d, want 73", got)
				}
				companions++
			}
		case "Armor", "Weapon", "Shield", "Item":
			if got, _ := savedStructureValue(r, "T0E"); got != 0 {
				t.Fatalf("a %s holds +0xe %d, want 0", r.Class, got)
			}
			pieces++
		}
	}
	if companions != 1 {
		t.Fatalf("the file holds %d companions and %d pieces", companions, pieces)
	}
	player := &doc.Objects[doc.Players[0]-1]
	var sum uint32
	for _, b := range documentRawBytes(t, player, sav.KillGradeBlock) {
		sum += uint32(b)
	}
	if got, _ := savedStructureValue(player, "F48"); got != sum {
		t.Fatalf("the Player's +0xa48 is %d, the grade sum %d", got, sum)
	}
	hero, err := secondTownHeroRecord(*doc)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := savedStructureValue(&doc.Objects[hero-1], "RuntimeID")
	leaf, present, err := doc.State.ReadLeaf("/Objects/Selection")
	if err != nil || !present || len(leaf.Bytes) != 4 || binary.LittleEndian.Uint32(leaf.Bytes) != key || key == 0 {
		t.Fatalf("the selection holds %v (present %v, %v), want the hero's +4 %d", leaf.Bytes, present, err, key)
	}
}

// The Player's raw 8 at +0x10 is the loaded file's: written from the campaign
// that kept it, kept by a LOAD of the whole file and of its records
// alone, and zero where none was loaded.
func TestSecondTownSaveWritesTheLoadedPlayerBlock(t *testing.T) {
	want := []byte{0xcd, 0x07, 0x86, 0xa4, 0x76, 0x01, 0, 0}
	block := func(file []byte) []byte {
		doc := townFieldsDoc(t, file)
		return documentRawBytes(t, &doc.Objects[doc.Players[0]-1], "Raw10")
	}
	kept := func(file []byte) []byte {
		sf, err := sav.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		s, err := secondFileSession(sf)
		if err != nil || s.Second.Kept == nil {
			t.Fatalf("the file's session: %+v %v", s, err)
		}
		return s.Second.Kept.PlayerRaw
	}
	zero := townFieldsSave(t, nil)
	if got := block(zero); !slices.Equal(got, make([]byte, 8)) {
		t.Fatalf("a new game's Player block is %x, want zero", got)
	}
	if got := kept(zero); got != nil {
		t.Fatalf("a loaded zero Player block is kept as %x, want none", got)
	}
	file := townFieldsSave(t, want)
	if got := block(file); !slices.Equal(got, want) {
		t.Fatalf("the SAVE writes %x, want %x", got, want)
	}
	if got := kept(file); !slices.Equal(got, want) {
		t.Fatalf("a LOAD keeps the Player block %x, want %x", got, want)
	}
	for _, from := range []struct {
		name string
		raw  []byte
	}{{"its records alone", withoutEngineDocument(t, file)}, {"the whole file", file}} {
		if got := block(exportTown(t, coldIdentityTown(t, from.raw))); !slices.Equal(got, want) {
			t.Fatalf("a LOAD of %s and a SAVE write %x, want %x", from.name, got, want)
		}
	}
}
