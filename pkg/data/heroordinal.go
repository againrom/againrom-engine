package data

import (
	"strings"

	"againrom/pkg/formats/reg"
)

// Ordinal k reads section npc(21+k) (TRIG-HEROTPL-076).
const heroOrdinalSection = 21

// The template table holds 256 sections (TRIG-HEROFAIL-078).
const heroOrdinalLimit = 256

// heroTemplate is one npc section with a Flags string.
type heroTemplate struct {
	Flags string
	Face  int32
}

// HeroTraits is what a hero ordinal reads of an actor (TRIG-HEROORD-075); Female is the typeID test.
type HeroTraits struct {
	Female bool
	Mage   bool
	Face   uint8
}

func (t HeroTraits) archetype() int {
	i := 0
	if t.Mage {
		i += 2
	}
	if t.Female {
		i++
	}
	return i
}

func (n *NPCDefs) loadHeroDefaults(r *reg.Reg) {
	for i, section := range []string{"MaleFighter", "FemaleFighter", "MaleMage", "FemaleMage"} {
		if face, ok := r.GetInt(section, npcFaceKey); ok {
			n.heroFaces[i] = face
		}
	}
}

func (n *NPCDefs) loadHeroTemplate(r *reg.Reg, section string, id int32, flags string) {
	if flags == "" {
		return
	}
	face, _ := r.GetInt(section, npcFaceKey)
	n.hero[id] = heroTemplate{Flags: flags, Face: face}
}

// heroToken is a case-sensitive substring test, off when the negated spelling is present.
func heroToken(flags, token string) bool {
	return strings.Contains(flags, token) && !strings.Contains(flags, "!"+token)
}

// HeroOrdinalMatches tests actor against hero ordinal k and the primary (TRIG-HEROORD-075, TRIG-HEROTPL-076).
func (n *NPCDefs) HeroOrdinalMatches(k int, primary, actor HeroTraits) bool {
	if k < 0 {
		return false
	}
	if k == 0 {
		return actor == primary
	}
	if n == nil {
		return false
	}
	section := heroOrdinalSection + k
	if section >= heroOrdinalLimit {
		return false
	}
	tpl, ok := n.hero[int32(section)]
	if !ok {
		return false
	}
	f := tpl.Flags
	if heroToken(f, "Mage") && !actor.Mage ||
		heroToken(f, "Female") && !actor.Female ||
		heroToken(f, "MySex") && actor.Female != primary.Female ||
		heroToken(f, "MyClass") && actor.Mage != primary.Mage ||
		heroToken(f, "!Mage") && actor.Mage ||
		heroToken(f, "!Female") && actor.Female ||
		heroToken(f, "!MySex") && actor.Female == primary.Female ||
		heroToken(f, "!MyClass") && actor.Mage == primary.Mage {
		return false
	}
	if heroToken(f, "Face") {
		return tpl.Face == int32(actor.Face)
	}
	return n.heroFaces[actor.archetype()] == int32(actor.Face)
}

// ResolveHero returns the first roster index that satisfies ordinal k.
func (n *NPCDefs) ResolveHero(k int, primary HeroTraits, roster []HeroTraits) (int, bool) {
	for i, a := range roster {
		if n.HeroOrdinalMatches(k, primary, a) {
			return i, true
		}
	}
	return 0, false
}
