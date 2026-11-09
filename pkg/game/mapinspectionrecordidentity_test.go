package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func recordIdentityEditor(t *testing.T) *ui.MapEditor {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12), Units: []synth.ALMUnit{
		{X: 3 * 256, Y: 4 * 256, ClassID: 40, ClassSubID: 2, UnitID: 11},
		{X: 7 * 256, Y: 8 * 256, ClassID: 40, ClassSubID: 2, UnitID: 12},
	}})
	m, err := mapedit.New(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Place(mapedit.KindLoot, m.GroundSack(12*256, 12*256, 0x101)); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if _, err := m.AddUnitStock(i); err != nil {
			t.Fatal(err)
		}
	}
	for _, tag := range []uint32{10, 20, 30} {
		r := make([]byte, 26)
		binary.LittleEndian.PutUint32(r, tag)
		if _, err := m.Place(mapedit.KindEnchantment, r); err != nil {
			t.Fatal(err)
		}
	}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: authoringTable()}
	source := filepath.Join(t.TempDir(), "identity.alm")
	e := ui.NewMapEditor(nil, nil, func(string) (*ui.InspectionDocument, error) { return x.inspect(m.Bytes(), source) })
	e.Layout(1280, 800)
	if err := e.Open(source); err != nil {
		t.Fatal(err)
	}
	return e
}

func recordIdentityToken(t *testing.T, e *ui.MapEditor, r ui.InspectionRecord) string {
	t.Helper()
	if r.Kind == ui.KindUnit {
		return fmt.Sprintf("unit:%d", r.ScriptID)
	}
	k := mapedit.KindLoot
	if r.Kind == ui.KindEnchantment {
		k = mapedit.KindEnchantment
	}
	b, err := e.Document().Edits.(*mapInspectionEdits).model.Record(k, r.Index)
	if err != nil {
		t.Fatal(err)
	}
	offset := alm.LootOwner
	if r.Kind == ui.KindEnchantment {
		offset = 0
	}
	return fmt.Sprintf("%s:%d", string(r.Kind), binary.LittleEndian.Uint32(b[offset:]))
}

func TestInspectionRememberedRecordEditSequences(t *testing.T) {
	type step struct {
		command string
		args    map[string]any
	}
	del := func(kind string, index int) step {
		return step{"delete_record", map[string]any{"kind": kind, "index": index}}
	}
	move := step{"move_record", map[string]any{"kind": "Unit", "index": 1, "x": 9, "y": 10}}
	paint := step{"paint", map[string]any{"plane": "height", "cells": [][2]int{{16, 16}}, "value": 2}}
	undo, redo := step{"undo", map[string]any{}}, step{"redo", map[string]any{}}
	cases := []struct {
		name  string
		steps []step
		gone  []string
	}{
		{"unit and owned stock", []step{del("Unit", 0)}, []string{"unit:11", "Stock:11"}},
		{"sack before stocks", []step{del("Sack", 0)}, []string{"Sack:0"}},
		{"stock before stock", []step{del("Stock", 1)}, []string{"Stock:11"}},
		{"unit before enchantment", []step{del("Unit", 0)}, []string{"unit:11", "Stock:11"}},
		{"enchantment before enchantment", []step{del("Enchantment", 0)}, []string{"Enchantment:10"}},
		{"move", []step{move}, nil},
		{"brush", []step{paint}, nil},
		{"undo", []step{del("Unit", 0), undo}, nil},
		{"redo", []step{del("Unit", 0), undo, redo}, []string{"unit:11", "Stock:11"}},
		{"redo after new edit", []step{del("Unit", 0), undo, paint}, nil},
	}
	for _, tc := range cases {
		for _, target := range []struct {
			tab, token string
		}{
			{"units", "unit:11"}, {"units", "unit:12"}, {"sacks", "Sack:0"},
			{"sacks", "Stock:11"}, {"sacks", "Stock:12"},
			{"cellspells", "Enchantment:10"}, {"cellspells", "Enchantment:20"}, {"cellspells", "Enchantment:30"},
		} {
			t.Run(tc.name+"/"+target.token, func(t *testing.T) {
				e := recordIdentityEditor(t)
				c := &editorClicker{t: t, now: time.Unix(1, 0)}
				bar := func(tab string) {
					r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: tab})
					c.click(e, r, ok, tab)
				}
				bar(target.tab)
				at := slices.IndexFunc(e.Document().Records, func(r ui.InspectionRecord) bool {
					return r.Kind != ui.KindSection && (r.Kind == ui.KindUnit || r.Kind == ui.KindSack || r.Kind == ui.KindStock || r.Kind == ui.KindEnchantment) && recordIdentityToken(t, e, r) == target.token
				})
				if at < 0 || !e.Select(at, false) {
					t.Fatal("fixture target missing", target.token)
				}
				bar("structures")
				tool, must := editorMCP(t, e)
				before := bytes.Clone(e.Document().SourceBytes)
				for _, op := range tc.steps {
					must(op.command, op.args)
				}
				if tc.name == "redo after new edit" && e.Document().Edits.CanRedo() {
					t.Fatal("new edit kept the discarded redo branch")
				}
				if tc.name == "redo after new edit" {
					current := bytes.Clone(e.Document().SourceBytes)
					if result := tool("redo", map[string]any{}); !result.IsError || !bytes.Equal(current, e.Document().SourceBytes) {
						t.Fatal("discarded Redo changed the document")
					}
				}
				if (tc.name == "move" || tc.name == "brush") && bytes.Equal(before, e.Document().SourceBytes) {
					t.Fatal("edit witness changed no bytes")
				}
				bar(target.tab)
				want := target.token
				if slices.Contains(tc.gone, want) {
					want = ""
				}
				got := ""
				if slices.Contains(e.OpenWindows(), "object") && e.Selected() >= 0 {
					got = recordIdentityToken(t, e, e.Document().Records[e.Selected()])
				}
				if got != want {
					t.Fatalf("remembered window = %q, want %q", got, want)
				}
			})
		}
	}
}
