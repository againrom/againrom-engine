package game

import (
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

// On a Russian install the trap popup's field names and its range message
// are Russian words the editor converts to the font's bytes once; a name
// the host converted already would be converted again and drawn as '?'.
func TestEditorTrapWordsReachTheRussianFontOnce(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	x := s.inspector
	x.font = &text.Font{Selector: text.SelectorConverting}
	x.archives = &Archives{Base: base.Match{Profile: base.Profile{Language: "russian"}}}
	e.SetWords(words.For("russian"))
	e.SetHostTextEncoder(x.inspectionUTF8)
	e.Select(recordIndex(t, e, "Enchantment", 0), false)
	lines := e.PopupLines()
	want := x.inspectionUTF8(x.word("editor.cast.spell"))
	found := false
	for _, line := range lines {
		if strings.Contains(line, "?") {
			t.Errorf("popup line %q holds a question mark", line)
		}
		found = found || strings.HasPrefix(line, want)
	}
	if !found {
		t.Errorf("no popup line starts with the spell field name %q: %q", want, lines)
	}
	if _, err := s.SetField("Enchantment", 0, "power", 101); err == nil || strings.Contains(x.inspectionUTF8(err.Error()), "?") {
		t.Errorf("range message %v", err)
	}
}
