package game_test

import (
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/ui"
)

// itemTables is a definition table carrying the three collections the starting
// weapon resolves through, and nothing else. The factors and the row are the
// SHAPE of the shipped file rather than its content.
func itemTables(withSword bool) []byte {
	scale := func(name string, damage float64) synth.DataBinRow {
		d := make([]float64, 9)
		d[4], d[5], d[6] = damage, 1, 1
		return synth.DataBinRow{Name: name, Doubles: d}
	}
	var d synth.DataBin
	d.Rows[synth.DataBinShapes] = []synth.DataBinRow{scale("Common", 0.2)}
	d.Rows[synth.DataBinMaterials] = []synth.DataBinRow{scale("Iron", 1)}
	if withSword {
		d.Rows[synth.DataBinWeapons] = []synth.DataBinRow{{
			Name:   "Short Sword",
			Params: []int32{-1, -1, -1, -1, -1, 1, 23, 40, 0, 0, -1, 1, 9, 5, -1},
		}}
	}
	return d.Bytes()
}

// bowTables is itemTables' own shape, carrying a RANGED row instead of the
// blade one: a shape word ("Uncommon"), a material word ("Wood") and a
// Weapons row ("Short Bow") that together spell out the shoot slot's own
// literal, "Uncommon Wood Short Bow" — the same split startingWeapons'
// comment already describes for the shipped name. The attack type at slot 5
// is 10, which is FoldWeapon's own threshold for the ranged arm (data's
// meleeAttackTypes), and the range cell at slot 0xb is 6, so a hero holding
// this weapon derives a reach above 1 (spec 0120 AC-5).
func bowTables() []byte {
	scale := func(name string, damage float64) synth.DataBinRow {
		d := make([]float64, 9)
		d[4], d[5], d[6] = damage, 1, 1
		return synth.DataBinRow{Name: name, Doubles: d}
	}
	var d synth.DataBin
	d.Rows[synth.DataBinShapes] = []synth.DataBinRow{scale("Uncommon", 1)}
	d.Rows[synth.DataBinMaterials] = []synth.DataBinRow{scale("Wood", 1)}
	d.Rows[synth.DataBinWeapons] = []synth.DataBinRow{{
		Name: "Short Bow",
		//    0   1   2   3   4  kind min max toHit def  10 rng chg rlx  14
		Params: []int32{-1, -1, -1, -1, -1, 10, 10, 20, 5, 0, -1, 6, 8, 4, -1},
	}}
	return d.Bytes()
}

// THE STARTING WEAPON COMES OUT OF THE SAME WALK the placement collections do,
// and it is resolved rather than written down: 23 and 40 through a shape factor
// of 0.2 give (5, 3), which the retracted ladder would have made (23, 17).
func TestLoadDefinitionsResolvesTheStartingWeapon(t *testing.T) {
	d, err := game.LoadDefinitions(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: itemTables(true)}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	if d.Table == nil {
		t.Fatal("no table")
	}
	if d.StartWeaponErr != nil {
		t.Fatalf("StartWeaponErr = %v", d.StartWeaponErr)
	}
	got := *d.StartWeapon
	want := data.Weapon{
		// Row 1: index 0 of the Weapons collection is the reserved, never-written
		// entry every collection allocates, so "Short Sword" — the fixture's
		// only written row — resolves at index 1. Code 0x0101: material index 0
		// ("Iron", this fixture's only entry), weapon class 1, shape index 0 (no
		// shape word in "Iron Short Sword"), row 1.
		Name: "Iron Short Sword", Row: 1, Code: 0x0101, DamageBase: 5, DamageSpread: 3,
		ToHit: 0, Defence: 0, AttackType: data.SkillBlade,
		ChargeTime: 9, RelaxTime: 5, Range: 1,
	}
	if got != want {
		t.Fatalf("StartWeapon = %+v\nwant %+v", got, want)
	}
}

