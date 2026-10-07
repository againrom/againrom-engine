package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

type figureTownRig struct {
	t       *testing.T
	f       *FrontEnd
	s       *townScreen
	chapter int
	hero    figureHero
}

// figureTownHires are the squad types the tavern's controls hire in the town
// audit: a siege engine, a mage, a fighter, a rider and a heavy fighter.
var figureTownHires = []int{1, 3, 6, 8, 10, 13}

// openFigureTown builds the town of one chapter through the ordinary routes: a
// chosen hero, the chapter's companion grants, Brian from chapter 40 on and the
// squads above hired with the tavern's own toggle.
func openFigureTown(t *testing.T, chapter int, hero figureHero) *figureTownRig {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	result := ui.ChargenResult{Name: hero.name, Choices: []int{hero.sex, hero.class, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	town := NewTown(f.Campaign.Value())
	var earlier []int
	for number := range f.Campaign.Value().Chapters {
		if number < chapter {
			earlier = append(earlier, number)
		}
	}
	slices.Sort(earlier)
	for _, number := range earlier {
		town.Won(number)
	}
	town.gold = 50_000_000
	f.Town = town
	f.Carried = generated
	f.arriveInTown()
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("no town screen")
	}
	f.addChapterCompanions(f.Town.Chapter())
	if chapter >= 40 {
		brian, ok := mapload.CampaignNPCMember(f.Table, 25, 0, f.Carried)
		if !ok {
			t.Fatal("no Brian to carry")
		}
		f.Carried = append(f.Carried, brian)
	}
	s.room = roomTavern
	s.composeShopFaces()
	hired := 0
	for _, typ := range figureTownHires {
		f.Town.mercEnabled[typ] = true
		if _, ok := s.toggleMercenary(typ); ok {
			hired++
		}
	}
	if hired != len(figureTownHires) {
		t.Fatalf("the tavern hired %d of %d squads", hired, len(figureTownHires))
	}
	s.composeShopFaces()
	return &figureTownRig{t: t, f: f, s: s, chapter: chapter, hero: hero}
}

// coldLoad SAVEs the town with the ordinary producer and opens the bytes in a
// cold front end.
func (r *figureTownRig) coldLoad() *figureTownRig {
	t := r.t
	t.Helper()
	raw := cityProjectionSave(t, r.f)
	next := &FrontEnd{InstallResources: r.f.InstallResources}
	if _, town, err := next.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	s, ok := next.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("no town screen after LOAD")
	}
	s.room = roomTavern
	s.composeShopFaces()
	return &figureTownRig{t: t, f: next, s: s, chapter: r.chapter, hero: r.hero}
}

// auditTownParty checks the party the town holds: the dolls the town draws, the
// pack a hired unit does not have, and the pickers the shop, the school and the
// tavern step over (TOWN-138, MERC-TYPE-001, HERO-FIGURE-144).
func auditTownParty(r *figureTownRig) []string {
	t, f, s := r.t, r.f, r.s
	t.Helper()
	src := f.Archives.Containers
	var bad []string
	report := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	party := f.Carried
	if len(s.shopFigures) != len(party) {
		return []string{fmt.Sprintf("the town composed %d dolls for %d members", len(s.shopFigures), len(party))}
	}
	heroes, hires, riders := 0, 0, 0
	for i, p := range party {
		who := fmt.Sprintf("%s (party %d)", p.Name, i)
		got := s.shopFigures[i]
		eq := mapload.EquipmentFromParty(p)
		options := []data.Equipment{eq}
		if code, ok := s.shopWeaponFallbackCode(i); ok {
			fallback := eq
			fallback.SetCode(1, data.ItemCode(code))
			options = append(options, fallback)
		}
		dir, face := memberFigure(p)
		matches := func(fig figureID, cape bool) bool {
			for _, worn := range options {
				want := oracleFigure(src, fig, worn)
				if want != nil && cape {
					want = oracleCaped(t, src, dir, want)
				}
				if want != nil && got != nil && sameBytes(got.Pix, want.Pix) {
					return true
				}
			}
			return false
		}
		if !p.Hired() {
			heroes++
			cape := !dir.Mage()
			if !matches(figureID{Dir: dir, Face: face}, cape) {
				report("%s: the town doll is not the %s figure (warrior cape %t)", who, dir, cape)
			}
			continue
		}
		hires++
		if pack := mapload.MemberCarriedItems(p, nil); len(pack) != 0 {
			report("%s: a hired unit holds %d items in a pack", who, len(pack))
		}
		if !data.ComposesFigure(p.Class) {
			continue
		}
		horse := data.FigureHasHorse(p.Class)
		if horse {
			riders++
		}
		if !matches(figureID{Dir: dir, Face: face, Horse: horse}, false) {
			report("%s: the hired unit's town doll is not its own %s figure (horse %t, no cape)", who, dir, horse)
		}
	}
	if heroes < 2 || hires < len(figureTownHires) {
		report("the town audit saw %d heroes and %d hired units", heroes, hires)
	}
	if riders == 0 {
		report("the town party holds no hired rider")
	}
	// The shop, the school and the tavern picker step over the player
	// characters alone.
	picker := townPickerMembers(party)
	for _, i := range picker {
		if party[i].Hired() {
			report("the picker admits the hired %s", party[i].Name)
		}
	}
	if len(picker) != heroes {
		report("the picker admits %d members, the party holds %d player characters", len(picker), heroes)
	}
	for selected := range party {
		if at := townPickerIndex(party, selected); party[at].Hired() {
			report("selection %d resolves to the hired %s", selected, party[at].Name)
		}
	}
	return bad
}

