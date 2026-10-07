package sav

import (
	"bytes"
	"reflect"
	"testing"
)

func TestCityParticipantRenameIsExplicitAndDetached(t *testing.T) {
	f, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	update := CityUpdate{Label: []byte("participant"), Money: 123}
	for _, c := range p.Roster() {
		update.Characters = append(update.Characters, CityCharacterUpdate{Identity: c.Identity, Name: c.Name,
			Stats: c.Stats, SkillLevels: c.SkillLevels, SkillXP: c.SkillXP, Experience: c.Experience})
	}
	baseline, err := p.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	name := "NamedHero"
	update.ParticipantName = &name
	renamed, err := p.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	before, err := DecodeDocumentData(baseline)
	if err != nil {
		t.Fatal(err)
	}
	after, err := DecodeDocumentData(renamed)
	if err != nil {
		t.Fatal(err)
	}
	player := &before.Objects[before.Players[0]-1]
	for i := range player.Texts {
		if player.Texts[i].Name == "Name" {
			player.Texts[i].Value = name
		}
	}
	if !reflect.DeepEqual(before, after) || p.ParticipantName() != "Player" {
		t.Fatal("renaming changed other state or mutated provenance")
	}
	update.ParticipantName = nil
	repeated, err := p.Marshal(update)
	if err != nil || !bytes.Equal(baseline, repeated) {
		t.Fatalf("nil rename changed the saved participant: %v", err)
	}
	for _, invalid := range []string{"", "one\x00two"} {
		update.ParticipantName = &invalid
		if _, err := p.Marshal(update); err == nil {
			t.Fatalf("accepted invalid participant name %q", invalid)
		}
	}
}
