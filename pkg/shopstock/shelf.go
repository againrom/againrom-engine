package shopstock

// Stack is one game's stack-key rule for the one shelf insert. An item joins
// only an entry of an equal Key; Same (nil: the key decides) and Takes (nil:
// always) refine that; Add adds its units. An item that joins none opens
// Copy(item) (nil: the item) at the end, and Order (nil: none) files the
// shelf after an entry opened.
type Stack[I any, K comparable] struct {
	Key         func(I) K
	Same, Takes func(entry, it I) bool
	Add         func(entry, it I) I
	Copy        func(I) I
	Order       func([]I)
}

// Merges reports whether it joins entry under s.
func (s Stack[I, K]) Merges(entry, it I) bool {
	return s.Key(entry) == s.Key(it) && (s.Same == nil || s.Same(entry, it)) && (s.Takes == nil || s.Takes(entry, it))
}

// Find is the index of the first entry of list it joins under s, or -1.
func Find[I any, K comparable](list []I, s Stack[I, K], it I) int {
	for i := range list {
		if s.Merges(list[i], it) {
			return i
		}
	}
	return -1
}

// Insert files items onto list in order under s: each joins the first entry
// it merges with, else opens one. Order runs once, when any entry opened.
func Insert[I any, K comparable](list []I, s Stack[I, K], items ...I) []I {
	index := make(map[K][]int, len(list)+len(items))
	for i := range list {
		k := s.Key(list[i])
		index[k] = append(index[k], i)
	}
	opened := false
next:
	for _, it := range items {
		k := s.Key(it)
		for _, i := range index[k] {
			if s.Merges(list[i], it) {
				list[i] = s.Add(list[i], it)
				continue next
			}
		}
		if s.Copy != nil {
			it = s.Copy(it)
		}
		index[k] = append(index[k], len(list))
		list = append(list, it)
		opened = true
	}
	if opened && s.Order != nil {
		s.Order(list)
	}
	return list
}
