package ui

import "testing"

func TestGameMenuExitFollowsTheMapsExitValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		exit GameMenuExit
		want bool
	}{{"main", GameMenuExitToMain, false}, {"leaves", GameMenuExitLeavesMap, true}} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t, appRows(1), okLoader(t))
			asked := 0
			a.flow.menuContext = GameMenuContext{Exit: tc.exit}
			a.flow.menuBack = ScreenMap
			a.flow.advance = func(...NoticeAction) (NoticeDest, string, MapOpener) {
				asked++
				return NoticeToCharacters, "", nil
			}
			if got := a.flow.leaveSession(); got != tc.want || (asked == 1) != tc.want {
				t.Fatalf("leaveSession = %v after %d asks, want %v", got, asked, tc.want)
			}
		})
	}
}
