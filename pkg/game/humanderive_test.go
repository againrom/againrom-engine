package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/rules"
)

// humanRouteCase is one Human fed to both derived-stat routes: the native
// graph (data.Hero.RecomputeWithSkillXP) and the stored-state derive
// (data.HumanState.Derive). load is the carried load the speed step reads.
type humanRouteCase struct {
	label   string
	hero    data.Hero
	profile data.Profile
	loadout data.Loadout
	xp      [data.SkillSlots]int32
	load    int32
}

type humanRouteDiff struct {
	field          string
	native, stored int32
}

// nativeHumanState is the stored-state form of a native case: each stat word
// as an effect writes it (the stat plus its bonus, at most 100) with the bonus
// in the cap byte, the trained levels in the maintained block, the worn
// items' and the weapon's additive terms in the modifier block.
func nativeHumanState(c humanRouteCase, d data.Derived) data.HumanState {
	mod := c.loadout.Mod
	base := [4]int32{c.hero.Body, c.hero.Reaction, c.hero.Mind, c.hero.Spirit}
	bonus := [4]int32{mod.Body, mod.Reaction, mod.Mind, mod.Spirit}
	var stat [4]uint16
	for i := range base {
		stat[i] = uint16(min(min(base[i], data.StatCap)+bonus[i], 100))
	}
	h := data.HumanState{Body: stat[0], Reaction: stat[1], Mind: stat[2], Spirit: stat[3],
		Fighter: c.profile.Fighter, TypeID: 0x21, Experience: uint32(d.Experience), InventoryWeight: 2 * c.load}
	if c.profile.Rider {
		h.TypeID = 0x13
	}
	if c.profile.ManaColumn {
		h.ManaMax = 1
	}
	for i := range bonus {
		h.Modifier.StatCap[i] = int8(bonus[i])
	}
	h.Attack.Skill[0] = uint16(c.hero.Skill[0])
	for i := range h.Base.Skill {
		if i != 0 {
			h.Base.Skill[i] = uint16(c.hero.Skill[i])
		}
		h.Modifier.Attack.Skill[i] = uint16(mod.SkillBonus[i])
		h.SkillXP[i] = uint32(d.SkillXP[i])
	}
	h.Attack.Active = uint8(d.Combat.SkillSlot)
	weapon := data.FoldWeapon(data.Combat{}, c.loadout.Weapon, c.hero.Skill[0])
	m := &h.Modifier
	m.Speed, m.HealthMax, m.ManaMax, m.Sight = uint16(mod.Speed), uint16(mod.HealthMax), uint16(mod.ManaMax), uint16(mod.Sight*256)
	m.Attack.ToHit = uint16(mod.ToHit + weapon.ToHit)
	m.Attack.DamageBase, m.Attack.DamageSpread = uint8(mod.DamageBase+weapon.DamageBase), uint8(mod.DamageSpread+weapon.DamageSpread)
	m.Defence.Defence, m.Defence.Absorption = uint16(mod.Defence+weapon.Defence), uint16(mod.Absorption+weapon.Absorption)
	for i := range mod.Protection {
		m.Defence.Protection[i+1], m.Defence.Resistance[i+1] = uint16(mod.Protection[i]), uint8(mod.Resistance[i])
	}
	return h
}

