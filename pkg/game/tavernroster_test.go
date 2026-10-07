package game

import (
	"fmt"
	"image"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// ownerTavernMercenaries is the mission-130 owner SAV's `Mercenaries` list
// (SAV-1111): it names what the tavern offers, not the cell order.
var ownerTavernMercenaries = []int{14, 6, 10, 13, 4, 8, 7, 3, 1, 9, 2, 12, 5}

// ownerTavernFront is a tavern whose thirteen eligible types and one InnNPC
// element match the owner's mission-130 roster.
func ownerTavernFront(t *testing.T, party []mapload.PartyMember) *FrontEnd {
	t.Helper()
	nodes := make([]synth.RegNode, 0, 15)
	counts := make([]int, 15)
	for typ := 1; typ <= 15; typ++ {
		nodes = append(nodes, synth.RegNode{Name: fmt.Sprintf("npc%d", typ), Kind: kindDir, Children: []synth.RegNode{
			{Name: "DataBinID", Kind: kindInt, Int: int32(typ)}, {Name: "PriceA", Kind: kindInt, Int: 1},
			{Name: "PriceB", Kind: kindInt, Int: 2}}})
		counts[typ-1] = 1
	}
	r, err := reg.Parse(synth.Reg(kindRoot, nodes))
	if err != nil {
		t.Fatal(err)
	}
	c := Campaign{Main: []int{130}, Offered: []int{130}, MercenaryCount: counts,
		Chapters: map[int]Chapter{
			120: {Mission: 120, EnableMercenary: ownerTavernMercenaries},
			130: {Mission: 130, InnNPC: []int{25}, Inn: []int{0}, Mercenaries: ownerTavernMercenaries},
		}}
	town := NewTown(c)
	town.Won(120)
	town.Arrive()
	town.gold = 10_000_000
	humans := dbCollection{{}, {name: "NPC03_1", params: humansParams(30, 0, 1)}}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil),
		Table: &mapload.Table{Humans: humans, NPC: data.LoadNPCDefs(r)}, Humans: humans, Words: ui.AuthoredWords()},
		CampaignSession: CampaignSession{Town: town, Carried: party}}
	f.townUI = f.TownScreen().(*townScreen)
	f.townUI.room = roomTavern
	f.townUI.composeShopFaces()
	return f
}

// TestTavernRosterIsTheStockWalkThenTheTalkCells pins TAVERN-ORDER-015 and
// TOWN-468 on the owner's roster: mercenary cells in the stock actor-map walk,
// the talk-only cell after them at position 13, drawn in rect 13 (TOWN-467).
func TestTavernRosterIsTheStockWalkThenTheTalkCells(t *testing.T) {
	f := ownerTavernFront(t, []mapload.PartyMember{{Name: "Player", PlayerCharacter: true, StartingHero: true}})
	v := f.townUI.TownSurface()
	got := make([]string, len(v.Cells))
	for i, c := range v.Cells {
		got[i] = c.Semantic
	}
	want := []string{}
	for _, typ := range []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 14, 13, 12} {
		want = append(want, fmt.Sprintf("Mercenary %d", typ))
	}
	want = append(want, "NPC 25")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roster = %v\nwant     %v", got, want)
	}
	for i, p := range map[int]image.Point{0: {200, 450}, 5: {440, 450}, 6: {200, 380}, 12: {200, 320}, 13: {240, 320}} {
		if c, ok := ui.TownSurfaceControlAt(v, p); !ok || c.Index != i {
			t.Fatalf("point %v hits %+v, %v; want position %d", p, c, ok, i)
		}
	}
	if !v.Cells[0].Selected {
		t.Fatal("activation did not select position 0")
	}
}

func brianParty() []mapload.PartyMember {
	return []mapload.PartyMember{
		{Name: "Player", PlayerCharacter: true, StartingHero: true, Mage: true,
			FigureDir: string(data.FigureDirWomanMage), FigureFace: 2, Hero: data.NewCampaignHero(1)},
		{Name: "Brian", PlayerCharacter: true, CompanionNPC: 25,
			FigureDir: string(data.FigureDirManFighter), FigureFace: 1, Hero: data.NewCampaignHero(2)},
	}
}

