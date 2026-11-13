package sav

import (
	"bytes"
	"encoding/binary"
	"slices"
	"strings"
	"testing"
)

// engineTownDocument is an engine town document in the first game's
// records: one Player with its 32 preference bytes, one Human whose Order
// block is not zero, and an action leaf.
func engineTownDocument(t *testing.T) DocumentData {
	t.Helper()
	const identity, owner = uint32(0x20), uint32(0x10)
	fixed := make([]byte, 51)
	binary.LittleEndian.PutUint16(fixed[0:2], 1)
	binary.LittleEndian.PutUint32(fixed[2:6], 1)
	binary.LittleEndian.PutUint32(fixed[43:47], identity)
	binary.LittleEndian.PutUint32(fixed[47:51], owner)
	order := make([]byte, 148)
	order[7] = 9
	preferences := make([]byte, 32)
	preferences[0x1f] = 2
	unit := CityUnitData{
		Token: cityTestToken(identity, owner, 0), RawA6: make([]byte, 24), RawBE: make([]byte, 22),
		Raw114: make([]byte, 24), RawD4: make([]byte, 64), Raw154: make([]byte, 180), Raw158: order,
		Scalar1: make([]byte, 19), Name: "Hero", Scalar2: make([]byte, 55),
		ScalarTail: make([]byte, 17), XP: make([]byte, 24), Equipment: make([]uint16, 13),
	}
	city, err := CityFromData(CityData{
		Version: CityDataVersion, FileVersion: MinVersion, Head: [13]uint32{12: 2},
		PlayerList: 1, Players: []uint16{1}, Marker: GeneratedCityMarker,
		TrailerState: make([]byte, generatedCityTrailerLen),
		Objects: []CityObjectData{
			{Class: "Player", Player: &CityPlayerData{
				Name: "Player", Fixed: fixed, Raw32: preferences,
				Groups: []CityGroupData{{Raw80: make([]byte, 80), F44: owner, Actors: []uint16{2}}},
			}},
			{Class: "Human", Unit: &unit},
		},
		State: NewCityStateData("Hero"),
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := city.DocumentData()
	if err != nil {
		t.Fatal(err)
	}
	if err := SetNativeActions(&doc.State, []byte(`{"Version":1}`)); err != nil {
		t.Fatal(err)
	}
	return doc
}

func documentRaw(t *testing.T, r DocumentRecordData, name string) []byte {
	t.Helper()
	for _, v := range r.Raw {
		if v.Name == name {
			return v.Bytes
		}
	}
	t.Fatalf("%s record holds no %s", r.Class, name)
	return nil
}

// TestEngineTownDocumentWritesTheSecondGameForm: a town document written in
// the second game's profile is a Bsg& file in the second game's records
// that decodes and re-encodes byte for byte, and its AgainromForm leaf
// answers the engine's own Asg& bytes; the first game's profile leaves the
// document as the engine wrote it.
func TestEngineTownDocumentWritesTheSecondGameForm(t *testing.T) {
	doc := engineTownDocument(t)
	engine, err := EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if same, changed, err := DocumentInProfile(doc, FirstGame, ProfileKeeps{}); err != nil || changed || same.Game != "" {
		t.Fatalf("the first game's profile changed the document: %v %v", changed, err)
	}
	second, changed, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{})
	if err != nil || !changed {
		t.Fatalf("second form: %v %v", changed, err)
	}
	second.SecondTail = &SecondTail{Pairs: [][2]uint32{{1, 1}}, Locations: []SecondLocation{{ID: 1, Kind: 2}}}
	second.SecondTail.Bank[768] = 10
	file := profileFile(t, second, engine)
	if !bytes.HasPrefix(file, []byte("Bsg&")) {
		t.Fatalf("the file opens with %q", file[:4])
	}
	got, err := DecodeDocumentData(file)
	if err != nil {
		t.Fatal(err)
	}
	again, err := EncodeDocumentData(got)
	if err != nil || !bytes.Equal(again, file) {
		t.Fatalf("the Bsg& file does not re-encode byte for byte: %v", err)
	}
	if got.Game != "rom2" || got.Marker != 0xbadface1 || got.GlobalDWord != 0xbadface1 || got.SecondTail == nil || got.SecondTail.Bank[768] != 10 {
		t.Fatalf("game %q marker %#x global %#x tail %v", got.Game, got.Marker, got.GlobalDWord, got.SecondTail != nil)
	}
	if _, found, _ := got.State.ReadLeaf(NativeActionsPath); found {
		t.Fatal("the second-game records carry the engine's action leaf")
	}
	if v, found, err := got.State.ReadLeaf("/SpellBook/Shortcuts"); err != nil || !found || v.Kind != 6 || len(v.Bytes) != 0 {
		t.Fatalf("the town file's shortcut array: found %v kind %d, %d bytes, %v", found, v.Kind, len(v.Bytes), err)
	}
	if v, _, _ := doc.State.ReadLeaf("/SpellBook/Shortcuts"); len(v.Bytes) != 16 {
		t.Fatalf("the engine document's shortcut array changed: %d bytes", len(v.Bytes))
	}
	player, human := got.Objects[got.Players[0]-1], got.Objects[1]
	if p := documentRaw(t, player, "PRaw2560"); len(p) != 2560 || slices.ContainsFunc(p, func(b byte) bool { return b != 0 }) {
		t.Fatal("Player's 2,560 bytes are not zero")
	}
	if p := documentRaw(t, player, "PRaw36"); len(p) != 36 || p[0x1f] != 2 || p[0x20] != 0xff {
		t.Fatalf("Player's preference bytes % x", p)
	}
	if u := documentRaw(t, human, "URaw184"); len(u) != 184 || u16(u, 0x16) != 65500 || slices.ContainsFunc(slices.Concat(u[:0x16], u[0x18:]), func(b byte) bool { return b != 0 }) {
		t.Fatal("Unit's sixth raw block is not the command word 65500 at +0x16 and zero bytes")
	}
	if len(player.Inline) != 0 || slices.ContainsFunc(human.RefSlots, func(v DocumentRefsData) bool { return v.Name == "Diary" }) {
		t.Fatal("a Diary site survived")
	}
	back, err := EngineDocument(file)
	if err != nil || !bytes.Equal(back, engine) {
		t.Fatalf("the leaf does not answer the engine document: %v", err)
	}
	if back, err := EngineDocument(engine); err != nil || !bytes.Equal(back, engine) {
		t.Fatalf("an Asg& document is not answered as itself: %v", err)
	}
	for name, raw := range secondWriterFixtures(t) {
		if back, err := EngineDocument(raw); err != nil || !bytes.Equal(back, raw) {
			t.Fatalf("%s: a Bsg& file without the leaf is not answered as itself: %v", name, err)
		}
	}
}

func TestEngineTownDocumentRefusals(t *testing.T) {
	doc := engineTownDocument(t)
	engine, err := EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{})
	if err != nil {
		t.Fatal(err)
	}
	second.SecondTail = &SecondTail{Pairs: [][2]uint32{{1, 1}}, Locations: []SecondLocation{{ID: 1, Kind: 2}}}
	file := profileFile(t, second, engine)
	if _, err := StoreEngineDocument(second, file); err == nil {
		t.Error("a Bsg& file was carried as the engine document")
	}
	if _, err := StoreEngineDocument(doc, engine); err == nil {
		t.Error("an engine document was written as a profile document")
	}
	if _, _, err := DocumentInProfile(second, SecondGame, ProfileKeeps{}); err == nil {
		t.Error("a second-game document was converted again")
	}
	// A leaf holding anything but an Asg& document refuses the LOAD.
	nested := second
	nested.State = second.State
	if err := nativeFormLeaf.storePayload(&nested.State, file); err != nil {
		t.Fatal(err)
	}
	bad, err := EncodeDocumentData(nested)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EngineDocument(bad); err == nil {
		t.Error("a leaf holding a Bsg& file was answered")
	}
}

