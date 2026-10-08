package game

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func mercenaryDefs(t *testing.T) *data.NPCDefs {
	t.Helper()
	b := synth.Reg(kindRoot, []synth.RegNode{{Name: "npc3", Kind: kindDir, Children: []synth.RegNode{
		{Name: "DataBinID", Kind: kindInt, Int: 3},
		{Name: "PriceA", Kind: kindInt, Int: 1},
		{Name: "PriceB", Kind: kindInt, Int: 2},
	}}})
	r, err := reg.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return data.LoadNPCDefs(r)
}

func shellCampaign() Campaign {
	return Campaign{
		Main: []int{30}, Offered: []int{30}, MercenaryCount: []int{0, 0, 2},
		Chapters: map[int]Chapter{
			20: {Mission: 20, EnableMercenary: []int{3}},
			30: {Mission: 30, InnNPC: []int{9}, Inn: []int{0}, Mercenaries: []int{3}},
		},
	}
}

func shellFrontEnd() *FrontEnd {
	humans := dbCollection{{}, {name: "NPC03_1", params: humansParams(30, 0, 1)}}
	table := &mapload.Table{Humans: humans, NPC: mercenaryDefsNoTest()}
	town := NewTown(shellCampaign())
	town.Won(20)
	town.Arrive()
	town.gold = 1000
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(shellCampaign(), nil), Table: table, Humans: humans, Words: ui.AuthoredWords()}, CampaignSession: CampaignSession{Town: town, Carried: []mapload.PartyMember{{Name: "Player", PlayerCharacter: true, Hero: data.NewCampaignHero(1)}}}}
	// THROUGH THE PRODUCTION CONSTRUCTOR, not a copy of it. This fixture used
	// to build the screen literally, which meant no test in this file could see
	// what TownScreen() initialises: a field added there stayed at Go's zero
	// value here, and the test that would have caught it passed.
	f.townUI = f.TownScreen().(*townScreen)
	f.townUI.room = roomTavern
	f.townUI.composeShopFaces()
	return f
}

// mercenaryDefsNoTest is the same synthetic registry helper without a testing
// dependency, for fixtures assembled before an assertion exists.
func mercenaryDefsNoTest() *data.NPCDefs {
	b := synth.Reg(kindRoot, []synth.RegNode{{Name: "npc3", Kind: kindDir, Children: []synth.RegNode{
		{Name: "DataBinID", Kind: kindInt, Int: 3}, {Name: "PriceA", Kind: kindInt, Int: 1},
		{Name: "PriceB", Kind: kindInt, Int: 2},
	}}})
	r, _ := reg.Parse(b)
	return data.LoadNPCDefs(r)
}

