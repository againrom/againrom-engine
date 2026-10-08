package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func currentHeroTownFixture(t *testing.T) (*FrontEnd, *ui.App, SaveStore, int) {
	t.Helper()
	f := releaseFront(t)
	entryApp := f.App("current hero mission return")
	if err := entryApp.OpenMission(f.MissionOpenerWith(90, skillCaseParty(f, skillCases()[0]))); err != nil {
		t.Fatal(err)
	}
	open, town, err := f.RestoreOriginal(skillExport(t, f, true, "current hero entry"))
	if err != nil || town {
		t.Fatal("actual current mission cold LOAD", town, err)
	}
	if err := entryApp.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	ms := f.live.mission
	if refusal := f.CampaignSession.carryMissionHome(f.townInstall(), 90, ms.party, f.live.world, ms.ids, nil); refusal != "" {
		t.Fatal(refusal)
	}
	store := SaveStore{Dir: t.TempDir()}
	seed, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	if _, err := seed(false); err != nil {
		t.Fatal(err)
	}
	app := f.App("current town hero")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal("actual town LOAD", err)
	}
	member := -1
	for i := range f.Carried {
		if f.Carried[i].StartingHero {
			member = i
		}
	}
	if member < 0 {
		t.Fatal("actual town lacks starting hero")
	}
	p := &f.Carried[member]
	if p.Carry == nil || p.Carry.NativeHistory == nil || p.Carry.LiveLoad != nil && p.Carry.LiveLoad.Inventory.Source.Class != 0 {
		t.Fatalf("actual returned hero lacks native current history: Carry=%+v", p.Carry)
	}
	return f, app, store, member
}

func TestReleaseTownCurrentHeroFieldsThroughF2(t *testing.T) {
	f, app, store, member := currentHeroTownFixture(t)
	p := &f.Carried[member]
	before := mapload.CloneParty([]mapload.PartyMember{*p})[0]
	p.Hero.Body += 3
	p.Hero.Reaction += 4
	p.Hero.Mind += 5
	p.Hero.Spirit += 6
	p.Hero.Skill = [6]int32{27, 130, 29, 31, 33, 35}
	if err := mapload.UpdatePartyLoad(before, p, f.Table, true, true); err != nil {
		t.Fatal("accepted current Hero mutation", err)
	}
	p.Saved = &mapload.Saved{HP: 17, MaxHP: 257, Mana: 19, MaxMana: 113, HealthRegenPeriod: 73, ManaRegenPeriod: 79}
	if p.Carry.LiveLoad == nil {
		p.Carry.LiveLoad = &sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true}}
	}
	load := p.Carry.LiveLoad
	load.Inventory.OwnWeight, load.Load, load.Capacity, load.Speed = 23, 47, 211, 69
	load.Movement = sim.HumanMovement{Present: true, RawSpeed: 67, NativeSpeed: 69, Load: 47, Capacity: 211}
	if err := load.Validate(); err != nil {
		t.Fatal("explicit current load", err)
	}
	d, _, _ := mapload.PartyDisplayWithTable(*p, f.Table)
	wantStats := [14]uint16{uint16(d.Body), uint16(d.Reaction), uint16(d.Mind), uint16(d.Spirit), 67, 23, 47, 211, 17, 257, 73, 19, 113, 79}
	wantHero := p.Hero
	id := p.ID
	store = SaveStore{Dir: t.TempDir()}
	if output := os.Getenv("AGAINROM_CURRENT_ROUTES_OUT"); output != "" {
		store.Dir = filepath.Join(output, filepath.Base(f.Archives.Root), "town-hero-fields")
		if err := os.MkdirAll(store.Dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil, func(route string, s Snapshot) {
		writeWriterSourceCapture(t, store.Dir, "source-"+route+".json", observeWriterSources(f, s, route))
	})
	raw := cityRosterF2Save(t, app, store, "current-hero")
	assert := func(phase string, front *FrontEnd, raw []byte) {
		t.Helper()
		h := routeBoundHero(t, front, raw)
		names := []string{"Body", "Reaction", "Mind", "Spirit", "Speed", "OwnWeight", "Load", "Capacity", "HP", "MaxHP", "HealthPeriod", "Mana", "MaxMana", "ManaPeriod"}
		for i, want := range wantStats {
			if h.Stats[i] != want {
				t.Errorf("%s %s got %d want current %d", phase, names[i], h.Stats[i], want)
			}
		}
		for slot := range d.Skill {
			if h.SkillLevels[slot] != uint16(d.Skill[slot]) {
				t.Errorf("%s skill%d got %d want current %d", phase, slot, h.SkillLevels[slot], d.Skill[slot])
			}
			if slot > 0 && binary.LittleEndian.Uint16(h.Basis.Human.Fields.Base[2+2*slot:]) != uint16(wantHero.Skill[slot]) {
				t.Errorf("%s trained skill%d lost current %d", phase, slot, wantHero.Skill[slot])
			}
		}
		for _, field := range []struct {
			name string
			at   int
			want int32
		}{{"HealthRegeneration", 10, d.HealthRegeneration}, {"ManaRegeneration", 14, d.ManaRegeneration}} {
			if got := binary.LittleEndian.Uint16(h.Basis.Human.Fields.Modifier[field.at:]); got != uint16(field.want) {
				t.Errorf("%s %s got %d want current %d", phase, field.name, got, field.want)
			}
		}
	}
	assert("town F2", f, raw)
	cold := currentTownReload(t, raw)
	for _, p := range cold.Carried {
		if p.ID == id && !reflect.DeepEqual(p.Hero, wantHero) {
			t.Errorf("cold Hero got %+v want %+v", p.Hero, wantHero)
		}
	}
	assert("cold next SAVE", cold, currentTownSave(t, cold))
	if err := cold.App("current town hero next mission").OpenMission(cold.MissionOpener(90)); err != nil {
		t.Fatal(err)
	}
	current, found := cold.live.world.Entity(cold.live.mission.ids[member])
	if !found {
		t.Fatal("next entry lost exact hero")
	}
	wantStats[4] = uint16(current.Speed)
	assert("next mission SAVE", cold, skillExport(t, cold, true, "current town hero next mission"))
	cold.LiveAdvance(3)
	e, ok := cold.live.world.Entity(cold.live.mission.ids[member])
	if !ok || e.NativeBasis.Body != wantStats[0] || e.NativeBasis.BodyKnown == false || e.MaxHP != 257 || e.MaxMana != 113 || e.Mana != 19 || e.HP != 17 {
		t.Errorf("next three ticks lost current Body/pools: %+v", e)
	}
}

