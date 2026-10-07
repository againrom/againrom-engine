package game

import (
	"bytes"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func victoryContinueDriver(t *testing.T, mission int) (*Mission, *mapWorld, *ui.Viewer) {
	t.Helper()
	ms := continuityMission(t, mission, sim.ScriptInstantWin)
	ms.Map = worldFixtureMap()
	v := worldFixtureViewer(t, ms.Map)
	v.SetFont(missionFont())
	mw := openMission(ms, nil, nil, v, missionSource{}, nil, nil)
	missionSteps(mw, 1)
	if text, kind, open := v.NoticeState(); !open || kind != ui.NoticeSuccess || text != MissionWonText {
		t.Fatalf("success notice = %q/%v/%v", text, kind, open)
	}
	return ms, mw, v
}

func TestCampaignVictoryContinueStateTableAndOneCompletionBoundary(t *testing.T) {
	// Immediate Victory and delayed Victory use fresh sessions but the same
	// continuity wrapper. Each must produce the campaign completion destination
	// once and then become inert.
	t.Run("immediate Victory", func(t *testing.T) {
		ms, mw, _ := victoryContinueDriver(t, 10)
		f := continuityFront(t)
		advance := f.continuity(10, ms, mw.advanceNotice)
		if dest, _, _ := advance(ui.NoticeVictory); dest != ui.NoticeToMission {
			t.Fatalf("Victory destination = %v, want successor mission", dest)
		}
		if dest, _, _ := advance(ui.NoticeVictory); dest != ui.NoticeStay {
			t.Fatalf("repeated Victory destination = %v, want stay", dest)
		}
	})

	t.Run("Continue then End Quest Victory", func(t *testing.T) {
		ms, mw, v := victoryContinueDriver(t, 10)
		f := continuityFront(t)
		advance := f.continuity(10, ms, mw.advanceNotice)

		beforeForm, err := mw.world.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if dest, _, _ := advance(ui.NoticeContinue); dest != ui.NoticeStay {
			t.Fatalf("Continue destination = %v, want stay", dest)
		}
		if v.NoticeOpen() || len(f.Carried) != 0 {
			t.Fatalf("Continue left notice=%v or completed roster=%d", v.NoticeOpen(), len(f.Carried))
		}
		afterForm, err := mw.world.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(beforeForm, afterForm) {
			t.Fatal("Continue changed persisted simulation bytes")
		}

		beforeTick := mw.world.Tick()
		mw.tick()
		if mw.world.Tick() <= beforeTick {
			t.Fatal("the completed world stopped after Continue")
		}
		ctx := gameMenuContext(mw, true, "")
		if !ctx.Campaign || !ctx.CampaignVictory || !ctx.VictoryAvailable {
			t.Fatalf("menu context after Continue = %+v", ctx)
		}

		if dest, _, _ := advance(ui.NoticeVictory); dest != ui.NoticeToMission {
			t.Fatalf("End Quest Victory destination = %v, want successor mission", dest)
		}
		if len(f.Carried) != 1 {
			t.Fatalf("completed roster = %d, want 1", len(f.Carried))
		}
		if dest, _, _ := advance(ui.NoticeVictory); dest != ui.NoticeStay {
			t.Fatalf("repeated End Quest Victory destination = %v, want stay", dest)
		}
	})
}

func TestLoadedCompletedWorldResetsDelayedVictoryPermission(t *testing.T) {
	ms, mw, _ := victoryContinueDriver(t, 10)
	if dest, _, _ := mw.advanceNotice(ui.NoticeContinue); dest != ui.NoticeStay {
		t.Fatalf("Continue destination = %v", dest)
	}
	if !gameMenuContext(mw, true, "").VictoryAvailable {
		t.Fatal("setup: delayed Victory was not enabled")
	}

	form, err := mw.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	loaded := &sim.World{}
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if loaded.Outcome() != sim.OutcomeWon {
		t.Fatalf("loaded outcome = %v, want won", loaded.Outcome())
	}
	loadedMission := *ms
	loadedMission.World = loaded
	v := worldFixtureViewer(t, loadedMission.Map)
	v.SetFont(missionFont())
	resumed := openMission(&loadedMission, nil, nil, v, missionSource{}, nil, nil)
	ctx := gameMenuContext(resumed, true, "")
	if !ctx.Campaign || !ctx.CampaignVictory || ctx.VictoryAvailable {
		t.Fatalf("first End Quest context after load = %+v", ctx)
	}
	// The restored outcome presents its success panel before the first tick.
	// This does not restore the previous session's delayed End Quest permission.
	if _, kind, open := resumed.view.NoticeState(); !open || kind != ui.NoticeSuccess {
		t.Fatal("restored completed world omitted the first-frame success panel")
	}
	if dest, _, _ := resumed.advanceNotice(ui.NoticeContinue); dest != ui.NoticeStay ||
		!gameMenuContext(resumed, true, "").VictoryAvailable {
		t.Fatalf("fresh Continue did not enable delayed Victory: %v", dest)
	}
}