// A TABLE THAT DOES NOT YIELD IT IS NOT FATAL: the reason is carried, the table
// still comes out, and LoadTable — which is this function's first field — is
// unaffected. That is the FontErr shape and it is deliberate: an install whose
// archive opens must not stop being a game over one row.
func TestATableWithNoStartingWeaponIsNotFatal(t *testing.T) {
	fsys := worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: itemTables(false)}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}})

	d, err := game.LoadDefinitions(fsys)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	if d.StartWeapon != nil {
		t.Fatalf("StartWeapon = %+v, want none", *d.StartWeapon)
	}
	if d.StartWeaponErr == nil || !strings.Contains(d.StartWeaponErr.Error(), "Short Sword") {
		t.Errorf("StartWeaponErr = %v, want a reason naming the row it looked for", d.StartWeaponErr)
	}
	if d.Table == nil {
		t.Error("a table that cannot arm the hero yielded no table either")
	}
	if _, err := game.LoadTable(fsys); err != nil {
		t.Errorf("LoadTable: %v — an absent weapon must not reach it", err)
	}
}

// The five ordinary literals are one per trained slot, and slot 0 — the one the
// character sheet does not show — names none.
func TestTheStartingWeaponLiterals(t *testing.T) {
	want := map[int32]string{
		data.SkillBlade:   "Iron Short Sword",
		data.SkillAxe:     "Uncommon Bronze Axe",
		data.SkillBludgen: "Uncommon Bronze Mace",
		data.SkillPike:    "Bronze Pike",
		data.SkillShoot:   "Uncommon Wood Short Bow",
	}
	for slot, w := range want {
		got, ok := game.StartingWeaponName(false, slot)
		if !ok || got != w {
			t.Errorf("slot %d = (%q, %v), want (%q, true)", slot, got, ok, w)
		}
	}
	for _, slot := range []int32{-1, data.SkillGeneral, data.SkillSlots, 99} {
		if got, ok := game.StartingWeaponName(false, slot); ok {
			t.Errorf("slot %d named %q, want none", slot, got)
		}
	}
}

// mageStaffLiteral is the shipped mage weapon literal, WHOLE (0139 spec
// D-13; cite HERO-START-039): `{castSpell=Fire_Arrow:10}` and all, as of
// this story — the attachment used to be cut and disclosed here (0134 R-4),
// because nothing in this tree could hold it; T1 gave a Weapon a place for
// it and T3 gives the loader a lookup to turn it into an id, so the literal
// this suite asserts against is the one the corpus actually names.
const mageStaffLiteral = "Wood Staff {castSpell=Fire_Arrow:10}"

func TestTheMageArmNamesOneLiteralWhateverSlot(t *testing.T) {
	for _, slot := range []int32{data.SkillGeneral, data.SkillBlade, data.SkillAxe,
		data.SkillBludgen, data.SkillPike, data.SkillShoot, -1, data.SkillSlots, 99} {
		got, ok := game.StartingWeaponName(true, slot)
		if !ok || got != mageStaffLiteral {
			t.Errorf("mage, slot %d = (%q, %v), want (%q, true)", slot, got, ok, mageStaffLiteral)
		}
	}
}

func TestChargenPartyResolvesTheChosenSlotsOwnWeapon(t *testing.T) {
	name, ok := game.StartingWeaponName(false, data.SkillAxe)
	if !ok {
		t.Fatal("setup: SkillAxe trains no weapon")
	}
	scale := func(rowName string, damage float64) synth.DataBinRow {
		d := make([]float64, 9)
		d[4], d[5], d[6] = damage, 1, 1
		return synth.DataBinRow{Name: rowName, Doubles: d}
	}
	var db synth.DataBin
	// Both shape words a real install carries, "Common" at the LOWER index —
	// the descending walk (FR-2a) tries "Uncommon" first and finds it, which
	// is why the chosen slot's own literal, "Uncommon Bronze Axe", leads
	// with that word and not the other. The row itself is named as an
	// install names it — "Axe" alone — never the full literal.
	db.Rows[synth.DataBinShapes] = []synth.DataBinRow{scale("Common", 0.2), scale("Uncommon", 0.4)}
	db.Rows[synth.DataBinMaterials] = []synth.DataBinRow{scale("Bronze", 1)}
	db.Rows[synth.DataBinWeapons] = []synth.DataBinRow{{
		Name:   "Axe",
		Params: []int32{-1, -1, -1, -1, -1, data.SkillAxe, 10, 20, 0, 0, -1, 1, 6, 4, -1},
	}}

	d, err := game.LoadDefinitions(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: db.Bytes()}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	front := &game.FrontEnd{InstallResources: game.InstallResources{Table: d.Table}}

	// Skill choice 1 is the SECOND warrior name in data.SkillNames(false)'s
	// own order — Axe, not the front end's own authored Blade (choice 0).
	res := ui.ChargenResult{Choices: []int{0, 0, 1}, Stats: []int{25, 25, 25, 25}}
	p := front.ChargenParty(res)
	if len(p) != 1 {
		t.Fatalf("party of %d, want 1", len(p))
	}
	if p[0].Weapon == nil || p[0].Weapon.Name != name {
		t.Fatalf("weapon = %+v, want the resolved %q — the CHOSEN slot's own, not Blade's", p[0].Weapon, name)
	}
}

