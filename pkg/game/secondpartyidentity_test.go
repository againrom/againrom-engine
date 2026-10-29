package game

import (
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// identityTable is a second-game definition table whose Humans rows 30 and
// 31 carry serverIDs 10220 and 10230, the edition's identity rows 22 and 23.
func identityTable() *mapload.Table {
	humans := make(dbCollection, 32)
	humans[30] = dbEntry{name: "Fighter_Hero", params: humansParams(40, 0, 2)}
	humans[31] = dbEntry{name: "Mage_Hero", params: humansParams(30, 60, 3)}
	humans[30].params[0x18], humans[31].params[0x18] = 10220, 10230
	e := base.GameROM2.Edition()
	return &mapload.Table{Humans: humans, Edition: &e, UnitKeys: mapload.ServerUnitKeys, Shapes: emptyScale{}, Materials: emptyScale{}}
}

func identityHero() mapload.PartyMember {
	return mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true, FigureDir: string(data.FigureDirManFighter)}
}

// identityNames is an npcnames.txt whose line 21 names identity 22.
func identityNames() synth.File {
	lines := make([]string, 23)
	lines[20], lines[21], lines[22] = "Aldor", "Igles", "Dina"
	return synth.File{Path: "text/npcnames.txt", Data: []byte(strings.Join(lines, "\r\n"))}
}

func carriedIdentities(party []mapload.PartyMember, t *mapload.Table) []int {
	var out []int
	for _, p := range party {
		if id, ok := mapload.PartyIdentity(p, t); ok {
			out = append(out, id)
		}
	}
	return out
}

// The end of the stage-30 talk with NPC 22 adds identity 22, built from the
// Humans row with serverID 10220 and named by npcnames line 21, once; a
// second talk adds nobody, and a talk with another NPC adds nobody.
func TestSecondInnTalkAddsIdentityTwentyTwoOnce(t *testing.T) {
	text := synth.File{Path: "text/town.txt", Data: []byte("#npc22talk30\r\n<NPC=21,PART=1>\r\nMay I sit here?\r\n<NPC=22,PART=2>\r\nCome along.\r\n#npc2108talk31\r\n<NPC=2108,PART=1>\r\nThirty-one.\r\n#other\r\n")}
	f, app, screen := secondCampaignFixtureFiles(t, false, text, identityNames())
	f.Table = identityTable()
	f.Carried = []mapload.PartyMember{identityHero()}
	c := innAt(2, 30, nil)
	f.Town.second = c
	app.SetTown(screen)
	screen.Choose(0)
	if rows := screen.Rows(); len(rows) < 2 || rows[0].Text != "TALK 22" || rows[1].Text != "TALK 2108" {
		t.Fatal("inn rows", rows)
	}
	screen.Choose(1)
	for _, ok := screen.dialogueBody(); ok; _, ok = screen.dialogueBody() {
		screen.AdvanceTownDialogue()
	}
	if len(f.Carried) != 1 {
		t.Fatalf("the talk with NPC 2108 carried %d members, want the hero alone", len(f.Carried))
	}
	screen.Choose(0)
	if _, ok := screen.dialogueBody(); !ok || len(f.Carried) != 1 {
		t.Fatalf("the open talk with NPC 22 carries %d members, want the hero until its end", len(f.Carried))
	}
	for _, ok := screen.dialogueBody(); ok; _, ok = screen.dialogueBody() {
		screen.AdvanceTownDialogue()
	}
	if got := carriedIdentities(f.Carried, f.Table); len(f.Carried) != 2 || len(got) != 2 || got[0] != 21 || got[1] != 22 {
		t.Fatalf("party after the talk = %d members, identities %v; want the hero and identity 22", len(f.Carried), got)
	}
	iglez := f.Carried[1]
	if iglez.Name != "Igles" || iglez.ID != "npc:22" || iglez.CompanionNPC != 22 || iglez.StartingHero || !iglez.PlayerCharacter || iglez.Hired() {
		t.Fatalf("identity 22 = %+v, want the companion Igles", iglez)
	}
	if row := mapload.PartyHumanRow(iglez, f.Carried[0], f.Table); row != 30 {
		t.Fatalf("identity 22's Humans row = %d, want 30 (serverID 10220)", row)
	}
	screen.Choose(0)
	for _, ok := screen.dialogueBody(); ok; _, ok = screen.dialogueBody() {
		screen.AdvanceTownDialogue()
	}
	if len(f.Carried) != 2 {
		t.Fatalf("a second talk carries %d members, want 2", len(f.Carried))
	}
	// The town speaker cast resolves the npc=22 lines of later talks to him.
	cast := newSpeakerCast(f.Carried, nil)
	if len(cast.actors) != 2 || cast.actors[1].npc != 22 {
		t.Fatalf("speaker cast %+v, want identity 22 as companion record 22", cast.actors)
	}
}

