package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// secondSpeechText is a planted town.txt: one section per name family, a
// sound= override, the hero's own parts, and two sections with no family.
const secondSpeechText = "#npc5talk7\r\n<npc=5, part=1>\r\nOne\r\n<npc=21, part=2>\r\nHero\r\n" +
	"<npc=22, iamfemale, part=3>\r\nShe\r\n<npc=22, iammale, part=3>\r\nHe\r\n" +
	"#npc5about\r\n<npc=5, part=1>\r\nAbout\r\n" +
	"#npc5accept100\r\n<npc=5, part=1>\r\nYes\r\n<npc=22, part=2, sound=\"accept/0022t100p99\">\r\nOverride\r\n" +
	"#npc5reject3\r\n<npc=5, part=1>\r\nNo\r\n" +
	"#druidinnkeeper2\r\n<npc=97, part=1>\r\nKeep\r\n" +
	"#kaargwoman3\r\n<npc=98, part=1>\r\nWoman\r\n" +
	"#plagatguard\r\n<npc=97, part=1>\r\nGuard\r\n" +
	"#event7\r\n<npc=5, part=1>\r\nMission\r\n" +
	"#tips1\r\n<part=1>\r\nTip\r\n"

// secondSpeechRecordings are the planted speech.res keys; recording i plays
// PCM 1000 × (i + 1).
var secondSpeechRecordings = []string{
	"talk/0005t007p01.wav", "talk/0021f007p02.wav", "talk/0021a007p02.wav", "talk/0021m007p02.wav", "talk/0021s007p02.wav",
	"talk/0022f007p03.wav", "talk/0022a007p03.wav", "talk/0022m007p03.wav", "talk/0022s007p03.wav",
	"about/0005t000p01.wav", "accept/0005t100p01.wav", "accept/0022t100p99.wav", "accept/0022t100p02.wav",
	"reject/0005t003p01.wav", "town/0097t002p01.wav", "town/0098t003p01.wav", "town/0097t000p01.wav",
	"event7p1.wav", "talk/0005t007p1.wav", "tips1p1.wav",
}

// secondSpeechHero is a starting hero of the given sex and class.
func secondSpeechHero(female, mage bool) mapload.PartyMember {
	dir := map[[2]bool]data.FigureDir{{false, false}: data.FigureDirManFighter, {true, false}: data.FigureDirWomanFighter,
		{false, true}: data.FigureDirManMage, {true, true}: data.FigureDirWomanMage}[[2]bool{female, mage}]
	return mapload.PartyMember{ID: "21", Name: "Hero", StartingHero: true, PlayerCharacter: true, FigureDir: string(dir), Mage: mage}
}

// secondSpeechTown is a second-game town screen in town 1's inn over the
// planted town.txt and speech archive; its speech device records each line.
func secondSpeechTown(t *testing.T, hero mapload.PartyMember) (*townScreen, *tavernInteriorRecorder) {
	t.Helper()
	files := make([]synth.File, len(secondSpeechRecordings))
	for i, name := range secondSpeechRecordings {
		files[i] = synth.File{Path: name, Data: speechWAV(int16(1000 * (i + 1)))}
	}
	c := newSecondCampaign()
	c.current, c.available, c.room = partyPlace{placeTown, 1}, []partyPlace{{placeTown, 1}}, secondTownInn
	f := &FrontEnd{
		InstallResources: InstallResources{Archives: &Archives{Base: base.Match{Profile: base.Profile{Game: base.GameROM2}},
			Containers: townTextFS(t, []synth.File{{Path: "text/town.txt", Data: []byte(secondSpeechText)}})},
			SpeechBank: responseBank(t, files)},
		CampaignSession: CampaignSession{Town: &Town{second: c}, Carried: []mapload.PartyMember{hero}},
	}
	speech := &tavernInteriorRecorder{}
	f.SoundPlayer, f.speech.player = &tavernInteriorRecorder{}, speech
	s := &townScreen{shopController: shopController{in: &f.InstallResources, sess: &f.CampaignSession},
		sound: runtimeTownAudio{&f.RuntimeServices}, art: fakeTownArt{}, tags: secondEventTags{}}
	s.adapter = destinationTownCampaign{s}
	s.defaults = townDescriptions["rom2-town1"]
	f.townUI = s
	return s, speech
}