// The party is one member carrying the class key its body DERIVES, the
// GENERATED hero and whatever weapon it was handed — and a fresh slice each
// time, so "every mission starts with the same party" holds by design and
// not by nobody having written to it.
func TestMissionPartyCarriesTheGeneratedHero(t *testing.T) {
	list := data.BodyList{"unarmed", "swordsman"}

	// A REAL, RESOLVABLE TABLE, and not a hand-typed Code (0134 T3):
	// GeneratedWornSet (mapload/spawn.go) substitutes the handed weapon's
	// own NAME into the weapon cell and re-resolves the whole row through
	// the table, so a fixture with no table at all cannot land anything in
	// slot 1 any more. "Placeholder" occupies row 1 deliberately —
	// HeroBodyFor's row-1 arm and its empty-slot arm answer the same list
	// entry (appearance.go's own doc), so a weapon actually at row 1 could
	// not be told apart from no weapon at all; "Iron Short Sword" is
	// written second, at row 2, so it lands on list[1], "swordsman".
	scale := func(name string, damage float64) synth.DataBinRow {
		d := make([]float64, 9)
		d[4], d[5], d[6] = damage, 1, 1
		return synth.DataBinRow{Name: name, Doubles: d}
	}
	var db synth.DataBin
	db.Rows[synth.DataBinShapes] = []synth.DataBinRow{scale("Common", 0.2)}
	db.Rows[synth.DataBinMaterials] = []synth.DataBinRow{scale("Iron", 1)}
	db.Rows[synth.DataBinWeapons] = []synth.DataBinRow{
		{Name: "Placeholder", Params: []int32{-1, -1, -1, -1, -1, data.SkillBlade, 10, 20, 0, 0, -1, 1, 6, 4, -1}},
		{Name: "Short Sword", Params: []int32{-1, -1, -1, -1, -1, data.SkillBlade, 23, 40, 0, 0, -1, 1, 9, 5, -1}},
	}
	d, err := game.LoadDefinitions(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: db.Bytes()}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	sword, err := data.ResolveWeapon("Iron Short Sword", d.Table.Shapes, d.Table.Materials, d.Table.Weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon(Iron Short Sword): %v", err)
	}

	p := game.MissionParty(&sword, list, d.Table)
	if len(p) != 1 {
		t.Fatalf("party of %d, want 1", len(p))
	}
	var eq data.Equipment
	eq.SetCode(1, sword.Code)
	wantBody, ok := data.HeroBodyFor(list, eq)
	if !ok {
		t.Fatalf("fixture: row %d derived no name", sword.Row)
	}
	want, matched := data.HeroBodyClass(wantBody)
	if !matched {
		t.Fatalf("the fixture body %q matches no arm of the name chain", wantBody)
	}
	if p[0].Class != want {
		t.Errorf("class %d, want the %q body's own %d", p[0].Class, wantBody, want)
	}
	if p[0].Body != string(wantBody) {
		t.Errorf("body %q, want the derived %q", p[0].Body, wantBody)
	}
	if p[0].Worn[0] != uint16(sword.Code) {
		t.Errorf("slot 1 = 0x%04x, want the handed sword's own 0x%04x", p[0].Worn[0], uint16(sword.Code))
	}
	if p[0].Hero != game.PartyHero() {
		t.Errorf("hero %+v, want the generated one trained in slot %d", p[0].Hero, game.PartySkillSlot())
	}
	// And he is NOT the generation start any more, which is the whole of what
	// this story changed about him.
	if p[0].Hero == data.NewCampaignHero(game.PartySkillSlot()) {
		t.Error("the party still carries the generation start")
	}
	if p[0].Weapon != &sword {
		t.Errorf("weapon %+v, want the one handed over", p[0].Weapon)
	}

	// A nil weapon is a BARE hero and is accepted, because that is a state of
	// the original rather than a caller's mistake.
	if b := game.MissionParty(nil, list, d.Table); b[0].Weapon != nil {
		t.Errorf("a nil weapon became %+v", b[0].Weapon)
	}

	p[0].Class = 999
	if game.MissionParty(&sword, list, d.Table)[0].Class != want {
		t.Error("writing the returned slice reached the next party")
	}
}

