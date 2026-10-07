package data

import (
	"fmt"
	"testing"

	"againrom/internal/synth"
)

func TestCampaignCompanionRowRestoresAllHeroAxesAndRejectsAmbiguity(t *testing.T) {
	n := npcReg(t,
		regDir("npc21", regStr("Flags", "Hero,Me"), regInt("DataBinID", 26)),
		regDir("npc22", regStr("Flags", "Hero,Mage,!MySex"), regInt("DataBinID", 26)),
		regDir("npc23", regStr("Flags", "Hero,!Mage,!MySex"), regInt("DataBinID", 26)),
		regDir("npc24", regStr("Flags", "Hero,!MyClass,MySex"), regInt("DataBinID", 26)),
		regDir("npc25", regStr("Flags", "Hero,!Mage"), regInt("DataBinID", 42)),
		regDir("npc51", regStr("Flags", "Human"), regInt("DataBinID", 99)),
	)
	humans := testCollection{{}}
	for server := int32(26); server <= 42; server++ {
		humans = append(humans, testEntry{name: fmt.Sprint(server), params: humanRow(1, server)})
	}
	for _, mage := range []bool{false, true} {
		for _, female := range []bool{false, true} {
			for band := 0; band < 4; band++ {
				sex := int32(0)
				if female {
					sex = 1
				}
				class := int32(0)
				if mage {
					class = 2
				}
				// Independent selectors: opposite sex mage/fighter, then
				// opposite class at the player's own sex. Brian is direct.
				for npc, server := range map[int32]int32{22: 26 + 4*int32(band) + 2 + (1 - sex), 23: 26 + 4*int32(band) + (1 - sex), 24: 26 + 4*int32(band) + (2 - class) + sex, 25: 42} {
					got, ok := n.CampaignCompanionForRow(humans, int(server-25), mage, female)
					if !ok || got != npc {
						t.Fatalf("mage=%v female=%v server=%d: %d/%v want npc%d", mage, female, server, got, ok, npc)
					}
				}
			}
		}
	}
	n.flags[99], n.byID[99] = NPCTokens(NPCTokenHero), 42
	if _, ok := n.CampaignCompanionForRow(humans, 17, false, false); ok {
		t.Fatal("ambiguous direct row acquired an identity")
	}
	if _, ok := n.CampaignCompanionForRow(humans, 0, false, false); ok {
		t.Fatal("reserved row acquired an identity")
	}
}

// The scenario NPC lookup, over synthetic registries. internal/synth writes the
// byte stream and pkg/formats/reg parses it back, so the loader only ever sees a
// tree it could have got from a real file.

func npcReg(t *testing.T, sections ...synth.RegNode) *NPCDefs {
	t.Helper()
	return LoadNPCDefs(parseReg(t, sections))
}

func TestNPCDefsReadsTheDefinitionKey(t *testing.T) {
	n := npcReg(t,
		regDir("npc51", regStr("Flags", "Human,Female"), regInt("DataBinID", 509)),
		regDir("npc52", regStr("Flags", "Human,Mage"), regInt("DataBinID", 510)),
	)
	for _, c := range []struct {
		id   int32
		want int32
	}{{51, 509}, {52, 510}} {
		got, ok := n.ServerID(c.id)
		if !ok || got != c.want {
			t.Fatalf("ServerID(%d) = %d, %v; want %d, true", c.id, got, ok, c.want)
		}
	}
	if n.Len() != 2 {
		t.Fatalf("Len = %d, want 2", n.Len())
	}
}

func TestNPCDefsPreservesTheExactHeroConstructorFlag(t *testing.T) {
	n := npcReg(t,
		regDir("npc51", regStr("Flags", "Human,Female"), regInt("DataBinID", 509)),
		regDir("npc52", regStr("Flags", "Human,Hero"), regInt("DataBinID", 510)),
	)
	if n.Hero(51) || !n.Hero(52) || n.Hero(99) {
		t.Fatalf("Hero flags = npc51:%v npc52:%v absent:%v; want false/true/false",
			n.Hero(51), n.Hero(52), n.Hero(99))
	}
}

func TestNPCDefsReadsMercenaryPriceTermsIndependently(t *testing.T) {
	n := npcReg(t, regDir("npc3", regInt("PriceA", 7), regInt("PriceB", 11)))
	got, ok := n.Mercenary(3)
	if !ok || got != (MercenaryTerms{PriceA: 7, PriceB: 11}) {
		t.Fatalf("Mercenary(3) = %#v, %v", got, ok)
	}
	if n.Len() != 0 {
		t.Fatalf("price-only NPC changed definition count to %d", n.Len())
	}
}

// The three absences answer alike, which is the contract: a caller has one thing
// to act on and no choice to make between them.
func TestNPCDefsAbsences(t *testing.T) {
	n := npcReg(t,
		regDir("npc21", regStr("Flags", "Hero,Me,Start"), regInt("DataBinID", 26)),
		regDir("npc30", regStr("Flags", "Human")),
	)
	for _, id := range []int32{21, 30, 99, 0, -1} {
		if got, ok := n.ServerID(id); ok {
			t.Fatalf("ServerID(%d) = %d, true; want no entry", id, got)
		}
	}
	if n.Len() != 0 {
		t.Fatalf("Len = %d, want 0", n.Len())
	}
}