var brianRecord = data.NPCFace{Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 1,
	Tokens: data.NPCTokens(data.NPCTokenHero) | data.NPCTokens(data.NPCTokenFace) |
		data.NPCTokens(data.NPCTokenNotFemale) | data.NPCTokens(data.NPCTokenNotMage)}

// TestTavernTalkCellDrawsItsObjectsSheet is TAVERN-TALKPIC-016: a Hero object
// draws HeroFighter or HeroMage by its mage bit, any other the Unit<+0x15b>
// sheet, and an unshipped sheet draws nothing.
func TestTavernTalkCellDrawsItsObjectsSheet(t *testing.T) {
	art := &ui.TownTavernArt{}
	sheet := func() []image.Image { return []image.Image{image.NewRGBA(image.Rect(0, 0, 48, 64))} }
	art.HeroFrames[0], art.HeroFrames[1] = sheet(), sheet()
	art.UnitFrames[41], art.UnitFrames[10] = sheet(), sheet()
	same := func(a, b []image.Image) bool { return len(a) > 0 && len(b) > 0 && &a[0] == &b[0] }

	live := resolveTavernTalk(25, brianRecord, true, brianParty(), nil)
	if live.live != 1 || !live.statistics() || !same(tavernTalkFrames(art, live), art.HeroFrames[0]) {
		t.Fatalf("Brian = %+v, want the live member 1 on HeroFighter", live)
	}
	synthHero := resolveTavernTalk(25, brianRecord, true, brianParty()[:1], nil)
	if synthHero.live != -1 || synthHero.statistics() || !same(tavernTalkFrames(art, synthHero), art.HeroFrames[0]) {
		t.Fatalf("unmatched Hero record = %+v, want synthesised on HeroFighter", synthHero)
	}
	mage := brianRecord
	mage.Tokens = data.NPCTokens(data.NPCTokenHero) | data.NPCTokens(data.NPCTokenMage) | data.NPCTokens(data.NPCTokenNotHuman)
	if o := resolveTavernTalk(22, mage, true, brianParty(), nil); !same(tavernTalkFrames(art, o), art.HeroFrames[1]) {
		t.Fatalf("Hero mage record = %+v, want HeroMage", o)
	}
	plain := data.NPCFace{Tokens: data.NPCTokens(data.NPCTokenNotHuman)}
	if o := resolveTavernTalk(41, plain, true, brianParty(), nil); !same(tavernTalkFrames(art, o), art.UnitFrames[41]) {
		t.Fatalf("npc41 = %+v, want its own Unit41 sheet", o)
	}
	if o := resolveTavernTalk(59, plain, true, brianParty(), nil); tavernTalkFrames(art, o) != nil {
		t.Fatal("npc59 drew a sheet; Unit59 does not ship")
	}
	hired := []mapload.PartyMember{{Name: "Guard", MercenaryType: 10}}
	if o := resolveTavernTalk(90, data.NPCFace{}, true, hired, nil); o.live != 0 || !same(tavernTalkFrames(art, o), art.UnitFrames[10]) {
		t.Fatalf("live stock mercenary = %+v, want Unit10", o)
	}
}

// TestTavernTalkCellShowsStatisticsOnlyForALiveObject is TAVERN-TALKSTATS-017:
// the owner's Brian shows his statistics; a synthesised object shows none.
func TestTavernTalkCellShowsStatisticsOnlyForALiveObject(t *testing.T) {
	f := ownerTavernFront(t, brianParty())
	f.NPCFaces = map[int32]data.NPCFace{25: brianRecord}
	f.TownTavernArt = resolved(&ui.TownTavernArt{LeftStats: image.NewRGBA(image.Rect(0, 0, 160, 238))}, nil)
	s := f.townUI
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 13}, false)
	v := s.TownSurface()
	if !v.Candidate.HasSubject || v.Candidate.Subject.Char.Name != "Brian" || v.CandidatePixels == nil {
		t.Fatalf("Brian's talk cell = subject %v %q pixels %v, want his statistics",
			v.Candidate.HasSubject, v.Candidate.Subject.Char.Name, v.CandidatePixels != nil)
	}

	f.Carried = f.Carried[:1]
	s.composeShopFaces()
	v = s.TownSurface()
	if v.Candidate.HasSubject {
		t.Fatal("a synthesised talk object showed statistics")
	}
}

