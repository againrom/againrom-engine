package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondDruidsCorpus is the owner's third original set: Kaarg after mission
// 30, mission 40 (game0025, and game0026 won), mission 50 won (game0029) and
// the original's Druids' town (game0030), read-only.
const secondDruidsCorpus = "../../../gameversions/saves/2026-10-31/rom2-gog-original-3"

func secondDruidsFile(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(secondDruidsCorpus, name))
	if err != nil {
		t.Skipf("no save corpus: %v", err)
	}
	return raw
}

// secondDruidsLoad LOADs name from the corpus through the game's own Load
// list into a second-game front.
func secondDruidsLoad(t *testing.T, name string) (*FrontEnd, *ui.App) {
	t.Helper()
	raw := secondDruidsFile(t, name)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := secondGameFront(t)
	app := f.App("second-druids")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	_, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	groundAppLoad(t, app, list, name)
	return f, app
}

func carriedCodes(m mapload.PartyMember) []uint16 {
	out := []uint16{}
	for _, it := range m.CarriedItems {
		out = append(out, it.Code)
	}
	return out
}

func sortedCodes(codes []uint16) []uint16 {
	out := slices.Clone(codes)
	slices.Sort(out)
	return slices.Compact(out)
}

// playerGroups is each group of raw's first Player as its members' identities.
func playerGroups(t *testing.T, raw []byte) [][]uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := [][]uint32{}
	for _, g := range doc.Objects[doc.Players[0]-1].Groups {
		refs, _ := savedObjectRefs(&g, "Actors")
		ids := []uint32{}
		for _, ref := range refs {
			id, _ := savedStructureValue(&doc.Objects[ref-1], "U148")
			ids = append(ids, id&0xffff)
		}
		out = append(out, ids)
	}
	return out
}

// checkDruidsPacks: the town file raw holds each member's pack as game0030
// does: identity 22's six pieces in the file's order, identity 23 none, and
// the hero the file's pieces, the bag's after his own (DIV-4305). Its Player
// holds game0030's groups, one of all three; ordered asks each group's
// member order too (DIV-4416).
func checkDruidsPacks(t *testing.T, name string, raw, original []byte, ordered bool) {
	t.Helper()
	got, file := humanPacks(t, raw), humanPacks(t, original)
	if !slices.Equal(got[22], file[22]) || len(got[23]) != 0 || !slices.Equal(sortedCodes(got[21]), sortedCodes(file[21])) {
		t.Fatalf("%s writes packs %x, want game0030's %x", name, got, file)
	}
	groups, want := playerGroups(t, raw), playerGroups(t, original)
	if !ordered {
		for _, g := range slices.Concat(groups, want) {
			slices.Sort(g)
		}
	}
	if !slices.EqualFunc(groups, want, slices.Equal) {
		t.Fatalf("%s: the Player holds groups %v, want game0030's %v", name, groups, want)
	}
}

// A LOAD of the original's Druids' town puts every carried piece where the
// file holds it: identity 22 keeps his six pieces in his own pack, the hero
// keeps the shield and the magic item the bag has no form for, and his other
// pieces are the campaign's bag. The SAVE after it writes each pack again.
func TestReleaseSecondTownDruidsLoadKeepsEachPack(t *testing.T) {
	raw := secondDruidsFile(t, "game0030.sav")
	file := humanPacks(t, raw)
	f := secondGameFront(t)
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("town LOAD", town, err)
	}
	if got := carriedIdentities(f.Carried, f.Table); !slices.Equal(sortedInts(got), []int{21, 22, 23}) {
		t.Fatalf("the LOAD carries identities %v, want 21, 22 and 23", got)
	}
	packs := map[string][]uint16{}
	for _, m := range f.Carried {
		packs[m.ID] = carriedCodes(m)
	}
	c := f.Town.second
	if !slices.Equal(packs["npc:22"], file[22]) || len(packs["npc:23"]) != 0 || !slices.Equal(packs["hero"], []uint16{0xb226, 0x0e4d}) {
		t.Fatalf("the LOAD's packs are %x, want identity 22's %x and the hero's shield and magic item", packs, file[22])
	}
	if c == nil || c.current != (partyPlace{placeTown, 3}) || len(c.party) != 1 ||
		!slices.Equal(carriedCodes(c.party[0]), []uint16{0x3142, 0xf8ab, 0x112a, 0x2133, 0x2133, 0x0103}) {
		t.Fatalf("the campaign is %+v, want town 3 and the hero's other pieces in its bag", c)
	}
	checkDruidsPacks(t, "the SAVE after the LOAD", exportTown(t, f), raw, true)
}

