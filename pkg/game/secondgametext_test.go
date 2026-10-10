package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestSecondGameNPCKeysUseDefinitionWordTransforms(t *testing.T) {
	units := make([]int32, 56)
	units[55] = 70000
	humans := make([]int32, 25)
	humans[24] = 12340
	table := readUnder(base.GameROM2, &mapload.Table{
		Units:  &fixtureCollection{names: []string{"", "unit"}, params: [][]int32{nil, units}},
		Humans: &fixtureCollection{names: []string{"", "person"}, params: [][]int32{nil, humans}},
	})
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	mission := &Mission{World: w, Map: &alm.Map{Units: []alm.Unit{
		{UnitID: 12, ServerID: 70000}, {UnitID: 13, Flags: 0x10, ServerID: 12340},
	}}}
	keys := secondGameNPCKeys(mission, table)
	if len(keys) != 2 || keys[0] != 4464 || keys[1] != 234 {
		t.Fatalf("NPC keys = %v, want entity0=4464 entity1=234", keys)
	}
}

func TestSecondGameBriefingUsesMissionSection(t *testing.T) {
	src := missionSource{"main/text/mission10.txt": []byte("#briefing\r\nFind the gate.\r\n#event1\r\n<part=1>\r\nHello.")}
	if got := MissionObjectivesFor(src, base.GameROM2, 10); got != "Find the gate." {
		t.Fatalf("briefing = %q", got)
	}
}

func TestSecondGameAudienceUsesControlledPresence(t *testing.T) {
	payload := []byte("<part=1 npcalive=7 npc=42>\r\nalive\r\n<part=1 npcdead=7 npc=42>\r\nabsent\r\n")
	for _, present := range []bool{false, true} {
		audience := EventAudience{tags: secondEventTags{}, NPCPresent: func(npc int) bool { return npc == 7 && present }}
		got, ok := dialoguePart(payload, 1, audience)
		want := "absent\r\n"
		if present {
			want = "alive"
		}
		if !ok || got != want {
			t.Errorf("controlled presence %v: body=%q found=%v, want %q", present, got, ok, want)
		}
	}
}

func TestSecondGameAudienceIgnoresBareSpeakerClassArms(t *testing.T) {
	audience := EventAudience{tags: secondEventTags{}, Speaker: func(int) (bool, bool, bool) { return false, true, true }}
	body, ok := dialoguePart([]byte("<part=1 female fighter npc=7>\r\nline"), 1, audience)
	if !ok || body != "line" {
		t.Fatalf("bare speaker-class tags rejected ROM2 page: %q %v", body, ok)
	}
	if _, ok := dialoguePart([]byte("<part=1 iamfemale>\r\nline"), 1, audience); ok {
		t.Fatal("male hero accepted the iamfemale alternative")
	}
}

func TestSecondGameSpeakerReadsFiveCharacters(t *testing.T) {
	npc, named := EventPartSpeaker([]byte("<part=1 npc=123456>\r\nline"), 1, EventAudience{tags: secondEventTags{}})
	if !named || npc != 12345 {
		t.Fatalf("speaker=%d named=%v, want 12345 true", npc, named)
	}
}

func TestSecondGameSectionUsesFirstSubstringSpan(t *testing.T) {
	for _, tc := range []struct{ source, key, want string }{
		{"prefix#event1!!body#event2!!other", "event1", "body"},
		{"#event10\r\nbody#event1\r\nother", "event1", "\nbody"},
		{"#Event1\r\nbody", "event1", ""},
		{"#event1\r\nfirst\x00#event1\r\nsecond", "event1", "first"},
	} {
		if got := secondGameTextSection([]byte(tc.source), tc.key); got != tc.want {
			t.Errorf("section %q of %q = %q, want %q", tc.key, tc.source, got, tc.want)
		}
	}
}
