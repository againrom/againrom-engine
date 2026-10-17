// Package textfit cuts a text to a width and marks the cut. It is the one text
// fit of every screen: the caller states how a text is measured, the unit a
// cut removes and the side it cuts, and the mark is placed once.
package textfit

// Mark is what a cut leaves: three full stops, which every font the engine
// draws holds.
const Mark = "..."

// Unit is the step a cut removes.
type Unit int

const (
	// Rune cuts UTF-8 text between runes, never inside one.
	Rune Unit = iota
	// Byte cuts a font's own byte alphabet, where one byte is one glyph.
	Byte
)

// Side is the end a cut removes and marks.
type Side int

const (
	// End keeps the head of the text and marks its end.
	End Side = iota
	// Start keeps the tail and marks its start: the end of a path is the part
	// that tells folders apart.
	Start
)

// Spec is how one screen fits its text.
type Spec struct {
	Measure func(string) int // the width of a text in the screen's own unit
	Unit    Unit
	Side    Side
}

// Fit is s when it fits max. Otherwise it is the longest whole-unit part of s
// on the kept side that fits max with Mark on the cut side. When not one unit
// fits beside Mark, it is the longest part that fits, unmarked.
func (f Spec) Fit(s string, max int) string {
	if f.Measure(s) <= max {
		return s
	}
	return f.cut(s, max)
}

// Continued fits s, a part of a longer text: it is marked whether or not s
// itself fits, and the mark is placed once.
func (f Spec) Continued(s string, max int) string { return f.cut(s, max) }

func (f Spec) cut(s string, max int) string {
	cuts := f.boundaries(s)
	part := func(n int) string { // the kept part of n units
		if f.Side == Start {
			return s[cuts[len(cuts)-1-n]:]
		}
		return s[:cuts[n]]
	}
	marked := func(p string) string {
		if f.Side == Start {
			return Mark + p
		}
		return p + Mark
	}
	units := len(cuts) - 1
	for n := units; n >= 1; n-- {
		if t := marked(part(n)); f.Measure(t) <= max {
			return t
		}
	}
	for n := units; n >= 1; n-- {
		if t := part(n); f.Measure(t) <= max {
			return t
		}
	}
	return ""
}

// boundaries are the byte offsets between units of s, 0 and len(s) included.
func (f Spec) boundaries(s string) []int {
	out := make([]int, 0, len(s)+1)
	if f.Unit == Byte {
		for i := 0; i <= len(s); i++ {
			out = append(out, i)
		}
		return out
	}
	for i := range s {
		out = append(out, i)
	}
	return append(out, len(s))
}
