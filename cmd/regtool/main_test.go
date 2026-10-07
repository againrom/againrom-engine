package main

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// TestDumpRender exercises render (SC-9) over one synthetic registry parsed
// by reg.Parse — no archive, no file, no install.
//
// The fixture is built so the whole rendered output can be compared against
// one hand-written expected string: that pins indentation (three levels),
// quoting, the 8/9-element abbreviation boundary and the empty-array case,
// every stated form for a double, and the AC-10 byte convention over a name
// and a string value with bytes outside 0x20-0x7E, in a single assertion.
func TestDumpRender(t *testing.T) {
	// Bytes >= 0x80 are written as a byte slice, never as literal non-ASCII
	// text (golden rule 2): "Hi" + 0x80 + 0xff, and "A" + 0x80 + 0xff.
	highName := string([]byte{0x48, 0x69, 0x80, 0xff})
	highStr := string([]byte{0x41, 0x80, 0xff})

	tree := []synth.RegNode{
		{Name: "Global", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Title", Kind: 0x00, Str: "rage"},
			{Name: "Count", Kind: 0x02, Int: 34},
			{Name: "Nested", Kind: 0x01, Children: []synth.RegNode{
				{Name: "Deep", Kind: 0x01, Children: []synth.RegNode{
					{Name: "Leaf", Kind: 0x02, Int: 1},
				}},
			}},
			{Name: "Arr8", Kind: 0x06, Ints: []int32{0, 1, 2, 3, 4, 5, 6, 7}},
			{Name: "Arr9", Kind: 0x06, Ints: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8}},
			{Name: "ArrEmpty", Kind: 0x06, Ints: nil},
			{Name: "Zero", Kind: 0x04, Float: 0.0},
			{Name: "One", Kind: 0x04, Float: 1.0},
			{Name: "Half", Kind: 0x04, Float: 1.5},
			{Name: "NotANumber", Kind: 0x04, Float: math.NaN()},
			{Name: "PosInf", Kind: 0x04, Float: math.Inf(1)},
			{Name: highName, Kind: 0x00, Str: highStr},
		}},
	}

	r, err := reg.Parse(synth.Reg(1, tree))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var buf bytes.Buffer
	if err := render(&buf, r); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()

	want := strings.Join([]string{
		`Global:`,
		`  Title = "rage"`,
		`  Count = 34`,
		`  Nested:`,
		`    Deep:`,
		`      Leaf = 1`,
		`  Arr8 = [0 1 2 3 4 5 6 7]`,
		`  Arr9 = [0 1 2 3 4 5 6 7 ... (9 total)]`,
		`  ArrEmpty = []`,
		`  Zero = 0.0`,
		`  One = 1.0`,
		`  Half = 1.5`,
		`  NotANumber = NaN`,
		`  PosInf = +Inf`,
		`  Hi\x80\xff = "A\x80\xff"`,
		``, // trailing newline after the last line
	}, "\n")

	if out != want {
		t.Fatalf("render output mismatch:\n--- got ---\n%s--- want ---\n%s", out, want)
	}

	for _, c := range []struct{ want, line string }{
		{"root's children at column 0", "Global:"},
		{"a nested directory's children two spaces deeper", "  Nested:"},
		{"a directory nested two levels deep four spaces deeper", "    Deep:"},
	} {
		if !strings.Contains(out, "\n"+c.line+"\n") && !strings.HasPrefix(out, c.line+"\n") {
			t.Errorf("%s: line %q not found at the expected indentation in:\n%s", c.want, c.line, out)
		}
	}
	if !strings.Contains(out, "Global:\n") {
		t.Error("no directory line ends with ':'")
	}
	if !strings.Contains(out, " = ") {
		t.Error("no value line contains ' = '")
	}

	// (b) quoting.
	if !strings.Contains(out, `"rage"`) {
		t.Error(`string value "rage" is not wrapped in quotes`)
	}
	if strings.Contains(out, `"34"`) {
		t.Error("int32 value 34 was quoted, want unquoted")
	}

	// (c) abbreviation, the 8-vs-9 boundary and the empty array.
	if !strings.Contains(out, "Arr8 = [0 1 2 3 4 5 6 7]") {
		t.Error("an 8-element array did not render in full")
	}
	if !strings.Contains(out, "Arr9 = [0 1 2 3 4 5 6 7 ... (9 total)]") {
		t.Error("a 9-element array did not abbreviate to the first eight, \"...\", \"(9 total)\"")
	}
	if !strings.Contains(out, "ArrEmpty = []") {
		t.Error("an empty array did not render \"[]\"")
	}

	// (d) doubles.
	for _, c := range []string{"Zero = 0.0", "One = 1.0", "Half = 1.5", "NotANumber = NaN", "PosInf = +Inf"} {
		if !strings.Contains(out, c) {
			t.Errorf("double rendering: %q not found in output", c)
		}
	}
	if strings.Contains(out, "Zero = 0\n") || strings.Contains(out, "One = 1\n") {
		t.Error("0.0 or 1.0 rendered without the forced .0 suffix")
	}

	// (e) AC-10: printable bytes verbatim, every other byte \xNN (two
	// lower-case hex digits), no code page, whole output valid UTF-8 and no
	// byte above 0x7E.
	if !strings.Contains(out, `Hi\x80\xff = "A\x80\xff"`) {
		t.Error("bytes outside 0x20-0x7E were not escaped as \\xNN with printable bytes verbatim")
	}
	if !utf8.ValidString(out) {
		t.Error("rendered output is not valid UTF-8")
	}
	for i := 0; i < len(out); i++ {
		if out[i] > 0x7e {
			t.Fatalf("rendered output contains byte %#x above 0x7E at offset %d", out[i], i)
		}
	}
}