// The room pane counts identity 22 and its NEXT arrow shows him; PREVIOUS
// returns to the hero.
func TestSecondPanePickerStepsToIdentityTwentyTwo(t *testing.T) {
	f, app, screen := secondCampaignFixtureFiles(t, false, synth.File{Path: "text/town.txt", Data: []byte("#other\r\n")})
	f.Table = identityTable()
	f.Carried = []mapload.PartyMember{identityHero()}
	f.Town.second = innAt(2, 30, nil)
	app.SetTown(screen)
	screen.Choose(0)
	screen.Choose(0)
	screen.composed = &townSceneState{}
	spec := &town.PaneSpec{Rect: town.Rect{480, 238, 640, 480}}
	screen.TownSquarePane(ui.CharacterPaneNext)
	if v, _, _ := screen.heroPane(spec); v.MemberCount != 2 || v.Member != 1 || screen.heroFace().member != 1 {
		t.Fatalf("NEXT shows member %d of %d, want identity 22, the second of 2", v.Member, v.MemberCount)
	}
	screen.TownSquarePane(ui.CharacterPanePrev)
	if v, _, _ := screen.heroPane(spec); v.Member != 0 {
		t.Fatalf("PREVIOUS shows member %d, want the hero", v.Member)
	}
}

// In the shop, PREVIOUS and NEXT step the party the pane shows: with
// identity 22 carried, NEXT shows him and PREVIOUS the hero. The doll is
// his while the strip stays the bag, so a purchase made while the pane
// shows him still lands in the bag (DIV-3043).
func TestSecondShopPanePickerStepsToIdentityTwentyTwo(t *testing.T) {
	f, _, screen := secondCampaignFixtureFiles(t, false)
	f.Table = identityTable()
	f.Carried = []mapload.PartyMember{identityHero()}
	if !f.addPartyIdentity(townInstall{table: f.Table}, 22) || len(f.Carried) != 2 {
		t.Fatalf("identity 22 not added: party of %d", len(f.Carried))
	}
	c := secondShopSample()
	f.Town.second, f.Town.gold = c, 300
	screen.bindBankShop()
	screen.shop.shown = nil
	screen.composed = &townSceneState{}
	spec := &town.PaneSpec{Rect: town.Rect{480, 238, 640, 480}}
	iglez := len(f.Carried[1].CarriedItems)
	screen.click(ui.ShopControlPickerNext, 0)
	if v, _, _ := screen.heroPane(spec); v.MemberCount != 2 || v.Member != 1 || screen.heroFace().member != 1 {
		t.Fatalf("NEXT shows member %d of %d, want identity 22, the second of 2", v.Member, v.MemberCount)
	}
	if screen.shopMemberIndex() != 1 || len(screen.shopPackItemInstances()) != len(*c.heroBag()) {
		t.Fatalf("the shop's doll is member %d with a strip of %d units, want identity 22 beside the bag", screen.shopMemberIndex(), len(screen.shopPackItemInstances()))
	}
	screen.click(ui.ShopControlShelfPick, 0)
	screen.click(ui.ShopControlShelfCell, 0)
	screen.click(ui.ShopControlButton, 1)
	if f.Town.gold != 300-125 || len(*c.heroBag()) != 4+1 || len(f.Carried[1].CarriedItems) != iglez {
		t.Fatalf("purchase: gold %d bag %d, identity 22 carries %d (was %d)", f.Town.gold, len(*c.heroBag()), len(f.Carried[1].CarriedItems), iglez)
	}
	screen.click(ui.ShopControlPickerPrev, 0)
	if v, _, _ := screen.heroPane(spec); v.Member != 0 {
		t.Fatalf("PREVIOUS shows member %d, want the hero", v.Member)
	}
}

