package game

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// townFixture is a scenario shaped like the shipped one at the two places this
// story reads: a chapter whose inn pairs InnNPC with InnMission and carries the
// zero sentinel, and a chapter whose main mission comes from the SHOP instead —
// which is what 2 of the 13 shipped chapters do (REG-SCN-064).
//
// EVERY BYTE IS SYNTHETIC (golden rule 2). The SHAPE is taken from the corpus
// census in REG-SCN-064 and REG-SCN-059; the numbers are this file's own.
func townFixture() []byte {
	return synth.Reg(kindRoot, []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{
			{Name: "TotalMissions", Kind: kindInt, Int: 5},
		}},
		// Two missions no building names — the prologue, which is what makes
		// the town's own boundary 30 rather than 10.
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{
			{Name: "AutoGetMission", Kind: kindInt, Int: 20},
		}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mercenaries", Kind: kindInt, Int: 1},
		}},
		// The first town chapter: NPC 22 holds the main mission, NPC 90 has
		// nothing to give, the shop holds the side mission, the school holds
		// nothing.
		{Name: "Mission30", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnNPC", Kind: kindIntArray, Ints: []int32{22, 90}},
			{Name: "InnMission", Kind: kindIntArray, Ints: []int32{30, 0}},
			{Name: "ShopMission", Kind: kindInt, Int: 31},
			{Name: "ShopMinPrice", Kind: kindInt, Int: 0},
			{Name: "ShopMaxPrice", Kind: kindInt, Int: 1000},
			{Name: "Payment", Kind: kindInt, Int: 1000},
		}},
		{Name: "Mission31", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Payment", Kind: kindInt, Int: 700},
		}},
		// The second: the SHOP hands out the main mission and the school a
		// side one, which is the arrangement REG-SCN-064 finds in the other
		// two shipped chapters.
		{Name: "Mission40", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnNPC", Kind: kindInt, Int: 23},
			{Name: "InnMission", Kind: kindInt, Int: 0},
			{Name: "ShopMission", Kind: kindInt, Int: 40},
			{Name: "TCMission", Kind: kindInt, Int: 41},
			{Name: "ShopMinPrice", Kind: kindInt, Int: 200},
			{Name: "ShopMaxPrice", Kind: kindInt, Int: 5000},
		}},
		{Name: "Mission41", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Payment", Kind: kindInt, Int: 900},
		}},
	})
}

func townCampaign(t *testing.T) Campaign {
	t.Helper()
	r, err := reg.Parse(townFixture())
	if err != nil {
		t.Fatalf("parse the town fixture: %v", err)
	}
	return ReadCampaign(r)
}

func newTownFixture(t *testing.T) *Town {
	t.Helper()
	town := NewTown(townCampaign(t))
	town.Arrive()
	return town
}

// The chapter is read out of the registry and never written, and the whole
// of what the reader had to keep is the three lists KEPT APART with the zero
// sentinel still in place.
//
// EVERY EXPECTATION IS THE WHOLE ARRAY. A reader that compacted the inn's
// arrays would produce a plausible chapter — one shorter pair, still paired —
// and a length assertion would not see it, while the NPC holding each mission
// would silently be the wrong one.
func TestAChapterKeepsTheThreeListsApartAndTheSentinelInPlace(t *testing.T) {
	c := townCampaign(t)

	ch, ok := c.Chapters[30]
	if !ok {
		t.Fatal("no chapter read for mission 30")
	}
	if want := []int{22, 90}; !reflect.DeepEqual(ch.InnNPC, want) {
		t.Errorf("InnNPC = %v, want %v", ch.InnNPC, want)
	}
	if want := []int{30, 0}; !reflect.DeepEqual(ch.Inn, want) {
		t.Errorf("Inn = %v, want %v — the zero sentinel is a POSITION, not an absence", ch.Inn, want)
	}
	if want := []int{31}; !reflect.DeepEqual(ch.Shop, want) {
		t.Errorf("Shop = %v, want %v", ch.Shop, want)
	}
	if ch.School != nil {
		t.Errorf("School = %v, want nothing — this chapter declares no TCMission", ch.School)
	}
	if ch.ShopMin != 0 || ch.ShopMax != 1000 || ch.Payment != 1000 {
		t.Errorf("min/max/payment = %d/%d/%d, want 0/1000/1000", ch.ShopMin, ch.ShopMax, ch.Payment)
	}

	// A mercenary roster is town data even when the section carries no mission
	// dialogue.
	if got := c.Chapters[20].Mercenaries; !reflect.DeepEqual(got, []int{1}) {
		t.Errorf("mission 20 Mercenaries = %v, want [1]", got)
	}

	// And the merged set the campaign already computed did not move.
	if want := []int{30, 31, 40, 41}; !reflect.DeepEqual(c.Offered, want) {
		t.Errorf("Offered = %v, want %v — Chapters must not have changed it", c.Offered, want)
	}
}

