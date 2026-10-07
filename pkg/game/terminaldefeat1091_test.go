package game

import (
	"bytes"
	"encoding/base64"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestTerminalDefeat1091DeathScriptAndSentinelActions(t *testing.T) {
	for _, cause := range []string{"death", "script", "sentinel"} {
		t.Run(cause, func(t *testing.T) {
			var mw *mapWorld
			if cause == "script" {
				ms := continuityMission(t, 30, sim.ScriptInstantLose)
				ms.Map = worldFixtureMap()
				mw = openMission(ms, nil, nil, worldFixtureViewer(t, ms.Map), missionSource{}, nil, nil)
			} else if cause == "sentinel" {
				mw, _ = missionAudienceDriver(t, ReservedMessageNumber, "", nil)
			} else {
				mw = grabWorld(t, heroWorld(t, nil, nil, nil), 99, missionSource{})
			}
			mw.view.SetFont(missionFont())
			missionSteps(mw, 2)
			if _, kind, open := mw.view.NoticeState(); !open || kind != ui.NoticeFailure {
				t.Fatalf("%s did not open terminal failure: %v/%v", cause, kind, open)
			}
			before, _ := mw.world.MarshalBinary()
			for _, action := range []ui.NoticeAction{ui.NoticeVictory, ui.NoticeContinue} {
				if dest, _, _ := mw.advanceNotice(action); dest != ui.NoticeStay || !mw.view.NoticeOpen() {
					t.Fatal("failure admitted Victory/Continue")
				}
			}
			if dest, _, _ := mw.advanceNotice(ui.NoticeLoadGame); dest != ui.NoticeToLoad || !mw.view.NoticeOpen() {
				t.Fatal("Load did not retain the terminal panel")
			}
			r := mw.residue()
			if !r.MissionLost {
				t.Fatal("native residue lost the terminal latch")
			}
			if dest, _, _ := mw.advanceNotice(ui.NoticeExitMain); dest != ui.NoticeToMenu {
				t.Fatal("Exit did not reach menu")
			}
			after, _ := mw.world.MarshalBinary()
			if !bytes.Equal(before, after) {
				t.Fatal("terminal actions repaired HP/MP or changed XP/world")
			}
			mw.applyResidue(r)
			mw.openLoadNotice([]string{"Older save disclosure"})
			mw.advanceNotice()
			if _, kind, open := mw.view.NoticeState(); !open || kind != ui.NoticeFailure {
				t.Fatal("load disclosure hid the terminal panel permanently")
			}
		})
	}
}

func TestTerminalDefeat1091NativeLatchRoundTripAndOldDefault(t *testing.T) {
	f := missionFrontEnd(t)
	f.Font = resolved(missionFont(), nil)
	f.SetDeterministicFrames(true)
	a := f.App("1091-native")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	// A reserved-message loss need not have a simulation outcome counter.
	f.live.mission.announced, f.live.mission.outcome = true, sim.OutcomeLost
	f.live.showOutcome()
	snap, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := DecodeSave(b)
	if err != nil || !back.Residue.MissionLost {
		t.Fatalf("native lost latch: %v/%v", back.Residue.MissionLost, err)
	}
	open, town, err := f.Restore(back)
	if err != nil || town {
		t.Fatalf("Restore: %v/%v", town, err)
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeFailure {
		t.Fatal("first restored frame is playable")
	}
	hash, tick := f.live.world.Hash(), f.live.world.Tick()
	for i := 0; i < 50; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Hash() != hash || f.live.world.Tick() != tick {
		t.Fatal("restored lost session advanced")
	}
	old, err := base64.StdEncoding.DecodeString(preWorldMapReturn1088Envelope)
	if err != nil {
		t.Fatal(err)
	}
	oldSnap, _, err := DecodeSave(old)
	if err != nil || oldSnap.Residue.MissionLost {
		t.Fatalf("old default: %v/%v", oldSnap.Residue.MissionLost, err)
	}
}

func TestTerminalDefeat1091LostDisclosureWithoutFontStaysTerminal(t *testing.T) {
	for _, loseFontAfterOpen := range []bool{false, true} {
		f := missionFrontEnd(t)
		f.Font = resolved(missionFont(), nil)
		f.SetDeterministicFrames(true)
		a := f.App("1091-disclosure")
		a.Layout(1024, 768)
		if err := a.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
		f.live.mission.announced, f.live.mission.outcome = true, sim.OutcomeLost
		f.live.showOutcome()
		snap, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		opener, town, err := f.Restore(snap)
		if err != nil || town {
			t.Fatalf("restore: town=%v error=%v", town, err)
		}
		if err := a.OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		mw := f.live
		if !loseFontAfterOpen {
			mw.view.SetFont(nil)
		}
		pages := mw.view.DialoguePages([]string{"Older-save disclosure"})
		mw.openLoadNotice(pages)
		mw.view.SetFont(nil)
		before, tick := mw.world.Hash(), mw.world.Tick()
		check := func() {
			t.Helper()
			if !mw.view.NoticeOpen() || mw.world.Tick() != tick || mw.world.Hash() != before {
				t.Fatal("Lost/disclosure/no-font released input or changed world")
			}
		}
		for i := 0; i < 10; i++ {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			check()
		}
		// Normal input remains captured; no game menu or save/load shortcut
		// can bypass the disclosure. Escape remains an available way out.
		for _, key := range []string{"a", "right", "f2", "f3"} {
			if err := a.HeadlessKey(key); err != nil {
				t.Fatal(err)
			}
			check()
		}
		for range pages {
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			check()
		}
		if _, kind, open := mw.view.NoticeState(); !open || kind != ui.NoticeFailure {
			t.Fatal("disclosure did not return to failure")
		}
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if a.Screen() != ui.ScreenMenu || mw.world.Tick() != tick || mw.world.Hash() != before {
			t.Fatal("terminal exit failed or advanced the world")
		}
	}
}