// secondSpeechHeard is the planted recording the recorder's line i plays, or
// "" when it has none.
func secondSpeechHeard(r *tavernInteriorRecorder, i int) string {
	if i >= len(r.samples) || len(r.samples[i].PCM) == 0 {
		return ""
	}
	n := int(r.samples[i].PCM[0])/1000 - 1
	if n < 0 || n >= len(secondSpeechRecordings) || int(r.samples[i].PCM[0]) != 1000*(n+1) {
		return "an unplanted sample"
	}
	return secondSpeechRecordings[n]
}

// Each town.txt section's first page plays the recording its family names
// (R2-ENGINE-802): talk, accept and reject under their own directories,
// about with t000, keeper, man and guard under town, the header's NPC in
// four digits, the key's trailing digits in three and the part in two. A
// sound= header names its own recording. A section with no family asks for
// nothing (R2-ASSET-209).
func TestSecondTownSectionSpeaksTheRecordingItsFamilyNames(t *testing.T) {
	for _, tc := range []struct {
		key   string
		pages []string
	}{
		{"npc5talk7", []string{"talk/0005t007p01.wav"}},
		{"npc5about", []string{"about/0005t000p01.wav"}},
		{"npc5accept100", []string{"accept/0005t100p01.wav", "accept/0022t100p99.wav"}},
		{"npc5reject3", []string{"reject/0005t003p01.wav"}},
		{"druidinnkeeper2", []string{"town/0097t002p01.wav"}},
		{"kaargwoman3", []string{"town/0098t003p01.wav"}},
		{"plagatguard", []string{"town/0097t000p01.wav"}},
		{"event7", []string{""}},
		{"tips1", []string{""}},
	} {
		t.Run(tc.key, func(t *testing.T) {
			s, speech := secondSpeechTown(t, secondSpeechHero(false, false))
			s.openTownText(tc.key)
			if !s.campaignTown().talking() {
				t.Fatal("the section did not open")
			}
			heard := 0
			for page, want := range tc.pages {
				if page > 0 {
					s.AdvanceTownDialogue()
				}
				s.TownDialogueActive(true)
				s.TownDialogueActive(true)
				got := secondSpeechHeard(speech, heard)
				if got != want {
					t.Fatalf("page %d played %q, want %q (%d lines)", page+1, got, want, len(speech.samples))
				}
				if want != "" {
					heard++
				}
			}
			if len(speech.samples) != heard {
				t.Fatalf("%d lines played, want %d", len(speech.samples), heard)
			}
		})
	}
}

// A page whose speaker is the hero (NPC 21) or whose header carries a hero
// token takes the letter of the hero's sex and class: f, a, m or s
// (R2-ENGINE-802). The pages after the first play on Next, and the end of
// the conversation stops the last line.
func TestSecondTownHeroPagesTakeTheHerosLetter(t *testing.T) {
	for _, tc := range []struct {
		female, mage bool
		letter       string
	}{
		{false, false, "f"}, {true, false, "a"}, {false, true, "m"}, {true, true, "s"},
	} {
		s, speech := secondSpeechTown(t, secondSpeechHero(tc.female, tc.mage))
		s.openTownText("npc5talk7")
		want := []string{"talk/0005t007p01.wav", "talk/0021" + tc.letter + "007p02.wav", "talk/0022" + tc.letter + "007p03.wav"}
		for page, name := range want {
			if page > 0 {
				s.AdvanceTownDialogue()
			}
			s.TownDialogueActive(true)
			if got := secondSpeechHeard(speech, page); got != name {
				t.Fatalf("hero %q page %d played %q, want %q", tc.letter, page+1, got, name)
			}
			if page > 0 && speech.voices[page-1].stops != 1 {
				t.Fatalf("hero %q page %d did not stop the previous line", tc.letter, page+1)
			}
		}
		s.AdvanceTownDialogue()
		s.TownDialogueActive(true)
		if s.campaignTown().talking() || len(speech.samples) != len(want) || speech.voices[len(want)-1].stops != 1 {
			t.Fatalf("hero %q: the end of the talk left %d lines, last stopped %d times", tc.letter, len(speech.samples), speech.voices[len(want)-1].stops)
		}
	}
}
