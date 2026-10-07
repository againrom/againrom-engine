package game

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

func TestGateDialogueCapturedPagerAndVoiceReplacement(t *testing.T) {
	for _, key := range []string{"enter", "escape", "click"} {
		t.Run(key, func(t *testing.T) {
			f, voices := gateLineFront(t, true)
			f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/inn/mercenary/npc35.txt", Data: []byte("<part=1>\r\nFirst\r\n<part=2>\r\nSecond")}})
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "speech.res"), synth.Archive([]synth.File{
				{Path: "inn/mercenary/npc35p1.wav", Data: speechWAV(111)},
				{Path: "inn/mercenary/npc35p2.wav", Data: speechWAV(222)},
			}), 0o600); err != nil {
				t.Fatal(err)
			}
			f.SpeechBank = OpenSpeech(root)
			app, s := exteriorApp(t, f)
			t.Cleanup(app.StopAudio)
			pressGate(t, app, squareGatePoint(t, f))
			if len(voices.samples) != 1 || s.dialogue.text != "First" {
				t.Fatal("initial pager/voice", s.dialogue.text, len(voices.samples))
			}
			shows := s.TownDialogueShows()
			pressGate(t, app, squareGatePoint(t, f))
			if s.TownDialogueShows() != shows || s.said != 1 || len(voices.samples) != 1 {
				t.Fatal("captured outside press retried the gate")
			}
			dialogueKeyPress(t, app, s, "space")
			if s.said != 1 || voices.voices[0].stops != 0 {
				t.Fatal("Space paged or stopped the voice")
			}
			dialogueKeyPress(t, app, s, key)
			if s.dialogue.text != "Second" || len(voices.samples) != 2 || voices.voices[0].stops != 1 || voices.samples[1].PCM[0] != 222 {
				t.Fatal("pager replacement", s.dialogue.text, len(voices.samples), voices.voices[0].stops)
			}
			dialogueKeyPress(t, app, s, key)
			if gateDialogueOpen(s) || voices.voices[1].stops != 1 || !s.AtTownSquare() || app.Screen() != ui.ScreenTown || app.HeadlessMessage() != "" || len(f.Town.Available()) != 0 {
				t.Fatal("final pager stop/return")
			}
			pressGate(t, app, squareGatePoint(t, f))
			if s.said != 1 || len(voices.samples) != 3 || voices.samples[2].PCM[0] != 111 {
				t.Fatal("repeat gate did not show initial part")
			}
		})
	}
}

func TestGateDialogueGeometryAndBoundedMissingResourceReport(t *testing.T) {
	f, _ := gateLineFront(t, true)
	f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/inn/mercenary/npc35.txt", Data: []byte("<part=1;npc=35>\r\nGate")}})
	app, s := exteriorApp(t, f)
	t.Cleanup(app.StopAudio)
	pressGate(t, app, squareGatePoint(t, f))
	l, _, ok := s.townDialogueLayout()
	if !ok || l.Box != image.Rect(76, 124, 564, 356) || l.Text != image.Rect(128, 36, 428, 171) || l.Portrait != image.Rect(30, 54, 118, 168) || l.Button != image.Rect(200, 172, 280, 198) {
		t.Fatal("npc35 shared geometry", l)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	f.Archives = nil
	for i := 0; i < 2; i++ {
		pressGate(t, app, squareGatePoint(t, f))
		if gateDialogueOpen(s) || s.AtWorldMap() || !s.AtTownSquare() || app.HeadlessMessage() != "Town dialogue unavailable: "+townGateTextPath {
			t.Fatal("missing resource report/return", app.HeadlessMessage(), s.room)
		}
	}
}
