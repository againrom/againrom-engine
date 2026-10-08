package game

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// cityGroupsColdApp is a cold LOAD of a town SAV through the main menu of a
// fresh front end.
func cityGroupsColdApp(t *testing.T, raw []byte) (*FrontEnd, *ui.App, SaveStore) {
	t.Helper()
	g := releaseFront(t)
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.WriteOriginal("", raw); err != nil {
		t.Fatal(err)
	}
	app := g.App("city groups cold LOAD")
	g.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("town LOAD through the main menu", err, app.Screen())
	}
	return g, app, store
}

// cityGroupsToggle presses the tavern's squad button for a type: through the
// Hire button when the tavern lists the type, otherwise through the toggle
// behind it. The same press hires a squad that is not hired and returns one
// that is.
func cityGroupsToggle(t *testing.T, f *FrontEnd, typ int) {
	t.Helper()
	f.Town.mercEnabled[typ] = true
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	was := f.Town.MercenaryHired(typ)
	listed := false
	for _, offer := range s.tavernMercenaries() {
		listed = listed || offer.Type == typ
	}
	if listed {
		s.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: typ}
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonHire}, false)
	} else if msg, ok := s.toggleMercenary(typ); !ok {
		t.Fatalf("squad %d refused: %s", typ, msg)
	}
	if f.Town.MercenaryHired(typ) == was {
		t.Fatalf("squad %d press did not change its hire (was hired %v, listed %v)", typ, was, listed)
	}
}

// cityGroupKind names an actor of a written document by what the document
// holds: its class, its Name and its hire type.
func cityGroupKind(data sav.CityData, actor uint16) string {
	object := data.Objects[actor-1]
	hire := uint32(0)
	if object.Unit != nil && len(object.Unit.Scalar2) == 55 {
		hire = binary.LittleEndian.Uint32(object.Unit.Scalar2[47:51])
	}
	return fmt.Sprintf("%s/%s/%d", object.Class, object.Unit.Name, hire)
}

// cityGroupKinds is the document's groups, each as the kinds of its actors in
// order.
func cityGroupKinds(t *testing.T, raw []byte) [][]string {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	city, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	data := city.Data()
	player := data.Objects[data.Players[0]-1].Player
	var out [][]string
	for _, g := range player.Groups {
		var kinds []string
		for _, a := range g.Actors {
			kinds = append(kinds, cityGroupKind(data, a))
		}
		out = append(out, kinds)
	}
	return out
}

// cityGroupsWant is the groups the party's current membership calls for,
// computed from the party alone: one group per unhired member in party order
// (the town holds its Player's members in groups of one, as an original town
// can), then one group per hired squad in hire order. A member's kind is what its written
// actor holds: an unhired member its Name, a siege hire a Unit with no Name and
// its hire type, any other hire a Human with no Name and its hire type.
func cityGroupsWant(t *testing.T, raw []byte, party []mapload.PartyMember, hired []int) [][]string {
	t.Helper()
	var want [][]string
	for _, member := range party {
		if !member.Hired() {
			want = append(want, []string{fmt.Sprintf("Human/%s/0", member.Name)})
		}
	}
	for _, typ := range hired {
		var squad []string
		for _, member := range party {
			switch {
			case int(member.MercenaryType) != typ:
			case typ <= 2:
				squad = append(squad, fmt.Sprintf("Unit//%d", typ))
			default:
				squad = append(squad, fmt.Sprintf("Human//%d", typ))
			}
		}
		want = append(want, squad)
	}
	got := cityGroupKinds(t, raw)
	if len(got) != len(want) {
		t.Fatalf("written groups %v, want %v", got, want)
	}
	for g := range want {
		if !slices.Equal(got[g], want[g]) {
			t.Fatalf("written groups %v, want %v", got, want)
		}
	}
	return got
}

func cityGroupsEqual(a, b []cityLiveGroup) bool {
	return slices.EqualFunc(a, b, func(x, y cityLiveGroup) bool {
		return slices.Equal(x.Members, y.Members) && slices.Equal(x.Payload.Words20, y.Payload.Words20) &&
			slices.Equal(x.Payload.Raw80, y.Payload.Raw80) && slices.Equal(x.Payload.Words3c, y.Payload.Words3c) &&
			x.Payload.F1c == y.Payload.F1c && x.Payload.F40 == y.Payload.F40 && x.Payload.F44 == y.Payload.F44
	})
}

