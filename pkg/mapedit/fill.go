package mapedit

// FillMode names how a tile stroke chooses the word of a cell whose four
// corners all show the painted class: the centre of the brush. The words that
// join the painted class to its neighbours (edges, corners, tips) are chosen
// by the stroke's transition rules and ignore the fill.
type FillMode uint8

const (
	FillPattern FillMode = iota // the cell's position picks among the class's look-alike words
	FillVariant                 // every centre cell takes one look-alike word, Fill.Variant
	FillRandom                  // each centre cell takes a look-alike word Fill.Draw picks
)

// Fill is the centre fill of a tile stroke. Variant indexes Variants(class)
// and is clamped to it. Draw is the random source of FillRandom: it answers a
// number in 0..n-1. The caller owns the source and makes a fresh one for each
// stroke, so one source gives one interior; a random fill without a source is
// the pattern.
type Fill struct {
	Mode    FillMode
	Variant int
	Draw    func(n int) int
}

// Variants lists the look-alike words a pure cell of class c can show, in the
// order the editor numbers them; nil for a class with none.
func Variants(c uint8) []uint16 {
	var ws []word
	group := ownStrip(c).group
	switch c {
	case ClassWater:
		ws, group = waterLayout.pureP, waterGroup
	case ClassLand:
		ws = landLayout.pureS
	default:
		ws = landLayout.pureP
	}
	out := make([]uint16, len(ws))
	for i, w := range ws {
		out[i] = compose(group, w)
	}
	return out
}

// fillWord is the word a centre cell of class c takes under fill; ok is
// false for the pattern, which leaves the choice to the stroke's own rules.
func (f *TileFix) fillWord(c uint8) (uint16, bool) {
	ws := Variants(c)
	if len(ws) == 0 {
		return 0, false
	}
	switch f.Fill.Mode {
	case FillVariant:
		return ws[max(0, min(f.Fill.Variant, len(ws)-1))], true
	case FillRandom:
		if f.Fill.Draw == nil {
			return 0, false
		}
		return ws[max(0, min(f.Fill.Draw(len(ws)), len(ws)-1))], true
	}
	return 0, false
}
