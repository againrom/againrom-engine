package ui

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/render/menu"
)

// The request sites of VIDEO-SFX-058..060 on the pages this package owns:
// the pre-create and detailed generator pages, the main menu and the Hall of
// Fame. The school room's site is pkg/game's. Every row drives App input and
// reads the recorder; no row writes a result state.

// soundAt is one pointer edge at p, d after the previous input.
func soundAt(a *App, now *time.Time, d time.Duration, p image.Point, in appInput) {
	*now = now.Add(d)
	in.CursorX, in.CursorY = p.X, p.Y
	in.Viewer.CursorX, in.Viewer.CursorY = p.X, p.Y
	a.step(in, *now)
}

func soundPress(a *App, now *time.Time, d time.Duration, p image.Point) {
	soundAt(a, now, d, p, appInput{PrimaryPressed: true, Viewer: Input{PrimaryDown: true}})
}

func soundRelease(a *App, now *time.Time, p image.Point) {
	soundAt(a, now, 0, p, appInput{PrimaryReleased: true})
}

// soundKey is one keyboard edge.
func soundKey(a *App, now *time.Time, in appInput) {
	*now = now.Add(time.Second)
	a.step(in, *now)
}

// soundFocus moves the showing page's keyboard focus to focus with Down.
func soundFocus(t *testing.T, a *App, c *Chargen, now *time.Time, focus int) {
	t.Helper()
	for i := 0; c.Focus() != focus; i++ {
		if i == 32 {
			t.Fatalf("Down never reached focus %d", focus)
		}
		soundKey(a, now, appInput{Down: true})
	}
}

// endAll reports every started instance as ended. The recorder does not
// advance sample time, so a row that needs a sample to have finished ends it.
func (r *memberRecorder) endAll() {
	for _, v := range r.voices {
		v.playing = false
	}
}

// toDetailed presses a hero and OK on the pre-create page, then clears the
// recorder of both requests.
func toDetailed(t *testing.T, a *App, c *Chargen, rec *memberRecorder, now *time.Time, hero chargenControl) {
	t.Helper()
	soundClick(a, now, preControlRect(c, hero).Min)
	soundClick(a, now, preControlRect(c, chargenForward).Min)
	if c.Stage() != DetailedStage {
		t.Fatalf("OK left stage %v", c.Stage())
	}
	*rec = memberRecorder{}
}