func TestMissionPartyAsResolvesTheMageArmsOwnWeapon(t *testing.T) {
	list := data.BodyList{"unarmed"}
	scale := func(name string, damage float64) synth.DataBinRow {
		d := make([]float64, 9)
		d[4], d[5], d[6] = damage, 1, 1
		return synth.DataBinRow{Name: name, Doubles: d}
	}
	var db synth.DataBin
	db.Rows[synth.DataBinShapes] = []synth.DataBinRow{scale("Common", 0.2)}
	db.Rows[synth.DataBinMaterials] = []synth.DataBinRow{scale("Wood", 1), scale("Iron", 1)}
	db.Rows[synth.DataBinWeapons] = []synth.DataBinRow{
		{Name: "Staff", Params: []int32{-1, -1, -1, -1, -1, data.SkillBlade, 10, 20, 0, 0, -1, 1, 6, 4, -1}},
		{Name: "Short Sword", Params: []int32{-1, -1, -1, -1, -1, data.SkillBlade, 23, 40, 0, 0, -1, 1, 9, 5, -1}},
	}
	d, err := game.LoadDefinitions(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: db.Bytes()}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	fighter, err := data.ResolveWeapon("Iron Short Sword", d.Table.Shapes, d.Table.Materials, d.Table.Weapons)
	if err != nil {
		t.Fatalf("setup: ResolveWeapon(Iron Short Sword): %v", err)
	}

	const wantMageName = "Wood Staff {castSpell=Fire_Arrow:10}"
	p := game.MissionPartyAs(true, &fighter, list, d.Table)
	if len(p) != 1 {
		t.Fatalf("party of %d, want 1", len(p))
	}
	if !p[0].Mage {
		t.Error("Mage = false, want true for the mage arm")
	}
	if p[0].Weapon == &fighter {
		t.Fatal("a mage was handed the fighter's own weapon object")
	}
	if p[0].Weapon == nil || p[0].Weapon.Name != wantMageName {
		t.Errorf("weapon = %+v, want the mage's own %q", p[0].Weapon, wantMageName)
	}
	if p[0].Weapon != nil && (p[0].Weapon.SpellName != "Fire_Arrow" || p[0].Weapon.SpellPower != 10) {
		t.Errorf("weapon spell = (%q, %d), want (\"Fire_Arrow\", 10) — D-13's own point",
			p[0].Weapon.SpellName, p[0].Weapon.SpellPower)
	}

	// MissionParty IS MissionPartyAs at mage false (D-11): the fighter arm
	// below takes neither of the two mage-only branches, so the two calls
	// must agree exactly, weapon object and class flag both.
	fighterParty := game.MissionParty(&fighter, list, d.Table)
	asParty := game.MissionPartyAs(false, &fighter, list, d.Table)
	if fighterParty[0].Weapon != asParty[0].Weapon || fighterParty[0].Mage != asParty[0].Mage {
		t.Errorf("MissionParty = %+v, MissionPartyAs(false, ...) = %+v, want them to agree",
			fighterParty[0], asParty[0])
	}
	if asParty[0].Weapon != &fighter {
		t.Errorf("mage false discarded the handed weapon: got %+v, want the same object", asParty[0].Weapon)
	}
}

