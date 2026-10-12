package game

import (
	"path"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// townSpeakerText is one installed town conversation and the way the town
// opens it.
type townSpeakerText struct {
	address string
	open    func(*townScreen) bool
}

// townSpeakerTexts is every installed inn mission-giver, inn mercenary, shop
// and school conversation.
func townSpeakerTexts(t *testing.T, f *FrontEnd) []townSpeakerText {
	t.Helper()
	pattern := regexp.MustCompile(`^main/text/(?:inn/npc/npc(\d+)m(\d+)|inn/mercenary/npc(\d+)|shop/npc31m(\d+)|training/npc34m(\d+))\.txt$`)
	var out []townSpeakerText
	for _, e := range f.Archives.Containers.Entries() {
		m := pattern.FindStringSubmatch(strings.ToLower(e.Address))
		if m == nil {
			continue
		}
		n := func(i int) int { v, _ := strconv.Atoi(m[i]); return v }
		var open func(*townScreen) bool
		switch {
		case m[1] != "":
			npc, mission := n(1), n(2)
			open = func(s *townScreen) bool { return s.openTownDialogue(TownTavern, TownOffer{Mission: mission}, npc) }
		case m[3] != "":
			typ := n(3)
			if _, ok := TownMercenaryTextPath(typ); !ok {
				continue
			}
			open = func(s *townScreen) bool { s.openMercenaryDialogue(typ); return s.talkDiagnostic == "" }
		case m[4] != "":
			mission := n(4)
			open = func(s *townScreen) bool { return s.openTownDialogue(TownShop, TownOffer{Mission: mission}, 0) }
		default:
			mission := n(5)
			open = func(s *townScreen) bool { return s.openTownDialogue(TownSchool, TownOffer{Mission: mission}, 0) }
		}
		out = append(out, townSpeakerText{address: e.Address, open: open})
	}
	return out
}

// Every installed town conversation part, opened through the town's own
// dialogue route with a party of the four hero kinds and each kind in turn as
// the starting hero, starts exactly one recording, the one the route names,
// whenever the install holds that name; the route never names an absent
// recording while the install holds the pager's leaf name (DLG-SOUND-028,
// DLG-VOICE-074); a part whose route names an absent recording starts none.
// Mission 111's inn speaker, NPC 2, speaks both parts for every starting
// hero. Loss control: the parts whose picture-less speaker matched a party
// member and still play the leaf name, where the route named an absent
// sex/class variant before, are counted and include mission 111's two.
func TestReleaseTownSpeakersPlayTheirInstalledLines(t *testing.T) {
	f := releaseFront(t)
	r := &tavernInteriorRecorder{}
	f.SpeechPlayer = r
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil
	texts := townSpeakerTexts(t, f)
	if len(texts) == 0 {
		t.Fatal("the install lists no town conversation")
	}
	kinds := []data.FigureDir{data.FigureDirManFighter, data.FigureDirManMage, data.FigureDirWomanFighter, data.FigureDirWomanMage}
	for first, kind := range kinds {
		f.Carried = nil
		for i, member := range kinds {
			f.Carried = append(f.Carried, mapload.PartyMember{PlayerCharacter: true, StartingHero: i == first, FigureDir: string(member), Mage: member.Mage()})
		}
		s := f.TownScreen().(*townScreen)
		speakers := map[string]bool{}
		parts, voiced, unvoiced, otherVariant, leafOnly, catapult := 0, 0, 0, 0, 0, 0
		for _, text := range texts {
			if !text.open(s) {
				continue
			}
			for n := 0; s.room == roomTalk && n < 64; n++ {
				textPath, _ := s.townTextPath()
				leafName, _ := townSpeechTagPath(textPath, s.dialogue.tag, s.dialogue.displayPart)
				before := len(r.samples)
				s.TownDialogueActive(true)
				started := len(r.samples) - before
				got, _ := s.townSpeechPath()
				parts++
				speaker, named := tagSpeaker(s.dialogue.tag)
				if named {
					speakers[textPath+"#"+strconv.Itoa(speaker)] = true
				}
				part := s.dialogue.displayPart
				switch {
				case f.SpeechBank.has(got):
					if started != 1 {
						t.Fatalf("%s %s part %d: %s is installed, %d recordings started", kind, text.address, part, got, started)
					}
					voiced++
				case f.SpeechBank.has(leafName):
					t.Fatalf("%s %s part %d: the route names absent %s while %s is installed", kind, text.address, part, got, leafName)
				case started != 0:
					t.Fatalf("%s %s part %d: nothing installed, %d recordings started", kind, text.address, part, started)
				default:
					unvoiced++
					dir, leaf := path.Dir(leafName), path.Base(leafName)
					for _, prefix := range []string{"ff_", "fm_", "mf_", "mm_"} {
						if strings.HasPrefix(leaf, "npc") && f.SpeechBank.has(dir+"/"+prefix+strings.TrimPrefix(leaf, "npc")) {
							otherVariant++
							break
						}
					}
				}
				if rec, known := f.NPCFaces[int32(speaker)]; started == 1 && named && known && rec.Kind == data.NPCNoPicture && got == leafName &&
					strings.HasPrefix(textPath, "main/text/inn/npc/") && townSpeechOverride(s.dialogue.tag) == "" && townSpeakerInParty(s, rec) {
					leafOnly++
					if strings.HasSuffix(textPath, "/npc02m111.txt") {
						catapult++
					}
				}
				s.AdvanceTownDialogue()
			}
			s.resetTownSpeech()
		}
		if catapult != 2 {
			t.Fatalf("starting %s: mission 111's speaker voiced %d of 2 parts", kind, catapult)
		}
		t.Logf("starting %s: %d conversations, %d speakers, %d parts, %d voiced, %d unvoiced (%d of them with another speaker's variant installed); loss control %d parts, mission 111's two among them",
			kind, len(texts), len(speakers), parts, voiced, unvoiced, otherVariant, leafOnly)
	}
}

// townSpeakerInParty reports whether a carried member passes the speaker
// record's terms, the condition under which the route tries a sex/class
// variant.
func townSpeakerInParty(s *townScreen, rec data.NPCFace) bool {
	cast := speakerCast{}
	for _, member := range s.sess.Carried {
		if primaryPlayerHero(member) {
			cast.playerDir, _ = memberFigure(member)
			cast.hasPlayer = true
		}
	}
	for _, member := range s.sess.Carried {
		dir, face := memberFigure(member)
		if cast.matches(rec, speakerActor{me: primaryPlayerHero(member), hero: member.PlayerCharacter && member.MercenaryType == 0,
			fig: figureID{Dir: dir, Face: face}, face: int32(face), typeID: member.Class}) {
			return true
		}
	}
	return false
}