// storedHumanCase is the native form of a stored Human: the stat less its cap
// byte as the trained statistic and the cap byte as the bonus, the maintained
// levels and the modifier block as the worn items' terms, and a weapon with no
// additive terms that names the stored active slot.
func storedHumanCase(label string, h data.HumanState) humanRouteCase {
	s := func(v uint16) int32 { return int32(int16(v)) }
	m := h.Modifier
	c := humanRouteCase{label: label,
		profile: data.Profile{Fighter: h.Fighter, ManaColumn: h.ManaMax != 0, HealthColumn: true, Rider: data.RiderTypeID(int32(h.TypeID))},
		load:    int32(int16(h.Weight)) + h.InventoryWeight/2}
	if h.InventoryWeight >= 64000 {
		c.load = 32000
	}
	c.hero = data.Hero{Body: s(h.Body) - int32(m.StatCap[0]), Reaction: s(h.Reaction) - int32(m.StatCap[1]),
		Mind: s(h.Mind) - int32(m.StatCap[2]), Spirit: s(h.Spirit) - int32(m.StatCap[3])}
	mod := &c.loadout.Mod
	mod.Body, mod.Reaction, mod.Mind, mod.Spirit = int32(m.StatCap[0]), int32(m.StatCap[1]), int32(m.StatCap[2]), int32(m.StatCap[3])
	c.hero.Skill[0] = s(h.Attack.Skill[0])
	for i := range c.hero.Skill {
		if i != 0 {
			c.hero.Skill[i] = s(h.Base.Skill[i])
		}
		mod.SkillBonus[i] = s(m.Attack.Skill[i])
		c.xp[i] = int32(h.SkillXP[i])
	}
	mod.Speed, mod.HealthMax, mod.ManaMax, mod.Sight = s(m.Speed), s(m.HealthMax), s(m.ManaMax), s(m.Sight)>>8
	mod.HealthRegeneration, mod.ManaRegeneration = s(m.HealthRegeneration), s(m.ManaRegeneration)
	mod.ToHit, mod.DamageBase, mod.DamageSpread = s(m.Attack.ToHit), int32(m.Attack.DamageBase), int32(m.Attack.DamageSpread)
	mod.Defence, mod.Absorption = s(m.Defence.Defence), s(m.Defence.Absorption)
	for i := range mod.Protection {
		mod.Protection[i], mod.Resistance[i] = s(m.Defence.Protection[i+1]), int32(m.Defence.Resistance[i+1])
	}
	if h.Attack.Active >= 1 && h.Attack.Active <= 5 {
		c.loadout.Weapon = &data.Weapon{AttackType: int32(h.Attack.Active)}
	}
	return c
}

// nativeSpeedWord is the native route's speed word: the unencumbered sum with
// the overload step pkg/sim applies (humanSpeedWord).
func nativeSpeedWord(d data.Derived, load int32) (word, kept int32) {
	if d.Speed <= 0 || d.Capacity <= 0 {
		return d.Speed, d.SpeedModifier
	}
	w, m, _ := rules.HumanSpeed(int16(d.Speed-d.SpeedModifier), int16(d.SpeedModifier), int16(load), int16(d.Capacity))
	return int32(w), int32(m)
}

// compareHumanRoutes derives c on both routes and lists every field whose
// values differ. stored, when non-nil, is the stored Human c was read from;
// the stored route then derives it unmapped.
func compareHumanRoutes(c humanRouteCase, stored *data.HumanState) ([]humanRouteDiff, error) {
	d := c.hero.RecomputeWithSkillXP(c.profile, c.loadout, c.xp)
	h := nativeHumanState(c, d)
	if stored != nil {
		h = *stored
	}
	n, err := h.Derive()
	if err != nil {
		return nil, err
	}
	s := func(v uint16) int32 { return int32(int16(v)) }
	word, kept := nativeSpeedWord(d, c.load)
	pairs := []humanRouteDiff{
		{"Body", d.Body, s(n.Body)}, {"Reaction", d.Reaction, s(n.Reaction)}, {"Mind", d.Mind, s(n.Mind)}, {"Spirit", d.Spirit, s(n.Spirit)},
		{"HealthMax", d.HealthMax, s(n.HealthMax)}, {"ManaMax", d.ManaMax, s(n.ManaMax)},
		{"Sight", d.Sight, s(n.Sight) >> 8}, {"Capacity", d.Capacity, s(n.Capacity)},
		{"UnencumberedSpeed", d.Speed - d.SpeedModifier + kept, n.NativeMovementBase()}, {"SpeedWord", word, s(n.Speed)}, {"SpeedModifier", kept, s(n.Modifier.Speed)},
		{"ToHit", d.Combat.ToHit, s(n.Attack.ToHit)},
		{"DamageBase", d.Combat.DamageBase, int32(n.Attack.DamageBase)}, {"DamageSpread", d.Combat.DamageSpread, int32(n.Attack.DamageSpread)},
		{"Defence", d.Combat.Defence, s(n.Defence.Defence)}, {"Absorption", d.Combat.Absorption, s(n.Defence.Absorption)},
	}
	for i := range d.Skill {
		pairs = append(pairs, humanRouteDiff{fmt.Sprintf("Skill%d", i), d.Skill[i], s(n.Attack.Skill[i])})
	}
	for i := range d.Protection {
		pairs = append(pairs, humanRouteDiff{fmt.Sprintf("Protection%d", i+1), d.Protection[i], s(n.Defence.Protection[i+1])},
			humanRouteDiff{fmt.Sprintf("Resistance%d", i+1), d.Resistance[i], int32(n.Defence.Resistance[i+1])})
	}
	var out []humanRouteDiff
	for _, p := range pairs {
		if p.native != p.stored {
			out = append(out, p)
		}
	}
	return out, nil
}

