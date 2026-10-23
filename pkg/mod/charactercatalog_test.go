package mod

import (
	"strings"
	"testing"
)

func TestCharacterAdditionsKeepTheirLayerRule(t *testing.T) {
	d := CharacterData{Rows: []CharacterRow{
		{Mod: "a", File: CharactersFile, Line: 1, Key: "guard", Row: "NPC06"},
		{Mod: "b", File: CharactersFile, Line: 4, Key: "guard", Face: 5},
	}}
	if _, _, _, err := MergeCharacters(d, nil); err == nil || !strings.Contains(err.Error(), "neither loads after the other") {
		t.Fatalf("unordered addition: %v", err)
	}
	d.Order = Precedence{"b": {"a"}}
	_, chars, _, err := MergeCharacters(d, nil)
	if err != nil || len(chars) != 1 || chars[0].Row != "NPC06" || chars[0].Face != 5 {
		t.Fatalf("ordered addition: %+v, %v", chars, err)
	}
}
