package ui

import (
	"strings"
	"testing"
)

// The command answers carry a record's name as UTF-8, as they carry its
// label: a name in the font's Windows-1251 bytes is converted.
func TestEditorCommandRecordNameIsUTF8(t *testing.T) {
	const font, text = "\xc2\xee\xeb\xea", "Волк"
	e := NewMapEditor(nil, nil, nil)
	e.SetHostTextEncoder(func(s string) string { return s }, func(s string) string { return strings.ReplaceAll(s, font, text) })
	got := e.plainRecord(InspectionRecord{Kind: KindUnit, Name: font + ", unit 7", Label: font})
	if got.Name != text+", unit 7" || got.Label != text {
		t.Fatalf("the command record reads name %q, label %q; want %q, %q", got.Name, got.Label, text+", unit 7", text)
	}
}