func TestTheCurrentChapterAdvancesOnlyOnMainCompletion(t *testing.T) {
	town := newTownFixture(t)

	if got := town.Chapter(); got != 30 {
		t.Fatalf("chapter %d after first arrival, want 30", got)
	}
	town.Won(31)
	if got := town.Chapter(); got != 30 {
		t.Errorf("chapter %d after the SIDE mission 31, want 30", got)
	}
	town.Won(30)
	if got := town.Chapter(); got != 40 {
		t.Errorf("chapter %d after the MAIN mission 30, want 40", got)
	}
	town.Won(40)
	if got := town.Chapter(); got != 0 {
		t.Errorf("chapter %d with nothing left to offer, want 0", got)
	}
}

// Winning pays what the mission's own section declares, once, and a mission
// with no Payment key pays nothing.
func TestWinningPaysOnceAndOnlyWhatTheSectionDeclares(t *testing.T) {
	town := newTownFixture(t)
	if got := town.Gold(); got != initialPlayerPurse {
		t.Fatalf("fresh participant purse = %d, want %d", got, initialPlayerPurse)
	}

	town.Won(30)
	if got := town.Gold(); got != initialPlayerPurse+1000 {
		t.Fatalf("gold %d after mission 30, want %d", got, initialPlayerPurse+1000)
	}
	town.Won(30)
	if got := town.Gold(); got != initialPlayerPurse+1000 {
		t.Errorf("gold %d after winning mission 30 twice, want %d — Won must be idempotent", got, initialPlayerPurse+1000)
	}
	town.Won(21)
	if got := town.Gold(); got != initialPlayerPurse+1000 {
		t.Errorf("gold %d after a mission absent from the fixture, want %d", got, initialPlayerPurse+1000)
	}
}

func TestTheAuthoredTownTransitionRewardIsSeparateConfigurableAndPaidOnce(t *testing.T) {
	c := Campaign{Main: []int{10, 20, 30}, Offered: []int{30}}
	c = WithDefaultTransitionRewards(c)
	if got := c.Chapters[20].Payment; got != 0 {
		t.Fatalf("scenario Payment for mission 20 = %d, want absent/zero", got)
	}
	if got := c.TransitionRewards[20]; got != firstTownTransitionReward {
		t.Fatalf("authored transition reward = %d, want %d", got, firstTownTransitionReward)
	}
	town := NewTown(c)
	town.Won(20)
	town.Won(20)
	if got := town.Gold(); got != initialPlayerPurse+firstTownTransitionReward {
		t.Fatalf("purse after transition won twice = %d, want exactly-once %d", got, initialPlayerPurse+firstTownTransitionReward)
	}
	var snap Snapshot
	snapshotTown(town, &snap)
	restored := restoreTown(c, snap)
	restored.Won(20)
	if got := restored.Gold(); got != initialPlayerPurse+firstTownTransitionReward {
		t.Fatalf("restored purse after replayed transition = %d, want persisted exactly-once %d", got, initialPlayerPurse+firstTownTransitionReward)
	}

	c.TransitionRewards = map[int]int{20: 37}
	configured := NewTown(c)
	configured.Won(20)
	if got := configured.Gold(); got != initialPlayerPurse+37 {
		t.Fatalf("configured transition purse = %d, want %d", got, initialPlayerPurse+37)
	}
}

