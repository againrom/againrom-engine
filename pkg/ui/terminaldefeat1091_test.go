package ui

import (
	"errors"
	"image"
	"testing"
	"time"
)

func failureApp1091(t *testing.T, entries []SaveEntry, load LoadGame) (*App, *noticeSeam) {
	t.Helper()
	a, seam := noticeApp(t)
	paused := false
	a.flow.cadence = func(_ int, stopped, _, _ bool) { paused = stopped }
	a.flow.tick = func() {
		if !paused {
			seam.ticks++
		}
	}
	a.SetSaveSeams(nil, func() []SaveEntry { return entries }, load)
	seam.v.SetNotice("Failed", NoticeFailure)
	a.flow.advance = func(actions ...NoticeAction) (NoticeDest, string, MapOpener) {
		seam.actions = append(seam.actions, actions[0])
		switch actions[0] {
		case NoticeExitMain:
			return NoticeToMenu, "", nil
		case NoticeLoadGame:
			return NoticeToLoad, "", nil
		}
		return NoticeStay, "", nil
	}
	return a, seam
}

func TestTerminalDefeat1091NoSavesDisablesVisibleLoad(t *testing.T) {
	for _, tc := range []struct {
		entries []SaveEntry
		load    LoadGame
	}{
		{},
		{entries: []SaveEntry{{Name: "old.ags"}}},
		{load: func(string) (MapOpener, bool, error) { return nil, false, nil }},
	} {
		a, seam := failureApp1091(t, tc.entries, tc.load)
		if !seam.v.noticeLayout().SecondaryDisabled {
			t.Fatal("Load enabled without a usable store/loader")
		}
		if err := a.HeadlessActivate("load game"); err == nil {
			t.Fatal("disabled load activated")
		}
		if _, hit := seam.v.noticeActionAt(320, 320); hit {
			t.Fatal("disabled button still hit")
		}
		before := seam.ticks
		for _, in := range []appInput{{}, {LoadGame: true}, {SaveGame: true}, {Attack: true}} {
			a.step(in, time.Unix(1_700_001_000, 0))
		}
		if seam.ticks != before || a.Screen() != ScreenMap || !seam.v.NoticeOpen() {
			t.Fatal("failure panel admitted gameplay or F2/F3")
		}
	}
}

func TestTerminalDefeat1091DisclosureGateClearsWithNotice(t *testing.T) {
	_, seam := failureApp1091(t, nil, nil)
	v := seam.v
	v.SetFont(nil)
	v.SetNotice("lost", NoticeFailure)
	v.SetDialogue(Dialogue{Text: "disclosure"})
	if !v.NoticeOpen() {
		t.Fatal("replacement disclosure cleared terminal input gate")
	}
	v.ClearNotice()
	v.SetDialogue(Dialogue{Text: "ordinary"})
	if v.NoticeOpen() {
		t.Fatal("closed failure leaked its gate to an ordinary fontless dialogue")
	}
}

func TestTerminalDefeat1091LoadRefusalsAndCancelKeepTheOldSession(t *testing.T) {
	for _, refusal := range []string{"read", "open", "nil opener", "missing town"} {
		t.Run(refusal, func(t *testing.T) {
			load := func(string) (MapOpener, bool, error) {
				switch refusal {
				case "read":
					return nil, false, errors.New("malformed or missing save")
				case "open":
					return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
						return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, errors.New("mission refused")
					}, false, nil
				case "missing town":
					return nil, true, nil
				}
				return nil, false, nil
			}
			a, seam := failureApp1091(t, []SaveEntry{{Name: "failed.ags", Label: "save"}}, load)
			before := seam.ticks
			if err := a.HeadlessActivate("load game"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != ScreenLoad || a.flow.viewer != seam.v {
				t.Fatal("load did not retain the failed mission")
			}
			a.HeadlessKey("enter")
			if a.Screen() != ScreenLoad || a.flow.msg == "" {
				t.Fatal("load refusal was not reported on the list")
			}
			a.HeadlessKey("escape")
			a.HeadlessStep()
			if a.Screen() != ScreenMap || a.flow.viewer != seam.v || !seam.v.NoticeOpen() || a.flow.advance == nil || seam.ticks != before {
				t.Fatal("cancel resumed a playable or torn-down mission")
			}
			if err := a.HeadlessActivate("exit to main menu"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != ScreenMenu || a.flow.viewer != nil {
				t.Fatal("explicit terminal exit retained the map")
			}
		})
	}
}

func TestTerminalDefeat1091LoadSuccessReplacesSession(t *testing.T) {
	opener := &successorOpener{}
	a, seam := failureApp1091(t, []SaveEntry{{Name: "ok.ags"}}, func(string) (MapOpener, bool, error) { return opener.open, false, nil })
	before := seam.ticks
	if err := a.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	a.HeadlessKey("enter")
	if a.Screen() != ScreenMap || a.flow.viewer == seam.v || seam.ticks != before {
		t.Fatal("successful Load did not replace the failed session atomically")
	}
}

func TestTerminalDefeat1091ButtonsUseLocalizedWordsAndDisabledPaint(t *testing.T) {
	a, seam := failureApp1091(t, nil, nil)
	w := AuthoredWords()
	w.MenuExitMain, w.MenuLoad = "~Exit test", "~Load test"
	seam.v.SetWords(w)
	l := seam.v.noticeLayout()
	if l.ButtonLabel != w.MenuExitMain || l.SecondaryButtonLabel != w.MenuLoad {
		t.Fatal("failure captions ignored the installed word set")
	}
	disabled := RenderNotice(l, panelFont(), "Failed", nil)
	l.SecondaryDisabled = false
	enabled := RenderNotice(l, panelFont(), "Failed", nil)
	different := 0
	for p := l.SecondaryButton.Min; p.Y < l.SecondaryButton.Max.Y; p.Y++ {
		for x := p.X; x < l.SecondaryButton.Max.X; x++ {
			if disabled.RGBAAt(x, p.Y) != enabled.RGBAAt(x, p.Y) {
				different++
			}
		}
	}
	if different == 0 || !image.Pt(320, 320).In(l.SecondaryButton.Add(l.Box.Min)) {
		t.Fatal("disabled Load has no distinct visible paint")
	}
	a.HeadlessKey("enter")
	if a.Screen() != ScreenMenu {
		t.Fatal("Return did not activate Exit")
	}
}

func TestTerminalDefeat1091MissingFontCannotResumePlay(t *testing.T) {
	a, seam := failureApp1091(t, nil, nil)
	seam.v.SetFont(nil)
	before := seam.ticks
	a.HeadlessStep()
	a.HeadlessKey("f3")
	if seam.ticks != before || a.Screen() != ScreenMap || !seam.v.NoticeOpen() {
		t.Fatal("missing font removed the terminal gate")
	}
	a.HeadlessKey("escape")
	if a.Screen() != ScreenMenu {
		t.Fatal("missing-font terminal state has no exit")
	}
}
