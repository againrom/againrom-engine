package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

// TestReleaseTavernRosterOnAReloadedTownSave walks the lawful campaign to the
// mission-130 town, writes a native SAV, reloads it into a fresh front end and
// opens the tavern: mercenary cells in the stock walk (TAVERN-ORDER-015), then
// one talk cell per InnNPC element in array order (TOWN-468, SAV-1112), each
// on its object's sheet (TAVERN-TALKPIC-016), statistics only for a live
// object (TAVERN-TALKSTATS-017), and grouped prices (TOWN-469).
func TestReleaseTavernRosterOnAReloadedTownSave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Tavern Roster")
	f.Town.gold = 5_000_000
	for _, m := range reachabilityWalkStates() {
		screen.markWorldSelected(m)
		if _, accepted := f.Town.Won(m); !accepted {
			t.Fatalf("Town.Won(%d) was not accepted", m)
		}
		f.addChapterCompanions(f.Town.Chapter())
		if m == 120 {
			break
		}
	}
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	if _, err := save(false); err != nil {
		t.Fatalf("native town save: %v", err)
	}
	r := releaseFront(t)
	_, list, load := r.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := list()
	if len(entries) != 1 {
		t.Fatalf("save list = %+v", entries)
	}
	if _, town, err := load(entries[0].Name); err != nil || !town {
		t.Fatalf("reload = town %v err %v", town, err)
	}
	s := r.TownScreen().(*townScreen)
	s.atSquare()
	s.Choose(0)
	if s.room != roomTavern {
		t.Fatalf("Choose(0) opened room %v, want the tavern", s.room)
	}
	offers := r.Town.Offers(TownTavern)
	mercs := s.tavernMercenaries()
	if len(offers) == 0 || len(mercs) == 0 {
		t.Fatalf("chapter %d roster has %d talk and %d mercenary cells; want both", r.Town.Chapter(), len(offers), len(mercs))
	}
	walk := tavernStockWalk(r.Town.tavernStockIDs)
	v := s.TownSurface()
	if len(v.Cells) != len(mercs)+len(offers) || !v.Cells[0].Selected || v.RosterUnpainted {
		t.Fatalf("roster = %d cells, first selected %v", len(v.Cells), v.Cells[0].Selected)
	}
	next := 0
	for i, o := range mercs {
		for next < len(walk) && walk[next] != o.Type {
			next++
		}
		if next == len(walk) {
			t.Fatalf("mercenary cell %d type %d breaks the stock walk %v", i, o.Type, walk)
		}
		if want := ui.GroupDigits(int64(o.Price)); v.Cells[i].Price != want || v.Cells[i].TalkOnly {
			t.Fatalf("mercenary cell %d = %+v, want price %s", i, v.Cells[i], want)
		}
	}
	art := r.TownTavernArt.Value()
	for j, o := range offers {
		i := len(mercs) + j
		c := v.Cells[i]
		if !c.TalkOnly || c.Semantic != fmt.Sprintf("NPC %d", o.NPC) {
			t.Fatalf("cell %d = %+v, want the talk cell for npc%d", i, c, o.NPC)
		}
		obj := s.tavernTalkObject(o.NPC)
		frames := tavernTalkFrames(art, obj)
		if len(frames) == 0 || c.Picture != frames[0] {
			t.Fatalf("npc%d cell draws %v, want its object's sheet %+v", o.NPC, c.Picture != nil, obj)
		}
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: i}, false)
		got := s.TownSurface()
		if got.Candidate.HasSubject != obj.statistics() {
			t.Fatalf("npc%d statistics shown %v, want %v for %+v", o.NPC, got.Candidate.HasSubject, obj.statistics(), obj)
		}
		t.Logf("chapter %d position %d npc%d live %d hero %v mage %v sheet %d statistics %v",
			r.Town.Chapter(), i, o.NPC, obj.live, obj.hero, obj.mage, obj.sheet, obj.statistics())
	}
}

// kitTavernPrices is the mercenary cell prices the EN original shows on its
// first inn entry after loading each town save of AGAINROM_ROUTE_SAVES, from
// the first cell, the bottom-left one, onwards.
var kitTavernPrices = map[string][]string{
	"game9713.sav": {"3,080", "2,800", "2,080", "4,000", "2,920"},
	"game9714.sav": {"2,800", "3,080", "2,080", "4,000", "2,920"},
	"game9715.sav": {"42,000", "36,000", "46,200", "42,000", "43,800", "40,800", "55,200", "31,200", "60,000"},
	"game9716.sav": {"712,000", "584,000", "624,000", "544,000", "736,000", "616,000", "560,000", "560,000", "560,000",
		"480,000", "640,000", "416,000", "800,000"},
	"game9718.sav": {"770,000", "700,000", "700,000", "700,000", "600,000", "800,000", "520,000", "1,000,000", "890,000",
		"730,000", "780,000", "680,000", "920,000"},
	"game9721.sav": {"780,000", "680,000", "920,000", "770,000", "700,000", "700,000", "600,000", "800,000", "520,000",
		"1,000,000", "890,000", "730,000"},
	"game9724.sav": {"770,000", "700,000", "700,000", "600,000", "800,000", "520,000", "1,000,000", "890,000", "730,000",
		"780,000", "680,000", "920,000"},
}

// TestReleaseTavernOrderAfterAKitTownLoad loads each kit town save and opens
// the inn: the mercenary cells follow the stock ids the loaded allocator gives
// (TAVERN-ORDER-015, SAV-1372, SAV-1136), and entry selects the first cell.
func TestReleaseTavernOrderAfterAKitTownLoad(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := os.Getenv("AGAINROM_ROUTE_SAVES")
	if dir == "" {
		t.Skip("no AGAINROM_ROUTE_SAVES: the kit town saves are owner files")
	}
	checked := 0
	for name, want := range kitTavernPrices {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		f := releaseFront(t)
		if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
			t.Fatalf("%s: town %v err %v", name, town, err)
		}
		s := f.TownScreen().(*townScreen)
		s.atSquare()
		s.Choose(0)
		v := s.TownSurface()
		var got []string
		for _, c := range v.Cells {
			if !c.TalkOnly {
				got = append(got, c.Price)
			}
		}
		if !slices.Equal(got, want) || !v.Cells[0].Selected {
			t.Errorf("%s: cells %v first selected %v\nwant %v", name, got, v.Cells[0].Selected, want)
		}
		checked++
	}
	if checked == 0 {
		t.Skipf("%s holds none of the kit town saves", dir)
	}
}
