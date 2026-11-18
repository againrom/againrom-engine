package ui

import "testing"

// A card row holds its own rows: the head spans them, a struck card and an
// armed button read their flags, and a label takes its arguments as the
// format's verbs say.
func TestDescriptionCardHoldsItsRows(t *testing.T) {
	d, err := parsePopupDesc([]byte(`{
		"window": "card",
		"rows": [
			{"each": "cards", "rows": [
				{"kind": "card", "when": "!struck", "labelBind": "number", "value": "speaker",
					"buttons": [{"word": "pick", "armed": "picking"}, {"word": "split", "tip": "splittip"}],
					"rows": [
						{"each": "cells", "rows": [{"kind": "multiline", "value": "text", "whole": true}]},
						{"each": "never", "rows": [{"kind": "text", "label": "never", "labelArgs": ["line"]}]}
					]},
				{"kind": "card", "when": "struck", "struck": "struck", "label": "removed"}
			]}
		],
		"sample": {"lists": {"cards": [
			{"values": {"number": "1", "speaker": "44"}, "flags": ["picking"], "lists": {
				"cells": [{"values": {"text": "a"}}, {"values": {"text": "b"}}],
				"never": [{"values": {"line": "27"}}]}},
			{"values": {"number": "2"}, "flags": ["struck"]}
		]}}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	words := map[string]string{"pick": "Pick", "split": "Split", "splittip": "Split it", "never": "Never: %s", "removed": "Removed"}
	_, made := d.build(d.Sample, func(id string) string { return words[id] })
	if len(made) != 5 {
		t.Fatalf("rows = %d, want 5 (head, two cells, never, struck card)", len(made))
	}
	head := made[0].Row
	if head.Kind != popupCard || head.Label != "1" || head.Span != 3 {
		t.Fatalf("head = kind %d label %q span %d, want card \"1\" spanning 3", head.Kind, head.Label, head.Span)
	}
	if len(head.Armed) != 2 || !head.Armed[0] || head.Armed[1] {
		t.Fatalf("armed = %v, want [true false]", head.Armed)
	}
	if len(head.ButtonTips) != 2 || head.ButtonTips[0] != "" || head.ButtonTips[1] != "Split it" {
		t.Fatalf("button tips = %q, want [\"\" \"Split it\"]", head.ButtonTips)
	}
	if !made[1].Row.Whole || !made[2].Row.Whole || made[1].Row.Kind != popupMultiline {
		t.Fatal("the cells' texts are not whole multiline rows")
	}
	if got := made[3].Row.Label; got != "Never: 27" {
		t.Fatalf("never label = %q, want a number kept as text for %%s", got)
	}
	if s := made[4].Row; !s.Struck || s.Label != "Removed" || s.Armed != nil {
		t.Fatalf("struck card = %+v", s)
	}
}

func TestDescriptionRefusesWholeOutsideMultiline(t *testing.T) {
	if _, err := parsePopupDesc([]byte(`{"window": "w", "rows": [{"kind": "text", "label": "x", "whole": true}]}`)); err == nil {
		t.Fatal("a whole text row was accepted")
	}
}
