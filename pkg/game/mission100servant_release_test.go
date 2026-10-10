package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Mission 100's servant: unit 135, owner slot 5, the only member of group 21
// (TRIG-M100-095, PARTY-M100-033).
const (
	mission100Servant      = 135
	mission100ServantOwner = 5
	mission100ServantGroup = 21
)

// mission100Party is the party mission 100 is entered with: the primary
// alone, or the campaign roster with the four companions npc 22..25.
func mission100Party(t *testing.T, f *FrontEnd, full bool) []mapload.PartyMember {
	t.Helper()
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Servant witness", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	for _, mission := range f.Campaign.Value().Main {
		if mission >= 100 {
			break
		}
		if full {
			f.addChapterCompanions(mission)
		}
		f.Town.Won(mission)
	}
	// Ordinal k reads npc section 21+k (TRIG-HEROTPL-076): 10002..10005 are
	// npc 22..25, each carried when the campaign has not carried it already.
	for npc := 22; full && npc <= 25; npc++ {
		held := false
		for _, p := range f.Carried {
			held = held || p.CompanionNPC == npc
		}
		if held {
			continue
		}
		member, ok := mapload.CampaignNPCMember(f.Table, int32(npc), 0, f.Carried)
		if !ok {
			t.Fatalf("no npc %d to carry", npc)
		}
		f.Carried = append(f.Carried, member)
	}
	takeCampaignOffer(t, f, 100)
	return f.NextParty()
}

// mission100Unit245 is the npc24 placement the ordinal scan reaches after the
// party: it takes 10004 for every primary (TRIG-MAPORD-107).
const mission100Unit245 = 245

// TestReleaseMission100ServantLeavesAtTheWin enters mission 100 with three
// rosters: the primary alone; the owner's four-member roster, the primary and
// npc 22, 23 and 25, whose members resolve 10002, 10003 and 10005 but not
// 10004; and the full campaign roster. T1 starts the servant's Follow
// (AI-437). At the win T16 runs its four take slots; each role of
// 10002..10005 the scan leaves unresolved names an unbuilt node and runs
// instant subscript 0, action 2, group 21's Move to (13,14) (TRIG-M100-096).
// The scan reaches unit 245 after the party, which takes 10004 when no member
// does (TRIG-MAPORD-107, TRIG-MAPORD-108). The primary alone therefore ends
// the Follow, and the four-member and full rosters keep it. The two rosters
// with an unresolved or map-bound role are saved and cold-loaded before the
// win: the loaded program equals the one the mission load built, and the
// LOAD path's role compile keeps its shape.
func TestReleaseMission100ServantLeavesAtTheWin(t *testing.T) {
	for _, roster := range []string{"primary", "owner", "full"} {
		t.Run(roster, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := mission100Party(t, f, roster != "primary")
			if roster == "owner" {
				party = slices.DeleteFunc(party, func(p mapload.PartyMember) bool { return p.CompanionNPC == 24 })
				if len(party) != 4 {
					t.Fatalf("the four-member roster holds %d", len(party))
				}
			}
			app := f.App("mission 100 servant")
			if err := app.OpenMission(f.MissionOpenerWith(100, party)); err != nil {
				t.Fatal(err)
			}
			m := f.live.mission.state.Map
			refs := campaignScriptRefs(m, f.Table, party, 100)
			units := mapload.ScriptUnits(m, party)
			resolved := 0
			if refs.HasCompanion {
				resolved++
			}
			for v := uint32(10003); v <= 10005; v++ {
				if _, ok := refs.Roles[v]; ok {
					resolved++
				}
			}
			want := map[string]int{"primary": 1, "owner": 4, "full": 4}[roster]
			if resolved != want {
				t.Fatalf("the scan resolves %d of 10002..10005, want %d", resolved, want)
			}
			id, ok := refs.Roles[10004]
			if map245 := ok && id == units[mission100Unit245]; map245 == (roster == "full") {
				t.Fatalf("10004 is bound to %d (resolved %v); unit 245 is %d", id, ok, units[mission100Unit245])
			}
			every := refs
			every.Roster = false
			whole, _, err := mapload.CompileScript(m, every)
			if err != nil {
				t.Fatal(err)
			}
			wantInstants := len(whole.Instants()) - (4 - resolved)
			if n := len(f.live.world.Script().Instants()); n != wantInstants {
				t.Fatalf("the mission built %d instants, want %d", n, wantInstants)
			}
			for i := 0; i < 20; i++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if roster != "full" {
				built := f.live.world.Script()
				store, _, _ := campaignSave(t, f, true)
				f, app = campaignCold(t, store)
				loaded := f.live.world.Script()
				if !reflect.DeepEqual(loaded.Checks(), built.Checks()) || !reflect.DeepEqual(loaded.Instants(), built.Instants()) ||
					!reflect.DeepEqual(loaded.Triggers(), built.Triggers()) {
					t.Fatal("the loaded program differs from the one the mission load built")
				}
				ms := f.live.mission.state
				again, _, err := mapload.CompileScript(ms.Map, campaignScriptPartyRefs(ms.Map, f.Table, ms.Party,
					func(i int) sim.EntityID { return ms.Start.IDs[i] }, loadedPlacedHeroes(ms, f.Table)))
				if err != nil || len(again.Instants()) != len(loaded.Instants()) || len(again.Checks()) != len(loaded.Checks()) ||
					!reflect.DeepEqual(again.Triggers(), loaded.Triggers()) {
					t.Fatal("the LOAD path's role compile does not keep the program's shape", err)
				}
			}
			w := f.live.world
			units = mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
			servant, ok := w.Entity(units[mission100Servant])
			if !ok || servant.ActorState != 0x11 {
				t.Fatalf("the servant does not follow before the win: %+v", servant)
			}
			if err := w.HeadlessHeal(units[mission100Unit245]); err != nil {
				t.Fatal(err)
			}
			if err := w.HeadlessKill(units[168]); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 300 && w.Outcome() != sim.OutcomeWon; i++ {
				if _, kind, up := f.LiveNotice(); up && kind != ui.NoticeSuccess {
					if err := app.HeadlessActivate("notice"); err != nil {
						t.Fatal(err)
					}
				}
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if w.Outcome() != sim.OutcomeWon {
				t.Fatal("mission 100 did not reach script victory")
			}
			order, _, _ := w.FrozenGroupAI(mission100ServantOwner, mission100ServantGroup)
			servant, _ = w.Entity(units[mission100Servant])
			moved := order == 4 && servant.ActorState != 0x11 && !servant.HasEscortTarget &&
				servant.HasTarget && servant.TargetX == 13 && servant.TargetY == 14
			if moved != (roster == "primary") {
				t.Fatalf("at the win group 21 order=%d, servant state=%#x escort=%v target=(%d,%d,%v); want Move to (13,14) %v",
					order, servant.ActorState, servant.HasEscortTarget, servant.TargetX, servant.TargetY, servant.HasTarget, roster == "primary")
			}
			if roster != "primary" && servant.ActorState != 0x11 {
				t.Fatalf("the %s roster ends the Follow: state %#x", roster, servant.ActorState)
			}
		})
	}
}
