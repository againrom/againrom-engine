package mod

import (
	"strings"
	"testing"
)

// The override line of a file-form mission names the file and no line.
func TestTheOverrideLineOfAMapFileHasNoLineNumber(t *testing.T) {
	a := mapMod(t, map[string]string{"20.alm": "A"})
	b := mapMod(t, map[string]string{"20.alm": "B"})
	var d Data
	for _, m := range []struct{ id, dir string }{{"a", a}, {"b", b}} {
		if _, err := declaredMissions(t, &d, m.id, m.dir, "en"); err != nil {
			t.Fatal(err)
		}
	}
	admitted, _, err := d.Missions.Listed(campaignOf(20))
	if err != nil {
		t.Fatal(err)
	}
	admitted.Order = Precedence{"b": {"a"}}
	_, overrides, err := MergeMissions(admitted)
	if err != nil || len(overrides) != 1 {
		t.Fatalf("%v %v", overrides, err)
	}
	if got := overrides[0].String(); !strings.Contains(got, `(20.alm) overrides mod "a"`) || strings.Contains(got, ":0") {
		t.Errorf("%s", got)
	}
	row := Override{Kind: "row", Key: "k", Field: "f", Earlier: "a", Later: Source{Mod: "b", File: "data/x.toml", Line: 5}}
	if got := row.String(); !strings.Contains(got, "(data/x.toml:5)") {
		t.Errorf("%s", got)
	}
}
