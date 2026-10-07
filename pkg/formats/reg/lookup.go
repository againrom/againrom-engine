package reg

// Two-level typed accessors over a parsed Reg: a section, then a key inside
// it. Neither level goes deeper than that — the full tree stays exported
// (Reg.Root) for anything that needs to walk further.
//
// Matching is case-insensitive over the ASCII range only: A-Z (0x41-0x5A)
// folds with a-z (0x61-0x7A), and every other byte — including every byte
// >= 0x80 — compares as itself. 0xC0 and 0xFF) — see plan.md DD11.

// foldASCII maps 'A'-'Z' to 'a'-'z' and leaves every other byte, ASCII or
// not, unchanged.
func foldASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}

// nameEqualFold reports whether a and b are equal under the ASCII-only fold:
// equal length, and equal at every position after foldASCII. Whole names are
// compared, not a truncated prefix.
func nameEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if foldASCII(a[i]) != foldASCII(b[i]) {
			return false
		}
	}
	return true
}

// findChild returns the first child of n, in node-table order, whose Name
// matches name under the ASCII-only fold, or nil if none does. The format
// permits duplicate sibling names and assigns no meaning to one, so "first
// in table order" is the rule two runs agree on.
func findChild(n *Node, name string) *Node {
	for _, c := range n.Children {
		if nameEqualFold(c.Name, name) {
			return c
		}
	}
	return nil
}

// lookup resolves section among r.Root's children and then key among that
// child's children, both by findChild. It reports (nil, false) — never a
// panic — for a missing section, a section name that resolves to a value
// node rather than a directory, or a missing key. It does not check the
// resulting node's type: that is each accessor's own job, since a directory
// used as a value miss the same way a type mismatch does.
func (r *Reg) lookup(section, key string) (*Node, bool) {
	sec := findChild(r.Root, section)
	if sec == nil || !sec.Dir {
		return nil, false
	}
	k := findChild(sec, key)
	if k == nil {
		return nil, false
	}
	return k, true
}

// GetString looks up a string value at ("section", "key"). It reports false
// for a missing section, a missing key, a key of any other type — a
// directory included, since a directory's Type reads as TypeString (0) but
// Dir is checked first — and a section that is not itself a directory.
func (r *Reg) GetString(section, key string) (string, bool) {
	n, ok := r.lookup(section, key)
	if !ok || n.Dir || n.Type != TypeString {
		return "", false
	}
	return n.Str, true
}

// GetInt looks up an int32 value at ("section", "key"), with the same misses
// as GetString.
func (r *Reg) GetInt(section, key string) (int32, bool) {
	n, ok := r.lookup(section, key)
	if !ok || n.Dir || n.Type != TypeInt {
		return 0, false
	}
	return n.Int, true
}

// GetFloat looks up a float64 value at ("section", "key"), with the same
// misses as GetString.
func (r *Reg) GetFloat(section, key string) (float64, bool) {
	n, ok := r.lookup(section, key)
	if !ok || n.Dir || n.Type != TypeFloat {
		return 0, false
	}
	return n.Float, true
}

// GetIntArray looks up an int32 array value at ("section", "key"), with the
// same misses as GetString. Unlike the other three accessors it returns a
// fresh copy, since a slice is the one aliasing surface in this API and
// Node.Ints is the parser's own memory.
func (r *Reg) GetIntArray(section, key string) ([]int32, bool) {
	n, ok := r.lookup(section, key)
	if !ok || n.Dir || n.Type != TypeIntArray {
		return nil, false
	}
	out := make([]int32, len(n.Ints))
	copy(out, n.Ints)
	return out, true
}
