package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestTheParticipantsOwnCharacterLeadsTheRestoredParty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		chars []sav.Character
		from  int
		rule  LeadRule
		order string
	}{{
		name: "the file names him and the runtime id agrees",
		chars: []sav.Character{
			{Name: "Witch", RuntimeID: 22, MapUnitID: 21},
			{Name: "Danath", RuntimeID: 1, Hero: true},
		},
		from: 1, rule: LeadNamed, order: "Danath,Witch",
	}, {
		// THE FIELD WINS. If the two ever disagree on a real file this is the
		// arm that decides it, and no test that agreed with both would say so.
		name: "the file names one character and the runtime id another",
		chars: []sav.Character{
			{Name: "Witch", RuntimeID: 1},
			{Name: "Danath", RuntimeID: 22, Hero: true},
		},
		from: 1, rule: LeadNamed, order: "Danath,Witch",
	}, {
		name: "the file names nobody, so the runtime id decides",
		chars: []sav.Character{
			{Name: "a", RuntimeID: 114}, {Name: "b", RuntimeID: 113},
			{Name: "c", RuntimeID: 112}, {Name: "Danath", RuntimeID: 1},
			{Name: "e", RuntimeID: 111},
		},
		from: 3, rule: LeadRuntimeID, order: "Danath,a,b,c,e",
	}, {
		// Zero is a real answer and has to be told apart from "no rule
		// applied", which is what -1 is for.
		name: "the leader is already first",
		chars: []sav.Character{
			{Name: "Danath", RuntimeID: 1, Hero: true}, {Name: "Witch", RuntimeID: 22},
		},
		from: 0, rule: LeadNamed, order: "Danath,Witch",
	}, {
		name:  "neither rule resolves anybody",
		chars: []sav.Character{{Name: "a", RuntimeID: 22}, {Name: "b", RuntimeID: 23}},
		from:  -1, rule: LeadKept, order: "a,b",
	}, {
		// Two matches is not a tie to break. SAV-ID-015 is Medium as a law:
		// a corpse decay frees the bitmap bit and the next spawn reuses the
		// lowest free one, so a save with mid-session churn could carry two.
		name: "two characters carry the runtime id",
		chars: []sav.Character{{Name: "a", RuntimeID: 22}, {Name: "b", RuntimeID: 1},
			{Name: "c", RuntimeID: 1}},
		from: -1, rule: LeadKept, order: "a,b,c",
	}, {
		name: "two characters are named by the file, so the fallback decides",
		chars: []sav.Character{{Name: "a", RuntimeID: 22, Hero: true},
			{Name: "b", RuntimeID: 1}, {Name: "c", RuntimeID: 23, Hero: true}},
		from: 1, rule: LeadRuntimeID, order: "b,a,c",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got, from, rule := leadFirst(tc.chars)
			if from != tc.from {
				t.Errorf("the leader came from position %d, want %d", from, tc.from)
			}
			if rule != tc.rule {
				t.Errorf("the rule was %d (%s), want %d", rule, rule, tc.rule)
			}
			var order []string
			for _, c := range got {
				order = append(order, c.Name)
			}
			if strings.Join(order, ",") != tc.order {
				t.Errorf("the party is %v, want %s", order, tc.order)
			}
		})
	}
	if got, from, rule := leadFirst(nil); from != -1 || got != nil || rule != LeadKept {
		t.Errorf("no characters gave leader %d and rule %d", from, rule)
	}
}

// TestWithdrawRestoredTakesTheDoubleAway. A hired mercenary is in the human
// participant's own group in the file AND the map still carries the record
// he was placed from, so a resume that restored him as a party member
// without withdrawing the record would build two entities for one person.
//
// Measured over the owner's fourteen install saves, 27 of the 42 restored
// characters carry a map unit id, so this is the ordinary case and not an edge.
func TestRestoredCharactersKeepTheirFigureClassAndSex(t *testing.T) {
	row := func(typeID, face, gender int32) []int32 {
		p := make([]int32, data.MinHumanRow)
		for i := range p {
			p[i] = -1
		}
		p[16], p[17], p[18] = typeID, face, gender
		return p
	}

	for _, tc := range []struct {
		name   string
		params []int32
		dir    data.FigureDir
		face   int
		mage   bool
	}{
		{name: "woman fighter", params: row(3, 7, 1), dir: data.FigureDirWomanFighter, face: 7},
		{name: "woman mage", params: row(0x17, 4, 1), dir: data.FigureDirWomanMage, face: 4, mage: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := &mapload.Table{Humans: dbCollection{{}, {name: tc.name, params: tc.params}}}
			report := &RestoredParty{}
			member := restoredMember(sav.Character{Name: "Naira", Class: "Human", DefRow: 1, Hero: true}, data.BodyList{}, table, report)
			if member.FigureDir != string(tc.dir) || member.FigureFace != tc.face {
				t.Errorf("figure = %q face %d, want %q face %d", member.FigureDir, member.FigureFace, tc.dir, tc.face)
			}
			if member.Name != "Naira" || !member.PlayerCharacter || !member.StartingHero {
				t.Errorf("identity = %q player=%v starting=%v, want Naira/true/true", member.Name, member.PlayerCharacter, member.StartingHero)
			}
			if member.Mage != tc.mage {
				t.Errorf("Mage = %v, want %v", member.Mage, tc.mage)
			}
			if report.Definitions != 1 {
				t.Errorf("resolved definitions = %d, want 1", report.Definitions)
			}
		})
	}
}