func TestChargenSoundSites(t *testing.T) {
	seen := map[string]bool{}
	centred := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	check := func(t *testing.T, rec *memberRecorder, want []string) {
		t.Helper()
		got := rec.members()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("requested %q, want %q", got, want)
		}
		for i, p := range rec.places {
			if p != centred {
				t.Errorf("request %d placement %+v, want centred", i, p)
			}
		}
		if rec.plays != 0 {
			t.Errorf("%d one-shot plays, want every member as a voice", rec.plays)
		}
		for _, m := range got {
			seen[m] = true
		}
	}
	start := func(t *testing.T) (*App, *Chargen, *memberRecorder, *time.Time) {
		now := time.Unix(1_700_000_000, 0)
		rec := &memberRecorder{}
		a, c := openSoundChargen(t, rec)
		return a, c, rec, &now
	}

	t.Run("difficulty press restarts its level member, selected or not", func(t *testing.T) {
		a, c, rec, now := start(t)
		var want []string
		for _, id := range []chargenControl{chargenLevel0, chargenLevel1, chargenLevel2, chargenLevel2} {
			soundClick(a, now, preControlRect(c, id).Min)
			want = append(want, chrgenMembers[int(id-chargenLevel0)])
		}
		check(t, rec, want)
		if rec.voices[2].playing || rec.voices[2].stops != 1 || !rec.voices[3].playing {
			t.Fatalf("second level3 press did not stop and request again: %+v %+v", *rec.voices[2], *rec.voices[3])
		}
	})

	t.Run("hero press restarts char.wav", func(t *testing.T) {
		a, c, rec, now := start(t)
		for _, id := range []chargenControl{chargenChoice0, chargenChoice1, chargenChoice2, chargenChoice3} {
			soundClick(a, now, preControlRect(c, id).Min)
		}
		check(t, rec, []string{"chrgen/char.wav", "chrgen/char.wav", "chrgen/char.wav", "chrgen/char.wav"})
		for i, v := range rec.voices[:3] {
			if v.playing || v.stops != 1 {
				t.Fatalf("char.wav instance %d not stopped by the next press: %+v", i, *v)
			}
		}
	})

	t.Run("OK requests ok.wav and the close stops it", func(t *testing.T) {
		a, c, rec, now := start(t)
		soundClick(a, now, preControlRect(c, chargenLevel0).Min)
		soundClick(a, now, preControlRect(c, chargenForward).Min)
		check(t, rec, []string{"chrgen/level1.wav", "chrgen/ok.wav"})
		for _, v := range rec.voices {
			if v.playing {
				t.Fatalf("pre-create close left %s playing", v.member)
			}
		}
		if c.Stage() != DetailedStage {
			t.Fatalf("OK press left stage %v", c.Stage())
		}
	})

	t.Run("Enter on OK requests ok.wav; Enter elsewhere requests nothing", func(t *testing.T) {
		a, c, rec, now := start(t)
		soundFocus(t, a, c, now, 7) // easy
		soundKey(a, now, appInput{Enter: true})
		soundFocus(t, a, c, now, 2) // male mage
		soundKey(a, now, appInput{Enter: true})
		if c.Difficulty() != 1 || c.PreChoice() != 1 {
			t.Fatalf("Enter did not select: difficulty %d hero %d", c.Difficulty(), c.PreChoice())
		}
		soundFocus(t, a, c, now, 6) // OK
		soundKey(a, now, appInput{Enter: true})
		check(t, rec, []string{"chrgen/ok.wav"})
		if rec.voices[0].playing || c.Stage() != DetailedStage {
			t.Fatalf("Enter continue: playing %v stage %v", rec.voices[0].playing, c.Stage())
		}
	})

	t.Run("OK and Enter with an empty name request nothing", func(t *testing.T) {
		for _, key := range []bool{false, true} {
			a, c, rec, now := start(t)
			for i := 0; c.NameText() != ""; i++ {
				if i == 16 {
					t.Fatal("Backspace did not empty the name")
				}
				soundKey(a, now, appInput{Backspace: true})
			}
			if key {
				soundFocus(t, a, c, now, 6)
				soundKey(a, now, appInput{Enter: true})
			} else {
				soundClick(a, now, preControlRect(c, chargenForward).Min)
			}
			check(t, rec, []string{})
		}
	})

	t.Run("amulet Back and Escape request ok.wav and the close stops it", func(t *testing.T) {
		for _, escape := range []bool{false, true} {
			a, _, rec, now := start(t)
			if escape {
				soundKey(a, now, appInput{Escape: true})
			} else {
				soundClick(a, now, image.Pt(600, 150))
			}
			check(t, rec, []string{"chrgen/ok.wav"})
			if rec.voices[0].playing || rec.voices[0].stops != 1 || a.Screen() != ScreenMenu {
				t.Fatalf("escape=%v: %+v screen %v", escape, *rec.voices[0], a.Screen())
			}
		}
	})

	t.Run("pre-create open, motion, release, right button, name and double-click request nothing", func(t *testing.T) {
		a, c, rec, now := start(t)
		controls := []chargenControl{chargenLevel0, chargenLevel1, chargenLevel2, chargenChoice0, chargenChoice1,
			chargenChoice2, chargenChoice3, chargenForward, chargenName}
		for _, id := range controls {
			p := preControlRect(c, id).Min
			soundAt(a, now, time.Second, p, appInput{})
			soundAt(a, now, time.Second, p, appInput{PrimaryReleased: true})
			soundAt(a, now, time.Second, p, appInput{SecondaryPressed: true, Viewer: Input{SecondaryDown: true}})
			soundAt(a, now, time.Second, p, appInput{SecondaryReleased: true})
		}
		soundClick(a, now, preControlRect(c, chargenName).Min)
		check(t, rec, []string{})
		// A double-click's second click on a difficulty button requests
		// nothing; the press after it starts a new pair.
		p := preControlRect(c, chargenLevel0).Min
		soundPress(a, now, time.Second, p)
		soundRelease(a, now, p)
		rec.endAll()
		soundPress(a, now, 100*time.Millisecond, p)
		soundRelease(a, now, p)
		soundPress(a, now, 100*time.Millisecond, p)
		soundRelease(a, now, p)
		check(t, rec, []string{"chrgen/level1.wav", "chrgen/level1.wav"})
		// On a hero the second click continues, requesting nothing, and the
		// close stops the first click's char.wav.
		h := preControlRect(c, chargenChoice3).Min
		soundPress(a, now, time.Second, h)
		soundRelease(a, now, h)
		soundPress(a, now, 100*time.Millisecond, h)
		check(t, rec, []string{"chrgen/level1.wav", "chrgen/level1.wav", "chrgen/char.wav"})
		if c.Stage() != DetailedStage || rec.voices[2].playing {
			t.Fatalf("hero double-click: stage %v, char.wav playing %v", c.Stage(), rec.voices[2].playing)
		}
	})

	t.Run("statistic steps restart +_-.wav; a refused step is silent", func(t *testing.T) {
		a, c, rec, now := start(t)
		toDetailed(t, a, c, rec, now, chargenChoice0)
		var want []string
		for stat := 0; stat < 4; stat++ {
			for _, id := range []chargenControl{chargenStatPlus0 + chargenControl(stat), chargenStatMinus0 + chargenControl(stat)} {
				soundClick(a, now, detailedControlRect(c, id).Min)
				want = append(want, "chrgen/+_-.wav")
			}
		}
		check(t, rec, want)
		for i, v := range rec.voices[:len(rec.voices)-1] {
			if v.playing || v.stops != 1 {
				t.Fatalf("step %d instance not restarted by the next step: %+v", i, *v)
			}
		}
		// Statistic 0 sits at its start value 20 over a floor of 15.
		minus := detailedControlRect(c, chargenStatMinus0).Min
		for i := 0; i < 6; i++ {
			soundClick(a, now, minus)
		}
		if got := len(rec.voices) - len(want); got != 5 || c.statValue[0] != 15 {
			t.Fatalf("six decreases from 20 made %d requests, value %d; want 5 and 15", got, c.statValue[0])
		}
	})

	t.Run("statistic double-click second click steps and requests", func(t *testing.T) {
		a, c, rec, now := start(t)
		toDetailed(t, a, c, rec, now, chargenChoice0)
		p := detailedControlRect(c, chargenStatPlus1).Min
		before := c.statValue[1]
		soundPress(a, now, time.Second, p)
		soundRelease(a, now, p)
		soundPress(a, now, 100*time.Millisecond, p)
		soundRelease(a, now, p)
		check(t, rec, []string{"chrgen/+_-.wav", "chrgen/+_-.wav"})
		if c.statValue[1] != before+2 {
			t.Fatalf("double-click stepped %d -> %d, want two steps", before, c.statValue[1])
		}
	})

	for class, hero := range []chargenControl{chargenChoice0, chargenChoice1} {
		members := chrgenMembers[6+5*class : 11+5*class]
		t.Run("skill press requests its class member unless it plays: "+members[0], func(t *testing.T) {
			a, c, rec, now := start(t)
			toDetailed(t, a, c, rec, now, hero)
			for skill := 0; skill < 5; skill++ {
				soundClick(a, now, detailedControlRect(c, chargenSkill0+chargenControl(skill)).Min)
				if c.choiceIndex[2] != skill {
					t.Fatalf("skill press %d selected %d", skill, c.choiceIndex[2])
				}
			}
			s0 := detailedControlRect(c, chargenSkill0).Min
			soundClick(a, now, s0) // plays: skipped
			check(t, rec, members)
			rec.endAll()
			soundClick(a, now, s0) // ended: requested again, no comparison with the selection
			check(t, rec, append(append([]string{}, members...), members[0]))
		})
	}

	t.Run("skill double-click second click requests nothing", func(t *testing.T) {
		a, c, rec, now := start(t)
		toDetailed(t, a, c, rec, now, chargenChoice0)
		p := detailedControlRect(c, chargenSkill3).Min
		soundPress(a, now, time.Second, p)
		soundRelease(a, now, p)
		rec.endAll()
		soundPress(a, now, 100*time.Millisecond, p)
		soundRelease(a, now, p)
		check(t, rec, []string{"chrgen/skill/fpike.wav"})
		soundPress(a, now, 100*time.Millisecond, p) // a new pair's first click
		soundRelease(a, now, p)
		check(t, rec, []string{"chrgen/skill/fpike.wav", "chrgen/skill/fpike.wav"})
	})

	t.Run("detailed open, motion, release, right button, keyboard and navigation request nothing", func(t *testing.T) {
		a, c, rec, now := start(t)
		toDetailed(t, a, c, rec, now, chargenChoice1)
		ids := []chargenControl{chargenSkill0, chargenSkill4, chargenStatMinus2, chargenStatPlus3}
		for _, id := range ids {
			p := detailedControlRect(c, id).Min
			soundAt(a, now, time.Second, p, appInput{})
			soundAt(a, now, time.Second, p, appInput{PrimaryReleased: true})
			soundAt(a, now, time.Second, p, appInput{SecondaryPressed: true, Viewer: Input{SecondaryDown: true}})
			soundAt(a, now, time.Second, p, appInput{SecondaryReleased: true})
		}
		before := c.statValue[2]
		soundFocus(t, a, c, now, 10) // statistic 2 up
		soundKey(a, now, appInput{Enter: true})
		soundFocus(t, a, c, now, 3) // skill 3
		soundKey(a, now, appInput{Enter: true})
		if c.statValue[2] != before+1 || c.choiceIndex[2] != 3 {
			t.Fatalf("keyboard: statistic %d -> %d, skill %d", before, c.statValue[2], c.choiceIndex[2])
		}
		soundClick(a, now, detailedControlRect(c, chargenReset).Min)
		soundClick(a, now, detailedControlRect(c, chargenBack).Min)
		if c.Stage() != PreCreateStage {
			t.Fatalf("detailed Back left stage %v", c.Stage())
		}
		check(t, rec, []string{})
	})

	t.Run("main menu buttons request the menu's ok.wav unless it plays", func(t *testing.T) {
		a := newTestApp(t, appRows(1), okLoader(t))
		rec := &memberRecorder{}
		a.SetAudio(rec, memberBank{})
		a.SetInterfaceGenerator(testLayout)
		now := time.Unix(1_700_000_000, 0)
		off := image.Pt(2, 2)
		if a.buttonAt(off.X, off.Y) != 0 {
			t.Fatal("the release point lies on a button")
		}
		var want []string
		for b := 1; b <= menu.ButtonCount; b++ {
			x, y := centreOf(b)
			soundAt(a, &now, time.Second, image.Pt(x, y), appInput{})
			soundPress(a, &now, time.Second, image.Pt(x, y))
			soundRelease(a, &now, off) // released off the button: nothing activates
			want = append(want, "chrgen/ok.wav")
			check(t, rec, want)
			rec.endAll()
		}
		x, y := centreOf(1)
		soundPress(a, &now, time.Second, image.Pt(x, y))
		soundRelease(a, &now, off)
		x, y = centreOf(2)
		soundPress(a, &now, time.Second, image.Pt(x, y)) // the first still plays
		soundRelease(a, &now, off)
		soundPress(a, &now, time.Second, off)
		check(t, rec, append(want, "chrgen/ok.wav"))
	})

	t.Run("Hall of Fame OK requests the hall's ok.wav unless it plays", func(t *testing.T) {
		a := newTestApp(t, appRows(1), okLoader(t))
		rec := &memberRecorder{}
		a.SetAudio(rec, memberBank{})
		a.SetInterfaceGenerator(testLayout)
		now := time.Unix(1_700_000_000, 0)
		a.SetHallOfFame(func() EndingView { return EndingView{} })
		a.flow.showHallOfFame()
		if a.Screen() != ScreenEnding || a.flow.endingPage != 2 {
			t.Fatalf("hall did not open: screen %v page %d", a.Screen(), a.flow.endingPage)
		}
		ok := a.flow.endingButtonRect(0).Min.Add(image.Pt(4, 4))
		off := image.Pt(320, 200)
		soundAt(a, &now, time.Second, ok, appInput{})
		soundPress(a, &now, time.Second, ok)
		soundRelease(a, &now, off)
		soundPress(a, &now, time.Second, ok) // still plays
		soundRelease(a, &now, off)
		soundPress(a, &now, time.Second, off)
		check(t, rec, []string{"chrgen/ok.wav"})
		rec.endAll()
		soundPress(a, &now, time.Second, ok)
		soundRelease(a, &now, ok)
		check(t, rec, []string{"chrgen/ok.wav", "chrgen/ok.wav"})
		if a.Screen() != ScreenMenu {
			t.Fatalf("hall OK release left screen %v, want the menu", a.Screen())
		}
	})

	for _, m := range chrgenMembers {
		if !seen[m] {
			t.Errorf("no site requested %s", m)
		}
	}
}