// The tavern is per NPC, pairs positionally, and its zero entry is an NPC
// who is LISTED, can be spoken to, gives nothing and is not consumed
// (REG-SCN-064).
func TestTheTavernIsPerNPCAndTheZeroEntryIsNotConsumed(t *testing.T) {
	town := newTownFixture(t)

	offers := town.Offers(TownTavern)
	want := []TownOffer{{Index: 0, Mission: 30, NPC: 22}, {Index: 1, Mission: 0, NPC: 90}}
	if !reflect.DeepEqual(offers, want) {
		t.Fatalf("tavern offers %+v, want %+v", offers, want)
	}

	// The sentinel: nothing accepted, nothing consumed, and he is still there.
	if m, ok := town.Take(TownTavern, 1); ok || m != 0 {
		t.Errorf("Take on the sentinel = (%d, %v), want (0, false)", m, ok)
	}
	if len(town.Offers(TownTavern)) != 2 {
		t.Error("the sentinel NPC was consumed — only an ACCEPTED mission is removed")
	}

	// The real one: accepted once, then gone from the shelf and on the list.
	m, ok := town.Take(TownTavern, 0)
	if !ok || m != 30 {
		t.Fatalf("Take(tavern, 0) = (%d, %v), want (30, true)", m, ok)
	}
	if got := town.Available(); !reflect.DeepEqual(got, []int{30}) {
		t.Errorf("available %v, want [30]", got)
	}
	if got := town.Offers(TownTavern); len(got) != 1 || got[0].NPC != 90 {
		t.Errorf("tavern offers %+v after accepting, want only NPC 90", got)
	}
	if m, ok := town.Take(TownTavern, 0); ok {
		t.Errorf("the same entry was accepted twice, second = %d", m)
	}
}

// The three buildings write ONE list and the gates read it, and nothing a
// building hands out survives being won.
func TestThreeBuildingsWriteOneListAndAWinLeavesIt(t *testing.T) {
	town := newTownFixture(t)

	if got := town.Available(); len(got) != 0 {
		t.Fatalf("available %v in a town nobody has visited, want nothing", got)
	}
	if _, ok := town.Take(TownTavern, 0); !ok {
		t.Fatal("the tavern gave nothing")
	}
	if _, ok := town.Take(TownShop, 0); !ok {
		t.Fatal("the shop gave nothing")
	}
	if got := town.Available(); !reflect.DeepEqual(got, []int{30, 31}) {
		t.Fatalf("available %v, want [30 31] — one list, two producers", got)
	}

	town.Won(31)
	if got := town.Available(); !reflect.DeepEqual(got, []int{30}) {
		t.Errorf("available %v after winning 31, want [30]", got)
	}

	// The school's list belongs to the NEXT chapter, so it has nothing here —
	// which is what makes a chapter a chapter.
	if got := town.Offers(TownSchool); len(got) != 0 {
		t.Errorf("school offers %+v in chapter 30, want none", got)
	}
}

// Every method answers for a nil receiver (Town's own contract), so a
// hand-built front end is one that has not reached a town rather than one that
// panics.
func TestANilTownAnswersEverything(t *testing.T) {
	var town *Town
	town.Arrive()
	town.Won(30)
	if town.Open() || town.Gold() != 0 || town.Done(30) || town.Chapter() != 0 ||
		town.Available() != nil || town.Offers(TownTavern) != nil {
		t.Error("a nil town answered something")
	}
	if m, ok := town.Take(TownTavern, 0); ok || m != 0 {
		t.Errorf("Take on a nil town = (%d, %v)", m, ok)
	}
}

func townTextFS(t *testing.T, files []synth.File) *vfs.FS {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, MainArchive)
	if err := os.WriteFile(path, synth.Archive(files), 0o600); err != nil {
		t.Fatalf("write synthetic main archive: %v", err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return fsys
}

func TestTownNPCTextPathNamesTheInnLeaf(t *testing.T) {
	if got, ok := TownNPCTextPath(7, 30); !ok || got != "main/text/inn/npc/npc07m30.txt" {
		t.Fatalf("TownNPCTextPath(7,30) = %q/%v", got, ok)
	}
	if got, ok := TownBuildingTextPath(TownShop, 31); !ok || got != "main/text/shop/npc31m31.txt" {
		t.Fatalf("shop text path = %q/%v", got, ok)
	}
	if got, ok := TownBuildingTextPath(TownSchool, 32); !ok || got != "main/text/training/npc34m32.txt" {
		t.Fatalf("school text path = %q/%v", got, ok)
	}
}

func TestShopDialogueUsesThePortraitNoticeWindow(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Font: resolved(missionFont(), nil), Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte("<part=1,npc=22>\r\nShipped shop line")}})}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	s := f.TownScreen().(*townScreen)
	s.Choose(1)
	pic, ok := s.TownDialogue()
	if !ok || pic == nil {
		t.Fatal("shop dialogue did not compose the shared portrait notice")
	}
	if pic.Bounds().Size() != ui.AuthoredDialogueLayout().WithPortrait(true).Box.Size() {
		t.Fatalf("dialogue size = %v, want portrait layout %v", pic.Bounds().Size(), ui.AuthoredDialogueLayout().WithPortrait(true).Box.Size())
	}
}