func sortedInts(v []int) []int {
	out := slices.Clone(v)
	slices.Sort(out)
	return out
}

// The departure of mission 50 from the owner's won mission file enters the
// Druids' town as the original's file holds it: town 3 current at stage 60,
// the same places and bank, the hero and identities 22 and 23 in one group,
// each member's pack.
func TestReleaseSecondTownDruidsAfterMissionFifty(t *testing.T) {
	want := secondGameFront(t)
	original := secondDruidsFile(t, "game0030.sav")
	if _, town, err := want.RestoreOriginal(original); err != nil || !town {
		t.Fatal("game0030 LOAD", town, err)
	}
	f, app := secondDruidsLoad(t, "game0029.sav")
	if f.live == nil || f.liveMission != 50 {
		t.Fatalf("the LOAD opened mission %d, want 50", f.liveMission)
	}
	for _, target := range []string{"victory", "town 3"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(target, err, app.HeadlessMessage())
		}
	}
	c, w := f.Town.second, want.Town.second
	if app.Screen() != ui.ScreenTown || c.current != (partyPlace{placeTown, 3}) || c.bank[768] != 60 {
		t.Fatalf("the departure stands at %v on %v, stage %d; want town 3 at stage 60", c.current, app.Screen(), c.bank[768])
	}
	if !slices.Equal(c.available, w.available) || c.bank != w.bank {
		t.Fatalf("the departure's places %v, want game0030's %v; banks equal %v", c.available, w.available, c.bank == w.bank)
	}
	if got := carriedIdentities(f.Carried, f.Table); !slices.Equal(sortedInts(got), []int{21, 22, 23}) {
		t.Fatalf("the departure carries identities %v, want 21, 22 and 23 once each", got)
	}
	checkDruidsPacks(t, "the departure's town SAVE", exportTown(t, f), original, false)
}

// Leaving the owner's mission 40 adds identity 23 through the party's add
// producer (R2-SESSION-301); a second leave adds nobody. game0026's Player
// holds Outcome 1, so its LOAD stands on the victory panel.
func TestReleaseSecondDinaJoinsAtMissionFortysLeave(t *testing.T) {
	for _, file := range []struct {
		name string
		won  bool
	}{{"game0025.sav", false}, {"game0026.sav", true}} {
		t.Run(file.name, func(t *testing.T) {
			f, app := secondDruidsLoad(t, file.name)
			if f.live == nil || f.liveMission != 40 {
				t.Fatalf("the LOAD opened mission %d, want 40", f.liveMission)
			}
			_, kind, up := f.LiveNotice()
			if won := f.live.world.Outcome() == sim.OutcomeWon && up && kind == ui.NoticeSuccess; won != file.won {
				t.Fatalf("the LOAD stands won %v, want %v", won, file.won)
			}
			if !file.won {
				secondContinuedWin(t, f, app)
			}
			if err := app.HeadlessActivate("victory"); err != nil {
				t.Fatal(err)
			}
			if f.live != nil || f.Town.second.bank[773] != 23 {
				t.Fatalf("the victory left mission %d with slot 773 %d", f.liveMission, f.Town.second.bank[773])
			}
			if got := carriedIdentities(f.Carried, f.Table); !slices.Equal(got, []int{21, 22, 23}) {
				t.Fatalf("the leave carries identities %v, want 21, 22 and 23", got)
			}
			f.CampaignSession.leftLocation(f.townInstall())
			if got := carriedIdentities(f.Carried, f.Table); len(got) != 3 {
				t.Fatalf("a second leave carries identities %v", got)
			}
		})
	}
}
