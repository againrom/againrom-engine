package game

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseTerminalDefeat1091AppNativeContinuation(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1091-terminal-defeat")
	app.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	write := func(label string) string {
		t.Helper()
		snap, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		data, err := f.ExportCurrentSave(snap, label)
		if err != nil {
			t.Fatal(err)
		}
		name, err := store.WriteOriginal("", data)
		if err != nil {
			t.Fatal(err)
		}
		return name
	}
	runningName := write("running 1091")
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
	words := LoadInstallWords(f.Archives.Containers, f.textCode())
	wantTitle, _ := words.Global(141)
	exit, _ := words.Dialogs(44)
	load, _ := words.Dialogs(35)
	if title != wantTitle || f.live.view.Words().MenuExitMain != exit || f.live.view.Words().MenuLoad != load {
		t.Fatal("failure labels do not match main[141], dialogs[44]/[35]")
	}
	for _, key := range []string{"f2", "f4"} {
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
		rows, err := store.List()
		if err != nil || len(rows) != 1 || app.Screen() != ui.ScreenMap || !app.HeadlessNoticeOpen() {
			t.Fatalf("terminal %s exposed SAVE: rows=%d screen=%s notice=%t err=%v", key, len(rows), app.Screen(), app.HeadlessNoticeOpen(), err)
		}
	}
	if err := app.HeadlessActivate("save game"); err == nil {
		t.Fatal("terminal panel exposed a SAVE control")
	}
	t.Logf("terminal no-SAVE: F2 and F4 keep one pre-loss SAV and the failure panel; no SAVE control")
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
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "docs", "1091", "terminal-defeat.json"))
	if err != nil {
		t.Fatal(err)
	}
	// DIV-610: the forced post-panel checkpoint belonged only to the AGS
	// instrument. Keep the live failure, cancel, exit and pre-loss SAV LOAD.
	if len(scenario.Steps) < 9 || scenario.Steps[7].Target != "lost 1091" {
		t.Fatal("terminal scenario checkpoint shape changed")
	}
	scenario.Steps = append(scenario.Steps[:6], scenario.Steps[9:]...)
	var trace bytes.Buffer
	if err := RunHeadlessScenario(f, app, scenario, io.Discard, &trace); err != nil {
		t.Fatalf("scenario: %v\n%s", err, trace.String())
	}
	if f.live == old || f.live.mission.outcome == sim.OutcomeLost {
		t.Fatal("successful Load did not replace lost session")
	}
	t.Logf("m10 primary death failure tick=%d; installed title/Exit/Load; missing+malformed refusal; terminal SAVE blocked; cancel/exit and pre-loss SAV restore: PASS", old.world.Tick())
}
