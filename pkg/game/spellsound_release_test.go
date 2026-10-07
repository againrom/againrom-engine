package game

import (
	"os"
	"testing"
)

// TestReleaseSpellSoundSelectorPopulation proves the production SoundBank's
// decoded result over a lawful install matches VIDEO-SFX-019's bounded shipped
// population. check-release-tests invokes it once on each preserved EN/RU root.
func TestReleaseSpellSoundSelectorPopulation(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: spell SFX need a lawful install")
	}
	bank := OpenSounds(root)
	if bank == nil {
		t.Fatal("production SoundBank did not open")
	}

	castResolved := map[uint16]bool{3: false, 4: false, 21: false}
	effectResolved := map[uint16]bool{2: true, 4: true, 21: true}
	for spell := uint16(firstSpellSoundID); spell <= lastSpellSoundID; spell++ {
		castSlot := castSpellSoundSlot(spell)
		_, missingCast := castResolved[spell]
		wantCast := !missingCast
		if sample, ok := bank.Sample(castSlot); ok != wantCast {
			t.Errorf("spell %d cast Sfx%d resolved=%v, want %v", spell, castSlot, ok, wantCast)
		} else if ok && (sample.Rate <= 0 || len(sample.PCM) == 0) {
			t.Errorf("spell %d cast Sfx%d decoded empty: rate=%d frames=%d",
				spell, castSlot, sample.Rate, len(sample.PCM))
		}

		effectSlot := effectSpellSoundSlot(spell)
		wantEffect := effectResolved[spell]
		if sample, ok := bank.Sample(effectSlot); ok != wantEffect {
			t.Errorf("spell %d effect Sfx%d resolved=%v, want %v", spell, effectSlot, ok, wantEffect)
		} else if ok && (sample.Rate <= 0 || len(sample.PCM) == 0) {
			t.Errorf("spell %d effect Sfx%d decoded empty: rate=%d frames=%d",
				spell, effectSlot, sample.Rate, len(sample.PCM))
		}
	}
}
