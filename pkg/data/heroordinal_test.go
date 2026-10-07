package data

import "testing"

func shippedHeroRegistry(t *testing.T) *NPCDefs {
	t.Helper()
	return npcReg(t,
		regDir("npc21", regStr("Flags", "Hero,Me,Start")),
		regDir("npc22", regStr("Flags", "Hero,Mage,!MySex,Start")),
		regDir("npc23", regStr("Flags", "Hero,!Mage,!MySex,Start")),
		regDir("npc24", regStr("Flags", "Hero,!MyClass,MySex,Start")),
		regDir("npc25", regStr("Flags", "Hero,Face,!Female,!Mage"), regInt("Face", 1)),
		regDir("npc26", regStr("Flags", "Hero,Face,Mage,!Female"), regInt("Face", 4)),
		regDir("MaleFighter", regInt("Face", 5)),
		regDir("FemaleFighter", regInt("Face", 1)),
		regDir("MaleMage", regInt("Face", 3)),
		regDir("FemaleMage", regInt("Face", 1)),
	)
}

func TestHeroOrdinalPredicatePerOrdinal(t *testing.T) {
	n := shippedHeroRegistry(t)
	maleMage := HeroTraits{Female: false, Mage: true, Face: 3}
	femaleMage := HeroTraits{Female: true, Mage: true, Face: 1}
	maleFighter := HeroTraits{Female: false, Mage: false, Face: 5}
	femaleFighter := HeroTraits{Female: true, Mage: false, Face: 1}
	maleFighterFace1 := HeroTraits{Face: 1}
	maleMageFace4 := HeroTraits{Mage: true, Face: 4}
	for _, tc := range []struct {
		name      string
		k         int
		primary   HeroTraits
		actor     HeroTraits
		want      bool
		mustMatch bool
	}{
		{"0 self", 0, maleMage, maleMage, true, true},
		{"0 other sex", 0, maleMage, femaleMage, false, true},
		{"0 other face", 0, maleMage, HeroTraits{Mage: true, Face: 2}, false, true},
		{"1 mage of the other sex with its default face", 1, maleMage, femaleMage, true, true},
		{"1 wrong face", 1, maleMage, HeroTraits{Female: true, Mage: true, Face: 2}, false, true},
		{"1 same sex", 1, maleMage, maleMage, false, true},
		{"1 non-mage", 1, maleMage, femaleFighter, false, true},
		{"2 non-mage of the other sex", 2, maleMage, femaleFighter, true, true},
		{"2 mage", 2, maleMage, femaleMage, false, true},
		{"3 other class same sex", 3, maleMage, maleFighter, true, true},
		{"3 other class other sex", 3, maleMage, femaleFighter, false, true},
		{"3 same class", 3, maleMage, HeroTraits{Mage: true, Face: 3}, false, true},
		{"4 male non-mage face 1", 4, femaleMage, maleFighterFace1, true, true},
		{"4 default face 5 is not face 1", 4, femaleMage, maleFighter, false, true},
		{"4 female", 4, femaleMage, femaleFighter, false, true},
		{"5 male mage face 4", 5, femaleMage, maleMageFace4, true, true},
		{"5 male mage default face", 5, femaleMage, maleMage, false, true},
		{"5 female mage face 4", 5, femaleMage, HeroTraits{Female: true, Mage: true, Face: 4}, false, true},
	} {
		if got := n.HeroOrdinalMatches(tc.k, tc.primary, tc.actor); got != tc.want {
			t.Errorf("%s: ordinal %d = %v, want %v", tc.name, tc.k, got, tc.want)
		}
	}
	if !n.HeroOrdinalMatches(4, maleFighter, maleFighterFace1) || !n.HeroOrdinalMatches(0, maleFighterFace1, maleFighterFace1) {
		t.Fatal("a face 1 male fighter must satisfy ordinal 4 as well as ordinal 0")
	}
}

func TestHeroOrdinalResolverTakesTheFirstQualifyingActorInListOrder(t *testing.T) {
	n := shippedHeroRegistry(t)
	primary := HeroTraits{Mage: true, Face: 3}
	roster := []HeroTraits{primary, {Face: 1}, {Female: true, Mage: true, Face: 1}, {Female: true, Mage: true, Face: 1}}
	if i, ok := n.ResolveHero(1, primary, roster); !ok || i != 2 {
		t.Fatalf("ordinal 1 = %d/%v, want the first qualifying actor, 2", i, ok)
	}
	if i, ok := n.ResolveHero(0, primary, roster); !ok || i != 0 {
		t.Fatalf("ordinal 0 = %d/%v, want the primary", i, ok)
	}
	if i, ok := n.ResolveHero(4, primary, roster); !ok || i != 1 {
		t.Fatalf("ordinal 4 = %d/%v, want the face 1 male non-mage", i, ok)
	}
}

func TestHeroOrdinalUnresolvedCases(t *testing.T) {
	n := shippedHeroRegistry(t)
	primary := HeroTraits{Mage: true, Face: 3}
	if _, ok := n.ResolveHero(1, primary, []HeroTraits{primary}); ok {
		t.Fatal("a roster of the primary alone resolved ordinal 1")
	}
	if _, ok := n.ResolveHero(1, primary, nil); ok {
		t.Fatal("an empty roster resolved an ordinal")
	}
	for _, k := range []int{-1, 6, 7, 234, 235, 999} {
		if _, ok := n.ResolveHero(k, primary, []HeroTraits{primary, {Female: true, Mage: true, Face: 1}}); ok {
			t.Fatalf("ordinal %d has no template record yet resolved", k)
		}
	}
	var none *NPCDefs
	if i, ok := none.ResolveHero(0, primary, []HeroTraits{{Face: 1}, primary}); !ok || i != 1 {
		t.Fatal("ordinal 0 needs no registry", i, ok)
	}
	if _, ok := none.ResolveHero(1, primary, []HeroTraits{{Female: true, Mage: true, Face: 1}}); ok {
		t.Fatal("a missing registry resolved a template ordinal")
	}
}

func TestHeroTemplateTokensAreCaseSensitiveSubstrings(t *testing.T) {
	n := npcReg(t,
		regDir("npc22", regStr("Flags", "hero,mage,!mysex")),
		regDir("npc23", regStr("Flags", "Mage,!Mage")),
		regDir("npc24", regStr("Flags", "Female"), regInt("Face", 9)),
		regDir("npc25", regStr("Flags", "")),
		regDir("FemaleMage", regInt("Face", 7)),
	)
	primary := HeroTraits{Mage: true, Face: 3}
	if !n.HeroOrdinalMatches(1, primary, HeroTraits{Face: 1}) {
		t.Fatal("lower-case tokens were read; the default male fighter face 1 should pass")
	}
	if n.HeroOrdinalMatches(2, primary, HeroTraits{Mage: true, Face: 1}) {
		t.Fatal("Mage,!Mage kept the positive Mage test active")
	}
	if !n.HeroOrdinalMatches(2, primary, HeroTraits{Female: true, Face: 1}) {
		t.Fatal("Mage,!Mage did not apply the negated test")
	}
	if !n.HeroOrdinalMatches(3, primary, HeroTraits{Female: true, Mage: false, Face: 1}) {
		t.Fatal("Female template rejected a female fighter with the default face")
	}
	if n.HeroOrdinalMatches(3, primary, HeroTraits{Female: true, Face: 9}) {
		t.Fatal("Face value was read without a Face token")
	}
	if n.HeroOrdinalMatches(4, primary, HeroTraits{Face: 1}) {
		t.Fatal("empty Flags produced a template record")
	}
}
