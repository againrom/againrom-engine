package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// selectionVoicePCM gives every fixture recording one distinct sample value, so
// a played sample names the file it came from.
var selectionVoicePCM = map[string]int16{
	"mf_hero/select1.wav": 101, "mf_hero/select2.wav": 102,
	"mf_hero/command1.wav": 111, "mf_hero/command2.wav": 112, "mf_hero/command3.wav": 113, "mf_hero/retreat.wav": 106,
	"f_mage/select1.wav": 201, "f_mage/select2.wav": 202,
	"f_mage/command1.wav": 211, "f_mage/command2.wav": 212, "f_mage/command3.wav": 213,
	"m_peasant/select1.wav": 301, "m_peasant/select2.wav": 302, "m_peasant/command1.wav": 311,
}

func selectionVoiceBank() *SoundBank {
	named := map[string]soundCacheEntry{}
	for name, pcm := range selectionVoicePCM {
		named[name] = soundCacheEntry{sample: audio.Sample{PCM: []int16{pcm}}, ok: true}
	}
	return &SoundBank{named: named}
}

// selectionVoiceMission opens fixture mission 10 through the ordinary App: one
// neutral unit (entity 0), a hired female mage (1) and the male fighter hero (2).
func selectionVoiceMission(t *testing.T) (*FrontEnd, *ui.App, *acknowledgmentRecorder, *scriptedDraw) {
	t.Helper()
	script := &scriptedDraw{}
	previous := newViewerVoiceDraw
	newViewerVoiceDraw = func() voiceDraw { return script.draw }
	t.Cleanup(func() { newViewerVoiceDraw = previous })
	dir := t.TempDir()
	raw := synth.Archive([]synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 128, Height: 128,
			Units: []synth.ALMUnit{{X: 57 << 8, Y: 100 << 8}}})},
		{Path: "npc.reg", Data: synth.NPCReg(nil)},
	})
	path := filepath.Join(dir, ScenarioArchive)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	containers, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	voices := &acknowledgmentRecorder{}
	f := &FrontEnd{
		InstallResources: InstallResources{Archives: &Archives{Containers: containers}, Tiles: &terrain.Tileset{}, SoundBank: selectionVoiceBank()},
		RuntimeServices:  RuntimeServices{SpeechPlayer: voices},
	}
	f.SetDeterministicFrames(true)
	party := MissionParty(nil, nil, nil)
	mage := party[0]
	mage.ID, mage.Name, mage.PlayerCharacter, mage.StartingHero, mage.MercenaryType, mage.Class = "mage", "Mage", false, false, 3, 0x18
	mage.FigureDir = string(data.FigureDirWomanMage)
	mage.Carry, mage.Carried, mage.CarriedItems = nil, nil, nil
	a := f.App("selection voice")
	if err := a.OpenMission(f.MissionOpenerWith(10, append([]mapload.PartyMember{mage}, party...))); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	for id, bank := range map[sim.EntityID]string{0: "", 1: "f_mage", 2: "mf_hero"} {
		e, ok := f.live.world.Entity(id)
		if !ok || f.live.voiceBank(e) != bank || (e.Owner == sim.SelfSlot) != (bank != "") {
			t.Fatalf("fixture entity %d: %+v bank %q", id, e, f.live.voiceBank(e))
		}
	}
	return f, a, voices, script
}

// heard is the fixture recordings requested since the mark.
func heard(voices *acknowledgmentRecorder, mark int) []int16 {
	var out []int16
	for _, s := range voices.samples[mark:] {
		out = append(out, s.PCM[0])
	}
	return out
}

