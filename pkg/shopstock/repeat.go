package shopstock

// Repeat is the effect-repeat rule: how an effect of a kind an item already
// holds joins the item.
type Repeat uint8

const (
	// RepeatAppend appends every effect; a repeated kind is a further entry.
	RepeatAppend Repeat = iota
	// RepeatReplace gives the held entry of that kind the new value in place.
	RepeatReplace
)

// Held is the index of the entry of held that an effect of kind k replaces
// under rule, or -1 when the effect is appended.
func Held[E any, K comparable](held []E, kind func(E) K, k K, rule Repeat) int {
	if rule != RepeatReplace {
		return -1
	}
	for i, e := range held {
		if kind(e) == k {
			return i
		}
	}
	return -1
}