// When identity 22 falls, the guarded loss shows the mission text's failure3
// section; the hero's fall shows failure2 (R2-ENGINE-625).
func TestSecondGuardedLossOfIdentityTwentyTwoShowsFailureThree(t *testing.T) {
	src := missionSource{"main/text/mission7.txt": []byte("#failure2\r\nThe hero is dead.\r\n#failure3\r\nIglez is dead.\r\n")}
	for _, tc := range []struct {
		name   string
		fallen sim.EntityID
		want   string
	}{{"identity 22", 2, "Iglez is dead.\r\n"}, {"hero", 1, "The hero is dead.\r\n"}} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := sim.NewROM2Script(missionChecks(), []sim.ScriptInstant{}, []sim.ScriptTrigger{})
			if err != nil {
				t.Fatal(err)
			}
			ents := []sim.Entity{
				{ID: 1, Owner: sim.SelfSlot, X: 20, Y: 20, HP: 40, MaxHP: 40, TypeID: sim.HumanTypeID, Humanoid: true},
				{ID: 2, Owner: sim.SelfSlot, X: 22, Y: 20, HP: 40, MaxHP: 40, TypeID: sim.HumanTypeID, Humanoid: true},
			}
			ents[tc.fallen-1].HP = -10001
			w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
				sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), ents, s)
			if err != nil {
				t.Fatal(err)
			}
			mw, v := missionDriverFor(t, w, nil, src)
			mw.mission.table = identityTable()
			mw.mission.party = []mapload.PartyMember{identityHero(), {ID: "npc:22", CompanionNPC: 22, PlayerCharacter: true}}
			mw.mission.ids = []sim.EntityID{1, 2}
			mw.mission.guarded = []sim.EntityID{1, 2}
			mw.mission.failureText = secondGameFailureText(src, TextCode{}, 7)
			missionSteps(mw, 1)
			want := tc.want
			if want == "" {
				want = missionOutcomeText(mw.view, sim.OutcomeLost)
			}
			if body, kind, shown := v.NoticeState(); !shown || kind != ui.NoticeFailure || body != want {
				t.Fatalf("the fall of entity %d shows %q/%v/%v, want %q", tc.fallen, body, kind, shown, want)
			}
		})
	}
}

// A talk with no section ends at once and adds identity 22 then.
func TestSecondInnTalkWithoutSectionAddsAtOnce(t *testing.T) {
	f, app, screen := secondCampaignFixtureFiles(t, false, synth.File{Path: "text/town.txt", Data: []byte("#other\r\n")})
	f.Table = identityTable()
	f.Carried = []mapload.PartyMember{identityHero()}
	f.Town.second = innAt(2, 30, nil)
	app.SetTown(screen)
	screen.Choose(0)
	screen.Choose(0)
	if got := carriedIdentities(f.Carried, f.Table); len(got) != 2 || got[1] != 22 {
		t.Fatalf("identities %v after a talk without a section, want 21 and 22", got)
	}
}

// A member saved and loaded through the party policy keeps identity 22, and
// the producer reuses him: a talk after the LOAD adds nobody.
func TestSecondIdentityKeepsThroughPolicyAndIsReused(t *testing.T) {
	table := identityTable()
	s := &CampaignSession{Town: NewTown(Campaign{}), Carried: []mapload.PartyMember{identityHero()}}
	in := townInstall{table: table}
	if !s.addPartyIdentity(in, 22) {
		t.Fatal("identity 22 not added")
	}
	loaded := make([]mapload.PartyMember, len(s.Carried))
	for i, p := range s.Carried {
		record := sav.DocumentCharacter{Character: sav.Character{Basis: &sav.ActorBasis{}}}
		loaded[i] = capturePartyPolicy(p, record, table).member()
		loaded[i].StartingHero = p.StartingHero
	}
	if got := carriedIdentities(loaded, table); len(got) != 2 || got[1] != 22 {
		t.Fatalf("loaded identities %v, want 21 and 22", got)
	}
	s.Carried = loaded
	s.addTalkIdentities(in, 22, 30)
	if s.addPartyIdentity(in, 22) || len(s.Carried) != 2 {
		t.Fatalf("the producer built identity 22 again over a loaded party of %d", len(s.Carried))
	}
}

