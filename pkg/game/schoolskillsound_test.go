package game

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// schoolSkillMembers are the ten chrgen members the school requests, fighter
// cells 0..4 then mage cells 5..9, in stored skill slot order. They are
// spelled here from VIDEO-SFX-059, not read from production.
var schoolSkillMembers = [10]string{
	"chrgen/skill/fsword.wav", "chrgen/skill/faxe.wav", "chrgen/skill/fclub.wav", "chrgen/skill/fpike.wav", "chrgen/skill/fbow.wav",
	"chrgen/skill/mfire.wav", "chrgen/skill/mwater.wav", "chrgen/skill/mair.wav", "chrgen/skill/mearth.wav", "chrgen/skill/mastral.wav",
}

// schoolMemberTag is the one PCM value member i's test sample carries.
func schoolMemberTag(i int) int16 { return int16(401 + i) }

// playingSchoolSounds is a school voice owner holding one playing chrgen
// instance, made through its own request.
func playingSchoolSounds() ui.SFXVoices {
	var s ui.SFXVoices
	bank := &SoundBank{named: map[string]soundCacheEntry{
		schoolSkillMembers[0]: {sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{schoolMemberTag(0)}}, ok: true},
	}}
	s.Request(&exteriorRecorder{}, bank, schoolSkillMembers[0])
	return s
}

// addSchoolSkillSounds attaches a voice recorder whose bank answers the ten
// school members with tagged samples.
func addSchoolSkillSounds(f *FrontEnd) *exteriorRecorder {
	rec := &exteriorRecorder{}
	f.SoundPlayer = rec
	for i, m := range schoolSkillMembers {
		f.SoundBank.named[m] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{schoolMemberTag(i)}}, ok: true}
	}
	return rec
}

// schoolRequested names each voice the recorder started by its member tag.
func schoolRequested(rec *exteriorRecorder) []string {
	out := []string{}
	for _, s := range rec.samples {
		name := "untagged"
		for i, m := range schoolSkillMembers {
			if len(s.PCM) == 1 && s.PCM[0] == schoolMemberTag(i) {
				name = m
			}
		}
		out = append(out, name)
	}
	return out
}

func schoolPress(s *townScreen, cell int) {
	s.TownSurfacePress(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell})
}

// TestSchoolSkillPressRequestsItsChrgenMember is VIDEO-SFX-059's school
// site: a press in an enabled skill column requests the member of the
// column's class and stored skill slot unless its instance plays. The press
// selects nothing; the selection stays with the click.
func TestSchoolSkillPressRequestsItsChrgenMember(t *testing.T) {
	f, s, now, _ := schoolTrainingFixture(t)
	finishSchoolEntryTransition(t, s, now)
	rec := addSchoolSkillSounds(f)
	centred := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}

	// The fighter shows cells 0..4; the mage cells are disabled.
	for cell := 0; cell < 10; cell++ {
		schoolPress(s, cell)
	}
	if got, want := schoolRequested(rec), schoolSkillMembers[:5]; !reflect.DeepEqual(got, want) {
		t.Fatalf("fighter presses requested %q, want %q", got, want)
	}
	if s.schoolCell != schoolNoSelection {
		t.Fatalf("a press selected cell %d; the click selects", s.schoolCell)
	}
	// Skip while playing, then request again once the sample has ended. The
	// recorder does not advance sample time, so the test ends it.
	schoolPress(s, 0)
	if len(rec.samples) != 5 {
		t.Fatalf("press on a playing member requested again: %q", schoolRequested(rec))
	}
	rec.voices[0].playing = false
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	schoolPress(s, 0) // on the selected column
	if got := schoolRequested(rec); len(got) != 6 || got[5] != schoolSkillMembers[0] || rec.voices[0].stops != 0 {
		t.Fatalf("ended member press requested %q with %d stops of the old instance", got, rec.voices[0].stops)
	}

	// Stepping to the mage turns the column; no skill press requests until
	// its panel shows, and then the mage cells request the mage members.
	for _, v := range rec.voices {
		v.playing = false
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	if !s.schoolTrainingBusy() {
		t.Fatal("stepping to the mage did not start the class transition")
	}
	for cell := 0; cell < 10; cell++ {
		schoolPress(s, cell)
	}
	if len(rec.samples) != 6 {
		t.Fatalf("presses during the class transition requested %q", schoolRequested(rec)[6:])
	}
	for i := 0; s.schoolTrainingBusy() && i < 64; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if s.schoolTrainingBusy() || s.schoolPanelClass() != schoolMageClass {
		t.Fatalf("mage transition did not settle: busy%v class%d", s.schoolTrainingBusy(), s.schoolPanelClass())
	}
	for cell := 0; cell < 10; cell++ {
		schoolPress(s, cell)
	}
	if got, want := schoolRequested(rec)[6:], schoolSkillMembers[5:]; !reflect.DeepEqual(got, want) {
		t.Fatalf("mage presses requested %q, want %q", got, want)
	}
	for i, p := range rec.places {
		if p != centred {
			t.Errorf("request %d placement %+v, want centred", i, p)
		}
	}

	// A new game stops the school's playing instances.
	playing := rec.voices[len(rec.voices)-1]
	s.resetForNewGame()
	if playing.playing || playing.stops != 1 {
		t.Fatalf("new game left the school member playing=%v stops=%d", playing.playing, playing.stops)
	}
}