// The whole walk the owner asked for, at the model and wiring tiers: arrive,
// find the gates empty, talk an NPC through his lines, accept, and find the
// mission at the gates (AC-2, AC-5).
//
// IT DRIVES THE SCREEN AND NOT THE MODEL. Every press below goes through
// Choose and Back — the two calls pkg/ui makes and the only two that mutate —
// so what is witnessed is the rooms a player walks through, not a shortcut
// into the state behind them.
func TestTheTavernWalkPutsAMissionAtTheGates(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/inn/npc/npc22m30.txt", Data: []byte("<part=1>\r\nFirst real line<part=2>\r\nSecond real line")}})}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	s := f.TownScreen()

	// The square lists four doors and the gates are the last of them.
	if n := len(s.Rows()); n != 4 {
		t.Fatalf("the square has %d rows, want 4", n)
	}

	// The gates with nothing on offer keep the town view; no world map opens.
	s.Choose(3)
	if ts := s.(*townScreen); ts.AtWorldMap() || !ts.AtTownSquare() || len(s.Rows()) != 4 {
		t.Fatalf("the gate press with nothing on offer left the square: room %v, %d rows", ts.room, len(s.Rows()))
	}

	// The tavern, then NPC 22, then his lines to the end.
	s.Choose(0)
	if n := len(s.Rows()); n != 2 {
		t.Fatalf("the tavern lists %d NPCs, want 2", n)
	}
	s.Choose(0)
	ts := s.(*townScreen)
	if rows := s.Rows(); len(rows) != 0 {
		t.Fatalf("dialogue exposed legacy town controls: %+v", rows)
	}
	first := ts.AdvanceTownDialogue()
	if first.Msg != "" {
		t.Fatalf("the first OK press ended the two-part dialogue: %+v", first)
	}
	act := ts.AdvanceTownDialogue()
	if act.Msg != "" {
		t.Errorf("the last press posted %q; the original's close posts no line", act.Msg)
	}
	// Nothing is committed inside the tavern: the mission is queued and leaving
	// registers it.
	if got := f.Town.Available(); len(got) != 0 {
		t.Fatalf("available %v inside the tavern after the conversation, want none", got)
	}

	// The gates now show it as a mission scroll. This fixture intentionally has
	// no global-map registry, so the scroll states why it cannot be opened.
	if !s.Back() {
		t.Fatal("Back in the tavern reported it left no room")
	}
	if got := f.Town.Available(); !reflect.DeepEqual(got, []int{30}) {
		t.Fatalf("available %v after leaving the tavern, want [30]", got)
	}
	s.Choose(3)
	view := ts.WorldMapView()
	if len(view.Missions) != 1 || view.Missions[0].Number != 30 {
		t.Fatalf("world-map missions = %+v, want mission 30", view.Missions)
	}
	if view.Missions[0].Enabled || view.Missions[0].Problem == "" {
		t.Fatalf("mission without registry mapping = %+v, want an explicit disabled reason", view.Missions[0])
	}

	// The square is where Back reports it has run out of rooms, which is what
	// lets pkg/ui leave the town without knowing what a room is.
	s.Back()
	if s.Back() {
		t.Error("Back at the square reported it left a room")
	}
}

// The shop states its own prices and hands out the HEAD of its list, and
// only the head (AC-6).
func TestTheShopStatesItsPricesAndGivesTheHeadOnlyOnce(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte("<part=1>\r\nShipped shop offer")}})}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	s := f.TownScreen()

	s.Choose(1) // the shop opens its head offer's dialogue
	if rows := s.Rows(); len(rows) != 0 {
		t.Fatalf("the shop dialogue exposed legacy controls: %+v", rows)
	}

	if act := s.(*townScreen).AdvanceTownDialogue(); act.Msg != "" {
		t.Errorf("accepting the shop's offer posted %q; the original's close posts no line", act.Msg)
	}
	if got := f.Town.Available(); !reflect.DeepEqual(got, []int{31}) {
		t.Fatalf("available %v, want [31]", got)
	}
	// Past the head there is nothing left to take.
	if offers := f.Town.Offers(TownShop); len(offers) != 0 {
		t.Errorf("the shop still offers %+v after the head was taken", offers)
	}
}

