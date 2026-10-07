package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestQuickSpellsCapabilityUsesLiveClientClassWithoutArt(t *testing.T) {
	var entities []sim.Entity
	for i := 1; i <= 5; i++ {
		entities = append(entities, sim.Entity{ID: sim.EntityID(i), X: int32(i), Y: 1, HP: 10, MaxHP: 10, TypeID: sim.HumanTypeID, Class: 1, KnownSpells: 1 << 1})
	}
	stock := []sim.Stock{{ID: 2, Equipped: [sim.EquipSlots]uint16{2}}, {ID: 4, Equipped: [sim.EquipSlots]uint16{2}}}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, sim.Terrain{}, entities, nil, sim.Relations{}, nil, stock)
	if err != nil {
		t.Fatal(err)
	}
	mw := &mapWorld{world: w, mission: &missionNotices{list: data.BodyList{"unarmed", "swordsman"},
		ids: []sim.EntityID{1, 2, 4, 5}, party: []mapload.PartyMember{
			{ID: "mage", Mage: true}, {ID: "mage-with-sword", Mage: true},
			{ID: "hired", MercenaryType: 1, Class: 23}, {ID: "fighter", Class: 23},
		}}}
	// A placed NPC follows the same current-equipment class chain through its
	// roster record; a mercenary retains his own fixed class instead.
	mw.mission.state = &Mission{}
	mw.mission.state.Start.Roster = map[sim.EntityID]mapload.PartyMember{3: {ID: "npc", Mage: true, Class: 24}}
	before := w.Hash()
	draws := mw.entityDraws()
	for i, want := range []bool{true, false, true, true, false} {
		if draws[i].CastCapable != want || !draws[i].SpellStateKnown || draws[i].Art != nil {
			t.Fatalf("same server type, client class%d: draw%d capability%v want%v", mw.spellClientClass(entities[i].ID, entities[i].Class), i, draws[i].CastCapable, want)
		}
	}
	if before != w.Hash() {
		t.Fatal("capability projection changed hashed state")
	}
	if got := mw.spellClientClass(99, 24); got != 24 {
		t.Fatalf("unbound actor class = %d, want 24", got)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindUnequip, Entity: 2, X: 1}})
	if !mw.entityDraws()[1].CastCapable {
		t.Fatal("live sword removal did not restore the mage client class")
	}
	// No installed body-name resolution: retain an explicit custom class,
	// without consulting art success, mana or a guessed server-band mapping.
	mw.mission.list = nil
	mw.mission.party[0].Class = 24
	if !mw.entityDraws()[0].CastCapable {
		t.Fatal("explicit unresolved/custom client class was discarded")
	}
}

func TestQuickSpellsCatalogIdentityUnionAndCustomTable(t *testing.T) {
	// Literal claim table, independently listed; table order is deliberately
	// reversed and includes two non-book IDs. Compact arithmetic cannot pass.
	want := []uint32{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12, 6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}
	var table []sim.SpellRule
	for id := 28; id >= 1; id-- {
		table = append(table, sim.SpellRule{ID: uint16(id)})
	}
	selected := []sim.Entity{{KnownSpells: 1 << 23}, {KnownSpells: 1<<16 | 1<<18}}
	book, fixed := selectedSpellbook(sim.Rules{}, selected, table, nil, ui.Words{}, nil)
	if !fixed || len(book) != 24 {
		t.Fatalf("catalog fixed%v len%d", fixed, len(book))
	}
	for i, entry := range book {
		available := entry.ID == 23 || entry.ID == 16 || entry.ID == 18
		if entry.ID != want[i] || entry.Unavailable == available {
			t.Fatalf("cell%d=%+v want ID%d availability%v", i, entry, want[i], available)
		}
	}
	book, fixed = selectedSpellbook(sim.Rules{}, selected[1:], table, nil, ui.Words{}, nil)
	if !book[5].Unavailable || book[7].Unavailable || !fixed {
		t.Fatal("changing selection retained an old union member")
	}
	custom := []sim.SpellRule{{ID: 89}, {ID: 16}, {ID: 3}}
	book, fixed = selectedSpellbook(sim.Rules{}, selected, custom, nil, ui.Words{}, nil)
	if fixed || len(book) != 3 || book[0].ID != 89 || !book[0].Unavailable || book[1].ID != 16 || book[1].Unavailable {
		t.Fatalf("custom table identity/order lost: %+v fixed%v", book, fixed)
	}
}

