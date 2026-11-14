package game

import (
	"bytes"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestEditorIdentityGroupsKeepTableOrderAndEveryRow(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s, _ := unitIdentityEditor(t)
			x := s.inspector
			names := []string{"Dragon.2", "Goblin_Sling.1", "Catapult", "Goblin_Pike.4", "Ballista", "Orc_Bow.1", "Orc_Sword.1", "Troll.1", "Ogre.1", "Bee.1", "Ghost.1", "Bat_Sonic.1", "Turtle.1", "Foot_Animated.1", "Squirrel.1", "Daemon", "Human", "Modded.7"}
			rows := dbCollection{{}}
			for i, name := range names {
				rows = append(rows, unitsRow(name, map[int]int32{0x1d: int32(40 + i), 0x1e: 1}))
			}
			x.table.Units = rows
			humans := x.table.Humans.(dbCollection)
			humans[1].name, humans[2].name = "NPC_Guard", "PC_Courier"
			other := append([]int32(nil), humans[2].params...)
			other[24] = 5678
			x.table.Humans = append(humans, dbEntry{name: "Modded_person", params: other})
			x.recordOptions, x.identities = nil, nil
			if lang == "ru" {
				x.archives.Base.Profile.Language = "russian"
			}
			before := append([]byte(nil), s.model.Bytes()...)
			step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
			heads := []string{"Humans", "Catapults", "Goblins", "Orcs", "Ogres and trolls", "Bats, ghosts and bees", "Squirrels, caterpillars and turtles", "Dragons", "Daemon", "Other creatures", "Story heroes", "Player characters", "Others: fighters"}
			blocks := []string{"Humans and Creatures", "Named characters"}
			if lang == "ru" {
				heads = []string{"Люди", "Катапульты", "Гоблины", "Орки", "Людоеды и тролли", "Мыши, духи и осы", "Белки, гусеницы и черепахи", "Драконы", "Демон", "Другие существа", "Сюжетные герои", "Персонажи игрока", "Прочие: бойцы"}
				blocks = []string{"Люди и Существа", "Именованные персонажи"}
			}
			if len(step.Groups) != len(heads) {
				t.Fatalf("grid has %d groups %+v, want %d headed groups", len(step.Groups), step.Groups, len(heads))
			}
			wantRows := [][]int{{1}, {3, 5}, {2, 4}, {6, 7}, {8, 9}, {10, 11, 12}, {13, 14, 15}, {1}, {16}, {17, 18}, {0, 7, 8}, {2}, {5, 6, 1}}
			var seen []int64
			for i, g := range step.Groups {
				kind, block := mapload.UnitCreature, blocks[0]
				if i == 0 {
					kind = mapload.UnitPerson
				}
				if i >= 10 {
					kind, block = mapload.UnitPerson, blocks[1]
				}
				if i == 10 {
					kind = mapload.UnitNamed
				}
				var want []int64
				for _, row := range wantRows[i] {
					want = append(want, identityValue(int(kind), row))
				}
				if g.Head != x.inspectionUTF8(heads[i]) || g.Block != x.inspectionUTF8(block) || !slices.Equal(g.Values, want) {
					t.Errorf("group %d = %+v, want %q / %q / %v", i, g, block, heads[i], want)
				}
				if i > 0 {
					seen = append(seen, g.Values...)
				}
			}
			var offered []int64
			for _, c := range step.Choices {
				if !slices.Contains(step.Shut, c.Value) {
					offered = append(offered, c.Value)
				}
			}
			slices.Sort(seen)
			slices.Sort(offered)
			if !slices.Equal(seen, offered) {
				t.Fatalf("grouped values %v differ from placeable values %v", seen, offered)
			}
			popup, shut := x.identityChoices(s.mission())
			if !reflect.DeepEqual(popup, step.Choices) || !slices.Equal(shut, step.Shut) || !bytes.Equal(before, s.model.Bytes()) {
				t.Fatal("grouping changed the popup choices or authored bytes")
			}
		})
	}
}

func TestEditorIdentityPicturesUseThePlacedRowsTier(t *testing.T) {
	s, _ := unitIdentityEditor(t)
	x := s.inspector
	x.units.Classes[40] = tierArtClass(2)
	var pictures [][]byte
	for _, row := range []int{1, 2} {
		id := mapload.Identity{Kind: mapload.UnitCreature, Row: row}
		u, ok := mapload.Place(alm.Unit{}, id, x.table, s.mission())
		if !ok {
			t.Fatal("fixture identity cannot be placed", id)
		}
		tier := entityTiers(&alm.Map{Units: []alm.Unit{u}}, x.table)[0]
		c := x.units.Classes[40]
		frames := c.TierFrames(tier)
		n, _ := terrain.SelectUnitFrame(c.Anim, len(frames), false, 0, 0, 0)
		want := frames[n].RGBA()
		pic := s.NewRecordPreview(ui.KindUnit, "identity", identityChoiceValue(id))
		if pic == nil || pic.Bounds() != want.Bounds() || !bytes.Equal(pic.Pix, want.Pix) {
			t.Errorf("row %d preview differs from its placed tier %d", row, tier)
		}
		if pic != nil {
			pictures = append(pictures, pic.Pix)
		}
	}
	if len(pictures) != 2 || bytes.Equal(pictures[0], pictures[1]) {
		t.Fatal("two rows of one class show the same tier colours")
	}
}