// auditTavernSpeakers opens the conversation of every talk cell the tavern
// lists and checks the picture of each named speaker. A speaker a stock
// mercenary unit answers for is that unit in its own set, a carried companion
// is his town doll, and a record nobody answers for is the bare sheet, on the
// warrior backdrop when it states Hero (TAVERN-TALKPIC-016, DLG-SPEAKER-022).
// stock collects the stock-unit speakers found.
func auditTavernSpeakers(r *figureTownRig, stock map[int]bool) []string {
	t, f, s := r.t, r.f, r.s
	t.Helper()
	src := f.Archives.Containers
	var bad []string
	report := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	s.room = roomTavern
	s.composeShopFaces()
	checked := 0
	for i, cell := range s.tavernSurfaceCells() {
		if !strings.HasPrefix(cell.Semantic, "NPC ") {
			continue
		}
		s.room = roomTavern
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: i}, true)
		if s.room != roomTalk {
			continue
		}
		payload, audience := s.townPayload(), HeroAudience(f.Carried)
		for part := 1; ; part++ {
			if part > 1 {
				s.AdvanceTownDialogue()
			}
			if _, ok := expectedInstalledDialoguePart(t, payload, part, audience); !ok {
				break
			}
			npc, named := EventPartSpeaker(payload, part, audience)
			if !named {
				continue
			}
			rec, known := f.NPCFaces[int32(npc)]
			// The dialogue's cast holds the stock units alone, so a hired
			// squad in the party never answers a speaker.
			object := resolveTavernTalk(npc, rec, known, nil, s.tavernStockActors())
			pic, _, found := s.speakerFace(npc)
			who := fmt.Sprintf("chapter %d %s part %d speaker npc%d", r.chapter, cell.Semantic, part, npc)
			companion := -1
			for k, p := range f.Carried {
				if p.CompanionNPC == npc && !p.Hired() {
					companion = k
				}
			}
			switch {
			case object.stock > 0:
				checked++
				stock[npc] = true
				if !found || !imagesEqual(pic, expectedInstalledMercenaryTalkPicture(t, f, object.stock)) {
					report("%s: not stock unit %d drawn in its own set", who, object.stock)
				}
				if sheet := oracleFigure(src, figureID{Dir: rec.Dir, Face: rec.Face}, data.Equipment{}); found && sheet != nil && sameBytes(pic.Pix, sheet.Pix) {
					report("%s: the stock unit is drawn as the bare sheet", who)
				}
			case companion >= 0 && companion < len(s.shopFigures):
				checked++
				if !found || !imagesEqual(pic, s.shopFigures[companion]) {
					report("%s: the carried companion is not drawn as his town doll", who)
				}
			case object.live < 0 && rec.Kind == data.NPCFigure && found:
				checked++
				hero := rec.Tokens.Has(data.NPCTokenHero)
				want := oracleFigure(src, figureID{Dir: rec.Dir, Face: rec.Face, Hero: hero}, data.Equipment{})
				if hero && !rec.Dir.Mage() {
					want = oracleCaped(t, src, rec.Dir, want)
				}
				if want == nil || !sameBytes(pic.Pix, want.Pix) {
					report("%s: an unanswered %s record (Hero %t) is not the bare sheet on its backdrop", who, rec.Dir, hero)
				}
			}
		}
	}
	if checked == 0 {
		report("chapter %d's tavern named no speaker to audit", r.chapter)
	}
	return bad
}

func TestReleaseFigureRegressionTown(t *testing.T) {
	stock := map[int]bool{}
	ran := 0
	for n, chapter := range []int{40, 70, 100, 140} {
		hero := figureHeroes[(n+1)%len(figureHeroes)]
		t.Run(fmt.Sprintf("chapter %d %s", chapter, hero.name), func(t *testing.T) {
			fresh := openFigureTown(t, chapter, hero)
			ran++
			requireClean(t, "fresh town party", auditTownParty(fresh))
			requireClean(t, "fresh tavern speakers", auditTavernSpeakers(fresh, stock))
			cold := fresh.coldLoad()
			requireClean(t, "loaded town party", auditTownParty(cold))
			requireClean(t, "loaded tavern speakers", auditTavernSpeakers(cold, stock))
		})
	}
	if ran > 0 && (!stock[90] || !stock[59]) {
		t.Errorf("the tavern walk found stock-unit speakers %v, want npc90 (chapter 40) and npc59 (chapter 70)", stock)
	}
}