// TestSchoolSkillPressThroughAppRequestsOnThePress drives one skill cell
// through App pointer input: the press requests the member, motion and the
// release request nothing, and the release keeps selecting the cell.
func TestSchoolSkillPressThroughAppRequestsOnThePress(t *testing.T) {
	f, s, now, _ := schoolTrainingFixture(t)
	f.TownSquareArt = resolved(exteriorTestArt(t), nil)
	f.Town.announceMission(f.Town.currentMain())
	f.TownAnimationRandom = func(int) int { return 0 }
	f.SoundBank.cache = map[int]soundCacheEntry{} // the load route's click sound caches a slot
	a, _ := exteriorApp(t, f)
	s.Back()
	s.Choose(2)
	if s.room != roomSchool {
		t.Fatalf("school entry = room%d", s.room)
	}
	s.CloseTip()
	finishSchoolEntryTransition(t, s, now)
	rec := addSchoolSkillSounds(f)

	view := s.TownSurface()
	at, found := image.Point{}, false
	for y := 0; y < 480 && !found; y++ {
		for x := 0; x < 640; x++ {
			if c, ok := ui.TownSurfaceControlAt(view, image.Pt(x, y)); ok && c.Kind == ui.TownSurfaceControlCell && c.Index == 2 {
				at, found = image.Pt(x, y), true
				break
			}
		}
	}
	if !found {
		t.Fatal("no point of the school view hits fighter cell 2")
	}
	if err := a.HeadlessPointer("press", at.X, at.Y); err != nil {
		t.Fatal(err)
	}
	if got := schoolRequested(rec); !reflect.DeepEqual(got, []string{schoolSkillMembers[2]}) {
		t.Fatalf("App press requested %q, want [%s]", got, schoolSkillMembers[2])
	}
	if s.schoolCell != 2 {
		t.Fatalf("App press selected cell %d, want 2 before release", s.schoolCell)
	}
	rec.voices[0].playing = false
	for _, action := range []string{"move", "release", "hover"} {
		point := at
		if action == "release" {
			point = image.Pt(2, 2)
		}
		if err := a.HeadlessPointer(action, point.X, point.Y); err != nil {
			t.Fatal(err)
		}
	}
	if len(rec.samples) != 1 {
		t.Fatalf("motion or release requested %q", schoolRequested(rec)[1:])
	}
	if s.schoolCell != 2 {
		t.Fatalf("outside release changed cell to %d, want retained 2", s.schoolCell)
	}
}
