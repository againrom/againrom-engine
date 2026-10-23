package shopstock

import (
	"reflect"
	"testing"
)

// seq answers its values in turn, each taken modulo n, and counts the draws.
type seq struct {
	values []int
	at     int
}

func (s *seq) Intn(n int) int {
	v := s.values[s.at%len(s.values)]
	s.at++
	return v % n
}

func TestRollBoundsCountAttemptsOrFailures(t *testing.T) {
	odd := func(c int, _ Source) (int, bool) { return c, c%2 == 1 }
	cands := []int{0, 1}
	// Draws alternate a failing and a passing candidate.
	for _, tc := range []struct {
		name  string
		bound Bound
		want  []int
		draws int
	}{
		{"attempts", Bound{Attempts: func(n int) int { return n + 1 }}, []int{1, 1}, 4},
		{"failures", Bound{Failures: func(n int) int { return n + 1 }}, []int{1, 1, 1}, 6},
		{"none", Bound{}, []int{1, 1, 1}, 6},
	} {
		src := &seq{values: []int{0, 1}}
		got, done := roll(Category[int, int]{Make: odd}, tc.bound, cands, 3, nil, src)
		if !reflect.DeepEqual(got, tc.want) || done != len(tc.want) || src.at != tc.draws {
			t.Errorf("%s: %v took %d after %d draws, want %v after %d", tc.name, got, done, src.at, tc.want, tc.draws)
		}
	}
}

func TestRollRedrawsMarkedPick(t *testing.T) {
	cat := Category[int, int]{
		Redraw:  func(c int) bool { return c == 0 },
		Redraws: 2,
		Make:    func(c int, _ Source) (int, bool) { return c, true },
	}
	src := &seq{values: []int{0, 0, 1, 0, 0, 0, 1}}
	got, done := roll(cat, Bound{Failures: func(int) int { return 5 }}, []int{0, 1}, 2, nil, src)
	// The first pick takes two redraws; the second exhausts them, fails,
	// and the next pick passes.
	if !reflect.DeepEqual(got, []int{1, 1}) || done != 2 || src.at != 7 {
		t.Fatalf("got %v took %d after %d draws", got, done, src.at)
	}
}

func TestGenerateOrderAndMultiplier(t *testing.T) {
	var trace []string
	cat := func(name string) Category[int, string] {
		return Category[int, string]{
			Candidates: func(Source) []int { trace = append(trace, name+" candidates"); return []int{7} },
			Lead:       func([]int, Source) []string { trace = append(trace, name+" lead"); return []string{name + "-lead"} },
			Count:      func(Source) int { return 2 },
			Make:       func(c int, _ Source) (string, bool) { return name, true },
			Finish:     func(l []string, _ Source) []string { trace = append(trace, name+" finish"); return l },
		}
	}
	p := Profile[int, string]{Categories: []Category[int, string]{cat("a"), cat("b")}}
	p.Place = func(s [][]string) { trace = append(trace, "place"); s[0] = append(s[0], "placed") }
	got := Generate(p, &seq{values: []int{0}})
	want := [][]string{{"a-lead", "a", "a", "placed"}, {"b-lead", "b", "b"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stock %v, want %v", got, want)
	}
	if w := []string{"a candidates", "a lead", "a finish", "b candidates", "b lead", "b finish", "place"}; !reflect.DeepEqual(trace, w) {
		t.Fatalf("order %v, want %v", trace, w)
	}
	again := &seq{values: []int{0}}
	p.Multiplier = Multiplier[string]{Factor: 3, Source: func() Source { return again },
		Join: func(_ int, l, more []string) []string { return append(l, more...) }}
	got = Generate(p, &seq{values: []int{0}})
	if len(got[0]) != 8 || len(got[1]) != 7 || again.at != 8 {
		t.Fatalf("multiplied stock %v after %d further draws", got, again.at)
	}
	for _, off := range []int{0, 1} {
		p.Multiplier.Factor = off
		p.Multiplier.Source = nil
		if got := Generate(p, &seq{values: []int{0}}); len(got[0]) != 4 {
			t.Fatalf("factor %d rolled again: %v", off, got)
		}
	}
}

func TestHeldFollowsRepeatRule(t *testing.T) {
	kinds := []int{3, 5, 3}
	id := func(k int) int { return k }
	if got := Held(kinds, id, 3, RepeatReplace); got != 0 {
		t.Fatalf("replace answers %d, want the first held entry", got)
	}
	if got := Held(kinds, id, 4, RepeatReplace); got != -1 {
		t.Fatalf("replace of a new kind answers %d", got)
	}
	if got := Held(kinds, id, 3, RepeatAppend); got != -1 {
		t.Fatalf("append answers %d", got)
	}
}