// A mission document's world half is written in the second game's widths:
// zero past the first game's cell fields, result slots and 50x50 stances,
// and no first-game trigger tail in a cell.
func TestEngineMissionDocumentInSecondGameWidths(t *testing.T) {
	world := documentDataFixture(t)
	if world.World == nil || len(world.World.Cells) == 0 {
		t.Fatal("the fixture has no world half")
	}
	world.World.Cells[0].Power = 7
	out, ok, err := DocumentInProfile(world, SecondGame, ProfileKeeps{})
	if err != nil || !ok || out.World == nil {
		t.Fatalf("a world document: %v %v", ok, err)
	}
	w := out.World
	if len(w.CellRest) != SecondGame.cellRest()*len(w.Cells) || len(w.SessionResults) != SecondGame.wideResults() ||
		len(w.SessionDiplomacy) != SecondGame.wideDiplomacy() {
		t.Fatalf("cell rest %d, results %d, diplomacy %d", len(w.CellRest), len(w.SessionResults), len(w.SessionDiplomacy))
	}
	if w.Cells[0].Power != 0 || world.World.Cells[0].Power != 7 {
		t.Fatal("the cell trigger tail was carried, or the input changed")
	}
}

// A mission document's terrain and session follow the second game's
// serializer: the key list holds only the window's cells above 0x0f, the
// entries go bucket by bucket with their own key at +0x3a and their ground
// and air occupants again at +0x2c and +0x30, result slot 901 is 1 and the
// diplomacy block opens 1 1 0 0.
func TestSecondMissionWorldFollowsTheTerrainAndSessionRules(t *testing.T) {
	doc := documentDataFixture(t)
	if doc.World == nil {
		t.Fatal("the fixture has no world half")
	}
	doc.World.Blocks = []BlockRecord{{Cell: 0x0005, Dyn: 0x1f, Static: 0x1f}, {Cell: 0x0900, Dyn: 0x07, Static: 0x07}, {Cell: 0x0901, Dyn: 0x1f, Static: 0x05}}
	doc.World.Cells = []DocumentCellData{{Cell: 0x0630, Cost: 6, GroundActor: 0x51000010}, {Cell: 0x0640, Cost: 7, AirActor: 0x51000020}}
	doc.World.CellRest = nil
	out, ok, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{})
	if err != nil || !ok {
		t.Fatalf("a world document: %v %v", ok, err)
	}
	w := out.World
	if want := []BlockRecord{{Cell: 0x0901, Dyn: 0x1f, Static: 0x05}}; !slices.Equal(w.Blocks, want) {
		t.Errorf("key list %v, want %v", w.Blocks, want)
	}
	if len(w.Cells) != 2 || w.Cells[0].Cell != 0x0640 || w.Cells[1].Cell != 0x0630 || w.Cells[0].Cost != 7 {
		t.Errorf("entries %+v, want bucket 0 (0x640) before bucket 99 (0x630)", w.Cells)
	}
	rest := SecondGame.cellRest()
	for i, c := range w.Cells {
		if got := uint16(w.CellRest[i*rest+14]) | uint16(w.CellRest[i*rest+15])<<8; got != c.Cell {
			t.Errorf("entry %#x holds key %#x at +0x3a", c.Cell, got)
		}
		if ground, air := binary.LittleEndian.Uint32(w.CellRest[i*rest:]), binary.LittleEndian.Uint32(w.CellRest[i*rest+4:]); ground != c.GroundActor || air != c.AirActor || ground+air == 0 {
			t.Errorf("entry %#x holds footprint occupants %#x and %#x, occupants %#x and %#x", c.Cell, ground, air, c.GroundActor, c.AirActor)
		}
	}
	if w.SessionResults[901-100] != 1 || !bytes.Equal(w.Session.DiplomacyHeader[:4], []byte{1, 1, 0, 0}) {
		t.Errorf("result 901 %d, diplomacy head % x", w.SessionResults[801], w.Session.DiplomacyHeader)
	}
	if len(doc.World.Blocks) != 3 || doc.World.Cells[0].Cell != 0x0630 {
		t.Error("the input document changed")
	}
}

