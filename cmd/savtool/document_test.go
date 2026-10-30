package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareNamesThePartThatDiffers(t *testing.T) {
	path := write(t, "game0000.sav", synthetic())
	other := filepath.Join(filepath.Dir(path), "other.sav")
	var out bytes.Buffer
	if err := run([]string{"set", "-out", other, "-label", "two", path}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run([]string{"compare", path, path}, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "differs") {
		t.Fatalf("a file differs from itself:\n%s", out.String())
	}
	out.Reset()
	if err := run([]string{"compare", path, other}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"body  equal", "label differs", "[0x0,0x3)"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("compare does not say %q:\n%s", want, out.String())
		}
	}
	for _, args := range [][]string{{"compare", path}, {"document"}} {
		if err := run(args, &out); err == nil {
			t.Fatalf("%v was accepted", args)
		}
	}
}

func TestEqualTextRanges(t *testing.T) {
	for _, c := range []struct {
		x, y string
		want string
	}{
		{"abc", "abc", "equal (3 bytes)"},
		{"abcd", "xbcy", "2 ranges: [0x0,0x1)61>78 [0x3,0x4)64>79"},
		{"ab", "abc", "1 ranges: [0x2,end)"},
	} {
		if got := equalText([]byte(c.x), []byte(c.y)); !strings.Contains(got, c.want) {
			t.Errorf("%q %q: %s", c.x, c.y, got)
		}
	}
}
