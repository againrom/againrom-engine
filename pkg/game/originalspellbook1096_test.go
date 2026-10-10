package game

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"hash/crc32"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func spellbookFront1096(t *testing.T) *FrontEnd {
	f := poolFixtureFront(t, 91, 92)
	row := make([]int32, data.MinHumanRow)
	row[16], row[17], row[18], row[25] = 0x17, 4, 1, 1<<1
	rows := dbCollection{{}}
	for i := 1; i <= 28; i++ {
		p := make([]int32, 19)
		p[1], p[4], p[6], p[8], p[16], p[17], p[18] = 1, 1, 10, 1, 1, 1, 1
		rows = append(rows, dbEntry{name: "spell", params: p})
	}
	f.Table = &mapload.Table{Humans: dbCollection{{}, {name: "PC_fixture", params: row}}, Spells: rows}
	return f
}

func spellbookSave1096(present bool, ids ...byte) []byte {
	var spells []byte
	if present {
		spells = make([]byte, 0)
		for _, id := range ids {
			for len(spells) < int(id) {
				spells = append(spells, 0)
			}
			spells[int(id)-1] = id
		}
	}
	a := &poolFixtureActor{mapID: 91, cell: 0x0a0a, hp: 10, maxHP: 30, mana: 50, maxMana: 50, human: true, name: "First", spells: spells}
	b := &poolFixtureActor{mapID: 92, cell: 0x0a0b, hp: 10, maxHP: 30, mana: 50, maxMana: 50, human: true, name: "Second", spells: []byte{}}
	return savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a, b}}}}, nil))
}

func openSpellbookSave1096(t *testing.T, f *FrontEnd, bytes []byte) {
	t.Helper()
	opener, town, err := f.RestoreOriginal(bytes)
	if err != nil || town {
		t.Fatalf("RestoreOriginal: town=%v err=%v", town, err)
	}
	if err := f.App("saved-spellbooks").OpenMission(opener); err != nil {
		t.Fatal(err)
	}
}

func assertSpellbook1096(t *testing.T, f *FrontEnd, masks ...uint32) {
	t.Helper()
	if len(f.liveParty) != len(masks) {
		t.Fatalf("party length %d", len(f.liveParty))
	}
	for i, p := range f.liveParty {
		if p.KnownSpells != masks[i] || !p.SpellbookRestored {
			t.Fatalf("member %d book=%#x restored=%v", i, p.KnownSpells, p.SpellbookRestored)
		}
		e := poolEntity(t, f.live.world, uint16(91+i))
		if e.KnownSpells != masks[i] {
			t.Fatalf("entity %d book=%#x want%#x", i, e.KnownSpells, masks[i])
		}
		rows := spellbookOf(sim.Rules{}, e, f.live.world.Spells(), nil, f.live.view.Words(), nil)
		var shown uint32
		for _, row := range rows {
			shown |= uint32(1) << row.ID
		}
		if shown != masks[i] {
			t.Fatalf("production book rows=%#x want%#x", shown, masks[i])
		}
	}
}

func TestOriginalSpellbook1096AppNativeAndNewLoad(t *testing.T) {
	f := spellbookFront1096(t)
	openSpellbookSave1096(t, f, spellbookSave1096(true, 6, 26))
	assertSpellbook1096(t, f, 1<<6|1<<26, 0)
	if !f.liveParty[0].SpellbookPresent || !f.liveParty[1].SpellbookPresent {
		t.Fatal("empty and nonempty presence metadata lost")
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSave(snapshot, "independent")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}
	back := spellbookFront1096(t)
	opener, town, err := back.Restore(decoded)
	if err != nil || town {
		t.Fatalf("native restore: %v", err)
	}
	if err := back.App("native-book").OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	assertSpellbook1096(t, back, 1<<6|1<<26, 0)
	if f.live.world.Hash() != back.live.world.Hash() {
		t.Fatal("native changed canonical book state")
	}
	for range 8 {
		sim.Step(f.live.world, nil)
		sim.Step(back.live.world, nil)
		if f.live.world.Hash() != back.live.world.Hash() {
			t.Fatal("native continuation diverged")
		}
	}
	openSpellbookSave1096(t, back, spellbookSave1096(false))
	assertSpellbook1096(t, back, 0, 0)
	if back.liveParty[0].SpellbookPresent || !back.liveParty[1].SpellbookPresent {
		t.Fatal("absent and empty books were conflated")
	}
}