// A mission document's session matrix holds bit 0x10 of M[owner][viewer]
// for each pair of the document's Player slots in which the World's fog
// lets viewer see by owner's bodies (R2-ENGINE-826); a slot no Player holds
// and every other bit keep the engine's bytes, and no fog writes no bit.
func TestSecondMissionSessionHoldsTheWorldsSight(t *testing.T) {
	sees := func(viewer, owner uint32) bool { return viewer == owner || viewer == 2 && owner == 3 }
	stances := func(m *[50][50]uint8) {
		m[2][2], m[3][3], m[3][2], m[2][3], m[5][5] = 2, 2, 1, 2, 2
	}
	diff := func(name string, got, want [50][50]uint8) {
		for a := range got {
			for b := range got[a] {
				if got[a][b] != want[a][b] {
					t.Errorf("%s: M[%d][%d] is %#02x, want %#02x", name, a, b, got[a][b], want[a][b])
				}
			}
		}
	}

	// The fixture's one Player record, listed twice, at slot 2.
	doc := documentDataFixture(t)
	player := slices.IndexFunc(doc.Objects, func(r DocumentRecordData) bool { return r.Class == "Player" })
	if doc.World == nil || player < 0 {
		t.Fatal("the fixture has no world half or no Player record")
	}
	at := slices.IndexFunc(doc.Objects[player].Values, func(v DocumentValueData) bool { return v.Name == "Slot" })
	if at < 0 {
		t.Fatal("the Player record holds no Slot")
	}
	doc.Objects[player].Values[at].Value = 2
	stances(&doc.World.Session.Diplomacy)
	before := doc.World.Session.Diplomacy
	out, _, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{Sees: sees})
	if err != nil {
		t.Fatal(err)
	}
	want := before
	want[2][2] = 0x12
	diff("one Player", out.World.Session.Diplomacy, want)
	if doc.World.Session.Diplomacy != before {
		t.Error("the input document changed")
	}
	blind, _, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{})
	if err != nil {
		t.Fatal(err)
	}
	diff("no fog", blind.World.Session.Diplomacy, before)

	// Two Players, slots 2 and 3; slot 2 sees by slot 3's bodies.
	two := DocumentData{World: &DocumentWorldData{}, Players: []uint16{0, 1, 2},
		Objects: []DocumentRecordData{{Class: "Player", Values: []DocumentValueData{{Name: "Slot", Value: 2}}},
			{Class: "Player", Values: []DocumentValueData{{Name: "Slot", Value: 3}}}}}
	stances(&two.World.Session.Diplomacy)
	want = two.World.Session.Diplomacy
	want[2][2], want[3][3], want[3][2] = 0x12, 0x12, 0x11
	secondForm.sight(&two, sees)
	diff("two Players", two.World.Session.Diplomacy, want)
	firstGame := two.World.Session.Diplomacy
	(&documentForm{}).sight(&two, sees)
	diff("a form with no sight bit", two.World.Session.Diplomacy, firstGame)
}

