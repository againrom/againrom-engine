package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func currentSkillBook(t *testing.T, raw []byte) (*sav.DocumentData, *sav.DocumentRecordData, *sav.DocumentRecordData) {
	t.Helper()
	doc, actor := trainingActor(t, raw, 0)
	refs, ok := savedObjectRefs(actor, "Spells")
	if !ok || len(refs) < 2 || refs[1] == 0 {
		t.Fatal("Fire Ball ordinary book binding", ok, refs)
	}
	return doc, actor, &doc.Objects[refs[1]-1]
}

func currentSkillRangeCheck(t *testing.T, f *FrontEnd, live, ordinary int32, bonus int16) string {
	t.Helper()
	e := releaseEntity(t, f.live, f.live.mission.ids[0])
	rule, ok := f.live.world.Spell(2)
	if !ok || rule.MaxRange != 10 || rule.School != 1 || e.Skill[1] != 100+int32(bonus) {
		t.Fatal("installed Fire Ball/skill fixture", rule, e.Skill)
	}
	if got := sim.SpellCharacteristicsFor(f.live.world.Rules(), e, rule).Range; got != int64(live) {
		t.Fatalf("live Fire Ball reach %d, want %d", got, live)
	}
	before := f.live.world.Hash()
	path := saveCorpseMission(t, f, t.TempDir())
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, actor, spell := currentSkillBook(t, raw)
	level := int16(binary.LittleEndian.Uint16(modBlock(actor, "UA6")[4:]))
	base := int16(binary.LittleEndian.Uint16(modBlock(actor, "U114")[4:]))
	modifier := int16(binary.LittleEndian.Uint16(modBlock(actor, "UD4")[22:]))
	reach, _ := savedStructureValue(spell, "S09")
	if level != 100 || base != 100 || modifier != bonus || reach != uint32(ordinary) {
		t.Fatalf("ordinary Fire Ball: UA6=%d U114=%d UD4=%d S09=%d; want 100/100/%d/%d (live %d)", level, base, modifier, reach, bonus, ordinary, live)
	}
	if bytes.Contains(raw, modMarkName) || f.live.world.Hash() != before {
		t.Fatal("ordinary SAVE added a mod mark or changed the live world")
	}
	return path
}

func TestReleaseCurrentSkillBookRangeProjection(t *testing.T) {
	for _, tc := range []struct {
		name           string
		mind           int32
		bonus          int16
		live, ordinary int32
	}{{"low Mind lifted school", 10, 10, 13, 12}, {"no bonus", 10, 0, 12, 12}, {"power ceiling", 40, 10, 14, 13}} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := f.ChargenParty(ui.ChargenResult{Name: "current reach", Choices: []int{1, 1, 0}, Stats: []int{31, 27, 24, 29}})
			party[0].Hero.Skill[1], party[0].Hero.Mind, party[0].KnownSpells = 100, tc.mind, 1<<2
			if tc.bonus != 0 {
				item := mapload.SourceConstructedItem(sim.PlainItem(0xec7e), f.Table)
				item.Effects = []sim.ItemEffect{{Kind: 33, Operand: uint32(tc.bonus)}}
				party[0].WornItems[11], party[0].Worn[11] = item, item.Code
			}
			if err := f.App("current reach").OpenMission(f.MissionOpenerWith(20, party)); err != nil {
				t.Fatal(err)
			}
			path := currentSkillRangeCheck(t, f, tc.live, tc.ordinary, tc.bonus)
			out := filepath.Join(t.TempDir(), "second.sav")
			cmd := exec.Command(os.Args[0], "-test.run=^TestCurrentSkillColdProcess$", "-test.v")
			cmd.Env = append(os.Environ(), "AGAINROM_CURRENT_SKILL_INPUT="+path, "AGAINROM_CURRENT_SKILL_OUTPUT="+out,
				"AGAINROM_CURRENT_SKILL_LIVE="+strconv.Itoa(int(tc.live)), "AGAINROM_CURRENT_SKILL_ORDINARY="+strconv.Itoa(int(tc.ordinary)), "AGAINROM_CURRENT_SKILL_BONUS="+strconv.Itoa(int(tc.bonus)))
			if log, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("cold process: %v\n%s", err, log)
			} else {
				t.Log(string(log))
			}
			currentSkillRangeCheck(t, loadAreaContinuation(t, out), tc.live, tc.ordinary, tc.bonus)
		})
	}
}

