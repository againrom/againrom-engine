package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// loadedParty is the party the running mission was opened with, as its SAV
// restored it.
func loadedParty(g *FrontEnd) []mapload.PartyMember {
	m := g.live.mission
	return mapload.CloneParty(m.party[:min(m.entryCount, len(m.party))])
}

// homeParty is the part of a loaded party that comes home: hired men go back to
// the tavern pool at the mission's return, as at a win.
func homeParty(party []mapload.PartyMember) []mapload.PartyMember {
	var out []mapload.PartyMember
	for _, p := range party {
		if !p.Hired() {
			out = append(out, p)
		}
	}
	return out
}

func partyIDs(party []mapload.PartyMember) []string {
	var out []string
	for _, p := range party {
		out = append(out, p.ID)
	}
	return out
}

// requireTownParty checks that the town holds the members of want, each with
// every item it was loaded with, and that a town SAVE writes and a cold LOAD of
// it restores the same party.
func requireTownParty(t *testing.T, what string, g *FrontEnd, want []mapload.PartyMember, dir string, ids ...string) {
	t.Helper()
	if len(g.Carried) == 0 || !slices.Equal(partyIDs(g.Carried), partyIDs(want)) {
		t.Fatalf("%s: the town holds %v, the loaded mission held %v", what, partyIDs(g.Carried), partyIDs(want))
	}
	for i, p := range want {
		worn, pack := memberItemCodes(p)
		gotWorn, gotPack := memberItemCodes(g.Carried[i])
		if worn != gotWorn {
			t.Fatalf("%s: member %s wears %x, was loaded wearing %x", what, p.ID, gotWorn, worn)
		}
		for _, code := range pack {
			if !slices.Contains(gotPack, code) {
				t.Fatalf("%s: member %s lost pack item %x (now %x)", what, p.ID, code, gotPack)
			}
		}
	}
	raw, err := abandonSaveTown(g)
	if err != nil {
		t.Fatalf("%s: the town SAVE was refused: %v", what, err)
	}
	h := abandonReloadUnder(t, raw, dir, ids...)
	if !slices.Equal(partyKey(h.Carried), partyKey(g.Carried)) || h.Town.Gold() != g.Town.Gold() {
		t.Fatalf("%s: the cold LOAD holds %v gold %d, the town %v gold %d", what, partyKey(h.Carried), h.Town.Gold(), partyKey(g.Carried), g.Town.Gold())
	}
}

// abandonReloadUnder is a cold LOAD of a town SAV in a fresh front end under the
// mods ids of dir.
func abandonReloadUnder(t *testing.T, raw []byte, dir string, ids ...string) *FrontEnd {
	t.Helper()
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g, _, _ := abandonFront(t, dir, ids...)
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: d}, nil)
	if _, town, err := load("town.sav"); err != nil || !town {
		t.Fatalf("cold LOAD of the town SAV: town %v err %v", town, err)
	}
	return g
}

// A mission SAV taken inside mission 30 and loaded cold in a fresh front end
// carries the party the mission was opened with and no town party. Abandoning it
// brings that party home, the town SAVE writes, a cold LOAD restores it, the
// quick-spell bindings are the ones the mission was opened with, and the mission
// is entered again with that party and not the default hero.
func TestReleaseModAbandonOfALoadedMissionBringsTheSavedPartyHome(t *testing.T) {
	store, entry := abandonMissionSave(t, abandonModDir, "mission-abandon")
	g, a, screens := abandonLoadFront(t, store, OriginalStore{}, false, abandonModDir, "mission-abandon")
	abandonLoadMission(t, g, a, 0)
	if len(g.Carried) != 0 {
		t.Fatalf("a mission LOAD installed a town party of %d, the witness needs none", len(g.Carried))
	}
	want := homeParty(loadedParty(g))
	if len(want) != len(entry.party) {
		t.Fatalf("the loaded mission holds %d members, %d entered", len(want), len(entry.party))
	}
	gold := g.Town.Gold()
	bound := g.quickSpells
	g.quickSpells[0] ^= 0x55
	abandonThroughMenu(t, g, a, screens.Screens[0])
	if a.Screen() != ui.ScreenTown {
		t.Fatalf("abandoning left screen %s", a.Screen())
	}
	s := abandonWalkHome(t, g, a)
	if g.quickSpells != bound {
		t.Fatalf("the quick-spell bindings are %v, the mission was opened with %v", g.quickSpells, bound)
	}
	if g.Town.Gold() != gold || g.Town.Done(30) {
		t.Fatalf("the town holds gold %d (loaded %d), mission 30 done %v", g.Town.Gold(), gold, g.Town.Done(30))
	}
	requireTownParty(t, "abandoned loaded mission", g, want, abandonModDir, "mission-abandon")
	// The mission is taken again: it opens with the saved party.
	abandonEnter(t, g, a, s, 30)
	if got := loadedParty(g); !slices.Equal(partyIDs(homeParty(got)), partyIDs(want)) {
		t.Fatalf("the mission entered after abandoning opens with %v, want %v", partyIDs(got), partyIDs(want))
	}
}