// THE AUTHORED SPREAD IS THE RULE'S UNIQUE ANSWER, and the search is EXHAUSTIVE
// over the whole space the two bounds admit rather than a check of the places
// the answer was expected. A rule that only looked where it was pointed could
// not say "no legal spread has a higher Body"; this one can.
//
// The rule is: floor Mind and Spirit, maximise Body, then maximise Reaction with
// what is left. The last case below is why the FLOORS ARE LOAD-BEARING and not
// decoration — the two maximalities alone admit more than one spread, so a rule
// stated without its first step would not pick this build.
func TestThePartySpreadIsTheRulesUniqueAnswer(t *testing.T) {
	s := game.PartySpread()
	if !s.Legal() {
		t.Fatalf("the party's spread %+v could not be generated: it costs %d of %d",
			s, s.Cost(), data.ChargenBudget)
	}
	if s.Mind != data.StatFloor || s.Spirit != data.StatFloor {
		t.Errorf("Mind %d and Spirit %d, want both at the floor %d", s.Mind, s.Spirit, data.StatFloor)
	}

	// Pass one: the highest Body ANY legal spread reaches, over all four axes.
	maxBody := int32(0)
	each := func(f func(data.Spread)) {
		for b := data.StatFloor; b <= data.StatCeiling; b++ {
			for r := data.StatFloor; r <= data.StatCeiling; r++ {
				for m := data.StatFloor; m <= data.StatCeiling; m++ {
					for sp := data.StatFloor; sp <= data.StatCeiling; sp++ {
						c := data.Spread{Body: b, Reaction: r, Mind: m, Spirit: sp}
						if c.Legal() {
							f(c)
						}
					}
				}
			}
		}
	}
	each(func(c data.Spread) {
		if c.Body > maxBody {
			maxBody = c.Body
		}
	})
	if s.Body != maxBody {
		t.Errorf("the spread's Body is %d and the legal maximum is %d", s.Body, maxBody)
	}

	// Pass two: at that Body and both floors, the highest Reaction — and that
	// the three conditions together admit exactly this one spread.
	maxReaction, matches := int32(0), 0
	each(func(c data.Spread) {
		if c.Body != maxBody || c.Mind != data.StatFloor || c.Spirit != data.StatFloor {
			return
		}
		if c.Reaction > maxReaction {
			maxReaction = c.Reaction
		}
	})
	each(func(c data.Spread) {
		if c.Body == maxBody && c.Reaction == maxReaction &&
			c.Mind == data.StatFloor && c.Spirit == data.StatFloor {
			matches++
			if c != s {
				t.Errorf("the rule also admits %+v", c)
			}
		}
	})
	if s.Reaction != maxReaction || matches != 1 {
		t.Errorf("Reaction %d against a maximum of %d, and %d spread(s) satisfy the rule, want 1",
			s.Reaction, maxReaction, matches)
	}

	// The point that is left over is UNSPENDABLE, not unspent: it buys no step
	// on either axis a blow reads.
	if got := s.Remaining(); got != 1 {
		t.Errorf("the spread leaves %d of the budget, want 1", got)
	}
	for _, up := range []data.Spread{
		{Body: s.Body + 1, Reaction: s.Reaction, Mind: s.Mind, Spirit: s.Spirit},
		{Body: s.Body, Reaction: s.Reaction + 1, Mind: s.Mind, Spirit: s.Spirit},
	} {
		if up.Legal() {
			t.Errorf("%+v is legal, so the leftover point was merely unspent", up)
		}
	}

	// WITHOUT THE FLOORS the two maximalities do not identify a spread — which
	// is why the rule's first step is a step and not a preference.
	loose := 0
	each(func(c data.Spread) {
		if c.Body == maxBody && c.Reaction == maxReaction {
			loose++
		}
	})
	if loose < 2 {
		t.Errorf("Body-maximal and Reaction-maximal alone admit %d spread(s); "+
			"the floors are then doing no work and the rule should not state them", loose)
	}
}

