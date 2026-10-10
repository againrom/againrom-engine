package game

import (
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

func TestSchoolCaptionSourcesAndFallbacks(t *testing.T) {
	// Literal indices and neighbouring sentinels keep a production constant
	// typo, a table-local lookup and a one-based interpretation observable.
	src := installFixture{
		MainTextPath: textFile(234, map[int]string{
			230: "before", 231: string([]byte{0x93, 0xe7, 0xa8}),
			232: "school-leave", 233: "after",
		}),
		DialogsTextPath: textFile(234, map[int]string{231: "wrong-table", 232: "wrong-table"}),
	}
	w := LoadInstallWords(src, TextCode{}).Words()
	if w.SchoolTrain != string([]byte{0x93, 0xe7, 0xa8}) || w.SchoolExit != "school-leave" {
		t.Fatalf("school captions: %q / %q", w.SchoolTrain, w.SchoolExit)
	}
	for _, lines := range []int{0, 231, 232, 234} {
		w = LoadInstallWords(installFixture{MainTextPath: textFile(lines, map[int]string{231: "resolved-train"})}, TextCode{}).Words()
		want := ui.AuthoredWords()
		if lines > 231 {
			want.SchoolTrain = "resolved-train"
			want.Hover[231] = "resolved-train"
		}
		if w != want {
			t.Fatalf("partial table length %d did not resolve per field", lines)
		}
	}
}

func TestTavernCaptionSourcesAndFallbacks(t *testing.T) {
	src := installFixture{MainTextPath: textFile(261, map[int]string{
		231: "school-train", 232: "leave", 233: "after-exit",
		241: "before-talk", 242: "speak", 243: "constructor-only",
		257: "before-hire", 258: string([]byte{0x8d, 0xa0, 0xad}),
		259: "dismiss", 260: "after-fire",
	}), DialogsTextPath: textFile(261, map[int]string{242: "wrong-table", 258: "wrong-table", 259: "wrong-table"})}
	w := LoadInstallWords(src, TextCode{}).Words()
	if w.TavernHire != string([]byte{0x8d, 0xa0, 0xad}) || w.TavernFire != "dismiss" || w.TavernTalk != "speak" || w.TavernExit != "leave" {
		t.Fatalf("tavern caption resolution: %q / %q / %q / %q", w.TavernHire, w.TavernFire, w.TavernTalk, w.TavernExit)
	}
	for _, lines := range []int{0, 242, 243, 258, 259, 261} {
		w = LoadInstallWords(installFixture{MainTextPath: textFile(lines, map[int]string{258: "resolved-hire"})}, TextCode{}).Words()
		want := ui.AuthoredWords()
		if lines > 258 {
			want.TavernHire = "resolved-hire"
			want.Hover[258] = "resolved-hire"
		}
		if w != want {
			t.Fatalf("partial tavern table length %d did not resolve per field", lines)
		}
	}
}

func TestTavernSurfaceSelectsResolvedHireAndFireCaptions(t *testing.T) {
	f := shellFrontEnd()
	f.Words.TavernHire, f.Words.TavernFire = "installed-hire", "installed-fire"
	f.Words.TavernTalk, f.Words.TavernExit = "installed-talk", "installed-exit"
	s := f.townUI
	check := func(label string, enabled bool) {
		t.Helper()
		v := s.TownSurface()
		if v.Buttons[0].Label != "Sleep" || v.Buttons[1].Label != label || v.Buttons[1].Enabled != enabled ||
			v.Buttons[2].Label != "installed-talk" || v.Buttons[3].Label != "installed-exit" {
			t.Fatalf("tavern captions: %+v", v.Buttons)
		}
		if label == "" && v.Buttons[1].Value != "" {
			t.Fatal("unselected squad retained a price")
		}
	}
	check("installed-hire", true) // activation selects the first mercenary
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
	check("", false) // a talk-only candidate is not a squad
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	check("installed-hire", true)
	s.townSurfaceButton(tavernButtonHire)
	check("installed-fire", true)
	s.townSurfaceButton(tavernButtonHire)
	check("installed-hire", true)
}

func TestSchoolSurfaceUsesResolvedCaptionsWithoutChangingButtonValues(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	s.room = roomSchool
	before := s.TownSurface()
	gold := f.Town.Gold()
	f.Words.SchoolTrain = "installed-train"
	f.Words.SchoolExit = "installed-exit"
	after := s.TownSurface()
	if len(after.Buttons) != 2 || after.Buttons[0].Label != "installed-train" || after.Buttons[1].Label != "installed-exit" {
		t.Fatalf("production school captions: %+v", after.Buttons)
	}
	after.Buttons[0].Label, after.Buttons[1].Label = before.Buttons[0].Label, before.Buttons[1].Label
	if !reflect.DeepEqual(after.Buttons, before.Buttons) || f.Town.Gold() != gold {
		t.Fatal("caption resolution changed button values, enabled state or gold")
	}
}