// Hire, depart, mission and return repeat three times on one campaign with
// every SAVE and LOAD through App input. Human and siege squads are hired in
// either order, one squad is returned before the visit ends, and a companion
// joins after the first load. Every city SAVE writes the groups of the live
// membership, the cold LOAD holds the same membership, and the next mission
// continues on the live World's hashes.
func TestReleaseCityGroupsRepeatedHireCyclesF2SAV(t *testing.T) {
	plans := [][]int{{14, 1, 14, 6}, {14, 6, 10, 6, 1}, {6, 2, 14}}
	hired := [][]int{{1, 6}, {14, 10, 1}, {6, 2, 14}}
	lineage := releaseFront(t)
	reachabilityWalkArrive(t, lineage, "Group Cycles")
	for cycle, plan := range plans {
		lineage.Town.gold = 5_000_000
		seed := currentTownSave(t, lineage)
		cityGroupsWant(t, seed, lineage.Carried, nil)
		live, app, store := cityGroupsColdApp(t, seed)
		if cycle == 0 {
			if len(live.Carried) != len(lineage.Carried)+1 {
				t.Fatalf("the companion did not join after the first load: %d members from %d", len(live.Carried), len(lineage.Carried))
			}
			// The Player of an original town can hold its members in several
			// groups. Split the loaded town's one group in two, each with its
			// own field values, as that state.
			groups, _ := reconcileCityGroups(live.Town.cityGroups, live.Carried)
			if len(groups) != 1 || len(groups[0].Members) != 2 {
				t.Fatalf("the loaded town holds groups %v, want one group of two", liveGroupIDs(groups))
			}
			second := cityLiveGroup{Payload: cityConstructedGroup(), Members: groups[0].Members[1:]}
			groups[0].Members = groups[0].Members[:1]
			groups[0].Payload.F1c, groups[0].Payload.Raw80[0], second.Payload.Raw80[0] = 3, 1, 2
			live.Town.cityGroups = []cityLiveGroup{groups[0], second}
		}
		for _, typ := range plan {
			cityGroupsToggle(t, live, typ)
		}
		raw := cityRosterF2Save(t, app, store, fmt.Sprintf("groups-%d", cycle))
		saved := cityGroupsWant(t, raw, live.Carried, hired[cycle])
		t.Logf("cycle %d chapter %d: %d members, SAV groups %q", cycle, live.Town.Chapter(), len(live.Carried), saved)

		cold, coldApp, coldStore := cityGroupsColdApp(t, raw)
		if len(cold.Carried) != len(live.Carried) || !cityGroupsEqual(live.Town.cityGroups, cold.Town.cityGroups) {
			t.Fatalf("cycle %d: cold LOAD membership %v, live %v", cycle, liveGroupIDs(cold.Town.cityGroups), liveGroupIDs(live.Town.cityGroups))
		}
		cityRosterSame(t, live, cold)
		if first := cold.Town.cityGroups[0].Payload; first.F1c != 3 || first.Raw80[0] != 1 || cold.Town.cityGroups[1].Payload.Raw80[0] != 2 {
			t.Fatalf("cycle %d: the groups lost their own field values", cycle)
		}
		// Loss control: a town that does not write its live membership folds
		// the party into one group and the check above fails.
		kept := cold.Town.cityGroups
		cold.Town.cityGroups = nil
		if broken := cityGroupKinds(t, currentTownSave(t, cold)); slices.EqualFunc(broken, saved, slices.Equal[[]string]) {
			t.Fatalf("cycle %d: the groups do not depend on live membership", cycle)
		}
		cold.Town.cityGroups = kept
		again := cityRosterF2Save(t, coldApp, coldStore, fmt.Sprintf("groups-%d-again", cycle))
		if got := cityGroupsWant(t, again, cold.Carried, hired[cycle]); !slices.EqualFunc(got, saved, slices.Equal[[]string]) {
			t.Fatalf("cycle %d: the cold town's SAVE moved members between groups: %v, then %v", cycle, saved, got)
		}

		var worlds []*sim.World
		for _, target := range []*FrontEnd{live, cold} {
			mission := target.App("city groups mission")
			if err := mission.OpenMission(target.MissionOpener(target.Town.Chapter())); err != nil {
				t.Fatalf("cycle %d mission entry: %v", cycle, err)
			}
			if len(target.live.mission.ids) != len(live.Carried) {
				t.Fatalf("cycle %d mission holds %d members, the city held %d", cycle, len(target.live.mission.ids), len(live.Carried))
			}
			worlds = append(worlds, target.live.world)
		}
		for tick := 0; tick < 4; tick++ {
			if worlds[0].Hash() != worlds[1].Hash() {
				currentValueDiagnostics(t, "live/cold World", reflect.ValueOf(worlds[0]).Elem(), reflect.ValueOf(worlds[1]).Elem())
				t.Fatalf("cycle %d: live and cold World hashes differ %d ticks after mission entry", cycle, tick)
			}
			sim.Step(worlds[0], nil)
			sim.Step(worlds[1], nil)
		}

		m, chapter := cold.live.mission, cold.Town.Chapter()
		if next, line := cold.FinishMissionWithRoster(chapter, m.party, cold.live.world, m.ids, m.state.Start.Roster); next < 0 {
			t.Fatalf("cycle %d mission return: %s", cycle, line)
		}
		for _, member := range cold.Carried {
			if member.Hired() {
				t.Fatalf("cycle %d: a hired member %s returned from the mission", cycle, member.ID)
			}
		}
		lineage = cold
	}
}
