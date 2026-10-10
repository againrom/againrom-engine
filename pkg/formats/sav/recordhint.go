package sav

import "sync"

// recordValueHints holds, per class, how many scalar members its programme
// names: the size a record's Value map reaches when it is read or rebuilt.
// It is only a capacity; no answer depends on it.
var recordValueHints sync.Map

func recordValueHint(class string) int {
	if v, ok := recordValueHints.Load(class); ok {
		return v.(int)
	}
	n := programmeScalarMembers(programmes[class], 0)
	recordValueHints.Store(class, n)
	return n
}

func programmeScalarMembers(prog []step, depth int) int {
	if depth > 16 {
		return 0
	}
	n := 0
	for _, s := range prog {
		switch s.op {
		case stpRun:
			for _, m := range s.members {
				if m.Kind != KindCString && m.Kind != KindRaw {
					n++
				}
			}
		case stpClass:
			n += programmeScalarMembers(programmes[s.class], depth+1)
		case stpFlagged:
			n += 1 + programmeScalarMembers(s.sub, depth+1)
		}
	}
	return n
}