func selectionVoiceClick(t *testing.T, a *ui.App, x, y int) {
	t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

// TestSelectingAnOwnCharacterSpeaksOneReply clicks the hero on the map: one
// request, mf_hero/select1.wav.
func TestSelectingAnOwnCharacterSpeaksOneReply(t *testing.T) {
	_, a, voices, _ := selectionVoiceMission(t)
	if err := a.HeadlessSelectEntity(2); err != nil {
		t.Fatal(err)
	}
	if got := heard(voices, 0); len(got) != 1 || got[0] != 101 {
		t.Fatalf("selecting the hero requested %v, want one mf_hero/select1.wav", got)
	}
}

// selectionVoiceScript drives one fixed sequence of map input and returns the
// recordings requested after each step.
func selectionVoiceScript(t *testing.T, f *FrontEnd, a *ui.App, voices *acknowledgmentRecorder, script *scriptedDraw) [][]int16 {
	t.Helper()
	var steps [][]int16
	record := func(draws []int, do func()) {
		mark := len(voices.samples)
		script.set(draws...)
		do()
		steps = append(steps, heard(voices, mark))
	}
	selectEntity := func(id uint32) func() {
		return func() {
			if err := a.HeadlessSelectEntity(id); err != nil {
				t.Fatal(err)
			}
		}
	}
	key := func(name string) func() {
		return func() {
			if err := a.HeadlessKey(name); err != nil {
				t.Fatal(err)
			}
		}
	}
	clock := func() {
		if err := headlessReplyClock(a); err != nil {
			t.Fatal(err)
		}
	}
	gx, gy, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	none := []int(nil)
	record(none, selectEntity(1))                              // 0: the mage alone
	record(none, key("e"))                                     // 1: E asks for no reply
	record(none, selectEntity(2))                              // 2: the hero alone
	record(none, selectEntity(2))                              // 3: the hero again at once
	record(none, selectEntity(0))                              // 4: the neutral unit
	record(none, clock)                                        // 5
	record([]int{0, 16384}, selectEntity(2))                   // 6: the hero after one response clock, second recording
	record(none, func() { selectionVoiceClick(t, a, gx, gy) }) // 7: an order inside the hero's clock
	record(none, clock)                                        // 8
	record(none, key("r"))                                     // 9: retreat after the clock
	record(none, clock)                                        // 10
	record(none, func() {                                      // 11: the option off
		if err := f.setAcknowledgments(false); err != nil {
			t.Fatal(err)
		}
		selectEntity(2)()
	})
	return steps
}

// TestSelectionRepliesThroughTheMapInput drives clicks, the E key and orders
// through the ordinary App: one speaker per gesture by priority, a cycle over
// select1 and select2, one response clock shared with command replies, silence
// for a foreign unit and with the option off. The same input with the option
// off from the start leaves the same World.
func TestSelectionRepliesThroughTheMapInput(t *testing.T) {
	f, a, voices, script := selectionVoiceMission(t)
	got := selectionVoiceScript(t, f, a, voices, script)
	want := [][]int16{{201}, nil, {101}, nil, nil, nil, {102}, nil, nil, {106}, nil, nil}
	if len(got) != len(want) {
		t.Fatalf("recorded %d steps, want %d", len(got), len(want))
	}
	for i := range want {
		if len(got[i]) != len(want[i]) || len(want[i]) == 1 && got[i][0] != want[i][0] {
			t.Errorf("step %d requested %v, want %v", i, got[i], want[i])
		}
	}

	silent, b, quiet, quietScript := selectionVoiceMission(t)
	if err := silent.setAcknowledgments(false); err != nil {
		t.Fatal(err)
	}
	selectionVoiceScript(t, silent, b, quiet, quietScript)
	if len(quiet.samples) != 0 {
		t.Fatalf("the option off requested %d recordings", len(quiet.samples))
	}
	if f.live.world.Tick() != silent.live.world.Tick() || f.live.world.Hash() != silent.live.world.Hash() {
		t.Fatalf("replies changed the World: tick %d hash %#x, silent tick %d hash %#x",
			f.live.world.Tick(), f.live.world.Hash(), silent.live.world.Tick(), silent.live.world.Hash())
	}
}
