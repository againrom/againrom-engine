package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// The units-row slot numbers this suite writes the two families into,
// spelled out here rather than imported, on spawn_test.go's own rule: a test
// asserting a family landed in the right array must not read that array's
// position out of the code it is testing. They match data/unitdef.go's slot
// switch, cases 19..23 and 24..28.
const (
	slotProtectionFrom = 19
	slotResistanceFrom = 24
)

// The humans-row slot numbers this suite writes a person's stats and skill
// levels into, for the same reason as above. They match
// data/humandef.go's slot switch, cases 0..3 and 10..15.
const (
	slotPersonBody      = 0
	slotPersonReaction  = 1
	slotPersonMind      = 2
	slotPersonSpirit    = 3
	slotPersonSkillFrom = 10
)

// TestCreatureSheetFamiliesAndSkillPositions is R-1's own mitigation: a
// units row carrying TEN DISTINCT values across the two families, so a
// transposition of Elemental and WeaponKind, or a skill row reading the
// wrong array, shows up as a wrong number rather than as two families that
// happen to agree. It also pins AC-2, AC-3 and FR-4a.
func TestCreatureSheetFamiliesAndSkillPositions(t *testing.T) {
	elemental := [5]int32{11, 12, 13, 14, 15}  // Fire, Water, Air, Earth, Astral
	weaponKind := [5]int32{21, 22, 23, 24, 25} // Blade, Axe, Bludgeon, Pike, Shooting

	row := unitDefRow(50, 0, 30)
	for i, v := range elemental {
		row[slotProtectionFrom+i] = v
	}
	for i, v := range weaponKind {
		row[slotResistanceFrom+i] = v
	}

	tbl := &mapload.Table{Units: defCollection{
		{}, // 0: reserved
		{name: "creature", params: row},
	}}
	m := &alm.Map{Units: []alm.Unit{{ClassID: 50, ClassSubID: 0}}}

	sheets := mapload.PlacedSheets(m, tbl)
	s, ok := sheets[0]
	if !ok {
		t.Fatalf("PlacedSheets did not state a sheet for the one placement")
	}

	if s.Band != mapload.BandCreature {
		t.Errorf("Band = %v, want BandCreature", s.Band)
	}
	if s.Elemental != elemental {
		t.Errorf("Elemental = %v, want %v (the row's Protection columns)", s.Elemental, elemental)
	}
	if s.WeaponKind != weaponKind {
		t.Errorf("WeaponKind = %v, want %v (the row's Resistance columns)", s.WeaponKind, weaponKind)
	}

	wantSkill := [data.SkillSlots]int32{0, weaponKind[0], weaponKind[1], weaponKind[2], weaponKind[3], weaponKind[4]}
	if s.Skill != wantSkill {
		t.Errorf("Skill = %v, want %v", s.Skill, wantSkill)
	}

	// Belt and braces against R-1's own failure mode: the skill positions
	// must NOT hold the elemental family.
	if s.Skill[data.SkillBlade] == elemental[0] {
		t.Errorf("skill position Blade equals the elemental family's own value %d; the two families were crossed", elemental[0])
	}
}

// personRow builds a legal Humans parameter row of the streamed width,
// every cell empty but the stats and the six skill levels — the same
// defRow shape spawn_test.go's own Units rows use, reused here because
// NewHumanDef reads no slot past 25 and a row of rowWidth is still legal
// (data.MinHumanRow is 26).
func personRow(body, reaction, mind, spirit int32, skill [data.SkillSlots]int32, healthMax, manaMax int32) []int32 {
	slots := map[int]int32{
		slotPersonBody: body, slotPersonReaction: reaction, slotPersonMind: mind, slotPersonSpirit: spirit,
		slotHealthMax: healthMax, // slot 4, shared with the units row's own HealthMax slot
		5:             manaMax,
	}
	for i, v := range skill {
		slots[slotPersonSkillFrom+i] = v
	}
	return defRow(slots)
}

