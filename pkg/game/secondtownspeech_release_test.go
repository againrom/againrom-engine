package game

import (
	"slices"
	"testing"
)

// TestReleaseSecondTownAcceptSpeaksTheClaimedRecordings: on either ROM2 root
// the hire reply npc5accept100 asks speech.res, page by page, for the
// recordings R2-ENGINE-802 names: accept/0005t100p01 and p03 by the accept
// family, and on page 2 accept/0022t100p02 by the family on EN or the
// header's sound= accept/0022t100p99 on RU (R2-ASSET-209). Each name
// decodes from the installed archive and reaches the speech device; the end
// of the reply stops the last line.
func TestReleaseSecondTownAcceptSpeaksTheClaimedRecordings(t *testing.T) {
	f, _, s, _ := secondTownTwo(t)
	speech := &tavernInteriorRecorder{}
	f.speech.player = speech
	second := "speech/accept/0022t100p02.wav"
	if f.Archives.Base.Profile.Language == "russian" {
		second = "speech/accept/0022t100p99.wav"
	}
	want := []string{"speech/accept/0005t100p01.wav", second, "speech/accept/0005t100p03.wav"}
	s.openTownText("npc5accept100")
	if !s.campaignTown().talking() {
		t.Fatal("npc5accept100 did not open")
	}
	for page, name := range want {
		if page > 0 {
			s.AdvanceTownDialogue()
		}
		s.TownDialogueActive(true)
		sample, ok := f.SpeechBank.Sample(name)
		if !ok || len(speech.samples) != page+1 || !slices.Equal(speech.samples[page].PCM, sample.PCM) {
			t.Fatalf("page %d played %d lines, want %d ending in %s (decoded %v)", page+1, len(speech.samples), page+1, name, ok)
		}
	}
	s.AdvanceTownDialogue()
	s.TownDialogueActive(true)
	if s.campaignTown().talking() || speech.voices[len(want)-1].stops != 1 {
		t.Fatalf("the reply's end left the talk open %v, last line stopped %d times", s.campaignTown().talking(), speech.voices[len(want)-1].stops)
	}
}
