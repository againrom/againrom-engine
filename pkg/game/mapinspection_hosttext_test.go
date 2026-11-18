package game

import (
	"testing"

	"againrom/pkg/render/text"
)

// The editor's host text folds Ё, ё and the guillemets to their plain forms on
// the Russian selector and keeps every other byte of the install's alphabet.
func TestInspectionHostTextFoldsEditorGlyphs(t *testing.T) {
	x := &MapInspector{font: &text.Font{Selector: 1}}
	want := EncodeInstallText(`"Елка" еще`, 1)
	if got := x.inspectionUTF8("«Ёлка» ещё"); got != want || want == "" {
		t.Errorf("host text = % x, want % x", got, want)
	}
	if got := x.inspectionUTF8("“x” ° №"); got != `"x" ? ?` {
		t.Errorf("host text = %q", got)
	}
	en := &MapInspector{font: &text.Font{Selector: 0}}
	if got, want := en.inspectionUTF8("«a"), "\xaba"; got != want {
		t.Errorf("selector 0 host text = % x, want % x", got, want)
	}
}

// A mod's text takes the same plain forms.
func TestModTextFoldsYo(t *testing.T) {
	if got := EncodeModText("Ёё", 1); got != EncodeInstallText("Ее", 1) {
		t.Errorf("mod text = % x", got)
	}
}