func TestReleaseTownPotionCurrentFieldsThroughF2(t *testing.T) {
	for _, tc := range []struct {
		name   string
		effect sim.ItemEffect
		at     int
	}{{"Body", sim.ItemEffect{Kind: 2, Operand: 3}, 10}, {"HealthRegeneration", sim.ItemEffect{Kind: 8, Mode: 1, Operand: 3 | 20<<16}, 10}, {"ManaRegeneration", sim.ItemEffect{Kind: 11, Mode: 1, Operand: 3 | 20<<16}, 14}} {
		t.Run(tc.name, func(t *testing.T) {
			f, app, store, member := currentHeroTownFixture(t)
			p := &f.Carried[member]
			p.Saved = &mapload.Saved{HP: 17, MaxHP: 257, Mana: 19, MaxMana: 113}
			result, ok := mapload.ApplyTownPotion(*p, sim.ItemInstance{Code: 0xe08, Kind: 3, Effects: []sim.ItemEffect{tc.effect}}, f.Table)
			if !ok || result.Carry.NativeHistory == nil {
				t.Fatal("actual native potion preflight")
			}
			if !commitTownPotion(p, result) {
				t.Fatal("actual native potion commit")
			}
			raw := cityRosterF2Save(t, app, store, "potion-hero")
			h := routeBoundHero(t, f, raw)
			if h.Stats[8] != 17 || h.Stats[9] != 257 || h.Stats[11] != 19 || h.Stats[12] != 113 {
				t.Errorf("potion attachment lost current pools: %v", h.Stats)
			}
			if h.Stats[0] != result.Carry.NativeHistory.Basis.Body {
				t.Errorf("potion Body got %d want current %d", h.Stats[0], result.Carry.NativeHistory.Basis.Body)
			}
			want := binary.LittleEndian.Uint16(result.Carry.NativeHistory.Basis.Modifier[tc.at:])
			if got := binary.LittleEndian.Uint16(h.Basis.Human.Fields.Modifier[tc.at:]); got != want {
				t.Errorf("potion %s got %d want current %d", tc.name, got, want)
			}
		})
	}
}
