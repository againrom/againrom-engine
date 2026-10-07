package game

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func mageCity1101(t *testing.T, present, shared bool) []byte {
	t.Helper()
	d := cityfixture.City(false)
	for _, obj := range d.Objects {
		if u := obj.Unit; u != nil {
			u.Scalar1[3] = 4
			for i, v := range []uint16{30, 20, 45, 30, 16, 4, 4, 301, 50, 60, 100, 90, 100, 50} {
				binary.LittleEndian.PutUint16(u.Scalar2[2*i:], v)
			}
			binary.LittleEndian.PutUint16(u.RawA6[4:], 32)
			binary.LittleEndian.PutUint16(u.Raw114[4:], 29)
			binary.LittleEndian.PutUint16(u.RawD4[22:], 3)
			binary.LittleEndian.PutUint32(u.XP[4:], 2000)
			binary.LittleEndian.PutUint32(u.Scalar2[35:], 9000)
			u.RawA6[22], u.Raw114[23], u.RawD4[41], u.Raw154[179] = 0xa1, 0xb2, 0xc3, 0xd4
		}
	}
	if present {
		// Literal Spell body: id1, range99, raw Defensive2, mana -1,
		// identity 0x4567. These parameters deliberately disagree with table.
		d.Objects = append(d.Objects, sav.CityObjectData{Class: "Spell", Spell: &sav.CitySpellData{Fields: []byte{1, 99, 2, 255, 255, 0x67, 0x45, 0, 0}}})
		u := d.Objects[2].Unit
		u.SpellbookFlag, u.SpellbookCount, u.Spells = 1, 2, []uint16{4}
		if shared {
			u = d.Objects[1].Unit
			u.SpellbookFlag, u.SpellbookCount, u.Spells = 1, 2, []uint16{4}
		}
	}
	p, err := sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	u := sav.CityUpdate{Money: 20000}
	for _, c := range p.Roster() {
		u.Characters = append(u.Characters, originalCityBaselineUpdate(c))
	}
	raw, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func mageFront1101(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	rows := make(dbCollection, 30)
	for _, row := range []int{28, 29} {
		params := make([]int32, data.MinHumanRow)
		params[16], params[17], params[18], params[25] = 0x17, 4, 1, 0
		rows[row] = dbEntry{name: "PC_mage", params: params}
	}
	spell := make([]int32, 19)
	spell[1], spell[2], spell[4], spell[6], spell[8], spell[16], spell[17] = 7, 1, 1, 5, 1, 2, 2
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: &mapload.Table{Humans: rows, Spells: dbCollection{{}, {name: "test spell", params: spell}}}}}
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("mage city LOAD: %t %v", town, err)
	}
	return f
}

func TestMageCityTrainDerivesAllFieldsAndOrdinarySAVBook(t *testing.T) {
	source := mageCity1101(t, true, false)
	f := mageFront1101(t, source)
	initial, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := f.ExportCurrentSave(initial, "mage baseline")
	if err != nil {
		t.Fatal(err)
	}
	sourceFile, _ := sav.Open(source)
	sourceCity, _ := sourceFile.CityProvenance()
	baselineFile, _ := sav.Open(baseline)
	baselineCity, _ := baselineFile.CityProvenance()
	assertTrainingOpaqueFields(t, sourceCity.Data(), baselineCity.Data())
	before := trainingPartyMember(t, f, "hero")
	if before.Book.Slots[0] != (sim.BookSpell{Range: 99, Defensive: 2, ManaCost: 65535}) {
		t.Fatal("saved parameters lost")
	}
	if msg := train1099(t, f, "hero", 5); !strings.Contains(msg, "to 33 for 3172") {
		t.Fatal(msg)
	}
	m := trainingPartyMember(t, f, "hero")
	h, ok := m.OriginalHumanState()
	if !ok || h.Fighter || h.HealthMax != 56 || h.ManaMax != 112 || h.Health != 50 || h.Mana != 90 ||
		h.Experience != 23450 || h.SkillXP[1] != 16450 || h.Base.Skill[1] != 30 || h.Attack.Skill[1] != 33 ||
		h.Sight != 1689 || h.Attack.ToHit != 4 || h.Attack.DamageBase != 0 || h.Defence.Defence != 6 ||
		h.Defence.Protection[1] != 15 || h.MoverSpeed != 16 || h.Attack.Tail[0] != 0xa1 || h.Base.Tail[1] != 0xb2 {
		t.Fatalf("mage derive: %+v valid=%v", h, ok)
	}
	if m.Book.Slots[0] != (sim.BookSpell{Range: 6, Defensive: 2, ManaCost: 65535}) {
		t.Fatalf("trained book %+v", m.Book)
	}
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !strings.HasSuffix(name, ".sav") {
		t.Fatalf("SAVE %q %v", name, err)
	}
	raw, err := f.ExportOriginalSave(snapshot, "mage")
	if err != nil {
		t.Fatal(err)
	}
	assertMageFieldDiff1101(t, baseline, raw)
	file, _ := sav.Open(raw)
	p, _ := file.CityProvenance()
	for _, c := range p.Roster() {
		if c.Hero && (len(c.Spells) != 1 || c.Spells[0].Range != 6 || c.Spells[0].Defensive != 2 || c.Spells[0].ManaCost != 65535) {
			t.Fatalf("SAV instance %+v", c.Spells)
		}
	}
	fresh := mageFront1101(t, raw)
	got := trainingPartyMember(t, fresh, "hero")
	if got.Book != m.Book || got.Hero != m.Hero || got.Saved.Mana != 90 || got.Saved.MaxMana != 112 {
		t.Fatal("SAV continuation lost mage state")
	}
	encoded, err := EncodeSave(snapshot, "mage native")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, town, err := fresh.Restore(decoded); err != nil || !town {
		t.Fatalf("AGS LOAD %v", err)
	}
	if got := trainingPartyMember(t, fresh, "hero"); !reflect.DeepEqual(got, m) {
		t.Fatal("native mage continuation differs")
	}
}