func TestRestoredNPCDefinitionKeepsCorpseLootPolicyWithoutReadingDisplayName(t *testing.T) {
	params := make([]int32, data.MinHumanRow)
	for i := range params {
		params[i] = -1
	}
	params[16] = 3
	table := &mapload.Table{Humans: dbCollection{{}, {name: "NPC14_1", params: params}}}
	member := restoredMember(sav.Character{
		Name: "localized mercenary", Class: "Human", DefRow: 1,
	}, data.BodyList{}, table, &RestoredParty{})
	if !member.SuppressCorpseLoot {
		t.Fatal("restored NPC template lost corpse-loot suppression")
	}

	m := &alm.Map{Width: 8, Height: 8, Tiles: make([]uint16, 64), Overlay: make([]uint8, 64)}
	w, st, err := mapload.StartMission(m, table, mapload.DifficultyNormal, []mapload.PartyMember{member})
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if len(st.IDs) != 1 || !w.Entities()[0].SuppressCorpseLoot {
		t.Fatalf("restored NPC mission entity = %+v, ids=%v; want corpse-loot suppression", w.Entities(), st.IDs)
	}
}

func TestPersistentOriginalCharacterExcludesTheTemporaryMission20Roster(t *testing.T) {
	rows := make(dbCollection, 202)
	rows[27].name, rows[28].name, rows[29].name = "PC_Naira", "PC_Fergard", "PC_Reniesta"
	rows[42].name, rows[58].name, rows[201].name = "PC_Paladin", "NPC_Mercenary", "NPC_Sarindar"
	table := &mapload.Table{Humans: rows}
	chars := []sav.Character{
		{Name: "", Class: "Human", DefRow: 58},
		{Name: "", Class: "Human", DefRow: 58},
		{Name: "", Class: "Human", DefRow: 58},
		{Name: "Danath", Class: "Human", DefRow: 26, Hero: true},
		{Name: "Sarindar", Class: "Human", DefRow: 201},
	}
	var kept []string
	for _, c := range chars {
		if persistentOriginalCharacter(c, table) {
			kept = append(kept, c.Name)
		}
	}
	if len(kept) != 1 || kept[0] != "Danath" {
		t.Fatalf("persistent characters = %v, want Danath alone", kept)
	}
	for _, c := range []sav.Character{
		{Name: "Naira", Class: "Human", DefRow: 27},
		{Name: "Fergard", Class: "Human", DefRow: 28},
		{Name: "Reniesta", Class: "Human", DefRow: 29},
		{Name: "Brian", Class: "Human", DefRow: 42},
	} {
		if !persistentOriginalCharacter(c, table) {
			t.Errorf("persistent hero template row %d (%s) was excluded", c.DefRow, c.Name)
		}
	}
}

