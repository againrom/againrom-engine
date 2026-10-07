package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func speechWAV(sample int16) []byte {
	b := make([]byte, 48)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], 40)
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], 22050)
	binary.LittleEndian.PutUint32(b[28:], 44100)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], 4)
	binary.LittleEndian.PutUint16(b[44:], uint16(sample))
	binary.LittleEndian.PutUint16(b[46:], uint16(-sample))
	return b
}

func TestTownSpeechPathUsesTheDisplayedPartAndOverride(t *testing.T) {
	payload := []byte(`<part=1 iammale sound="man">M<part=1 iamfemale sound="woman">F<part=2>Next`)
	for _, tc := range []struct {
		part   int
		female bool
		want   string
	}{
		{1, false, "speech/inn/npc/man.wav"},
		{1, true, "speech/inn/npc/woman.wav"},
		{2, false, "speech/inn/npc/npc23m80p2.wav"},
	} {
		got, ok := TownSpeechPath(`MAIN\TEXT\INN\NPC\NPC23M80.TXT`, payload, tc.part, EventAudience{HeroFemale: tc.female})
		if !ok || got != tc.want {
			t.Fatalf("part %d female %v: %q %v, want %q", tc.part, tc.female, got, ok, tc.want)
		}
	}
	for _, tag := range []string{`<part=1 sound="../other">`, `<part=1 sound="C:/other">`} {
		if name, ok := TownSpeechPath("main/text/shop/npc31m31.txt", []byte(tag), 1, EventAudience{}); ok {
			t.Fatalf("invalid speech name accepted: %s", name)
		}
	}
}

func TestTownSpeechUsesArchiveAndStopsAcrossDialogueLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		text                    string
		building                TownBuilding
		npc, mission, mercenary int
	}{
		{"tavern", "inn/npc/npc25m100", TownTavern, 25, 100, 0},
		{"mercenary", "inn/mercenary/npc01", TownTavern, 0, 0, 1},
		{"merchant", "shop/npc31m31", TownShop, 0, 31, 0},
		{"teacher", "training/npc34m61", TownSchool, 0, 61, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			mainPath := filepath.Join(root, "main.res")
			if err := os.WriteFile(mainPath, synth.Archive([]synth.File{{Path: "text/" + tc.text + ".txt", Data: []byte("<part=1 NPC=22>\r\nOne\r\n<part=2 NPC=22>\r\nTwo")}}), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "SPEECH.RES"), synth.Archive([]synth.File{
				{Path: tc.text + "p1.wav", Data: speechWAV(1000)},
				{Path: tc.text + "p2.wav", Data: speechWAV(2000)},
			}), 0600); err != nil {
				t.Fatal(err)
			}
			src, err := vfs.Open([]string{mainPath}, nil)
			if err != nil {
				t.Fatal(err)
			}
			f := shellFrontEnd()
			if tc.mercenary != 0 {
				f.NPCFaces = map[int32]data.NPCFace{22: {Kind: data.NPCNoPicture}}
			}
			f.Archives = &Archives{Containers: src}
			f.SpeechBank = OpenSpeech(root)
			recorder := &tavernInteriorRecorder{}
			effects := &tavernInteriorRecorder{}
			f.SoundPlayer, f.SpeechPlayer = effects, recorder
			a := f.App("speech lifecycle")
			defer a.StopAudio()
			a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} }, func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
			if err := a.HeadlessKey("load"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessActivate("@first"); err != nil {
				t.Fatal(err)
			}
			s := f.townUI
			open := func() {
				if tc.mercenary != 0 {
					s.openMercenaryDialogue(tc.mercenary)
				} else if !s.openTownDialogue(tc.building, TownOffer{Mission: tc.mission}, tc.npc) {
					t.Fatal("dialogue missing")
				}
				if err := a.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			open()
			if len(recorder.samples) != 1 || recorder.samples[0].Rate != audio.DeviceRate || recorder.samples[0].PCM[0] != 1000 {
				t.Fatalf("first line: %+v", recorder.samples)
			}
			for i := 0; i < 4; i++ {
				_ = a.HeadlessStep()
				s.TownDialogue()
			}
			if len(recorder.samples) != 1 {
				t.Fatal("render/update replayed a line")
			}
			s.AdvanceTownDialogue()
			_ = a.HeadlessStep()
			if len(recorder.samples) != 2 || recorder.samples[1].PCM[0] != 2000 || recorder.voices[0].stops != 1 {
				t.Fatal("paging did not replace the voice")
			}
			s.AdvanceTownDialogue()
			_ = a.HeadlessStep()
			if recorder.voices[1].stops != 1 {
				t.Fatal("closed dialogue retained voice")
			}
			open()
			if len(recorder.samples) != 3 {
				t.Fatal("reopened conversation did not speak")
			}
			a.StopAudio()
			if recorder.voices[2].stops != 1 {
				t.Fatal("application exit retained voice")
			}
			if len(effects.voices) != 0 {
				t.Fatal("dialogue leaked into the effects device")
			}
		})
	}
}

func TestTownSpeechHeroVariantFollowsTheNamedSpeaker(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.res")
	if err := os.WriteFile(mainPath, synth.Archive([]synth.File{{Path: "text/inn/npc/npc23m80.txt", Data: []byte("<part=1 NPC=22>\r\nHero")}}), 0600); err != nil {
		t.Fatal(err)
	}
	src, err := vfs.Open([]string{mainPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := shellFrontEnd()
	f.Archives = &Archives{Containers: src}
	f.NPCFaces = map[int32]data.NPCFace{22: {Kind: data.NPCNoPicture, Tokens: data.NPCTokens(data.NPCTokenMe)}}
	f.Carried = []mapload.PartyMember{
		{PlayerCharacter: true, FigureDir: string(data.FigureDirManFighter)},
		{PlayerCharacter: true, StartingHero: true, FigureDir: string(data.FigureDirWomanMage), Mage: true},
	}
	s := f.TownScreen().(*townScreen)
	if !s.openTownDialogue(TownTavern, TownOffer{Mission: 80}, 23) {
		t.Fatal("dialogue missing")
	}
	if got, ok := s.townSpeechPath(); !ok || got != "speech/inn/npc/fm_23m80p1.wav" {
		t.Fatalf("speaker variant = %q %v", got, ok)
	}
}