func TestCurrentSkillColdProcess(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("set AGAINROM_ASSETS to a lawful install")
	}
	path := os.Getenv("AGAINROM_CURRENT_SKILL_INPUT")
	if path == "" {
		t.Skip("cold-process child")
	}
	number := func(name string) int32 {
		x, err := strconv.Atoi(os.Getenv(name))
		if err != nil {
			t.Fatal(err)
		}
		return int32(x)
	}
	f := loadAreaContinuation(t, path)
	for range 8 {
		f.live.tick()
	}
	second := currentSkillRangeCheck(t, f, number("AGAINROM_CURRENT_SKILL_LIVE"), number("AGAINROM_CURRENT_SKILL_ORDINARY"), int16(number("AGAINROM_CURRENT_SKILL_BONUS")))
	raw, err := os.ReadFile(second)
	if err == nil {
		err = os.WriteFile(os.Getenv("AGAINROM_CURRENT_SKILL_OUTPUT"), raw, 0600)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func currentSkillClassAction(t *testing.T, f *FrontEnd, fighter, award bool) {
	t.Helper()
	id := f.live.mission.ids[0]
	e := releaseEntity(t, f.live, id)
	bonus := int32(70)
	if fighter {
		bonus = 10
	}
	base := e.NativeTraining.Levels[1]
	check := func(want int32) {
		e = releaseEntity(t, f.live, id)
		if e.NativeClass != (sim.NativeClass{Present: true, Fighter: fighter}) || e.Skill[1] != want || f.live.world.NativeTrainingNeedsProducer(id) || (e.MaxMana > 0) != fighter {
			t.Fatal("class-selected current sheet", e.NativeClass, e.Skill, e.MaxMana, want)
		}
	}
	check(base + bonus)
	f.live.invSubjectSet, f.live.invSubject.ID = true, uint32(id)
	f.live.enqueueUnequip(11)
	for range 4 {
		f.live.tick()
	}
	check(base)
	stacks, _ := f.live.world.CarriedStacks(id)
	index := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == 0xec7e })
	if index < 0 {
		t.Fatal("unequipped class item is absent")
	}
	f.live.enqueueEquip(index)
	for range 4 {
		f.live.tick()
	}
	check(base + bonus)
	if !award {
		return
	}
	var victim sim.Entity
	distance := int32(1 << 30)
	for _, candidate := range f.live.world.Entities() {
		dx, dy := candidate.X-e.X, candidate.Y-e.Y
		if candidate.HP > 0 && candidate.Domain == e.Domain && f.live.world.Relations().Hostile(e.Owner, candidate.Owner) && candidate.XPValue > 0 && dx*dx+dy*dy < distance {
			distance, victim = dx*dx+dy*dy, candidate
		}
	}
	if victim.MaxHP == 0 || e.XPSlot != 1 {
		t.Fatal("class award lacks eligible target/Blade", e.XPSlot)
	}
	if err := f.live.world.HeadlessPlace(id, victim.X-1, victim.Y); err != nil {
		t.Fatal(err)
	}
	for range 2400 {
		f.live.strike(uint32(id), uint32(victim.ID))
		f.live.tick()
		after := releaseEntity(t, f.live, id)
		if after.HP <= 0 {
			t.Fatal("class award actor fell")
		}
		if after.NativeTraining.Levels[1] > base {
			if after.NativeTraining.Levels[1] != base+1 || after.Skill[1] != base+1+bonus || after.SkillXP[1] <= e.SkillXP[1] {
				t.Fatal("class-selected physical award", after.Skill, after.NativeTraining, after.SkillXP)
			}
			t.Logf("fighter with mana: base%d→%d, effective%d→%d; actual unequip/equip/strike", base, base+1, e.Skill[1], after.Skill[1])
			return
		}
	}
	t.Fatal("eligible physical award did not train the fighter")
}

