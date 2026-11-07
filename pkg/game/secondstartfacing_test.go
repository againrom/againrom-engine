package game

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// secondFacingMission is mission 10 holding five knights of Player 2, map
// unit IDs 7..11, along row 48.
func secondFacingMission() secondMissionContent {
	c := secondGuardMission()
	c.units = nil
	for i := range 5 {
		c.units = append(c.units, alm.Unit{X: uint32(44+2*i)<<8 | 0x80, Y: 48<<8 | 0x80, ClassID: 9, ClassSubID: 1, Flags: 0x10,
			ServerID: 7301, Owner: 2, UnitID: uint16(7 + i), GroupID: 5})
	}
	return c
}

// secondKnightFacings is the World's knights' facings by map unit ID; each
// must equal its heading.
func secondKnightFacings(t *testing.T, w *sim.World) map[uint16]uint8 {
	t.Helper()
	out := map[uint16]uint8{}
	for _, e := range w.Entities() {
		if e.MapUnitID >= 7 && e.MapUnitID <= 11 && e.Owner == 2 {
			if e.DesiredFacing != e.Facing {
				t.Errorf("knight %d faces %#x heading %#x", e.MapUnitID, e.Facing, e.DesiredFacing)
			}
			out[e.MapUnitID] = e.Facing
		}
	}
	if len(out) != 5 {
		t.Fatalf("the World holds %d knights, want 5", len(out))
	}
	return out
}

// secondKnightRecords is each knight record's Mover block (U154) in doc, by
// map unit ID.
func secondKnightRecords(t *testing.T, doc sav.DocumentData) map[uint16][]byte {
	t.Helper()
	out := map[uint16][]byte{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		if v, err := savedStructureValue(r, "T08"); err == nil && v >= 7 && v <= 11 {
			out[uint16(v)] = secondRecordRaw(r, "U154")
		}
	}
	if len(out) != 5 {
		t.Fatalf("the file holds %d knight records, want 5", len(out))
	}
	return out
}

// A second-game START MISSION gives each map unit the facing Unit
// initialization draws, 0x40 + 0..128, equal to its heading (R2-ENGINE-959,
// R2-ENGINE-960); the SAVE writes it in byte 0 and the heading in byte 1 of
// the unit's fifth raw block, and LOAD restores it (R2-SESSION-362). The
// file's records alone, as of an original save, load the bytes they hold.
// Fails while every map unit starts facing north.
func TestSecondMissionStartFacingsSurviveSAVEAndLOAD(t *testing.T) {
	c := secondFacingMission()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	live := secondKnightFacings(t, f.live.world)
	distinct := map[uint8]bool{}
	for id, facing := range live {
		if facing < 0x40 || facing > 0xC0 {
			t.Errorf("knight %d starts facing %#x, want 0x40..0xC0", id, facing)
		}
		distinct[facing] = true
	}
	if len(distinct) < 2 {
		t.Fatalf("all five knights start facing %#x", live)
	}

	raw := secondPageSave(t, f, app)
	doc := checkSecondMissionFile(t, raw, 10)
	for id, block := range secondKnightRecords(t, doc) {
		if len(block) != 180 || block[0] != live[id] || block[1] != live[id] {
			t.Errorf("knight %d writes Mover bytes %#x, want facing and heading %#x", id, block[:2], live[id])
		}
	}
	cold := secondMissionLoadBytesOf(t, c, raw)
	if cold.live == nil {
		t.Fatal("the file does not LOAD")
	}
	if got := secondKnightFacings(t, cold.live.world); !maps.Equal(got, live) {
		t.Fatalf("LOAD: facings %#x, want %#x", got, live)
	}

	// The records alone, one knight's bytes planted as an original save
	// holds them: LOAD takes the planted facing.
	const planted = 0x55
	bareDoc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	bareDoc.State.ValueRecords = slices.DeleteFunc(slices.Clone(bareDoc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
		return strings.EqualFold(r.Path, sav.NativeFormPath)
	})
	block := secondKnightRecords(t, bareDoc)[9]
	block[0], block[1] = planted, planted
	bare, err := sav.EncodeDocumentData(bareDoc)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), bare, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, _ := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if loaded.live == nil {
		t.Fatal("the file's records alone do not LOAD")
	}
	want := maps.Clone(live)
	want[9] = planted
	if got := secondKnightFacings(t, loaded.live.world); !maps.Equal(got, want) {
		t.Fatalf("the records' LOAD: facings %#x, want %#x", got, want)
	}
}