// Mission 40's departure stores slot 773=23; the location leave then adds
// identity 23 from serverID 10230 once, and a later leave adds nobody.
func TestSecondLeaveAfterFortyAddsIdentityTwentyThree(t *testing.T) {
	table := identityTable()
	c := innAt(2, 40, nil)
	c.current = partyPlace{kind: placeMission, id: 40}
	c.available = []partyPlace{c.current}
	s := &CampaignSession{Town: NewTown(Campaign{}), Carried: []mapload.PartyMember{identityHero()}}
	s.Town.second = c
	in := townInstall{table: table}
	s.leftLocation(in)
	if len(s.Carried) != 1 {
		t.Fatal("a leave with slot 773 zero added a member")
	}
	c.completeBank(c.bank)
	if c.bank[773] != 23 {
		t.Fatal("mission 40's departure slot 773", c.bank[773])
	}
	s.leftLocation(in)
	s.leftLocation(in)
	if got := carriedIdentities(s.Carried, table); len(got) != 2 || got[1] != 23 {
		t.Fatalf("identities %v after the leave, want 21 and 23", got)
	}
	if row := mapload.PartyHumanRow(s.Carried[1], s.Carried[0], table); row != 31 {
		t.Fatalf("identity 23's Humans row = %d, want row 31 (serverID 10230)", row)
	}
}

// The second game binds literal 10001+k to the party member with identity
// 21+k: 10001 the hero, 10002 identity 22; without him 10002 is unbound.
func TestSecondHeroBandBindsByIdentity(t *testing.T) {
	table := identityTable()
	m := &alm.Map{}
	entity := func(i int) sim.EntityID { return sim.EntityID(100 + i) }
	s := &CampaignSession{Town: NewTown(Campaign{}), Carried: []mapload.PartyMember{identityHero()}}
	refs := campaignScriptPartyRefs(m, table, s.Carried, entity, nil)
	if !refs.HasHero || refs.Hero != 100 || refs.HasCompanion || len(refs.Roles) != 0 {
		t.Fatalf("hero alone binds %+v, want 10001 only", refs)
	}
	s.addPartyIdentity(townInstall{table: table}, 23)
	s.addPartyIdentity(townInstall{table: table}, 22)
	refs = campaignScriptPartyRefs(m, table, s.Carried, entity, nil)
	if !refs.HasHero || refs.Hero != 100 || !refs.HasCompanion || refs.Companion != 102 || refs.Roles[10003] != 101 {
		t.Fatalf("party of identities 21, 23, 22 binds %+v, want 10001 the hero, 10002 member 2, 10003 member 1", refs)
	}
}

// The first game's edition keeps the hero ordinal role test: a companion
// carrying npc 22 binds no literal by identity.
func TestFirstHeroBandKeepsTheRoleTest(t *testing.T) {
	table := identityTable()
	table.Edition = nil
	party := []mapload.PartyMember{identityHero(), {ID: "npc:22", CompanionNPC: 22, PlayerCharacter: true}}
	refs := campaignScriptPartyRefs(&alm.Map{}, table, party, func(i int) sim.EntityID { return sim.EntityID(i) }, nil)
	if refs.HasCompanion {
		t.Fatalf("the first game bound 10002 to %d without an ordinal template", refs.Companion)
	}
	if !refs.HasHero || refs.Hero != 0 {
		t.Fatalf("the first game's role test bound 10001 to %d (bound %v), want the hero", refs.Hero, refs.HasHero)
	}
	if _, ok := mapload.PartyIdentity(party[1], table); ok {
		t.Fatal("the first game's companion carries a party identity")
	}
}
