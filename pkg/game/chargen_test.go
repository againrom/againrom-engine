package game

// Tests for chargen.go: ChargenSetup's own wiring of pkg/data's decoded
// arithmetic into a screen setup, ChargenParty's own construction of a
// party from a confirmed result — including its totality over a malformed
// one — and AC-7's own base-row selection reaching the party's Profile
// through this package's wiring rather than through data.ChargenBase
// directly (pkg/data/chargenbase_test.go's own job).
//
// EVERY FIXTURE IS SYNTHETIC. humansParams and fourBaseHumans build a
// Humans collection in test code, on defsearch_test.go's own "row built by
// slot number" convention (pkg/data), restated here because that file's
// own helpers are unexported a package away.

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// humansParams is a synthetic Humans-row parameter array carrying the three
// columns this story's own party assembly reads OFF A RESOLVED ROW: the
// health maximum (slot 4), the mana maximum (slot 5), and the face (slot
// 17). Every other cell is -1, the format's own "not written" sentinel
// (data.NewHumanDef's per-cell law), so the constructor's defaults stand
// for every column this file's tests do not read. The length is one past
// data's own lastHumanSlot, which is the shortest row NewHumanDef accepts —
// so it is derived from that constant rather than written out, because it
// moved once already (0127 raised the bound to 25 to read the row's own
// knownSpells column) and a literal here made four unrelated tests fail
// with a zero profile instead of a refusal anyone could read.
//
// IT CARRIES NO TYPE ID: the row's own TypeID column plays no part in
// selecting it any more — data.ChargenBase resolves a base row by NAME, at
// the archetype's own SLOT, never by decomposing a column — see
// pkg/data/chargenbase.go's own file header for the shipped values (3, 14,
// 24, 24) that made that reading unusable in the first place. A fixture
// keyed on a type id would prove nothing about the slot selection this
// package's own wiring performs; naming a row correctly is what does.
func humansParams(healthMax, manaMax, face int32) []int32 {
	p := make([]int32, data.MinHumanRow)
	for i := range p {
		p[i] = -1
	}
	p[4] = healthMax
	p[5] = manaMax
	p[17] = face
	return p
}

// fourBaseHumans is a synthetic Humans collection carrying the four
// ChargenBaseNames entries, each AT ITS OWN SLOT — the name is what places
// a row under an archetype now, so each entry is named for the slot
// data.ChargenBase will search it at rather than tagged with a type id
// nothing reads any more. The four rows carry the REAL SHIPPED figures (this
// story's own brief, corroborated by pkg/data/chargenbase.go's file header):
// PC_Danath 50/0/5, PC_Naira 20/0/1, PC_Fergard 30/70/3, PC_Reniesta 30/70/1
// — health, mana, face. Using the real numbers rather than convenient
// round ones is what makes the profile these tests check an HONEST one: a
// mage row here carries a real health column (30, not the arbitrarily zeroed
// value an earlier version of this fixture used), so a test that asserted a
// mage's HealthColumn were false would have been asserting something this
// tree's own shipped data does not say.
func fourBaseHumans() dbCollection {
	return dbCollection{
		{}, // 0: the reserved entry
		{name: "PC_Danath", params: humansParams(50, 0, 5)},    // slot 0: fighter male
		{name: "PC_Naira", params: humansParams(20, 0, 1)},     // slot 1: fighter female
		{name: "PC_Fergard", params: humansParams(30, 70, 3)},  // slot 2: mage male
		{name: "PC_Reniesta", params: humansParams(30, 70, 1)}, // slot 3: mage female
	}
}

// sameStrings is a small equality helper this file's own setup tests need
// and pkg/data's own stringSlicesEqual (chargen_test.go, a package away) is
// unexported to reach.
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestChargenSetupBuildsTheDecodedRows(t *testing.T) {
	f := &FrontEnd{}
	s := f.ChargenSetup()

	if s.Title == "" {
		t.Error("Title is empty")
	}
	if s.Confirm == "" {
		t.Error("Confirm is empty")
	}
	if s.Budget != int(data.ChargenBudget) {
		t.Errorf("Budget = %d, want data.ChargenBudget = %d", s.Budget, data.ChargenBudget)
	}

	if len(s.Choices) != 3 {
		t.Fatalf("%d choice rows, want 3 (sex, class, skill)", len(s.Choices))
	}
	sex, class, skill := s.Choices[0], s.Choices[1], s.Choices[2]
	if len(sex.Options) != 2 || sex.Parent != -1 {
		t.Errorf("sex row = %+v, want two flat options (Parent -1)", sex)
	}
	if len(class.Options) != 2 || class.Parent != -1 {
		t.Errorf("class row = %+v, want two flat options (Parent -1)", class)
	}
	if skill.Parent != 1 {
		t.Errorf("skill row's Parent = %d, want 1 (the class row's own index)", skill.Parent)
	}
	wantWarrior, wantMage := data.SkillNames(false), data.SkillNames(true)
	if len(skill.OptionsFor) != 2 ||
		!sameStrings(skill.OptionsFor[0], wantWarrior) || !sameStrings(skill.OptionsFor[1], wantMage) {
		t.Errorf("skill row's OptionsFor = %+v, want [%v, %v]", skill.OptionsFor, wantWarrior, wantMage)
	}

	// Terms "The spread": Body, Reaction, Mind, Spirit, in that display
	// order — decoded, not a taste.
	if len(s.Stats) != 4 {
		t.Fatalf("%d stat rows, want 4 (Body, Reaction, Mind, Spirit)", len(s.Stats))
	}
	wantNames := []string{"Body", "Reaction", "Mind", "Spirit"}
	for i, stat := range s.Stats {
		if stat.Name != wantNames[i] {
			t.Errorf("Stats[%d].Name = %q, want %q", i, stat.Name, wantNames[i])
		}
		if stat.Floor != int(data.StatFloor) || stat.Ceiling != int(data.StatCeiling) || stat.Start != int(data.ChargenStat) {
			t.Errorf("Stats[%d] = %+v, want Floor %d, Ceiling %d, Start %d",
				i, stat, data.StatFloor, data.StatCeiling, data.ChargenStat)
		}
	}

	if len(s.Cost) != int(data.StatCeiling)+1 {
		t.Fatalf("Cost has %d entries, want %d (0 through the ceiling inclusive)", len(s.Cost), data.StatCeiling+1)
	}
	for _, v := range []int32{0, data.StatFloor, 25, data.StatCeiling} {
		if want := int(data.PointCost(v)); s.Cost[v] != want {
			t.Errorf("Cost[%d] = %d, want data.PointCost(%d) = %d", v, s.Cost[v], v, want)
		}
	}
}

func TestChargenPreviewUsesTheConfirmedPartyProjection(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}
	result := ui.ChargenResult{Name: "Hero", Choices: []int{1, 1, 3}, Stats: []int{25, 26, 27, 28}}
	party := f.ChargenParty(result)
	if len(party) != 1 {
		t.Fatalf("ChargenParty length = %d", len(party))
	}
	preview := f.ChargenPreview(result)
	confirmed := partyPanelSubject(party[0], f.Table)
	// Both values are projections before a mission has placed the party. The
	// shared panel therefore receives the same explicit unplaced context.
	confirmed.Unplaced = true
	if got, want := preview.Subject, confirmed; got != want {
		t.Fatalf("preview subject = %+v, want %+v", got, want)
	}
	if got, want := ui.PanelStatement(ui.AuthoredPanelLayout(), preview.Subject), ui.PanelStatement(ui.AuthoredPanelLayout(), confirmed); !reflect.DeepEqual(got, want) {
		t.Fatalf("generator production-sheet statement = %q, confirmed first-sheet statement = %q", got, want)
	}
}

