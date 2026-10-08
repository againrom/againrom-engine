package game

import (
	"againrom/pkg/formats/sav"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCensusScalarProbesExposeBothPhysicalWordBytes(t *testing.T) {
	t.Setenv("AGAINROM_CENSUS_HIGH", "1")
	doc, _, _ := structureProjectionFixture(t)
	env := censusEnv{t: t, prepare: func() *censusRun {
		clone, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		return &censusRun{leaves: censusLeaves(&clone), export: func() ([]byte, error) { return sav.EncodeDocumentData(clone) }, done: func() {}}
	}}
	base, _ := env.baseline()
	_, by := censusPatterns(censusLeaves(&doc))
	result := env.region(base, "Building.v.B46", by["Building.v.B46"])
	if result.source != sourceDocument || !reflect.DeepEqual(result.docOffsets, []int{0, 1}) {
		t.Fatalf("physical word bytes not probed: %+v", result)
	}
	if !reflect.DeepEqual(result.acceptedMasks, []string{"1", "100", "80", "4000"}) {
		t.Fatalf("physical width controls: %+v", result)
	}
}

func TestSAVDocumentFallbacksNameOnlyUnknownSpans(t *testing.T) {
	seen := map[string]bool{}
	for _, entry := range savDocumentFallbacks {
		key := entry.Pattern + "/" + entry.Span + "/" + entry.Indices
		if seen[key] || entry.Pattern == "" || entry.Span == "" || !strings.Contains(entry.Reason, "Meaning remains unknown") {
			t.Fatalf("fallback must name a unique unknown-meaning span: %+v", entry)
		}
		seen[key] = true
		if _, err := os.Stat(filepath.Join("..", "..", "knowledge", "formats", entry.Page)); err != nil {
			t.Fatalf("fallback authority page %q: %v", entry.Page, err)
		}
		for _, known := range []string{".Identity", ".This", ".RuntimeID", ".Reference", ".Body", ".Name", ".U44"} {
			if strings.HasSuffix(entry.Pattern, known) {
				t.Fatalf("known state or graph anchor is not unknown-meaning debt: %+v", entry)
			}
		}
	}
	if got := censusUnknownSpans("Human.v.Body"); got != "" {
		t.Fatal("known current field entered unknown-meaning fallback", got)
	}
	if got := censusUnknownSpans("Building.r.B52"); !strings.Contains(got, "0-13,16-17") || strings.Contains(got, "18-21") {
		t.Fatal("current width/height/blocking overlap entered fallback", got)
	}
}
