package mod

import (
	"strings"
	"testing"
)

func TestOnePairWriterKeepsAnUnchangedLineAndItsComment(t *testing.T) {
	src := "a = \"x\"   # keep\nb = [\n  \"p\", # one\n  \"q\",\n] # after\nc = 'z' # also\n"
	for _, at := range []struct {
		n     int
		value string
	}{{0, `"x"`}, {1, `["p", "q"]`}, {5, `"z"`}} {
		lines := strings.SplitAfter(src, "\n")
		if _, err := rewritePair(lines, at.n, 0, at.value); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(lines, ""); got != src {
			t.Fatalf("unchanged line %d moved: %q", at.n, got)
		}
	}
	lines := strings.SplitAfter(src, "\n")
	last, err := rewritePair(lines, 1, 0, `["r"]`)
	if err != nil || last != 4 || strings.Join(lines, "") != "a = \"x\"   # keep\nb = [\"r\"] # after\nc = 'z' # also\n" {
		t.Fatalf("%d %v %q", last, err, strings.Join(lines, ""))
	}
}

func TestOnePairWriterRefusesAMalformedOldValue(t *testing.T) {
	lines := []string{"a = \"open\n"}
	if _, err := rewritePair(lines, 0, 0, `"x"`); err == nil || !strings.HasPrefix(err.Error(), "a: ") {
		t.Fatalf("got %v", err)
	}
}