// AND THE WHOLE CHAIN, END TO END: an installed table to the numbers the
// player's own unit fights at. It is the story's own claim, asserted once at
// full length — nothing between the file's bytes and the band is stubbed.
func TestAnInstalledTableBecomesTheHerosBand(t *testing.T) {
	d, err := game.LoadDefinitions(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: itemTables(true)}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	p := game.MissionParty(d.StartWeapon, nil, nil)
	c := p[0].Hero.Derive(p[0].Weapon)
	// 10-16 IS ONE OF THE OWNER'S OWN FOUR MEASURED BAND EDGES — the one he read
	// at Body 43, which is the Body the rule bought. His measurement and this
	// tree's arithmetic meet at a number neither was fitted to the other on.
	if lo, hi := c.DamageBase, c.DamageBase+c.DamageSpread; lo != 10 || hi != 16 {
		t.Fatalf("band %d-%d, want 10-16 — the owner's own reading at Body %d",
			lo, hi, game.PartySpread().Body)
	}
	// To-hit is 44 and not the install's 49 because THIS table is synthetic and
	// its row carries no to-hit column: the stat term is 14, the blade skill
	// adds 30, and the shipped file's own column adds the remaining 5.
	if c.ToHit != 44 || c.Defence != 8 {
		t.Errorf("to-hit %d, defence %d; want 44 and 8", c.ToHit, c.Defence)
	}
}

// AC-11, END TO END: a party member's ui.UnitCharacter carries the
// recompute's own experience, protections and resistances, and a unit the
// map placed carries none of them.
//
// partyCharacters ITSELF — pkg/game's own unexported seam that performs this
// conversion — is witnessed directly by TestPartyCharactersPairsTheStartsOwnSlices
// in sheet_test.go, an internal test this external file cannot be, because
// partyCharacters is unexported. This is the WHOLE CHAIN's own witness
// instead, an installed table through MissionParty to the SAME conversion
// partyCharacters performs, on TestAnInstalledTableBecomesTheHerosBand's own
// precedent for the eight combat numbers beside these three.
func TestAPartyMembersCharacterCarriesTheRecomputesFamilies(t *testing.T) {
	d, err := game.LoadDefinitions(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: itemTables(true)}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	p := game.MissionParty(d.StartWeapon, nil, nil)
	hero, weapon := p[0].Hero, p[0].Weapon
	// THE SAME CALL partyCharacters MAKES (0113 plan T3): Profile{} because a
	// generated character has no shipped row to read a class flag or a pool
	// column off, and Loadout{Weapon: weapon} because that is all this member
	// carries.
	rec := hero.Recompute(data.Profile{}, data.Loadout{Weapon: weapon})
	got := ui.UnitCharacter{Known: true, Experience: int(rec.Experience)}
	for i := range got.Protection {
		got.Protection[i], got.Resistance[i] = int(rec.Protection[i]), int(rec.Resistance[i])
	}

	if got.Experience == 0 {
		t.Error("a hero trained in one skill slot at level 10 carries zero experience")
	}
	if got.Protection == ([5]int{}) {
		t.Error("every protection is zero for a hero whose capped Spirit is above zero")
	}

	// A UNIT THE MAP PLACED has no entry in the loader's character lookup at
	// all, which states through the ZERO VALUE of ui.UnitCharacter — Known
	// false, and every field this story adds zero along with it (world.go's
	// own "AN ID WITH NO ENTRY STATES NO CHARACTER").
	var placed ui.UnitCharacter
	if placed.Known || placed.Experience != 0 || placed.Protection != ([5]int{}) || placed.Resistance != ([5]int{}) {
		t.Errorf("a placed unit's zero-value character is %+v, want every field at its zero", placed)
	}
}