// A restart from a loaded mission reopens it with the party its SAV carried, and
// the restarted mission can be left, bringing that party home.
func TestReleaseModRestartOfALoadedMissionKeepsTheSavedParty(t *testing.T) {
	dir := writeLeaveMod(t)
	store, _ := abandonMissionSave(t, dir, "mission-leave")
	g, a, screens := abandonLoadFront(t, store, OriginalStore{}, false, dir, "mission-leave")
	abandonLoadMission(t, g, a, 0)
	want := homeParty(loadedParty(g))
	first := g.live
	abandonThroughMenu(t, g, a, screens.Screens[1])
	if a.Screen() != ui.ScreenMap || g.live == first || g.live.mission.number != 30 {
		t.Fatalf("restart left screen %s, same driver %v", a.Screen(), g.live == first)
	}
	if got := loadedParty(g); !slices.Equal(partyIDs(homeParty(got)), partyIDs(want)) {
		t.Fatalf("the restarted mission opens with %v, want %v", partyIDs(got), partyIDs(want))
	}
	abandonThroughMenu(t, g, a, screens.Screens[0])
	abandonWalkHome(t, g, a)
	requireTownParty(t, "restarted then abandoned", g, want, dir, "mission-leave")
}

const abandonCorpus = "../../../gameversions/saves/2027-09-07"

// An original mission SAV of the corpus, loaded with unmarked-SAV acceptance
// under the mod, is abandoned and restarted: the party its SAV holds comes home,
// the town SAVE writes and a cold LOAD restores it.
func TestReleaseModAbandonOfAnOriginalMissionSAV(t *testing.T) {
	if _, err := os.Stat(abandonCorpus); err != nil {
		t.Skipf("no save corpus: %v", err)
	}
	loaded := 0
	for _, name := range []string{"game0005.sav", "game0011.sav", "game0028.sav", "game0030.sav"} {
		for _, restart := range []bool{false, true} {
			orig := OriginalStore{Dir: abandonCorpus}
			dir, ids := abandonModDir, "mission-abandon"
			if restart {
				dir, ids = writeLeaveMod(t), "mission-leave"
			}
			g, a, screens := abandonLoadFront(t, SaveStore{Dir: t.TempDir()}, orig, true, dir, ids)
			_, list, _ := g.SaveSeams(SaveStore{Dir: t.TempDir()}, orig, nil)
			row := slices.IndexFunc(list(), func(e ui.SaveEntry) bool { return e.Name == name })
			if row < 0 {
				t.Fatalf("%s is not in the corpus list", name)
			}
			keys := []string{"load"}
			for i := 0; i < row; i++ {
				keys = append(keys, "down")
			}
			keys = append(keys, "enter")
			for _, k := range keys {
				if err := a.HeadlessKey(k); err != nil {
					t.Fatalf("%s key %q: %v", name, k, err)
				}
			}
			if a.Screen() != ui.ScreenMap || g.live == nil || g.live.mission == nil {
				t.Logf("%s: not a mission this install loads (screen %s)", name, a.Screen())
				continue
			}
			if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenGameMenu {
				t.Fatalf("%s: Escape: %v on %s", name, err, a.Screen())
			}
			rows := len(a.HeadlessRows())
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if rows < 8 {
				t.Logf("%s: mission %d offers no row (%d rows)", name, g.live.mission.number, rows)
				continue
			}
			if len(g.Carried) != 0 {
				t.Fatalf("%s: the LOAD installed a town party of %d", name, len(g.Carried))
			}
			want := homeParty(loadedParty(g))
			if len(want) == 0 {
				t.Fatalf("%s: the SAV holds no party", name)
			}
			loaded++
			t.Logf("%s mission %d party %d restart %v", name, g.live.mission.number, len(want), restart)
			if restart {
				first := g.live
				abandonThroughMenu(t, g, a, screens.Screens[1])
				if a.Screen() != ui.ScreenMap || g.live == first {
					t.Fatalf("%s: restart left screen %s", name, a.Screen())
				}
				if got := loadedParty(g); !slices.Equal(partyIDs(homeParty(got)), partyIDs(want)) {
					t.Fatalf("%s: restarted with %v, want %v", name, partyIDs(got), partyIDs(want))
				}
			}
			abandonThroughMenu(t, g, a, screens.Screens[0])
			abandonWalkHome(t, g, a)
			requireTownParty(t, name, g, want, dir, ids)
		}
	}
	if loaded == 0 {
		t.Skip("no corpus mission SAV of this install offered the abandon row")
	}
}
