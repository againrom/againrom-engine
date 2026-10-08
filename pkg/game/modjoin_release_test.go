package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

const modJoinID = "reniesta-joins-on-talk"

// modJoinFront is a front end on the lawful install running under the example
// mod, or under no mod, started through the launcher's own steps.
func modJoinFront(t *testing.T, withMod bool) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if !withMod {
		return f
	}
	entries, err := mod.Resolve(modItemsDir, []string{modJoinID})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if err := f.SetModItems(res.Items); err != nil {
		t.Fatal(err)
	}
	if err := f.SetModCompanions(res.Companions); err != nil {
		t.Fatal(err)
	}
	return f
}

// modJoinTown is the player's route to the chapter-30 town: chargen, mission
// 10 and mission 20, each finished through the production completion path.
func modJoinTown(t *testing.T, withMod bool) *FrontEnd {
	t.Helper()
	f := modJoinFront(t, withMod)
	party := f.ChargenParty(ui.ChargenResult{Name: "Mod join", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("mod join")
	for _, n := range []int{10, 20} {
		if n == 10 {
			if err := app.OpenMission(f.MissionOpenerWith(n, party)); err != nil {
				t.Fatalf("open mission %d: %v", n, err)
			}
		} else if err := app.OpenMission(f.MissionOpenerWith(n, f.NextParty())); err != nil {
			t.Fatalf("open mission %d: %v", n, err)
		}
		if _, _, err := f.LiveCompleteCampaign(); err != nil {
			t.Fatalf("finish mission %d: %v", n, err)
		}
	}
	return f
}

func modJoinCold(t *testing.T, raw []byte, withMod bool) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := modJoinFront(t, withMod)
	app, s := releaseTavernApp(t, f, store)
	return f, app, s
}

func modJoinSavedAddHero(t *testing.T, raw []byte) []uint16 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return doc.Campaign.Base.Arrays[0]
}

func modJoinRoster(f *FrontEnd) (out []string) {
	for _, p := range f.Carried {
		out = append(out, fmt.Sprintf("%s/%d/%s", p.ID, p.CompanionNPC, p.Name))
	}
	return out
}

func modJoinHas(f *FrontEnd) bool {
	for _, p := range f.Carried {
		if p.CompanionNPC == townGrantedCompanion {
			return true
		}
	}
	return false
}

// modJoinPickerNames steps the town's character picker once round and returns
// the names it shows.
func modJoinPickerNames(t *testing.T, s *townScreen) []string {
	t.Helper()
	var names []string
	roster := townPickerMembers(s.shopParty())
	for range roster {
		names = append(names, s.shopParty()[s.shopMemberIndex()].Name)
		s.shopStepMember(1)
	}
	return names
}

// modJoinInShop opens the merchant, closes the offer's conversation that opens
// with it and leaves the room on the shop.
func modJoinInShop(t *testing.T, app *ui.App, s *townScreen) {
	t.Helper()
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	for n := 0; s.room == roomTalk && n < 64; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomShop {
		t.Fatalf("the shop opened room %v", s.room)
	}
}

