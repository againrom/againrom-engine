package ui

import (
	"os"
	"testing"
)

// firstMusic is the first game's music description, read from the file the
// game package embeds.
var firstMusic = func() *MusicDescription {
	data, err := os.ReadFile("../game/musics/rom1.json")
	if err != nil {
		panic(err)
	}
	d, err := DecodeMusic(data)
	if err != nil {
		panic(err)
	}
	return d
}()

func TestDecodeMusicRefusesUnknownRulesAndNames(t *testing.T) {
	for name, doc := range map[string]string{
		"list":     `{"music":"x","archive":{"path":"m.res"},"tracks":[],"list":"random","silence":"clear","unlisted":"keep","scenes":{},"towns":{},"areas":null}`,
		"silence":  `{"music":"x","archive":{"path":"m.res"},"tracks":[],"list":"ordered","silence":"mute","unlisted":"keep","scenes":{},"towns":{},"areas":null}`,
		"unlisted": `{"music":"x","archive":{"path":"m.res"},"tracks":[],"list":"ordered","silence":"pause","unlisted":"stop","scenes":{},"towns":{},"areas":null}`,
		"scene":    `{"music":"x","archive":{"path":"m.res"},"tracks":[],"list":"ordered","silence":"pause","unlisted":"keep","scenes":{"hall":{"tracks":[]}},"towns":{},"areas":null}`,
		"track":    `{"music":"x","archive":{"path":"m.res"},"tracks":["a.wav"],"list":"ordered","silence":"pause","unlisted":"keep","scenes":{"menu":{"tracks":["b.wav"]}},"towns":{},"areas":null}`,
		"town":     `{"music":"x","archive":{"path":"m.res"},"tracks":["a.wav"],"list":"ordered","silence":"pause","unlisted":"keep","scenes":{},"towns":{"1":{"track":"b.wav"}},"areas":null}`,
		"period":   `{"music":"x","archive":{"path":"m.res"},"tracks":[],"list":"ordered","silence":"pause","unlisted":"keep","scenes":{},"towns":{},"areas":{"period":0}}`,
		"field":    `{"music":"x","archive":{"path":"m.res"},"tracks":[],"list":"ordered","silence":"pause","unlisted":"keep","scenes":{},"towns":{},"areas":null,"extra":1}`,
	} {
		if _, err := DecodeMusic([]byte(doc)); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
}