// TestChargenPreviewLeavesPersistentFrontEndStateAlone exercises the negative
// half of previewing. It is deliberately seeded with live and carried state:
// a preview may construct an ephemeral party projection, but must not create
// or replace a running world, party, campaign/town state, or the configured
// skill seed while a player is still editing the draft.
func TestChargenPreviewLeavesPersistentFrontEndStateAlone(t *testing.T) {
	priorSkill := PartySkillSlot()
	t.Cleanup(func() {
		if err := SetPartySkill(priorSkill); err != nil {
			t.Fatalf("restore party skill: %v", err)
		}
	})
	if err := SetPartySkill(data.SkillPike); err != nil {
		t.Fatalf("SetPartySkill: %v", err)
	}

	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}
	persistent := f.ChargenParty(ui.ChargenResult{Name: "Persist", Choices: []int{0, 0, 0}, Stats: []int{25, 25, 25, 25}})
	if len(persistent) != 1 {
		t.Fatalf("persistent fixture party length = %d", len(persistent))
	}
	live := &mapWorld{}
	f.live, f.liveMission = live, 77
	f.liveParty = mapload.CloneParty(persistent)
	f.Carried = mapload.CloneParty(persistent)
	f.Offered = 31
	f.Town = NewTown(Campaign{})
	f.Town.Arrive()
	f.Town.Won(20)

	beforeLiveParty := mapload.CloneParty(f.liveParty)
	beforeCarried := mapload.CloneParty(f.Carried)
	beforeTownOpen, beforeTownGold, beforeTownDone := f.Town.Open(), f.Town.Gold(), f.Town.Done(20)
	beforeSkill := PartySkillSlot()
	for _, result := range []ui.ChargenResult{
		{Name: "Preview", Choices: []int{0, 0, 4}, Stats: []int{25, 25, 25, 25}},
		{Name: "Mage", Choices: []int{1, 1, 2}, Stats: []int{30, 24, 26, 28}},
	} {
		if got := f.ChargenPreview(result).Subject.Name; got != result.Name {
			t.Fatalf("preview name = %q, want draft name %q", got, result.Name)
		}
	}

	if f.live != live || f.liveMission != 77 {
		t.Fatalf("preview changed live mission to world=%p mission=%d, want world=%p mission=77", f.live, f.liveMission, live)
	}
	if !reflect.DeepEqual(f.liveParty, beforeLiveParty) {
		t.Fatalf("preview changed live party: got %#v, want %#v", f.liveParty, beforeLiveParty)
	}
	if !reflect.DeepEqual(f.Carried, beforeCarried) {
		t.Fatalf("preview changed carried party: got %#v, want %#v", f.Carried, beforeCarried)
	}
	if f.Offered != 31 || f.Town.Open() != beforeTownOpen || f.Town.Gold() != beforeTownGold || f.Town.Done(20) != beforeTownDone {
		t.Fatalf("preview changed persistent campaign state: offered=%d town=(open=%v gold=%d done20=%v)",
			f.Offered, f.Town.Open(), f.Town.Gold(), f.Town.Done(20))
	}
	if got := PartySkillSlot(); got != beforeSkill {
		t.Fatalf("preview changed configured skill seed to %d, want %d", got, beforeSkill)
	}
}

// TestChargenPreviewCardCoversEveryGeneratedIdentity keeps the native Card
// honest from the player-facing side. All twenty sex/class/skill choices must
// show exactly one trained visible school and the class's actual starting
// weapon, while the card may state only the projection fields it was made for
// rather than map, worn-item, or attack-clock state.
func TestChargenPreviewCardCoversEveryGeneratedIdentity(t *testing.T) {
	weapons := dbCollection{
		{},
		{name: "Iron Short Sword", params: chargenWeaponParams(data.SkillBlade)},
		{name: "Uncommon Bronze Axe", params: chargenWeaponParams(data.SkillAxe)},
		{name: "Uncommon Bronze Mace", params: chargenWeaponParams(data.SkillBludgen)},
		{name: "Bronze Pike", params: chargenWeaponParams(data.SkillPike)},
		{name: "Uncommon Wood Short Bow", params: chargenWeaponParams(data.SkillShoot)},
		{name: "Wood Staff", params: chargenWeaponParams(data.SkillBlade)},
	}
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{Weapons: weapons}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}
	fighterWeapons := []string{"Iron Short Sword", "Uncommon Bronze Axe", "Uncommon Bronze Mace", "Bronze Pike", "Uncommon Wood Short Bow"}

	for sex := 0; sex < 2; sex++ {
		for class := 0; class < 2; class++ {
			for skill := 0; skill < 5; skill++ {
				name := fmt.Sprintf("Hero%d%d%d", sex, class, skill)
				res := ui.ChargenResult{Name: name, Choices: []int{sex, class, skill}, Stats: []int{25, 26, 27, 28}}
				party := f.ChargenParty(res)
				if len(party) != 1 || party[0].Weapon == nil {
					t.Fatalf("sex/class/skill %d/%d/%d produced party=%+v", sex, class, skill, party)
				}
				wantWeapon := fighterWeapons[skill]
				if class != 0 {
					wantWeapon = "Wood Staff {castSpell=Fire_Arrow:10}"
				}
				if got := party[0].Weapon.Name; got != wantWeapon {
					t.Errorf("sex/class/skill %d/%d/%d weapon = %q, want %q", sex, class, skill, got, wantWeapon)
				}

				preview := f.ChargenPreview(res)
				visible := 0
				for slot := data.SkillBlade; slot < data.SkillSlots; slot++ {
					got := preview.Subject.Char.Skills[slot]
					if slot == int(data.SkillBlade)+skill {
						if got != int(data.ChargenSkill) {
							t.Errorf("sex/class/skill %d/%d/%d slot %d = %d, want level %d", sex, class, skill, slot, got, data.ChargenSkill)
						} else {
							visible++
						}
					} else if got != 0 {
						t.Errorf("sex/class/skill %d/%d/%d unchosen slot %d = %d, want 0", sex, class, skill, slot, got)
					}
				}
				if visible != 1 {
					t.Errorf("sex/class/skill %d/%d/%d visible level-ten count = %d, want 1", sex, class, skill, visible)
				}

				if got := ui.PanelStatement(ui.AuthoredPanelLayout(), preview.Subject); len(got) == 0 {
					t.Errorf("sex/class/skill %d/%d/%d production card states no lines", sex, class, skill)
				}
			}
		}
	}
}

func TestChargenSetupSeedsTheSkillRowFromPartySkillSlot(t *testing.T) {
	prior := PartySkillSlot()
	t.Cleanup(func() {
		if err := SetPartySkill(prior); err != nil {
			t.Fatalf("restoring PartySkillSlot to %d: %v", prior, err)
		}
	})

	if err := SetPartySkill(data.SkillPike); err != nil {
		t.Fatalf("SetPartySkill(SkillPike): %v", err)
	}
	f := &FrontEnd{}
	s := f.ChargenSetup()
	const wantStart = int(data.SkillPike - data.SkillBlade) // position 3
	if got := s.Choices[chargenChoiceSkill].Start; got != wantStart {
		t.Errorf("skill row's Start = %d, want %d (SkillPike's own position in SkillNames)", got, wantStart)
	}
}

func TestChargenPartyResultOverridesTheSkillSeed(t *testing.T) {
	prior := PartySkillSlot()
	t.Cleanup(func() {
		if err := SetPartySkill(prior); err != nil {
			t.Fatalf("restoring PartySkillSlot to %d: %v", prior, err)
		}
	})
	if err := SetPartySkill(data.SkillPike); err != nil {
		t.Fatalf("SetPartySkill(SkillPike): %v", err)
	}

	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed")}}
	// Skill choice index 2 -- SkillBlade+2 = SkillBludgen -- CONFIRMED
	// despite the seed above naming SkillPike.
	res := ui.ChargenResult{Choices: []int{0, 0, 2}, Stats: []int{25, 25, 25, 25}}
	p := f.ChargenParty(res)
	const wantSlot = data.SkillBludgen
	if p[0].Hero.Skill[wantSlot] != data.ChargenSkill {
		t.Errorf("trained level at slot %d (Bludgen) = %d, want %d -- the CONFIRMED choice, not the seed",
			wantSlot, p[0].Hero.Skill[wantSlot], data.ChargenSkill)
	}
	if p[0].Hero.Skill[data.SkillPike] != 0 {
		t.Errorf("slot Pike trained at %d, want 0 -- the seed must not itself train anything", p[0].Hero.Skill[data.SkillPike])
	}
}

