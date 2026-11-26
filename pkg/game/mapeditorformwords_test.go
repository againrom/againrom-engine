package game

import (
	"errors"
	"fmt"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/text"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// A bare library limit asks the extend question with a worded reason: the
// limit by name and number, in the editor's language.
func TestFormQuestionForABareLimitNamesTheLimit(t *testing.T) {
	type limitRow struct {
		limit mapedit.ScriptLimit
		args  []any
		key   string
		op    scriptprofile.ExtendedOp
	}
	limits := []limitRow{
		{limit: mapedit.LimitPairs, args: []any{3, 4}, key: "editor.script.limit.pairs"},
		{limit: mapedit.LimitActions, args: []any{3, 4}, key: "editor.script.limit.actions"},
		{limit: mapedit.LimitCompare, args: []any{3, 4}, key: "editor.script.limit.compare"},
		{limit: mapedit.LimitGroups, args: []any{4}, key: "editor.script.limit.groups"},
		{limit: mapedit.LimitTriggers, args: []any{4, 3}, key: "editor.script.limit.triggers"},
		{limit: mapedit.LimitChecks, args: []any{4, 3}, key: "editor.script.limit.checks"},
		{limit: mapedit.LimitVariables, args: []any{2, 4}, key: "editor.script.limit.variables"},
		{limit: mapedit.LimitID, args: []any{3, 4}, key: "editor.script.limit.id"},
	}
	for _, o := range scriptprofile.ExtendedOps {
		row := limitRow{limit: mapedit.LimitExtendedOnly, key: o.Word, op: o}
		if o.Counted {
			row.args = []any{4}
		}
		limits = append(limits, row)
	}
	for _, c := range []struct {
		profile base.Profile
		sel     int
	}{{base.Profile{Language: "english"}, 0}, {base.Profile{Language: "russian"}, text.SelectorConverting}} {
		x := heroLabelInspector(t, c.profile, c.sel)
		m := formHost(openHeroLabelMap(t, x, formMap(t, 2))).model
		for _, l := range limits {
			err := &mapedit.ScriptLimitError{Limit: l.limit, Max: 3, Got: 4, Op: l.op}
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