func TestNPCDefsComposesTheFourStartArchetypesFromTheirTokens(t *testing.T) {
	n := npcReg(t,
		regDir("npc21", regStr("Flags", "Hero,Me,Start"), regInt("DataBinID", 26)),
		regDir("npc22", regStr("Flags", "Hero,Mage,!MySex,Start"), regInt("DataBinID", 26)),
		regDir("npc23", regStr("Flags", "Hero,!Mage,!MySex,Start"), regInt("DataBinID", 26)),
		regDir("npc24", regStr("Flags", "Hero,!MyClass,MySex,Start"), regInt("DataBinID", 26)),
		regDir("npc25", regStr("Flags", "Hero,!Mage"), regInt("DataBinID", 42)),
	)
	for _, tc := range []struct {
		id           int32
		playerMage   bool
		playerFemale bool
		mage         bool
		female       bool
	}{{21, true, false, true, false}, {22, false, false, true, true},
		{23, true, false, false, true}, {24, true, true, false, true}} {
		mage, female, ok := n.ComposedArchetype(tc.id, tc.playerMage, tc.playerFemale)
		if !ok || mage != tc.mage || female != tc.female {
			t.Errorf("ComposedArchetype(%d, %v, %v) = %v, %v, %v; want %v, %v, true",
				tc.id, tc.playerMage, tc.playerFemale, mage, female, ok, tc.mage, tc.female)
		}
	}
	if _, _, ok := n.ComposedArchetype(25, false, false); ok {
		t.Fatal("a direct DataBinID record was reported as composed")
	}
}

func TestNPCDefsResolvesEveryPersistentCampaignHeroTier(t *testing.T) {
	n := npcReg(t,
		regDir("npc22", regStr("Flags", "Hero,Mage,!MySex,Start"), regInt("DataBinID", 26)),
		regDir("npc23", regStr("Flags", "Hero,!Mage,!MySex,Start"), regInt("DataBinID", 26)),
		regDir("npc24", regStr("Flags", "Hero,!MyClass,MySex,Start"), regInt("DataBinID", 26)),
		regDir("npc25", regStr("Flags", "Hero,!Mage"), regInt("DataBinID", 42)),
		regDir("npc26", regStr("Flags", "Hero,!Mage"), regInt("DataBinID", 43)),
	)
	for _, tc := range []struct {
		name               string
		npc                int32
		mission            int
		playerMage, female bool
		want               int32
	}{
		{"town female primary makes Fergard", 22, 30, false, true, 28},
		{"town male primary makes Reniesta", 22, 30, false, false, 29},
		{"mission 70 female primary makes Danath", 23, 70, false, true, 30},
		{"mission 70 male primary makes Naira", 23, 70, false, false, 31},
		{"mission 100 male fighter makes Fergard", 24, 100, false, false, 36},
		{"mission 100 female fighter makes Reniesta", 24, 100, false, true, 37},
		{"mission 100 male mage makes Danath", 24, 100, true, false, 34},
		{"mission 100 female mage makes Naira", 24, 100, true, true, 35},
		{"Brian keeps his fixed row", 25, 40, true, true, 42},
		{"Rood keeps his fixed row", 26, 140, false, false, 43},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := n.CampaignServerID(tc.npc, tc.mission, tc.playerMage, tc.female)
			if !ok || got != tc.want {
				t.Fatalf("CampaignServerID = %d, %v; want %d, true", got, ok, tc.want)
			}
		})
	}
	if _, ok := n.CampaignServerID(99, 70, false, false); ok {
		t.Fatal("an absent NPC resolved a campaign row")
	}
}

// A section that is not `npc<decimal>` is not a per-NPC section, so the
// archetype blocks and the multiplayer face lists stay out with no list of names
// anywhere.
func TestNPCDefsSkipsNonSections(t *testing.T) {
	n := npcReg(t,
		regDir("MaleFighter", regInt("DataBinID", 1)),
		regDir("Multiplayer", regInt("DataBinID", 2)),
		regDir("npc", regInt("DataBinID", 3)),
		regDir("npc21x", regInt("DataBinID", 4)),
		regDir("NPC7", regInt("DataBinID", 5)),
	)
	if n.Len() != 1 {
		t.Fatalf("Len = %d, want 1 (only NPC7 is a section)", n.Len())
	}
	if got, ok := n.ServerID(7); !ok || got != 5 {
		t.Fatalf("ServerID(7) = %d, %v; want 5, true — the stem folds case", got, ok)
	}
}

// A nil table is "no registry", which is what a caller that never opened the
// campaign container holds.
func TestNPCDefsNil(t *testing.T) {
	var n *NPCDefs
	if _, ok := n.ServerID(51); ok {
		t.Fatal("a nil table answered an entry")
	}
	if n.Len() != 0 {
		t.Fatalf("Len = %d, want 0", n.Len())
	}
	if _, _, ok := n.ComposedArchetype(23, false, false); ok {
		t.Fatal("a nil table answered a composed archetype")
	}
	if got := LoadNPCDefs(nil); got.Len() != 0 {
		t.Fatalf("LoadNPCDefs(nil).Len = %d, want 0", got.Len())
	}
}