func TestChargenPartyBuildsAPlayableCharacterFromAConfirmedResult(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}
	// Female (index 1), mage (index 1), skill index 2 — the THIRD warrior/mage
	// name, which is SkillBlade+2 = SkillBludgen.
	res := ui.ChargenResult{
		Choices: []int{1, 1, 2},
		Stats:   []int{30, 20, 18, 16},
	}
	p := f.ChargenParty(res)
	if len(p) != 1 {
		t.Fatalf("party of %d, want 1", len(p))
	}
	m := p[0]

	if m.Hero.Body != 30 || m.Hero.Reaction != 20 || m.Hero.Mind != 18 || m.Hero.Spirit != 16 {
		t.Errorf("spread = %d/%d/%d/%d, want 30/20/18/16", m.Hero.Body, m.Hero.Reaction, m.Hero.Mind, m.Hero.Spirit)
	}
	const wantSlot = data.SkillBludgen // SkillBlade + 2
	if m.Hero.Skill[wantSlot] != data.ChargenSkill {
		t.Errorf("trained level at slot %d = %d, want %d", wantSlot, m.Hero.Skill[wantSlot], data.ChargenSkill)
	}
	for slot := int32(0); slot < data.SkillSlots; slot++ {
		if slot == wantSlot {
			continue
		}
		if m.Hero.Skill[slot] != 0 {
			t.Errorf("slot %d trained at %d, want 0 — only the chosen slot is trained", slot, m.Hero.Skill[slot])
		}
	}

	// PC_Reniesta is the row at slot 3 (class=true, female=true) —
	// fourBaseHumans' own mage-female entry, 30/70/1. Its ManaMax (70) is
	// nonzero, so Profile.Fighter is false (HumanDef.Profile's own rule:
	// Fighter is set for a row carrying NO mana column) and its HealthMax
	// (30) is nonzero too, so HealthColumn is true — the real shipped
	// figures, not the arbitrarily zeroed health an earlier fixture gave
	// every mage row.
	wantProfile := data.Profile{Fighter: false, HealthColumn: true, ManaColumn: true}
	if m.Profile != wantProfile {
		t.Errorf("profile = %+v, want PC_Reniesta's own %+v", m.Profile, wantProfile)
	}
	if m.FigureFace != 1 {
		t.Errorf("FigureFace = %d, want PC_Reniesta's own face, 1", m.FigureFace)
	}
	if m.FigureDir != string(data.FigureDirWomanMage) {
		t.Errorf("FigureDir = %q, want %q — the CHOSEN pair's directory", m.FigureDir, data.FigureDirWomanMage)
	}
}

// TestChargenPartyIsTotalOverAMalformedResult is the fallback this file's
// own doc promises: a ui.ChargenResult shorter than the setup it was
// supposedly confirmed against, and one carrying values far outside any
// option's range, must both build a playable party rather than panic.
//
// THE EMPTY RESULT reads every choice as its first option (male, fighter,
// the first skill) and every statistic as data.ChargenStat — chargenChoiceIndex
// and chargenStatValue's own documented fallback (chargen.go) — so it is not
// merely "does not crash" but "answers exactly the screen's own opening
// state", which is a total function's stronger and more useful property.
func TestChargenPartyIsTotalOverAMalformedResult(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed")}}

	t.Run("empty result", func(t *testing.T) {
		p := f.ChargenParty(ui.ChargenResult{})
		if len(p) != 1 {
			t.Fatalf("party of %d, want 1", len(p))
		}
		m := p[0]
		if m.Hero.Body != data.ChargenStat || m.Hero.Reaction != data.ChargenStat ||
			m.Hero.Mind != data.ChargenStat || m.Hero.Spirit != data.ChargenStat {
			t.Errorf("spread = %+v, want every statistic at data.ChargenStat (%d)", m.Hero, data.ChargenStat)
		}
		if m.Hero.Skill[data.SkillBlade] != data.ChargenSkill {
			t.Errorf("trained level at Blade (skill choice 0) = %d, want %d",
				m.Hero.Skill[data.SkillBlade], data.ChargenSkill)
		}
		// Choice 0 for both sex and class: male, fighter — slot 0, PC_Danath's
		// own row among fourBaseHumans (50/0/5). ManaMax is 0, so Fighter is
		// TRUE (HumanDef.Profile's own rule sets the flag for a row with NO
		// mana column) and HealthMax (50) is nonzero, so HealthColumn is true.
		if want := (data.Profile{Fighter: true, HealthColumn: true, ManaColumn: false}); m.Profile != want {
			t.Errorf("profile = %+v, want PC_Danath's own %+v", m.Profile, want)
		}
	})

	t.Run("choices and stats out of any option's range", func(t *testing.T) {
		// Neither slice is SHORT — chargenChoiceIndex's own length guard is
		// not what is exercised here — but every value is nonsense: a skill
		// choice with no fifth option, statistics far outside [15, 45].
		res := ui.ChargenResult{
			Choices: []int{99, -7, 12345},
			Stats:   []int{-500, 99999, 0, -1},
		}
		p := f.ChargenParty(res) // must not panic
		if len(p) != 1 {
			t.Fatalf("party of %d, want 1", len(p))
		}
		m := p[0]
		if m.Hero.Body != -500 || m.Hero.Reaction != 99999 || m.Hero.Mind != 0 || m.Hero.Spirit != -1 {
			t.Errorf("spread = %+v, want the out-of-range values carried through verbatim "+
				"(data.NewHero does not validate a spread; that is Spread.Legal's own question)", m.Hero)
			// A malformed value is not this file's to reject a second time —
			// see chargenChoiceIndex and chargenStatValue's own doc.
		}
		// slot = SkillBlade + 12345 trains nothing (data.NewHero's own
		// bounds), and the front end's own weapon resolution answers no
		// weapon for it either — both total, neither a panic.
		if m.Weapon != nil {
			t.Errorf("weapon = %+v, want none for an out-of-range slot", m.Weapon)
		}
		for slot := int32(0); slot < data.SkillSlots; slot++ {
			if m.Hero.Skill[slot] != 0 {
				t.Errorf("slot %d trained at %d, want 0 — the out-of-range choice trains nothing", slot, m.Hero.Skill[slot])
			}
		}
	})

	t.Run("fewer choices and stats than the setup declares", func(t *testing.T) {
		res := ui.ChargenResult{Choices: []int{1}, Stats: []int{40}} // sex alone, Body alone
		p := f.ChargenParty(res)                                     // must not panic
		if len(p) != 1 {
			t.Fatalf("party of %d, want 1", len(p))
		}
		m := p[0]
		if m.Hero.Body != 40 {
			t.Errorf("Body = %d, want the one stated value, 40", m.Hero.Body)
		}
		if m.Hero.Reaction != data.ChargenStat || m.Hero.Mind != data.ChargenStat || m.Hero.Spirit != data.ChargenStat {
			t.Errorf("the three unstated statistics = %d/%d/%d, want all at data.ChargenStat (%d)",
				m.Hero.Reaction, m.Hero.Mind, m.Hero.Spirit, data.ChargenStat)
		}
		// Class and skill are both unstated — read as choice 0 (fighter,
		// Blade) — but sex IS stated, at 1 (female). Slot 1 is PC_Naira's own
		// row (20/0/1): ManaMax 0 sets Fighter true, HealthMax 20 sets
		// HealthColumn true.
		if want := (data.Profile{Fighter: true, HealthColumn: true, ManaColumn: false}); m.Profile != want {
			t.Errorf("profile = %+v, want PC_Naira's own %+v", m.Profile, want)
		}
	})
}

