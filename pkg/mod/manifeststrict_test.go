package mod

import (
	"strings"
	"testing"
)

const strictHead = "id = \"a\"\ntitle = \"A\"\nversion = \"1\"\n"

func TestManifestMalformedValueIsRefusedWithItsLine(t *testing.T) {
	_, err := ParseManifest([]byte(strictHead + "author = \"open\n"))
	if err == nil || !strings.Contains(err.Error(), "line 4: author") {
		t.Fatalf("got %v", err)
	}
	_, err = WriteManifest([]byte(strictHead+"author = \"open\n"), map[string]any{"author": "B"})
	if err == nil || !strings.Contains(err.Error(), "line 4: author") {
		t.Fatalf("writer got %v", err)
	}
}

func TestManifestLinesAfterTheFirstTableAreChecked(t *testing.T) {
	if _, err := ParseManifest([]byte(strictHead + "[extra]\nid = \"x\"\n")); err != nil {
		t.Fatal(err)
	}
	for _, tail := range []string{"[extra]\nnot a pair\n", "[extra]\n[extra]\n", "[[rows]]\n", "[open\n"} {
		_, err := ParseManifest([]byte(strictHead + tail))
		if err == nil || !strings.Contains(err.Error(), "line 4: ") && !strings.Contains(err.Error(), "line 5: ") {
			t.Fatalf("%q: got %v", tail, err)
		}
	}
}

func TestRowFileWriterKeepsAnUnchangedLineAndItsComment(t *testing.T) {
	src := "a = 'z'   # keep\n"
	out, err := RewriteValues([]byte(src), []ValueEdit{{Line: 1, Value: "z"}})
	if err != nil || string(out) != src {
		t.Fatalf("%q %v", out, err)
	}
}
