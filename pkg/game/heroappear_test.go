package game_test

import (
	"bytes"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
)

// The party member's drawn class, derived from the body he wears (AC-7,
// AC-11, AC-12, plan SC-5, SC-9, SC-10, AC-6).
//
// 0085 authored the body name outright and derived only the class from it;
// 0105 retires that authored term — PartyBody is deleted, not kept as a
// fallback — and derives the name too, off the weapon's own definition row
// and the shipped list. What this file still tests is the wiring from a
// resolved weapon to a drawn class, now through two derivations chained
// rather than one.

// AC-7 (0085) / AC-6 (0105) — a party member holding a weapon whose row a
// (invented, SC-3) list names is NOT drawn as the bare-handed fallback: the
// class the law derives for the row's own name is a matched arm, not
// HeroUnmatchedClass.
func TestThePartyMemberIsNoLongerDrawnAsAnUnarmedMan(t *testing.T) {
	list := data.NewBodyList("unarmed", "swordsman")

	// A REAL, RESOLVABLE TABLE, and not a hand-typed Code (0134 T3):
	// GeneratedWornSet (mapload/spawn.go) substitutes the handed weapon's
	// own NAME into the weapon cell and re-resolves the whole row through
	// the table, so a fixture with no table cannot land anything in slot 1
	// any more. "Placeholder" occupies row 1 deliberately — HeroBodyFor's
	// row-1 arm and its empty-slot arm answer the same list entry
	// (appearance.go's own doc), so a weapon actually at row 1 could not be
	// told apart from no weapon at all; "fixture sword" is written second,
	// at row 2, so it lands on list[1], "swordsman".
	scale := func(name string, damage float64) synth.DataBinRow {
		d := make([]float64, 9)
		d[4], d[5], d[6] = damage, 1, 1
		return synth.DataBinRow{Name: name, Doubles: d}
	}
	var db synth.DataBin
	db.Rows[synth.DataBinShapes] = []synth.DataBinRow{scale("Common", 0.2)}
	db.Rows[synth.DataBinMaterials] = []synth.DataBinRow{scale("Iron", 1)}
	db.Rows[synth.DataBinWeapons] = []synth.DataBinRow{
		{Name: "Placeholder", Params: []int32{-1, -1, -1, -1, -1, data.SkillBlade, 10, 20, 0, 0, -1, 1, 6, 4, -1}},
		{Name: "fixture sword", Params: []int32{-1, -1, -1, -1, -1, data.SkillBlade, 10, 20, 0, 0, -1, 1, 6, 4, -1}},
	}
	d, err := game.LoadDefinitions(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: db.Bytes()}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	sword, err := data.ResolveWeapon("fixture sword", d.Table.Shapes, d.Table.Materials, d.Table.Weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon(fixture sword): %v", err)
	}

	var eq data.Equipment
	eq.SetCode(1, sword.Code)
	wantBody, ok := data.HeroBodyFor(list, eq)
	if !ok || wantBody != "swordsman" {
		t.Fatalf("fixture: row %d derived (%q, %v), want (%q, true)", sword.Row, wantBody, ok, "swordsman")
	}
	want, matched := data.HeroBodyClass(wantBody)
	if !matched {
		t.Fatalf("the fixture body %q matches no arm of the name chain — the member "+
			"would fall back on the bare-handed body it was drawn as before", wantBody)
	}
	if want == data.HeroUnmatchedClass {
		t.Fatalf("the fixture body %q resolves to the forced class %d itself",
			wantBody, data.HeroUnmatchedClass)
	}

	p := game.MissionParty(&sword, list, d.Table)
	if len(p) != 1 {
		t.Fatalf("party of %d, want 1", len(p))
	}
	if p[0].Worn[0] != uint16(sword.Code) {
		t.Errorf("slot 1 = 0x%04x, want the handed sword's own 0x%04x", p[0].Worn[0], uint16(sword.Code))
	}
	if p[0].Class != want {
		t.Errorf("the member carries class %d, want the %q body's own %d", p[0].Class, wantBody, want)
	}
	if p[0].Body != string(wantBody) {
		t.Errorf("the member wears %q, want the derived %q", p[0].Body, wantBody)
	}
}

