package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// gateLineNodes holds the raw main.res node of the gate line, by its size, in
// each installed root (TOWN-475): EN 223 bytes, RU 233 bytes. gateLineVoices
// holds the size of its recording in speech.res under the same key.
var gateLineNodes = map[int]string{
	223: "660b51c0ff6c4e79b58724a67d424cf5a21451c4dca45a22a907a37a270a5284",
	233: "f7ca39b3ceb910b262bbad00854e77342988ea6379252dc657a9db14e7e3c06a",
}

var gateLineVoices = map[int]int{223: 630792, 233: 634548}

// TestReleaseGatePressFollowsWhatIsOnOffer presses the gate door of the
// installed square through App input. With nothing on offer the town view stays
// and the gate line opens over it with its recording, and nothing is posted
// (TOWN-475); once a mission is accepted through the tavern the same press
// enters the world map and the line stays silent. Every audio device is a
// recording double and the master volume is 0, so the witness reads requests
// and nothing is heard.
func TestReleaseGatePressFollowsWhatIsOnOffer(t *testing.T) {
	f := releaseFront(t)
	root := f.Archives.Root
	f.Sound = SoundOptions{Enabled: true, Volume: 0}
	voices := &exteriorRecorder{}
	f.SpeechPlayer = voices
	now, rolls := time.Unix(100, 0), 0
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(n int) int { rolls++; return 0 }
	app, s := roomExitApp(t, f, 0)
	t.Cleanup(app.StopAudio)

	// Installed bytes and recordings are identified independently by size/hash.
	mainPath, err := cutsceneInstallPath(root, "main.res")
	if err != nil {
		t.Fatal(err)
	}
	mainRes, err := OpenContainers(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := mainRes.ReadFile("main/text/inn/mercenary/npc35.txt")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if want, ok := gateLineNodes[len(raw)]; !ok || hex.EncodeToString(sum[:]) != want {
		t.Fatalf("the gate line node is %d bytes, sha256 %x; want one of the claim's two roots", len(raw), sum)
	}
	speechPath, err := cutsceneInstallPath(root, "speech.res")
	if err != nil {
		t.Fatal(err)
	}
	speechRes, err := vfs.OpenFileBacked([]string{speechPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	wav, err := speechRes.ReadFile("speech/inn/mercenary/npc35p1.wav")
	if err != nil || len(wav) != gateLineVoices[len(raw)] {
		t.Fatalf("the gate line recording is %d bytes (%v), want %d", len(wav), err, gateLineVoices[len(raw)])
	}
	recording, err := audio.DecodeWAV(wav, audio.DeviceRate)
	if err != nil || len(recording.PCM) == 0 {
		t.Fatalf("the gate line recording does not decode: %v", err)
	}
	headerLF := bytes.IndexByte(raw, '\n')
	if headerLF < 0 || bytes.IndexByte(raw[:headerLF], '>') < 0 || bytes.IndexByte(raw[headerLF+1:], '<') >= 0 {
		t.Fatal("the identified gate node is not one header-LF-delimited part")
	}
	lines := []string{string(raw[headerLF+1:])}
	if lines[0] == "" {
		t.Fatal("the identified gate node has an empty body")
	}

	if app.Screen() != ui.ScreenTown || !s.AtTownSquare() || len(f.Town.Available()) != 0 || len(voices.samples) != 0 {
		t.Fatalf("setup: screen %v, square %v, on offer %v, recordings %d; want the square with nothing on offer and no recording",
			app.Screen(), s.AtTownSquare(), f.Town.Available(), len(voices.samples))
	}
	s.CloseTip()
	gate := squareGatePoint(t, f)

	closeLine := func(step, key string) {
		t.Helper()
		for n := 0; gateDialogueOpen(s) && n <= len(lines); n++ {
			var err error
			if key == "click" {
				err = app.HeadlessActivate("dialogue")
			} else {
				err = app.HeadlessKey(key)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if gateDialogueOpen(s) || !s.AtTownSquare() || s.AtWorldMap() || app.Screen() != ui.ScreenTown {
			t.Fatalf("%s left the line open %v, square %v, map %v, screen %v", step, gateDialogueOpen(s), s.AtTownSquare(), s.AtWorldMap(), app.Screen())
		}
		if got := app.HeadlessMessage(); got != "" {
			t.Errorf("%s posted %q, want nothing", step, got)
		}
	}

	// Nothing on offer: the town view stays, the line and its recording open.
	for press, key := range []string{"enter", "escape", "click"} {
		pressGate(t, app, gate)
		if app.Screen() != ui.ScreenTown || s.AtWorldMap() || !s.AtTownSquare() || !gateDialogueOpen(s) {
			t.Fatalf("press %d with nothing on offer: screen %v, map %v, square %v, line open %v",
				press+1, app.Screen(), s.AtWorldMap(), s.AtTownSquare(), gateDialogueOpen(s))
		}
		if path, ok := s.townTextPath(); !ok || path != "main/text/inn/mercenary/npc35.txt" {
			t.Fatalf("press %d opened %q", press+1, path)
		}
		if got := s.townLines(); !reflect.DeepEqual(got, lines) {
			t.Fatalf("press %d shows %q, want the node's own %q", press+1, got, lines)
		}
		if got := app.HeadlessMessage(); got != "" {
			t.Errorf("press %d posted %q, want nothing", press+1, got)
		}
		if len(voices.samples) != press+1 || !reflect.DeepEqual(voices.samples[press], recording) {
			t.Fatalf("press %d requested %d recordings, want %d and the last one the install's npc35p1.wav", press+1, len(voices.samples), press+1)
		}
		if want := (audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}); voices.places[press] != want {
			t.Errorf("press %d placed the recording at %+v, want centred %+v", press+1, voices.places[press], want)
		}
		beforeFrame, beforeClock, beforeRolls := s.exterior.frame, s.townPaintLast, rolls
		now = now.Add(68 * time.Millisecond)
		if pix, _, err := app.HeadlessFrame(); err != nil || pix == nil {
			t.Fatalf("press %d: the town with the line open does not compose: %v", press+1, err)
		}
		if s.exterior.frame != beforeFrame || s.townPaintLast != beforeClock || rolls != beforeRolls || voices.voices[press].stops != 0 {
			t.Fatalf("shown npc35: frame %+v=>%+v clock %v=>%v rolls %d=>%d voice stops=%d", beforeFrame, s.exterior.frame, beforeClock, s.townPaintLast, beforeRolls, rolls, voices.voices[press].stops)
		}
		if pic, open := s.TownDialogue(); !open || pic == nil {
			t.Fatalf("press %d: the line has no window", press+1)
		}
		checkInstalledDialoguePointer(t, app, "gate line", image.Rect(276, 296, 356, 322), image.Pt(640, 480), func() *image.RGBA {
			t.Helper()
			pix, _, err := app.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			return pix
		})
		if s.exterior.frame != beforeFrame || s.townPaintLast != beforeClock || voices.voices[press].stops != 0 {
			t.Fatalf("captured pointer controls: frame %+v=>%+v clock %v=>%v voice stops=%d", beforeFrame, s.exterior.frame, beforeClock, s.townPaintLast, voices.voices[press].stops)
		}
		closeLine(fmt.Sprintf("press %d's line, ended by %s", press+1, key), key)
		if voices.voices[press].stops != 1 {
			t.Errorf("press %d: ending the line stopped its recording %d times, want 1", press+1, voices.voices[press].stops)
		}
		if s.townPaintLast != beforeClock {
			t.Fatalf("pager close changed the blocked timestamp: %v=>%v, now=%v", beforeClock, s.townPaintLast, now)
		}
		beforeComposeRolls := rolls
		if pix, _, err := app.HeadlessFrame(); err != nil || pix == nil {
			t.Fatal("post-pager square composition", err)
		}
		if rolls != beforeComposeRolls+2 || s.townPaintLast != now {
			t.Fatalf("post-pager pure frame: rolls %d=>%d want +2; clock %v=>%v want %v", beforeComposeRolls, rolls, beforeClock, s.townPaintLast, now)
		}
		if pix, _, err := app.HeadlessFrame(); err != nil || pix == nil {
			t.Fatal("same-clock square composition", err)
		}
		if rolls != beforeComposeRolls+2 || s.townPaintLast != now {
			t.Fatalf("same-clock pure frame: rolls=%d want %d; clock=%v want %v", rolls, beforeComposeRolls+2, s.townPaintLast, now)
		}
		if got := f.Town.Available(); len(got) != 0 {
			t.Fatalf("press %d made %v available", press+1, got)
		}
	}
	installed := f.Archives.Containers
	empty, err := vfs.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = empty
	if _, err := empty.ReadFile(townGateTextPath); err == nil {
		t.Fatal("missing-resource loss control still supplies npc35")
	}
	beforeVoices := len(voices.samples)
	for i := 0; i < 2; i++ {
		pressGate(t, app, gate)
		if gateDialogueOpen(s) || s.AtWorldMap() || !s.AtTownSquare() || len(voices.samples) != beforeVoices || app.HeadlessMessage() != "Town dialogue unavailable: "+townGateTextPath {
			t.Fatal("installed square missing-resource report created a substitute panel or voice")
		}
	}
	f.Archives.Containers = installed

	// A mission a tavern conversation hands over is on offer once the player has
	// left the tavern, and the same press enters the world map without a line or a
	// recording.
	mission, npc := 0, 0
	for _, offer := range f.Town.Offers(TownTavern) {
		if offer.Mission > 0 {
			mission, npc = offer.Mission, offer.NPC
			break
		}
	}
	if mission == 0 {
		t.Fatal("the first town's tavern holds no mission to accept")
	}
	roomExitEnter(t, app, s, "TAVERN", roomTavern)
	if err := app.HeadlessActivate(fmt.Sprintf("NPC %d", npc)); err != nil {
		t.Fatal(err)
	}
	roomExitReader{t, app}.finish(s, fmt.Sprintf("NPC %d's conversation", npc), false)
	if slices.Contains(f.Town.Available(), mission) {
		t.Fatalf("ending NPC %d's conversation put mission %d at the gates inside the tavern", npc, mission)
	}
	if err := app.HeadlessActivate(s.TownSurface().Buttons[tavernButtonExit].Label); err != nil {
		t.Fatal(err)
	}
	if s.room != roomSquare {
		t.Fatalf("the tavern's Exit left room %d, want the square", s.room)
	}
	if !slices.Contains(f.Town.Available(), mission) {
		t.Fatalf("leaving the tavern left available %v, want mission %d", f.Town.Available(), mission)
	}
	before := len(voices.samples)
	pressGate(t, app, gate)
	if !s.AtWorldMap() || s.AtTownSquare() || gateDialogueOpen(s) || app.Screen() != ui.ScreenTown {
		t.Fatalf("the press with mission %d on offer: map %v, square %v, line open %v, screen %v",
			mission, s.AtWorldMap(), s.AtTownSquare(), gateDialogueOpen(s), app.Screen())
	}
	if len(voices.samples) != before {
		t.Errorf("the press with a mission on offer requested %d more recordings", len(voices.samples)-before)
	}
	if got := app.HeadlessMessage(); got != "" {
		t.Errorf("the press with a mission on offer posted %q", got)
	}
	if view := s.WorldMapView(); !slices.ContainsFunc(view.Missions, func(m ui.WorldMapMission) bool { return m.Number == mission }) {
		t.Errorf("the world map lists %d scrolls without mission %d", len(view.Missions), mission)
	}
}