// humanRouteTally collects differences by field over a population.
type humanRouteTally struct {
	cases  int
	failed int
	fields map[string][]string
}

func (r *humanRouteTally) add(t *testing.T, c humanRouteCase, stored *data.HumanState) {
	t.Helper()
	r.cases++
	diffs, err := compareHumanRoutes(c, stored)
	if err != nil {
		r.failed++
		t.Logf("%s: stored derive refused: %v", c.label, err)
		return
	}
	if r.fields == nil {
		r.fields = map[string][]string{}
	}
	for _, d := range diffs {
		r.fields[d.field] = append(r.fields[d.field], fmt.Sprintf("%s native=%d stored=%d", c.label, d.native, d.stored))
	}
}

func (r *humanRouteTally) report(t *testing.T, population string) int {
	t.Helper()
	names := make([]string, 0, len(r.fields))
	for name := range r.fields {
		names = append(names, name)
	}
	sort.Strings(names)
	total := 0
	for _, name := range names {
		rows := r.fields[name]
		total += len(rows)
		shown := rows
		if len(shown) > 4 {
			shown = shown[:4]
		}
		t.Logf("%s: field %s differs in %d case(s): %s", population, name, len(rows), strings.Join(shown, "; "))
	}
	t.Logf("%s: %d case(s), %d refused by the stored derive, %d differing field value(s)", population, r.cases, r.failed, total)
	return total
}

// humanRouteSyntheticCases are the load, modifier and rider cases.
func humanRouteSyntheticCases() []humanRouteCase {
	hero := data.Hero{Body: 30, Reaction: 25, Mind: 20, Spirit: 35, Skill: [data.SkillSlots]int32{4, 12, 0, 7, 0, 3}}
	sword := &data.Weapon{AttackType: 1, DamageBase: 3, DamageSpread: 4, ToHit: 5}
	var out []humanRouteCase
	add := func(label string, mutate func(*humanRouteCase)) {
		c := humanRouteCase{label: label, hero: hero, profile: data.Profile{Fighter: true, HealthColumn: true}, loadout: data.Loadout{Weapon: sword}}
		mutate(&c)
		for i, level := range c.hero.Skill {
			c.xp[i] = data.SkillXPFor(level)
		}
		out = append(out, c)
	}
	add("bare", func(c *humanRouteCase) { c.loadout.Weapon = nil })
	add("sword", func(*humanRouteCase) {})
	add("mage", func(c *humanRouteCase) { c.profile = data.Profile{HealthColumn: true, ManaColumn: true} })
	add("rider", func(c *humanRouteCase) { c.profile.Rider = true })
	add("load below capacity", func(c *humanRouteCase) { c.load = 300 })
	add("load at capacity", func(c *humanRouteCase) { c.load = 301 })
	add("overload", func(c *humanRouteCase) { c.load = 4000 })
	add("overload under haste", func(c *humanRouteCase) { c.load = 4000; c.loadout.Mod.Speed = 4 })
	add("overload under slow", func(c *humanRouteCase) { c.load = 4000; c.loadout.Mod.Speed = -9 })
	add("speed bonus", func(c *humanRouteCase) { c.loadout.Mod.Speed = 3 })
	add("stat bonus", func(c *humanRouteCase) { c.loadout.Mod.Body, c.loadout.Mod.Reaction = 5, 20 })
	add("stat above base cap", func(c *humanRouteCase) { c.hero.Body = 50; c.loadout.Mod.Body = 30 })
	add("sight bonus", func(c *humanRouteCase) { c.loadout.Mod.Sight = 2 })
	add("defence and protection", func(c *humanRouteCase) {
		c.loadout.Mod.Defence, c.loadout.Mod.Absorption = 6, 2
		c.loadout.Mod.Protection = [5]int32{10, 80, -30, 0, 5}
		c.loadout.Mod.Resistance = [5]int32{1, 2, 3, 4, 5}
	})
	add("negative defence", func(c *humanRouteCase) { c.loadout.Mod.Defence = -40 })
	add("pool bonus", func(c *humanRouteCase) { c.loadout.Mod.HealthMax, c.loadout.Mod.ManaMax = 7, 9 })
	add("skill bonus", func(c *humanRouteCase) { c.loadout.Mod.SkillBonus = [data.SkillSlots]int32{0, 5, 0, 0, 0, 0} })
	add("skill bonus above 100", func(c *humanRouteCase) {
		c.hero.Skill[1] = 98
		c.loadout.Mod.SkillBonus[1] = 10
	})
	add("General bonus", func(c *humanRouteCase) { c.loadout.Mod.SkillBonus[0] = 6 })
	add("trained", func(c *humanRouteCase) { c.hero.Skill = [data.SkillSlots]int32{0, 40, 30, 20, 10, 5} })
	add("strong", func(c *humanRouteCase) { c.hero.Body, c.loadout.Mod.Body = 50, 50 })
	return out
}

