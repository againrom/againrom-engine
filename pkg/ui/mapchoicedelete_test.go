package ui

import (
	"testing"
	"time"
)

// The map choice deletes no row: the Delete key asks nothing even while the
// saves' delete seams would accept the row's name, and Delete stays off.
func TestMapChoiceDeletesNoRow(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for name, a := range multiplayerApps(t) {
		var prepared []string
		a.SetSaveDelete(func(string) bool { return true }, func(token string) (func() error, error) {
			prepared = append(prepared, token)
			return func() error { return nil }, nil
		})
		a.SetMapChoiceWords(MapChoiceWords{Title: "Select Map", OK: "OK", Cancel: "Cancel", Empty: "No maps"})
		a.OpenMapChoice(mapChoiceSampleItems(), ScreenMenu, func(MapChoiceResult) {})
		a.step(appInput{}, now)
		if err := a.HeadlessKey("delete"); err != nil {
			t.Fatal(err)
		}
		v := a.flow.loadView()
		if len(prepared) != 0 || a.flow.loadUI.confirm || !a.flow.pageButtonDisabled(v, loadDeleteButton) {
			t.Fatalf("%s: Delete key on the map choice prepared %v, confirm %v", name, prepared, a.flow.loadUI.confirm)
		}
		if a.Screen() != ScreenLoad {
			t.Fatalf("%s: Delete key left the map choice for %v", name, a.Screen())
		}
	}
}
