package game

import (
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

func TestEditorNativeChoiceWordsUseFontBytesInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			_, s, _ := nativeInventoryEditor(t)
			x := s.inspector
			x.archives.Base.Profile.Language = lang
			x.font = &text.Font{Selector: 0}
			check := func(key string, choice int, word string) {
				t.Helper()
				fields := s.Fields("Unit", 0)
				f := fields[fieldIndex(t, fields, key)]
				if choice >= len(f.Choices) {
					t.Fatal("missing choice", key, choice)
				}
				want, _ := textinput.EncodeText(words.For(lang).Text(word), 0, textinput.Substitute)
				if f.Choices[choice].Label != want {
					t.Fatalf("%s choice %d is not font-encoded: %q != %q", key, choice, f.Choices[choice].Label, want)
				}
			}
			check("npc", 0, "editor.unit.npc.none")
			check("face", 0, "editor.unit.look.inherit")
			check("sex", 0, "editor.unit.look.male")
			check("sex", 1, "editor.unit.look.female")
			check("gear", 0, "editor.unit.meaning.gear")
			check("wear1", 0, "editor.equipment.inherit")
			check("wear1", 1, "editor.equipment.empty")
			check("carryadd", 0, "editor.inventory.choose")
			if _, err := s.SetField("Unit", 0, "carryadd", 0x102); err != nil {
				t.Fatal(err)
			}
			check("carry1", 0, "editor.inventory.remove")
			check("carry1", 1, "editor.inventory.duplicate")
			check("carryrecipe1", 0, "editor.equipment.newrecipe")
			if _, err := s.SetField("Unit", 0, "npc", 8); err != nil {
				t.Fatal(err)
			}
			check("wear1", 1, "editor.equipment.npcempty")
		})
	}
}