// TestSchoolButtonsGroupTheirValues: the school widget values go through the
// grouping routine (TOWN-469).
func TestSchoolButtonsGroupTheirValues(t *testing.T) {
	f := shellFrontEnd()
	f.townUI.room = roomSchool
	if got := f.townUI.TownSurface().Buttons[1].Value; got != "1,000" {
		t.Fatalf("school Exit value = %q, want 1,000", got)
	}
}

// TestTavernTalkResolvesOverTheStockUnits: a Human record no party member
// answers resolves to the stock mercenary unit it admits, which draws its type
// sheet and counts as live (TAVERN-TALKPIC-016, TAVERN-ORDER-015).
func TestTavernTalkResolvesOverTheStockUnits(t *testing.T) {
	art := &ui.TownTavernArt{}
	art.UnitFrames[8], art.UnitFrames[10] = []image.Image{image.NewRGBA(image.Rect(0, 0, 1, 1))}, []image.Image{image.NewRGBA(image.Rect(0, 0, 1, 1))}
	stock := []tavernStockActor{
		{typ: 10, actor: speakerActor{fig: figureID{Dir: data.FigureDirWomanFighter, Face: 2}, face: 1}},
		{typ: 8, actor: speakerActor{fig: figureID{Dir: data.FigureDirManFighter, Face: 2}, face: 2}},
	}
	rec := data.NPCFace{Face: 2, Tokens: data.NPCTokens(data.NPCTokenHuman) | data.NPCTokens(data.NPCTokenFace) |
		data.NPCTokens(data.NPCTokenNotFemale)}
	o := resolveTavernTalk(59, rec, true, brianParty(), stock)
	if o.stock != 8 || o.live != len(brianParty())+1 || !o.statistics() || &tavernTalkFrames(art, o)[0] != &art.UnitFrames[8][0] {
		t.Fatalf("npc59 = %+v, want the type-8 stock unit", o)
	}
}

// TestHumanTermExcludesHeroes: a Hero party member of the record's sex and
// face does not answer a Human record; the Human bit is never set on a hero.
func TestHumanTermExcludesHeroes(t *testing.T) {
	rec := data.NPCFace{Face: 1, Tokens: data.NPCTokens(data.NPCTokenHuman) | data.NPCTokens(data.NPCTokenFace) |
		data.NPCTokens(data.NPCTokenNotFemale)}
	if o := resolveTavernTalk(90, rec, true, brianParty(), nil); o.live >= 0 {
		t.Fatalf("Human record = %+v, want no live hero", o)
	}
	hired := []mapload.PartyMember{{Name: "Guard", MercenaryType: 10, FigureDir: string(data.FigureDirManFighter), FigureFace: 1}}
	if o := resolveTavernTalk(90, rec, true, hired, nil); o.live != 0 {
		t.Fatalf("Human record over a hired human = %+v, want member 0", o)
	}
}

// TestTavernTalkDetailIsRenderedOncePerSelection: the talk panel is cached
// while its cell stays selected and rebuilt after another selection.
func TestTavernTalkDetailIsRenderedOncePerSelection(t *testing.T) {
	f := ownerTavernFront(t, brianParty())
	f.NPCFaces = map[int32]data.NPCFace{25: brianRecord}
	f.TownTavernArt = resolved(&ui.TownTavernArt{LeftStats: image.NewRGBA(image.Rect(0, 0, 160, 238))}, nil)
	s := f.townUI
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 13}, false)
	first := s.TownSurface().CandidatePixels
	if second := s.TownSurface().CandidatePixels; first == nil || second != first {
		t.Fatalf("talk panel = %p then %p, want one cached image", first, second)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	s.TownSurface()
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 13}, false)
	if third := s.TownSurface().CandidatePixels; third == nil || third == first {
		t.Fatalf("reselected talk panel = %p, want a new image after the selection changed", third)
	}
}