// TestPersonSheetStatsSkillsFamiliesExperience is SC-3, AC-4: a person's
// four statistics come out capped, his six skill positions are his row's
// own levels rather than the graph's restored ones, and both families and
// the experience are the graph's.
//
// ONE SKILL LEVEL IS WRITTEN OUTSIDE [SkillFloor, SkillCap] ON PURPOSE
// (slot Shooting, -7): the graph's own restore would clamp it to 0
// (data/recompute.go step 6a), so if personSheet ever read derived.Skill
// instead of the row's own d.Skill, this is the one value in this row that
// would show it.
func TestPersonSheetStatsSkillsFamiliesExperience(t *testing.T) {
	body, reaction, mind, spirit := int32(80), int32(45), int32(60), int32(20) // Body and Mind exceed StatCap (50)
	skill := [data.SkillSlots]int32{7, 3, 4, 5, 6, -7}
	row := personRow(body, reaction, mind, spirit, skill, 100, 50)
	row[slotHumanType] = 5 // the humans-by-type search's own key column

	tbl := &mapload.Table{Humans: defCollection{
		{}, // 0: reserved
		{name: "person", params: row},
	}}
	m := &alm.Map{Units: []alm.Unit{{ClassID: 5}}} // below unitsKeyFloor: the humans band, by type

	sheets := mapload.PlacedSheets(m, tbl)
	s, ok := sheets[0]
	if !ok {
		t.Fatalf("PlacedSheets did not state a sheet for the one placement")
	}
	if s.Band != mapload.BandPerson {
		t.Fatalf("Band = %v, want BandPerson", s.Band)
	}

	// The reference: the SAME row, taken through NewHumanDef and Recompute
	// directly, with an empty Loadout — exactly what personSheet is
	// supposed to wire up. This does not re-derive the graph's arithmetic
	// (that is recompute_test.go's job); it pins that PlacedSheets reaches
	// this same graph, over this same row, with this same Loadout.
	d, err := data.NewHumanDef("person", row)
	if err != nil {
		t.Fatalf("data.NewHumanDef: %v", err)
	}
	want := d.Hero().Recompute(d.Profile(), data.Loadout{})

	if s.Body != want.Body || s.Reaction != want.Reaction || s.Mind != want.Mind || s.Spirit != want.Spirit {
		t.Errorf("stats = {%d %d %d %d}, want the capped {%d %d %d %d}",
			s.Body, s.Reaction, s.Mind, s.Spirit, want.Body, want.Reaction, want.Mind, want.Spirit)
	}
	if s.Body != 50 || s.Mind != 50 {
		t.Errorf("Body/Mind = %d/%d, want both capped to 50", s.Body, s.Mind)
	}

	if s.Skill != skill {
		t.Errorf("Skill = %v, want the row's own %v (unclamped)", s.Skill, skill)
	}
	if s.Skill == want.Skill {
		t.Errorf("Skill equals the graph's restored/clamped Skill %v; the row's own levels were not carried through", want.Skill)
	}

	if s.Elemental != want.Protection {
		t.Errorf("Elemental = %v, want the derived Protection %v", s.Elemental, want.Protection)
	}
	if s.WeaponKind != want.Resistance {
		t.Errorf("WeaponKind = %v, want the derived Resistance %v", s.WeaponKind, want.Resistance)
	}
	if s.Experience != want.Experience {
		t.Errorf("Experience = %d, want the graph's own %d", s.Experience, want.Experience)
	}
}

func TestPersonSheetIgnoresEquipment(t *testing.T) {
	body, reaction, mind, spirit := int32(30), int32(30), int32(20), int32(20)
	skill := [data.SkillSlots]int32{5, 1, 2, 3, 4, 5}
	row := personRow(body, reaction, mind, spirit, skill, 100, 50)

	// Two entries on an otherwise identical row, one bare and one carrying
	// equipment names in its trailing strings, keyed apart by server id
	// (slot 0x18) so each placement below reaches its own entry rather
	// than the ascending search's first match on a shared type.
	tbl := &mapload.Table{Humans: defCollection{
		{}, // 0: reserved
		{name: "bare", params: withServerID(row, 101)},
		{name: "equipped", params: withServerID(row, 102), strings: []string{"a-sword", "a-shield", "a-helmet"}},
	}}
	m := &alm.Map{Units: []alm.Unit{{DefID: 101}, {DefID: 102}}}

	sheets := mapload.PlacedSheets(m, tbl)
	sBare, ok := sheets[0]
	if !ok {
		t.Fatalf("no sheet for the bare placement")
	}
	sEquipped, ok := sheets[1]
	if !ok {
		t.Fatalf("no sheet for the equipped placement")
	}

	if sBare != sEquipped {
		t.Errorf("bare sheet %+v != equipped sheet %+v; equipment moved a value FR-5a says it must not", sBare, sEquipped)
	}
}

// withServerID returns a copy of row with slot 0x18 (the humans search's
// own server-id column) set to id, so two otherwise identical rows can be
// told apart by data.FindHumanByServerID.
func withServerID(row []int32, id int32) []int32 {
	out := append([]int32(nil), row...)
	out[slotServerID] = id
	return out
}

func TestPlacedSheetsSkipsAnUnresolvedPlacement(t *testing.T) {
	tbl := &mapload.Table{Units: defCollection{
		{},
		{name: "u", params: unitDefRow(50, 0, 30)},
	}}
	m := &alm.Map{Units: []alm.Unit{
		{ClassID: 50, ClassSubID: 0}, // resolves
		{ClassID: 99, ClassSubID: 9}, // resolves to nothing
	}}

	sheets := mapload.PlacedSheets(m, tbl)
	if _, ok := sheets[0]; !ok {
		t.Errorf("placement 0 should have a sheet")
	}
	if _, ok := sheets[1]; ok {
		t.Errorf("placement 1 resolved to nothing and must have no sheet, got %+v", sheets[1])
	}
	if len(sheets) != 1 {
		t.Errorf("len(sheets) = %d, want 1", len(sheets))
	}
}

// TestPlacedSheetsWithNoMapOrTable is AC-9: a nil map, a nil table and a
// table naming neither collection all state no character and none panics.
func TestPlacedSheetsWithNoMapOrTable(t *testing.T) {
	realMap := &alm.Map{Units: []alm.Unit{{ClassID: 50, ClassSubID: 0}, {ClassID: 6}}}

	if got := mapload.PlacedSheets(nil, &mapload.Table{}); got != nil {
		t.Errorf("PlacedSheets(nil map, ...) = %v, want nil", got)
	}
	if got := mapload.PlacedSheets(realMap, nil); len(got) != 0 {
		t.Errorf("PlacedSheets(map, nil table) = %v, want empty", got)
	}
	if got := mapload.PlacedSheets(realMap, &mapload.Table{}); len(got) != 0 {
		t.Errorf("PlacedSheets(map, table with neither collection) = %v, want empty", got)
	}
	// And an empty map, over a real table, is empty rather than a panic on
	// an empty range.
	if got := mapload.PlacedSheets(&alm.Map{}, &mapload.Table{Units: defCollection{{}}}); len(got) != 0 {
		t.Errorf("PlacedSheets(empty map, ...) = %v, want empty", got)
	}
}