func TestMissingTownDialogueDoesNotOpenOrConsumeAnOffer(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []synth.File
	}{
		{name: "missing payload"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := townCampaign(t)
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Font: resolved(missionFont(), nil), Archives: &Archives{Containers: townTextFS(t, tc.files)}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
			f.Town.Arrive()
			s := f.TownScreen().(*townScreen)

			s.Choose(1)
			if s.room != roomShop {
				t.Fatalf("missing shipped dialogue entered room %v, want the shop", s.room)
			}
			if pic, ok := s.TownDialogue(); ok || pic != nil {
				t.Fatal("missing shipped dialogue produced an authored substitute")
			}
			if act := s.AdvanceTownDialogue(); act.Msg != "" || act.Open != nil {
				t.Fatalf("advancing absent dialogue acted: %+v", act)
			}
			if got := f.Town.Available(); len(got) != 0 {
				t.Fatalf("absent dialogue consumed its offer into the gates: %v", got)
			}
			if offers := f.Town.Offers(TownShop); len(offers) != 1 || offers[0].Mission != 31 {
				t.Fatalf("absent dialogue changed shop offers: %+v", offers)
			}
		})
	}
}

// TestTownOfferIsRegisteredOnceWhenItsRoomOpensIt witnesses that the school's
// and the shop's offer opens on room entry and is registered as it opens, so
// the once-per-mission gate is Town.taken alone and no key decides it: Enter
// and Escape both end the conversation, and a later entry finds nothing to
// open.
func TestTownOfferIsRegisteredWhenItsRoomOpensIt(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Font: resolved(missionFont(), nil), Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/training/npc34m41.txt", Data: []byte("<part=1>\r\nThe trainer has a job")}, {Path: "text/shop/npc31m40.txt", Data: []byte("<part=1>\r\nThe merchant has a job")}})}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	// Chapter 40's own side missions: TCMission 41 for the school, ShopMission
	// 40 for the shop (townFixture's own comment). Chapter 30 offers neither
	// building anything, so its main mission is won first.
	f.Town.Won(30)
	if got := f.Town.Chapter(); got != 40 {
		t.Fatalf("chapter %d after winning 30, want 40", got)
	}
	s := f.TownScreen().(*townScreen)

	// School: entering the room opens its offer and registers it, before a key.
	s.Choose(2)
	if s.room != roomTalk || s.dialogueBuilding != TownSchool {
		t.Fatalf("entering the school with an offer stayed at room %v, want an open dialogue", s.room)
	}
	if offers := f.Town.Offers(TownSchool); len(offers) != 0 || !containsMission(f.Town.Available(), 41) {
		t.Fatalf("with the school's conversation open, offers %+v and available %v; want none and mission 41", offers, f.Town.Available())
	}
	if act := s.AdvanceTownDialogue(); act.Msg != "" {
		t.Errorf("ending the school's conversation posted %q; the original's close posts no line", act.Msg)
	}
	if s.room != roomSchool || !containsMission(f.Town.Available(), 41) {
		t.Fatalf("after the school's conversation: room %v, available %v", s.room, f.Town.Available())
	}
	// Re-entering the school does not reopen a dialogue: nothing is left to
	// offer.
	s.Back()
	s.Choose(2)
	if s.room != roomSchool {
		t.Fatalf("re-entering the school with nothing left to offer opened room %v", s.room)
	}

	// Shop: entering the room opens and registers its offer, the same. Escape
	// ends the conversation as the button does and registers nothing more.
	s.Back()
	s.Choose(1)
	if s.room != roomTalk || s.dialogueBuilding != TownShop {
		t.Fatalf("entering the shop with an offer stayed at room %v, want an open dialogue", s.room)
	}
	if offers := f.Town.Offers(TownShop); len(offers) != 0 || !containsMission(f.Town.Available(), 40) {
		t.Fatalf("with the shop's conversation open, offers %+v and available %v; want none and mission 40", offers, f.Town.Available())
	}
	if !s.Back() {
		t.Fatal("Back from the shop's open dialogue reported no motion")
	}
	if s.room != roomShop {
		t.Fatalf("escaping the shop dialogue left room %v, want the shop", s.room)
	}
	if offers := f.Town.Offers(TownShop); len(offers) != 0 || !containsMission(f.Town.Available(), 40) {
		t.Fatalf("after escaping the shop's conversation, offers %+v and available %v; want none and mission 40", offers, f.Town.Available())
	}
	// A press on the merchant reaches no handler and opens nothing.
	if act := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlMerchant}); act.Open != nil || act.Msg != "" || s.room != roomShop {
		t.Fatalf("the merchant press answered %q and left room %v, want no answer in the shop", act.Msg, s.room)
	}
	// Entering again opens no dialogue.
	s.Back()
	s.Choose(1)
	if s.room != roomShop {
		t.Fatalf("re-entering the shop after its offer was registered opened room %v, want the shop", s.room)
	}
}