func TestTavernWholeSquadHireAndReturnAreImmediate(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	offers := s.tavernMercenaries()
	if len(offers) != 1 || offers[0].Type != 3 || offers[0].Count != 2 || offers[0].Capacity != 2 || offers[0].Price != 50 {
		t.Fatalf("offers = %#v", offers)
	}
	cells := s.tavernSurfaceCells()
	if len(cells) != 2 || cells[0].Label != "" || cells[0].Semantic != "Mercenary 3" ||
		cells[0].Detail != "2/2" || cells[0].Price != "50" || !cells[0].Portrait {
		t.Fatalf("compact squad cell = %#v, want portrait with no visible name, 2/2 and price 50", cells)
	}
	if !cells[1].Portrait || !cells[1].TalkOnly || cells[1].Detail != "" || cells[1].Price != "" {
		t.Fatalf("mission NPC cell = %#v, want compact talk-only portrait without mercenary text", cells[1])
	}
	if cells[1].Semantic != "NPC 9" {
		t.Fatalf("mission NPC headless target = %q, want the existing stable NPC label", cells[1].Semantic)
	}
	if _, ok := s.toggleMercenary(3); !ok {
		t.Fatal("hire refused")
	}
	if got := f.Town.Gold(); got != 950 {
		t.Fatalf("gold after hire = %d, want 950", got)
	}
	if got := f.Town.MercenaryPool(3); got != 2 {
		t.Fatalf("working pool after Hire = %d, want 2", got)
	}
	if got := s.mercenaryPartyCount(3); got != 2 {
		t.Fatalf("party squad = %d, want 2", got)
	}
	snap, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	other := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(shellCampaign(), nil), Table: f.Table, Humans: f.Humans}, CampaignSession: CampaignSession{Town: NewTown(shellCampaign())}}
	other.TownScreen()
	if _, inTown, err := other.Restore(snap); err != nil || !inTown {
		t.Fatalf("restore hired squad = inTown %v, err %v", inTown, err)
	}
	if other.Town.Gold() != 950 || !other.Town.MercenaryHired(3) || other.Town.MercenaryPool(3) != 2 || len(other.Carried) != 3 {
		t.Fatalf("restored hire = gold %d hired %v working %d party %d", other.Town.Gold(),
			other.Town.MercenaryHired(3), other.Town.MercenaryPool(3), len(other.Carried))
	}
	legacy := snap
	legacy.MercenaryState = false
	legacy.MercenaryPool = [16]int{}
	legacy.MercenaryEnabled = [16]bool{}
	legacy.MercenaryHired = [16]bool{}
	legacyFront := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(shellCampaign(), nil), Table: f.Table, Humans: f.Humans}, CampaignSession: CampaignSession{Town: NewTown(shellCampaign())}}
	legacyFront.TownScreen()
	if _, inTown, err := legacyFront.Restore(legacy); err != nil || !inTown {
		t.Fatalf("restore pre-mercenary snapshot = inTown %v, err %v", inTown, err)
	}
	if !legacyFront.Town.MercenaryEnabled(3) || !legacyFront.Town.MercenaryHired(3) ||
		legacyFront.Town.MercenaryPool(3) != 0 || len(legacyFront.Carried) != 3 {
		t.Fatalf("pre-mercenary snapshot inference = enabled %v hired %v pool %d party %d",
			legacyFront.Town.MercenaryEnabled(3), legacyFront.Town.MercenaryHired(3),
			legacyFront.Town.MercenaryPool(3), len(legacyFront.Carried))
	}
	if _, ok := s.toggleMercenary(3); !ok {
		t.Fatal("return refused")
	}
	if got := f.Town.Gold(); got != 1000 {
		t.Fatalf("gold after return = %d, want 1000", got)
	}
	if got := f.Town.MercenaryPool(3); got != 2 {
		t.Fatalf("restored pool = %d, want 2", got)
	}
}

func TestSelectedMercenaryButtonChangesFromHireToFire(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	merc := 0 // mercenary cells come first (TOWN-468)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, false)

	button := s.TownSurface().Buttons[tavernButtonHire]
	if button.Label != "Hire" || !button.Enabled {
		t.Fatalf("available squad button = %+v, want enabled Hire", button)
	}
	if act := s.townSurfaceButton(tavernButtonHire); act.Msg != "" || !f.Town.MercenaryHired(3) {
		t.Fatalf("Hire action = %q", act.Msg)
	}
	f.Town.gold = 0 // returning an already hired squad must not require money
	button = s.TownSurface().Buttons[tavernButtonHire]
	if button.Label != "Fire" || !button.Enabled || !f.Town.MercenaryHired(3) {
		t.Fatalf("hired squad button/state = %+v/%v, want enabled Fire/hired", button, f.Town.MercenaryHired(3))
	}
	f.Town.gold = 950
	if act := s.townSurfaceButton(tavernButtonHire); act.Msg != "" || f.Town.MercenaryHired(3) || f.Town.Gold() != 1000 {
		t.Fatalf("Fire action = %q", act.Msg)
	}
	button = s.TownSurface().Buttons[tavernButtonHire]
	if button.Label != "Hire" || !button.Enabled || f.Town.MercenaryHired(3) {
		t.Fatalf("returned squad button/state = %+v/%v, want enabled Hire/not hired", button, f.Town.MercenaryHired(3))
	}
}

