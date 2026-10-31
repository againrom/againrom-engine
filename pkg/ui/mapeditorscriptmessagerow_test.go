package ui

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/words"
)

func TestScriptsListMessageRowsShowTheirOwnKind(t *testing.T) {
	ru, err := words.Load(words.Engine(), "ru")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		book *words.Book
		want []string
	}{
		{"en", nil, []string{"Message 2  Hello there", "Message 3", "Action 4  Win"}},
		{"ru", &ru, []string{"Сообщение 2  Hello there", "Сообщение 3", "Действие 4  Win"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := newModesEditor(t, "full")
			if tc.book != nil {
				e.SetWords(*tc.book)
			}
			e.doc.Records = []InspectionRecord{{Kind: "Action", Index: 0}, {Kind: "Action", Index: 1}, {Kind: "Action", Index: 2}, {Kind: "Action", Index: 3}}
			e.doc.Script = &InspectionScriptModel{
				Actions: []InspectionScriptNode{
					{Record: 0, ID: 10, Opcode: 3, Label: "Set variable"},
					{Record: 1, ID: 11, Opcode: 2, Types: [10]uint32{scriptInt}, Values: [10]uint32{1}, Label: "Send message"},
					{Record: 2, ID: 12, Opcode: 2, Types: [10]uint32{scriptInt}, Values: [10]uint32{2}, Label: "Send message"},
					{Record: 3, ID: 13, Opcode: 4, Label: "Win"},
				},
				Targets: map[uint32][]InspectionScriptTarget{ScriptMessageParameter: {
					{Value: 1, Text: "\nHello there\nSecond line"}, {Value: 2, Text: ""},
				}},
			}
			_ = e.Key("scripts")
			rows := scriptsTreeRows(e)
			var got []string
			for _, r := range rows {
				got = append(got, strings.TrimSpace(r.Text))
			}
			for _, w := range tc.want {
				if !slices.Contains(got, w) {
					t.Errorf("row %q missing in %q", w, got)
				}
			}
		})
	}
}
