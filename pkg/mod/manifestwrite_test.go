package mod

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestWriteManifestPreservesEveryOtherByte(t *testing.T) {
	for _, key := range manifestKeyNames() {
		t.Run(key, func(t *testing.T) {
			value := any("new \"value\"\n\\\t")
			old := "'old'"
			if key == "api" {
				value, old = 1, "2"
			}
			if key == "requires" || key == "conflicts" || key == "load-after" || key == "applies-to" {
				value, old = []string{"a", "b"}, "['old']"
			}
			prefix := "\xef\xbb\xbf# comment\r\nunknown = 'kept'\r\n\r\n\t" + key + "\t=  "
			suffix := "  # own comment\r\nentry = 'kept.star'\r\n[custom]\nvalue = 7\n"
			if key == "entry" {
				suffix = "  # own comment\r\n[custom]\nentry = 'table entry'\n"
			}
			source := []byte(prefix + old + suffix)
			out, err := WriteManifest(source, map[string]any{key: value})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(out, []byte(prefix)) || !bytes.HasSuffix(out, []byte(suffix)) {
				t.Fatalf("changed surrounding bytes:\n%q", out)
			}
			same, err := WriteManifest(out, map[string]any{key: value})
			if err != nil || !bytes.Equal(out, same) {
				t.Fatalf("repeat changes bytes: %q %v", same, err)
			}
		})
	}
}

func TestWriteManifestAddsKeysBeforeTablesAndRoundTrips(t *testing.T) {
	values := map[string]any{"id": "sample", "title": "Sample", "version": "2.1", "author": "Writer", "description": "Two\nlines", "api": 1,
		"applies-to": []string{"rom1-en", "rom2-ru"}, "requires": []string{"rules>=1.2"}, "conflicts": []string{"other"}, "load-after": []string{"rules"}}
	out, err := WriteManifest([]byte("# source\r\n[ignored]\r\nunknown = 'kept'\r\n"), values)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseManifest(out)
	if err != nil {
		t.Fatal(err)
	}
	want := Manifest{ID: "sample", Title: "Sample", Version: "2.1", Author: "Writer", Description: "Two\nlines", API: 1,
		AppliesTo: []string{"rom1-en", "rom2-ru"}, Requires: []string{"rules>=1.2"}, Conflicts: []string{"other"}, LoadAfter: []string{"rules"}}
	if !reflect.DeepEqual(m, want) || strings.Contains(string(out), "entry =") {
		t.Fatalf("%+v\n%s", m, out)
	}
	if !bytes.HasSuffix(out, []byte("[ignored]\r\nunknown = 'kept'\r\n")) {
		t.Fatalf("table moved: %q", out)
	}
}

func TestWriteManifestReplacesMultilineList(t *testing.T) {
	source := []byte("requires = [\r\n  'a', # member\r\n  'b',\r\n] # retained\r\nentry = 'logic.star'\r\n")
	out, err := WriteManifest(source, map[string]any{"requires": []string{"core>=1"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "requires = [\"core>=1\"] # retained\r\nentry = 'logic.star'\r\n" {
		t.Fatalf("%q", out)
	}
}

// The list writer reads its span through the same scanner as the readers:
// an unchanged list keeps its bytes, comment lines included, and a changed
// list ends at its first depth-zero bracket with the text after it kept.
func TestWriteManifestListSpanUsesTheReaderScanner(t *testing.T) {
	source := "applies-to = [ # first\n  \"a\", # one\n  \"b\"\n] # done\nentry = \"x.star\"\n"
	same, err := WriteManifest([]byte(source), map[string]any{"applies-to": []string{"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(same) != source {
		t.Fatalf("unchanged list rewritten:\n%q", same)
	}
	changed, err := WriteManifest([]byte(source), map[string]any{"applies-to": []string{"c"}})
	if err != nil {
		t.Fatal(err)
	}
	want := "applies-to = [\"c\"] # done\nentry = \"x.star\"\n"
	if string(changed) != want {
		t.Fatalf("changed list:\n%q\nwant\n%q", changed, want)
	}
}
