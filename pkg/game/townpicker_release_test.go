package game

import (
	"fmt"
	"image"
	"slices"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// TestReleaseTownPickersSkipAHiredSquadAndTheMissionCarriesIt is the EN and RU
// App-level witness for TOWN-138, PARTY-FLAG-003 and MERC-HIRE-003. A town save
// with two player characters is loaded through the Load window, a squad is
// hired with the tavern's own controls, and the tavern, school and shop pickers
// are pressed by pointer in both directions. Each one stands on player
// characters alone, numbers them without the squad, and visits every one of
// them. The squad's equipment is unchanged afterwards, and the mission the gates
// open still holds the whole squad as units.
func TestReleaseTownPickersSkipAHiredSquadAndTheMissionCarriesIt(t *testing.T) {
	src, _ := hireForOrderTest(t, "Picker Witness")
	dir := t.TempDir()
	save, _, _ := src.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("town save = %q, %v, want a SAV", name, err)
	}

	f := releaseFront(t)
	f.Options = OptionsStore{}
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	app := openLocalTownSAV(t, f, dir, name)
	t.Cleanup(app.StopAudio)
	app.Layout(640, 480)
	s := f.TownScreen().(*townScreen)

	click := func(p image.Point) {
		t.Helper()
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	settle := func() {
		t.Helper()
		for i := 0; i < 600 && s.schoolTrainingBusy(); i++ {
			now = now.Add(84 * time.Millisecond)
			if _, _, err := app.HeadlessFrame(); err != nil {
				t.Fatal(err)
			}
		}
		if s.schoolTrainingBusy() {
			t.Fatal("the school stayed busy after a picker step")
		}
	}
	enter := func(door string, want townRoom) {
		t.Helper()
		if err := app.HeadlessActivate(door); err != nil {
			t.Fatal(err)
		}
		for n := 0; s.room == roomTalk && n < 32; n++ {
			if err := app.HeadlessActivate("dialogue"); err != nil {
				t.Fatal(err)
			}
		}
		if s.room != want {
			t.Fatalf("%s opened room %d, want %d", door, s.room, want)
		}
	}
	leave := func(room string, press func()) {
		t.Helper()
		press()
		if s.room != roomSquare {
			t.Fatalf("leaving the %s ended in room %d, want the square", room, s.room)
		}
	}

	enter("TAVERN", roomTavern)
	offers := s.tavernMercenaries()
	if len(offers) == 0 {
		t.Fatalf("chapter %d offers no mercenary squad", f.Town.Chapter())
	}
	typ := offers[0].Type
	if err := app.HeadlessActivate(fmt.Sprintf("Mercenary %d", typ)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(f.Words.TavernHire); err != nil {
		t.Fatal(err)
	}
	if !f.Town.MercenaryHired(typ) {
		t.Fatalf("Hire left squad type %d not hired", typ)
	}
	players := playerCharacters(f)
	var squad []mapload.PartyMember
	for _, m := range f.Carried {
		if m.MercenaryType == uint8(typ) {
			squad = append(squad, m)
		}
	}
	if len(players) < 2 || len(squad) == 0 || len(players)+len(squad) != len(f.Carried) {
		t.Fatalf("party of %d holds %d player characters and %d hires of type %d", len(f.Carried), len(players), len(squad), typ)
	}
	equipment := func(m mapload.PartyMember) string {
		worn, pack := memberItemCodes(m)
		return fmt.Sprint(worn, pack)
	}
	wantSquad := make(map[string]string)
	for _, m := range squad {
		wantSquad[m.ID] = equipment(m)
	}

	walk := func(room string, press func(next bool), shown func() (string, int, int)) {
		t.Helper()
		visited := make(map[string]bool)
		for _, next := range []bool{true, false} {
			for i := 0; i <= len(players); i++ {
				before := s.shopMemberIndex()
				press(next)
				settle()
				at := s.shopMemberIndex()
				m := f.Carried[at]
				if m.Hired() {
					t.Fatalf("%s: the picker stands on the hired %q (party member %d)", room, m.Name, at)
				}
				if at == before {
					t.Fatalf("%s: a press left the picker on party member %d", room, at)
				}
				if who, position, count := shown(); who != m.Name || count != len(players) ||
					position != slices.Index(players, at) {
					t.Fatalf("%s: the panel shows %q as %d of %d, want %q as %d of %d",
						room, who, position, count, m.Name, slices.Index(players, at), len(players))
				}
				visited[m.ID] = true
			}
		}
		if len(visited) != len(players) {
			t.Fatalf("%s: the picker visited %d of %d player characters", room, len(visited), len(players))
		}
	}
	surfacePress := func(next bool) {
		t.Helper()
		pane, kind := ui.CharacterPanePrev, ui.TownSurfaceControlPrevious
		if next {
			pane, kind = ui.CharacterPaneNext, ui.TownSurfaceControlNext
		}
		r := ui.CharacterPaneCornerRect(ui.TownCharacterRegion, pane)
		at := r.Min.Add(r.Max).Div(2)
		if c, ok := ui.TownSurfaceControlAt(s.TownSurface(), at); !ok || c.Kind != kind {
			t.Fatalf("picker corner %v answers %+v, %v", at, c, ok)
		}
		click(at)
	}
	surfaceShown := func() (string, int, int) {
		h := s.TownSurface().Hero
		return h.Subject.Name, h.Member, h.MemberCount
	}

	walk("tavern", surfacePress, surfaceShown)
	leave("tavern", func() {
		if err := app.HeadlessActivate(f.Words.TavernExit); err != nil {
			t.Fatal(err)
		}
	})

	enter("SCHOOL", roomSchool)
	walk("school", surfacePress, surfaceShown)
	leave("school", func() {
		if err := app.HeadlessActivate(f.Words.SchoolExit); err != nil {
			t.Fatal(err)
		}
	})

	enter("SHOP", roomShop)
	walk("shop", func(next bool) {
		t.Helper()
		kind := "picker_prev"
		if next {
			kind = "picker_next"
		}
		x, y, err := app.HeadlessShopPoint(kind, 0)
		if err != nil {
			t.Fatal(err)
		}
		click(image.Pt(x, y))
	}, func() (string, int, int) {
		v := s.ShopScreen()
		return v.Character.Subject.Name, v.Member, v.MemberCount
	})
	leave("shop", func() {
		x, y, err := app.HeadlessShopPoint("button", 3)
		if err != nil {
			t.Fatal(err)
		}
		click(image.Pt(x, y))
	})

	for _, m := range f.Carried {
		if want, hired := wantSquad[m.ID]; hired && equipment(m) != want {
			t.Fatalf("hired %q carries %s after the rooms, want %s", m.Name, equipment(m), want)
		}
	}
	if got := s.mercenaryPartyCount(typ); got != len(squad) {
		t.Fatalf("the party holds %d of the squad after the rooms, want %d", got, len(squad))
	}

	mission := f.Town.Chapter()
	if avail := f.Town.Available(); len(avail) > 0 {
		mission = avail[0]
	} else {
		takeCampaignOffer(t, f, mission)
	}
	party := mapload.CloneParty(f.Carried)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(fmt.Sprintf("walk out to mission %d", mission)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4000 && app.Screen() == ui.ScreenTown; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap || f.live == nil {
		t.Fatalf("walking out to mission %d ended on %s", mission, app.Screen())
	}
	carried, ids := f.live.mission.party, f.live.mission.ids
	if len(carried) != len(party) || len(ids) != len(party) {
		t.Fatalf("mission %d holds %d members and %d units, want %d", mission, len(carried), len(ids), len(party))
	}
	inMission := 0
	for i, m := range carried {
		if m.ID != party[i].ID {
			t.Fatalf("mission member %d is %q, want %q", i, m.ID, party[i].ID)
		}
		if e, ok := f.live.world.Entity(ids[i]); !ok || !e.Alive() {
			t.Fatalf("mission member %d (%q) has no living unit", i, m.Name)
		}
		if m.MercenaryType == uint8(typ) {
			inMission++
		}
	}
	if inMission != len(squad) {
		t.Fatalf("mission %d holds %d units of the squad, want %d", mission, inMission, len(squad))
	}
	t.Logf("chapter %d: %d player characters, squad type %d of %d units; mission %d holds %d of them as living units",
		f.Town.Chapter(), len(players), typ, len(squad), mission, inMission)
}