// TestADecodedNonPrimaryCompanionSurvivesTheWholeConstructionBoundary is the
// synthetic shape of the owner-reported Brian save. No original-save bytes or
// installed data enter the fixture: it begins at the lawful decoder's Character
// output and follows the generic PC_ membership rule, mission construction,
// canonical stock/experience state, byte-form round trip and UI sheet/doll
// projection. The numbers are intentionally the decoded fields, never a
// character-name exception in the loader.
func TestADecodedNonPrimaryCompanionSurvivesTheWholeConstructionBoundary(t *testing.T) {
	params := make([]int32, data.MinHumanRow)
	for i := range params {
		params[i] = -1
	}
	params[16], params[17], params[18] = 3, 7, 0
	rows := make(dbCollection, 43)
	rows[42] = dbEntry{name: "PC_Companion", params: params}
	table := &mapload.Table{Humans: rows}
	worn := []sav.Piece{{Code: 0x2126, Stack: 1}, {Code: 0xb223, Stack: 1}, {Code: 0x3344, Stack: 1}}
	pack := []sav.Piece{{Code: 0x5001, Stack: 3}, {Code: 0x5002, Stack: 2}}
	c := sav.Character{
		Name: "Companion", Class: "Human", DefRow: 42, RuntimeID: 79, MapUnitID: 6,
		Experience: 10575, Cell: 15<<8 | 14,
		SkillLevels: [sav.CharacterSkillSlots]uint16{0, 25, 3, 0, 0, 1},
		SkillXP:     [sav.CharacterSkillSlots]uint32{0, 10144, 331, 0, 0, 100},
		Worn:        worn, Items: pack,
	}
	// The file's aggregate is a cross-check on the six canonical fields, not a
	// value the loader injects into one slot.
	var total uint32
	for _, xp := range c.SkillXP {
		total += xp
	}
	if total != c.Experience {
		t.Fatalf("fixture aggregate %d differs from per-skill total %d", c.Experience, total)
	}
	if !persistentOriginalCharacter(c, table) {
		t.Fatal("a non-primary PC_ companion was filtered out")
	}
	member := restoredMember(c, data.BodyList{}, table, &RestoredParty{})
	owned := mapload.OwnParty([]mapload.PartyMember{member})
	if owned[0].ID != "player:companion" {
		t.Fatalf("stable identity = %q, want the generic player identity", owned[0].ID)
	}

	m := &alm.Map{Width: 40, Height: 40, Tiles: make([]uint16, 40*40), Overlay: make([]uint8, 40*40)}
	w, start, err := mapload.StartMission(m, table, mapload.DifficultyNormal, owned)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if len(start.IDs) != 1 {
		t.Fatalf("mission minted %d party ids, want one", len(start.IDs))
	}
	id := start.IDs[0]
	var live sim.Entity
	for _, e := range w.Entities() {
		if e.ID == id {
			live = e
		}
	}
	for i := range c.SkillXP {
		if live.SkillXP[i] != int32(c.SkillXP[i]) || live.Skill[i] != int32(c.SkillLevels[i]) {
			t.Errorf("slot %d live level/xp = %d/%d, want %d/%d", i, live.Skill[i], live.SkillXP[i], c.SkillLevels[i], c.SkillXP[i])
		}
	}
	if !live.Humanoid {
		t.Error("the restored original-save party member is not classified Humanoid")
	}
	if got, _ := w.Equipped(id); got != member.Worn {
		t.Errorf("live worn set = %#v, decoded %#v", got, member.Worn)
	}
	if got, _ := w.Carried(id); len(got) != len(member.Carried) {
		t.Errorf("live container has %d pieces, decoded %d: %v", len(got), len(member.Carried), got)
	}

	ms := &Mission{Map: m, World: w, Start: start, Party: owned}
	chars := PartyCharacters(ms)
	if got := chars[id]; !got.Known || got.Name != c.Name {
		t.Errorf("party sheet projection = %+v, want known companion %q", got, c.Name)
	}
	_, unread := buildInventorySubject(nil, ms)
	// Base + an icon per occupied slot + a layer per slot the figure pass draws
	// (slot 3 has no step) proves the actual doll compositor received the
	// saved worn slots.
	if len(unread) < 2*len(worn) {
		t.Errorf("doll composition attempted only %d paths for %d worn slots: %v", len(unread), len(worn), unread)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("companion state changed hash across save/load")
	}
	backEntity := back.Entities()[0]
	if len(back.Entities()) > 1 {
		for _, e := range back.Entities() {
			if e.ID == id {
				backEntity = e
			}
		}
	}
	if backEntity.SkillXP != live.SkillXP || backEntity.Skill != live.Skill {
		t.Errorf("round-tripped skill state = %v/%v, want %v/%v", backEntity.Skill, backEntity.SkillXP, live.Skill, live.SkillXP)
	}
	if got, _ := back.Equipped(id); got != member.Worn {
		t.Errorf("round-tripped worn set = %#v, want %#v", got, member.Worn)
	}
}

