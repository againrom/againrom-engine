package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestMovieHistoryColdStartPreservesPreferences1184(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	if err := os.WriteFile(path, []byte("TipsMode=0\nSoundVolume=23\nUnknown=value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s := OptionsStore{Path: path}
	for _, key := range []string{"intro", "m20", "intro"} {
		if err := s.EncounterCutscene(key); err != nil {
			t.Fatal(err)
		}
	}
	cold := OptionsStore{Path: path}
	seen, err := cold.EncounteredCutscenes()
	if err != nil || !reflect.DeepEqual(seen, []string{"intro", "m20"}) {
		t.Fatal(seen, err)
	}
	m, _ := cold.readAll()
	if m["TipsMode"] != "0" || m["SoundVolume"] != "23" || m["Unknown"] != "value" {
		t.Fatal("history lost unrelated preferences", m)
	}
	for _, bad := range []string{"../intro", "m20/01.smk", "intro\nSoundVolume=0", "logos", "m020", "M20"} {
		if s.EncounterCutscene(bad) == nil {
			t.Fatal("invalid group accepted", bad)
		}
	}
	if err := (OptionsStore{Path: t.TempDir()}).EncounterCutscene("intro"); err == nil {
		t.Fatal("write error hidden")
	}
}

func TestFemaleHumanVoiceKeepsAttackAndOtherActors1184(t *testing.T) {
	mw := &mapWorld{sounds: map[int32]UnitSound{24: {Slots: []int32{510, 211, 222, 223, 241}}}, figures: map[sim.EntityID]figureID{1: {Dir: data.FigureDirWomanMage}, 2: {Dir: data.FigureDirManMage}, 4: {Dir: data.FigureDirWomanMage, Horse: true}}}
	for _, tc := range []struct {
		id    sim.EntityID
		voice string
	}{{1, "f_mage"}, {2, "m_mage"}, {3, ""}, {4, "f_mage"}} {
		if got := mw.voiceBank(sim.Entity{ID: tc.id, TypeID: 24, Class: 24}); got != tc.voice {
			t.Fatal("actor", tc.id, "voice", got, "want", tc.voice)
		}
	}
	if got := soundSlots(mw.sounds, 24); !reflect.DeepEqual(got, []int{510, 211, 222, 223, 241}) {
		t.Fatal("the class array changed", got)
	}
	if !reflect.DeepEqual(mw.sounds[24].Slots, []int32{510, 211, 222, 223, 241}) {
		t.Fatal("shared registry was mutated")
	}
}
