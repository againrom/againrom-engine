package game

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Inspect final ordinary SAV fields before the game's DefRow cosmetics or
// native recompute can conceal them. Original layout/consumer authorities:
// HERO-APPEAR-051, SAV-667, SAV-HUMRUN-444, SAV-CITYSALE-513/515.
func checkNativeCityKnownFields(t *testing.T, f *FrontEnd, p *sav.CityProvenance) {
	t.Helper()
	document := p.Data()
	player := document.Objects[0].Player
	if player == nil || binary.LittleEndian.Uint32(player.Fixed[39:43]) != 95 {
		t.Fatal("fresh Player reserve is not95")
	}
	if len(player.Diary.Words) != f.Table.Units.Len() || len(player.Diary.DWords) != len(player.Diary.Words) || player.Diary.Reference != binary.LittleEndian.Uint32(player.Fixed[47:51]) {
		t.Fatal("Player Diary count/self-owner")
	}
	for i, word := range player.Diary.Words {
		if word != 0x400 || player.Diary.DWords[i] != 0 {
			t.Fatal("fresh Diary defaults", i)
		}
	}
	roster := p.Roster()
	if len(roster) != len(f.Carried) {
		t.Fatal("native Human population", len(roster), len(f.Carried))
	}
	for _, character := range roster {
		var member mapload.PartyMember
		found := false
		for i, m := range f.Carried {
			if nativeCityIdentity(i) == character.Identity {
				member = m
				found = true
				break
			}
		}
		var unit *sav.CityUnitData
		for _, object := range document.Objects {
			if object.Unit != nil && binary.LittleEndian.Uint32(object.Unit.Token[29:33]) == character.Identity {
				unit = object.Unit
				break
			}
		}
		if unit == nil || !found {
			t.Fatal("missing Human identity", character.Identity)
		}
		typeID := binary.LittleEndian.Uint16(unit.Token[17:19])
		face := int(unit.Scalar1[2])
		female, mage := false, false
		if typeID >= 0x20 && typeID < 0x40 {
			female, mage = (typeID-0x21)&1 != 0, (typeID-0x21)&2 != 0
		} else if typeID < 0x1a {
			female, mage, face = face&0x80 != 0, typeID == 0x17 || typeID == 0x18, face&0x7f
		}
		sex, class := "m", "fighter"
		if female {
			sex = "f"
		}
		if mage {
			class = "mage"
		}
		art := fmt.Sprintf("graphics/equipment/%s%s/%d.256", sex, class, face)
		if b, err := f.Archives.Containers.ReadFile(art); err != nil || len(b) == 0 {
			t.Fatalf("wire portrait %s: %v", art, err)
		}
		if face != member.FigureFace || unit.Scalar1[3]&4 != 0 != member.Mage {
			t.Fatal("wire face/class differs", character.Name)
		}
		if _, err := f.Archives.Containers.ReadFile("graphics/equipment/ffighter/0.256"); err == nil {
			t.Fatal("N2 negative path exists")
		}
		source, err := p.Human(character.Identity)
		if err != nil {
			t.Fatal(err)
		}
		h := cityHumanState(character, source)
		d, _, _ := mapload.PartyDisplayWithTable(member, f.Table)
		if h.HealthPeriod != 100 || h.ManaPeriod != 50 {
			t.Fatal("periods replaced by modifier amounts", h.HealthPeriod, h.ManaPeriod)
		}
		if h.Attack.ToHit != uint16(d.Combat.ToHit) || h.Attack.DamageBase != uint8(d.Combat.DamageBase) || h.Attack.Active != uint8(d.Combat.SkillSlot) || h.Defence.Defence != uint16(d.Combat.Defence) || h.Defence.Absorption != uint16(d.Combat.Absorption) {
			t.Fatal("current combat fields", character.Name)
		}
		if h.Modifier.HealthRegeneration != uint16(d.HealthRegeneration) || h.Modifier.ManaRegeneration != uint16(d.ManaRegeneration) {
			t.Fatal("regeneration modifiers", character.Name)
		}
		for i := 1; i <= 5; i++ {
			if h.Base.Skill[i] != uint16(member.Hero.Skill[i]) || h.Attack.Skill[i] != uint16(d.Skill[i]) {
				t.Fatal("maintained/live skill", i)
			}
			price, err := h.TrainingPrice(i)
			if err != nil || price != int32(math.Pow(1.1, float64(member.Hero.Skill[i]))*200) {
				t.Fatal("school price", i, price, err)
			}
		}
		var worn int16
		for _, ref := range append([]uint16{unit.Reference74, unit.Reference78}, unit.Equipment[:12]...) {
			if ref != 0 {
				worn += int16(binary.LittleEndian.Uint16(document.Objects[ref-1].Item.Fields[9:11]))
			}
		}
		wantLoad := worn + int16(int32(unit.ContainerTails[1])/2)
		if h.Weight != uint16(worn) || h.Load != uint16(wantLoad) || h.ManaFloor != uint16(int32(int16(h.ManaMax))*95/100) || h.Sight == 0 || h.MoverSpeed != byte(h.Speed) {
			t.Fatal("weight/load/floor/sight/mover", character.Name, h)
		}
		// Equal-load sale is a no-op; the current stored basis must remain
		// coherent through the derive used by a school purchase.
		if next, err := h.Derive(); err != nil || next != h {
			t.Fatal("derive changes an unchanged generated Human", character.Name, err)
		}
		if next, derived, err := h.RefreshInventoryLoad(); err != nil || derived || next != h {
			t.Fatal("equal-load sale changed Human", err)
		}
		trained, err := h.Train(3)
		if err != nil || trained.Base.Skill[3] != h.Base.Skill[3]+1 || trained.Attack.Skill[3] != h.Attack.Skill[3]+1 {
			t.Fatal("school loses maintained base or equipment bonus", err)
		}
	}
}