func TestCurrentSkillClassColdProcess(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("set AGAINROM_ASSETS to a lawful install")
	}
	path := os.Getenv("AGAINROM_CURRENT_CLASS_INPUT")
	if path == "" {
		t.Skip("cold-process child")
	}
	f := loadAreaContinuation(t, path)
	currentSkillClassAction(t, f, true, false)
	raw, err := os.ReadFile(saveCorpseMission(t, f, t.TempDir()))
	if err == nil {
		err = os.WriteFile(os.Getenv("AGAINROM_CURRENT_CLASS_OUTPUT"), raw, 0600)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestReleaseCurrentSkillClassOrdinaryBit(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "class bit", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
	party[0].Hero.Mind, party[0].Hero.Body = 400, 300
	party[0].Profile.Fighter, party[0].Profile.ManaColumn = false, true
	item := mapload.SourceConstructedItem(sim.PlainItem(0xec7e), f.Table)
	item.Effects = []sim.ItemEffect{{Kind: 27, Operand: 10}, {Kind: 33, Operand: 70}}
	party[0].WornItems[11], party[0].Worn[11] = item, item.Code
	party[0].Carry = &mapload.Carry{Equipped: party[0].Worn, EquippedItems: party[0].WornItems}
	party[0].Carry.SkillXP[1] = (sim.Rules{}).SkillXP(10)
	if err := f.App("class bit").OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	path := saveCorpseMission(t, f, t.TempDir())
	raw, _ := os.ReadFile(path)
	doc, actor := trainingActor(t, raw, 0)
	leaf, _, _ := sav.NativeActions(doc.State)
	flags, _ := savedStructureValue(actor, "U4C")
	savedObjectSetValue(actor, "U4C", flags&^4)
	binary.LittleEndian.PutUint16(modBlock(actor, "UA6")[4:], 20)
	binary.LittleEndian.PutUint16(modBlock(actor, "UD4")[22:], 10)
	unchanged, _, _ := sav.NativeActions(doc.State)
	if !bytes.Equal(leaf, unchanged) {
		t.Fatal("class control changed continuation")
	}
	raw, err := sav.EncodeDocumentData(*doc)
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(t.TempDir(), "fighter-with-mana.sav")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	cold := loadAreaContinuation(t, path)
	id := cold.live.mission.ids[0]
	e := releaseEntity(t, cold.live, id)
	if !cold.live.mission.party[0].Profile.Fighter || e.MaxMana <= 0 || e.Skill[1] != 20 {
		t.Fatal("ordinary fighter-with-mana fixture", e.MaxMana, e.Skill)
	}
	if cold.live.world.NativeTrainingNeedsProducer(id) {
		t.Fatal("class-selected worn fighter bonus misclassified as an unknown producer")
	}
	currentSkillClassAction(t, cold, true, true)
	path = saveCorpseMission(t, cold, t.TempDir())
	out := filepath.Join(t.TempDir(), "second-class.sav")
	cmd := exec.Command(os.Args[0], "-test.run=^TestCurrentSkillClassColdProcess$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_CURRENT_CLASS_INPUT="+path, "AGAINROM_CURRENT_CLASS_OUTPUT="+out)
	if log, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cold class process: %v\n%s", err, log)
	} else {
		t.Log(string(log))
	}
	second := loadAreaContinuation(t, out)
	e = releaseEntity(t, second.live, id)
	if e.NativeClass != (sim.NativeClass{Present: true, Fighter: true}) || e.NativeTraining.Levels[1] != 11 || e.Skill[1] != 21 || e.MaxMana <= 0 {
		t.Fatal("second class SAVE", e.NativeClass, e.Skill, e.NativeTraining, e.MaxMana)
	}
	raw, _ = os.ReadFile(out)
	_, actor = trainingActor(t, raw, 0)
	flags, _ = savedStructureValue(actor, "U4C")
	if flags&4 != 0 {
		t.Fatal("second SAVE changed ordinary fighter bit", flags)
	}
}

func TestReleaseCurrentSkillInverseClass(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "mage without mana", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
	item := mapload.SourceConstructedItem(sim.PlainItem(0xec7e), f.Table)
	item.Effects = []sim.ItemEffect{{Kind: 27, Operand: 10}, {Kind: 33, Operand: 70}}
	party[0].WornItems[11], party[0].Worn[11] = item, item.Code
	party[0].Carry = &mapload.Carry{Equipped: party[0].Worn, EquippedItems: party[0].WornItems}
	if err := f.App("inverse class").OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(saveCorpseMission(t, f, t.TempDir()))
	doc, actor := trainingActor(t, raw, 0)
	flags, _ := savedStructureValue(actor, "U4C")
	savedObjectSetValue(actor, "U4C", flags|4)
	binary.LittleEndian.PutUint16(modBlock(actor, "UA6")[4:], 80)
	binary.LittleEndian.PutUint16(modBlock(actor, "UD4")[22:], 70)
	raw, err := sav.EncodeDocumentData(*doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "mage-without-mana.sav")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		cold := loadAreaContinuation(t, path)
		if cold.live.mission.party[0].Profile.Fighter {
			t.Fatal("ordinary mage bit lost")
		}
		currentSkillClassAction(t, cold, false, false)
		path = saveCorpseMission(t, cold, t.TempDir())
	}
}

