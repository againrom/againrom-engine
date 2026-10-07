//go:build savdocumentaudit

package game

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Opt-in read-only corpus audit. This is a production import/native round trip
// with an independent FrontEnd, not a new process or a ROM1 runtime witness.
func TestDocument1115NativeLawfulCorpus(t *testing.T) {
	root := os.Getenv("AGAINROM_DOCUMENT_CORPUS")
	if root == "" {
		t.Fatal("AGAINROM_DOCUMENT_CORPUS must name explicit read-only inputs")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit consumer install")
	}
	// This separately tagged audit has mandatory inputs, not a skipped
	// release-test population. Construct both production frontends directly.
	warm, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	cold, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	warm.SetDeterministicFrames(true)
	cold.SetDeterministicFrames(true)
	worlds := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".sav") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			raw, err := ReadSaveFile(path)
			if err != nil {
				t.Fatal(err)
			}
			before := sha256.Sum256(raw)
			source, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			if source.World == nil {
				t.Log("city belongs to the existing city path")
				return
			}
			worlds++
			open, town, err := warm.RestoreOriginal(raw)
			if err != nil || town {
				t.Fatal("original prepare", town, err)
			}
			if err := warm.App("native document audit").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			clear(raw)
			snapshot, label, err := warm.Snapshot(true)
			if err != nil {
				t.Fatal("snapshot", err)
			}
			if snapshot.SavedDocument == nil || snapshot.SavedDocument.Document == nil {
				t.Fatal("complete document unavailable")
			}
			encoded, err := EncodeSave(snapshot, label)
			if err != nil {
				t.Fatal("native encode", err)
			}
			decoded, _, err := DecodeSave(encoded)
			if err != nil {
				t.Fatal("native decode", err)
			}
			again, err := EncodeSave(decoded, label)
			if err != nil || !bytes.Equal(encoded, again) {
				t.Fatal("native encoding unstable", err)
			}
			open, town, err = cold.Restore(decoded)
			if err != nil || town {
				t.Fatal("native prepare", town, err)
			}
			if err := cold.App("independent native document audit").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			if warm.live.world.Hash() != cold.live.world.Hash() {
				t.Fatal("native world hash differs")
			}
			for range 2 {
				sim.Step(warm.live.world, nil)
				sim.Step(cold.live.world, nil)
			}
			if warm.live.world.Hash() != cold.live.world.Hash() {
				t.Fatal("next native steps differ")
			}
			next, _, err := cold.Snapshot(true)
			if err != nil || next.SavedDocument == nil || next.SavedDocument.Document == nil {
				t.Fatal("next snapshot", err)
			}
			check, err := ReadSaveFile(path)
			if err != nil || sha256.Sum256(check) != before {
				t.Fatal("read-only source changed", err)
			}
			t.Logf("source=%x objects=%d actor-bindings=%d native-bytes=%d", before, len(snapshot.SavedDocument.Document.Objects), len(snapshot.SavedDocument.Actors), len(encoded))
		})
	}
	if worlds == 0 {
		t.Fatal("no world inputs")
	}
	t.Logf("world paths audited: %d", worlds)
}
