package data

import "strings"

// The ONE prefix walk a shape table and a material table are both searched
// by, and the ONE re-attachment a shield's or an armour's own name puts back
// afterwards (FR-2a, FR-2b). Nothing here composes an item code or opens a
// collection; this file's whole job is turning a name into an index, a
// matched entry name and what is left of the name.

// takePrefix is FR-2a's descending substring walk, replacing the
// longest-word-boundary-prefix match this package carried before 0128.
//
// t IS WALKED FROM ITS LAST ENTRY DOWN TO ITS FIRST — the opposite direction
// of ascending/descending in defsearch.go, which choose a record by an exact
// key over a different interface; this chooses by which entry's NAME shows
// up first counting from the table's own end. The first entry whose
// non-empty name is a SUBSTRING of s ANYWHERE — strings.Index, not a prefix
// test and not a word-boundary test — wins, and its own index is what a
// caller composing an item code reads back.
//
// THE REBUILD IS POSITIONAL, NOT A TRIM (FR-2a): s[:found] + s[len(name)+1:].
// There is no `found` term in the second slice — that is the original's own
// arithmetic, carried forward rather than corrected, because correcting it
// would be inventing a fourth rule this package does not own. When the match
// sits at the front of s, found is 0 and the two slices are exactly what a
// plain trim would also produce, which is the ordinary case and the only one
// weapon.go's own fixtures have ever exercised. When it does not, the second
// slice starts len(name)+1 bytes from the FRONT of s rather than one past
// where the match actually ended, so a name authored out of order comes back
// MANGLED, not refused — the contract FR-2a states, not an accident of it.
// Both slices are guarded against running past len(s): s[:found] cannot,
// found being strings.Index's own answer, and the second slice is omitted
// entirely once len(name)+1 >= len(s).
//
// NO MATCH LEAVES s WHOLE AND ANSWERS INDEX 0, THE EMPTY NAME: an absent
// table (t == nil) and a table with no entry that occurs in s both take this
// arm. Index 0 is the shape default. Materials deliberately replace this
// answer with index 15 in materialPrefix below.
func takePrefix(s string, t ScaleTable) (index int, name string, rest string) {
	if t == nil {
		return 0, "", s
	}
	for i := t.Len() - 1; i >= 0; i-- {
		n := t.EntryName(i)
		if n == "" {
			continue
		}
		found := strings.Index(s, n)
		if found < 0 {
			continue
		}
		out := s[:found]
		if tail := len(n) + 1; tail < len(s) {
			out += s[tail:]
		}
		return i, n, out
	}
	return 0, "", s
}

// materialPrefix applies the material resolver's decoded missing-word default.
// Shapes and materials share takePrefix's search, but not its no-match value:
// the original leaves a missing shape at index 0 and writes material index 15
// (the shipped `None` row). ITEM-PICT-048 establishes that asymmetry and the
// shipped art population depends on it.
func materialPrefix(s string, t ScaleTable) (index int, name string, rest string) {
	index, name, rest = takePrefix(s, t)
	if name == "" {
		index = 15
	}
	return index, name, rest
}

const castSpellPrefix = "castSpell="

func takeCastSpell(suffix string) (token string, level int32, ok bool) {
	if !strings.HasPrefix(suffix, "{") || !strings.HasSuffix(suffix, "}") {
		return "", 0, false
	}
	body := suffix[1 : len(suffix)-1]
	if !strings.HasPrefix(body, castSpellPrefix) {
		return "", 0, false
	}
	body = body[len(castSpellPrefix):]

	i := strings.IndexByte(body, ':')
	if i < 0 {
		return "", 0, false
	}
	token, levelText := body[:i], body[i+1:]
	if levelText == "" {
		return "", 0, false
	}

	var v int32
	for j := 0; j < len(levelText); j++ {
		d := levelText[j]
		if d < '0' || d > '9' {
			return "", 0, false
		}
		v = v*10 + int32(d-'0')
	}
	return token, v, true
}

// impliedShapePrefix is FR-2b: after a shield's or an armour's material word
// is taken, a material NAME containing `Leather` puts `Soft ` back on the
// front of what is left, and one containing `Wood` puts `Wooden ` back.
//
// THE TWO ARMS ARE MUTUALLY EXCLUSIVE HERE, and that is a fact about the
// shipped material names and not a rule this function enforces: no shipped
// material name contains both `Leather` and `Wood`, so the order the two
// are tested in never matters and a name that could take either arm never
// arises for this build to have to choose between them.
//
// SUBJECT IS LEFT-TRIMMED BEFORE THE WORD GOES BACK ON (FR-2b). The routine
// this build reads off the image calls one further no-argument method on the
// subject immediately before it prepends the implied word. The inference is
// what takes the shipped corpus from 902 of 909 equipment cells resolved to
// 908 — every one of the six is one authored name carrying a double space,
// which FR-2a's positional rebuild preserves. The trim is why the
// re-attached word is followed by exactly one space however many the rebuild
// left standing. TEXT-PATTERN-020.
func impliedShapePrefix(materialName, subject string) string {
	switch {
	case strings.Contains(materialName, "Leather"):
		return "Soft " + strings.TrimLeft(subject, " ")
	case strings.Contains(materialName, "Wood"):
		return "Wooden " + strings.TrimLeft(subject, " ")
	default:
		return subject
	}
}
