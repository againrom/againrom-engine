package game

import (
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondPageTalk opens target and pages its dialogue to its end; it returns
// the party size while the dialogue was open.
func secondPageTalk(t *testing.T, f *FrontEnd, app *ui.App, target string) int {
	t.Helper()
	secondLaterChoose(t, app, target)
	open := len(f.Carried)
	secondInnNotice(t, app)
	return open
}

// secondCarriedIdentities is the party identities of f's carried members.
func secondCarriedIdentities(f *FrontEnd) []int {
	var out []int
	for _, p := range f.Carried {
		if id, ok := mapload.PartyIdentity(p, f.Table); ok {
			out = append(out, id)
		}
	}
	return out
}

// secondIglezJoins enters the Kaarg inn at stage 30, talks to NPC 2108 (who
// adds nobody) and then to NPC 22, and checks that identity 22 joins at the
// end of that talk and not before (DIV-4186). The position is secondTownTwo's
// constructed stage 30.
func secondIglezJoins(t *testing.T) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f, app, s, _ := secondTownTwo(t)
	c := f.Town.second
	secondLaterChoose(t, app, "TAVERN")
	s.TownSquareActive(true)
	if rows := secondInnRows(s); !slices.Contains(rows, "TALK 22") || !slices.Contains(rows, "TALK 2108") || c.room != secondTownInn {
		t.Fatalf("stage 30 inn rows %v", rows)
	}
	if secondPageTalk(t, f, app, "TALK 2108"); len(f.Carried) != 1 {
		t.Fatalf("the talk with NPC 2108 left %d members, want the hero alone", len(f.Carried))
	}
	if open := secondPageTalk(t, f, app, "TALK 22"); open != 1 {
		t.Fatalf("the open talk with NPC 22 carried %d members, want the hero until its end", open)
	}
	if got := secondCarriedIdentities(f); len(f.Carried) != 2 || !slices.Equal(got, []int{21, 22}) {
		t.Fatalf("party after the talk: %d members, identities %v; want 21 and 22", len(f.Carried), got)
	}
	iglez := f.Carried[1]
	want := f.townInstall().localizedNPC(21, "")
	if iglez.Name == "" || iglez.Name != want || iglez.ID != "npc:22" || iglez.Hired() || !iglez.PlayerCharacter {
		t.Fatalf("identity 22 = %q %q, want the companion named %q", iglez.ID, iglez.Name, want)
	}
	if row := mapload.PartyHumanRow(iglez, f.Carried[0], f.Table); f.Table.Humans.EntryName(row) != "Fighter_Hero" {
		t.Fatalf("identity 22 bound to Humans row %d, want serverID 10220's Fighter_Hero", row)
	}
	return f, app, s
}

// TestReleaseSecondKaargTalkJoinsIglez: the stage 30 talk with NPC 22 in
// the Kaarg inn adds Iglez once; a second talk adds nobody; the speaker cast
// gives his lines to him.
func TestReleaseSecondKaargTalkJoinsIglez(t *testing.T) {
	f, app, _ := secondIglezJoins(t)
	secondPageTalk(t, f, app, "TALK 22")
	if len(f.Carried) != 2 {
		t.Fatalf("a second talk left %d members, want 2", len(f.Carried))
	}
	if cast := newSpeakerCast(f.Carried, nil); len(cast.actors) != 2 || cast.actors[1].npc != 22 {
		t.Fatalf("speaker cast %+v, want Iglez as speaker 22", cast.actors)
	}
	t.Logf("party %q", []string{f.Carried[0].Name, f.Carried[1].Name})
}

// TestReleaseSecondKaargIglezKeepsThroughTownSave: a town SAV written after
// the talk loads cold with Iglez in the party, and a talk after the LOAD
// adds nobody.
func TestReleaseSecondKaargIglezKeepsThroughTownSave(t *testing.T) {
	f, app, _ := secondIglezJoins(t)
	out := t.TempDir()
	secondTownNamedSave(t, f, app, out, "Kaarg Iglez")
	cold, coldApp := secondTownCold(t, out, secondSaveFile(t, out, "Kaarg Iglez"))
	cs := cold.TownScreen().(*townScreen)
	coldApp.SetTown(cs)
	cs.TownSquareActive(true)
	if got := secondCarriedIdentities(cold); !slices.Equal(got, []int{21, 22}) || cold.Carried[1].Name != f.Carried[1].Name {
		t.Fatalf("cold LOAD identities %v, want 21 and 22 named %q", got, f.Carried[1].Name)
	}
	secondPageTalk(t, cold, coldApp, "TALK 22")
	if len(cold.Carried) != 2 {
		t.Fatalf("a talk after the cold LOAD left %d members, want 2", len(cold.Carried))
	}
}

