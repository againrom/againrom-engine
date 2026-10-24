package main

import "testing"

func TestStarterFieldsUseSharedTextModel(t *testing.T) {
	for _, field := range []struct {
		name string
		id   fieldID
	}{
		{"add-base", fAddBase}, {"mods", fModsDir}, {"executable", fAgainrom},
		{"volume", fVolume}, {"saves", fSaves}, {"mission", fMission},
		{"skill", fSkill}, {"stock", fShopStockMultiplier}, {"arguments", fExtra},
	} {
		t.Run(field.name, func(t *testing.T) {
			a := app{focus: field.id}
			a.typed("Qя\n")
			if got := *a.field(field.id); got != "Qя" {
				t.Fatal("typed", got)
			}
			a.backspace()
			if got := *a.field(field.id); got != "Q" {
				t.Fatal("backspace", got)
			}
			a.dirty = false
			a.typed("\r")
			if a.dirty {
				t.Fatal("rejected input changed dirty state")
			}
		})
	}
}