func TestTavernSleepOnlyRestocksTheShop(t *testing.T) {
	f := shellFrontEnd()
	table := shopTable()
	table.Humans, table.NPC = f.Table.Humans, f.Table.NPC
	f.Table = table
	f.Shop = NewShop(1000)
	baseSeed := shopSeed(f.Town.Chapter(), f.Town.finishedCount(), f.Shop.Ceiling())
	f.Shop.Generate(f.Table, baseSeed)
	wantShelves := func(ordinal uint64) [numShopShelves][]ShopItem {
		shop := NewShop(1000)
		shop.Generate(f.Table, shopRestockSeed(baseSeed, ordinal))
		var out [numShopShelves][]ShopItem
		for shelf := range out {
			out[shelf] = shop.Shelf(ShopShelf(shelf))
		}
		return out
	}
	gotShelves := func() [numShopShelves][]ShopItem {
		var out [numShopShelves][]ShopItem
		for shelf := range out {
			out[shelf] = f.Shop.Shelf(ShopShelf(shelf))
		}
		return out
	}
	for i := range f.Shop.shelves {
		f.Shop.shelves[i] = nil // an exhausted shop must be filled again
	}

	s := f.townUI
	merc := 0 // mercenary cells come first (TOWN-468)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, false)
	beforeParty := append([]mapload.PartyMember(nil), f.Carried...)
	beforeGold := f.Town.Gold()
	beforeSelection, beforeRoom := s.tavernSelection, s.room
	button := s.TownSurface().Buttons[tavernButtonSleep]
	if button.Label != "Sleep" || !button.Enabled {
		t.Fatalf("Sleep button = %+v, want enabled Sleep", button)
	}

	if act := s.townSurfaceButton(tavernButtonSleep); act.Msg != "" {
		t.Fatalf("Sleep action = %q", act.Msg)
	}
	if f.Shop.restocks != 1 || !reflect.DeepEqual(gotShelves(), wantShelves(1)) {
		t.Fatalf("restocked shop = ordinal %d, weapons %d, armour %d", f.Shop.restocks,
			len(f.Shop.Shelf(ShelfWeapons)), len(f.Shop.Shelf(ShelfArmour)))
	}
	if act := s.townSurfaceButton(tavernButtonSleep); act.Msg != "" ||
		f.Shop.restocks != 2 || !reflect.DeepEqual(gotShelves(), wantShelves(2)) {
		t.Fatalf("second Sleep = %q ordinal %d", act.Msg, f.Shop.restocks)
	}
	if f.Town.Gold() != beforeGold || !reflect.DeepEqual(f.Carried, beforeParty) ||
		s.tavernSelection != beforeSelection || s.room != beforeRoom || f.Town.MercenaryHired(3) {
		t.Fatalf("Sleep changed non-shop state: gold %d, partyChanged %v, selection %+v, room %d, hired %v",
			f.Town.Gold(), !reflect.DeepEqual(f.Carried, beforeParty), s.tavernSelection, s.room, f.Town.MercenaryHired(3))
	}
	if shopRestockSeed(baseSeed, 1) == baseSeed || shopRestockSeed(baseSeed, 2) == shopRestockSeed(baseSeed, 1) ||
		reflect.DeepEqual(wantShelves(1), wantShelves(2)) {
		t.Fatal("successive Sleep commands did not receive distinct deterministic seeds")
	}
}

func TestTavernActivatesTheFirstMercenaryAndDispatchesOnlyTheStableCandidate(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	// Activation stores position 0 when the roster has a mercenary (TOWN-468).
	v := s.TownSurface()
	want := tavernCandidateKey{kind: tavernCandidateMercenary, id: 3}
	if s.tavernSelection != want || !v.Cells[0].Selected || v.Cells[1].Selected || v.RosterUnpainted {
		t.Fatalf("entry selection = key %+v cells %#v", s.tavernSelection, v.Cells)
	}
	if len(v.Buttons) != 4 || v.Buttons[tavernButtonSleep].Label != "Sleep" ||
		v.Buttons[tavernButtonHire].Label != "Hire" || !v.Buttons[tavernButtonHire].Enabled ||
		v.Buttons[tavernButtonTalk].Label != "Talk" || v.Buttons[tavernButtonExit].Label != "EXIT" {
		t.Fatalf("tavern buttons on entry = %+v", v.Buttons)
	}

	// The talk-only producer follows the mercenaries and owns Talk but never Hire.
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
	v = s.TownSurface()
	if !v.Cells[1].Selected || v.Buttons[tavernButtonHire].Enabled || !v.Buttons[tavernButtonTalk].Enabled ||
		s.tavernSelection != (tavernCandidateKey{kind: tavernCandidateOffer, id: 0}) {
		t.Fatalf("offer selection = key %+v buttons %+v cells %#v", s.tavernSelection, v.Buttons, v.Cells)
	}

	// Losing every mercenary leaves the talk selection alone.
	f.Town.mercPool[3] = 0
	v = s.TownSurface()
	if s.tavernSelection != (tavernCandidateKey{kind: tavernCandidateOffer, id: 0}) || len(v.Cells) != 1 || v.RosterUnpainted {
		t.Fatalf("talk selection after the squad left = key %+v cells %#v", s.tavernSelection, v.Cells)
	}
}

