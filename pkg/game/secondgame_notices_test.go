package game

import (
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestSecondGameFailureShowsAuthoredReasonBeforeVictory(t *testing.T) {
	w := secondGameScriptWorld(t,
		[]sim.ScriptInstant{{Op: 4}, {Op: 5, Args: [10]int32{2}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0, 1)})
	src := missionSource{"main/text/mission7.txt": []byte("#failure2\r\nKeep him alive.\r\n#failure4\r\nOther failure.")}
	mw, v := missionDriverFor(t, w, nil, src)
	mw.mission.table = &mapload.Table{Game: base.GameROM2}
	mw.mission.failureText = secondGameFailureText(src, TextCode{}, 7)
	missionSteps(mw, 1)
	if body, kind, shown := v.NoticeState(); !shown || kind != ui.NoticeFailure || body != "Keep him alive.\r\n" {
		t.Fatalf("failure reason = %q/%v/%v", body, kind, shown)
	}
	if w.Outcome() != sim.OutcomeLost {
		t.Fatal("positive failure reason lost precedence over victory")
	}
}

func TestSecondGameProductionAudienceUsesCollectionPresence(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64),
		[]sim.Entity{{ID: 0, Owner: 2, X: 2, Y: 2, HP: 100, MaxHP: 100}})
	if err != nil {
		t.Fatal(err)
	}
	mw := &mapWorld{world: w, mission: &missionNotices{
		table: &mapload.Table{Game: base.GameROM2}, npcKeys: map[sim.EntityID]uint16{0: 7},
	}}
	payload := []byte("<part=1 npcalive=7 npc=42>\r\nretained\r\n<part=1 npcdead=7 npc=42>\r\nabsent")
	sim.Step(w, []sim.Command{sim.Kill(0)})
	if actor, ok := w.Entity(0); !ok || actor.HP > 0 {
		t.Fatal("control entity was not retained dead")
	}
	for _, present := range []bool{true, false} {
		if !present {
			mw.mission.npcKeys = nil
		}
		got, ok := dialoguePart(payload, 1, mw.eventAudience(1))
		want := "retained"
		if !present {
			want = "absent"
		}
		if !ok || got != want {
			t.Fatalf("collection presence %v selected %q/%v, want %q", present, got, ok, want)
		}
	}
}

func TestSecondGameObjectivesReadLiveFlagsWithoutAcknowledging(t *testing.T) {
	w := secondGameScriptWorld(t,
		[]sim.ScriptInstant{{Op: 35, Args: [10]int32{752, 3}}, {Op: 35, Args: [10]int32{753, 5}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0, 1)})
	mw := &mapWorld{world: w, mission: &missionNotices{
		table: &mapload.Table{Game: base.GameROM2}, objectiveLabels: []string{"first", "second", "hidden"},
	}}
	for i := 0; i < 48; i++ {
		sim.Step(w, nil)
	}
	got := gameMenuContext(mw, true, "briefing").Objective
	if got != "briefing\n\n[+] first\n[-] second" || strings.Contains(got, "hidden") {
		t.Fatalf("objectives = %q", got)
	}
	for index, want := range map[int32]int32{752: 3, 753: 5, 754: 0} {
		if value, ok := w.ROM2ScenarioValue(index); !ok || value != want {
			t.Fatalf("opening objectives wrote scenario[%d]: %d %v", index, value, ok)
		}
	}
}

func TestSecondGameMessagesKeepActionOrderBeforeVictory(t *testing.T) {
	w := secondGameScriptWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}, {Op: 2, Args: [10]int32{3}}, {Op: 4}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0, 1, 2)})
	// Reverse announcement metadata cannot reorder actual script execution.
	mw, v := missionDriverFor(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 3}, {Latch: 3, Event: 4}},
		missionSource{"main/text/mission7.txt": []byte("#event3\r\n<part=1>\r\nthree\r\n#event4\r\n<part=1>\r\nfour\r\n")})
	mw.mission.table = &mapload.Table{Game: base.GameROM2}
	missionSteps(mw, 1)
	assert := func(want string, kind ui.NoticeKind) {
		t.Helper()
		got, gotKind, shown := v.NoticeState()
		if !shown || gotKind != kind || want != "" && got != want {
			t.Fatalf("notice = %q/%v/%v, want %q/%v", got, gotKind, shown, want, kind)
		}
	}
	assert("four\r\n", ui.NoticeDialogue)
	mw.advanceNotice()
	assert("three\r\n", ui.NoticeDialogue)
	mw.advanceNotice()
	assert("", ui.NoticeSuccess)
	if w.Outcome() != sim.OutcomeWon {
		t.Fatal("dialogue delivery lost the victory trigger")
	}
	mw.advanceNotice(ui.NoticeContinue)
	missionSteps(mw, 3)
	if _, _, shown := v.NoticeState(); shown {
		t.Fatal("one-shot messages or victory reopened after dismissal")
	}
}

func secondGameScriptWorld(t *testing.T, instants []sim.ScriptInstant, triggers []sim.ScriptTrigger) *sim.World {
	t.Helper()
	s, err := sim.NewROM2Script(missionChecks(), instants, triggers)
	if err != nil {
		t.Fatal(err)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), nil, s)
	if err != nil {
		t.Fatal(err)
	}
	return w
}