// TestAC7BaseRowSelectionReachesThePartysProfile is AC-7's own four cases,
// exercised through THIS package's wiring (ChargenParty) rather than
// re-testing data.ChargenBase's own selection a second time — that is
// pkg/data/chargenbase_test.go's job. What this proves is that a chosen sex
// and class actually reach data.ChargenBase through f.Humans, and that its
// answer actually reaches the party's own Profile, FigureFace and
// FigureDir, through ChargenParty's one call.
func TestAC7BaseRowSelectionReachesThePartysProfile(t *testing.T) {
	resultFor := func(female, class bool) ui.ChargenResult {
		sex, cls := 0, 0
		if female {
			sex = 1
		}
		if class {
			cls = 1
		}
		// Skill and statistics are irrelevant to this test; left at the
		// screen's own start.
		return ui.ChargenResult{Choices: []int{sex, cls, 0}, Stats: []int{25, 25, 25, 25}}
	}

	t.Run("matching pair", func(t *testing.T) {
		f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans()}}
		for _, tc := range []struct {
			female, class                     bool
			wantFighter, wantHealth, wantMana bool
		}{
			{false, false, true, true, false}, // PC_Danath  (fighter male)
			{true, false, true, true, false},  // PC_Naira   (fighter female)
			{false, true, false, true, true},  // PC_Fergard (mage male)
			{true, true, false, true, true},   // PC_Reniesta(mage female)
		} {
			p := f.ChargenParty(resultFor(tc.female, tc.class))
			got := p[0].Profile
			want := data.Profile{Fighter: tc.wantFighter, HealthColumn: tc.wantHealth, ManaColumn: tc.wantMana}
			if got != want {
				t.Errorf("female=%v class=%v: profile = %+v, want %+v", tc.female, tc.class, got, want)
			}
		}
	})

	t.Run("same slot, first in order wins", func(t *testing.T) {
		// A DUPLICATE NAME is what "the same pair" means now — the archetype is
		// the slot a NAME occupies, not a column a type id decomposes, so two rows
		// can only collide on the request below by sharing the name
		// FindHumanByName is asked to resolve, ascending from index 1.
		humans := dbCollection{
			{},
			{name: "PC_Danath", params: humansParams(40, 0, 1)},
			{name: "PC_Naira", params: humansParams(41, 0, 2)},
			{name: "PC_Danath", params: humansParams(42, 0, 3)}, // DUPLICATE name — later, must lose
			{name: "PC_Reniesta", params: humansParams(43, 0, 4)},
		}
		f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: humans}}
		p := f.ChargenParty(resultFor(false, false))
		if !p[0].Profile.HealthColumn || p[0].FigureFace != 1 {
			t.Errorf("profile/face = %+v/%d, want the FIRST PC_Danath entry (health column, face 1) — "+
				"not the later duplicate at face 3", p[0].Profile, p[0].FigureFace)
		}
	})

	t.Run("none matches, first that resolved wins", func(t *testing.T) {
		// Neither PC_Fergard nor PC_Reniesta — slots 2 and 3 — is present, so
		// the request below (class=true, female=true, slot 3) falls back to
		// the first of the four NAMES that resolves, in ChargenBaseNames' own
		// order: PC_Danath, at index 1.
		humans := dbCollection{
			{},
			{name: "PC_Danath", params: humansParams(40, 0, 1)},
			{name: "PC_Naira", params: humansParams(41, 0, 2)},
		}
		f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: humans}}
		p := f.ChargenParty(resultFor(true, true)) // wants slot 3, PC_Reniesta — absent
		if !p[0].Profile.HealthColumn || p[0].FigureFace != 1 {
			t.Errorf("profile/face = %+v/%d, want PC_Danath's own — the first NAME that resolved, "+
				"in ChargenBaseNames' own order", p[0].Profile, p[0].FigureFace)
		}
		if p[0].FigureDir != string(data.FigureDirWomanMage) {
			t.Errorf("FigureDir = %q, want %q — the player's own choice", p[0].FigureDir, data.FigureDirWomanMage)
		}
	})

	t.Run("none resolves, requested archetype's fighter bit", func(t *testing.T) {
		// DIV-1389: chargenProfile's own fallback used to return Go's zero
		// Profile regardless of which archetype (class) the caller asked
		// ChargenBase for, silently naming every unresolvable request a
		// mage. class=false here requests the fighter axis, so the fallback
		// now keeps Fighter=true. HealthColumn now reads true here too: the
		// health graph needs only Body and the class bit, both already
		// resolved on this arm, and ManaColumn stays false because this
		// story's scope is the health maximum alone.
		for _, humans := range []dbCollection{nil, {}, {{name: "Someone Else", params: humansParams(1, 1, 9)}}} {
			f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: humans}}
			p := f.ChargenParty(resultFor(false, false))
			want := data.Profile{Fighter: true, HealthColumn: true}
			if p[0].Profile != want {
				t.Errorf("humans=%v: profile = %+v, want %+v", humans, p[0].Profile, want)
			}
			if p[0].FigureFace != 1 {
				t.Errorf("humans=%v: FigureFace = %d, want 1", humans, p[0].FigureFace)
			}
		}
	})
}

// TestAC1TheNoFlagPartyMatchesTodaysHeroLiterally pins MissionParty's
// no-flag output against the values THIS TREE SHIPPED before 0119-chargen —
// written out here rather than reread from PartySpread/PartySkillSlot/
// PartyHero, which would let this test pass even if both this file and
// hero.go had drifted together (spec AC-1).
func TestAC1TheNoFlagPartyMatchesTodaysHeroLiterally(t *testing.T) {
	humans := dbCollection{
		{},
		{name: "PC_Danath", params: humansParams(50, 0, 5)},
	}
	// A REAL, RESOLVABLE TABLE, and not a hand-typed Code (0134 T3):
	// GeneratedWornSet (mapload/spawn.go) substitutes the handed weapon's
	// own NAME into the base row's weapon cell and re-resolves the whole
	// row through the table, so a fixture with no table cannot land
	// anything in slot 1 any more. "Placeholder" occupies row 1
	// deliberately — HeroBodyFor's row-1 arm and its empty-slot arm answer
	// the same list entry (appearance.go's own doc), so a weapon actually
	// at row 1 could not be told apart from no weapon at all.
	weapons := dbCollection{
		{},
		{name: "Placeholder", params: chargenWeaponParams(data.SkillBlade)},
		{name: "Iron Short Sword", params: chargenWeaponParams(data.SkillBlade)},
	}
	tbl := &mapload.Table{Humans: humans, Weapons: weapons, Shapes: emptyScale{}, Materials: emptyScale{}}

	const wantWeaponName = "Iron Short Sword" // hero.go's own startingWeapons[data.SkillBlade]
	if got, ok := StartingWeaponName(false, data.SkillBlade); !ok || got != wantWeaponName {
		t.Fatalf("setup: StartingWeaponName(false, SkillBlade) = (%q, %v), want (%q, true)", got, ok, wantWeaponName)
	}
	sword, err := data.ResolveWeapon(wantWeaponName, tbl.Shapes, tbl.Materials, tbl.Weapons)
	if err != nil {
		t.Fatalf("setup: ResolveWeapon(%q): %v", wantWeaponName, err)
	}
	list := data.NewBodyList("unarmed", "swordsman")

	p := MissionParty(&sword, list, tbl)
	if len(p) != 1 {
		t.Fatalf("party of %d, want 1", len(p))
	}
	m := p[0]

	if m.Hero.Body != 43 || m.Hero.Reaction != 26 || m.Hero.Mind != 15 || m.Hero.Spirit != 15 {
		t.Errorf("spread = %d/%d/%d/%d, want 43/26/15/15", m.Hero.Body, m.Hero.Reaction, m.Hero.Mind, m.Hero.Spirit)
	}
	if m.Hero.Skill[data.SkillBlade] != 10 {
		t.Errorf("trained level at Blade = %d, want 10", m.Hero.Skill[data.SkillBlade])
	}
	for slot := int32(0); slot < data.SkillSlots; slot++ {
		if slot == data.SkillBlade {
			continue
		}
		if m.Hero.Skill[slot] != 0 {
			t.Errorf("slot %d trained at %d, want 0 — only Blade is trained", slot, m.Hero.Skill[slot])
		}
	}
	if m.Weapon != &sword || m.Weapon.Name != wantWeaponName {
		t.Errorf("weapon = %+v, want the handed sword named %q", m.Weapon, wantWeaponName)
	}

	var eq data.Equipment
	eq.SetCode(1, sword.Code)
	wantBody, ok := data.HeroBodyFor(list, eq)
	if !ok || wantBody != "swordsman" {
		t.Fatalf("fixture: row %d derived (%q, %v), want (swordsman, true)", sword.Row, wantBody, ok)
	}
	if m.Body != string(wantBody) {
		t.Errorf("body = %q, want %q", m.Body, wantBody)
	}
	if m.Worn[0] != uint16(sword.Code) {
		t.Errorf("slot 1 = 0x%04x, want the handed sword's own 0x%04x", m.Worn[0], uint16(sword.Code))
	}
	wantClass, matched := data.HeroBodyClass(wantBody)
	if !matched {
		t.Fatalf("fixture: %q matches no arm of the class chain", wantBody)
	}
	if m.Class != wantClass {
		t.Errorf("class = %d, want %d", m.Class, wantClass)
	}

	if m.FigureDir != string(data.FigureDirManFighter) {
		t.Errorf("FigureDir = %q, want %q", m.FigureDir, data.FigureDirManFighter)
	}
	if m.FigureFace != 5 {
		t.Errorf("FigureFace = %d, want 5 — PC_Danath's own shipped Face", m.FigureFace)
	}

	if !m.Profile.HealthColumn {
		t.Errorf("Profile.HealthColumn = false, want true — PC_Danath's own HealthMax (50) is nonzero")
	}
	if m.Profile.ManaColumn {
		t.Error("Profile.ManaColumn = true, want false — this fixture's ManaMax is 0")
	}
	if !m.Profile.Fighter {
		t.Error("Profile.Fighter = false, want true — PC_Danath's own ManaMax (0) sets the flag " +
			"(HumanDef.Profile's own rule: Fighter is set for a row carrying NO mana column)")
	}
}