// TestTavernWithoutMercenariesPaintsNoCellUntilASelection is the paint guard:
// with no mercenary cell activation stores -1 and neither paint loop draws,
// while the hit test still reaches the talk cell (TOWN-468).
func TestTavernWithoutMercenariesPaintsNoCellUntilASelection(t *testing.T) {
	f := shellFrontEnd()
	f.Town.mercPool[3] = 0
	s := f.townUI
	v := s.TownSurface()
	if s.tavernSelection != (tavernCandidateKey{}) || !v.RosterUnpainted || len(v.Cells) != 1 {
		t.Fatalf("no-mercenary entry = key %+v unpainted %v cells %d", s.tavernSelection, v.RosterUnpainted, len(v.Cells))
	}
	if c, ok := ui.TownSurfaceControlAt(v, image.Pt(200, 450)); !ok || c.Index != 0 {
		t.Fatalf("unpainted talk cell hit = %+v, %v", c, ok)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	if v = s.TownSurface(); v.RosterUnpainted || !v.Cells[0].Selected {
		t.Fatalf("after the click = unpainted %v cells %#v", v.RosterUnpainted, v.Cells)
	}
}

func TestSelectedTavernCandidateReusesItsRenderedInspection(t *testing.T) {
	f := shellFrontEnd()
	f.TownTavernArt = resolved(&ui.TownTavernArt{LeftStats: image.NewRGBA(image.Rect(0, 0, 160, 238))}, nil)
	s := f.townUI
	merc := 0 // mercenary cells come first (TOWN-468)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, false)

	first := s.TownSurface().CandidatePixels
	second := s.TownSurface().CandidatePixels
	if first == nil || second != first {
		t.Fatalf("stable selected candidate cache = %p then %p, want one non-nil image", first, second)
	}

	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
	if got := s.TownSurface().CandidatePixels; got != nil {
		t.Fatalf("mission-NPC selection retained mercenary inspection %p", got)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, false)
	third := s.TownSurface().CandidatePixels
	if third == nil || third == first {
		t.Fatalf("reselected mercenary cache = %p, want a new non-nil image after invalidation", third)
	}

	if !s.Back() {
		t.Fatal("Back refused to leave the tavern")
	}
	s.Choose(0)
	if s.tavernDetailPixels != nil {
		t.Fatal("tavern re-entry retained the previous visit's rendered inspection")
	}
}

func TestSelectedTalkOnlyCandidateShowsTheResolvedRealFigure(t *testing.T) {
	f := shellFrontEnd()
	f.TownTavernArt = resolved(&ui.TownTavernArt{
		LeftStats:   image.NewRGBA(image.Rect(0, 0, 160, 238)),
		LeftPicture: image.NewRGBA(image.Rect(0, 0, 160, 242)),
	}, nil)
	s := f.townUI
	want := image.NewRGBA(image.Rect(0, 0, 160, 240))
	want.SetRGBA(0, 0, color.RGBA{R: 0x84, G: 0x37, B: 0x62, A: 0xff})
	// `!Human` keeps every live member out; `Human` sends the synthesised
	// object down the composed-figure arm (TAVERN-TALKSTATS-017).
	rec := data.NPCFace{Kind: data.NPCNoPicture,
		Tokens: data.NPCTokens(data.NPCTokenNotHuman) | data.NPCTokens(data.NPCTokenHuman)}
	f.NPCFaces = map[int32]data.NPCFace{9: rec}
	s.resolver.npcFaces = f.NPCFaces
	s.resolver.playerFigure = want

	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
	v := s.TownSurface()
	if v.Candidate.HasSubject {
		t.Fatal("talk-only candidate invented a statistics subject")
	}
	if v.Candidate.Figure != want || v.CandidatePixels == nil {
		t.Fatalf("talk-only candidate figure/cache = %p/%p, want %p/non-nil", v.Candidate.Figure, v.CandidatePixels, want)
	}
	if got := v.CandidatePixels.RGBAAt(8, 240); got != want.RGBAAt(0, 0) {
		t.Fatalf("talk-only rendered doll origin = %#v, want %#v", got, want.RGBAAt(0, 0))
	}
}

func TestMercenaryKeepsItsTemplateMagicStaffInStatsAndEquipment(t *testing.T) {
	f := shellFrontEnd()
	const staff = "Wood Staff {castSpell=Fire_Arrow:10}"
	params := humansParamsBook(30, 70, 1, 1)
	params[16] = 24 // shipped mage type-id band
	cells := make([]string, 10)
	cells[0] = staff
	humans := dbCollection{{}, {name: "NPC03_1", params: params, strings: cells}}
	weapons := dbCollection{{}, {name: "Wood Staff", params: chargenWeaponParams(data.SkillBlade)}}
	spells := dbCollection{{}, {name: "Fire Arrow", params: make([]int32, 19)}}
	f.Humans, f.Table.Humans = humans, humans
	f.Table.Weapons, f.Table.Spells = weapons, spells
	f.Table.Shapes, f.Table.Materials = emptyScale{}, emptyScale{}
	s := f.townUI

	members, ok := s.buildMercenarySquad(3, 1)
	if !ok || len(members) != 1 {
		t.Fatalf("mage mercenary build = %d members, %v", len(members), ok)
	}
	m := members[0]
	if !m.Mage || m.Weapon == nil {
		t.Fatalf("mage/template weapon = %v/%v, want mage with staff", m.Mage, m.Weapon)
	}
	if m.Weapon.SpellName != "Fire_Arrow" || m.Weapon.SpellPower != 10 {
		t.Fatalf("staff spell = %q power %d, want Fire_Arrow power 10", m.Weapon.SpellName, m.Weapon.SpellPower)
	}
	if m.Worn[0] == 0 || m.WornItems[0].Code != m.Worn[0] {
		t.Fatalf("weapon projections = code 0x%04x item 0x%04x", m.Worn[0], m.WornItems[0].Code)
	}
	foundCast := false
	for _, effect := range m.WornItems[0].Effects {
		foundCast = foundCast || effect.Kind == 41
	}
	if !foundCast {
		t.Fatalf("staff item effects = %+v, want castSpell effect", m.WornItems[0].Effects)
	}
	eq := mapload.EquipmentFromParty(m)
	if code, ok := eq.Code(1); !ok || uint16(code) != m.Worn[0] {
		t.Fatalf("doll equipment slot 1 = 0x%04x, %v; want 0x%04x", uint16(code), ok, m.Worn[0])
	}
	candidate, _, info, ok := s.tavernCandidateDetail(3)
	if !ok || candidate.Subject.Char.Weapon == "" || len(info[0]) == 0 {
		t.Fatalf("candidate stats weapon/info = %q/%q, %v", candidate.Subject.Char.Weapon, info[0], ok)
	}
}

func TestTavernDisabledCardStillSelectsButCannotHire(t *testing.T) {
	f := shellFrontEnd()
	f.Town.gold = 0
	s := f.townUI
	v := s.TownSurface()
	merc := 0 // mercenary cells come first (TOWN-468)
	if c, ok := ui.TownSurfaceControlAt(v, image.Pt(200, 417)); !ok || c.Index != merc {
		t.Fatalf("unaffordable card hit = %+v, %v", c, ok)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, false)
	v = s.TownSurface()
	if !v.Cells[merc].Selected || v.Buttons[tavernButtonHire].Enabled || !v.Buttons[tavernButtonTalk].Enabled {
		t.Fatalf("disabled selection = selected %v Hire %v Talk %v", v.Cells[merc].Selected,
			v.Buttons[tavernButtonHire].Enabled, v.Buttons[tavernButtonTalk].Enabled)
	}
}

func TestTavernDialoguePreservesSelectionButLeavingAndReentryResetIt(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	merc := 0 // mercenary cells come first (TOWN-468)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, false)
	want := s.tavernSelection
	s.townSurfaceButton(tavernButtonTalk) // synthetic fixture has no archive, so this is the disclosed diagnostic path
	if s.room != roomTalk || s.talkDiagnostic == "" {
		t.Fatalf("missing external payload did not open disclosed diagnostic: room %v text %q", s.room, s.talkDiagnostic)
	}
	s.AdvanceTownDialogue()
	if s.room != roomTavern || s.tavernSelection != want {
		t.Fatalf("dialogue return = room %v selection %+v, want tavern %+v", s.room, s.tavernSelection, want)
	}
	if !s.Back() || s.room != roomSquare {
		t.Fatalf("Back did not leave tavern: room %v", s.room)
	}
	s.Choose(0)
	if s.room != roomTavern || s.tavernSelection != (tavernCandidateKey{kind: tavernCandidateMercenary, id: 3}) || s.tavernDetailType != 0 {
		t.Fatalf("reentry retained room %v selection %+v detail %d", s.room, s.tavernSelection, s.tavernDetailType)
	}
}

func TestTownMercenaryTextPathEnumeratesExactlyTheShippedCandidates(t *testing.T) {
	want := map[int]string{}
	for _, typ := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14} {
		want[typ] = fmt.Sprintf("main/text/inn/mercenary/npc%02d.txt", typ)
	}
	for typ := -1; typ <= 35; typ++ {
		got, ok := TownMercenaryTextPath(typ)
		if path, exists := want[typ]; ok != exists || got != path {
			t.Errorf("TownMercenaryTextPath(%d) = %q, %v; want %q, %v", typ, got, ok, path, exists)
		}
	}
}

