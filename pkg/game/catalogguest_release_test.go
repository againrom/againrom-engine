package game

import (
	"fmt"
	"os"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

func TestReleaseCatalogCharactersCanBeGuests(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := guestMods(t, guestModID)
	entries, err := mod.Resolve(dir, []string{guestModID})
	if err != nil {
		t.Fatal(err)
	}
	base := BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))
	res, err := modrt.Load(entries, base, nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	original := res.Missions.Missions[0]
	for _, key := range []string{"npc1", "npc31", "npc100", "npc22"} {
		t.Run(key, func(t *testing.T) {
			f := releaseFront(t)
			if err := f.SetMods(res.Rules, res.Set, false); err != nil {
				t.Fatal(err)
			}
			src := fmt.Sprintf("[[mission]]\nkey = \"any_character\"\nnumber = 25\nmap = \"mission/{lang}/map.alm\"\nafter = 20\nparty = [{ character = %q }]\n", key)
			d, err := mod.ParseMissions(guestModID, mod.MissionsFile, []byte(src), func(string) (string, bool) { return "", false }, modrt.LanguageFor(base))
			if err != nil {
				t.Fatal(err)
			}
			d.Missions[0].MapBytes, d.Missions[0].TextFiles = original.MapBytes, original.TextFiles
			if _, err := f.SetModMissions(d); err != nil {
				t.Fatalf("catalog member refused: %v", err)
			}
			ref := f.ChargenParty(ui.ChargenResult{Name: "Guest", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
			party, err := f.guestParty(d.Missions[0], 30, ref, nil)
			if err != nil || len(party) != 1 {
				t.Fatalf("party = %+v, %v", party, err)
			}
			if party[0].ID != "npc:"+key[3:] || party[0].Name == "" || !party[0].StartingHero {
				t.Fatalf("identity = %+v", party[0])
			}
			app, next := guestRoute(t, f, []int{0, 0, 0})
			if next != guestMissionNo {
				t.Fatalf("campaign win opened %d", next)
			}
			if err := app.OpenMission(f.MissionOpener(guestMissionNo)); err != nil {
				t.Fatalf("open catalog party: %v", err)
			}
			saved := guestMissionSave(t, f)
			cold := releaseFront(t)
			if err := cold.SetMods(res.Rules, res.Set, false); err != nil {
				t.Fatal(err)
			}
			if _, err := cold.SetModMissions(d); err != nil {
				t.Fatal(err)
			}
			open, town, err := cold.RestoreOriginal(saved)
			if err != nil || town {
				t.Fatalf("cold LOAD: town=%v, %v", town, err)
			}
			if err := cold.App("catalog guest load").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			loaded := cold.LiveParty()
			if len(loaded) != 1 || loaded[0].ID != party[0].ID || loaded[0].Hero != party[0].Hero ||
				loaded[0].Name != party[0].Name || loaded[0].FigureDir != party[0].FigureDir || loaded[0].FigureFace != party[0].FigureFace {
				t.Fatalf("cold party = %+v; declared %+v", loaded, party)
			}
			if _, _, err := cold.LiveCompleteCampaign(); err != nil || cold.Town.Chapter() != 30 {
				t.Fatalf("next action: chapter=%d, %v", cold.Town.Chapter(), err)
			}
		})
	}
}

func TestReleaseCatalogGuestMemberCoverage(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	ref := f.ChargenParty(ui.ChargenResult{Name: "Guest", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	var refused []string
	built := 0
	for _, c := range f.Table.NPCCatalog.Records() {
		key := mod.InstallCharacterKey(c.ID)
		m := mod.Mission{Party: []mod.GuestMember{{Character: key}}}
		party, err := f.guestParty(m, 30, ref, nil)
		if err != nil {
			refused = append(refused, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		if len(party) != 1 || party[0].CompanionNPC != int(c.ID) {
			t.Fatalf("%s built %+v", key, party)
		}
		if old, ok := mapload.CampaignNPCMember(f.Table, c.ID, 30, ref); ok && old.Hero != party[0].Hero {
			t.Fatalf("%s changed the existing row's statistics", key)
		}
		built++
	}
	t.Logf("chapter=30 reference=male fighter catalog=%d built=%d refused=%v", len(f.Table.NPCCatalog.Records()), built, refused)
	if len(f.Table.NPCCatalog.Records()) != 105 || len(refused) != 0 {
		t.Fatalf("catalog coverage: built=%d refused=%v", built, refused)
	}
}