// TestMissionPartyWithNoHumansCollectionCarriesTheRequestedArchetype covers
// the exact reproduction path DIV-1389 traced the owner's game9248/game9249
// kit to: MissionParty(nil, nil, nil) reaches chargenProfile's "no base row
// resolves" fallback (missionHumans returns an empty Collection for a nil
// *mapload.Table), which used to always return Go's zero Profile — Fighter
// false — even though FigureDir here already names the fighter direction.
// The fallback now keeps the requested archetype's own Fighter bit AND now
// reports HealthColumn true, which is this story's own fix: the owner's
// witnessed constant 100 (docs/1224, docs/1225) traced to this exact Profile
// still gating PartySpawn's health arm shut even though the graph had
// everything it needed.
func TestMissionPartyWithNoHumansCollectionCarriesTheRequestedArchetype(t *testing.T) {
	p := MissionParty(nil, data.BodyList{}, nil)
	if len(p) != 1 {
		t.Fatalf("party of %d, want 1", len(p))
	}
	m := p[0]
	want := data.Profile{Fighter: true, HealthColumn: true}
	if m.Profile != want {
		t.Errorf("Profile = %+v, want %+v", m.Profile, want)
	}
	if m.FigureDir != string(data.FigureDirManFighter) {
		t.Errorf("FigureDir = %q, want %q", m.FigureDir, data.FigureDirManFighter)
	}
	if m.FigureFace != 1 {
		t.Errorf("FigureFace = %d, want 1", m.FigureFace)
	}
}

func TestChargenDerivedShowsTheMintsOwnNumbers(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}
	res := ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{30, 20, 18, 16}}

	got := f.ChargenDerived(res)
	// The eight authored labels in their declared order, then the five skill
	// names of the CHOSEN class — res picks Mage above, so the tail is the mage
	// set. The tail is read from data.SkillNames rather than written out, so
	// this pins the ORDER and the class selection without restating the five
	// strings pkg/data already owns.
	wantNames := append([]string{
		chargenDerivedHealth, chargenDerivedMana, chargenDerivedToHit,
		chargenDerivedDefence, chargenDerivedDamage, chargenDerivedSpeed,
		chargenDerivedSight, chargenDerivedMagicRes,
	}, data.SkillNames(true)...)
	if len(got) != len(wantNames) {
		t.Fatalf("ChargenDerived reported %d lines, want %d", len(got), len(wantNames))
	}
	for i, want := range wantNames {
		if got[i].Name != want {
			t.Errorf("line %d is %q, want %q — the order is the block's contract", i, got[i].Name, want)
		}
	}

	// THE CROSS-CHECK, and the whole reason PartySpawn is exported: the
	// preview's numbers are the mint's numbers, for the same member.
	d, health, mana := mapload.PartySpawn(f.ChargenParty(res)[0])
	want := map[string]string{
		chargenDerivedHealth:  strconv.FormatInt(int64(health), 10),
		chargenDerivedMana:    strconv.FormatInt(int64(mana), 10),
		chargenDerivedToHit:   strconv.FormatInt(int64(d.Combat.ToHit), 10),
		chargenDerivedDefence: strconv.FormatInt(int64(d.Combat.Defence), 10),
		chargenDerivedSpeed:   strconv.FormatInt(int64(d.Speed), 10),
		chargenDerivedSight:   strconv.FormatInt(int64(d.Sight), 10),
		chargenDerivedDamage: fmt.Sprintf("%d-%d",
			d.Combat.DamageBase, d.Combat.DamageBase+d.Combat.DamageSpread),
		chargenDerivedMagicRes: fmt.Sprintf("Fire %d / Water %d / Air %d / Earth %d / Astral %d",
			d.Protection[0], d.Protection[1], d.Protection[2],
			d.Protection[3], d.Protection[4]),
	}
	// The five skill lines, against the mint's own restored levels. The magic
	// resistance row is written out longhand above rather than through
	// namedRow(), so the separator, the five names and the column order are
	// pinned by this test rather than by the very function it is checking.
	for i, name := range data.SkillNames(true) {
		want[name] = strconv.FormatInt(int64(d.Skill[data.SkillBlade+int32(i)]), 10)
	}
	for _, line := range got {
		if line.Value != want[line.Name] {
			t.Errorf("%s = %q, want the mint's own %q", line.Name, line.Value, want[line.Name])
		}
	}
	if health == mapload.SpawnHP {
		t.Errorf("health = %d, which is SpawnHP — this fixture's row carries a health column, "+
			"so the profile gate must have taken the recompute's maximum instead", health)
	}
	if mana == 0 {
		t.Error("mana = 0 for a row carrying a mana column; the mana gate is not on its true arm")
	}
}

// A mage sees only the effective spell interval of a staff attack on the
// owner-facing generation screen. It is cross-checked against
// World.WeaponSpellDamage, the live attack reader, over the same generated
// entity and installed row; the unused physical pair is not presented.
func TestChargenDerivedUsesTheLiveStaffSpellAsDamage(t *testing.T) {
	weapons := dbCollection{
		{},
		{name: "Wood Staff", params: chargenWeaponParams(data.SkillBlade)},
	}
	spell := make([]int32, 19)
	spell[1], spell[2], spell[4], spell[6] = 3, 1, 1, 7
	spell[16], spell[17] = 15, 18
	spells := dbCollection{{}, {name: "Fire Arrow", params: spell}}
	humans := fourBaseHumans()
	table := &mapload.Table{Humans: humans, Weapons: weapons, Spells: spells,
		Shapes: emptyScale{}, Materials: emptyScale{}}
	f := &FrontEnd{InstallResources: InstallResources{Table: table, Humans: humans, Bodies: data.NewBodyList("unarmed", "mage")}}
	res := ui.ChargenResult{Choices: []int{0, 1, 0}, Stats: []int{25, 25, 25, 25}}

	lines := make(map[string]string)
	for _, line := range f.ChargenDerived(res) {
		lines[line.Name] = line.Value
	}
	if lines[chargenDerivedDamage] != "20-24" {
		t.Fatalf("generation Damage = %q, want staff spell interval 20-24", lines[chargenDerivedDamage])
	}

	member := f.ChargenParty(res)[0]
	d, _, mana := mapload.PartySpawn(member)
	spellID, ok := mapload.SpellIDByToken(table, d.Combat.SpellName)
	if !ok {
		t.Fatal("fixture weapon spell did not resolve")
	}
	e := sim.Entity{ID: 7, MaxMana: mana, WeaponSpell: spellID, WeaponSpellLevel: d.Combat.SpellPower}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical, nil,
		[]sim.Entity{e}, nil, mapload.SpellRules(table))
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	base, spread, ok := w.WeaponSpellDamage(7)
	if !ok {
		t.Fatal("live world refused the generated staff spell")
	}
	want := fmt.Sprintf("%d-%d", base, base+spread)
	if got := lines[chargenDerivedDamage]; got != want {
		t.Errorf("generation Damage = %q, live attack reader = %q", got, want)
	}
}

// TestChargenDerivedMovesWithTheSpread is the owner's own request stated as
// a property: raising Body raises what the block reports for health, so the
// player can see what the point he just spent bought him.
func TestChargenDerivedMovesWithTheSpread(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}
	low := f.ChargenDerived(ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{20, 20, 18, 16}})
	high := f.ChargenDerived(ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{40, 20, 18, 16}})

	if low[0].Name != chargenDerivedHealth || high[0].Name != chargenDerivedHealth {
		t.Fatalf("the first line is %q/%q, want %q", low[0].Name, high[0].Name, chargenDerivedHealth)
	}
	if low[0].Value == high[0].Value {
		t.Errorf("health reads %q at Body 20 and %q at Body 40 — a block that does not move "+
			"tells the player nothing about what he is spending", low[0].Value, high[0].Value)
	}
}