func TestOriginalSpellbook1096MalformedLoadIsTransactional(t *testing.T) {
	for _, mission := range []uint32{0, 10} {
		f := spellbookFront1096(t)
		openSpellbookSave1096(t, f, spellbookSave1096(true, 6))
		before, _, _ := f.Snapshot(true)
		priorLive, priorTown := f.live, f.Town
		a := &poolFixtureActor{human: true, hp: 5, maxHP: 10, spells: []byte{29}}
		body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}}, nil)
		binary.LittleEndian.PutUint32(body[59:], mission)
		_, _, err := f.RestoreOriginal(savedContainer(body))
		if !errors.Is(err, sav.ErrSpellbook) {
			t.Fatalf("mission%d rejected with %v", mission, err)
		}
		after, _, _ := f.Snapshot(true)
		if priorLive != f.live || priorTown != f.Town || !reflect.DeepEqual(before, after) {
			t.Fatal("bad spellbook changed active session")
		}
	}
}

func TestOriginalSpellbook1096EarlierActorErrorCannotMaskBookFailure(t *testing.T) {
	for _, mission := range []uint32{0, 10} {
		for _, badCount := range []int{1, 64} {
			good := &poolFixtureActor{human: true, name: "Retained", hp: 10, maxHP: 30, spells: []byte{0, 0, 0, 0, 0, 6}}
			bad := &poolFixtureActor{human: true, name: "Rejected", hp: 10, maxHP: 30, spells: []byte{29}}
			actors := []*poolFixtureActor{good}
			for i := 0; i < badCount; i++ {
				actors = append(actors, bad)
			}
			body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{actors}}}, nil)
			binary.LittleEndian.PutUint32(body[59:], mission)
			tag := bytes.Index(body, []byte{255, 255, 1, 0, 5, 0, 'H', 'u', 'm', 'a', 'n'})
			if tag < 4 || binary.LittleEndian.Uint32(body[tag-4:tag]) != uint32(len(actors)) {
				t.Fatal("fixture actor-count anchor changed")
			}
			// A valid reference to the wrong class occurs BEFORE both Humans.
			binary.LittleEndian.PutUint32(body[tag-4:tag], uint32(len(actors)+1))
			body = append(append(append([]byte(nil), body[:tag]...), 2, 0), body[tag:]...)
			payload := savedContainer(body)
			parsed, err := sav.Open(payload)
			if err != nil {
				t.Fatal(err)
			}
			party, err := parsed.Party()
			if len(party) != 1 || !errors.Is(err, sav.ErrSpellbook) || !strings.Contains(err.Error(), "actor list holds a Player") {
				t.Fatalf("mixed errors lost failure identity: party=%d err=%v", len(party), err)
			}
			if strings.Count(err.Error(), "sav: invalid spellbook") != 1 {
				t.Fatal("book failures grew an unbounded error chain")
			}
			front := spellbookFront1096(t)
			openSpellbookSave1096(t, front, spellbookSave1096(true, 6, 26))
			before, _, err := front.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			priorLive, priorTown := front.live, front.Town
			if _, _, err := front.RestoreOriginal(payload); !errors.Is(err, sav.ErrSpellbook) {
				t.Fatalf("mixed-error LOAD: %v", err)
			}
			after, _, err := front.Snapshot(true)
			if err != nil || priorLive != front.live || priorTown != front.Town || !reflect.DeepEqual(before, after) {
				t.Fatal("mixed-error LOAD published partial state")
			}
			if _, _, err := loadOriginalMission(front, payload); !errors.Is(err, sav.ErrSpellbook) {
				t.Fatalf("diagnostic mixed-error LOAD: %v", err)
			}
		}
	}
}

