package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type messageRecordSource struct {
	missionSource
	reads []string
}

func (s *messageRecordSource) ReadFile(name string) ([]byte, error) {
	s.reads = append(s.reads, name)
	return s.missionSource.ReadFile(name)
}

func messageRecordSAV(t *testing.T, f *FrontEnd, mw *mapWorld) string {
	t.Helper()
	raw, err := mw.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	s := Snapshot{Mission: 7, World: raw, Residue: mw.residue()}
	f.live = mw
	doc, err := f.currentMissionDocument(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := projectCurrentSession(&doc, s); err != nil {
		t.Fatal(err)
	}
	b, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

// This records the mission document producer even where the save-point policy
// refuses a player SAVE. No admission flag is changed to manufacture a save.
func TestMissionMessageDeliveryRecord(t *testing.T) {
	var lines []string
	for _, game := range []base.Game{base.GameROM1, base.GameROM2} {
		for _, scenario := range []struct {
			name    string
			events  []int32
			repeat  bool
			outcome int32
		}{
			{"repeated and discarded", []int32{4, 3, 4, 999}, true, 0},
			{"missing then readable", []int32{999, 3}, false, 0},
			{"specials", []int32{250, 253, 254, 255}, false, 0},
			{"reserved loss", []int32{255, 3}, false, 0},
			{"dialogues and victory", []int32{4, 3}, false, 4},
			{"victory", nil, false, 4},
			{"dialogue and defeat", []int32{3}, false, 5},
		} {
			var instants []sim.ScriptInstant
			var acts []int32
			var raises []mapload.ScriptRaise
			for _, e := range scenario.events {
				acts = append(acts, int32(len(instants)))
				instants = append(instants, sim.ScriptInstant{Op: 2, Args: [10]int32{e}})
				raises = append(raises, mapload.ScriptRaise{Latch: 3, Event: e})
			}
			if scenario.outcome != 0 {
				acts = append(acts, int32(len(instants)))
				instants = append(instants, sim.ScriptInstant{Op: scenario.outcome, Args: [10]int32{2}})
			}
			triggers := []sim.ScriptTrigger{missionTrigger(3, 1, !scenario.repeat, acts...)}
			build := missionWorld
			if game == base.GameROM2 {
				build = secondGameScriptWorld
			}
			w := build(t, instants, triggers)
			pages := missionSource{}
			var combined strings.Builder
			for _, e := range []int{3, 4, 250, 253, 254, 255} {
				body := fmt.Sprintf("<part=1>\r\ntext-key-%d\r\n", e)
				pages[missionEvent(t, 7, e)] = []byte(body)
				fmt.Fprintf(&combined, "#event%d\r\n%s", e, body)
			}
			pages["main/text/mission7.txt"] = []byte(combined.String())
			mw, v := missionDriverFor(t, w, raises, pages)
			front := currentPoolFixtureFront(t)
			front.Table = readUnder(game, front.Table)
			mw.mission.table = front.Table
			src := &messageRecordSource{missionSource: pages}
			mw.mission.src = src
			voice := &heldVoicePlayer{}
			bank := fixedUISoundBank{}
			for _, slot := range []ui.UISoundSlot{ui.UISoundMissionComplete, ui.UISoundMissionFailed} {
				bank[int(slot)] = audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(slot)}}
			}
			v.SetAudio(voice, bank)
			words := ui.AuthoredWords()
			words.NewMission = "new-mission-key"
			v.SetWords(words)
			lines = append(lines, fmt.Sprintf("%s %s input=%v repeat=%t outcome-op=%d", game, scenario.name, scenario.events, scenario.repeat, scenario.outcome))
			record := func(label string) {
				body, kind, shown := v.NoticeState()
				won, lost := w.ScriptCounters()
				bank, hasBank := w.ROM2ScenarioState()
				lines = append(lines, fmt.Sprintf("%s tick=%d notice=%q/%d/%t pending=%v reads=%v sounds=%v lines=%v outcome=%v/%t/%t reason=%d counters=%d/%d scenario=%t/%x world=%x sav=%s",
					label, w.Tick(), body, kind, shown, mw.mission.pendingMessages, src.reads, voice.slots, v.MessageLines(), mw.mission.outcome, mw.mission.announced, mw.mission.outcomeShown, mw.mission.lossReason,
					won, lost, hasBank, sha256.Sum256([]byte(fmt.Sprint(bank))), w.Hash(), messageRecordSAV(t, front, mw)))
			}
			record("before")
			for i := 1; i <= 34; i++ {
				mw.tick()
				if i <= 3 || i == 16 || i == 17 || i == 18 || i >= 32 {
					record("after-step")
				}
				if i == 2 || i == 3 || i == 17 || i == 18 || i == 33 || i == 34 {
					if mw.mission.open {
						place, text, err := mw.advanceNotice(ui.NoticeContinue)
						lines = append(lines, fmt.Sprintf("ack tick=%d place=%d text=%q err=%v", w.Tick(), place, text, err))
						record("after-ack")
					}
				}
			}
		}
	}
	want, err := os.ReadFile("testdata/missionmessages.txt")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lines, "\n") + "\n"
	if got != string(want) {
		t.Fatalf("mission message delivery changed:\n%s", got)
	}
}
