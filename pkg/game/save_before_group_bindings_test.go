package game

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"testing"
)

// These asset-free envelopes were produced by exact published
// 15c707b2fa029c47aaa4cfa58c02a6e581c35b62 without optional Group bindings in
// SnapshotSAVDocument. Form77 peels the two absent Group/Structure spans
// from releasedSaveFixtureSnapshot; Form72 uses preCurrentProfileForm1107.
// The populated document comes from documentSnapshotFixture's synthetic literal
// SAV, not an installed asset or owner save. Preserve these bytes and hashes;
// never regenerate them from a later gob descriptor.
const beforeGroupBindingsDocumentLabel = "complete native document before group bindings"

const beforeGroupBindingsForm77 = "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" +
	"//+NVFcpHz0AAA=="

const beforeGroupBindingsForm72 = "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" +
	"AAD//zKctVsSPQAA"

const beforeGroupBindingsDocument = "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" +
	"bWgDwX8EAAD//wPZdZZOfAAA"

func beforeGroupBindingsEnvelope(t *testing.T, fixture string) []byte {
	t.Helper()
	compressed, err := base64.StdEncoding.DecodeString(fixture)
	if err != nil {
		t.Fatal(err)
	}
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	raw, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestBeforeGroupBindings1115FrozenNativeEnvelopes(t *testing.T) {
	for _, tc := range []struct {
		name, fixture, label, hash string
		length                     int
		worldForm                  byte
		document                   bool
	}{
		{"form77", beforeGroupBindingsForm77, releasedSaveFixtureLabel, "a04b2664e40bf7a502129050189c218c777b4f9d49d3a13e058d288b3b70a213", 15647, 77, false},
		{"form72", beforeGroupBindingsForm72, releasedSaveFixtureLabel, "aa15a5a933eb2ac8e73ff55bbe51641c6207be1df385ef33520115c6aa6301e6", 15634, 72, false},
		{"document", beforeGroupBindingsDocument, beforeGroupBindingsDocumentLabel, "fb534cca57c55d86bfcb90bd3bb8e757d38173e312f8bd89df984026fca2a1cb", 31822, 79, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := beforeGroupBindingsEnvelope(t, tc.fixture)
			if hash := fmt.Sprintf("%x", sha256.Sum256(raw)); hash != tc.hash || len(raw) != tc.length {
				t.Fatalf("frozen predecessor changed: hash %s, length %d", hash, len(raw))
			}
			loaded, label, err := DecodeSave(raw)
			if err != nil {
				t.Fatal("predecessor LOAD", err)
			}
			if label != tc.label || len(loaded.World) == 0 || loaded.World[0] != tc.worldForm || loaded.Mission != 10 || !loaded.Open {
				t.Fatalf("predecessor envelope semantics: label %q mission %d open %t world %v", label, loaded.Mission, loaded.Open, loaded.World[:min(1, len(loaded.World))])
			}
			if !tc.document {
				if loaded.SavedDocument != nil || loaded.Gold != 321 || loaded.Offered != 20 ||
					!reflect.DeepEqual(loaded.Won, []int{10}) || !reflect.DeepEqual(loaded.Available, []int{20}) ||
					!reflect.DeepEqual(loaded.WorldSelectedOnce, []int{10, 20}) {
					t.Fatal("empty-world predecessor state changed")
				}
			} else {
				assertBeforeGroupBindingsDocument1115(t, loaded)
			}
			current, err := EncodeSave(loaded, label)
			if err != nil {
				t.Fatal("current descriptor cannot SAVE predecessor", err)
			}
			resaved, currentLabel, err := DecodeSave(current)
			if err != nil || currentLabel != label || !reflect.DeepEqual(loaded, resaved) {
				t.Fatal("predecessor state changed across current SAVE/LOAD", err)
			}
		})
	}
}

func assertBeforeGroupBindingsDocument1115(t *testing.T, snapshot Snapshot) {
	t.Helper()
	document := snapshot.SavedDocument
	if document == nil || document.Version != 1 || document.Document == nil || document.Unavailable != "" ||
		len(document.Actors) != 1 || document.Actors[0].ObjectIndex == 0 ||
		document.Actors[0].Retired || document.Document.Head.Mission != 10 {
		t.Fatal("frozen populated document was lost or changed")
	}
	// Reflection lets this fixture be captured and tested before the additive
	// descriptor exists. Once present, absent predecessor metadata must stay
	// absent: source containment cannot reconstruct a current native binding.
	if field := reflect.ValueOf(document).Elem().FieldByName("GroupBindings"); field.IsValid() && !field.IsZero() {
		t.Fatal("predecessor acquired GroupBindings absent from its descriptor")
	}
}
