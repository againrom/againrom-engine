package game

import (
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// releaseVoiceLeaves are the eleven recordings a human voice bank holds that
// the five readers play.
var releaseVoiceLeaves = []string{"select1", "select2", "command1", "command2", "command3", "retreat", "defend", "idle"}

// TestReleaseUnitRepliesPlayInstalledRecordings drives installed mission 20
// through the map input: every human bank's reader recordings decode to
// nonzero PCM; a click requests select1 or select2 by the draw, none inside the
// stamp, E none, the retreat key the hero-shaped speaker's retreat, a move
// order the command recording the draw names.
func TestReleaseUnitRepliesPlayInstalledRecordings(t *testing.T) {
	script := &scriptedDraw{}
	previous := newViewerVoiceDraw
	newViewerVoiceDraw = func() voiceDraw { return script.draw }
	t.Cleanup(func() { newViewerVoiceDraw = previous })
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.SoundBank = OpenSounds(f.Archives.Root)
	voices := &acknowledgmentRecorder{}
	f.SpeechPlayer = voices
	installed := map[string]audio.Sample{}
	for _, banks := range humanVoiceBanks {
		for _, bank := range banks {
			for _, leaf := range releaseVoiceLeaves {
				name := bank + "/" + leaf + ".wav"
				sample, ok := f.SoundBank.namedSample(name)
				if !ok || !slices.ContainsFunc(sample.PCM, func(v int16) bool { return v != 0 }) {
					t.Fatalf("%s absent or silent", name)
				}
				installed[name] = sample
			}
		}
	}
	a := f.App("unit reply witness")
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	for n := 0; a.HeadlessNoticeOpen() && n < 16; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if a.HeadlessNoticeOpen() {
		t.Fatal("notice did not close")
	}
	clock := func() {
		t.Helper()
		if err := headlessReplyClock(a); err != nil {
			t.Fatal(err)
		}
	}
	var speaker sim.Entity
	bank := ""
	var hero sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.Owner != sim.SelfSlot || e.HP <= 0 || f.live.voiceBank(e) == "" {
			continue
		}
		if data.FigureIsHero(e.TypeID) && hero.ID == 0 {
			hero = e
		}
		if bank != "" {
			continue
		}
		mark := len(voices.samples)
		script.set(0, 0)
		if err := a.HeadlessSelectEntity(uint32(e.ID)); err != nil {
			continue
		}
		speaker, bank = e, f.live.voiceBank(e)
		expectReply(t, "first click", voices, mark, installed, bank+"/select1.wav")
	}
	if bank == "" {
		t.Fatal("no own human could be clicked")
	}
	click := func(what, want string, draws ...int) {
		t.Helper()
		mark := len(voices.samples)
		script.set(draws...)
		if err := a.HeadlessSelectEntity(uint32(speaker.ID)); err != nil {
			t.Fatal(err)
		}
		expectReply(t, what, voices, mark, installed, want)
	}
	click("click inside the stamp", "", 0)
	clock()
	click("click after the stamp", bank+"/select2.wav", 0, 16384)
	clock()
	mark := len(voices.samples)
	script.set()
	if err := a.HeadlessKey("e"); err != nil {
		t.Fatal(err)
	}
	if len(voices.samples) != mark || script.used != 0 {
		t.Fatalf("E requested %d recordings after %d draws, want none", len(voices.samples)-mark, script.used)
	}
	clock()
	if hero.ID != 0 {
		mark = len(voices.samples)
		script.set(0)
		if err := a.HeadlessKey("r"); err != nil {
			t.Fatal(err)
		}
		expectReply(t, "retreat key", voices, mark, installed, f.live.voiceBank(hero)+"/retreat.wav")
	}
	clock()
	click("click before a move order", bank+"/select1.wav", 0, 0)
	clock()
	x, y, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	mark = len(voices.samples)
	script.set(0, 8192)
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	expectReply(t, "move order", voices, mark, installed, bank+"/command2.wav")
	t.Logf("unit replies: speaker %d bank %s; hero %d; requests=%d", speaker.ID, bank, hero.ID, len(voices.samples))
}

// expectReply checks the requests since mark: none when want is empty, else
// exactly the installed recording want names.
func expectReply(t *testing.T, what string, voices *acknowledgmentRecorder, mark int, installed map[string]audio.Sample, want string) {
	t.Helper()
	got := voices.samples[mark:]
	if want == "" {
		if len(got) != 0 {
			t.Fatalf("%s requested %d recordings, want none", what, len(got))
		}
		return
	}
	if len(got) != 1 || !slices.Equal(got[0].PCM, installed[want].PCM) {
		t.Fatalf("%s requested %d recordings, want one %s", what, len(got), want)
	}
}
