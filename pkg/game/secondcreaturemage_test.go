package game

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The mage fixture's creatures, by placement: a necromancer, a bat, a
// creature with no spell cell and one with mana and no mask, as Urd.
const (
	secondMageNecro = 21 + iota
	secondMageBat
	secondMagePlain
	secondMageUrd
)

// secondMageSpells is the necromancer row's knownSpells mask: spells 1, 26
// and 27, as Necro_Male1's book holds them (R2-SESSION-350).
const secondMageSpells = uint32(1)<<1 | uint32(1)<<26 | uint32(1)<<27

// secondMageDrainLife is the staff's spell, Drain Life.
const secondMageDrainLife = 26

// secondMageMission is the guard mission with four creatures of Player 2.
// The necromancer's Units row holds secondMageSpells, mana 150 and a staff
// casting Drain Life; the bat's Spell 1 14; the plain creature's no spell
// cell; the Urd-like row mana 10000 and the mask -1.
func secondMageMission() secondMissionContent {
	c := secondGuardMission()
	row := func(name string, typeID, server int32, set func(p []int32)) dbEntry {
		p := make([]int32, 62)
		for i := range p {
			p[i] = -1
		}
		p[4], p[29], p[54], p[55] = 200, typeID, 30, server
		set(p)
		return dbEntry{name: name, params: p}
	}
	necro := row("Necro_Male1", 99, 9101, func(p []int32) {
		p[6], p[56] = 150, int32(secondMageSpells)
		copy(p[57:], []int32{10, 10, 10, 10, 10})
	})
	necro.strings = []string{"Staff {castSpell=Drain_Life:10}"}
	c.creatures = map[int]dbEntry{
		2: necro,
		3: row("Bat_Sonic.4", 70, 9102, func(p []int32) { p[48], p[49] = 14, 10 }),
		4: row("Squirrel", 76, 9103, func([]int32) {}),
		5: row("Urd", 72, 9104, func(p []int32) { p[6] = 10000 }),
	}
	for i, server := range []uint32{9101, 9102, 9103, 9104} {
		c.units = append(c.units, alm.Unit{X: uint32(56+2*i)<<8 | 0x80, Y: 48<<8 | 0x80, ClassID: 9, ClassSubID: 1,
			ServerID: server, Owner: 2, UnitID: uint16(secondMageNecro + i), GroupID: 6})
	}
	spell := make([]int32, 19)
	spell[1], spell[4], spell[6], spell[8], spell[16], spell[17], spell[18] = 1, 1, 7, 1, 1, 1, 1
	c.spells = map[int]dbEntry{secondMageDrainLife: {name: "Drain Life", params: spell}}
	return c
}

// secondMageItems gives f's table the staff the necromancer's row names.
func secondMageItems(f *FrontEnd) {
	weapon := chargenWeaponParams(data.SkillBlade)
	weapon[6], weapon[7], weapon[11], weapon[15] = 0, 0, 5, 2
	f.Table.Shapes, f.Table.Materials = emptyScale{}, emptyScale{}
	f.Table.Weapons = dbCollection{{}, {name: "Staff", params: weapon}}
}

// secondMageLoad LOADs raw into a fresh front of the mage fixture.
func secondMageLoad(t *testing.T, raw []byte) (*FrontEnd, *ui.App) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := secondMissionFrontOf(t, secondMageMission())
	secondMageItems(f)
	app := f.App("second-mission")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0001.sav")
	if f.live == nil {
		t.Fatal("the file does not LOAD")
	}
	return f, app
}

// secondMageEntity is the World's creature of placement id.
func secondMageEntity(t *testing.T, w *sim.World, id uint16) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.MapUnitID == id {
			return e
		}
	}
	t.Fatalf("no entity of placement %d", id)
	return sim.Entity{}
}

// secondMageFlags is each Unit record's U4C in doc by its placement.
func secondMageFlags(t *testing.T, doc *sav.DocumentData) map[uint32]uint32 {
	t.Helper()
	out := map[uint32]uint32{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" {
			continue
		}
		id, _ := savedStructureValue(r, "T08")
		flags, err := savedStructureValue(r, "U4C")
		if err != nil {
			t.Fatal(err)
		}
		out[id] = flags
	}
	return out
}