// secondThirtyFixture is the hero-band values mission 30's compile leaves
// unbound under its party.
func secondThirtyFixture(t *testing.T, f *FrontEnd) []uint32 {
	t.Helper()
	m := f.live.mission.state
	_, rep, err := mapload.CompileROM2Script(m.Map, campaignScriptRefs(m.Map, f.Table, m.Party, m.Number))
	if err != nil {
		t.Fatal(err)
	}
	var out []uint32
	for _, u := range rep.Unresolved {
		if u.Value >= 10001 && u.Value <= 11000 && !slices.Contains(out, u.Value) {
			out = append(out, u.Value)
		}
	}
	return out
}

// secondThirtyParty checks mission 30's party: the hero and Iglez, both
// placed, both the player's, and 10002 bound to Iglez.
func secondThirtyParty(t *testing.T, f *FrontEnd, label string) {
	t.Helper()
	live := f.live
	if f.liveMission != 30 || len(live.mission.party) != 2 || len(live.mission.ids) != 2 {
		t.Fatalf("%s: mission %d party of %d", label, f.liveMission, len(live.mission.party))
	}
	hero, ok1 := live.world.Entity(live.mission.ids[0])
	iglez, ok2 := live.world.Entity(live.mission.ids[1])
	if !ok1 || !ok2 || hero.HP <= 0 || iglez.HP <= 0 || hero.Owner != iglez.Owner {
		t.Fatalf("%s: hero %v %+v, Iglez %v %+v", label, ok1, hero.ID, ok2, iglez.ID)
	}
	if id, ok := mapload.PartyIdentity(live.mission.party[1], f.Table); !ok || id != 22 {
		t.Fatalf("%s: second member identity %d", label, id)
	}
	if got := secondThirtyFixture(t, f); len(got) != 0 {
		t.Fatalf("%s: hero-band values %v unbound", label, got)
	}
	t.Logf("%s: hero at (%d,%d), Iglez at (%d,%d)", label, hero.X, hero.Y, iglez.X, iglez.Y)
}

// TestReleaseSecondKaargShopPaneStepsToIglez: after the talk, the Kaarg
// shop's hero pane counts 2; NEXT by pointer shows Iglez, a purchase then
// lands in the bag member (DIV-3043), and PREVIOUS shows the hero again.
func TestReleaseSecondKaargShopPaneStepsToIglez(t *testing.T) {
	f, app, s := secondIglezJoins(t)
	c := f.Town.second
	for n := 10; n < int(c.bank[768]); n += 10 {
		c.bank[896+n] = 1
	}
	c.shop.records = replayRecords(c.bank)
	secondLaterChoose(t, app, "EXIT")
	secondShopEnter(t, f, app, s)
	f.Town.gold = 5000000
	pane := func() (int, int) {
		v := s.ShopScreen()
		return v.Character.Member, v.Character.MemberCount
	}
	if m, n := pane(); m != 0 || n != 2 {
		t.Fatalf("the shop pane shows member %d of %d, want the hero of 2", m, n)
	}
	cityShopPointer(t, app, "picker_next", 0, "press", "release")
	if m, n := pane(); m != 1 || n != 2 {
		t.Fatalf("NEXT shows member %d of %d, want Iglez, the second of 2", m, n)
	}
	entry := -1
	for cat := 0; cat < 4 && entry < 0; cat++ {
		if !s.shelfOpen(cat) {
			continue
		}
		secondShopActivate(t, f, app, []string{"SHELF 1", "SHELF 2", "SHELF 3", "SHELF 4"}[cat])
		for i, it := range c.shop.shelf() {
			if it.Count >= 1 && i < len(s.ShopScreen().Shelf) {
				entry = i
				break
			}
		}
	}
	if entry < 0 {
		t.Fatal("no shown shelf entry")
	}
	bag, iglez, gold := len(*c.heroBag()), len(f.Carried[1].CarriedItems), f.Town.gold
	shopTap(t, app, secondShopPagePoint(t, app, "shelf", entry), "press", "release")
	secondShopActivate(t, f, app, "BUY")
	// Since the shop's doll follows the shown member and the strip is the
	// bag whichever member is shown, the customer is Iglez beside the bag.
	if f.Town.gold >= gold || len(*c.heroBag()) <= bag || len(f.Carried[1].CarriedItems) != iglez ||
		s.shopMemberIndex() != 1 || len(s.shopPackItemInstances()) != len(*c.heroBag()) {
		t.Fatalf("a purchase with Iglez shown: gold %d -> %d, bag %d -> %d, Iglez carries %d (was %d), customer %d with a strip of %d units",
			gold, f.Town.gold, bag, len(*c.heroBag()), len(f.Carried[1].CarriedItems), iglez, s.shopMemberIndex(), len(s.shopPackItemInstances()))
	}
	if m, _ := pane(); m != 1 {
		t.Fatalf("the purchase moved the pane to member %d", m)
	}
	cityShopPointer(t, app, "picker_prev", 0, "press", "release")
	if m, _ := pane(); m != 0 {
		t.Fatalf("PREVIOUS shows member %d, want the hero", m)
	}
}