// TestChargenDerivedIsTotalOverAMalformedResult is the same totality
// ChargenParty already promises, read from the consequence block's side: a
// short result must not panic, because this is called on every frame the
// screen paints and a screen that crashes is an incident.
func TestChargenDerivedIsTotalOverAMalformedResult(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed")}}
	// Eight authored labels plus the five skill slots either class names. A
	// malformed result still answers the WHOLE block — the skill labels come
	// from a class choice read through chargenChoiceIndex's own fallback, so a
	// result with no choices at all is read as the fighter set rather than as
	// no set (0140).
	const want = 8 + data.SkillSlots - 1
	for _, res := range []ui.ChargenResult{
		{},
		{Choices: []int{0}},
		{Stats: []int{1}},
		{Choices: []int{9, 9, 9}, Stats: []int{-400, 900, 0, 0}},
	} {
		if got := f.ChargenDerived(res); len(got) != want {
			t.Errorf("ChargenDerived(%+v) reported %d lines, want %d — a malformed result is read "+
				"through the same fallbacks a confirmed one is, never refused", res, len(got), want)
		}
	}
}

// TestTheChosenBaseRowsBookReachesTheParty is 0127 FR-4a on this side of the
// seam: which spells a generated character starts knowing is a column of the
// base row he was generated from — chargenProfile's own third return — and not
// something this package chooses. The two archetypes are asserted against each
// other rather than one in isolation, because a wiring that read the same row
// for both, or read none, passes any single-case check.
//
// The masks are the shipped ones. PC_Fergard and PC_Reniesta both carry 266306
// (bits 1, 6, 12, 18); PC_Danath and PC_Naira carry the -1 empty cell, which
// data.NewHumanDef lands as an empty book.
func TestTheChosenBaseRowsBookReachesTheParty(t *testing.T) {
	humans := dbCollection{
		{},
		{name: "PC_Danath", params: humansParamsBook(50, 0, 5, -1)},
		{name: "PC_Naira", params: humansParamsBook(20, 0, 1, -1)},
		{name: "PC_Fergard", params: humansParamsBook(30, 70, 3, 266306)},
		{name: "PC_Reniesta", params: humansParamsBook(30, 70, 1, 266306)},
	}
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: humans, Bodies: data.NewBodyList("unarmed", "mage")}}

	for _, tc := range []struct {
		name  string
		class int // 0 fighter, 1 mage
		want  uint32
	}{
		{"a fighter's base row states no spells", 0, 0},
		{"a mage's base row states four", 1, 266306},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := f.ChargenParty(ui.ChargenResult{Choices: []int{0, tc.class, 0}, Stats: []int{30, 20, 18, 16}})
			if len(p) != 1 {
				t.Fatalf("party of %d, want 1", len(p))
			}
			if p[0].KnownSpells != tc.want {
				t.Errorf("KnownSpells = %d, want %d", p[0].KnownSpells, tc.want)
			}
		})
	}
}

// humansParamsBook is humansParams with the knownSpells column named too. It
// is separate rather than a fourth parameter on humansParams because every
// other test in this file is about columns this one does not read, and a
// signature change there would touch all of them to say nothing.
func humansParamsBook(healthMax, manaMax, face, book int32) []int32 {
	p := humansParams(healthMax, manaMax, face)
	p[25] = book
	return p
}

// chargenWeaponParams is a synthetic Weapons-row parameter array carrying
// only what data.ResolveWeapon reads: the attack type at slot 5 and a
// physical band, a to-hit and a defence past it, none of which any test in
// this file below checks — the fixtures below only need each named row to
// RESOLVE, not to arm anyone with particular numbers.
func chargenWeaponParams(attackType int32) []int32 {
	return []int32{-1, -1, -1, -1, -1, attackType, 10, 20, 0, 0, -1, 1, 6, 4, -1, eqSuitAny}
}

// emptyScale is a data.ScaleTable with no entries — this file's own
// stand-in for a shape or a material table its fixtures never name a word
// of. A real ScaleTable value (as opposed to a nil one) is what
// (*mapload.Table).items() and its two siblings require before they will
// resolve a weapon, a shield or an armour at all (spawn.go's own "the
// three go together" rule): passing nil for Shapes or Materials answers
// every one of the three false, so a fixture that wants a real resolution
// but has no shape or material word to spend needs a present, if hollow,
// table rather than an absent one.
type emptyScale struct{}

func (emptyScale) Len() int                   { return 0 }
func (emptyScale) EntryName(int) string       { return "" }
func (emptyScale) EntryDoubles(int) []float64 { return nil }

func TestAC3TheHandedWeaponDisplacesTheBaseRowsOwnWeaponCell(t *testing.T) {
	weapons := dbCollection{
		{},
		{name: "RowWeapon", params: chargenWeaponParams(data.SkillBlade)},
		{name: "HandedWeapon", params: chargenWeaponParams(data.SkillAxe)},
	}
	armors := dbCollection{
		{}, // 0: reserved
		{name: "Robe", params: []int32{-1, -1, -1, -1, 3}}, // slot 3 (armorSlotColumn = 4)
	}
	handed, err := data.ResolveWeapon("HandedWeapon", emptyScale{}, emptyScale{}, weapons)
	if err != nil {
		t.Fatalf("setup: ResolveWeapon(HandedWeapon): %v", err)
	}
	wantRobe, err := data.ResolveArmor("Robe", emptyScale{}, emptyScale{}, armors)
	if err != nil {
		t.Fatalf("setup: ResolveArmor(Robe): %v", err)
	}

	cells := []string{"RowWeapon", "", "Robe", "", "NoSuchArmour", "", "", "", "", ""}
	humans := dbCollection{
		{},
		{name: "PC_Danath", params: humansParams(50, 0, 5), strings: cells},
	}
	tbl := &mapload.Table{Humans: humans, Weapons: weapons, Armors: armors, Shapes: emptyScale{}, Materials: emptyScale{}}

	p := MissionParty(&handed, data.NewBodyList("unarmed"), tbl)
	if len(p) != 1 {
		t.Fatalf("party of %d, want 1", len(p))
	}
	m := p[0]

	if m.Worn[0] != uint16(handed.Code) {
		t.Errorf("slot 1 = 0x%04x, want the HANDED weapon's own 0x%04x, not the row's RowWeapon", m.Worn[0], uint16(handed.Code))
	}
	if m.Weapon == nil || m.Weapon.Code != handed.Code {
		t.Errorf("Weapon = %+v, want the same resolution slot 1 carries (P-6)", m.Weapon)
	}
	if m.Worn[2] != uint16(wantRobe.Code) {
		t.Errorf("slot 3 = 0x%04x, want the row's own Robe, 0x%04x — AC-15's resolvable cell must still land",
			m.Worn[2], uint16(wantRobe.Code))
	}
	for i, code := range m.Worn {
		if i == 0 || i == 2 {
			continue
		}
		if code != 0 {
			t.Errorf("slot %d = 0x%04x, want empty — the empty cell and the unresolvable "+
				"NoSuchArmour must cost nothing (AC-15)", i+1, code)
		}
	}
	rest := make([]uint16, 0, len(m.Carried))
	for _, code := range m.Carried {
		if code == uint16(data.QuestDocumentCode) {
			continue
		}
		rest = append(rest, code)
	}
	if len(rest) != 0 {
		t.Errorf("Carried = %v, want nothing but the documents access item — nothing in this "+
			"row's cells names a piece with no slot among the twelve, so the worn resolution "+
			"must put nothing in the pack (AC-15's other half)", m.Carried)
	}
	if len(rest) == len(m.Carried) {
		t.Errorf("Carried = %v, want it to hold the documents access item 0x%04x (1035 B5)",
			m.Carried, uint16(data.QuestDocumentCode))
	}
}

func TestAC4AMageHoldsTheMagesWeaponForEverySkillChoice(t *testing.T) {
	weapons := dbCollection{
		{},
		{name: "Wood Staff", params: chargenWeaponParams(data.SkillBlade)},
	}
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{Weapons: weapons, Shapes: emptyScale{}, Materials: emptyScale{}}, Bodies: data.NewBodyList("unarmed")}}
	const wantName = "Wood Staff {castSpell=Fire_Arrow:10}"
	for skill := 0; skill < 5; skill++ {
		res := ui.ChargenResult{Choices: []int{0, 1, skill}, Stats: []int{25, 25, 25, 25}}
		p := f.ChargenParty(res)
		if len(p) != 1 {
			t.Fatalf("skill choice %d: party of %d, want 1", skill, len(p))
		}
		if !p[0].Mage {
			t.Errorf("skill choice %d: Mage = false, want true for the chosen class", skill)
		}
		if p[0].Weapon == nil || p[0].Weapon.Name != wantName {
			t.Errorf("skill choice %d: weapon = %+v, want the mage's own %q", skill, p[0].Weapon, wantName)
		}
		if p[0].Weapon != nil && (p[0].Weapon.SpellName != "Fire_Arrow" || p[0].Weapon.SpellPower != 10) {
			t.Errorf("skill choice %d: weapon spell = (%q, %d), want (\"Fire_Arrow\", 10) — D-13's own point",
				skill, p[0].Weapon.SpellName, p[0].Weapon.SpellPower)
		}
	}
}