func TestOriginalSpellbook1096CarryDoesNotReteachAndRetainsNewLearning(t *testing.T) {
	f := spellbookFront1096(t)
	openSpellbookSave1096(t, f, spellbookSave1096(true, 6))
	party := mapload.CloneParty(f.liveParty)
	party[0].WornItems[2] = sim.ItemInstance{Code: 0x0301, Kind: 1, Effects: []sim.ItemEffect{{Kind: 42, Operand: 1}}}
	party[0].CarriedItems = []sim.ItemInstance{{Code: 0x0e17, Kind: 5, Effects: []sim.ItemEffect{{Kind: 42, Operand: 26}}}}
	party[0].Carry = nil
	m := &alm.Map{Width: 40, Height: 40, Tiles: make([]uint16, 1600), Overlay: make([]byte, 1600)}
	w, st, err := mapload.StartMission(m, f.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	if w.Entities()[0].KnownSpells != 1<<6 {
		t.Fatal("loaded equipment taught an absent spell")
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindReadBook, Entity: st.IDs[0], X: 0}})
	if w.Entities()[0].KnownSpells != 1<<6|1<<26 {
		t.Fatal("ordinary new book learning failed after original load")
	}
	carried := mapload.CarryParty(party, w, st.IDs)
	if !carried[0].SpellbookRestored || !carried[0].SpellbookPresent || carried[0].KnownSpells != 1<<6|1<<26 {
		t.Fatal("carry lost membership or provenance")
	}
	next, _, err := mapload.StartMission(m, f.Table, mapload.DifficultyNormal, carried)
	if err != nil {
		t.Fatal(err)
	}
	if next.Entities()[0].KnownSpells != 1<<6|1<<26 {
		t.Fatal("next mission lost learning or replayed equipment")
	}
	// Existing native/generated members keep the old zero-metadata behaviour.
	party[0].SpellbookRestored = false
	party[0].Book = sim.Spellbook{}
	legacy, _, err := mapload.StartMission(m, f.Table, mapload.DifficultyNormal, party)
	if err != nil || legacy.Entities()[0].KnownSpells != 1<<1|1<<6 {
		t.Fatalf("old native default: %v entitybook=%#x", err, legacy.Entities()[0].KnownSpells)
	}
}

func TestOriginalSpellbook1096NewTownEquipmentTeachesOnlyItsOwnSpells(t *testing.T) {
	f, screen := shopRoom(t, nil)
	member := &f.Carried[0]
	member.Mage, member.SpellbookRestored, member.SpellbookPresent = true, true, true
	member.KnownSpells = 1 << 6
	old := sim.ItemInstance{Code: 0x0301, Kind: 1, Effects: []sim.ItemEffect{{Kind: 42, Operand: 1}}}
	new := sim.ItemInstance{Code: 0x0302, Kind: 1, Effects: []sim.ItemEffect{{Kind: 42, Operand: 26}}}
	(*screen.shopWornItemSlots(0))[2] = old
	screen.shopWearItemInto(3, new)
	if member.KnownSpells != 1<<6|1<<26 {
		t.Fatalf("new town equip book=%#x", member.KnownSpells)
	}
	// Viewing the card and removing the item do not replay or unlearn it.
	screen.composeShopFaces()
	screen.shopUnequipDoll(3)
	if member.KnownSpells != 1<<6|1<<26 || !member.SpellbookRestored || !member.SpellbookPresent {
		t.Fatal("town follow-up lost exact learned state")
	}
}

func TestOriginalSpellbook1096OldNativeMissingMetadataDefaults(t *testing.T) {
	// An old gob's type omits the new fields entirely, rather than asking
	// today's encoder to write their zero values under today's descriptor.
	var body bytes.Buffer
	old := struct {
		Party []struct {
			ID          string
			KnownSpells uint32
		}
	}{}
	old.Party = append(old.Party, struct {
		ID          string
		KnownSpells uint32
	}{"old-hero", 1 << 6})
	if err := gob.NewEncoder(&body).Encode(old); err != nil {
		t.Fatal(err)
	}
	encoded := append([]byte(saveMagic), saveVersion, 0, 0)
	encoded = binary.LittleEndian.AppendUint32(encoded, crc32.ChecksumIEEE(body.Bytes()))
	encoded = binary.LittleEndian.AppendUint32(encoded, uint32(body.Len()))
	encoded = append(encoded, body.Bytes()...)
	decoded, _, err := DecodeSave(encoded)
	if err != nil || len(decoded.Party) != 1 {
		t.Fatalf("old descriptor: %v", err)
	}
	p := decoded.Party[0]
	if p.KnownSpells != 1<<6 || p.SpellbookRestored || p.SpellbookPresent {
		t.Fatalf("old metadata defaults: %+v", p)
	}
}

