package game

import (
	"os"
	"testing"
)

func TestReleaseFixedUISoundSelectorsResolveFromTheLawfulInstall(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: fixed UI SFX need a lawful install")
	}
	bank := OpenSounds(root)
	if bank == nil {
		t.Fatal("production SoundBank did not open")
	}
	for _, slot := range []int{1, 2, 7, 8, 14, 16, 100} {
		s, ok := bank.Sample(slot)
		if !ok {
			t.Errorf("shipped Sfx%d did not resolve", slot)
			continue
		}
		if s.Rate <= 0 || len(s.PCM) == 0 {
			t.Errorf("shipped Sfx%d decoded empty: rate=%d frames=%d", slot, s.Rate, len(s.PCM))
		}
	}
}