func TestTavernOfferRequiresMissionUnlockAndStockAndUnaffordableHireIsAtomic(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	if got := s.tavernMercenaries(); len(got) != 1 {
		t.Fatalf("baseline offers = %#v", got)
	}

	chapter := f.Town.camp.Chapters[30]
	chapter.Mercenaries = nil
	f.Town.camp.Chapters[30] = chapter
	if got := s.tavernMercenaries(); len(got) != 0 {
		t.Fatalf("type omitted from mission list is visible: %#v", got)
	}
	chapter.Mercenaries = []int{3}
	f.Town.camp.Chapters[30] = chapter
	f.Town.mercEnabled[3] = false
	if got := s.tavernMercenaries(); len(got) != 0 {
		t.Fatalf("locked type is visible: %#v", got)
	}
	f.Town.mercEnabled[3], f.Town.mercPool[3] = true, 0
	if got := s.tavernMercenaries(); len(got) != 0 {
		t.Fatalf("empty type is visible: %#v", got)
	}

	f.Town.mercPool[3], f.Town.gold = 2, 49
	beforeParty := append([]mapload.PartyMember(nil), f.Carried...)
	if msg, ok := s.toggleMercenary(3); ok || msg == "" {
		t.Fatalf("unaffordable hire = %q, %v", msg, ok)
	}
	if f.Town.Gold() != 49 || f.Town.MercenaryPool(3) != 2 || f.Town.MercenaryHired(3) ||
		!reflect.DeepEqual(f.Carried, beforeParty) {
		t.Fatalf("unaffordable hire mutated gold=%d pool=%d hired=%v party=%#v",
			f.Town.Gold(), f.Town.MercenaryPool(3), f.Town.MercenaryHired(3), f.Carried)
	}
	sounds := s.tavernSlotRequests[tavernSlotHireRefused]
	s.pressMercenary(3)
	if got := s.tavernSlotRequests[tavernSlotHireRefused]; got != sounds+1 {
		t.Fatalf("unaffordable hire requested the refusal sound %d times, want %d", got, sounds+1)
	}
}

