package ui

import (
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
)

// chrgenMembers are the sixteen chrgen members of sfx.res, spelled as their
// archive paths. The list is written out here, not read from production, so
// a wrong production path cannot agree with itself.
var chrgenMembers = []string{
	"chrgen/level1.wav", "chrgen/level2.wav", "chrgen/level3.wav",
	"chrgen/char.wav", "chrgen/ok.wav", "chrgen/+_-.wav",
	"chrgen/skill/fsword.wav", "chrgen/skill/faxe.wav", "chrgen/skill/fclub.wav", "chrgen/skill/fpike.wav", "chrgen/skill/fbow.wav",
	"chrgen/skill/mfire.wav", "chrgen/skill/mwater.wav", "chrgen/skill/mair.wav", "chrgen/skill/mearth.wav", "chrgen/skill/mastral.wav",
}

// memberTag is the one PCM value memberBank gives member i.
func memberTag(i int) int16 { return int16(1000 + i) }

// memberBank answers every chrgen member with a one-frame sample carrying the
// member's own tag, and no numbered slot.
type memberBank struct{}

func (memberBank) Sample(int) (audio.Sample, bool) { return audio.Sample{}, false }

func (memberBank) NamedSample(path string) (audio.Sample, bool) {
	for i, m := range chrgenMembers {
		if m == path {
			return audio.Sample{Rate: audio.DeviceRate, PCM: []int16{memberTag(i)}}, true
		}
	}
	return audio.Sample{}, false
}

// memberVoice is one instance the recorder started. It plays until a test
// ends it or something stops it.
type memberVoice struct {
	member  string
	playing bool
	stops   int
}

func (v *memberVoice) Playing() bool { return v.playing }
func (v *memberVoice) Stop()         { v.playing = false; v.stops++ }

// memberRecorder is an audio.VoicePlayer that makes no sound. It keeps every
// request in order with its placement and the member its sample was tagged
// with; one-shot slot sounds are only counted.
type memberRecorder struct {
	voices []*memberVoice
	places []audio.Placement
	plays  int
}

func (r *memberRecorder) Play(audio.Sample, audio.Placement) { r.plays++ }

func (r *memberRecorder) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	return r.StartVoice(s, request.Placement)
}

func (r *memberRecorder) StartVoice(s audio.Sample, p audio.Placement) audio.Voice {
	v := &memberVoice{member: "untagged", playing: true}
	if len(s.PCM) == 1 {
		for i := range chrgenMembers {
			if s.PCM[0] == memberTag(i) {
				v.member = chrgenMembers[i]
			}
		}
	}
	r.voices = append(r.voices, v)
	r.places = append(r.places, p)
	return v
}

func (r *memberRecorder) members() []string {
	out := []string{}
	for _, v := range r.voices {
		out = append(out, v.member)
	}
	return out
}

func soundTile(w, h int) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pic.SetRGBA(x, y, color.RGBA{R: 90, G: 60, B: 30, A: 255})
		}
	}
	return pic
}

// soundChargenSetup is a two-page generator whose pictures are small, opaque
// and apart, so each press lands on exactly one control. The name is not
// empty, so OK continues.
func soundChargenSetup() ChargenSetup {
	art := &ChargenPresentation{Layout: testGenerator(), Forward: soundTile(96, 74)}
	for choice := range art.Choices {
		for state := range art.Choices[choice] {
			art.Choices[choice][state] = soundTile(20, 20)
		}
	}
	for level := range art.Levels {
		for state := range art.Levels[level] {
			art.Levels[level][state] = soundTile(10, 10)
		}
	}
	for class := range art.Skills {
		for skill := range art.Skills[class] {
			for state := range art.Skills[class][skill] {
				art.Skills[class][skill][state] = soundTile(5, 5)
			}
		}
	}
	stats := make([]ChargenStat, 4)
	for i := range stats {
		stats[i] = ChargenStat{Name: "S", Floor: 15, Ceiling: 45, Start: 20}
	}
	return ChargenSetup{
		Name:      "Hero",
		PreCreate: &ChargenPreCreate{Art: art},
		Choices: []ChargenChoice{
			{Options: []string{"male", "female"}, Parent: -1},
			{Options: []string{"fighter", "mage"}, Parent: -1},
			{Options: []string{"blade", "axe", "bludgeon", "pike", "shooting"}, Parent: -1},
		},
		Stats:  stats,
		Cost:   triangular(50),
		Budget: 2000,
	}
}

// openSoundChargen opens soundChargenSetup's pre-create page over rec.
func openSoundChargen(t *testing.T, rec *memberRecorder) (*App, *Chargen) {
	t.Helper()
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetAudio(rec, memberBank{})
	c := NewChargen(soundChargenSetup())
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	return a, c
}

// soundClick is one left press and its release at p, a second after the
// previous input, so no two clicks pair into a double-click.
func soundClick(a *App, now *time.Time, p image.Point) {
	*now = now.Add(time.Second)
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true, Viewer: Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}}, *now)
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true, Viewer: Input{CursorX: p.X, CursorY: p.Y}}, *now)
}

// TestChargenPressesRequestTheirChrgenMembers is the owner's defect: in
// character generation a press on a difficulty button, a hero button, OK,
// the amulet Back, a statistic step and a skill cell each request their chrgen
// member (VIDEO-SFX-058, VIDEO-SFX-059), centred and one shot.
func TestChargenPressesRequestTheirChrgenMembers(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rec := &memberRecorder{}
	a, c := openSoundChargen(t, rec)
	soundClick(a, &now, preControlRect(c, chargenLevel2).Min)
	soundClick(a, &now, preControlRect(c, chargenChoice1).Min) // male mage
	soundClick(a, &now, preControlRect(c, chargenForward).Min)
	if c.Stage() != DetailedStage {
		t.Fatalf("OK left stage %v, want the detailed page", c.Stage())
	}
	soundClick(a, &now, detailedControlRect(c, chargenStatPlus0).Min)
	soundClick(a, &now, detailedControlRect(c, chargenSkill2).Min)

	back := &memberRecorder{}
	b, _ := openSoundChargen(t, back)
	soundClick(b, &now, image.Pt(600, 150)) // the amulet, clear of every hero
	if b.Screen() != ScreenMenu {
		t.Fatalf("amulet press left screen %v, want the menu", b.Screen())
	}

	want := []string{"chrgen/level3.wav", "chrgen/char.wav", "chrgen/ok.wav", "chrgen/+_-.wav", "chrgen/skill/mair.wav"}
	if got := rec.members(); !reflect.DeepEqual(got, want) {
		t.Errorf("generator presses requested %q, want %q", got, want)
	}
	if got := back.members(); !reflect.DeepEqual(got, []string{"chrgen/ok.wav"}) {
		t.Errorf("amulet press requested %q, want [chrgen/ok.wav]", got)
	}
	centred := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	for i, p := range append(rec.places, back.places...) {
		if p != centred {
			t.Errorf("request %d placement %+v, want centred %+v", i, p, centred)
		}
	}
	if rec.plays != 0 || back.plays != 0 {
		t.Errorf("chargen presses made %d and %d one-shot plays, want every member as a voice", rec.plays, back.plays)
	}
}