// TestSetPartySkillGeneratesAnArmedHero is AC-5's whole contract: unset, the
// party hero is exactly today's hero holding today's weapon at reach 1;
// set to the shooting slot, he holds that slot's own bow — resolved off a
// SYNTHETIC weapons table, as every fixture in this file is — and his
// derived reach follows the weapon above 1.
func TestSetPartySkillGeneratesAnArmedHero(t *testing.T) {
	t.Run("unset yields today's hero exactly", func(t *testing.T) {
		d, err := game.LoadDefinitions(worldAndScenarioFS(t,
			[]synth.File{{Path: "data/data.bin", Data: itemTables(true)}},
			[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
		if err != nil {
			t.Fatalf("LoadDefinitions: %v", err)
		}
		if d.StartWeapon == nil || d.StartWeapon.Name != "Iron Short Sword" {
			t.Fatalf("StartWeapon = %+v, want today's blade", d.StartWeapon)
		}
		p := game.MissionParty(d.StartWeapon, nil, nil)
		if p[0].Hero != game.PartyHero() {
			t.Errorf("hero %+v, want today's generated hero", p[0].Hero)
		}
		c := p[0].Hero.Derive(p[0].Weapon)
		if c.Reach != 1 {
			t.Errorf("reach %d, want 1 for the unset (blade) hero", c.Reach)
		}
	})

	t.Run("set to the shooting slot yields a bow-armed hero", func(t *testing.T) {
		// partySkillSlot is PACKAGE STATE: a leaked value here would silently
		// retrain every later test's hero in this package, so this restores
		// whatever was in effect before this subtest ran.
		before := game.PartySkillSlot()
		t.Cleanup(func() { _ = game.SetPartySkill(before) })

		if err := game.SetPartySkill(data.SkillShoot); err != nil {
			t.Fatalf("SetPartySkill(SkillShoot): %v", err)
		}
		d, err := game.LoadDefinitions(worldAndScenarioFS(t,
			[]synth.File{{Path: "data/data.bin", Data: bowTables()}},
			[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
		if err != nil {
			t.Fatalf("LoadDefinitions: %v", err)
		}
		if d.StartWeaponErr != nil {
			t.Fatalf("StartWeaponErr = %v", d.StartWeaponErr)
		}
		wantName, ok := game.StartingWeaponName(false, data.SkillShoot)
		if !ok {
			t.Fatalf("fixture: the shoot slot trains no weapon by StartingWeaponName's own account")
		}
		if d.StartWeapon == nil || d.StartWeapon.Name != wantName {
			t.Fatalf("StartWeapon = %+v, want %q", d.StartWeapon, wantName)
		}

		p := game.MissionParty(d.StartWeapon, nil, nil)
		c := p[0].Hero.Derive(p[0].Weapon)
		if c.Reach <= 1 {
			t.Errorf("reach %d, want above 1 for a bow-armed hero", c.Reach)
		}
	})
}

// TestSetPartySkillRefusesAnUntrainedSlot covers the setter's own contract:
// it refuses exactly the slots StartingWeaponName itself would train no
// weapon for — out of range on either side, and the General slot which is
// in range but names nothing — and a refused call leaves the stored value
// untouched.
func TestSetPartySkillRefusesAnUntrainedSlot(t *testing.T) {
	// See the sibling test above: this mutates package state and must
	// restore it, or a later test in this package silently inherits
	// whatever slot the last refusal attempt left standing.
	before := game.PartySkillSlot()
	t.Cleanup(func() { _ = game.SetPartySkill(before) })

	if err := game.SetPartySkill(data.SkillPike); err != nil {
		t.Fatalf("SetPartySkill(SkillPike): %v", err)
	}
	trained := game.PartySkillSlot()

	for _, slot := range []int32{-1, data.SkillGeneral, data.SkillSlots, 99} {
		if err := game.SetPartySkill(slot); err == nil {
			t.Errorf("SetPartySkill(%d) = nil, want an error: this slot trains no weapon", slot)
		}
		if got := game.PartySkillSlot(); got != trained {
			t.Errorf("SetPartySkill(%d) changed the stored slot to %d, want it left at %d (the refusal must be a no-op)",
				slot, got, trained)
		}
	}
}
