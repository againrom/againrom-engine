package ui

import (
	"fmt"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// noticeWords gives every notice slot a distinct line naming its slot, so a
// test reads which main.txt line was posted.
func noticeWords() Words {
	w := AuthoredWords()
	for i := range w.SettingNotice {
		w.SettingNotice[i] = fmt.Sprintf("n%d", i)
	}
	return w
}

func noticeTexts(v *Viewer) []string {
	var out []string
	for _, l := range v.MessageLines() {
		out = append(out, l.Text)
	}
	return out
}

// noticeMapApp is a map screen whose viewer and front end hold noticeWords.
// With seam true a settings seam is installed over a plain value array.
func noticeMapApp(t *testing.T, seam bool) (*App, *Viewer, *GameOptionValues) {
	t.Helper()
	a, s := noticeApp(t)
	w := noticeWords()
	a.flow.words = w
	s.v.SetWords(w)
	values := &GameOptionValues{}
	if seam {
		a.flow.gameOptions = GameOptionControls{
			Read: func(bool) GameOptionValues { return *values },
			Write: func(_ bool, o GameOption, value int) error {
				values[o] = value
				return nil
			},
		}
	}
	return a, s.v, values
}

// Each settings key posts the line of the setting's new state: base plus the
// state, with the setting's own base (MENU-057).
func TestSettingKeysPostTheNewStateLine(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		name   string
		in     appInput
		option GameOption
		base   int
		states int
	}{
		{"retreat", appInput{Retreat: true}, GameOptionRetreat, 94, 3},
		{"formation", appInput{Formation: true}, GameOptionFormation, 97, 3},
		{"show health", appInput{ShowHealth: true}, GameOptionHealth, 100, 2},
		{"flying damage", appInput{Numerals: true}, GameOptionDamage, 102, 2},
		{"day/night", appInput{TimeFlow: true}, GameOptionDayNight, 104, 2},
		{"smoothing", appInput{Smoothing: true}, GameOptionSmoothing, 106, 2},
		{"autohealing", appInput{AutoHealing: true}, GameOptionAutoHealing, 218, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v, values := noticeMapApp(t, true)
			for press := 1; press <= tc.states+1; press++ {
				before := len(v.MessageLines())
				a.step(tc.in, now)
				state := press % tc.states
				want := fmt.Sprintf("n%d", tc.base+state)
				got := noticeTexts(v)
				if len(got) != before+1 || got[len(got)-1] != want {
					t.Fatalf("press %d: lines %v, want a new last line %q", press, got, want)
				}
				if values[tc.option] != state {
					t.Fatalf("press %d: setting holds %d, want %d", press, values[tc.option], state)
				}
			}
			last := v.MessageLines()[len(v.MessageLines())-1]
			if last.Ink != MessageGrey || last.Life != 2000*time.Millisecond {
				t.Errorf("notice ink %v life %v, want grey and 2000 ms", last.Ink, last.Life)
			}
		})
	}
}

// Without a settings seam the viewer-backed toggles still post the new state.
func TestViewerBackedTogglesPostWithoutASeam(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v, _ := noticeMapApp(t, false)
	a.step(appInput{ShowHealth: true}, now)
	a.step(appInput{Numerals: true}, now)
	a.step(appInput{TimeFlow: true}, now)
	a.step(appInput{Smoothing: true}, now)
	damage, _ := v.DamageNumerals()
	want := []string{
		fmt.Sprintf("n%d", 100+noticeState(v.HealthBarsShown())),
		fmt.Sprintf("n%d", 102+noticeState(damage)),
		fmt.Sprintf("n%d", 104+noticeState(v.TimeFlow())),
		fmt.Sprintf("n%d", 106+noticeState(v.GraphicsOptions().Smoothing)),
	}
	got := noticeTexts(v)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("lines %v, want %v", got, want)
	}
}

