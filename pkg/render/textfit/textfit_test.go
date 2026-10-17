package textfit

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func byteCount(s string) int { return len(s) }

// A rune cut never splits a rune even when the measure counts bytes; a byte
// cut of the same text does, which is what makes the unit a real choice.
func TestARuneCutKeepsWholeRunes(t *testing.T) {
	s := "Победа героя"
	for max := 0; max <= len(s); max++ {
		got := Spec{Measure: byteCount, Unit: Rune}.Fit(s, max)
		if !utf8.ValidString(got) || len(got) > max {
			t.Fatalf("width %d: %q (% x)", max, got, got)
		}
	}
	if got := (Spec{Measure: byteCount, Unit: Byte}).Fit(s, 8); utf8.ValidString(got) {
		t.Fatalf("the byte cut %q kept whole runes; the control does not discriminate", got)
	}
}

// Every cut carries exactly one mark, on its side, including a part that is
// itself the tail of a longer text.
func TestACutIsMarkedOnce(t *testing.T) {
	head := Spec{Measure: byteCount, Unit: Byte}
	if got := head.Fit("abcdefghij", 7); got != "abcd..." {
		t.Fatalf("Fit = %q", got)
	}
	if got := head.Fit("abc", 7); got != "abc" {
		t.Fatalf("a text that fits was cut: %q", got)
	}
	tail := Spec{Measure: byteCount, Unit: Byte, Side: Start}
	if got := tail.Fit("abcdefghij", 7); got != "...ghij" {
		t.Fatalf("Start = %q", got)
	}
	for _, line := range []string{"abc", "abcdefghij", "abcdef"} {
		got := head.Continued(line, 7)
		if strings.Count(got, Mark) != 1 || !strings.HasSuffix(got, Mark) || len(got) > 7 {
			t.Fatalf("Continued(%q) = %q", line, got)
		}
	}
}

// With no room for one unit beside the mark the text is cut unmarked, so a
// four-column limit is the first that shows a mark.
func TestNoRoomForTheMarkCutsSilently(t *testing.T) {
	count := Spec{Measure: utf8.RuneCountInString, Unit: Rune}
	for n, want := range map[int]string{0: "", 1: "П", 3: "Поб", 4: "П...", 5: "По..."} {
		if got := count.Fit("Победа", n); got != want {
			t.Errorf("Fit(%d) = %q, want %q", n, got, want)
		}
	}
}