func TestRestoredMemberUsesItsSavedSkillLevelsAndPerSkillXP(t *testing.T) {
	params := make([]int32, data.MinHumanRow)
	for i := range params {
		params[i] = -1
	}
	params[10], params[11], params[12], params[13], params[14], params[15] = 99, 98, 97, 96, 95, 94
	params[16], params[17], params[18] = 3, 1, 0
	table := &mapload.Table{Humans: dbCollection{{}, {name: "template", params: params}}}
	c := sav.Character{
		Name: "independent", Class: "Human", DefRow: 1, Hero: true, Experience: 210,
		SkillLevels: [sav.CharacterSkillSlots]uint16{1, 2, 3, 4, 5, 6},
		SkillXP:     [sav.CharacterSkillSlots]uint32{10, 20, 30, 40, 50, 60},
	}

	member := restoredMember(c, data.BodyList{}, table, &RestoredParty{})
	wantLevels := [data.SkillSlots]int32{1, 2, 3, 4, 5, 6}
	wantXP := [data.SkillSlots]int32{10, 20, 30, 40, 50, 60}
	if member.Hero.Skill != wantLevels {
		t.Fatalf("restored skill levels = %v, want saved values %v", member.Hero.Skill, wantLevels)
	}
	if member.Carry == nil || member.Carry.SkillXP != wantXP {
		t.Fatalf("restored per-skill XP = %v, want saved values %v", member.Carry, wantXP)
	}
}

func TestRestoredLoadoutKeepsSavedStackCounts(t *testing.T) {
	report := &RestoredParty{}
	worn, carried := restoredLoadout(sav.Character{
		Worn:  []sav.Piece{{Code: 0x0103, Stack: 2}},
		Items: []sav.Piece{{Code: 0x0e06, Stack: 3}},
	}, report)
	if worn[0] != 0x0103 {
		t.Errorf("weapon slot = %#04x, want %#04x", worn[0], 0x0103)
	}
	want := []uint16{0x0103, 0x0e06, 0x0e06, 0x0e06}
	if len(carried) != len(want) {
		t.Fatalf("pack = %v, want %v", carried, want)
	}
	for i := range want {
		if carried[i] != want[i] {
			t.Errorf("pack[%d] = %#04x, want %#04x", i, carried[i], want[i])
		}
	}
	if report.Worn != 1 || report.Carried != 4 {
		t.Errorf("report worn/carried = %d/%d, want 1/4", report.Worn, report.Carried)
	}
}