// Entering the shop shows the five-place table, and the table starts empty:
// walking in takes nothing off a shelf and puts nothing down.
func TestEnteringTheShopShowsTheEmptyFivePlaceTable(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c), Shop: NewShop(1000), Carried: []mapload.PartyMember{{Carry: &mapload.Carry{Items: []uint16{9, 9, 5}}}}}}
	f.Town.gold = 500
	s := f.TownScreen().(*townScreen)

	s.Choose(1)

	if !s.AtTownShop() {
		t.Fatal("entering the shop did not activate its table")
	}
	got := s.ShopScreen()
	for i, cell := range got.Table {
		if cell.Occupied() {
			t.Fatalf("the table opened holding %+v at place %d", cell, i)
		}
	}
	if got.Purse != 500 || got.Buy != 0 || got.Sell != 0 {
		t.Fatalf("the buttons print %+v, want the 500 purse and two zero totals", got)
	}

	// The player's own pack is drawn in its own grid, so what he walked in
	// with is on screen the moment he arrives.
	pack := 0
	for _, cell := range got.Pack {
		if cell.Occupied() {
			pack++
		}
	}
	if pack != 2 {
		t.Fatalf("the pack grid shows %d places, want the two stacks he carries", pack)
	}
}

// Every room states something and every room can be left (AC-5).
func TestEveryRoomSaysWhatItIsAndCanBeLeft(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	f.Town.announceMission(30) // the gates open a room only while a mission is on offer
	s := f.TownScreen()

	for door := 0; door < 4; door++ {
		s.Choose(door)
		if s.Header() == "" {
			t.Errorf("door %d opened a room with no header", door)
		}
		// The shop is drawn as its own screen and has no rows at all; a
		// conversation draws its own window over the room.
		room := s.(*townScreen).room
		if len(s.Rows()) == 0 && room != roomTalk && room != roomShop && room != roomGates {
			t.Errorf("door %d opened a room with no rows — an empty rectangle", door)
		}
		if len(s.Footer()) == 0 {
			t.Errorf("door %d opened a room stating nothing about the player", door)
		}
		if !s.Back() {
			t.Errorf("door %d opened a room that could not be left", door)
		}
		if n := len(s.Rows()); n != 4 {
			if !s.Back() || len(s.Rows()) != 4 {
				t.Errorf("leaving door %d did not return to the square (%d rows)", door, n)
			}
		}
	}
}

// townScreen satisfies the seam, and the seam names no simulation type. The
// compile is the assertion; the body is what makes it one an author cannot
// delete by accident.
func TestTheTownCrossesTheSeam(t *testing.T) {
	var _ ui.TownScreen = (*townScreen)(nil)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(townCampaign(t), nil)}}
	f.Town = NewTown(f.Campaign.Value())
	if f.TownScreen() == nil {
		t.Fatal("TownScreen built nothing")
	}
}

// The headless report of a win that ends in the town states the chapter and
// what each building holds there, rather than "opens nothing".
//
// IT IS THE LINE AND NOT AdvanceLine, because AdvanceLine starts a mission out
// of an archive and this package's tests read no install (golden rule 2). What
// is under test is the composition, over a town driven to exactly the state a
// win at the boundary leaves it in.
func TestTheHeadlessLineNamesTheTownAndItsBuildings(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()

	line := f.townLine(20)

	for _, want := range []string{
		"leads to the town, chapter 30",
		"tavern holds npc 22 -> 30, npc 90 -> 0",
		"shop holds 31",
		"shop prices 0-1000",
		"gold 100",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("the line %q does not carry %q", line, want)
		}
	}
	if strings.Contains(line, "school") {
		t.Errorf("the line %q names the school, which holds nothing in chapter 30", line)
	}
	if strings.Contains(line, "opens nothing") {
		t.Errorf("the line %q still says a win into the town opens nothing", line)
	}
}