func TestMovingTheWeaponMovesTheDerivedBody(t *testing.T) {
	// Invented, not the shipped list (SC-3). The names are the law's own so
	// that HeroBodyClass matches an arm; the ORDER is this fixture's.
	list := data.NewBodyList(data.BodyUnarmed, data.BodyPikeman, data.BodySwordsman, data.BodyArcher)

	// A REAL, RESOLVABLE TABLE (0134 T3): GeneratedWornSet re-resolves the
	// handed weapon by NAME against the table, so a fixture with no table
	// cannot land anything in slot 1, and moving "the weapon" now means
	// moving which NAME is handed, not typing a bare Code by hand.
	// "Placeholder" occupies row 1 deliberately and is never resolved by
	// name below: HeroBodyFor's row-1 arm and its empty-slot arm answer the
	// same list entry (appearance.go's own doc), so a weapon actually at
	// row 1 could not be told apart from no weapon at all — which is
	// exactly the coincidence the {"", data.BodyUnarmed} case below relies
	// on to test the OTHER arm.
	scale := func(name string, damage float64) synth.DataBinRow {
		d := make([]float64, 9)
		d[4], d[5], d[6] = damage, 1, 1
		return synth.DataBinRow{Name: name, Doubles: d}
	}
	weaponRow := func(name string) synth.DataBinRow {
		return synth.DataBinRow{
			Name:   name,
			Params: []int32{-1, -1, -1, -1, -1, data.SkillBlade, 10, 20, 0, 0, -1, 1, 6, 4, -1},
		}
	}
	var db synth.DataBin
	db.Rows[synth.DataBinShapes] = []synth.DataBinRow{scale("Common", 0.2)}
	db.Rows[synth.DataBinMaterials] = []synth.DataBinRow{scale("Iron", 1)}
	db.Rows[synth.DataBinWeapons] = []synth.DataBinRow{
		weaponRow("Placeholder"), // row 1
		weaponRow("Pike"),        // row 2
		weaponRow("Sword"),       // row 3
		weaponRow("Bow"),         // row 4
	}
	d, err := game.LoadDefinitions(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: db.Bytes()}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	for _, c := range []struct {
		name string
		want data.HeroBody
	}{
		{"", data.BodyUnarmed}, // no weapon: the empty first slot's own arm
		{"Pike", data.BodyPikeman},
		{"Sword", data.BodySwordsman},
		{"Bow", data.BodyArcher},
	} {
		var w *data.Weapon
		if c.name != "" {
			resolved, err := data.ResolveWeapon(c.name, d.Table.Shapes, d.Table.Materials, d.Table.Weapons)
			if err != nil {
				t.Fatalf("ResolveWeapon(%q): %v", c.name, err)
			}
			w = &resolved
		}
		p := game.MissionParty(w, list, d.Table)
		if len(p) != 1 {
			t.Fatalf("weapon %q: party of %d, want 1", c.name, len(p))
		}
		if p[0].Body != string(c.want) {
			t.Errorf("weapon %q: the member wears %q, want the list's own %q",
				c.name, p[0].Body, c.want)
		}
		key, matched := data.HeroBodyClass(c.want)
		if !matched {
			t.Fatalf("fixture: %q matches no arm of the law", c.want)
		}
		if p[0].Class != key {
			t.Errorf("weapon %q: the member carries class %d, want %q's own %d",
				c.name, p[0].Class, c.want, key)
		}
		if w != nil && p[0].Worn[0] != uint16(w.Code) {
			t.Errorf("weapon %q: slot 1 = 0x%04x, want the handed weapon's own 0x%04x",
				c.name, p[0].Worn[0], uint16(w.Code))
		}
	}
}

// AC-12's surviving half (0085): the authored slot still trains a real
// weapon. Its other half — that the authored body agreed with the authored
// slot — is retired along with PartyBody: there is no second authored fact
// left for it to drift from, because the body is no longer authored at all.
func TestTheAuthoredSkillTrainsARealWeapon(t *testing.T) {
	if n, ok := game.StartingWeaponName(false, game.PartySkillSlot()); !ok || n == "" {
		t.Errorf("the authored slot trains no weapon (%q, %v)", n, ok)
	}
}

// AC-11 — the body name is a LOADER INPUT. It reaches no entity field, no byte
// form and no digest, and this is the falsifiable form of that: two parties
// differing in nothing but the body encode to the same bytes and hash the same.
//
// IT DOES NOT PIN THE VERSION LITERAL, deliberately. "The form is still the one
// 0085 was written beside" is a claim this story has no standing to make and a
// tax on every later bump — 0084 deleted exactly such a pin after 0081 landed a
// version underneath it. What replaces it is stronger and costs nothing: pkg/sim
// does not appear in this story's diff at all, which is recorded in the
// verification rather than asserted here.
func TestABodyNameReachesNoWorldByte(t *testing.T) {
	const w, h = 24, 24
	newMap := func() *alm.Map {
		return &alm.Map{Width: w, Height: h,
			Tiles: make([]uint16, w*h), Overlay: make([]uint8, w*h)}
	}
	start := func(body string) ([]byte, uint64) {
		t.Helper()
		p := []mapload.PartyMember{{Class: 3, Body: body, Hero: game.PartyHero()}}
		world, _, err := mapload.StartMission(newMap(), nil, mapload.DifficultyNormal, p)
		if err != nil {
			t.Fatalf("StartMission(%q): %v", body, err)
		}
		b, err := world.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary(%q): %v", body, err)
		}
		return b, world.Hash()
	}

	a, ha := start("swordsman")
	b, hb := start("xbowman")
	if !bytes.Equal(a, b) {
		t.Errorf("two parties differing only in the body name encode to %d and %d bytes "+
			"that differ — the name reached the world", len(a), len(b))
	}
	if ha != hb {
		t.Errorf("digests %#x and %#x differ for one body-name change", ha, hb)
	}
}