// TestChargenHeldStatisticRepeatsInTicks holds the left button on a
// statistic control. The first repeat comes on the 10th App tick after the
// last mouse message and each later one on the 4th tick after the previous
// repeat (DIV-1493, from VIDEO-SFX-059's 150 ms and 66 ms at 60 ticks per
// second); motion restores the first delay and the release ends the repeat.
func TestChargenHeldStatisticRepeatsInTicks(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rec := &memberRecorder{}
	a, c := openSoundChargen(t, rec)
	toDetailed(t, a, c, rec, &now, chargenChoice0)
	p := detailedControlRect(c, chargenStatPlus2).Min
	hold := func(at image.Point) {
		now = now.Add(time.Second / 60)
		a.step(appInput{CursorX: at.X, CursorY: at.Y, Viewer: Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}}, now)
	}
	requestedOn := func(ticks int, at image.Point) []int {
		var out []int
		for tick := 1; tick <= ticks; tick++ {
			n := len(rec.voices)
			hold(at)
			if len(rec.voices) > n {
				out = append(out, tick)
			}
		}
		return out
	}
	before := c.statValue[2]
	soundPress(a, &now, time.Second, p)
	if got := requestedOn(22, p); !reflect.DeepEqual(got, []int{10, 14, 18, 22}) {
		t.Fatalf("held repeats on ticks %v, want [10 14 18 22]", got)
	}
	// Motion inside the control is a mouse message: the delay starts again.
	moved := p.Add(image.Pt(1, 0))
	if detailedControlAt(c, moved) != chargenStatPlus2 {
		t.Fatal("the moved point left the control")
	}
	hold(moved)
	if got := requestedOn(14, moved); !reflect.DeepEqual(got, []int{10, 14}) {
		t.Fatalf("after motion repeats on ticks %v, want [10 14]", got)
	}
	if len(rec.voices) != 7 || c.statValue[2] != before+7 {
		t.Fatalf("press and six repeats: %d requests, value %d -> %d", len(rec.voices), before, c.statValue[2])
	}
	for _, v := range rec.voices[:6] {
		if v.playing || v.stops != 1 {
			t.Fatalf("a repeat did not restart the playing +_-.wav: %+v", *v)
		}
	}
	soundRelease(a, &now, moved)
	for i := 0; i < 20; i++ {
		now = now.Add(time.Second / 60)
		a.step(appInput{CursorX: moved.X, CursorY: moved.Y, Viewer: Input{CursorX: moved.X, CursorY: moved.Y}}, now)
	}
	if len(rec.voices) != 7 {
		t.Fatalf("released button repeated: %d requests", len(rec.voices))
	}

	// A held skill press and a held pre-create press never repeat.
	s := detailedControlRect(c, chargenSkill1).Min
	soundPress(a, &now, time.Second, s)
	rec.endAll()
	if got := requestedOn(30, s); got != nil {
		t.Fatalf("held skill repeated on ticks %v", got)
	}
	soundRelease(a, &now, s)
	soundClick(a, &now, detailedControlRect(c, chargenBack).Min)
	l := preControlRect(c, chargenLevel2).Min
	soundPress(a, &now, time.Second, l)
	rec.endAll()
	if got := requestedOn(30, l); got != nil {
		t.Fatalf("held difficulty press repeated on ticks %v", got)
	}
}
