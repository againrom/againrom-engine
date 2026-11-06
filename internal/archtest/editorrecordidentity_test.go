package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestEditorRecordIdentityGuardNamesPlantedRules(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"decrement", "func shift(ref RecordRef) { ref.Index-- }", "index arithmetic"},
		{"alias increment", "func shift(ref RecordRef) { at := ref; at.Index++ }", "index arithmetic"},
		{"subtract assignment", "func shift(ref RecordRef) { ref.Index -= 2 }", "index arithmetic"},
		{"add assignment", "func shift(ref RecordRef) { ref.Index += 2 }", "index arithmetic"},
		{"explicit assignment", "func shift(ref RecordRef) { ref.Index = ref.Index - 1 }", "index arithmetic"},
		{"new reference", "func shift(ref RecordRef) RecordRef { return RecordRef{Index: ref.Index - 1} }", "index arithmetic"},
		{"loot match", "func f(was InspectionRecord, d *InspectionDocument) { for _, r := range d.Records { if r.Label == was.Label {} } }", "cross-document"},
		{"detail match", "func f(was InspectionRecord, d *InspectionDocument) { for _, r := range d.Records { if slices.Equal(r.Details, was.Details) {} } }", "cross-document"},
		{"selection match", "func f(a, b *InspectionDocument) { index := a.Records[0].Index; for _, r := range b.Records { if r.Index == index {} } }", "cross-document"},
		{"new records by count", "func f(a, b *InspectionDocument) { next := map[RecordKind]int{}; for _, r := range a.Records { next[r.Kind] = r.Index + 1 }; for _, r := range b.Records { if r.Index >= next[r.Kind] {} } }", "index boundary"},
		{"address match", "func f(a, b []InspectionRecord) { old := map[RecordRef]InspectionRecord{}; _ = old }", "mutable record address"},
		{"key table", "func f() { keys := map[RecordRef]uint64{}; _ = keys }", "mutable record address"},
		{"key writer", "func f(d *InspectionDocument) { for _, r := range d.Records { r.Key = 1 } }", "UI writes"},
		{"key builder", "func f() { _ = InspectionRecord{Key: 1} }", "UI builds"},
		{"document hash", "func f(d *InspectionDocument) { _ = sha256.Sum256(d.SourceBytes) }", "document hash"},
		{"retired predictor", "func carryBarRecordKeys() {}", "retired"},
		{"host key", "func f(a, b *InspectionDocument) { key := a.Records[0].Key; for _, r := range b.Records { if r.Key == key {} } }", ""},
		{"same document address", "func f(ref RecordRef, d *InspectionDocument) { for _, r := range d.Records { if r.Kind == ref.Kind && r.Index == ref.Index {} } }", ""},
		{"shop grid", "func f(origin ShopControl, base int) { origin.Index += base }", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := "pkg/ui/planted.go"
			vs := CheckEditorRecordIdentity(map[string]string{file: "package ui\n" + tc.source})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatal(vs)
				}
			} else if len(vs) == 0 || !strings.Contains(vs[0].Reason, tc.want) || !strings.HasPrefix(vs[0].From, file+":") || !strings.Contains(vs[0].Reason, "InspectionRecord.Key") {
				t.Fatalf("violations = %v, want named %s", vs, tc.want)
			}
		})
	}
	if len(CheckEditorRecordIdentity(nil)) != 1 {
		t.Fatal("empty census passed")
	}
}

func TestEditorRecordIdentityGuardLiveTree(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckEditorRecordIdentity(files) {
		t.Error(v)
	}
}
