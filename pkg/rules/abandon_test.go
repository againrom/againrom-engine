package rules

import "testing"

func TestAbandonRulesDefaultAndOverride(t *testing.T) {
	if Defaults().Abandon != 1 || !Default().Abandon() || !(Rules{}).Abandon() {
		t.Fatal("abandon is not on by default")
	}
	for _, enabled := range []int32{0, 1} {
		p := Defaults()
		p.Abandon = enabled
		r, err := New(p)
		if err != nil || r.Abandon() != (enabled == 1) || r.Params() != p || r.IsDefault() != (enabled == 1) {
			t.Fatalf("abandon=%d: %+v %v", enabled, r.Params(), err)
		}
	}
	for _, invalid := range []int32{-1, 2} {
		p := Defaults()
		p.Abandon = invalid
		if _, err := New(p); err == nil {
			t.Fatalf("accepted abandon=%d", invalid)
		}
	}
}