func TestARestoredMageStaffRecoversItsAuthoredSpellAttachment(t *testing.T) {
	params := make([]int32, data.MinHumanRow)
	for i := range params {
		params[i] = -1
	}
	params[16], params[17], params[18] = 0x17, 4, 1
	weapons := dbCollection{{}, {name: "Wood Staff", params: chargenWeaponParams(data.SkillBlade)}}
	table := &mapload.Table{
		Humans:  dbCollection{{}, {name: "mage", params: params}},
		Weapons: weapons, Shapes: emptyScale{}, Materials: emptyScale{},
	}
	staff, err := data.ResolveWeapon("Wood Staff", table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	member := restoredMember(sav.Character{
		Name: "restored mage", Class: "Human", DefRow: 1, Hero: true,
		Worn: []sav.Piece{{Code: uint16(staff.Code), Stack: 1}},
	}, data.BodyList{}, table, &RestoredParty{})
	if member.Weapon == nil || member.Weapon.SpellName != "Fire_Arrow" || member.Weapon.SpellPower != 10 {
		t.Fatalf("restored staff = %+v, want authored Fire_Arrow:10 attachment", member.Weapon)
	}
	derived, _, _ := mapload.PartySpawnWithTable(member, table)
	if derived.Combat.SpellName != "Fire_Arrow" || derived.Combat.SpellPower != 10 {
		t.Fatalf("mission construction derived spell = %q:%d, want Fire_Arrow:10",
			derived.Combat.SpellName, derived.Combat.SpellPower)
	}
	legacyCurrent := mapload.CloneParty([]mapload.PartyMember{member})[0]
	legacyCurrent.Carry.LiveLoad = &sim.ActorLoadSnapshot{}
	legacyCurrent.Carry.LiveLoad.Inventory.Source.Class = 2
	derived, _, _ = mapload.PartySpawnWithTable(legacyCurrent, table)
	if derived.Combat.SpellName != "Fire_Arrow" || derived.Combat.SpellPower != 10 {
		t.Fatal("current Human dropped its code-only attachment", derived.Combat)
	}
	table.Spells = dbCollection{{}, {name: "Fire Arrow"}, {name: "Fire Ball"}}
	for _, tc := range []struct {
		name    string
		effects []sav.ItemEffect
		spell   string
		power   int32
	}{
		{name: "explicit empty"},
		{name: "non-cast effect", effects: []sav.ItemEffect{{Kind: 12, Operand: 4}}},
		{name: "explicit zero ID", effects: []sav.ItemEffect{{Kind: 41}}},
		{name: "zero", effects: []sav.ItemEffect{{Kind: 41, Operand: 1}}, spell: "Fire_Arrow"},
		{name: "current one", effects: []sav.ItemEffect{{Kind: 41, Operand: 1<<16 | 1}}, spell: "Fire_Arrow", power: 1},
		{name: "signed", effects: []sav.ItemEffect{{Kind: 41, Operand: 0xfffe<<16 | 1}}, spell: "Fire_Arrow", power: -2},
		{name: "above template", effects: []sav.ItemEffect{{Kind: 41, Operand: 23<<16 | 1}}, spell: "Fire_Arrow", power: 23},
		{name: "other spell", effects: []sav.ItemEffect{{Kind: 41, Operand: 7<<16 | 2}}, spell: "Fire_Ball", power: 7},
		{name: "first ordered attachment", effects: []sav.ItemEffect{{Kind: 41, Operand: 0xfffe<<16 | 1}, {Kind: 41, Operand: 2<<16 | 2}}, spell: "Fire_Arrow", power: -2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := restoredMember(sav.Character{Name: "current mage", Class: "Human", DefRow: 1, Hero: true,
				Worn: []sav.Piece{{Class: "Weapon", Row: uint8(staff.Row), Code: uint16(staff.Code), Stack: 1, Kind: 2, Effects: tc.effects}}}, data.BodyList{}, table, &RestoredParty{})
			if current.Weapon == nil || current.Weapon.SpellName != tc.spell || current.Weapon.SpellPower != tc.power {
				t.Fatal("current attachment was replaced by the code template", current.Weapon)
			}
			// A stale cached projection must not replace an explicit current Item.
			current.Weapon = member.Weapon
			current = mapload.CloneParty([]mapload.PartyMember{current})[0]
			before := mapload.CloneParty([]mapload.PartyMember{current})[0]
			loadout := mapload.PartyLoadout(current, table)
			if loadout.Weapon == nil || loadout.Weapon.SpellName != tc.spell || loadout.Weapon.SpellPower != tc.power {
				t.Fatal("next mission discarded the current weapon projection", loadout.Weapon)
			}
			for _, liveLoad := range []bool{false, true} {
				p := mapload.CloneParty([]mapload.PartyMember{current})[0]
				if liveLoad {
					p.Carry.LiveLoad = &sim.ActorLoadSnapshot{}
					p.Carry.LiveLoad.Inventory.Source.Class = 2
				}
				got, _, _ := mapload.PartySpawnWithTable(p, table)
				if got.Combat.SpellName != tc.spell || got.Combat.SpellPower != tc.power {
					t.Fatal("current spawn changed attachment", liveLoad, got.Combat.SpellName, got.Combat.SpellPower)
				}
			}
			if !reflect.DeepEqual(current, before) {
				t.Fatal("loadout/spawn changed the current Item or ordered effects")
			}
		})
	}

}

func TestWithdrawRestoredTakesTheDoubleAway(t *testing.T) {
	m := resumeMap()
	before := len(m.Units)
	// Two ids the map holds and one it does not: a character can carry an id no
	// map record answers to, and the count has to say what the map actually gave
	// up rather than what was claimed.
	n := WithdrawRestored(m, []uint16{100, 102, 555})
	if n != 2 {
		t.Errorf("withdrew %d records, want the two the map held", n)
	}
	if len(m.Units) != before-2 {
		t.Errorf("the map holds %d units, want %d", len(m.Units), before-2)
	}
	for _, u := range m.Units {
		if u.UnitID == 100 || u.UnitID == 102 {
			t.Errorf("unit %d is still placed", u.UnitID)
		}
	}
	// EVERY OTHER RECORD SURVIVES INTACT, in order. The withdrawal filters a
	// slice in place, and a filter that wrote past its own read cursor would
	// leave a duplicate rather than a gap — which builds a second entity for a
	// unit nobody claimed.
	for _, want := range []uint16{101, 103, 900} {
		found := 0
		for _, u := range m.Units {
			if u.UnitID == want {
				found++
			}
		}
		if found != 1 {
			t.Errorf("unit %d appears %d times after the withdrawal", want, found)
		}
	}
}

