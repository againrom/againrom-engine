package game

import (
	"os"
	"testing"
)

// TestReleaseMissionAmbientPopulation is run once against each preserved EN/RU
// root. It proves the five shipped leaves used by the story decode, the three
// documented holes remain quiet, and the object selector population is exact.
func TestReleaseMissionAmbientPopulation(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: mission ambience needs a lawful install")
	}
	bank := OpenSounds(root)
	if bank == nil {
		t.Fatal("sound archive did not open")
	}
	for _, slot := range []int{50, 60, 61, 70, 90} {
		if sample, ok := bank.Sample(slot); !ok || sample.Rate != 22050 || len(sample.PCM) == 0 {
			t.Errorf("ambient slot %d did not resolve to non-empty 22050 Hz PCM", slot)
		}
	}
	for _, slot := range []int{62, 80, 81} {
		if _, ok := bank.Sample(slot); ok {
			t.Errorf("documented absent ambient slot %d unexpectedly resolved", slot)
		}
	}

	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	set, err := LoadStatics(archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	minusTwo, minusOne, nonNegative := 0, 0, 0
	for _, class := range set.Classes {
		if class == nil {
			continue
		}
		switch {
		case class.FireObject == -2:
			minusTwo++
		case class.FireObject == -1:
			minusOne++
		case class.FireObject >= 0:
			nonNegative++
		}
	}
	if minusTwo != 21 || minusOne != 61 || nonNegative != 0 {
		t.Fatalf("FireObject population = -2:%d -1:%d >=0:%d, want 21/61/0", minusTwo, minusOne, nonNegative)
	}
}
