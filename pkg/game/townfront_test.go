package game

import "sync"

// townFronts maps a fixture screen to its front end.
var townFronts sync.Map

func registerTownFront(s *townScreen, f *FrontEnd) *townScreen {
	townFronts.Store(s, f)
	return s
}

func frontOf(s *townScreen) *FrontEnd {
	f, ok := townFronts.Load(s)
	if !ok {
		panic("town screen was not registered with a front end")
	}
	return f.(*FrontEnd)
}
