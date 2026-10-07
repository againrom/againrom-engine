package game

import (
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// gateLinePayload stands for the shipped gate line: one part, one recording.
const gateLinePayload = "<part=1>\r\nNobody has given you work yet."

// gateLineFront is the town over the square's picture whose archives ship the
// gate line and its recording, or only an unrelated inn line when ship is false.
// Every audio device is a recorder, so nothing is heard.
func gateLineFront(t *testing.T, ship bool) (*FrontEnd, *tavernInteriorRecorder) {
	t.Helper()
	root := t.TempDir()
	name := "npc01"
	if ship {
		name = "npc35"
	}
	mainPath := filepath.Join(root, "main.res")
	if err := os.WriteFile(mainPath, synth.Archive([]synth.File{{Path: "text/inn/mercenary/" + name + ".txt", Data: []byte(gateLinePayload)}}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "SPEECH.RES"), synth.Archive([]synth.File{{Path: "inn/mercenary/" + name + "p1.wav", Data: speechWAV(1234)}}), 0o600); err != nil {
		t.Fatal(err)
	}
	src, err := vfs.Open([]string{mainPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := shellFrontEnd()
	f.Archives = &Archives{Containers: src}
	f.SpeechBank = OpenSpeech(root)
	f.TownSquareArt = resolved(exteriorTestArt(t), nil)
	f.Font = resolved(missionFont(), nil)
	clearTownTestGateLatches(f.Town)
	recorder := &tavernInteriorRecorder{}
	f.SpeechPlayer = recorder
	return f, recorder
}

// squareGatePoint is a pixel of the square's mask that the gate door answers:
// the middle one in reading order, clear of the door's edge.
func squareGatePoint(t *testing.T, f *FrontEnd) image.Point {
	t.Helper()
	mask := f.TownSquareArt.Value().Mask
	var gate []image.Point
	for y := 0; y < mask.Bounds().Dy(); y++ {
		for x := 0; x < mask.Bounds().Dx(); x++ {
			p := image.Pt(x, y)
			if c, ok := ui.TownSquareControlAt(mask, p); ok && c.Kind == ui.TownSquareControlDoor && c.Door == 3 {
				gate = append(gate, p)
			}
		}
	}
	if len(gate) == 0 {
		t.Fatal("the square's mask has no gate pixel")
	}
	return gate[len(gate)/2]
}

func gateDialogueOpen(s *townScreen) bool {
	_, open := s.TownDialogue()
	return open
}

// pressGate presses and releases the primary button on the gate door.
func pressGate(t *testing.T, app *ui.App, p image.Point) {
	t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
}

// TestGatePressWithNothingOnOfferKeepsTheTownViewAndSpeaksTheGateLine presses
// the gate door through App input with no mission available. The original keeps
// the town view, opens the gate line with its recording and posts nothing
// (TOWN-475); it opens no map.
func TestGatePressWithNothingOnOfferKeepsTheTownViewAndSpeaksTheGateLine(t *testing.T) {
	f, voices := gateLineFront(t, true)
	app, s := exteriorApp(t, f)
	pressGate(t, app, squareGatePoint(t, f))

	if app.Screen() != ui.ScreenTown || s.AtWorldMap() {
		t.Fatalf("the gate press with nothing on offer left screen %v with the world map showing %v", app.Screen(), s.AtWorldMap())
	}
	if !s.AtTownSquare() || !gateDialogueOpen(s) {
		t.Fatalf("the town view stays with the gate line open over it: square %v, line open %v", s.AtTownSquare(), gateDialogueOpen(s))
	}
	if path, ok := s.townTextPath(); !ok || path != "main/text/inn/mercenary/npc35.txt" {
		t.Fatalf("the open line reads %q, want the gate line node", path)
	}
	if got := s.townLines(); !reflect.DeepEqual(got, []string{"Nobody has given you work yet."}) {
		t.Fatalf("the gate line says %q", got)
	}
	if got := app.HeadlessMessage(); got != "" {
		t.Errorf("the gate press posted %q, want nothing", got)
	}
	if len(voices.samples) != 1 || voices.samples[0].PCM[0] != 1234 {
		t.Fatalf("recordings started %+v, want the gate line's own one", voices.samples)
	}
	if path, _ := s.townSpeechPath(); path != "speech/inn/mercenary/npc35p1.wav" {
		t.Errorf("the line speaks %q", path)
	}

	// OK ends the line: the town view is where it was, nothing was posted, the
	// recording stops and nothing became available.
	if err := app.HeadlessActivate("dialogue"); err != nil {
		t.Fatal(err)
	}
	if gateDialogueOpen(s) || !s.AtTownSquare() || s.AtWorldMap() || app.Screen() != ui.ScreenTown {
		t.Fatalf("OK left the line open %v, square %v, map %v, screen %v", gateDialogueOpen(s), s.AtTownSquare(), s.AtWorldMap(), app.Screen())
	}
	if got := app.HeadlessMessage(); got != "" {
		t.Errorf("ending the gate line posted %q, want nothing", got)
	}
	if voices.voices[0].stops == 0 {
		t.Error("ending the gate line left its recording playing")
	}
	if got := f.Town.Available(); len(got) != 0 {
		t.Errorf("the gate line made %v available", got)
	}
	if len(s.Rows()) != len(townDoors) {
		t.Errorf("the square lists %d rows after the line, want its %d doors", len(s.Rows()), len(townDoors))
	}
}

// Escape ends the open line as OK does and leaves the town view, not the menu.
func TestGateLineEndsOnEscapeAndOnTheRowRoute(t *testing.T) {
	f, voices := gateLineFront(t, true)
	app, s := exteriorApp(t, f)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if s.AtWorldMap() || !gateDialogueOpen(s) || len(voices.samples) != 1 {
		t.Fatalf("the GATES row with nothing on offer: map %v, line open %v, recordings %d", s.AtWorldMap(), gateDialogueOpen(s), len(voices.samples))
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || gateDialogueOpen(s) || !s.AtTownSquare() {
		t.Fatalf("Escape left screen %v, line open %v, square %v", app.Screen(), gateDialogueOpen(s), s.AtTownSquare())
	}
	if voices.voices[0].stops == 0 {
		t.Error("Escape left the recording playing")
	}
}

// A press with a mission on offer still enters the world map and speaks nothing.
func TestGatePressWithAMissionOnOfferStillEntersTheWorldMap(t *testing.T) {
	f, voices := gateLineFront(t, true)
	f.Town.announceMission(f.Town.currentMain())
	app, s := exteriorApp(t, f)
	pressGate(t, app, squareGatePoint(t, f))
	if !s.AtWorldMap() || gateDialogueOpen(s) || len(voices.samples) != 0 {
		t.Fatalf("the gate press with a mission on offer: map %v, line open %v, recordings %d", s.AtWorldMap(), gateDialogueOpen(s), len(voices.samples))
	}
	if got := app.HeadlessMessage(); got != "" {
		t.Errorf("the gate press posted %q", got)
	}
}

func TestGatePressWithNothingOnOfferAndNoLineReportsMissingNode(t *testing.T) {
	f, voices := gateLineFront(t, false)
	app, s := exteriorApp(t, f)
	pressGate(t, app, squareGatePoint(t, f))
	if s.AtWorldMap() || gateDialogueOpen(s) || !s.AtTownSquare() || len(voices.samples) != 0 || app.Screen() != ui.ScreenTown {
		t.Fatalf("no shipped line: map %v, line open %v, square %v, recordings %d, screen %v",
			s.AtWorldMap(), gateDialogueOpen(s), s.AtTownSquare(), len(voices.samples), app.Screen())
	}
	if got := app.HeadlessMessage(); got != "Town dialogue unavailable: "+townGateTextPath {
		t.Errorf("missing gate node report: %q", got)
	}
}

// TestGatePressFollowsWhatTheTownHasOnOffer presses the gate in each state of
// the town model, every state reached through the routines play calls. The
// press opens the map only while a mission is available and not won.
func TestGatePressFollowsWhatTheTownHasOnOffer(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps func(*testing.T, *Town)
		offer bool
	}{
		{"arrived with nothing accepted", func(*testing.T, *Town) {}, false},
		{"a tavern offer heard and not accepted", func(t *testing.T, town *Town) {
			if got := town.Offers(TownTavern); len(got) == 0 || got[0].Mission != 30 {
				t.Fatalf("the tavern holds %+v, want mission 30 first", got)
			}
		}, false},
		{"a shop offer heard and not accepted", func(t *testing.T, town *Town) {
			if got := town.Offers(TownShop); len(got) == 0 {
				t.Fatal("the shop holds no offer")
			}
		}, false},
		{"a tavern offer accepted", func(t *testing.T, town *Town) {
			if m, ok := town.Take(TownTavern, 0); !ok || m != 30 {
				t.Fatalf("Take = %d %v", m, ok)
			}
		}, true},
		{"the accepted mission won", func(t *testing.T, town *Town) {
			if m, ok := town.Take(TownTavern, 0); !ok || m != 30 {
				t.Fatalf("Take = %d %v", m, ok)
			}
			if _, ok := town.Won(30); !ok {
				t.Fatal("Won refused mission 30")
			}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := townCampaign(t)
			town := NewTown(c)
			town.Arrive()
			tc.steps(t, town)
			if got := len(town.Available()) != 0; got != tc.offer {
				t.Fatalf("the town holds %v on offer, want on offer %v", town.Available(), tc.offer)
			}
			f, _ := gateLineFront(t, true)
			f.Campaign, f.Town = resolved(c, nil), town
			s := f.TownScreen().(*townScreen)
			s.atSquare()
			s.Choose(3)
			if s.AtWorldMap() != tc.offer || gateDialogueOpen(s) == tc.offer {
				t.Fatalf("gate press with on offer %v: world map %v, gate line open %v", tc.offer, s.AtWorldMap(), gateDialogueOpen(s))
			}
			if !tc.offer && !s.AtTownSquare() {
				t.Fatalf("the town view did not stay: room %v", s.room)
			}
		})
	}
}
