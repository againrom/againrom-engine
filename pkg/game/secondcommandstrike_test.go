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

// secondKnightRecord is the record of knight id in doc.
func secondKnightRecord(t *testing.T, doc *sav.DocumentData, id uint16) *sav.DocumentRecordData {
	t.Helper()
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if v, _ := savedStructureValue(r, "T08"); r.Class == "Human" && uint16(v) == id {
			return r
		}
	}
	t.Fatalf("no record of knight %d", id)
	return nil
}

// The planted command block of knight 7: struck by knight 8 at fine clock
// 40, walking away to 45/46, stamped at group tick 3, its alarm and the
// striker's cell set (R2-ENGINE-845, R2-ENGINE-846).
const (
	strikeStamp = 3
	strikeClock = 40
	strikeAway  = 45 | 46<<8
	strikeCell  = 50 | 48<<8
)

// checkSecondStrikeBlock fails unless r's command block holds the planted
// blow record, stamp and alarm, with +0x94 the record key striker.
func checkSecondStrikeBlock(t *testing.T, name string, r *sav.DocumentRecordData, striker uint32) {
	t.Helper()
	b := secondRecordRaw(r, "URaw184")
	if len(b) != 184 {
		t.Fatalf("%s: the command block holds %d bytes", name, len(b))
	}
	le := binary.LittleEndian
	if le.Uint16(b[0x16:]) != strikeStamp || b[0x54] != 1 || le.Uint16(b[0x58:]) != strikeCell || b[0x5a] != 0 ||
		le.Uint32(b[0x94:]) != striker || le.Uint32(b[0x98:]) != strikeClock || le.Uint16(b[0x9c:]) != strikeAway {
		t.Fatalf("%s: +0x16 %d, +0x54 %d, +0x58 %#04x, +0x5a %d, +0x94 %#x, +0x98 %d, +0x9c %#04x; want %d, 1, %#04x, 0, %#x, %d, %#04x",
			name, le.Uint16(b[0x16:]), b[0x54], le.Uint16(b[0x58:]), b[0x5a], le.Uint32(b[0x94:]), le.Uint32(b[0x98:]), le.Uint16(b[0x9c:]),
			strikeStamp, strikeCell, striker, strikeClock, strikeAway)
	}
}

// checkSecondStrikeOrder fails unless knight 7's order in w holds the
// planted blow record, stamp and alarm, struck by knight 8.
func checkSecondStrikeOrder(t *testing.T, name string, w *sim.World) {
	t.Helper()
	ids := map[uint16]sim.EntityID{}
	for _, e := range w.Entities() {
		if e.MapUnitID == 7 || e.MapUnitID == 8 {
			ids[uint16(e.MapUnitID)] = e.ID
		}
	}
	_, orders, _ := w.SavedGroups()
	at := slices.IndexFunc(orders, func(o sim.SavedActorOrder) bool { return o.Entity == ids[7] })
	if len(ids) != 2 || at < 0 {
		t.Fatalf("%s: knight 7 holds no order", name)
	}
	o := orders[at]
	want := sim.SavedStrike{Attacker: ids[8], Clock: strikeClock, Away: strikeAway, Stamp: strikeStamp, Held: true, Stamped: true}
	if o.Strike != want || o.Raw[0x54] != 1 || binary.LittleEndian.Uint16(o.Raw[0x58:]) != strikeCell || o.Raw[0x5a] != 0 {
		t.Fatalf("%s: knight 7 holds %+v, alarm %d, notice %#04x/%d; want %+v, 1, %#04x/0",
			name, o.Strike, o.Raw[0x54], binary.LittleEndian.Uint16(o.Raw[0x58:]), o.Raw[0x5a], want, strikeCell)
	}
}

// A second-game command block holds a guard member's blow record, its
// pursuit stamp and its alarm (R2-ENGINE-845, R2-ENGINE-846, DIV-4503). A
// LOAD of the file's records alone, as of an original's file, reads them
// into the member's order; a SAVE writes them from it, +0x94 the striker's
// record key in the written file; the engine's LOAD of that file keeps them.
// Fails while LOAD reads only the command post and SAVE writes +0x16 as
// 65500 and +0x54..+0x9f as zero.
func TestSecondMissionCommandBlockKeepsTheBlowAndTheStamp(t *testing.T) {
	c := secondGuardMission()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	raw := secondPageSave(t, f, app)
	bareDoc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	bareDoc.State.ValueRecords = slices.DeleteFunc(slices.Clone(bareDoc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
		return strings.EqualFold(r.Path, sav.NativeFormPath)
	})
	striker, err := savedStructureValue(secondKnightRecord(t, &bareDoc, 8), "Identity")
	if err != nil || striker == 0 {
		t.Fatalf("knight 8's record key %#x: %v", striker, err)
	}
	b := secondRecordRaw(secondKnightRecord(t, &bareDoc, 7), "URaw184")
	le := binary.LittleEndian
	le.PutUint16(b[0x16:], strikeStamp)
	b[0x54] = 1
	le.PutUint16(b[0x58:], strikeCell)
	le.PutUint32(b[0x94:], striker)
	le.PutUint32(b[0x98:], strikeClock)
	le.PutUint16(b[0x9c:], strikeAway)
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
		t.Fatal("the file's records alone do not LOAD")
	}
	checkSecondStrikeOrder(t, "the records' LOAD", loaded.live.world)

	resaved := secondPageSave(t, loaded, loadedApp)
	doc := checkSecondMissionFile(t, resaved, 10)
	key, err := savedStructureValue(secondKnightRecord(t, &doc, 8), "Identity")
	if err != nil || key == 0 {
		t.Fatalf("the resave's knight 8 record key %#x: %v", key, err)
	}
	checkSecondStrikeBlock(t, "the resave", secondKnightRecord(t, &doc, 7), key)

	cold := secondMissionLoadBytesOf(t, c, resaved)
	if cold.live == nil {
		t.Fatal("the resave does not LOAD")
	}
	checkSecondStrikeOrder(t, "the engine's LOAD", cold.live.world)
}