func TestReleaseNativeCityKnownHumanFieldsAndSchool(t *testing.T) {
	for _, class := range []int{0, 1} {
		t.Run(fmt.Sprint(class), func(t *testing.T) {
			f := releaseFront(t)
			f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Known fields", Choices: []int{1, class, 3}, Stats: []int{31, 27, 24, 29}})
			m := &f.Carried[0]
			items := mapload.MemberItemEquipment(*m, f.Table)
			kind := uint8(29)
			if m.Mage {
				kind = 35
			}
			items[0].Effects = append(items[0].Effects, sim.ItemEffect{Kind: kind, Operand: 3}, sim.ItemEffect{Kind: 8, Operand: 7}, sim.ItemEffect{Kind: 15, Operand: 4})
			m.WornItems = items
			f.arriveInTown()
			f.addChapterCompanions(f.Town.Chapter())
			dir := t.TempDir()
			save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
			name, err := save(false)
			if err != nil || !IsOriginal(name) {
				t.Fatal("ordinary generated SAV", name, err)
			}
			paths, _ := filepath.Glob(filepath.Join(dir, "*.sav"))
			if len(paths) != 1 {
				t.Fatal(paths)
			}
			b, err := os.ReadFile(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			file, err := sav.Open(b)
			if err != nil {
				t.Fatal(err)
			}
			p, err := file.CityProvenance()
			if err != nil {
				t.Fatal(err)
			}
			checkNativeCityKnownFields(t, f, p)
		})
	}
}

func TestReleaseNativeCityKnownHumanHireFields(t *testing.T) {
	for _, kind := range []int{3, 10, 14} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			f, screen := hireForSiegeOrderTest(t, "Hire fields")
			f.Town.mercEnabled[kind] = true
			if _, ok := screen.toggleMercenary(kind); !ok {
				t.Fatal("hire", kind)
			}
			snapshot, label, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			b, err := f.ExportNativeCitySave(snapshot, label)
			if err != nil {
				t.Fatal(err)
			}
			file, err := sav.Open(b)
			if err != nil {
				t.Fatal(err)
			}
			p, err := file.CityProvenance()
			if err != nil {
				t.Fatal(err)
			}
			checkNativeCityKnownFields(t, f, p)
		})
	}
}
