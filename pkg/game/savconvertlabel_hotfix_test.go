package game

import (
	"bytes"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

// An AGS header label is UTF-8 text typed into the old dialog; a SAV label is
// install code-page bytes. The conversion re-encodes the former into the
// latter through the dialog's own rune mapping and reports every compromise
// instead of refusing the file.
func TestMigrateLabelThroughTheInstallCodePage(t *testing.T) {
	f := &FrontEnd{}
	encoded := func(label string) string {
		t.Helper()
		out, err := encodeSaveLabel(label, f.textSelector())
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	for _, tc := range []struct {
		name, source, want string
		debt               int
	}{
		{"ascii", "renamed", "renamed", 0},
		{"cyrillic", "Моя игра", encoded("Моя игра"), 0},
		{"one cyrillic letter", "пgooo!", string([]byte{0xef, 'g', 'o', 'o', 'o', '!'}), 0},
		{"rune outside the code page", "café", "caf?", 1},
		{"control rune", "ok\x01", "ok?", 1},
		{"not utf-8", string([]byte{0xcc, 0xee, 0xff, 0x20, 0xe8}), string([]byte{0xcc, 0xee, 0xff, 0x20, 0xe8}), 1},
		{"too long", strings.Repeat("x", 300), strings.Repeat("x", maxConvertedLabel), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, debt := f.MigrateLabel(tc.source)
			if got != tc.want || len(debt) != tc.debt {
				t.Fatalf("MigrateLabel(%q) = %q with %d debt %v, want %q with %d", tc.source, got, len(debt), debt, tc.want, tc.debt)
			}
		})
	}
	if encoded("пgooo!") == "пgooo!" {
		t.Fatal("the fixture selector does not re-encode, so the cyrillic cases prove nothing")
	}
}

func TestConvertLegacySnapshotWritesTheMigratedLabel(t *testing.T) {
	f, _ := city1095Front(t)
	citySnapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	fresh := func() *FrontEnd {
		return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: f.Table, Bodies: f.Bodies}}
	}
	for _, source := range []string{"renamed", "Моя игра", "пgooo!"} {
		conv, err := fresh().ConvertLegacySnapshot(citySnapshot, source)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := fresh().MigrateLabel(source)
		doc, err := sav.DecodeDocumentData(conv.SAV)
		if err != nil {
			t.Fatal(err)
		}
		if conv.Label != want || !bytes.Equal(doc.Label, []byte(want)) || len(conv.LabelDebt) != 0 {
			t.Fatalf("source %q: label %q, slot %q, debt %v; want %q", source, conv.Label, doc.Label, conv.LabelDebt, want)
		}
	}
}