func TestSchoolPricesAndTrainingPersistThroughSave(t *testing.T) {
	for level, want := range map[int32]int{0: 200, 10: 518, 30: 3489, 50: 23478} {
		if got := heroSkillPrice(level); got != want {
			t.Errorf("price(%d) = %d, want %d", level, got, want)
		}
	}
	f := shellFrontEnd()
	f.townUI.room, f.townUI.schoolCell = roomSchool, 0 // fighter slot 1
	before := f.Carried[0].Hero.Skill[1]
	if msg := f.townUI.trainHeroSkill(1); msg == "" {
		t.Fatal("training returned no report")
	}
	if got := f.Carried[0].Hero.Skill[1]; got != before+1 {
		t.Fatalf("level = %d, want %d", got, before+1)
	}
	if got, want := f.Carried[0].Carry.SkillXP[1], data.SkillXPFor(before+1)+1; got != want {
		t.Fatalf("xp = %d, want %d", got, want)
	}
	if got := f.Town.Gold(); got != 482 {
		t.Fatalf("gold = %d, want 482", got)
	}
	snap, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeSave(snap, "training")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}
	other := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(shellCampaign(), nil), Table: f.Table, Humans: f.Humans}, CampaignSession: CampaignSession{Town: NewTown(shellCampaign())}}
	other.TownScreen()
	if _, inTown, err := other.Restore(decoded); err != nil || !inTown {
		t.Fatalf("restore = inTown %v, err %v", inTown, err)
	}
	if !reflect.DeepEqual(other.Carried[0].Hero, f.Carried[0].Hero) ||
		!reflect.DeepEqual(other.Carried[0].Carry, f.Carried[0].Carry) || other.Town.Gold() != 482 {
		t.Fatalf("restored training differs: %#v / %#v, gold %d", other.Carried[0], f.Carried[0], other.Town.Gold())
	}
}

