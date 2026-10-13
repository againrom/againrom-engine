package game

import "againrom/pkg/formats/alm"

func (mw *mapWorld) usePlayerColours(m *alm.Map) {
	if m == nil {
		return
	}
	mw.playerColours = make([]uint32, len(m.Groups))
	for i, p := range m.Groups {
		mw.playerColours[i] = p.Color
	}
	if mw.view != nil {
		mw.view.SetPlayerColours(mw.playerColours)
	}
}

func (mw *mapWorld) ownerShade(owner uint32) uint32 {
	if owner > 0 && uint64(owner) <= uint64(len(mw.playerColours)) {
		return mw.playerColours[owner-1] & 15
	}
	return owner & 15
}
