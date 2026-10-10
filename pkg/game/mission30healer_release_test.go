package game

import (
	"bytes"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Mission 30's healer: T6 "Healer" (latch 6) fires when the hero stands within
// three cells of unit 56. Its slots run message 16, take the cure from 10001,
// take the cure from 10002 and the win. Subscript 0 is action 2, message 4.
const (
	mission30HealerLatch = 6
	mission30Healer      = 56
)

// mission30Party is the party mission 30 is entered with: the primary alone,
// which leaves 10002 unresolved, or the primary with npc 22, which resolves it.
func mission30Party(t *testing.T, f *FrontEnd, companion bool) []mapload.PartyMember {
	t.Helper()
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Healer witness", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	for _, mission := range f.Campaign.Value().Main {
		if mission >= 30 {
			break
		}
		f.Town.Won(mission)
	}
	f.Carried = f.Carried[:1]
	if companion {
		member, ok := mapload.CampaignNPCMember(f.Table, 22, 0, f.Carried)
		if !ok {
			t.Fatal("no npc 22 to carry")
		}
		f.Carried = append(f.Carried, member)
	}
	takeCampaignOffer(t, f, 30)
	return f.NextParty()
}

// TestReleaseMission30HealerWinRaisesSubscriptZerosMessage enters mission 30
// with 10002 unresolved and with it resolved, and stands the hero beside the
// healer. Unresolved, action 14 is not built and T6's third slot runs
// subscript 0 (TRIG-BIND-010), so the healer win raises message 16 and then
// message 4, in slot order. Message 16 opens its dialogue; message 4 arrives
// while it is open and is dropped (DLG-LIFE-005). The screen therefore shows
// message 16 and then the win in both rosters, and the resolved roster raises
// message 16 alone.
func TestReleaseMission30HealerWinRaisesSubscriptZerosMessage(t *testing.T) {
	for _, companion := range []bool{false, true} {
		name := "primary"
		if companion {
			name = "companion"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := mission30Party(t, f, companion)
			app := f.App("mission 30 healer")
			if err := app.OpenMission(f.MissionOpenerWith(30, party)); err != nil {
				t.Fatal(err)
			}
			ms := f.live.mission.state
			if refs := campaignScriptRefs(ms.Map, f.Table, party); refs.HasCompanion != companion {
				t.Fatalf("10002 resolved %v, want %v", refs.HasCompanion, companion)
			}
			var raised []int32
			for _, r := range ms.Raises {
				if r.Latch == mission30HealerLatch {
					raised = append(raised, r.Event)
				}
			}
			want := []int32{16, 4}
			if companion {
				want = []int32{16}
			}
			if !slices.Equal(raised, want) {
				t.Fatalf("T6 raises %v, want %v", raised, want)
			}
			text := map[int][]byte{}
			for _, event := range []int{4, 16} {
				payload, ok := ReadEventTextFor(f.live.mission.src, tableGame(f.live.mission.table), 30, event)
				if !ok {
					t.Fatalf("mission 30 ships no readable message %d", event)
				}
				text[event] = payload
			}

			w := f.live.world
			units := mapload.ScriptUnits(ms.Map, f.live.mission.party)
			healer, ok := w.Entity(units[mission30Healer])
			if !ok {
				t.Fatal("no healer")
			}
			if err := w.HeadlessPlace(f.live.mission.ids[0], healer.X, healer.Y); err != nil {
				t.Fatal(err)
			}
			var shown []int
			var outcome bool
			for i := 0; i < 600 && !outcome; i++ {
				m := f.live.mission
				if _, kind, up := f.LiveNotice(); up {
					if kind == ui.NoticeSuccess {
						outcome = true
						break
					}
					if kind == ui.NoticeDialogue && m.part == 1 {
						event := 0
						for e, payload := range text {
							if bytes.Equal(m.payload, payload) {
								event = e
							}
						}
						if len(shown) == 0 || shown[len(shown)-1] != event {
							shown = append(shown, event)
						}
					}
					if err := app.HeadlessActivate("notice"); err != nil {
						t.Fatal(err)
					}
				}
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if w.Outcome() != sim.OutcomeWon || !outcome {
				t.Fatalf("the healer did not win mission 30: outcome %v, shown %v", w.Outcome(), shown)
			}
			if !w.ScriptLatched(mission30HealerLatch) {
				t.Fatal("T6 did not fire")
			}
			if !slices.Equal(shown, []int{16}) {
				t.Fatalf("dialogues shown %v, want message 16 alone: message 4 meets its open panel", shown)
			}
		})
	}
}
