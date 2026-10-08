package game

import (
	"crypto/sha256"
	"encoding/json"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Controlled relocation and healing avoid replaying a whole campaign. Items
// come from the installed sacks through App Grab. The installed script alone
// changes relations, takes the quest item, commands the ogre and declares WIN.
// No register, latch, outcome, program or inventory setter supplies the result.
//
// The script also sends group 1, one ranged unit, to the ogre valley on the
// Swarm 2 order. A group that sees nothing walks on to its cell after a fight
// (AI-SWARM2GATE-107, AI-MOVE-023), so once the holder leaves the valley that
// unit walks from (19,28) to (26,25), and a holder waiting within its reach
// falls before the reward. A lone holder does not out-fight it and no setter
// may remove it, so the club flow keeps the holder out of its reach by cells:
// the peace and the SAVE wait at (18,58), beside the party's start, and the
// handover is made at (14,18), 12 cells from (26,25) and from (26,22), the cell
// the script sends group 3's ranged unit to.
func TestReleaseMission81QuestItemHandoverAndColdSAV(t *testing.T) {
	f := releaseFront(t)
	m := releaseMissionMap(t, f, 81)
	raw, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		latch int
		name  string
	}{{4, "Near Ogre2 with Bonus.1"}, {8, "Victory?.1"}, {15, "Victory?.2"}, {16, "Victory?.3"}, {17, "Victory?.4"}} {
		if want.latch >= len(raw.Triggers) || raw.Triggers[want.latch].Name != want.name {
			t.Fatalf("installed trigger %d is not %q", want.latch, want.name)
		}
	}
	for _, mode := range []string{"club", "sword", "companion-sword"} {
		t.Run(mode, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := f.NextParty()
			prepareAcceptedCampaignMission(t, f, 81)
			f.Carried = mapload.CloneParty(party)
			if mode == "companion-sword" {
				npc, ok := mapload.CampaignNPCMember(f.Table, 22, 30, party)
				if !ok {
					t.Fatal("installed AddHero22 companion unavailable")
				}
				party = append(party, npc)
			}
			a := f.App("Mission81 quest regression")
			a.Layout(1024, 768)
			quest81OK(t, a.OpenMission(f.MissionOpenerWith(81, party)))
			w := f.live.world
			hero := f.live.mission.ids[0]
			holder := hero
			if mode == "companion-sword" {
				holder = f.live.mission.ids[1]
			}
			quest81OK(t, a.HeadlessSelectEntity(uint32(holder)))
			quest81OK(t, a.HeadlessKey("0"))
			program := quest81Program(t, w)
			item := uint16(0x0e2c)
			near, far, sack := [2]int32{60, 62}, [2]int32{57, 62}, [2]int32{40, 24}
			hand := near
			if mode == "club" {
				item = 0x0e2b
				near, far, sack = [2]int32{19, 20}, [2]int32{18, 58}, [2]int32{33, 10}
				hand = [2]int32{14, 18}
				// Isolate the item predicate: peace and proximity both hold in
				// this separate cold session, but no one has taken the club.
				store, _, _ := campaignSave(t, f, true)
				noClub, noClubApp := campaignCold(t, store)
				if noClub.live == nil || noClub.live.world == nil {
					t.Fatalf("cold mission LOAD failed: screen=%v message=%q", noClubApp.Screen(), noClubApp.HeadlessMessage())
				}
				ogre, ok := mapload.ScriptUnits(m, party)[50]
				if !ok {
					t.Fatal("authored downed ogre u50 is absent")
				}
				quest81OK(t, noClub.live.world.HeadlessHeal(ogre))
				noClub.LiveAdvance(32)
				livePlaceAndWalk(t, noClub.live, holder, near[0], near[1])
				noClub.LiveAdvance(32)
				if noClub.live.world.Relations().Byte(2, 1) != 0 || quest81Carries(noClub.live.world, holder, item) || noClub.live.world.ScriptLatched(4) {
					t.Fatal("peaceful proximity without a club did not preserve the handover guard")
				}
			}
			// A blow already loaded finishes before a move order runs
			// (AI-ORDER-039, HERO-CADENCE-112; DIV-1563). A holder relocated in
			// mid-cycle would keep the cycle running against a victim out of reach
			// and start the walk only when it ended, six ticks late at the sack,
			// where a guard the test leaves alive then reaches the club's cell
			// first. So the cycle ends where the holder stands and the order is
			// given on a tick that takes it.
			move := func(cell [2]int32) {
				t.Helper()
				finishLoadedCycle(t, f.live, holder)
				livePlaceAndWalk(t, f.live, holder, cell[0], cell[1])
			}
			unchanged := func() {
				t.Helper()
				if w.ScriptLatched(4) || w.Outcome() != sim.OutcomeUndecided {
					t.Fatal("handover fired without all installed preconditions")
				}
			}
			// Proximity alone neither consumes an item nor wins the mission.
			move(near)
			f.LiveAdvance(32)
			unchanged()
			if mode == "club" {
				// Hostile guards stand around the club; they fall first.
				guards := 0
				for _, e := range w.Entities() {
					if e.Alive() && w.Relations().Hostile(e.Owner, sim.SelfSlot) && max(e.X-sack[0], sack[0]-e.X, e.Y-sack[1], sack[1]-e.Y) <= 4 {
						quest81OK(t, w.HeadlessKill(e.ID))
						guards++
					}
				}
				if guards == 0 {
					t.Fatal("no hostile guard stands near the club")
				}
			}
			move(sack)
			f.LiveAdvance(1)
			quest81Dismiss(t, f, a)
			quest81OK(t, a.HeadlessKey("grab"))
			if !quest81Carries(w, holder, item) {
				t.Fatalf("App Grab did not take installed item %04x", item)
			}
			if mode == "club" {
				// Club plus proximity is still insufficient while ogres are hostile.
				quest81OK(t, w.HeadlessHeal(holder))
				move(near)
				f.LiveAdvance(32)
				unchanged()
				if w.Relations().Byte(2, 1) == 0 || !quest81Carries(w, holder, item) {
					t.Fatalf("hostile control: relation=%d club=%v peace latches=%v/%v", w.Relations().Byte(2, 1), quest81Carries(w, holder, item), w.ScriptLatched(5), w.ScriptLatched(7))
				}
				move(far)
				refs := mapload.ScriptUnits(m, party)
				ogre, ok := refs[50]
				if !ok {
					t.Fatal("authored downed ogre u50 is absent")
				}
				quest81OK(t, w.HeadlessHeal(ogre))
				f.LiveAdvance(32)
				if w.Relations().Byte(2, 1) != 0 || !w.ScriptLatched(5) || !w.ScriptLatched(7) {
					t.Fatal("installed healed-ogre script did not establish peace")
				}
			}
			move(far)
			f.LiveAdvance(32)
			unchanged()
			// A newly raised message while an earlier modal is open is discarded.
			// Finish the healed-ogre dialogue before approaching for the reward.
			quest81Dismiss(t, f, a)
			if mode == "club" {
				quest81OK(t, w.HeadlessHeal(holder))
			}
			store, _, _ := campaignSave(t, f, true)
			cold, _ := campaignCold(t, store)
			quest81Paired(t, f, cold, 32)
			unchanged()
			for _, g := range []*FrontEnd{f, cold} {
				livePlaceAndWalk(t, g.live, holder, hand[0], hand[1])
			}
			quest81Paired(t, f, cold, 32)
			if mode == "club" {
				groups := sim.ObserveScriptGroups(w, 9)
				if !w.ScriptLatched(4) || quest81Carries(w, holder, item) || w.Outcome() != sim.OutcomeUndecided ||
					len(groups) != 1 || groups[0].Order != 4 || groups[0].CommandedX != 11 || groups[0].CommandedY != 16 {
					t.Fatalf("club handover: latch=%v club=%v outcome=%v group9=%+v", w.ScriptLatched(4), quest81Carries(w, holder, item), w.Outcome(), groups)
				}
				quest81Reward(t, f, a)
			} else if mode == "sword" {
				if !w.ScriptLatched(8) || w.Outcome() != sim.OutcomeWon || quest81Carries(w, holder, item) {
					t.Fatal("primary sword handover did not consume the sword and win")
				}
			} else {
				// The AddHero22 companion resolves hero ordinal 2.
				if w.ScriptLatched(8) || !w.ScriptLatched(15) || w.Outcome() != sim.OutcomeWon || quest81Carries(w, holder, item) {
					t.Fatal("the companion holding the sword did not win through hero ordinal 2")
				}
			}
			// Values 10003 and 10005 are filled by no party member.
			for _, latch := range []int32{15, 16, 17} {
				if latch == 15 && mode == "companion-sword" {
					continue
				}
				if w.ScriptLatched(latch) {
					t.Fatalf("unbound sword latch %d fired", latch)
				}
			}
			afterStore, _, _ := campaignSave(t, f, true)
			after, _ := campaignCold(t, afterStore)
			quest81Paired(t, f, after, 32)
			if quest81Program(t, w) != program || quest81Program(t, after.live.world) != program {
				t.Fatal("handover or cold LOAD changed the compiled program")
			}
			t.Logf("%s: actual sack/Grab, proximity controls, latches4/8/15/16/17, before/after cold SAV and96 paired ticks; final=%016x", mode, w.Hash())
		})
	}
}