func TestReleaseCurrentSkillImportedBookWeaponAlias(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "retained book", Choices: []int{1, 1, 0}, Stats: []int{31, 27, 24, 29}})
	party[0].Hero.Skill[1], party[0].Hero.Mind, party[0].KnownSpells = 100, 10, 1<<2
	weapon := mapload.SourceConstructedItem(sim.PlainItem(0x1101), f.Table)
	party[0].WornItems[0], party[0].Worn[0] = weapon, weapon.Code
	if err := f.App("retained book").OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	book := sim.Spellbook{State: sim.BookPresent}
	book.Slots[1] = sim.BookSpell{Range: 17, Defensive: 3, ManaCost: 123}
	raw, err := os.ReadFile(saveCorpseMission(t, f, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	doc, actor, spell := currentSkillBook(t, raw)
	mustSetValue(spell, "S09", 17)
	mustSetValue(spell, "S0A", 3)
	mustSetValue(spell, "S0C", 123)
	bookRefs, _ := savedObjectRefs(actor, "Spells")
	weaponRefs, ok := savedObjectRefs(actor, "HeldWeapon")
	if !ok || len(weaponRefs) != 1 || weaponRefs[0] == 0 {
		t.Fatal("retained book fixture has no ordinary weapon", weaponRefs)
	}
	mustSetRefs(&doc.Objects[weaponRefs[0]-1], "WeaponSpell", []uint16{bookRefs[1]})
	indexed, _, err := sav.ReindexDocumentData(*doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(indexed)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "aliased.sav")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for cycle := range 3 {
		cold := loadAreaContinuation(t, path)
		e := releaseEntity(t, cold.live, cold.live.mission.ids[0])
		if e.Book.State != sim.BookPresent || e.Book.Slots[1] != book.Slots[1] {
			t.Fatal("imported book cache was normalized", e.Book)
		}
		if cycle == 2 {
			book.Slots[1].Range = 12
			if !cold.live.world.SetSkillLevels(e.ID, e.Skill) {
				t.Fatal("current skill refresh actor is absent")
			}
		}
		before := cold.live.world.Hash()
		path = saveCorpseMission(t, cold, t.TempDir())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		doc, actor, spell := currentSkillBook(t, raw)
		bookRefs, _ = savedObjectRefs(actor, "Spells")
		weaponRefs, _ = savedObjectRefs(actor, "HeldWeapon")
		spellRefs, _ := savedObjectRefs(&doc.Objects[weaponRefs[0]-1], "WeaponSpell")
		if len(spellRefs) != 1 || spellRefs[0] == 0 || (bookRefs[1] == spellRefs[0]) != (cycle < 2) {
			t.Fatal("book/weapon alias ownership", cycle, bookRefs, spellRefs)
		}
		weaponSpell, err := savedSpellRecord(&doc.Objects[spellRefs[0]-1])
		bookSpell, bookErr := savedSpellRecord(spell)
		if err != nil || bookErr != nil || weaponSpell.Value.Range != 17 || bookSpell.Value.Range != book.Slots[1].Range || bookSpell.Value.Defensive != 3 || bookSpell.Value.ManaCost != 123 || cold.live.world.Hash() != before {
			t.Fatal("retained caches changed", weaponSpell, bookSpell, err, bookErr)
		}
	}
	cold := loadAreaContinuation(t, path)
	if e := releaseEntity(t, cold.live, cold.live.mission.ids[0]); e.Book.Slots[1] != book.Slots[1] {
		t.Fatal("split imported book failed cold LOAD", e.Book)
	}
}

func TestReleaseCurrentSkillHistoricalClassDefault(t *testing.T) {
	for _, historical := range []bool{false, true} {
		t.Run(map[bool]string{false: "explicit absence", true: "old current fields"}[historical], func(t *testing.T) {
			f := releaseFront(t)
			party := f.ChargenParty(ui.ChargenResult{Name: "class default", Choices: []int{1, 1, 0}, Stats: []int{31, 27, 24, 29}})
			if err := f.App("class default").OpenMission(f.MissionOpenerWith(20, party)); err != nil {
				t.Fatal(err)
			}
			id := f.live.mission.ids[0]
			if !f.live.world.SetNativeClass(id, sim.NativeClass{}) {
				t.Fatal("class absence fixture")
			}
			raw, err := os.ReadFile(saveCorpseMission(t, f, t.TempDir()))
			if err != nil {
				t.Fatal(err)
			}
			doc, actor := trainingActor(t, raw, 0)
			a, err := readCurrentActions(doc)
			if err != nil || a == nil || len(a.Party) != 1 || !a.Party[0].Base.LegacyClass || a.Values[id].NativeClassPresent {
				t.Fatal("current absence marker", a, err)
			}
			if historical {
				a.Party[0].Base.LegacyClass = false
				flags, _ := savedStructureValue(actor, "U4C")
				savedObjectSetValue(actor, "U4C", flags&^4)
				leaf, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
					t.Fatal(err)
				}
			}
			raw, err = sav.EncodeDocumentData(*doc)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "class-default.sav")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				cold := loadAreaContinuation(t, path)
				e := releaseEntity(t, cold.live, cold.live.mission.ids[0])
				want := sim.NativeClass{}
				if historical {
					want = sim.NativeClass{Present: true, Fighter: true}
				}
				if e.NativeClass != want || e.MaxMana <= 0 {
					t.Fatal("historical class default", e.NativeClass, e.MaxMana, want)
				}
				path = saveCorpseMission(t, cold, t.TempDir())
			}
		})
	}
}
