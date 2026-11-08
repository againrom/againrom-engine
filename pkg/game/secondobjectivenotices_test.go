package game

import (
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// heldVoice is a started voice whose playing the test sets.
type heldVoice struct{ playing bool }

func (v *heldVoice) Playing() bool { return v.playing }
func (v *heldVoice) Stop()         { v.playing = false }

// heldVoicePlayer records each slot it starts and holds the voice it
// reports, playing until the test stops it.
type heldVoicePlayer struct {
	slots  []int
	voices []*heldVoice
}

func (p *heldVoicePlayer) Play(audio.Sample, audio.Placement) {}

func (p *heldVoicePlayer) RequestSample(s audio.Sample, _ audio.Request) audio.Voice {
	p.slots = append(p.slots, int(s.PCM[0]))
	v := &heldVoice{playing: true}
	p.voices = append(p.voices, v)
	return v
}

func (p *heldVoicePlayer) stopAll() {
	for _, v := range p.voices {
		v.Stop()
	}
}

// The objective events against R2-ENGINE-891: 253 plays entry 0xe and 254
// entry 0x10, each only when that entry is idle; 250 plays 0xe when idle and
// posts patch.txt row 0x61 for 30 s; none queues a dialogue, opens a notice
// or posts another line. An ordinary message still queues.
func TestSecondGameObjectiveEventsPlayTheirEntryWhenIdle(t *testing.T) {
	p := &heldVoicePlayer{}
	bank := fixedUISoundBank{}
	for _, slot := range []ui.UISoundSlot{ui.UISoundMissionComplete, ui.UISoundMissionFailed} {
		bank[int(slot)] = audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(slot)}}
	}
	v := &ui.Viewer{}
	v.Layout(640, 480)
	v.SetAudio(p, bank)
	words := ui.AuthoredWords()
	words.NewMission = "the new-mission row"
	v.SetWords(words)
	mw := &mapWorld{view: v, mission: &missionNotices{table: readUnder(base.GameROM2, &mapload.Table{})}}
	step := func(events ...int32) {
		t.Helper()
		mw.observeScriptMessages(events)
	}
	played := func(want ...int) {
		t.Helper()
		if !reflect.DeepEqual(p.slots, want) {
			t.Fatalf("played slots %v, want %v", p.slots, want)
		}
	}

	step(253)
	played(14)
	step(253)
	played(14)
	step(254)
	played(14, 16)
	step(254, 250)
	played(14, 16)
	if lines := v.MessageLines(); len(lines) != 1 || lines[0].Text != "the new-mission row" || lines[0].Life != 30*time.Second {
		t.Fatalf("event 250 left the message line %+v", lines)
	}
	p.stopAll()
	step(250)
	played(14, 16, 14)
	step(254)
	played(14, 16, 14, 16)
	if lines := v.MessageLines(); len(lines) != 2 {
		t.Fatalf("the message line holds %d lines, want the two 250 posts", len(lines))
	}
	if len(mw.mission.pendingMessages) != 0 || mw.mission.open {
		t.Fatal("an objective event queued a dialogue or opened a notice")
	}
	step(7)
	if !reflect.DeepEqual(mw.mission.pendingMessages, []int32{7}) {
		t.Fatalf("an ordinary message queued %v", mw.mission.pendingMessages)
	}
}

// The new-mission line is patch.txt row 0x61 of the install's words; with
// no patch table it stays the authored English row.
func TestInstallWordsTakeTheNewMissionRow(t *testing.T) {
	rows := make([]byte, 0, 1024)
	for i := 0; i < 0x61; i++ {
		rows = append(rows, "row\r\n"...)
	}
	rows = append(rows, "installed new mission\r\nnext\r\n"...)
	if got := (&InstallWords{patch: SplitTextTable(rows)}).Words().NewMission; got != "installed new mission" {
		t.Fatalf("row 0x61 reads %q", got)
	}
	if got := (&InstallWords{}).Words().NewMission; got != ui.AuthoredWords().NewMission || got == "" {
		t.Fatalf("no patch table reads %q", got)
	}
}
