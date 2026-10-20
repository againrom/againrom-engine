package game

import (
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// unitValues are the numbers the unit popup shows of one placement's actor.
type unitValues struct {
	HP, MaxHP, Mana, MaxMana                         int32
	Body, Mind, Spirit, Reaction                     int32
	ToHit, DamageMin, DamageMax, Defence, Absorption int32
	Speed, Sight, XP                                 int32
	Protection, Resistance                           [5]int32
	Skill                                            [data.SkillSlots]int32
	Spells                                           []int32 // spell ids the actor knows, ascending
	TypeID, GoldChance, GoldMin, GoldMax             int32
}

// unitSheet is one placement as the game builds it: its actor's values and
// the items its body holds, which are what its corpse drops.
type unitSheet struct {
	Values     unitValues
	Worn       [sim.EquipSlots]sim.ItemInstance
	Carried    []sim.ItemInstance
	Suppressed bool // the template's name holds NPC: the inventory is deleted at death
	// Life, Decay and Facing are how the game draws the actor at mission
	// start: its life state, decay stage and sheet facing.
	Life          uint8
	Decay, Facing int
}

// valuesOf reads the popup's numbers off a spawned actor. The damage range is
// the base to base plus spread the combat draw covers, the gold range the
// minimum to minimum plus maximum the death roll covers (HERO-KILL-027).
func valuesOf(e sim.Entity) unitValues {
	v := unitValues{HP: e.HP, MaxHP: e.MaxHP, Mana: e.Mana, MaxMana: e.MaxMana,
		Mind: e.Mind, Spirit: e.Spirit, Reaction: e.Reaction,
		ToHit: e.ToHit, DamageMin: e.DamageBase, DamageMax: e.DamageBase + e.DamageSpread,
		Defence: e.Defence, Absorption: e.Absorption,
		Speed: e.Speed, Sight: int32(e.ScanRange), XP: e.XPValue, Protection: e.Protection,
		Skill: e.Skill, TypeID: e.TypeID, GoldChance: e.GoldChance,
		GoldMin: e.TreasureMin, GoldMax: e.TreasureMin + e.TreasureMax}
	if e.NativeBasis.BodyPresent {
		v.Body = int32(e.NativeBasis.Body)
	}
	for i, r := range e.Resistance {
		v.Resistance[i] = int32(r)
	}
	for id := int32(1); id < 32; id++ {
		if e.KnownSpells&(uint32(1)<<uint(id)) != 0 {
			v.Spells = append(v.Spells, id)
		}
	}
	return v
}

// unitSheets builds the world the game builds from m at Normal difficulty,
// the middle choice and the identity (UNIT-GATE-012), and reads each
// placement's actor: the world builder mints placement i as entity i. A map
// the builder refuses yields its error and no sheet.
func unitSheets(m *alm.Map, t *mapload.Table) ([]unitSheet, error) {
	w, err := mapload.FromALMWith(m, t, mapload.DifficultyNormal)
	if err != nil {
		return nil, err
	}
	out := make([]unitSheet, len(m.Units))
	for i := range m.Units {
		id := sim.EntityID(i)
		e, ok := w.Entity(id)
		if !ok {
			return nil, fmt.Errorf("the world holds no actor for placement %d", i)
		}
		s := unitSheet{Values: valuesOf(e), Suppressed: e.SuppressCorpseLoot, Life: lifeOf(e), Decay: int(e.Decay), Facing: drawnFacing(e)}
		s.Worn, _ = w.EquippedItems(id)
		s.Carried, _ = w.CarriedItems(id)
		out[i] = s
	}
	return out, nil
}

// itemList names items in order with the install's own name bytes, "none"
// for none.
func (x *MapInspector) itemList(items []sim.ItemInstance) string {
	var names []string
	for _, it := range items {
		if !it.Empty() {
			names = append(names, itemName(data.ItemCode(it.Code), x.table))
		}
	}
	if len(names) == 0 {
		return x.inspectionUTF8(x.word("editor.unit.none"))
	}
	return strings.Join(names, ", ")
}

// sheetLines are the unit popup's characteristics and loot sections for
// placement u resolved as r, with headings, in the font's encoding: each
// word is encoded and every installed name keeps its own bytes.
func (x *MapInspector) sheetLines(u alm.Unit, r mapload.Resolution, s unitSheet) []ui.InspectionLine {
	v := s.Values
	enc := x.inspectionUTF8
	var out []ui.InspectionLine
	head := func(text string) { out = append(out, ui.InspectionLine{Text: text, Role: "heading"}) }
	line := func(text string) { out = append(out, ui.InspectionLine{Text: text}) }
	head(enc(x.word("editor.unit.sheet")))
	line(enc(x.word("editor.unit.scaling")))
	switch {
	case r.Arm == mapload.ArmUnits && r.Found():
		line(fmt.Sprintf(enc(x.word("editor.unit.row")), r.Index, x.table.Units.EntryName(r.Index)))
	case r.Found():
		line(fmt.Sprintf(enc(x.word("editor.unit.template")), r.Index, x.table.Humans.EntryName(r.Index)))
	default:
		line(enc(x.word("editor.unit.defaults")))
	}
	if r.Arm == mapload.ArmNPC {
		line(fmt.Sprintf(enc(x.word("editor.unit.npc")), u.ClassSubID, r.Index))
	}
	line(fmt.Sprintf(enc(x.word("editor.unit.vitals")), v.HP, v.MaxHP, v.Mana, v.MaxMana))
	line(fmt.Sprintf(enc(x.word("editor.unit.stats")), v.Body, v.Mind, v.Spirit, v.Reaction))
	line(fmt.Sprintf(enc(x.word("editor.unit.combat")), v.ToHit, v.DamageMin, v.DamageMax, v.Defence, v.Absorption))
	line(fmt.Sprintf(enc(x.word("editor.unit.motion")), v.Speed, v.Sight, v.XP))
	p, res, k := v.Protection, v.Resistance, v.Skill
	line(fmt.Sprintf(enc(x.word("editor.unit.protection")), p[0], p[1], p[2], p[3], p[4]))
	line(fmt.Sprintf(enc(x.word("editor.unit.resistance")), res[0], res[1], res[2], res[3], res[4]))
	line(fmt.Sprintf(enc(x.word("editor.unit.skills")), k[0], k[1], k[2], k[3], k[4], k[5]))
	spells := enc(x.word("editor.unit.none"))
	if len(v.Spells) > 0 {
		var names []string
		for _, id := range v.Spells {
			name := fmt.Sprintf("spell %d", id)
			if x.table != nil && x.table.Spells != nil && int(id) < x.table.Spells.Len() {
				name = x.table.Spells.EntryName(int(id))
			}
			names = append(names, name)
		}
		spells = strings.Join(names, ", ")
	}
	line(fmt.Sprintf(enc(x.word("editor.unit.spells")), spells))
	if r.Arm != mapload.ArmUnits {
		line(fmt.Sprintf(enc(x.word("editor.unit.worn")), x.itemList(s.Worn[:])))
	}
	head(enc(x.word("editor.unit.loot")))
	if s.Suppressed {
		line(enc(x.word("editor.unit.loot.nothing")))
	} else {
		weapon := s.Worn[0]
		line(fmt.Sprintf(enc(x.word("editor.unit.loot.carried")), x.itemList(s.Carried)))
		line(fmt.Sprintf(enc(x.word("editor.unit.loot.worn")), x.itemList(s.Worn[1:])))
		switch {
		case weapon.Empty():
			line(fmt.Sprintf(enc(x.word("editor.unit.loot.weapon")), enc(x.word("editor.unit.none"))))
		case weapon.InnateWeapon():
			line(fmt.Sprintf(enc(x.word("editor.unit.loot.innate")), x.itemList(s.Worn[:1])))
		default:
			line(fmt.Sprintf(enc(x.word("editor.unit.loot.weapon")), x.itemList(s.Worn[:1])))
		}
	}
	if v.TypeID > 0x40 && v.GoldChance > 0 {
		line(fmt.Sprintf(enc(x.word("editor.unit.loot.gold")), v.GoldChance, v.GoldMin, v.GoldMax))
	} else {
		line(enc(x.word("editor.unit.loot.nogold")))
	}
	return out
}

// fallHeld is an elapsed count past every fall: the dying block's last frame,
// which the game holds once the fall it starts at mission start has played.
const fallHeld = 1 << 30

// fallenDraw draws placement entity e of class c, which the game spawns not
// alive, as the game draws it once its fall has played: the game's own
// corpse selection at the actor's tier, facing and decay stage, in its
// owner's colours.
func (x *MapInspector) fallenDraw(e *ui.MapEntity, c *terrain.UnitClass, tier int, sheet unitSheet, shade uint32) {
	art, frame, mirror, ok := corpseFrame(c, tier, sheet.Facing>>1, sheet.Decay, fallHeld)
	if !ok {
		return
	}
	e.Art, e.Frame, e.Mirror, e.Life = art, frame, mirror, sheet.Life
	e.DrawCategory = terrain.UnitCategoryFor(c, uint8(sheet.Decay))
	if palette := x.units.OwnerPalette(art, shade); palette != nil {
		tinted := *frame
		tinted.Palette = *palette
		e.Frame = &tinted
	}
}