func assertMageFieldDiff1101(t *testing.T, source, trained []byte) {
	t.Helper()
	f, _ := sav.Open(source)
	p, _ := f.CityProvenance()
	want := p.Data()
	for _, obj := range want.Objects {
		if u := obj.Unit; u != nil && u.Name == "Leader" {
			// Literal independent output words from the declared fixture,
			// not HumanState.Train or cityHumanUpdate's output.
			for off, value := range map[int]uint16{18: 56, 24: 112, 32: 1689} {
				binary.LittleEndian.PutUint16(u.Scalar2[off:], value)
			}
			// The saved aggregate is the sum of the six slot experiences.
			binary.LittleEndian.PutUint32(u.Scalar2[35:], 16450)
			binary.LittleEndian.PutUint32(u.XP[4:], 16450)
			binary.LittleEndian.PutUint16(u.RawA6, 4)
			binary.LittleEndian.PutUint16(u.RawA6[4:], 33)
			binary.LittleEndian.PutUint16(u.Raw114[4:], 30)
			binary.LittleEndian.PutUint16(u.RawBE, 6)
			for i := 1; i <= 5; i++ {
				binary.LittleEndian.PutUint16(u.RawBE[4+2*i:], 15)
			}
			u.Raw154[10] = 16
		}
		if obj.Spell != nil {
			obj.Spell.Fields[1] = 6
		}
		if obj.Player != nil {
			binary.LittleEndian.PutUint32(obj.Player.Fixed[21:], 16828^0x5c073f4d)
		}
	}
	g, _ := sav.Open(trained)
	q, _ := g.CityProvenance()
	if !reflect.DeepEqual(q.Data(), want) {
		t.Fatal("mage SAV changed fields outside the manual allowed-field diff")
	}
}

func TestMageCitySharedSpellTrainingCopiesOnlyChangedBook(t *testing.T) {
	f := mageFront1101(t, mageCity1101(t, true, true))
	before := mapload.CloneParty(f.Carried)
	graph := f.Town.cityObjects.Clone()
	if msg := train1099(t, f, "hero", 5); !strings.Contains(msg, "to 33 for 3172") {
		t.Fatal(msg)
	}
	if f.Town.Gold() != 16828 || trainingPartyMember(t, f, "hero").Book.Slots[0] != (sim.BookSpell{Range: 6, Defensive: 2, ManaCost: 65535}) {
		t.Fatal("training lost current price or saved spell fields")
	}
	wantGraph := graph.Clone()
	wantGraph.Spells = append(wantGraph.Spells, graph.NextID)
	wantGraph.NextID++
	for i := range wantGraph.Books {
		if string(wantGraph.Books[i].PartyID) == "hero" {
			wantGraph.Books[i].Slots[0] = graph.NextID
		}
	}
	for _, member := range before {
		if member.ID != "hero" && !reflect.DeepEqual(trainingPartyMember(t, f, member.ID), member) {
			t.Fatal("training changed another shared-book owner")
		}
	}
	if !reflect.DeepEqual(f.Town.cityObjects, wantGraph) {
		t.Fatal("training changed an unrelated identity or edge")
	}
	fresh := mageFront1101(t, cityProjectionSave(t, f))
	if !reflect.DeepEqual(cityObjectIdentityTopology(fresh.Town.cityObjects), cityObjectIdentityTopology(wantGraph)) ||
		trainingPartyMember(t, fresh, "hero").Book != trainingPartyMember(t, f, "hero").Book {
		t.Fatal("cold SAV LOAD lost independently trained book")
	}
	for id, want := range wantGraph.SpellRecords {
		if got, ok := fresh.Town.cityObjects.SpellRecords[id]; !ok || !reflect.DeepEqual(got, want) {
			t.Fatal("cold SAV LOAD changed an existing Spell supplement", id, got, want)
		}
	}
	id := graph.NextID
	wantSpell := cityMemberBook(trainingPartyMember(t, f, "hero"), f.Table)[0]
	if got, ok := fresh.Town.cityObjects.SpellRecords[id]; !ok || got.ID != id || got.This == 0 || got.Value != wantSpell {
		t.Fatal("cold SAV LOAD did not materialize the trained Spell record", got, wantSpell)
	}
}

func TestMageCityAbsentBookStaysAbsentThroughTrainingAndTeaching(t *testing.T) {
	f := mageFront1101(t, mageCity1101(t, false, false))
	if msg := train1099(t, f, "hero", 5); !strings.HasPrefix(msg, "trained ") {
		t.Fatal(msg)
	}
	m := trainingPartyMember(t, f, "hero")
	learnPartySpell(&m, 1, f.Table)
	if m.Book.State != sim.BookAbsent || m.KnownSpells != 0 {
		t.Fatal("first book was invented")
	}
	s, _, _ := f.Snapshot(false)
	raw, err := f.ExportOriginalSave(s, "absent")
	if err != nil {
		t.Fatal(err)
	}
	if got := trainingPartyMember(t, mageFront1101(t, raw), "hero"); got.Book.State != sim.BookAbsent || got.KnownSpells != 0 {
		t.Fatal("SAV invented book")
	}
}