func quest81OK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func quest81Carries(w *sim.World, holder sim.EntityID, item uint16) bool {
	items, ok := w.Carried(holder)
	return ok && slices.Contains(items, item)
}

func quest81Program(t *testing.T, w *sim.World) [32]byte {
	t.Helper()
	p := w.Script()
	b, err := json.Marshal([]any{p.Checks(), p.Instants(), p.Triggers()})
	quest81OK(t, err)
	return sha256.Sum256(b)
}

func quest81Paired(t *testing.T, a, b *FrontEnd, ticks int) {
	t.Helper()
	if a.live == nil || a.live.world == nil || b.live == nil || b.live.world == nil {
		t.Fatal("paired continuation requires two loaded missions")
	}
	for i := 0; i <= ticks; i++ {
		left, right := a.live.world.Hash(), b.live.world.Hash()
		equal := left == right
		if !equal {
			// A source-free current mission has no original session timer.  Its
			// SAV writer constructs the documented 1115 timer block on the wire;
			// the paired live world still carries the intentional all-zero block.
			// Compare both sides after that one construction so this witness
			// continues to check gameplay state rather than the producer's
			// disclosure of an absent timer.
			leftConstructed := worldHashWithConstructedCurrentSessionHead1115(t, a.live.world)
			rightConstructed := worldHashWithConstructedCurrentSessionHead1115(t, b.live.world)
			equal = right == leftConstructed || left == rightConstructed || leftConstructed == rightConstructed
		}
		if !equal {
			logCurrentCarrierDiff(t, a.live.world, b.live.world)
			t.Fatalf("cold continuation differs at tick%d: %016x/%016x", a.live.world.Tick(), a.live.world.Hash(), b.live.world.Hash())
		}
		if i < ticks {
			a.LiveAdvance(1)
			b.LiveAdvance(1)
		}
	}
}

