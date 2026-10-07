package game

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseTerminalDefeat1091AppNativeContinuation(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1091-terminal-defeat")
	app.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	write := func(label string, stamp int64) string {
		t.Helper()
		snap, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		data, err := EncodeSave(snap, label)
		if err != nil {
			t.Fatal(err)
		}
		name, err := store.Write(time.Unix(stamp, 0), data)
		if err != nil {
			t.Fatal(err)
		}
		return name
	}
	runningName := write("running 1091", 1)
	runningBytes, err := store.Read(runningName)
	if err != nil {
		t.Fatal(err)
	}
	// Canonical lethal command on the installed primary, with the existing
	// DYING window and dialogue acknowledged through App. No outcome is set.
	f.LiveKill(uint32(f.live.mission.ids[0]))
	for i := 0; i < 5000; i++ {
		_, kind, up := f.LiveNotice()
		if up && kind == ui.NoticeFailure {
			break
		}
		if up {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		} else if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	title, kind, up := f.LiveNotice()
	if !up || kind != ui.NoticeFailure {
		t.Fatalf("m10 primary death did not fail: %v/%v tick=%d outcome=%v", kind, up, f.live.world.Tick(), f.live.world.Outcome())
	}
	words := LoadInstallWords(f.Archives.Containers)
	wantTitle, _ := words.Global(141)
	exit, _ := words.Dialogs(44)
	load, _ := words.Dialogs(35)
	if title != wantTitle || f.live.view.Words().MenuExitMain != exit || f.live.view.Words().MenuLoad != load {
		t.Fatal("failure labels do not match main[141], dialogs[44]/[35]")
	}
	old := f.live
	before, _ := old.world.MarshalBinary()
	// The actual store list is captured at panel open; remove and corrupt a
	// listed row afterwards. Both refusals must retain the terminal session.
	for _, corrupt := range []bool{false, true} {
		if err := app.HeadlessActivate("load game"); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(store.Dir, runningName)
		if corrupt {
			if err := os.WriteFile(path, []byte("malformed save"), 0o600); err != nil {
				t.Fatal(err)
			}
		} else if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" || f.live != old {
			t.Fatal("refused load changed the session or hid refusal")
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		after, _ := old.world.MarshalBinary()
		if app.Screen() != ui.ScreenMap || !app.HeadlessNoticeOpen() || !bytes.Equal(before, after) {
			t.Fatal("cancel resumed failed gameplay or changed XP/HP")
		}
		// Restore only the temporary fixture, never the lawful install.
		if err := os.WriteFile(path, runningBytes, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A current native lost session is admitted by the normal loader, but is
	// still terminal on its first frame. The initial running save is rebuilt
	// from a separate fresh mission and is the only playable continuation.
	lostName := write("lost 1091", 2)
	lostBytes, err := store.Read(lostName)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(store.Dir, runningName)); err != nil {
		t.Fatal(err)
	}
	write("running 1091", 3)
	lostSnap, _, err := DecodeSave(lostBytes)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := f.Restore(lostSnap)
	if err != nil || town {
		t.Fatalf("restore lost: %v/%v", town, err)
	}
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeFailure {
		t.Fatal("native loss resumed without failure panel")
	}
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "docs", "1091", "terminal-defeat.json"))
	if err != nil {
		t.Fatal(err)
	}
	var trace bytes.Buffer
	if err := RunHeadlessScenario(f, app, scenario, io.Discard, &trace); err != nil {
		t.Fatalf("scenario: %v\n%s", err, trace.String())
	}
	if f.live == old || f.live.mission.outcome == sim.OutcomeLost {
		t.Fatal("successful Load did not replace lost session")
	}
	t.Logf("m10 primary death failure tick=%d; installed title/Exit/Load; missing+malformed refusal; native lost load/cancel/exit and running restore: PASS", old.world.Tick())
}