// TestReleaseSecondMissionThirtyStartsWithIglez: after the talk, mission 30
// starts with Iglez beside the hero and its 10002 nodes build; the mission
// SAV loads cold with both.
func TestReleaseSecondMissionThirtyStartsWithIglez(t *testing.T) {
	f, app, _ := secondIglezJoins(t)
	if !f.Town.second.has(partyPlace{1, 30}) {
		t.Fatal("the talk did not offer mission 30", f.Town.second.available)
	}
	secondLaterChoose(t, app, "EXIT", "GATES")
	secondLaterEnter(t, f, app, 30)
	secondThirtyParty(t, f, "mission 30")
	out := t.TempDir()
	secondMissionNamedSave(t, f, app, out, "Thirty Iglez")
	cold, _ := secondMissionColdAt(t, out, secondSaveFile(t, out, "Thirty Iglez"), 30)
	secondThirtyParty(t, cold, "cold mission 30")
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("the cold LOAD of mission 30 differs from its SAV")
	}
}

// TestReleaseSecondIglezTalksDrawHisPortrait: every page of the stage-30
// Kaarg inn talk with NPC 22 draws a face, Iglez's on his own lines and the
// hero's on the hero's (DIV-4199); in mission 30 a line of NPC 22 draws the
// live Iglez and the hero's line draws the hero.
func TestReleaseSecondIglezTalksDrawHisPortrait(t *testing.T) {
	f, app, s, _ := secondTownTwo(t)
	secondLaterChoose(t, app, "TAVERN")
	s.TownSquareActive(true)
	secondLaterChoose(t, app, "TALK 22")
	lines := map[int]int{}
	for page := 0; page < 64; page++ {
		v := s.view()
		if !v.Open {
			break
		}
		r := s.dialogue
		speaker, named := tagSpeaker(r.tag)
		if named && v.Page.Portrait {
			lines[speaker]++
			if v.Page.Face == nil {
				t.Fatalf("page %d (%q): speaker %d draws no face", page, r.tag, speaker)
			}
		}
		if err := app.HeadlessActivate("notice"); err != nil {
			break
		}
	}
	if lines[22] == 0 || lines[21] == 0 {
		t.Fatalf("the talk's portrait lines by speaker %v, want lines of 22 and 21", lines)
	}
	if len(f.Carried) != 2 {
		t.Fatalf("the talk left %d members, want the hero and Iglez", len(f.Carried))
	}
	payload, _ := readSecondGameMissionText(f.Archives.Containers, InstallTextCode(f.Archives.Containers, tableEdition(f.Table)), 30)
	if !strings.Contains(strings.ToLower(string(payload)), "npc=22") {
		t.Fatal("mission 30's text names no line of NPC 22")
	}
	secondLaterChoose(t, app, "EXIT", "GATES")
	secondLaterEnter(t, f, app, 30)
	composed := func(speaker int) figureID {
		f.live.figurePics = map[figureCacheKey]*image.RGBA{}
		if _, _, ok := f.live.SpeakerFace(speaker); !ok || len(f.live.figurePics) != 1 {
			t.Fatalf("mission 30 speaker %d: ok %v, %d figures composed", speaker, ok, len(f.live.figurePics))
		}
		for key := range f.live.figurePics {
			return key.fig
		}
		return figureID{}
	}
	var iglez, hero figureID
	for _, a := range f.live.speakerActors {
		switch {
		case a.npc == 22:
			iglez = a.fig
		case a.me:
			hero = a.fig
		}
	}
	iglez.Hero, hero.Hero = true, true
	if got := composed(22); got != iglez || iglez.Dir == "" {
		t.Fatalf("mission 30 speaker 22 drew %+v, want the live Iglez %+v", got, iglez)
	}
	if got := composed(21); got != hero || got == iglez {
		t.Fatalf("mission 30 speaker 21 drew %+v, want the hero %+v", got, hero)
	}
}

