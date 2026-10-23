// Package shopstock is the one shop stock generator of both games. A
// location's stock is a list of categories; each category draws its candidate
// list, takes its lead items, then a draw count of items, each made of one
// picked candidate (the enchant is part of making it). Items that take no
// draw are placed after every category, and the count multiplier's further
// rolls come last. Each game states its rules as a Profile; this package
// holds no game's rule.
package shopstock

// Source answers one draw over n choices, 0..n-1.
type Source interface {
	Intn(n int) int
}

// Bound ends a category's draws: Attempts bounds every attempt and Failures
// the failed ones, each a function of the draw count. A nil function is no
// bound.
type Bound struct {
	Attempts, Failures func(n int) int
}

// Category is one category of a location's stock: C is a candidate and I
// an item.
type Category[C, I any] struct {
	// Candidates is the category's candidate list; it may draw.
	Candidates func(src Source) []C
	// Lead is the list the draws start from; it may draw. Nil starts empty.
	Lead func(cands []C, src Source) []I
	// Count is how many items the draws take. It is asked only when the
	// category has candidates, and it may draw.
	Count func(src Source) int
	// Bound ends the draws of the first roll; Again ends each further roll
	// of the multiplier.
	Bound, Again Bound
	// Redraw marks a candidate the pick draws again, up to Redraws times; a
	// pick still marked fails the attempt. Nil marks none.
	Redraw  func(c C) bool
	Redraws int
	// Make is the item an attempt makes of candidate c; false fails the
	// attempt.
	Make func(c C, src Source) (I, bool)
	// Insert files an item into the list; nil appends.
	Insert func(list []I, it I) []I
	// Finish completes the list after the first roll; it may draw. Nil keeps
	// the list.
	Finish func(list []I, src Source) []I
}

// Multiplier is the count multiplier step: with Factor above one, each
// category whose first roll took items rolls Factor-1 more times, each roll
// taking as many items as the first took, on the draws of Source; Join adds a
// category's further items to its list. A Factor of zero or one is off.
type Multiplier[I any] struct {
	Factor int
	Source func() Source
	Join   func(category int, list, more []I) []I
}

// Profile is one game's stock rules for one location: its categories in
// draw order, the items placed without a draw, the multiplier, and the
// effect-repeat rule its enchant applies.
type Profile[C, I any] struct {
	Categories []Category[C, I]
	// Place adds the items that take no draw to the stock; nil adds none.
	Place      func(stock [][]I)
	Multiplier Multiplier[I]
	Repeat     Repeat
}

// Generate draws the whole stock of p on src: one list per category.
func Generate[C, I any](p Profile[C, I], src Source) [][]I {
	stock := make([][]I, len(p.Categories))
	cands := make([][]C, len(p.Categories))
	took := make([]int, len(p.Categories))
	for k, cat := range p.Categories {
		cands[k] = cat.Candidates(src)
		var list []I
		if cat.Lead != nil {
			list = cat.Lead(cands[k], src)
		}
		if len(cands[k]) > 0 {
			list, took[k] = roll(cat, cat.Bound, cands[k], cat.Count(src), list, src)
		}
		if cat.Finish != nil {
			list = cat.Finish(list, src)
		}
		stock[k] = list
	}
	if p.Place != nil {
		p.Place(stock)
	}
	m := p.Multiplier
	if m.Factor <= 1 {
		return stock
	}
	again := m.Source()
	for k, cat := range p.Categories {
		if len(cands[k]) == 0 || took[k] == 0 {
			continue
		}
		var more []I
		for i := 1; i < m.Factor; i++ {
			more, _ = roll(cat, cat.Again, cands[k], took[k], more, again)
		}
		stock[k] = m.Join(k, stock[k], more)
	}
	return stock
}

// roll makes up to n items of cands onto list within bound, and answers the
// list and how many items it took.
func roll[C, I any](cat Category[C, I], bound Bound, cands []C, n int, list []I, src Source) ([]I, int) {
	attempts, failures := -1, -1
	if bound.Attempts != nil {
		attempts = bound.Attempts(n)
	}
	if bound.Failures != nil {
		failures = bound.Failures(n)
	}
	done, failed := 0, 0
	for done < n && (attempts < 0 || done+failed < attempts) && (failures < 0 || failed < failures) {
		c := cands[src.Intn(len(cands))]
		if cat.Redraw != nil {
			for i := 0; cat.Redraw(c) && i < cat.Redraws; i++ {
				c = cands[src.Intn(len(cands))]
			}
			if cat.Redraw(c) {
				failed++
				continue
			}
		}
		it, ok := cat.Make(c, src)
		if !ok {
			failed++
			continue
		}
		if cat.Insert != nil {
			list = cat.Insert(list, it)
		} else {
			list = append(list, it)
		}
		done++
	}
	return list, done
}
