package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// signpostPlacement is one Pointer of a fixture map: its map id, kind and
// cell, the action node its type-11c record names, and the slot that node
// gives. A placement with noRecord has no type-11c record and slot 0.
type signpostPlacement struct {
	id             uint16
	kind           uint32
	col, row       uint32
	position, slot uint32
	noRecord       bool
}

// word is the +0x6c word the placement's Pointer record holds: its slot, or
// -1, the constructor's value, which the type-4 pass keeps when no record
// names the Pointer (R2-ENGINE-512, R2-ENGINE-517).
func (p signpostPlacement) word() uint32 {
	if p.noRecord {
		return 0xffffffff
	}
	return p.slot
}

// signpostMission is mission n over a 128x128 map holding signs. Node 0 is a
// drop location (operation 0x10002), which no slot counts, as in 10.alm
// (R2-ASSET-152); every later node raises event 7.
func signpostMission(n int, signs []signpostPlacement) secondMissionContent {
	c := secondMissionContent{mission: n, side: 128}
	last := uint32(0)
	for _, s := range signs {
		c.objects = append(c.objects, alm.Object{X: s.col<<8 | 0x80, Y: s.row<<8 | 0x80, Kind: s.kind, Field0C: 1, Field12: s.id})
		if !s.noRecord {
			c.pointers = append(c.pointers, alm.StructureAction{ID: uint32(s.id), Gate: 1, Position: s.position})
		}
		last = max(last, s.position)
	}
	for i := uint32(0); i <= last; i++ {
		node := alm.ScriptNode{Opcode: 0x10002, ID: i + 1}
		if i != 0 {
			node.Opcode = 2
			node.Type[0], node.Value[0] = 1, 7
		}
		c.actions = append(c.actions, node)
	}
	return c
}

// The first mission's signpost: 10.alm id 42, kind 0x4a at cell 14/40, its
// record naming node 25, slot 24 (R2-ASSET-152).
func firstMissionSignposts() []signpostPlacement {
	return []signpostPlacement{{id: 42, kind: 0x4a, col: 14, row: 40, position: 25, slot: 24}}
}

// The three signposts of mission 20, ids 87, 88 and 89 (R2-ASSET-123).
func twentiethMissionSignposts() []signpostPlacement {
	return []signpostPlacement{
		{id: 87, kind: 0x4a, col: 37, row: 55, position: 23, slot: 22},
		{id: 88, kind: 0x48, col: 35, row: 55, position: 24, slot: 23},
		{id: 89, kind: 0x48, col: 47, row: 26, position: 25, slot: 24},
	}
}

// checkSignpostWorld: w holds each sign as a Pointer with its slot.
func checkSignpostWorld(t *testing.T, name string, w *sim.World, signs []signpostPlacement) {
	t.Helper()
	found := 0
	for _, s := range w.Structures() {
		if !sim.PointerKind(s.Kind) {
			continue
		}
		found++
		for _, p := range signs {
			if uint32(s.Col) == p.col && uint32(s.Row) == p.row && (s.UseAmount != int32(p.slot) || uint32(s.Kind) != p.kind) {
				t.Fatalf("%s: the Pointer at %d/%d is kind %#x slot %d, want kind %#x slot %d", name, p.col, p.row, s.Kind, s.UseAmount, p.kind, p.slot)
			}
		}
	}
	if found != len(signs) {
		t.Fatalf("%s: the World holds %d Pointers, want %d", name, found, len(signs))
	}
}

// checkSignpostFile: raw is a second-game mission file whose Buildings hold
// each sign as a Pointer record whose +0x6c word is its slot, and no other
// record of a Pointer kind.
func checkSignpostFile(t *testing.T, name string, raw []byte, mission int, signs []signpostPlacement) {
	t.Helper()
	checkSecondMissionFile(t, raw, mission)
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	buildings, present, err := f.Buildings()
	if err != nil || !present {
		t.Fatalf("%s: Buildings present %v: %v", name, present, err)
	}
	found := 0
	for _, b := range buildings {
		if !sim.PointerKind(uint16(b.Kind)) {
			continue
		}
		found++
		ok := false
		for _, p := range signs {
			if b.AuthoredID == uint32(p.id) {
				ok = b.Class == "Pointer" && b.Pointer6C == p.word() && uint32(b.Kind) == p.kind && uint32(b.Col) == p.col && uint32(b.Row) == p.row
			}
		}
		if !ok {
			t.Fatalf("%s: map id %d is a %s of kind %#x at %d/%d with +0x6c %#x", name, b.AuthoredID, b.Class, b.Kind, b.Col, b.Row, b.Pointer6C)
		}
	}
	if found != len(signs) {
		t.Fatalf("%s: the file holds %d records of a Pointer kind, want %d", name, found, len(signs))
	}
}