// TestWithdrawRestoredIsTotal. Nothing is claimed on the path a save with no
// restored character takes, and a nil map is what a caller holding no decode has.
func TestWithdrawRestoredIsTotal(t *testing.T) {
	if n := WithdrawRestored(nil, []uint16{1}); n != 0 {
		t.Errorf("a nil map withdrew %d", n)
	}
	m := resumeMap()
	before := len(m.Units)
	if n := WithdrawRestored(m, nil); n != 0 || len(m.Units) != before {
		t.Errorf("an empty claim withdrew %d and left %d units", n, len(m.Units))
	}
}

// TestARestoredPartyReportsEveryAxisItDidNotApply. The counted report
// distinguishes retained source words and spells from the journal arrays
// whose meaning is still Unknown.
func TestARestoredPartyReportsEveryAxisItDidNotApply(t *testing.T) {
	p := RestoredParty{
		Characters: 5, Pools: 5, Definitions: 5, LeadFrom: 3, LeadRule: LeadNamed,
		MapUnitIDs: []uint16{136, 139, 140, 141}, Withdrawn: 4, Rebound: 4,
		Worn: 29, Carried: 7, Weapons: 4, Armed: 4,
		SpellbookChars: 1, SpellRecords: 3, SpellParameters: 3,
		JournalChars: 1, JournalEntries: 238,
		Capacity: 5, Unknowns: 10, Skills: 5,
	}
	s := p.String()
	for _, want := range []string{
		"5 persistent characters restored",
		"led from position 3 by the character the file names as the player's own",
		"4 claimed a map placement and 4 of those records were withdrawn",
		"4 rebound to the script",
		"NOT APPLIED:",
		"1 spellbooks, 3 learned spell memberships RESTORED",
		"3 saved spell parameter sets RESTORED",
		"1 journals holding 238 entries",
		"5 carry capacities",
		"10 own-weight/load words RESTORED",
		"5 character-owned skill/experience sets restored",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the report does not say %q:\n%s", want, s)
		}
	}
	p.LeadRule = LeadRuntimeID
	if s := p.String(); !strings.Contains(s, "the file named none") {
		t.Errorf("the report does not say the fallback ordered the party:\n%s", s)
	}
	p.LeadFrom, p.LeadRule = -1, LeadKept
	if s := p.String(); !strings.Contains(s, "the file's own order was kept") {
		t.Errorf("the report does not say no rule applied:\n%s", s)
	}
}

// Learned membership must neither lose non-template spells nor add template
// spells that this saved character does not know (MAGIC-SPELL-001).
func TestARestoredCharacterKnowsItsSavedSpellsNotTheTemplate(t *testing.T) {
	row := func(spells int32) []int32 {
		p := make([]int32, data.MinHumanRow)
		for i := range p {
			p[i] = -1
		}
		p[16], p[17], p[18] = 0x17, 4, 1 // a woman mage
		p[25] = spells
		return p
	}

	for _, tc := range []struct {
		name    string
		cell    int32
		present bool
		spells  []sav.SavedSpell
		want    uint32
	}{
		{name: "learned non-template spells", cell: 1 << 1, present: true, spells: []sav.SavedSpell{{Slot: 6, ID: 6}, {Slot: 26, ID: 26}}, want: 1<<6 | 1<<26},
		{name: "empty book", cell: 1 << 1, present: true},
		{name: "absent book", cell: 1 << 1},
		{name: "missing template book", cell: -1, present: true, spells: []sav.SavedSpell{{Slot: 28, ID: 28}}, want: 1 << 28},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := &mapload.Table{Humans: dbCollection{{}, {name: "mage", params: row(tc.cell)}}}
			member := restoredMember(sav.Character{Name: "Naira", Class: "Human", DefRow: 1, Hero: true, HasSpellbook: tc.present, Spells: tc.spells},
				data.BodyList{}, table, &RestoredParty{})
			if member.KnownSpells != tc.want {
				t.Errorf("KnownSpells = %#x, want %#x", member.KnownSpells, tc.want)
			}
			if !member.SpellbookRestored || member.SpellbookPresent != tc.present {
				t.Fatalf("lost saved presence metadata: %+v", member)
			}
		})
	}

	// A character whose class names no Humans row keeps an empty book rather
	// than failing to restore.
	member := restoredMember(sav.Character{Name: "Rat", Class: "Unit"}, data.BodyList{}, &mapload.Table{}, &RestoredParty{})
	if member.KnownSpells != 0 {
		t.Errorf("a non-Human record restored KnownSpells = %#x, want 0", member.KnownSpells)
	}
}