// TestReleaseSecondDinaTalksDrawHerPortrait: once Dina is in the party, every
// page of the town talk npc23talk49 whose speaker is NPC 23 draws her, a woman
// mage who is a hero, and not the man mage the row at key 23 is (DIV-4203); in
// mission 50 a line of NPC 23 draws the live Dina and a line of NPC 22 still
// draws Iglez.
func TestReleaseSecondDinaTalksDrawHerPortrait(t *testing.T) {
	f, app, s, _ := secondTownTwo(t)
	in := f.townInstall()
	if !f.CampaignSession.addPartyIdentity(in, 22) || !f.CampaignSession.addPartyIdentity(in, 23) {
		t.Fatalf("the party holds identities %v, want 21, 22 and 23", secondCarriedIdentities(f))
	}
	payload, err := readSecondTownTalk(&f.InstallResources, "npc23talk49")
	if err != nil || len(payload) == 0 {
		t.Fatalf("no npc23talk49 section: %v", err)
	}
	s.setTownDialogue(payload)
	lines := 0
	for page := 0; page < 64; page++ {
		v := s.view()
		if !v.Open {
			break
		}
		r := s.dialogue
		if speaker, named := tagSpeaker(r.tag); named && v.Page.Portrait && speaker == 23 {
			lines++
			if v.Page.Face == nil {
				t.Fatalf("page %d (%q): speaker 23 draws no face", page, r.tag)
			}
		}
		if err := app.HeadlessActivate("notice"); err != nil {
			break
		}
	}
	dina := figureID{Dir: "fmage", Face: 20, Hero: true}
	drawn := map[figureID]bool{}
	for key := range s.resolver.figurePics {
		drawn[key.fig] = true
	}
	if lines == 0 || !drawn[dina] || drawn[figureID{Dir: "mmage", Face: 1}] {
		t.Fatalf("%d lines of speaker 23, figures drawn %v, want Dina %+v and no man mage", lines, drawn, dina)
	}
	c := f.Town.second
	c.available = append(c.available, partyPlace{1, 50})
	secondLaterChoose(t, app, "GATES")
	secondLaterEnter(t, f, app, 50)
	composed := func(speaker int) figureID {
		f.live.figurePics = map[figureCacheKey]*image.RGBA{}
		if _, _, ok := f.live.SpeakerFace(speaker); !ok || len(f.live.figurePics) != 1 {
			t.Fatalf("mission 50 speaker %d: ok %v, %d figures composed", speaker, ok, len(f.live.figurePics))
		}
		for key := range f.live.figurePics {
			return key.fig
		}
		return figureID{}
	}
	var iglez figureID
	for _, a := range f.live.speakerActors {
		if a.npc == 22 {
			iglez = a.fig
		}
	}
	iglez.Hero = true
	if got := composed(23); got != dina {
		t.Fatalf("mission 50 speaker 23 drew %+v, want the live Dina %+v", got, dina)
	}
	if got := composed(22); got != iglez || iglez.Dir == "" {
		t.Fatalf("mission 50 speaker 22 drew %+v, want the live Iglez %+v", got, iglez)
	}
}

// TestReleaseSecondMissionThirtyWithoutTheTalk is the loss control: mission
// 30 offered without the talk starts with the hero alone and leaves 10002
// unbound.
func TestReleaseSecondMissionThirtyWithoutTheTalk(t *testing.T) {
	f, app, _, _ := secondTownTwo(t)
	c := f.Town.second
	c.available = append(c.available, partyPlace{1, 30})
	secondLaterChoose(t, app, "GATES")
	secondLaterEnter(t, f, app, 30)
	if len(f.live.mission.party) != 1 || len(f.Carried) != 1 {
		t.Fatalf("mission 30 without the talk has %d members", len(f.live.mission.party))
	}
	if got := secondThirtyFixture(t, f); !slices.Contains(got, 10002) {
		t.Fatalf("mission 30 without Iglez leaves %v unbound, want 10002", got)
	}
	if f.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatal("mission 30 without Iglez decided at entry", f.live.world.Outcome())
	}
}