// A second-game mission started from its map writes each signpost as a
// Pointer record with its action slot as the +0x6c word (R2-ENGINE-517,
// R2-ENGINE-603): the start autosave and the save page's file. The file's
// LOAD, and the LOAD of its records alone, hold each Pointer with its slot,
// and their SAVE writes the same records. Mission 10's sign has slot 24
// (R2-ASSET-152); mission 20's three have 22, 23 and 24 (R2-ASSET-123). A
// planted Pointer no record names writes -1. Fails while a World started
// from a map writes each signpost as a Building.
func TestSecondMissionStartWritesEachSignpostAsAPointerWithItsSlot(t *testing.T) {
	unnamed := append(firstMissionSignposts(), signpostPlacement{id: 43, kind: 0x46, col: 30, row: 30, noRecord: true})
	for _, tc := range []struct {
		mission int
		signs   []signpostPlacement
	}{{10, firstMissionSignposts()}, {20, twentiethMissionSignposts()}, {10, unnamed}} {
		c := signpostMission(tc.mission, tc.signs)
		f, app, screen, _ := secondBagFrontOf(t, nil, c)
		f.Town.second.available = []partyPlace{{kind: placeMission, id: tc.mission}}
		dir := t.TempDir()
		f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
		f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
		destinationTownCampaign{screen}.leave()
		if err := app.OpenMission(f.MissionOpener(tc.mission)); err != nil || f.live == nil {
			t.Fatalf("mission %d: %v %s", tc.mission, err, app.HeadlessMessage())
		}
		checkSignpostWorld(t, "the start", f.live.world, tc.signs)
		prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
			OnMap: true, Directory: dir, Name: "page", Label: "page", Format: ui.SaveSAV})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := prepared.Commit(true); err != nil {
			t.Fatal(err)
		}
		var page []byte
		for _, name := range []string{"game0000.sav", "page.sav"} {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("mission %d: %v", tc.mission, err)
			}
			checkSignpostFile(t, name, raw, tc.mission, tc.signs)
			page = raw
		}
		for _, load := range []struct {
			name string
			raw  []byte
		}{{"the LOAD", page}, {"the records' LOAD", withoutEngineDocument(t, page)}} {
			cold, coldApp := signpostLoad(t, c, load.raw)
			checkSignpostWorld(t, load.name, cold.live.world, tc.signs)
			checkSignpostFile(t, load.name+"'s SAVE", secondPageSave(t, cold, coldApp), tc.mission, tc.signs)
		}
	}
}

// signpostLoad LOADs raw over the mission c.
func signpostLoad(t *testing.T, c secondMissionContent, raw []byte) (*FrontEnd, *ui.App) {
	t.Helper()
	f, app := secondMaskLoad(t, c, raw)
	if f.live == nil || f.live.world == nil {
		t.Fatalf("the file does not LOAD: %s", app.HeadlessMessage())
	}
	return f, app
}

// A LOAD of a file holding the signpost as a Building of a Pointer kind, as
// this engine wrote it before it wrote the Pointer, gives the World the
// Pointer with the slot its map placement derives, and its SAVE writes the
// Pointer record with the slot as its word. Fails while the SAVE writes the
// Building back.
func TestSecondMissionLoadOfASignpostBuildingWritesAPointer(t *testing.T) {
	p := pointerSaveRecord()
	p.building, p.word = true, 0
	f, raw := secondBuildingMission(t, p)
	if sign := pointerSaveSign(t, f.live.world); sign.UseAmount != 1 || !sign.Usable() {
		t.Fatalf("the LOAD holds %+v, want the Pointer with slot 1", sign)
	}
	check := func(name string, raw []byte) {
		t.Helper()
		r := secondFileStructure(t, name, raw)
		if word, ok := secondRecordValue(r, "P6C"); r.Class != "Pointer" || !ok || word != 1 {
			t.Fatalf("%s writes a %s with word %#x (%v), want a Pointer with 1", name, r.Class, word, ok)
		}
	}
	check("the SAVE", raw)
	cold, coldApp := signpostLoad(t, pointerSaveMission(), raw)
	if sign := pointerSaveSign(t, cold.live.world); sign.UseAmount != 1 {
		t.Fatalf("the SAVE's LOAD holds %+v, want slot 1", sign)
	}
	check("the resave", secondPageSave(t, cold, coldApp))
}