func TestAC14NoEquipIsRefusedForAClass(t *testing.T) {
	armors := dbCollection{
		{},
		{name: "Robe", params: []int32{-1, -1, -1, -1, 3}}, // slot 3
	}
	robeCells := []string{"", "", "Robe", "", "", "", "", "", "", ""}
	humans := dbCollection{
		{},
		{name: "PC_Danath", params: humansParams(50, 0, 5), strings: robeCells},   // fighter male
		{name: "PC_Naira", params: humansParams(20, 0, 1)},                        // fighter female
		{name: "PC_Fergard", params: humansParams(30, 70, 3), strings: robeCells}, // mage male
		{name: "PC_Reniesta", params: humansParams(30, 70, 1)},                    // mage female
	}
	tbl := &mapload.Table{Humans: humans, Armors: armors, Shapes: emptyScale{}, Materials: emptyScale{}}

	fighter := MissionParty(nil, data.NewBodyList("unarmed"), tbl)
	f := &FrontEnd{InstallResources: InstallResources{Table: tbl, Humans: humans, Bodies: data.NewBodyList("unarmed", "mage")}}
	mage := f.ChargenParty(ui.ChargenResult{Choices: []int{0, 1, 0}, Stats: []int{25, 25, 25, 25}})

	if fighter[0].Worn[2] == 0 || mage[0].Worn[2] == 0 {
		t.Fatalf("fixture: the robe did not land in slot 3 for both (fighter %v, mage %v)",
			fighter[0].Worn, mage[0].Worn)
	}
	if fighter[0].Worn[2] != mage[0].Worn[2] {
		t.Errorf("fighter wears 0x%04x in slot 3, mage wears 0x%04x for the identically-named "+
			"piece — an equip moved with the class flag", fighter[0].Worn[2], mage[0].Worn[2])
	}
	if !mage[0].Mage {
		t.Fatal("fixture: the second party was not generated as a mage")
	}
}

// TestNewGameChargenClaimsExactlyTheMissionRows is 0140's own wiring decision,
// at the tier that makes it: pkg/ui asks one row at a time and this front end
// answers "generation" for a mission row and nothing for any other. The gate is
// what makes the owner's ruling — a campaign the player STARTS opens generation
// — true of the MAP LIST, which NEW GAME opens. cmd/againrom's -mission is the
// other fresh start and is tested there; a campaign TRANSITION is neither, and
// pkg/ui's own TestCampaignAdvanceNeverArmsGeneration is where that is held.
//
// It is asserted on a hand-built Maps slice rather than on an install, so the
// property is decided with no game present (golden rule 2).
func TestNewGameChargenClaimsExactlyTheMissionRows(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{
		Table:  &mapload.Table{},
		Humans: fourBaseHumans(),
		Bodies: data.NewBodyList("unarmed", "mage"),
		Maps: []MapEntry{
			{Source: "loose.alm", Name: "A"},           // 0: a loose map, no mission
			{Source: "kv1.alm", Name: "B", Mission: 1}, // 1: a campaign mission
			{Source: "spare.alm", Name: "C"},           // 2: a loose map again
		},
	}}

	if e := f.newGameChargen(0); e != nil {
		t.Error("row 0 is a loose map and the gate claimed it — generating a character to walk " +
			"a map that carries no party would be inventing a game mode")
	}
	if e := f.newGameChargen(2); e != nil {
		t.Error("row 2 is a loose map and the gate claimed it")
	}

	e := f.newGameChargen(1)
	if e == nil {
		t.Fatal("row 1 is a mission and the gate did not claim it — a campaign started from " +
			"NEW GAME opens generation (owner, 2026-08-10)")
	}
	if e.Model == nil {
		t.Fatal("the entry carries no model; pkg/ui refuses one and reports it on the picker")
	}
	if e.Begin == nil {
		t.Fatal("the entry carries no begin callback; the screen could then never be confirmed")
	}

	// THE MODEL IS THE SAME SETUP THE COMMAND LINE'S OWN DOOR OPENS. Its header
	// carries the title ChargenSetup states, so a screen reached from a picker
	// row and one reached from -mission cannot be two different screens.
	if got, want := e.Model.HeaderText(), ui.NewChargen(f.ChargenSetup()).HeaderText(); got != want {
		t.Errorf("the gated model's header is %q, want the ChargenSetup one %q", got, want)
	}

	// A FRESH MODEL PER CALL. A player who backs out and picks another mission
	// starts again; a shared model would hand him the second mission with the
	// first one's half-spent points on it.
	if other := f.newGameChargen(1); other.Model == e.Model {
		t.Error("two calls answered the SAME *ui.Chargen — an abandoned spread would follow the " +
			"player onto the next mission he picks")
	}

	// THE CALLBACK ANSWERS AN OPENER AND DOES NOT OPEN ANYTHING. It cannot open
	// here: this front end has no install behind it. That it composes at all,
	// with no error, is what this line witnesses.
	open, err := e.Begin(ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{30, 20, 18, 16}})
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if open == nil {
		t.Error("begin answered no opener; a confirmed character would reach no mission")
	}
}

// TestNewGameChargenIsTotalOverARowThatIsNotThere is the same totality every other
// seam in this file promises, read from the gate's side: pkg/ui hands it a row
// index and a front end whose list has since changed, or a hand-built one with
// no list at all, must answer "no generation" rather than panic in the picker.
func TestNewGameChargenIsTotalOverARowThatIsNotThere(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Maps: []MapEntry{{Source: "a.alm", Mission: 3}}}}
	for _, row := range []int{-1, 1, 99} {
		if e := f.newGameChargen(row); e != nil {
			t.Errorf("newGameChargen(%d) claimed a row that is not in the list", row)
		}
	}
	if e := (&FrontEnd{}).newGameChargen(0); e != nil {
		t.Error("a front end with no map list claimed row 0")
	}
}

// TestChargenDerivedShowsEverySkillSlot is item 3 of the owner's own
// request: the consequence block lists all five schools of the chosen class,
// and the ones generation did not train read 0, so "he is a Fire mage and
// nothing else" is visible rather than inferable.
//
// IT ALSO VERIFIES pkg/data/recompute.go's OWN CLAIM, which is why the zeros
// are asserted against d.Skill rather than merely against the printed lines.
// That file's logBase11 doc states, as the ground for a float-margin argument,
// that "generation trains exactly one skill slot, at level 10 or level 20, the
// other five at zero". If that were false the zeros this test wants would be a
// display convention papering over the data; it is true, and this is where a
// change to the mint that broke it would be caught from the screen's side.
func TestChargenDerivedShowsEverySkillSlot(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}

	for _, tc := range []struct {
		name    string
		class   int // the class choice: 0 fighter, 1 mage
		skill   int // the skill choice, an index into that class's five names
		wantSet []string
	}{
		{"a fighter trained in the third school", 0, 2, data.SkillNames(false)},
		{"a mage trained in the first school", 1, 0, data.SkillNames(true)},
		{"a mage trained in the last school", 1, 4, data.SkillNames(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := ui.ChargenResult{Choices: []int{0, tc.class, tc.skill}, Stats: []int{25, 25, 25, 25}}

			// THE MINT'S OWN ARRAY FIRST. Every claim below is about what
			// PartySpawn derived for this very member, not about a second
			// derivation this test performs.
			d, _, _ := mapload.PartySpawn(f.ChargenParty(res)[0])
			trained := 0
			for slot := int32(data.SkillBlade); slot < data.SkillSlots; slot++ {
				if d.Skill[slot] != 0 {
					trained++
				}
			}
			if trained != 1 {
				t.Fatalf("the mint trained %d of the five named slots (%v); recompute.go's own "+
					"logBase11 doc rests on it training exactly one", trained, d.Skill)
			}
			if got := d.Skill[data.SkillBlade+int32(tc.skill)]; got == 0 {
				t.Fatalf("the slot the player chose reads 0; the whole array is %v", d.Skill)
			}

			// AND NOW THE BLOCK. Five lines, named for the chosen class, each
			// carrying that slot's own level.
			lines := make(map[string]string, len(f.ChargenDerived(res)))
			for _, l := range f.ChargenDerived(res) {
				lines[l.Name] = l.Value
			}
			zeros := 0
			for i, name := range tc.wantSet {
				want := strconv.FormatInt(int64(d.Skill[data.SkillBlade+int32(i)]), 10)
				got, ok := lines[name]
				if !ok {
					t.Errorf("the block has no line for skill %q", name)
					continue
				}
				if got != want {
					t.Errorf("%s = %q, want the mint's own %q", name, got, want)
				}
				if got == "0" {
					zeros++
				}
			}
			if zeros != len(tc.wantSet)-1 {
				t.Errorf("%d of the five schools read 0, want %d — the untrained schools are "+
					"shown as zeros, not omitted (owner, 2026-08-10)", zeros, len(tc.wantSet)-1)
			}

			// THE OTHER CLASS'S NAMES ARE NOT ON THE SCREEN. A block that
			// listed both sets would be five right lines and five wrong ones.
			for _, name := range data.SkillNames(tc.class == 0) {
				if _, ok := lines[name]; ok {
					t.Errorf("the block carries %q, which is the OTHER class's name for a slot", name)
				}
			}
		})
	}
}