func quest81Dismiss(t *testing.T, f *FrontEnd, a *ui.App) {
	t.Helper()
	for i := 0; i < 30; i++ {
		_, kind, open := f.LiveNotice()
		if !open {
			return
		}
		if kind != ui.NoticeDialogue {
			t.Fatal("unexpected terminal notice before handover", kind)
		}
		quest81OK(t, a.HeadlessKey("enter"))
	}
	t.Fatal("dialogue did not close")
}

func quest81Reward(t *testing.T, f *FrontEnd, a *ui.App) {
	t.Helper()
	payload, ok := ReadEventText(f.Archives.Containers, 81, 5)
	if !ok {
		t.Fatal("installed reward event5 unavailable")
	}
	expected, ok := expectedInstalledDialoguePart(t, payload, 1, HeroAudience(f.LiveParty()))
	if !ok || expected == "" {
		t.Fatal("installed reward first part unavailable")
	}
	before := f.live.world.Hash()
	found := false
	for i := 0; i < 30; i++ {
		text, kind, open := f.LiveNotice()
		if !open {
			break
		}
		found = found || text == expected
		if kind != ui.NoticeDialogue {
			t.Fatal("club unexpectedly reached a terminal notice", kind)
		}
		quest81OK(t, a.HeadlessKey("enter"))
	}
	if !found || f.live.world.Hash() != before {
		t.Fatalf("reward visible=%v; paused notice handling changed world=%v", found, f.live.world.Hash() != before)
	}
}
