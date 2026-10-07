package game

import (
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// Each settings key posts the installed main.txt line of the setting's new
// state on the map message line, and the speed step posts line 108 plus the
// speed index. The expected text is read from the install's own main.txt.
func TestReleaseSettingKeysPostInstalledStateLines(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.LoadOptions()
	raw, err := f.Archives.Containers.ReadFile(MainTextPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\r\n")
	if len(lines) < 221 {
		t.Fatalf("main.txt holds %d lines, want the setting lines to 220", len(lines))
	}
	for _, slots := range [][2]int{{94, 116}, {218, 220}} {
		for i := slots[0]; i <= slots[1]; i++ {
			if lines[i] == "" || f.Words.SettingNotice[i] != lines[i] {
				t.Fatalf("SettingNotice[%d] = %q, want installed main.txt[%d] %q", i, f.Words.SettingNotice[i], i, lines[i])
			}
		}
	}

	a := f.App("setting notices")
	a.SetCutscenes(nil)
	party := f.ChargenParty(ui.ChargenResult{Name: "Notice witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	view := f.live.view
	// A mission's opening notice holds the map arm's input until dismissed.
	dismiss := func() {
		t.Helper()
		for i := 0; i < 8 && view.NoticeOpen(); i++ {
			if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		if view.NoticeOpen() {
			t.Fatal("a notice stays open over the map")
		}
	}
	// A notice the advance raises on the pressing frame holds that frame's
	// input, so a press that met one is repeated once it is dismissed.
	tapKey := func(key string) {
		t.Helper()
		for try := 0; try < 4; try++ {
			dismiss()
			if err := a.HeadlessKey(key); err != nil {
				t.Fatal(err)
			}
			if !view.NoticeOpen() {
				return
			}
		}
		t.Fatalf("key %s never met an open map", key)
	}

	for _, tc := range []struct {
		key    string
		option ui.GameOption
		base   int
	}{
		{"ctrl-w", ui.GameOptionRetreat, 94},
		{"ctrl-f", ui.GameOptionFormation, 97},
		{"ctrl-h", ui.GameOptionHealth, 100},
		{"ctrl-l", ui.GameOptionDamage, 102},
		{"ctrl-n", ui.GameOptionDayNight, 104},
		{"ctrl-o", ui.GameOptionSmoothing, 106},
		{"ctrl-u", ui.GameOptionAutoHealing, 218},
	} {
		t.Run(tc.key, func(t *testing.T) {
			for press := 0; press < 3; press++ {
				tapKey(tc.key)
				state := f.gameOptionValues(true)[tc.option]
				got := view.MessageLines()
				if len(got) == 0 {
					t.Fatalf("%s press %d: no line posted", tc.key, press)
				}
				if last := got[len(got)-1]; last.Text != lines[tc.base+state] {
					t.Fatalf("%s press %d: posted %q for state %d, want main.txt[%d] %q",
						tc.key, press, last.Text, state, tc.base+state, lines[tc.base+state])
				}
			}
		})
	}

	// The speed step: index 4 is the default, so one press up posts index 5.
	tapKey("numpad-plus")
	got := view.MessageLines()
	if last := got[len(got)-1]; last.Text != lines[108+5] {
		t.Fatalf("speed up posted %q, want main.txt[113] %q", last.Text, lines[113])
	}
}
