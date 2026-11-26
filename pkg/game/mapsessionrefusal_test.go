package game

import (
	"path/filepath"
	"testing"
)

// A source outside the multiplayer map choice, a name outside the store and
// a statistic past a byte are refused before any map opens.
func TestMapSessionRefusals(t *testing.T) {
	c := sessionCases()[1]
	f, _, stored := sessionApp(t, c)
	store := CharacterStore{Dir: filepath.Dir(stored)}
	row := secondCharacterFiles
	if _, err := f.newMapSession(store, row, c.file, "10.alm"); err == nil {
		t.Fatal("a campaign map was entered as a multiplayer map")
	}
	if _, err := f.newMapSession(store, row, "../"+c.file, c.mapAt); err == nil {
		t.Fatal("a path outside the store was read")
	}
	if _, err := storedFrom("x", nil, [4]uint32{256, 20, 20, 20}); err == nil {
		t.Fatal("a statistic of 256 was accepted")
	}
	if s, err := f.newMapSession(store, row, c.file, c.mapAt); err != nil || s.purse != c.purse {
		t.Fatalf("session %+v, %v", s, err)
	}
}
