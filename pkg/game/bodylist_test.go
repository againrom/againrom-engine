package game_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

func TestHeroPictureAddressIsUnderMainsTextTree(t *testing.T) {
	const want = "main/text/heropicture.txt"
	if game.HeroPictureAddress != want {
		t.Fatalf("HeroPictureAddress = %q, want %q", game.HeroPictureAddress, want)
	}
}

// A shipped entry reads back parsed, and every way of not finding one
// answers nil, false rather than an error — there is no error for a caller
// to log, fall back on, or fail a front end's startup with.
func TestReadBodyListIsSilentWhenNothingShips(t *testing.T) {
	present := selSource{name: game.HeroPictureAddress, data: []byte("unarmed\nswordsman\n")}

	got, ok := game.ReadBodyList(present)
	if !ok {
		t.Fatal("a shipped entry: refused, want it read")
	}
	want := data.BodyList{"unarmed", "swordsman"}
	if len(got) != len(want) {
		t.Fatalf("ReadBodyList = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %q, want %q", i, got[i], want[i])
		}
	}

	for _, tc := range []struct {
		name string
		src  terrain.EntrySource
	}{
		{"no source", nil},
		{"entry absent", selSource{}},
		{"a different entry present", selSource{name: "main/text/other.txt", data: []byte("x")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := game.ReadBodyList(tc.src)
			if ok || got != nil {
				t.Fatalf("ReadBodyList = %#v, %v; want nil, false", got, ok)
			}
		})
	}
}
