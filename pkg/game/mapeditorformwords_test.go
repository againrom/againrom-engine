package game

import (
	"errors"
	"fmt"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// A bare library limit asks the extend question with a worded reason: the
// limit by name and number, in the editor's language.
func TestFormQuestionForABareLimitNamesTheLimit(t *testing.T) {
	limits := []struct {
		limit mapedit.ScriptLimit
		args  []any
		key   string
	}{
		{mapedit.LimitPairs, []any{3, 4}, "editor.script.limit.pairs"},
		{mapedit.LimitActions, []any{3, 4}, "editor.script.limit.actions"},
		{mapedit.LimitCompare, []any{3, 4}, "editor.script.limit.compare"},
		{mapedit.LimitGroups, []any{4}, "editor.script.limit.groups"},
		{mapedit.LimitTriggers, []any{4, 3}, "editor.script.limit.triggers"},
		{mapedit.LimitChecks, []any{4, 3}, "editor.script.limit.checks"},
		{mapedit.LimitVariables, []any{2, 4}, "editor.script.limit.variables"},
		{mapedit.LimitID, []any{3, 4}, "editor.script.limit.id"},
		{mapedit.LimitDialogue, []any{4}, "editor.script.limit.dialogue"},
	}
	for _, c := range []struct {
		profile base.Profile
		sel     int
	}{{base.Profile{Language: "english"}, 0}, {base.Profile{Language: "russian"}, text.SelectorConverting}} {
		x := heroLabelInspector(t, c.profile, c.sel)
		m := formHost(openHeroLabelMap(t, x, formMap(t, 2))).model
		for _, l := range limits {
			err := &mapedit.ScriptLimitError{Limit: l.limit, Max: 3, Got: 4}
			if !err.Lifted() {
				continue
			}
			var need *ui.ScriptFormNeeded
			want := fmt.Sprintf(words.For(c.profile.Language).Text(l.key), l.args...)
			if !errors.As(formNeeded(m, err, x.word), &need) || need.Why != want {
				t.Errorf("%s limit %d: the question reads %+v, want %q", c.profile.Language, l.limit, need, want)
			}
			if c.profile.Language == "russian" && !cyrillic.MatchString(need.Why) {
				t.Errorf("limit %d: the Russian question reads %q", l.limit, need.Why)
			}
		}
	}
}