func TestSchoolTrainingKeepsCanonicalCarriedAndWornItems(t *testing.T) {
	f := shellFrontEnd()
	carried := sim.ItemInstance{Code: 0x0104, Kind: 2, Price: 731,
		Effects: []sim.ItemEffect{{Kind: 17, Mode: 8, Operand: 9}}}
	worn := sim.ItemInstance{Code: 0x0205, Kind: 1, Price: 1193,
		Effects: []sim.ItemEffect{{Kind: 45, Mode: 0, Operand: 7 | 3<<8}}}
	m := &f.Carried[0]
	m.Carried, m.CarriedItems = []uint16{carried.Code}, []sim.ItemInstance{carried.Clone()}
	m.Worn[4], m.WornItems[4] = worn.Code, worn.Clone()

	if msg := f.townUI.trainHeroSkill(1); msg == "" {
		t.Fatal("training returned no report")
	}
	wantCarried := []sim.ItemInstance{carried}
	var wantWorn [sim.EquipSlots]sim.ItemInstance
	wantWorn[4] = worn
	if m.Carry == nil || !reflect.DeepEqual(m.Carry.ItemInstances, wantCarried) ||
		!reflect.DeepEqual(m.Carry.EquippedItems, wantWorn) {
		t.Fatalf("training carry = %#v, want carried %#v and worn %#v", m.Carry, wantCarried, wantWorn)
	}
	if !reflect.DeepEqual(m.Carry.Items, []uint16{carried.Code}) || m.Carry.Equipped[4] != worn.Code {
		t.Fatalf("training projections = %#v / %#v", m.Carry.Items, m.Carry.Equipped)
	}
	if got := mapload.MemberCarriedItems(*m, f.Table); !reflect.DeepEqual(got, wantCarried) {
		t.Fatalf("canonical carried reader after training = %#v, want %#v", got, wantCarried)
	}
	if got := mapload.MemberItemEquipment(*m, f.Table); !reflect.DeepEqual(got, wantWorn) {
		t.Fatalf("canonical worn reader after training = %#v, want %#v", got, wantWorn)
	}

	// The lazy Carry owns its instances. Later edits to the member's pre-carry
	// fields cannot reach the canonical mission-to-mission state it captured.
	m.CarriedItems[0].Effects[0].Operand++
	m.WornItems[4].Effects[0].Operand++
	if !reflect.DeepEqual(m.Carry.ItemInstances, wantCarried) ||
		!reflect.DeepEqual(m.Carry.EquippedItems, wantWorn) {
		t.Fatalf("training carry aliases member inputs: %#v / %#v", m.Carry.ItemInstances, m.Carry.EquippedItems)
	}
}

func TestSchoolUnaffordableTrainingIsAtomic(t *testing.T) {
	f := shellFrontEnd()
	f.Town.gold = 199
	before := f.Carried[0]
	if msg := f.townUI.trainHeroSkill(1); msg == "" {
		t.Fatal("unaffordable training returned no reason")
	}
	if f.Town.Gold() != 199 || !reflect.DeepEqual(f.Carried[0], before) {
		t.Fatalf("unaffordable training mutated gold=%d member=%#v", f.Town.Gold(), f.Carried[0])
	}
	f.townUI.room, f.townUI.schoolCell = roomSchool, 0
	if f.townUI.TownSurface().Buttons[0].Enabled {
		t.Fatal("Train is enabled below the shown price")
	}
}

func TestSchoolShowsNoGeneralAndTargetsSelectedMember(t *testing.T) {
	f := shellFrontEnd()
	f.Carried = append(f.Carried, mapload.PartyMember{Name: "Second", PlayerCharacter: true,
		Hero: data.NewCampaignHero(2)})
	s := f.townUI
	s.room = roomSchool
	view := s.TownSurface()
	if len(view.Cells) != 10 {
		t.Fatalf("school cells = %d, want two panels of five", len(view.Cells))
	}
	for _, c := range view.Cells {
		if c.Label == "General" {
			t.Fatal("General is visible")
		}
	}
	beforeFirst := f.Carried[0].Hero
	s.stepTownMember(+1)
	s.schoolCell = 1
	s.trainHeroSkill(2)
	if f.Carried[0].Hero != beforeFirst {
		t.Fatal("training the selected second member mutated the first")
	}
	if f.Carried[1].Hero.Skill[2] != 11 {
		t.Fatalf("second member slot 2 = %d, want 11", f.Carried[1].Hero.Skill[2])
	}
}

