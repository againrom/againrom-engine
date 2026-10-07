package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestTownEffectMissingReferenceWritesNull(t *testing.T) {
	path := os.Getenv("AGAINROM_TOWN_EFFECT_SAV")
	if path == "" {
		t.Skip("AGAINROM_TOWN_EFFECT_SAV is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "6bc49c417554999fe5d41c604f77cbaf9f29dc71e43035033f8ff125cc5d8f4b" {
		t.Fatalf("town source SHA-256 %s is not the owner input", got)
	}
	source, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	stale := 0
	for i := range source.Objects {
		for _, name := range []string{"Identity", "This"} {
			if key, err := savedStructureValue(&source.Objects[i], name); err == nil && key == 0x01000080 {
				t.Fatal("source Effect reference unexpectedly resolves")
			}
		}
		if source.Objects[i].Class == "Effect" {
			reference, err := savedStructureValue(&source.Objects[i], "Reference")
			if err == nil && reference == 0x01000080 {
				stale++
			}
		}
	}
	if stale != 1 {
		t.Fatalf("source has %d Effect records with missing key 0x01000080; want 1", stale)
	}
	f := releaseFront(t)
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("town LOAD town=%t: %v", town, err)
	}
	written, err := sav.DecodeDocumentData(currentTownSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	effects := 0
	for i := range written.Objects {
		if written.Objects[i].Class != "Effect" {
			continue
		}
		effects++
		reference, err := savedStructureValue(&written.Objects[i], "Reference")
		if err != nil || reference != 0 {
			t.Fatalf("resaved Effect %d Reference %#x, err=%v; want zero", i+1, reference, err)
		}
	}
	if effects == 0 {
		t.Fatal("town resave lost its Effect")
	}
}

func TestTownEffectResolvingReferenceIsPreserved(t *testing.T) {
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{
		{Class: "Item", Values: []sav.DocumentValueData{{Name: "Identity", Value: 0x101}}},
		{Class: "Effect", Values: []sav.DocumentValueData{{Name: "Reference", Value: 0x101}}},
		{Class: "Effect", Values: []sav.DocumentValueData{{Name: "Reference", Value: 0x102}}},
		{Class: "Effect", Values: []sav.DocumentValueData{{Name: "Reference", Value: 0}}},
	}}
	finalizeCurrentTownEffectOwners(&doc)
	for index, want := range []uint32{0x101, 0, 0} {
		got, err := savedStructureValue(&doc.Objects[index+1], "Reference")
		if err != nil || got != want {
			t.Fatalf("Effect %d Reference %#x, err=%v; want %#x", index+1, got, err, want)
		}
	}
}
