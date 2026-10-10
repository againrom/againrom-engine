package main

import (
	"testing"

	"againrom/pkg/game"
)

// -nomusic opens no music archive, so every request is silent; without it
// the front end names the game's archive.
func TestNoMusicOpensNoArchive(t *testing.T) {
	o, err := parse([]string{"-nomusic"})
	if err != nil || !o.noMusic {
		t.Fatalf("parse -nomusic: %+v, %v", o.noMusic, err)
	}
	dir := defaultInstall(t)
	for _, off := range []bool{false, true} {
		front, err := frontEnd(dir, options{assets: dir, noMusic: off}, game.OptionsStore{})
		if err != nil {
			t.Fatal(err)
		}
		if (front.MusicBank == nil) != off {
			t.Errorf("-nomusic %v: music bank %v", off, front.MusicBank)
		}
	}
}