func quickCitySource1119(t *testing.T, indices []int32) []byte {
	t.Helper()
	d := cityfixture.City(false)
	data := make([]byte, len(indices)*4)
	for i, n := range indices {
		binary.LittleEndian.PutUint32(data[i*4:], uint32(n))
	}
	d.State.Values["/SpellBook/Shortcuts"] = sav.CityStateValueData{Kind: 6, Bytes: data}
	p, err := sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	u := sav.CityUpdate{Label: []byte("quick source"), Money: 123}
	for _, c := range p.Roster() {
		u.Characters = append(u.Characters, sav.CityCharacterUpdate{Identity: c.Identity, Name: c.Name, Stats: c.Stats, SkillLevels: c.SkillLevels, SkillXP: c.SkillXP, Experience: c.Experience})
	}
	raw, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestQuickSpellsOriginalPopulatedMappingAndCurrentSave(t *testing.T) {
	indices := []int32{5, 7, 22, -1}
	want := [4]uint32{23, 16, 19, 0}
	f, _ := city1095Front(t)
	raw := quickCitySource1119(t, indices)
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town || f.quickSpells != want {
		t.Fatalf("source import slots%v town%v err%v", f.quickSpells, town, err)
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := f.originalCity.marshal(f, s, "unchanged populated")
	if err != nil {
		t.Fatal(err)
	}
	sf, err := sav.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := sf.StateStore()
	got, ok := r.GetIntArray("SpellBook", "Shortcuts")
	if !ok || !reflect.DeepEqual(got, indices) {
		t.Fatalf("supported SAV path changed source values %v", got)
	}
	// Native provenance reconstructs its source baseline; it cannot silently
	// treat the changed live bindings as the original values on fresh LOAD.
	s.QuickSpells = [4]uint32{1, 65535}
	b, err := EncodeSave(s, "changed")
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := city1095Front(t)
	if _, _, err := g.Restore(back); err != nil {
		t.Fatal(err)
	}
	if g.quickSpells != s.QuickSpells || g.originalCity.baselineQuickSpells != want {
		t.Fatal("native source and live values were conflated")
	}
	current, err := g.ExportCurrentSave(back, "changed")
	if err != nil {
		t.Fatal(err)
	}
	cold, _ := city1095Front(t)
	if _, town, err := cold.RestoreOriginal(current); err != nil || !town || cold.quickSpells != s.QuickSpells {
		t.Fatal("current SAV lost changed spell bindings", cold.quickSpells, town, err)
	}
}

func TestQuickSpellsMalformedOriginalAndNativeProvenanceAreAtomic(t *testing.T) {
	f, _ := city1095Front(t)
	f.quickSpells = [4]uint32{16, 1}
	for _, indices := range [][]int32{{-2, -1, -1, -1}, {24, -1, -1, -1}, {2147483647, -1, -1, -1}, {5, 5, -1, -1}} {
		raw := quickCitySource1119(t, indices)
		before := *f
		if _, _, err := f.RestoreOriginal(raw); err == nil {
			t.Fatalf("malformed source accepted %v", indices)
		}
		if !reflect.DeepEqual(*f, before) {
			t.Fatal("failed original LOAD changed session")
		}
	}
	for _, indices := range [][]int32{{}, {-1}, {-1, -1, -1}, {-1, -1, -1, -1, -1}} {
		if _, err := quickSpellsFromOriginalIndices(indices); err == nil {
			t.Fatal("non-four original slots accepted")
		}
	}
	for _, malformed := range []struct {
		offset int
		value  uint32
	}{{-4, 2}, {-8, 12}, {-8, 0}} {
		sf, err := sav.Open(quickCitySource1119(t, []int32{5, 7, 22, -1}))
		if err != nil {
			t.Fatal(err)
		}
		// Independent REG record mutation: name+0 is node+16, kind is
		// node+12 and byte length node+8. No shortcut decoder builds this input.
		at := bytes.Index(sf.Store, []byte("Shortcuts\x00"))
		if at < 16 {
			t.Fatal("synthetic source has no Shortcuts record")
		}
		binary.LittleEndian.PutUint32(sf.Store[at+malformed.offset:], malformed.value)
		before := *f
		if _, _, err := f.RestoreOriginal(sf.Marshal()); err == nil || !reflect.DeepEqual(*f, before) {
			t.Fatal("malformed source kind/length was not refused atomically")
		}
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	for i := range s.OriginalCity.Document.State.ValueRecords {
		r := &s.OriginalCity.Document.State.ValueRecords[i]
		if r.Path == "/SpellBook/Shortcuts" {
			binary.LittleEndian.PutUint32(r.Value.Bytes, 24)
		}
	}
	before := *f
	if _, _, err := DecodeSave(quickEnvelope1119(t, s)); err == nil {
		t.Fatal("malformed native source provenance decoded")
	}
	if _, _, err := f.Restore(s); err == nil || !reflect.DeepEqual(*f, before) {
		t.Fatal("malformed native source provenance was not atomic")
	}
}