// TestHumanRoutesCompareSynthetic feeds both derived-stat routes the load,
// modifier and rider cases and reports every field that differs.
func TestHumanRoutesCompareSynthetic(t *testing.T) {
	var tally humanRouteTally
	for _, c := range humanRouteSyntheticCases() {
		tally.add(t, c, nil)
	}
	tally.report(t, "synthetic")
}

// TestReleaseHumanRoutesCompare feeds both derived-stat routes every Humans
// template of the install, as placed with its own starting equipment, the
// same template overloaded under each speed modifier sign and as a rider, and
// every Human the save corpus holds.
func TestReleaseHumanRoutesCompare(t *testing.T) {
	f := releaseFront(t)
	humans := f.Table.Humans
	var templates humanRouteTally
	itemBonus := 0
	for i := 1; i < humans.Len(); i++ {
		name := humans.EntryName(i)
		def, err := data.NewHumanDef(name, humans.EntryParams(i))
		if err != nil {
			continue
		}
		worn, _, weapon, err := mapload.HumanRowEquipment(humans.EntryStrings(i), f.Table)
		if err != nil {
			continue
		}
		loadout, ok := mapload.ResolveItemLoadout(worn, weapon, false, f.Table)
		if !ok {
			loadout = data.Loadout{Weapon: weapon}
		}
		mapload.ApplyItemEffects(&loadout, worn, def.Profile().Fighter)
		if loadout.Mod.SkillBonus[data.SkillGeneral] != 0 {
			itemBonus++
		}
		c := humanRouteCase{label: fmt.Sprintf("row %d %s", i, name), hero: def.Hero(), profile: def.Profile(), loadout: loadout}
		for s, level := range def.Skill {
			c.xp[s] = loadout.Rules.SkillXPExtended(level)
		}
		templates.add(t, c, nil)
		for _, v := range []struct {
			label       string
			speed, load int32
			rider       bool
		}{{"overloaded", 0, 4000, false}, {"overloaded hasted", 4, 4000, false}, {"overloaded slowed", -9, 4000, false}, {"rider", 0, 0, true}} {
			o := c
			o.label += " " + v.label
			o.loadout.Mod.Speed += v.speed
			o.load = v.load
			o.profile.Rider = o.profile.Rider || v.rider
			templates.add(t, o, nil)
		}
	}
	templates.report(t, "Humans templates")
	t.Logf("Humans templates whose starting items carry a General skill bonus: %d", itemBonus)

	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Log("no AGAINROM_SAVE_CORPUS: the corpus Humans are not compared")
		return
	}
	var stored, unmapped humanRouteTally
	files, refused := 0, 0
	err := filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".sav") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source, err := sav.Open(raw)
		if err != nil {
			refused++
			return nil
		}
		holdings, err := source.ActorHoldings()
		if err != nil {
			refused++
			return nil
		}
		files++
		rel, _ := filepath.Rel(corpus, path)
		for _, a := range holdings {
			b := a.Basis
			if b == nil || b.Class != "Human" {
				continue
			}
			h := basisHumanState(*b)
			c := storedHumanCase(fmt.Sprintf("%s actor %d", filepath.ToSlash(rel), a.MapUnitID), h)
			unmapped.add(t, c, &h)
			stored.add(t, c, nil)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("save corpus: %d file(s) read, %d refused", files, refused)
	unmapped.report(t, "corpus Humans, stored state as loaded")
	stored.report(t, "corpus Humans, stored state rebuilt from the native inputs")
}

// basisHumanState is a SAV actor basis as the stored-state derive reads it.
func basisHumanState(b sav.ActorBasis) data.HumanState {
	c := sav.CityCharacter{Stats: b.Stats, SkillXP: b.SkillXP, Experience: b.Experience}
	return cityHumanState(c, b.Human)
}