// A town file whose engine document leaf is present but does not validate
// refuses; only a file with no value record at the leaf's path is read as
// the original's.
func TestEngineDocumentRefusesALeafThatDoesNotValidate(t *testing.T) {
	doc := engineTownDocument(t)
	engine, err := EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{})
	if err != nil {
		t.Fatal(err)
	}
	second.SecondTail = &SecondTail{Pairs: [][2]uint32{{1, 1}}, Locations: []SecondLocation{{ID: 1, Kind: 2}}}
	records, err := EncodeDocumentData(second)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := EngineDocument(records); err != nil || !bytes.Equal(got, records) {
		t.Fatalf("a file without the leaf: %v, unchanged %v", err, bytes.Equal(got, records))
	}
	file := profileFile(t, second, engine)
	at := bytes.Index(file, engine)
	if at < 8 || binary.LittleEndian.Uint32(file[at-8:]) != 1 {
		t.Fatalf("the leaf's framing is not found before the engine document (%d)", at)
	}
	damaged := bytes.Clone(file)
	binary.LittleEndian.PutUint32(damaged[at-8:], 7)
	if _, err := DecodeDocumentData(damaged); err == nil {
		t.Fatal("the damaged framing decoded")
	}
	got, err := EngineDocument(damaged)
	if err == nil {
		t.Fatalf("a damaged leaf was read as the original's file (%d bytes)", len(got))
	}
	if !strings.Contains(err.Error(), "engine document") || len(err.Error()) > 512 {
		t.Errorf("refusal: %v", err)
	}
}

// profileFile is the file a town SAVE writes: the profile document with the
// engine document in its leaf, encoded.
func profileFile(t *testing.T, data DocumentData, engine []byte) []byte {
	t.Helper()
	stored, err := StoreEngineDocument(data, engine)
	if err != nil {
		t.Fatal(err)
	}
	file, err := EncodeDocumentData(stored)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

// A Pointer the World keeps names a Building record of the document; a key
// no Building record holds refuses the form, and the first game's records,
// which have no form, are returned unchanged.
func TestEngineDocumentPointerNamesABuildingRecord(t *testing.T) {
	doc := documentDataFixture(t)
	keeps := ProfileKeeps{Pointers: map[uint32]uint32{0x7fffff01: 5}}
	if _, _, err := DocumentInProfile(doc, SecondGame, keeps); err == nil {
		t.Fatal("a Pointer naming no Building record was written")
	}
	if _, changed, err := DocumentInProfile(doc, FirstGame, keeps); err != nil || changed {
		t.Fatalf("the first game's records: changed %v: %v", changed, err)
	}
}