// TestSchoolPickerStepClearsThePendingSkillAndItsQuote covers TOWN-138: the
// original's school picker step writes -1 into the selected-slot field and 0
// into the price field, so a skill chosen for one member is not still pending,
// nor still quoted, for the next one. The second member is given the first
// member's class deliberately: with a different class the selection would stop
// resolving for a reason that has nothing to do with the reset.
func TestSchoolPickerStepClearsThePendingSkillAndItsQuote(t *testing.T) {
	f := shellFrontEnd()
	f.Carried = append(f.Carried, mapload.PartyMember{Name: "Second", PlayerCharacter: true,
		Mage: f.Carried[0].Mage, Hero: data.NewCampaignHero(2)})
	s := f.townUI
	s.room = roomSchool
	cell := 1
	if f.Carried[0].Mage {
		cell = 6
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell}, false)
	slot, price, ok := s.selectedSchoolSlot()
	if !ok || slot == 0 || price == 0 {
		t.Fatalf("clicking cell %d left slot %d price %d ok %v", cell, slot, price, ok)
	}
	view := s.TownSurface()
	if !view.Cells[cell].Selected || !view.Buttons[0].Enabled || view.Buttons[0].Value != "200" {
		t.Fatalf("before the step: selected %v, Train %+v", view.Cells[cell].Selected, view.Buttons[0])
	}

	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	if s.shopMemberIndex() != 1 {
		t.Fatalf("the picker did not step: member %d", s.shopMemberIndex())
	}
	if _, _, ok := s.selectedSchoolSlot(); ok {
		t.Fatal("the pending skill survived the picker step")
	}
	view = s.TownSurface()
	for i, c := range view.Cells {
		if c.Selected {
			t.Fatalf("cell %d is still selected after the picker step", i)
		}
	}
	if view.Buttons[0].Enabled || view.Buttons[0].Value != "0" {
		t.Fatalf("the quote survived the picker step: %+v", view.Buttons[0])
	}
	// Train has nothing to act on, and says so rather than buying slot 1.
	before := f.Carried[1].Hero
	if msg := s.townSurfaceButton(0); msg.Msg != "" {
		t.Fatalf("Train after the step = %q", msg.Msg)
	}
	if f.Carried[1].Hero != before {
		t.Fatal("Train mutated a hero with no pending skill")
	}

	// The shop and the tavern share this one step routine and are not its
	// subject: a step taken in the tavern leaves the school's own pending
	// selection alone, and the tavern's own cell is untouched by either.
	wantTavern := tavernCandidateKey{kind: tavernCandidateMercenary, id: 3}
	s.schoolCell, s.tavernSelection, s.room = cell, wantTavern, roomTavern
	s.stepTownMember(+1)
	if s.schoolCell != cell || s.tavernSelection != wantTavern {
		t.Fatalf("a tavern step wrote schoolCell %d tavernSelection %+v", s.schoolCell, s.tavernSelection)
	}
	s.room = roomShop
	s.shopStepMember(+1)
	if s.schoolCell != cell {
		t.Fatalf("a shop step cleared the school's pending selection: %d", s.schoolCell)
	}
}

// TestSchoolOpensWithNothingSelected covers the entry state of the school room.
//
// schoolCell's zero value is a real skill, so a freshly built town screen drew
// the first icon of the shown class lit and quoted its price on the Train
// button before the player had selected anything. Measured against the shipped
// EN campaign before the fix: entering the school showed cell 0 (Blade)
// Selected with Train reading 518. A party-picker step in the same room already
// cleared the selection (TOWN-138), so the two states disagreed about what an
// unselected school looks like. Both the constructor and resetForNewGame now
// start at schoolNoSelection.
func TestSchoolOpensWithNothingSelected(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	if s.schoolCell != schoolNoSelection {
		t.Fatalf("a freshly built town screen holds schoolCell %d, want %d", s.schoolCell, schoolNoSelection)
	}
	s.room = roomSchool
	v := s.TownSurface()
	for i, c := range v.Cells {
		if c.Selected {
			t.Fatalf("cell %d (%s) is selected before any click", i, c.Label)
		}
	}
	if v.Buttons[0].Value != "0" || v.Buttons[0].Enabled {
		t.Fatalf("Train quotes %q enabled=%v with nothing selected, want \"0\" and disabled",
			v.Buttons[0].Value, v.Buttons[0].Enabled)
	}
	// A click still selects, and the quote follows it.
	if a := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false); a.Msg != "" {
		t.Fatalf("selecting cell 0 reported %q", a.Msg)
	}
	v = s.TownSurface()
	if !v.Cells[0].Selected || v.Buttons[0].Value == "0" {
		t.Fatalf("after a click cell 0 selected=%v Train=%q, want selected and a price",
			v.Cells[0].Selected, v.Buttons[0].Value)
	}
}
