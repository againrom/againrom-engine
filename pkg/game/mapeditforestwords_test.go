package game

import (
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/words"
)

// forestWordIDs are the editor words the forest brush adds.
var forestWordIDs = []string{
	"editor.brush.tool",
	"editor.brush.tool.one",
	"editor.brush.tool.forest",
	"editor.brush.density",
	"editor.brush.density.sparse",
	"editor.brush.density.medium",
	"editor.brush.density.dense",
	"editor.brush.density.solid",
	"editor.brush.forest.hint",
	"editor.brush.forest.clear",
	"editor.brush.forest.none",
	"editor.brush.forest.status",
	"editor.brush.share.bad",
	"editor.prompt.share",
}

// TestForestWordsSurviveEditorEncoding passes every forest word through the
// encoding the editor draws host text with and fails on any character it
// would replace with a question mark.
func TestForestWordsSurviveEditorEncoding(t *testing.T) {
	for _, c := range []struct {
		lang     string
		selector int
	}{{"en", 0}, {"ru", 1}} {
		book := words.For(c.lang)
		for _, id := range forestWordIDs {
			s := book.Text(id)
			if s == id {
				t.Errorf("%s %s: no word", c.lang, id)
				continue
			}
			if _, err := textinput.EncodeText(s, c.selector, textinput.Refuse); err != nil {
				t.Errorf("%s %s %q: %v", c.lang, id, s, err)
			}
		}
	}
}