// TestChargenDerivedShowsSightAndTheMagicResistances is item 2, as the owner
// corrected it: it is the MAGIC resistance that must be shown, not the one
// against weapons. Both values are on data.Derived already and both are READ
// from the mint rather than recomputed.
//
// THE TWO ARRAYS ARE NAMED THE OPPOSITE WAY ROUND FROM THE WAY THEY READ, which
// is the mistake this test exists to make impossible to reintroduce silently.
// UNIT-COMBAT-015 binds columns 19…23, titled `prot Fire..Astral`, to
// data.Derived.Protection -- the elemental five -- and columns 24…28,
// `res.Blade..res.Shooting`, to data.Derived.Resistance, the weapon damage-kind
// five. The block shows Protection. Resistance is never re-derived for a human
// (HERO-RESIST-012), which is asserted below rather than assumed, because it is
// the reason showing it would be showing a row that can never move.
func TestChargenDerivedShowsSightAndTheMagicResistances(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}
	res := ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{30, 20, 18, 16}}

	lines := make(map[string]string)
	for _, l := range f.ChargenDerived(res) {
		lines[l.Name] = l.Value
	}
	d, _, _ := mapload.PartySpawn(f.ChargenParty(res)[0])

	if got, want := lines[chargenDerivedSight], strconv.FormatInt(int64(d.Sight), 10); got != want {
		t.Errorf("%s = %q, want the mint's own %q", chargenDerivedSight, got, want)
	}
	if d.Sight == 0 {
		t.Errorf("the fixture's sight is 0, so the line above would pass on a block that printed " +
			"a zero for every value; the fixture is not exercising the read")
	}

	// THE ROW IS d.Protection, WITH THE FIVE ELEMENT NAMES, IN COLUMN ORDER --
	// written out longhand so the names, the separator and the order are pinned
	// by this test and not by the function it is checking.
	want := fmt.Sprintf("Fire %d / Water %d / Air %d / Earth %d / Astral %d",
		d.Protection[0], d.Protection[1], d.Protection[2], d.Protection[3], d.Protection[4])
	if got := lines[chargenDerivedMagicRes]; got != want {
		t.Errorf("%s = %q, want %q", chargenDerivedMagicRes, got, want)
	}

	// IT IS THE WRONG ARRAY THAT WOULD BE ALL ZEROS. HERO-RESIST-012: the five
	// damage-kind resistances are never re-derived for a human, and PartySpawn
	// folds no EquipMod, so a block that had reached for d.Resistance would show
	// five zeros under a label a reader would take for the one he wanted. Both
	// halves are asserted, because "Protection is nonzero" alone would pass on a
	// tree where the two arrays had been swapped.
	for i, v := range d.Resistance {
		if v != 0 {
			t.Errorf("d.Resistance[%d] = %d, want 0 -- the damage-kind five are not derived "+
				"for a human, so this fixture cannot tell the two arrays apart any more", i, v)
		}
	}
	if d.Protection[0] == 0 {
		t.Error("d.Protection[0] = 0; the elemental five are Spirit/2 and this fixture spends " +
			"16 on Spirit, so a zero here means the row being shown is not the derived one")
	}

	// AND IT MOVES WITH SPIRIT, which is the whole reason it earns a line: a
	// point spent on Spirit visibly buys magic resistance (HERO-RESIST-012's
	// prot[i] = spirit / 2). The old note excluded these as "constant across
	// every spread the screen can reach"; they are not.
	high := f.ChargenDerived(ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{30, 20, 18, 40}})
	highLines := make(map[string]string)
	for _, l := range high {
		highLines[l.Name] = l.Value
	}
	if highLines[chargenDerivedMagicRes] == lines[chargenDerivedMagicRes] {
		t.Errorf("the magic resistance row reads %q at Spirit 16 and at Spirit 40 alike -- "+
			"a row that does not move tells the player nothing about what he is spending",
			lines[chargenDerivedMagicRes])
	}

	// A PERMUTED ARRAY WOULD PASS EVERY LINE ABOVE IF ALL FIVE HELD THE SAME
	// NUMBER, which for a hero they do. This pins the pairing against an array
	// with a shape only one order prints.
	if got, w := namedRow(data.ProtectionNames(), []int32{1, 2, 3, 4, 5}),
		"Fire 1 / Water 2 / Air 3 / Earth 4 / Astral 5"; got != w {
		t.Errorf("namedRow over a known array = %q, want %q -- names[i] must title values[i]", got, w)
	}
	// A name list shorter than the values prints the numbers it cannot title,
	// rather than dropping them: the number is the fact.
	if got, w := namedRow([]string{"A"}, []int32{7, 8}), "A 7 / 8"; got != w {
		t.Errorf("namedRow with a short name list = %q, want %q", got, w)
	}
}

// TestChargenDerivedFitsTheScreensLineBudget is the arithmetic pkg/ui measures,
// read from the side that decides what goes in the block. pkg/ui's own
// chargenDerivedFit answers 16 for this screen's seven rows and clips anything
// past it; this asserts the block does not need clipping, so the value at the
// bottom of it is one a player can actually see.
//
// THE 16 IS A LITERAL HERE ON PURPOSE. It is pkg/ui's own answer and this
// package cannot reach the unexported constant that produces it; the number is
// therefore restated with its derivation, and pkg/ui's own
// TestChargenDerivedFitIsMeasuredAgainstTheLayout is what keeps the derivation
// honest. y = chargenTop + (7 + 2 + k) * chargenLine = 184 + 16k, a line
// occupies [y, y+16), and the message line is at 452, so k <= 15.
func TestChargenDerivedFitsTheScreensLineBudget(t *testing.T) {
	const capacity = 16 // pkg/ui: chargenDerivedFit(7)
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}

	// +1 for ui.Chargen.DerivedText's own title line, which the draw path
	// counts against the same budget.
	got := len(f.ChargenDerived(ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{30, 20, 18, 16}})) + 1
	if got > capacity {
		t.Errorf("the block is %d lines against a budget of %d — the lines past it are clipped "+
			"by drawChargen and the player never sees them", got, capacity)
	}
}

func TestAGeneratedCharactersSheetStatesHisCarriedWeight(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Table: &mapload.Table{}, Humans: fourBaseHumans(), Bodies: data.NewBodyList("unarmed", "mage")}}
	result := ui.ChargenResult{Name: "Hero", Choices: []int{1, 1, 3}, Stats: []int{25, 26, 27, 28}}
	s := f.ChargenPreview(result).Subject

	if !s.WeightKnown {
		t.Fatal("the generated character's sheet subject states no carried weight at all")
	}
	if s.Weight != 0 {
		t.Errorf("the load is %d, want 0: this fixture's table resolves no item", s.Weight)
	}
	var found string
	for _, line := range ui.PanelStatement(ui.CompactPanelLayout(nil), s) {
		if strings.HasPrefix(line, "WEIGHT ") {
			found = line
		}
	}
	if found != "WEIGHT 0.0" {
		t.Errorf("the card's weight row reads %q, want only its weight: player bit 0 suppresses slot 190", found)
	}
}
