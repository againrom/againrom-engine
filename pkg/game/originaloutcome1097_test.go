package game

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestOriginalOutcome1097SelectsUniqueHumanNotPlayerPosition(t *testing.T) {
	f := &sav.File{Body: make([]byte, 4374), World: &sav.WorldHalf{}, Players: []sav.Player{
		{Participant: 1, Outcome: 2}, {Participant: 0, Outcome: 1}, {Participant: 1},
	}}
	binary.LittleEndian.PutUint32(f.Body[4362:], 2)
	binary.LittleEndian.PutUint32(f.Body[4370:], 1)
	state, present, err := originalSessionState(f)
	if err != nil || !present || state.Outcome != sim.OutcomeWon || state.Won != 2 || state.Lost != 1 {
		t.Fatalf("handoff = %+v/%v/%v", state, present, err)
	}
	for _, players := range [][]sav.Player{
		{{Participant: 1, Outcome: 1}},
		{{Participant: 0, Outcome: 1}, {Participant: 0, Outcome: 2}},
		{{Participant: 0, Outcome: 3}},
	} {
		f.Players = players
		if _, present, err := originalSessionState(f); err == nil || present {
			t.Fatalf("accepted invalid Player population %+v", players)
		}
	}
	// City construction must not smuggle a previous victory into a fresh map.
	f.World = nil
	f.Players = []sav.Player{{Participant: 0, Outcome: 1}}
	if state, present, err := originalSessionState(f); err != nil || present || state.Outcome != sim.OutcomeUndecided {
		t.Fatalf("city inherited mission outcome: %+v/%v/%v", state, present, err)
	}
}

func savedOutcome1097(t *testing.T, outcome byte, won, lost uint32) []byte {
	t.Helper()
	// This fixture isolates session state from actor materialization.
	f, err := sav.Open(savedFileWithSession(t, 10, nil, 7, 1, 3, 4, 1))
	if err != nil {
		t.Fatal(err)
	}
	// Independent field widths: CString "Hero", slot16, slot32, raw8,
	// byte44, participant32, word2c, encoded purse32, outcome8.
	f.Body[f.Players[0].Off+1+4+2+4+8+1+4+2+4] = outcome
	binary.LittleEndian.PutUint32(f.Body[f.World.SessionOff+4362:], won)
	binary.LittleEndian.PutUint32(f.Body[f.World.SessionOff+4370:], lost)
	return savedContainer(f.Body)
}

func TestOriginalOutcome1097MalformedOutcomeLeavesLiveGame(t *testing.T) {
	f := missionFrontEnd(t)
	f.Carried = nil
	beforeTown, beforeDifficulty := f.Town, f.Difficulty
	opener, town, err := f.RestoreOriginal(savedOutcome1097(t, 3, 1, 0))
	if err == nil || opener != nil || town {
		t.Fatalf("malformed outcome accepted: %v", err)
	}
	if f.Town != beforeTown || f.Difficulty != beforeDifficulty || f.live != nil || f.Carried != nil {
		t.Fatal("malformed original outcome changed live session")
	}
}

func TestOriginalOutcome1097CityStartsFreshMission(t *testing.T) {
	f := missionFrontEnd(t)
	if _, town, err := f.RestoreOriginal(savedFile(0, tenActors())); err != nil || !town {
		t.Fatalf("load completed city: %v/%v", town, err)
	}
	open := f.MissionOpener(10)
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	if won, lost := f.live.world.ScriptCounters(); won != 0 || lost != 0 || f.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("city carried terminal outcome into new mission: %v %d/%d", f.live.world.Outcome(), won, lost)
	}
}

func TestOriginalOutcome1097NativeBeforeAcknowledgementAndTerminalDefeat(t *testing.T) {
	for _, outcome := range []byte{1, 2} {
		f := missionFrontEnd(t)
		f.Font = resolved(missionFont(), nil)
		f.SetDeterministicFrames(true)
		a := f.App("1097-synthetic-original")
		a.Layout(1024, 768)
		payload := savedOutcome1097(t, outcome, 0, 0)
		opener, town, err := f.RestoreOriginal(payload)
		if err != nil || town {
			t.Fatalf("synthetic original restore: %v", err)
		}
		if err := a.OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		wantKind := ui.NoticeSuccess
		if outcome == 2 {
			wantKind = ui.NoticeFailure
		}
		if _, kind, open := f.LiveNotice(); !open || kind != wantKind || f.live.world.Tick() != rawSavedSubTick1112(t, payload) {
			t.Fatalf("first restored frame: open=%v kind=%v tick=%d", open, kind, f.live.world.Tick())
		}
		snap, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		b, err := EncodeSave(snap, label)
		if err != nil {
			t.Fatal(err)
		}
		back, _, err := DecodeSave(b)
		if err != nil {
			t.Fatal(err)
		}
		opener, town, err = f.Restore(back)
		if err != nil || town {
			t.Fatalf("native restore: %v", err)
		}
		if err := a.OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		after, _ := f.live.world.MarshalBinary()
		if !bytes.Equal(snap.World, after) {
			t.Fatal("native reload changed canonical outcome/counters/world")
		}
		if _, kind, open := f.LiveNotice(); !open || kind != wantKind {
			t.Fatal("native reload lost terminal first frame")
		}
		for i := 0; i < 32; i++ {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		after, _ = f.live.world.MarshalBinary()
		if !bytes.Equal(snap.World, after) {
			t.Fatal("unacknowledged outcome advanced the world or repeated rewards")
		}
		if outcome == 2 {
			for _, action := range []ui.NoticeAction{ui.NoticeVictory, ui.NoticeContinue} {
				if dest, _, _ := f.live.advanceNotice(action); dest != ui.NoticeStay {
					t.Fatal("saved defeat admitted victory or continuation")
				}
			}
			if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenMenu {
				t.Fatalf("defeat exit = %s/%v", a.Screen(), err)
			}
			if f.Town.Done(10) {
				t.Fatal("defeat completed a campaign mission")
			}
		}
	}
}

func TestOriginalOutcome1097ContinueCanFailButLossCannotRecover(t *testing.T) {
	f := missionFrontEnd(t)
	f.Font = resolved(missionFont(), nil)
	f.SetDeterministicFrames(true)
	a := f.App("1097-synthetic-win-to-loss")
	a.Layout(1024, 768)
	open, _, err := f.RestoreOriginal(savedOutcome1097(t, 1, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("escape"); err != nil { // Continue
		t.Fatal(err)
	}
	for i := 0; i < 64 && f.live.world.Outcome() != sim.OutcomeLost; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if text, kind, up := f.LiveNotice(); !up || kind != ui.NoticeFailure || f.live.world.Outcome() != sim.OutcomeLost {
		t.Fatalf("continued victory did not become terminal failure: %q %v/%v", text, kind, up)
	}
	if f.live.mission.delayedVictory || f.Town.Done(10) {
		t.Fatal("saved failure retained delayed Victory or completed the mission")
	}
	if got := (OriginalSaveResume{HasWorld: true, Outcome: 2, SessionApplied: true, WinCount: 1}).String(); !strings.Contains(got, "WIN/LOSE 1/0 and Player outcome 2 RESTORED") {
		t.Fatalf("report hides independent counters/outcome: %s", got)
	}
}