// setSecondMageFlags writes flags into the Unit records of doc by placement.
func setSecondMageFlags(t *testing.T, doc *sav.DocumentData, flags map[uint32]uint32) {
	t.Helper()
	for i := range doc.Objects {
		r := &doc.Objects[i]
		id, _ := savedStructureValue(r, "T08")
		if v, ok := flags[id]; ok && r.Class == "Unit" {
			if err := savedStructureSetValue(r, "U4C", v); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// withoutMageHolder runs save with the second game's mission form holding
// no U4C holder: the SAVE as it stood before the mage bit was written.
func withoutMageHolder(t *testing.T, save func() []byte) []byte {
	t.Helper()
	form := secondSavedForms[missionRecord]
	defer func() { secondSavedForms[missionRecord] = form }()
	bare := form
	bare.holders = slices.DeleteFunc(slices.Clone(form.holders), func(h savedFormHolder) bool { return h.field == "U4C" })
	if len(bare.holders) != len(form.holders)-1 {
		t.Fatal("the mission form holds no U4C holder")
	}
	secondSavedForms[missionRecord] = bare
	return save()
}

// checkSecondMageCreature: the necromancer holds the mask's book, the
// staff's Drain Life and mana 150.
func checkSecondMageCreature(t *testing.T, name string, w *sim.World) sim.Entity {
	t.Helper()
	e := secondMageEntity(t, w, secondMageNecro)
	if e.KnownSpells != secondMageSpells || e.Book.State != sim.BookPresent || e.WeaponSpell != secondMageDrainLife || e.MaxMana != 150 {
		t.Fatalf("%s: the necromancer knows %#x in book state %d, weapon spell %d, mana %d", name, e.KnownSpells, e.Book.State, e.WeaponSpell, e.MaxMana)
	}
	return e
}

// A second-game creature whose Units row holds a knownSpells mask is saved
// with the mage bit, 4, of +0x4c in the records the original reads
// (R2-ENGINE-652, R2-ENGINE-900, R2-SESSION-350): the necromancer's other
// bits, its staff's Weapon Spell and its book stay as the SAVE wrote them
// without the bit, and so does every other byte of the file, the engine
// document included. The World keeps no mage bit, so the engine's LOAD
// restores the World the SAVE left. A bat, a creature with no spell cell
// and one with mana and no mask keep their byte. Fails while the SAVE
// writes the constructor's 2.
func TestSecondMaskCreatureSavesTheMageBit(t *testing.T) {
	c := secondMageMission()
	f, app, screen, _ := secondBagFrontOf(t, nil, c)
	secondMageItems(f)
	secondBagEnter(t, f, app, screen)
	e := checkSecondMageCreature(t, "mission start", f.live.world)
	if e.SourceBinding.ClassFlags&4 != 0 {
		t.Fatalf("the World's necromancer holds class flags %#x", e.SourceBinding.ClassFlags)
	}
	hash := f.live.world.Hash()
	raw := secondPageSave(t, f, app)
	control := withoutMageHolder(t, func() []byte { return secondPageSave(t, f, app) })
	if f.live.world.Hash() != hash {
		t.Fatal("the SAVE changed the World")
	}

	got := checkSecondMissionFile(t, raw, 10)
	want := checkSecondMissionFile(t, control, 10)
	flags, base := secondMageFlags(t, &got), secondMageFlags(t, &want)
	if len(base) < 4 {
		t.Fatalf("the file holds Unit records %v, want the four creatures", base)
	}
	for id, v := range base {
		expect := v
		if id == secondMageNecro {
			expect |= 4
		}
		if flags[id] != expect {
			t.Errorf("placement %d writes U4C %#x, want %#x", id, flags[id], expect)
		}
	}
	if base[secondMageNecro]&4 != 0 {
		t.Fatalf("without the holder the necromancer writes %#x", base[secondMageNecro])
	}
	setSecondMageFlags(t, &got, base)
	gotBytes, err := sav.EncodeDocumentData(got)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := sav.EncodeDocumentData(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Fatal("bytes outside the necromancer's U4C changed")
	}
	if engine := secondMageFlags(t, ptrTo(engineSaveDocument(t, raw))); engine[secondMageNecro]&4 != 0 {
		t.Fatalf("the engine document's necromancer holds %#x", engine[secondMageNecro])
	}

	loaded, _ := secondMageLoad(t, raw)
	checkSecondMageCreature(t, "LOAD", loaded.live.world)
	cold, _ := secondMageLoad(t, control)
	if loaded.live.world.Hash() != cold.live.world.Hash() {
		t.Fatal("the engine's LOAD of the file differs from the LOAD without the bit")
	}
	bare, _ := secondMageLoad(t, secondRecordsAlone(t, raw))
	if e := checkSecondMageCreature(t, "LOAD of the records", bare.live.world); e.SourceBinding.ClassFlags&4 == 0 {
		t.Fatalf("the records' necromancer loads class flags %#x", e.SourceBinding.ClassFlags)
	}
}

func ptrTo[T any](v T) *T { return &v }

// A SAVE of the second game's records LOADed keeps each creature's loaded
// U4C, the mage bit added only to the creature whose row holds a mask: a
// necromancer this engine wrote with 2 is written with 6, one the original
// wrote with 6 keeps 6, and a bat, a plain creature and the Urd-like row keep
// the byte they loaded with.
func TestSecondLoadedCreatureKeepsItsClassByte(t *testing.T) {
	c := secondMageMission()
	f, app, screen, _ := secondBagFrontOf(t, nil, c)
	secondMageItems(f)
	secondBagEnter(t, f, app, screen)
	raw := secondRecordsAlone(t, secondPageSave(t, f, app))
	for _, tc := range []struct {
		name          string
		planted, want map[uint32]uint32
	}{
		{"engine-written bytes",
			map[uint32]uint32{secondMageNecro: 2, secondMageBat: 18, secondMagePlain: 0, secondMageUrd: 2},
			map[uint32]uint32{secondMageNecro: 6, secondMageBat: 18, secondMagePlain: 0, secondMageUrd: 2}},
		{"original-written bytes",
			map[uint32]uint32{secondMageNecro: 6, secondMageBat: 2, secondMagePlain: 16, secondMageUrd: 6},
			map[uint32]uint32{secondMageNecro: 6, secondMageBat: 2, secondMagePlain: 16, secondMageUrd: 6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			setSecondMageFlags(t, &doc, tc.planted)
			planted, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			loaded, loadedApp := secondMageLoad(t, planted)
			out := checkSecondMissionFile(t, secondPageSave(t, loaded, loadedApp), 10)
			got := secondMageFlags(t, &out)
			for id, v := range tc.want {
				if got[id] != v {
					t.Errorf("placement %d resaves U4C %#x, want %#x", id, got[id], v)
				}
			}
		})
	}
}

// The first game's SAVE takes no mage bit from a Units row's cell 56: its
// book source has no knownSpells cell, so the file is byte for byte the
// one its table without the cell writes.
func TestFirstGameSaveTakesNoMageBitFromTheUnitsRow(t *testing.T) {
	c := savedFormCases(t)[1]
	if c.name != "first game mission" {
		t.Fatalf("case %q", c.name)
	}
	want, err := c.f.ExportCurrentSave(c.s, c.label)
	if err != nil {
		t.Fatal(err)
	}
	source, ok := c.f.Table.Units.(dbCollection)
	if !ok {
		t.Fatalf("the first game's Units are %T", c.f.Table.Units)
	}
	units := slices.Clone(source)
	for i := range units {
		p := append(slices.Clone(units[i].params), make([]int32, max(0, 62-len(units[i].params)))...)
		p[56] = int32(secondMageSpells)
		units[i].params = p
	}
	table := *c.f.Table
	table.Units = units
	c.f.Table = &table
	if table.UnitMaskMage(1) {
		t.Fatal("the first game's row answers a mask")
	}
	got, err := c.f.ExportCurrentSave(c.s, c.label)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("a cell-56 value changed the first game's SAVE")
	}
	doc := engineSaveDocument(t, got)
	if len(secondMageFlags(t, &doc)) == 0 {
		t.Fatal("the first game's file holds no Unit record")
	}
}

// UnitMaskMage answers the second game's knownSpells mask: a row whose
// mask is neither 0 nor -1, and no other row.
func TestUnitMaskMageReadsTheKnownSpellsCell(t *testing.T) {
	cell := func(v int32, n int) dbEntry {
		p := make([]int32, n)
		for i := range p {
			p[i] = -1
		}
		if n > 56 {
			p[56] = v
		}
		return dbEntry{name: "row", params: p}
	}
	table := &mapload.Table{Units: dbCollection{cell(6, 62), cell(-1, 62), cell(0, 62), cell(6, 56)}, CreatureBooks: mapload.ClassCellBooks}
	for row, want := range []bool{true, false, false, false, false} {
		if got := table.UnitMaskMage(row); got != want {
			t.Errorf("row %d answers %v, want %v", row, got, want)
		}
	}
	if table.UnitMaskMage(-1) {
		t.Error("row -1 answers a mask")
	}
	table.CreatureBooks = nil
	if table.UnitMaskMage(0) {
		t.Error("the first game's book source answers a mask")
	}
}