func spellbookCitySource(t *testing.T) ([]byte, func() *FrontEnd) {
	t.Helper()
	model := cityfixture.City(false)
	book := model.Objects[2].Unit
	book.SpellbookFlag, book.SpellbookCount = 1, 27
	book.Spells = make([]uint16, 26)
	book.Spells[25] = 4
	model.Objects = append(model.Objects, sav.CityObjectData{Class: "Spell", Spell: &sav.CitySpellData{Fields: []byte{26, 0, 0, 0, 0, 0x67, 0x45, 0, 0}}})
	document, err := sav.CityFromData(model)
	if err != nil {
		t.Fatal(err)
	}
	update := sav.CityUpdate{Money: 123}
	for _, character := range document.Roster() {
		update.Characters = append(update.Characters, originalCityBaselineUpdate(character))
	}
	payload, err := document.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	rows := make(dbCollection, 30)
	for _, row := range []int{28, 29} {
		params := make([]int32, data.MinHumanRow)
		params[16], params[17], params[18], params[25] = 0x17, 4, 1, 1<<1
		rows[row] = dbEntry{name: "PC_fixture", params: params}
	}
	return payload, func() *FrontEnd {
		return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: &mapload.Table{Humans: rows}}}
	}
}

func assertCityBookSAV(t *testing.T, payload []byte, newFront func() *FrontEnd, want []mapload.PartyMember) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(payload)
	if err != nil {
		t.Fatal(err)
	}
	var actor uint16
	for i, record := range doc.Objects {
		if record.Class != "Human" {
			continue
		}
		for _, field := range record.Texts {
			if field.Name == "Name" && field.Value == want[0].Name {
				if actor != 0 {
					t.Fatal("ordinary Book actor name is ambiguous")
				}
				actor = uint16(i + 1)
			}
		}
	}
	if actor == 0 {
		t.Fatal("ordinary Book actor missing")
	}
	characters, err := sav.ReadDocumentCharacters(doc, []uint16{actor})
	if err != nil || len(characters) != 1 || characters[0].Character.KnownSpells() != want[0].KnownSpells&0x1ffffffe {
		t.Fatalf("ordinary Book lost current membership: %+v %v", characters, err)
	}
	cold := newFront()
	if _, town, err := cold.RestoreOriginal(payload); err != nil || !town {
		t.Fatalf("cold city SAV LOAD: town=%t err=%v", town, err)
	}
	if len(cold.Carried) != len(want) {
		t.Fatal("cold city SAV changed party membership")
	}
	for i, got := range cold.Carried {
		if got.ID != want[i].ID || got.Name != want[i].Name || got.Hero != want[i].Hero ||
			got.KnownSpells != want[i].KnownSpells || got.Book != want[i].Book ||
			got.SpellbookRestored != want[i].SpellbookRestored || got.SpellbookPresent != want[i].SpellbookPresent ||
			got.Carry.SkillXP != want[i].Carry.SkillXP {
			t.Fatalf("cold city SAV changed member %d current progress: got %+v want %+v", i, got, want[i])
		}
	}
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatalf("current city policy missing: %v", err)
	}
	r := &doc.Objects[actor-1]
	slots := append([]uint16(nil), cityGraftRefs(t, r, "Spells")...)
	if len(slots) == 0 || slots[0] == 0 {
		t.Fatal("ordinary Book control needs spell 1")
	}
	spell := slots[0]
	for len(slots) < 3 {
		slots = append(slots, 0)
	}
	slots[0], slots[2] = 0, spell
	mustSetRefs(r, "Spells", slots)
	mustSetCount(r, "Spells", uint32(len(slots)+1))
	mustSetValue(&doc.Objects[spell-1], "S08", 3)
	mustSetValue(&doc.Objects[spell-1], "S09", 73)
	mustSetValue(&doc.Objects[spell-1], "S0A", 0x83)
	mustSetValue(&doc.Objects[spell-1], "S0C", 54321)
	changed, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	back, err := sav.DecodeDocumentData(changed)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, _, err := sav.NativeActions(back.State)
	if err != nil || !bytes.Equal(leaf, unchanged) {
		t.Fatal("ordinary Book edit changed the supplemental leaf")
	}
	for cycle := 0; cycle < 2; cycle++ {
		cold = newFront()
		if _, town, err := cold.RestoreOriginal(changed); err != nil || !town {
			t.Fatalf("edited Book cold cycle %d: town=%t err=%v", cycle, town, err)
		}
		got := cold.Carried[0]
		if got.KnownSpells != want[0].KnownSpells&^2|1<<3 || got.Hero != want[0].Hero || got.Carry.SkillXP != want[0].Carry.SkillXP {
			t.Fatalf("ordinary edited Book lost to policy in cycle %d: %+v", cycle, got)
		}
		wantMode := sim.BookPresent
		if want[0].KnownSpells & ^uint32(0x1ffffffe) != 0 {
			wantMode = sim.BookNativePresent
		}
		if got.Book.State != wantMode || got.Book.Slots[2] != (sim.BookSpell{Range: 73, Defensive: 0x83, ManaCost: 54321}) {
			t.Fatalf("ordinary edited spell operands lost in cycle %d: %+v", cycle, got.Book)
		}
		if cycle == 0 {
			s, label, err := cold.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			changed, err = cold.ExportCurrentSave(s, label)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestOriginalSpellbook1096CurrentBookOverridesRetainedTemplate(t *testing.T) {
	payload, newFront := spellbookCitySource(t)
	f := newFront()
	if _, town, err := f.RestoreOriginal(payload); err != nil || !town {
		t.Fatalf("source city: %v", err)
	}
	// The retained document knows spell 26. The current member owns a different
	// book, and that progress must survive both SAV and a later ordinary edit.
	f.Carried[0].KnownSpells = 1 << 1
	f.Carried[0].Book = sim.Spellbook{State: sim.BookPresent}
	f.Carried[0].Book.Slots[0] = sim.BookSpell{Range: 19, Defensive: 2, ManaCost: 12345}
	before := mapload.CloneParty(f.Carried)
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	output, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, f.Carried) {
		t.Fatal("SAV export changed current native state")
	}
	assertCityBookSAV(t, output, newFront, before)
}

func TestOriginalSpellbook1096LegacyCityProvenanceKeepsNativeProgress(t *testing.T) {
	for _, edited := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchanged", true: "later progress"}[edited], func(t *testing.T) {
			payload, newFront := spellbookCitySource(t)
			front := newFront()
			if _, town, err := front.RestoreOriginal(payload); err != nil || !town {
				t.Fatalf("source city: %v", err)
			}
			legacy, _, err := front.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			if legacy.OriginalCity.Version != currentCityPartyImportVersion || legacy.Party[0].KnownSpells != 1<<26 {
				t.Fatal("new original import did not bind the saved book")
			}
			legacy.OriginalCity.Version = 1
			legacyCityFixtureIdentities(&legacy)
			for i := range legacy.OriginalCity.Bindings {
				p := &legacy.OriginalCity.Bindings[i].Baseline
				p.KnownSpells, p.SpellbookRestored, p.SpellbookPresent = 1<<1, false, false
				p.Book, p.OriginalHuman = sim.Spellbook{}, nil
				p.Carry.LiveLoad, p.Carry.OrderedStacks = nil, nil
			}
			for i := range legacy.Party {
				p := &legacy.Party[i]
				p.KnownSpells, p.SpellbookRestored, p.SpellbookPresent = 1<<1, false, false
				p.Book, p.OriginalHuman = sim.Spellbook{}, nil
				p.Carry.LiveLoad, p.Carry.OrderedStacks = nil, nil
			}
			if edited {
				legacy.Party[0].KnownSpells |= 1 << 6
				legacy.Party[0].Hero.Body++
				legacy.Party[0].Carry.SkillXP[2]++
				legacy.Party[0].Hero.Skill[2]++
			}
			want := mapload.CloneParty(legacy.Party)
			encoded, err := EncodeSave(legacy, "legacy native progress")
			if err != nil {
				t.Fatal(err)
			}
			decoded, _, err := DecodeSave(encoded)
			if err != nil {
				t.Fatal(err)
			}
			fresh := newFront()
			restore := legacyCompanionRegistry(fresh, 22)
			_, town, err := fresh.Restore(decoded)
			restore()
			if err != nil || !town {
				t.Fatalf("legacy LOAD must remain readable: %v", err)
			}
			back, _, err := fresh.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(mapload.CloneParty(back.Party), want) || back.OriginalCity.Version != 1 {
				t.Fatal("legacy LOAD replaced native progress or changed its baseline policy")
			}
			if _, err := EncodeSave(back, "again"); err != nil {
				t.Fatal(err)
			}
			output, err := fresh.ExportCurrentSave(back, "current book")
			if err != nil {
				t.Fatal(err)
			}
			assertCityBookSAV(t, output, newFront, want)
			// Version 1 is a strict previous import policy, not a general way to
			// trust whatever baseline matches the current native party.
			legacy.Party[0].Hero.Body++
			for i := range legacy.OriginalCity.Bindings {
				b := &legacy.OriginalCity.Bindings[i]
				if b.PartyID == legacy.Party[0].ID {
					b.Baseline.Hero.Body = legacy.Party[0].Hero.Body
				}
			}
			if _, _, err := newFront().Restore(legacy); err == nil {
				t.Fatal("version1 forged baseline accepted")
			}
		})
	}
}