// modJoinToSquare leaves whatever room is open for the town square.
func modJoinToSquare(t *testing.T, app *ui.App, s *townScreen) {
	t.Helper()
	for n := 0; s.room != roomSquare && n < 8; n++ {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomSquare {
		t.Fatalf("could not leave to the square: room %v", s.room)
	}
}

func modJoinHearCompanion(t *testing.T, app *ui.App, s *townScreen, shot string) {
	t.Helper()
	if err := app.HeadlessActivate("NPC 22"); err != nil {
		t.Fatalf("press on the companion's tavern cell: %v", err)
	}
	if s.room != roomTalk {
		t.Fatalf("the tavern cell opened room %v, want the conversation", s.room)
	}
	if shot != "" {
		writeModShot(t, shot, modFrame(t, app))
	}
	for n := 0; s.room == roomTalk && n < 64; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomTavern {
		t.Fatalf("the conversation ended in room %v, want the tavern", s.room)
	}
}

// TestReleaseModJoinWaitsForTheTavernTalk proves, on the lawful install, that
// under the example mod the town's companion is absent from the party, the
// picker and the saved roster until her tavern conversation opens, that both
// states SAVE and cold LOAD, and that without the mod she joins on arrival.
func TestReleaseModJoinWaitsForTheTavernTalk(t *testing.T) {
	// Control: no mod, she is in the party on arrival and the SAV holds no grant.
	plain := modJoinTown(t, false)
	if !modJoinHas(plain) || len(modJoinSavedAddHero(t, currentTownSave(t, plain))) != 0 {
		t.Fatalf("unmodded arrival roster %v", modJoinRoster(plain))
	}
	if cold, _, _ := modJoinCold(t, currentTownSave(t, plain), false); !modJoinHas(cold) {
		t.Fatalf("unmodded cold LOAD lost her: %v", modJoinRoster(cold))
	}

	f := modJoinTown(t, true)
	if f.Town.Chapter() != 30 || modJoinHas(f) || len(f.Carried) != 1 {
		t.Fatalf("modded arrival: chapter %d roster %v", f.Town.Chapter(), modJoinRoster(f))
	}
	before := currentTownSave(t, f)
	if got := modJoinSavedAddHero(t, before); len(got) != 1 || got[0] != townGrantedCompanion {
		t.Fatalf("the SAV before the talk holds AddHero %v, want the pending grant", got)
	}
	if _, err := modItemReload(t, before, true); err == nil {
		t.Fatal("a SAV written under the mod loaded without it")
	}

	// Pending state: cold LOAD keeps her absent and the SAV after it is the same.
	g, app, s := modJoinCold(t, before, true)
	if modJoinHas(g) || len(g.Carried) != 1 {
		t.Fatalf("cold LOAD before the talk: roster %v", modJoinRoster(g))
	}
	if names := modJoinPickerNames(t, s); len(names) != 1 {
		t.Fatalf("the picker before the talk shows %v", names)
	}
	if act := s.shopStepMember(1); act.Msg != "" || act.Open != nil || s.shopMemberIndex() != 0 {
		t.Fatalf("character switch before the talk: %+v, member %d", act, s.shopMemberIndex())
	}
	if containsMission(g.Town.Available(), 30) {
		t.Fatal("mission 30 is at the gates before its tavern conversation")
	}
	writeModShot(t, "join-tavern-before-talk", modFrame(t, app))
	modJoinToSquare(t, app, s)
	modJoinInShop(t, app, s)
	if got := s.shopParty(); len(got) != 1 {
		t.Fatalf("shop party before the talk: %v", modJoinRoster(g))
	}
	writeModShot(t, "join-shop-before-talk", modFrame(t, app))
	modJoinToSquare(t, app, s)
	if got := modJoinSavedAddHero(t, currentTownSave(t, g)); len(got) != 1 {
		t.Fatalf("re-SAVE of the pending state holds AddHero %v", got)
	}

	// The talk: she joins.
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	modJoinHearCompanion(t, app, s, "join-tavern-talk")
	if !modJoinHas(g) || len(g.Carried) != 2 || g.Carried[1].Name == "" {
		t.Fatalf("after the talk: roster %v", modJoinRoster(g))
	}
	writeModShot(t, "join-tavern-after-talk", modFrame(t, app))
	modJoinToSquare(t, app, s)
	if !containsMission(g.Town.Available(), 30) {
		t.Fatal("mission 30 is not at the gates after its tavern conversation")
	}
	modJoinInShop(t, app, s)
	if names := modJoinPickerNames(t, s); len(names) != 2 {
		t.Fatalf("the picker after the talk shows %v", names)
	}
	s.shopStepMember(1)
	writeModShot(t, "join-shop-after-talk", modFrame(t, app))
	modJoinToSquare(t, app, s)

	// Joined state: SAVE, cold LOAD, compare member by member.
	after := currentTownSave(t, g)
	if got := modJoinSavedAddHero(t, after); len(got) != 0 {
		t.Fatalf("the SAV after the talk still holds AddHero %v", got)
	}
	h, _, hs := modJoinCold(t, after, true)
	if len(h.Carried) != len(g.Carried) {
		t.Fatalf("cold LOAD after the talk: roster %v, saved from %v", modJoinRoster(h), modJoinRoster(g))
	}
	for i := range g.Carried {
		if diff := roundTripMemberDiff(g.Carried[i], h.Carried[i]); len(diff) != 0 {
			t.Fatalf("member %d (%s) changed across SAVE and LOAD: %v", i, g.Carried[i].Name, diff)
		}
		if g.Carried[i].ID != h.Carried[i].ID {
			t.Fatalf("member %d identity %q became %q", i, g.Carried[i].ID, h.Carried[i].ID)
		}
	}
	if names := modJoinPickerNames(t, hs); len(names) != 2 {
		t.Fatalf("the picker after the cold LOAD shows %v", names)
	}
	// Hearing her again joins nobody twice.
	happ, hs2 := func() (*ui.App, *townScreen) { _, a, s := modJoinCold(t, after, true); return a, s }()
	modJoinHearCompanion(t, happ, hs2, "")
	if n := len(hs2.sess.Carried); n != 2 {
		t.Fatalf("a second hearing left %d members", n)
	}
}