// A toggle never drops a duplicate: a setting whose state does not move posts
// the same line twice (MENU-059).
func TestTogglePostKeepsDuplicates(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v, _ := noticeMapApp(t, true)
	a.flow.gameOptions.Write = func(bool, GameOption, int) error { return nil }
	a.step(appInput{ShowHealth: true}, now)
	a.step(appInput{ShowHealth: true}, now)
	if got := noticeTexts(v); fmt.Sprint(got) != "[n100 n100]" {
		t.Fatalf("lines %v, want two copies of n100", got)
	}
}

// The speed step posts 108 plus the clamped index, drops a post equal to the
// newest line, and needs no Ctrl (MENU-058).
func TestSpeedStepPostsAndDropsDuplicates(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v, _ := noticeMapApp(t, false)

	a.flow.rung = terrain.CadenceShippedLo + 4
	a.step(appInput{Faster: true}, now)
	if got := noticeTexts(v); fmt.Sprint(got) != "[n113]" {
		t.Fatalf("after one faster step lines %v, want [n113]", got)
	}
	a.step(appInput{Slower: true}, now)
	if got := noticeTexts(v); fmt.Sprint(got) != "[n113 n112]" {
		t.Fatalf("after slower lines %v, want [n113 n112]", got)
	}

	// At the fastest shipped speed a further press holds the same index.
	a.flow.rung = terrain.CadenceShippedHi
	a.step(appInput{Faster: true}, now)
	a.step(appInput{Faster: true}, now)
	got := noticeTexts(v)
	if n := len(got); n != 3 || got[n-1] != "n116" {
		t.Fatalf("lines %v, want exactly one new n116 line after two presses at the end", got)
	}

	// The slowest end clamps to index 0 whatever the extended rung is.
	a.flow.rung = terrain.CadenceRungMin + 1
	a.step(appInput{Slower: true}, now)
	got = noticeTexts(v)
	if got[len(got)-1] != "n108" {
		t.Fatalf("lines %v, want a final n108 below the shipped speeds", got)
	}

	// Ctrl held posts nothing.
	n := len(got)
	a.step(appInput{Faster: true, AttackHeld: true}, now)
	if len(noticeTexts(v)) != n {
		t.Errorf("a Ctrl-held speed press posted a line")
	}
}

// No post while a popup stands over the map: the map arm's input is held.
func TestNoticesAreHeldWhileAPopupStands(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v, _ := noticeMapApp(t, true)
	v.SetNotice("mission words", NoticeDialogue)
	if !a.flow.popupOpen() {
		t.Fatal("setup: no popup open")
	}
	a.step(appInput{Retreat: true, Formation: true, ShowHealth: true, Faster: true}, now)
	if got := noticeTexts(v); len(got) != 0 {
		t.Fatalf("lines %v posted under a popup", got)
	}
}

// An entry the install does not state posts nothing rather than an authored line.
func TestNoticeWithoutInstalledLineIsSilent(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v, _ := noticeMapApp(t, true)
	v.SetWords(AuthoredWords())
	a.step(appInput{Retreat: true, Faster: true}, now)
	if got := noticeTexts(v); len(got) != 0 {
		t.Fatalf("lines %v posted with no installed words", got)
	}
}

// A notice stands for 2000 ms and then goes (MENU-059).
func TestNoticeExpiresAfterTwoSeconds(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v, _ := noticeMapApp(t, true)
	a.step(appInput{Retreat: true}, now)
	if len(v.MessageLines()) != 1 {
		t.Fatal("setup: no notice")
	}
	v.stepMessages(now)
	v.stepMessages(now.Add(1900 * time.Millisecond))
	if len(v.MessageLines()) != 1 {
		t.Fatal("notice gone before its lifetime")
	}
	v.stepMessages(now.Add(2100 * time.Millisecond))
	if len(v.MessageLines()) != 0 {
		t.Fatal("notice still standing after its lifetime")
	}
}
