package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func city1099Fixture(t *testing.T) []byte {
	t.Helper()
	d := cityfixture.City(false)
	for _, obj := range d.Objects {
		if u := obj.Unit; u != nil {
			for i, v := range []uint16{41, 35, 20, 15, 19, 40, 90, 411, 110, 130, 100, 0, 0, 50} {
				binary.LittleEndian.PutUint16(u.Scalar2[2*i:], v)
			}
			binary.LittleEndian.PutUint16(u.RawA6, 25)
			binary.LittleEndian.PutUint16(u.RawA6[4:], 11) // live Blade includes modifier
			binary.LittleEndian.PutUint16(u.Raw114[4:], 9)
			binary.LittleEndian.PutUint16(u.RawD4[22:], 2)
			u.RawA6[16], u.RawA6[22], u.RawA6[23] = 1, 0xab, 0xcd
			u.Raw114[2], u.Raw114[22], u.Raw114[23] = 0x77, 0x12, 0x34
			u.RawD4[34], u.RawD4[40], u.RawD4[41] = 5, 0xde, 0xad
			u.Raw154[10], u.Raw154[179], u.Raw158[147] = 19, 0xef, 0xed
			binary.LittleEndian.PutUint32(u.XP[4:], 1400)
			binary.LittleEndian.PutUint32(u.Scalar2[35:], 1400)
			binary.LittleEndian.PutUint16(u.Scalar2[32:], 1587)
			u.ContainerTails[1] = 100
		}
	}
	p, err := sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	u := sav.CityUpdate{Label: []byte("training fixture"), Money: 3000}
	for _, c := range p.Roster() {
		u.Characters = append(u.Characters, originalCityBaselineUpdate(c))
	}
	raw, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCurrentCityTrainingComparesHeldWeaponExceptDisplayName(t *testing.T) {
	const code = 0x0103
	actual := mapload.PartyMember{ID: "hero", Weapon: &data.Weapon{Code: code, Name: "Iron Short Sword", DamageBase: 5, ToHit: 5}}
	actual.WornItems[0] = sim.PlainItem(code)
	expected := actual
	base := *actual.Weapon
	base.Name = "Common Iron Short Sword"
	expected.Weapon = &base
	if equalTrainingMember(actual, expected) || !equalCurrentCityTrainingMember(actual, expected) {
		t.Fatal("current-city display literal did not stay separate from the strict baseline")
	}
	for _, change := range []struct {
		name   string
		mutate func(*mapload.PartyMember)
	}{
		{"code", func(p *mapload.PartyMember) { p.Weapon.Code++ }},
		{"damage", func(p *mapload.PartyMember) { p.Weapon.DamageBase++ }},
		{"equipment", func(p *mapload.PartyMember) { p.WornItems[0] = sim.PlainItem(0x0109) }},
		{"character", func(p *mapload.PartyMember) { p.ID = "other" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			changed := expected
			weapon := *expected.Weapon
			changed.Weapon = &weapon
			change.mutate(&changed)
			if equalCurrentCityTrainingMember(actual, changed) {
				t.Fatal("changed member passed the school baseline")
			}
		})
	}
	bare := actual
	bare.WornItems[0] = sim.ItemInstance{}
	bareExpected := expected
	bareExpected.WornItems[0] = sim.ItemInstance{}
	if equalCurrentCityTrainingMember(bare, bareExpected) {
		t.Fatal("name without a held item passed the school baseline")
	}
}

func TestTrainedCityWholeSemanticDiffPreservesOpaqueFields(t *testing.T) {
	source := city1099Fixture(t)
	f := city1099Front(t, source)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	current, err := f.ExportOriginalSave(s, "before training")
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(current)
	if err != nil {
		t.Fatal(err)
	}
	p, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	want := p.Data()
	binary.LittleEndian.PutUint32(want.Objects[want.Players[0]-1].Player.Fixed[21:], 2529^0x5c073f4d)
	sourceFile, _ := sav.Open(source)
	sourceP, _ := sourceFile.CityProvenance()
	assertTrainingOpaqueFields(t, sourceP.Data(), want)
	for _, obj := range want.Objects {
		if v := obj.Unit; v != nil && v.Name == "Leader" {
			// Literal expected stores from the independent fixture: no production
			// derive or field-update helper constructs this whole-document answer.
			binary.LittleEndian.PutUint16(v.RawA6, 51)
			binary.LittleEndian.PutUint16(v.RawA6[4:], 12)
			v.RawA6[14], v.RawA6[15] = 4, 2
			binary.LittleEndian.PutUint16(v.Raw114[4:], 10)
			binary.LittleEndian.PutUint16(v.RawBE, 11)
			for i := 1; i <= 5; i++ {
				binary.LittleEndian.PutUint16(v.RawBE[4+2*i:], 7)
			}
			binary.LittleEndian.PutUint32(v.XP[4:], 1594)
			binary.LittleEndian.PutUint32(v.Scalar2[35:], 1594)
		}
	}
	if msg := train1099(t, f, "hero", 0); !strings.HasPrefix(msg, "trained ") {
		t.Fatal(msg)
	}
	s, _, err = f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportOriginalSave(s, "typed field diff")
	if err != nil {
		t.Fatal(err)
	}
	gotFile, _ := sav.Open(raw)
	gotP, _ := gotFile.CityProvenance()
	got := gotP.Data()
	if !reflect.DeepEqual(got, want) {
		currentItemFieldDiagnostics(t, "City", reflect.ValueOf(want), reflect.ValueOf(got))
		t.Fatal("current document differs outside the independent training field diff")
	}
	reloadTrainingCity(t, f)
	if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 13 for 518" || f.Town.Gold() != 2011 {
		t.Fatal("cold training continuation", msg, f.Town.Gold())
	}
}

// assertTrainingOpaqueFields checks the document fields a town SAVE keeps and
// that every source actor is still written. An actor's and the Player's other
// fields are built from the party, not taken from the loaded town.
func assertTrainingOpaqueFields(t *testing.T, source, current sav.CityData) {
	t.Helper()
	if source.Counter04 != current.Counter04 || source.Counter00 != current.Counter00 || source.Head != current.Head ||
		source.Marker != current.Marker || source.GlobalDWord != current.GlobalDWord || !bytes.Equal(source.TrailerState, current.TrailerState) {
		t.Fatal("current city lost unowned document fields")
	}
	for _, old := range source.Objects {
		if old.Unit == nil {
			continue
		}
		found := 0
		for _, object := range current.Objects {
			if object.Unit != nil && object.Unit.Name == old.Unit.Name {
				found++
			}
		}
		if found != 1 {
			t.Fatal("current city lost or repeated actor", old.Unit.Name, found)
		}
	}
	if len(source.Objects[source.Players[0]-1].Player.Groups) != len(current.Objects[current.Players[0]-1].Player.Groups) {
		t.Fatal("current city lost a Player group")
	}
}

func TestTrainedCityPurchaseKeepsCurrentSecondPhysicalModifier(t *testing.T) {
	for _, offset := range []int{17, 18} {
		t.Run(strconv.Itoa(offset), func(t *testing.T) {
			file, _ := sav.Open(city1099Fixture(t))
			p, _ := file.CityProvenance()
			d := p.Data()
			for _, obj := range d.Objects {
				if obj.Unit != nil {
					obj.Unit.RawD4[18+offset] = 5
				}
			}
			p, err := sav.CityFromData(d)
			if err != nil {
				t.Fatal(err)
			}
			u := sav.CityUpdate{Label: []byte("second physical"), Money: 3000}
			for _, c := range p.Roster() {
				u.Characters = append(u.Characters, originalCityBaselineUpdate(c))
			}
			raw, err := p.Marshal(u)
			if err != nil {
				t.Fatal(err)
			}
			f := city1099Front(t, raw)
			buyTrainingItem(t, f)
			before := currentCitySaleHuman(t, trainingPartyMember(t, f, "hero"))
			graph := f.Town.cityObjects.Clone()
			if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 12 for 471" || f.Town.Gold() != 2519 {
				t.Fatal("current second component blocked training", msg, f.Town.Gold())
			}
			got := currentCitySaleHuman(t, trainingPartyMember(t, f, "hero"))
			wantPair := [2]byte{}
			wantPair[offset-17] = 5
			if got.Modifier != before.Modifier || [2]byte{got.Attack.SecondBase, got.Attack.SecondSpread} != wantPair ||
				got.Attack.Skill[1] != 12 || got.Base.Skill[1] != 10 || got.SkillXP[1] != 1594 || got.InventoryWeight != 101 ||
				!reflect.DeepEqual(f.Town.cityObjects, graph) {
				t.Fatal("training lost current modifier, progression or Item identity", got)
			}
			reloadTrainingCity(t, f)
			if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 13 for 518" || f.Town.Gold() != 2001 {
				t.Fatal("cold second-component continuation", msg, f.Town.Gold())
			}
			reloadTrainingCity(t, f)
		})
	}
}

func city1099Front(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	rows := make(dbCollection, 30)
	rows[28].name, rows[29].name = "PC_Leader", "PC_Companion"
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: &mapload.Table{Humans: rows}}}
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("city LOAD: %t %v", town, err)
	}
	return f
}

func currentTrainingCity(t *testing.T) *FrontEnd {
	t.Helper()
	return city1099Front(t, city1099Fixture(t))
}

func train1099(t *testing.T, f *FrontEnd, id string, cell int) string {
	t.Helper()
	s := f.TownScreen().(*townScreen)
	s.room, s.schoolCell = roomSchool, cell
	for i, m := range f.Carried {
		if m.ID == id {
			s.shopMember = i
		}
	}
	return s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false).Msg
}

func trainingPartyMember(t *testing.T, f *FrontEnd, id string) mapload.PartyMember {
	t.Helper()
	for _, m := range f.Carried {
		if m.ID == id {
			return m
		}
	}
	t.Fatal("missing member", id)
	return mapload.PartyMember{}
}

func TestTrainedCityNativeAndSAVContinueByPartyIdentity(t *testing.T) {
	f := currentTrainingCity(t)
	companionID := f.Carried[1].ID // this synthetic table has no NPC registry
	f.Carried[0], f.Carried[1] = f.Carried[1], f.Carried[0]
	companion := trainingPartyMember(t, f, companionID)
	if msg := train1099(t, f, "hero", 0); !strings.Contains(msg, "to 12 for 471") {
		t.Fatal(msg)
	}
	if f.Town.Gold() != 2529 || !reflect.DeepEqual(companion, trainingPartyMember(t, f, companionID)) {
		t.Fatal("wrong purse or training targeted roster ordinal")
	}
	hero := trainingPartyMember(t, f, "hero")
	if hero.OriginalHuman == nil || hero.Hero.Skill[1] != 12 || hero.Carry.SkillXP[1] != 1594 || hero.OriginalHuman.State.Base.Skill[1] != 10 {
		t.Fatal("incomplete trained state")
	}
	d, hp, _ := mapload.PartySpawnWithTable(hero, f.Table)
	if d.Combat.ToHit != 51 || d.Combat.DamageBase != 4 || hp != 110 || d.HealthMax != 130 {
		t.Fatalf("current projection %+v hp=%d", d, hp)
	}
	panel := partyPanelSubject(hero, f.Table)
	if panel.HP != 110 || panel.MaxHP != 130 || panel.Char.Skills[1] != 12 {
		t.Fatalf("town panel %+v", panel)
	}
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	native, err := EncodeSave(snapshot, "trained city")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(native)
	if err != nil {
		t.Fatal(err)
	}
	fresh := currentTrainingCity(t)
	if _, town, err := fresh.Restore(decoded); err != nil || !town {
		t.Fatalf("AGS continuation %t %v", town, err)
	}
	if !reflect.DeepEqual(trainingPartyMember(t, fresh, "hero"), hero) {
		t.Fatal("native lost typed current state")
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := fresh.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !strings.HasSuffix(name, ".sav") {
		t.Fatalf("ordinary SAVE %q %v", name, err)
	}
	output, err := os.ReadFile(filepath.Join(store.Dir, name))
	if err != nil || !bytes.HasPrefix(output, []byte(sav.Magic)) {
		t.Fatal("SAVE did not publish Asg&", err)
	}
	next := city1099Front(t, output)
	if !reflect.DeepEqual(trainingPartyMember(t, next, "hero"), hero) {
		t.Fatalf("SAV lost trained state\ngot %+v\nwant %+v", trainingPartyMember(t, next, "hero"), hero)
	}
	if msg := train1099(t, next, "hero", 0); !strings.Contains(msg, "to 13 for 518") {
		t.Fatal(msg)
	}
	if next.Town.Gold() != 2011 {
		t.Fatal("repeated training wrong purse", next.Town.Gold())
	}
	// The other character can train independently after source order and live
	// order both disagree. No first-Human or first-roster shortcut is valid.
	if msg := train1099(t, next, companionID, 0); !strings.Contains(msg, "to 12 for 471") {
		t.Fatal(msg)
	}
	if trainingPartyMember(t, next, "hero").Hero.Skill[1] != 13 {
		t.Fatal("companion training changed hero")
	}
}

func TestTrainedCityEverySchoolSlotSurvivesFreshSAV(t *testing.T) {
	for slot := 1; slot <= 5; slot++ {
		t.Run(strconv.Itoa(slot), func(t *testing.T) {
			f := currentTrainingCity(t)
			before := trainingPartyMember(t, f, "hero")
			wantSkills := [6]int32{0, 11}
			wantXP := [6]int32{0, 1400}
			wantBase, wantPrice, wantNext, wantAggregate := uint16(1), 200, 220, uint32(1501)
			wantSkills[slot], wantXP[slot] = 1, 101
			if slot == 1 {
				wantSkills[1], wantXP[1] = 12, 1594
				wantBase, wantPrice, wantNext, wantAggregate = 10, 471, 518, 1594
			}
			if msg := train1099(t, f, "hero", slot-1); !strings.HasPrefix(msg, "trained ") {
				t.Fatal(msg)
			}
			got := trainingPartyMember(t, f, "hero")
			h, ok := got.OriginalHumanState()
			if !ok || got.Hero.Skill != wantSkills || got.Carry.SkillXP != wantXP ||
				h.Base.Skill[slot] != wantBase || h.Experience != wantAggregate ||
				f.Town.Gold() != 3000-wantPrice || memberSchoolPrice(got, slot) != wantNext {
				t.Fatalf("slot %d has wrong independent skill/XP/base/price projection: %+v", slot, got)
			}
			if h.Base.Skill[0] != before.OriginalHuman.State.Base.Skill[0] || h.Attack.Skill[0] != before.OriginalHuman.State.Attack.Skill[0] {
				t.Fatal("school exposed or modified General")
			}
			snapshot, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportOriginalSave(snapshot, "every school slot")
			if err != nil {
				t.Fatal(err)
			}
			fresh := city1099Front(t, raw)
			if !reflect.DeepEqual(trainingPartyMember(t, fresh, "hero"), got) {
				t.Fatal("fresh SAV lost school state")
			}
		})
	}
}

func TestTrainedHumanPanelsUseStoredSpeedAndIndependentAggregate(t *testing.T) {
	f := currentTrainingCity(t)
	p := trainingPartyMember(t, f, "hero")
	h := p.OriginalHuman.State
	h.Experience = 1500 // deliberately not sum(SkillXP), which is 1400
	h.InventoryWeight, h.Modifier.Speed, h.Modifier.Capacity = 900, 2, 1000
	h, err := h.Train(1)
	if err != nil || h.Experience != 1694 || h.Speed != 20 {
		t.Fatalf("independent aggregate/order fixture: %+v %v", h, err)
	}
	mapload.ApplyOriginalHuman(&p, h)
	panel := partyPanelSubject(p, f.Table)
	if panel.Char.Experience != 1694 || panel.Speed != 20 || panel.Weight != 490 {
		t.Fatalf("town panel replaced stored fields: %+v", panel)
	}
	e := sim.Entity{ID: 9, X: 1, Y: 1, HP: 110, MaxHP: 130, Speed: 21, Capacity: 1411, SkillXP: p.Carry.SkillXP, Skill: p.Hero.Skill}
	w, err := sim.NewWorld(7, sim.Bounds{Width: 4, Height: 4}, sim.ModeCanonical, nil, []sim.Entity{e})
	if err != nil {
		t.Fatal(err)
	}
	w.SetHumanMovement(9, 20, 490)
	assertPanel := func(world *sim.World, member mapload.PartyMember) {
		t.Helper()
		mw := &mapWorld{world: world, mission: &missionNotices{party: []mapload.PartyMember{member}, ids: []sim.EntityID{9}},
			chars: map[sim.EntityID]ui.UnitCharacter{9: partyPanelSubject(member, f.Table).Char}}
		draws := mw.entityDraws()
		if len(draws) != 1 || draws[0].Speed != 20 || draws[0].Char.Experience != 1694 {
			t.Fatalf("mission panel replaced stored fields: %+v", draws)
		}
		changed := world.Entities()[0]
		changed.SkillXP[1]++
		if got := mw.retainedHumanExperience(changed, 1595); got != 1595 {
			t.Fatalf("changed XP retained stale aggregate: %d", got)
		}
		changed.HumanMovement = sim.HumanMovement{}
		if got := humanSpeedStat(changed); got != 21 {
			t.Fatalf("retired movement retained stale statistic: %d", got)
		}
	}
	assertPanel(w, p)
	world, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(Snapshot{Mission: 10, Party: []mapload.PartyMember{p}, World: world}, "retained panel")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	var fresh sim.World
	if err := fresh.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	assertPanel(&fresh, snapshot.Party[0])
}

func TestTrainedCityMalformedCurrentInputIsAtomic(t *testing.T) {
	for name, mutate := range map[string]func(*mapload.PartyMember){
		"body":         func(m *mapload.PartyMember) { m.Hero.Body++ },
		"item":         func(m *mapload.PartyMember) { m.Carry.Items = append(m.Carry.Items, 7) },
		"active skill": func(m *mapload.PartyMember) { m.Carry.LiveLoad.Inventory.Source.Attack[16] = 6 },
		"load width":   func(m *mapload.PartyMember) { m.Carry.LiveLoad.Load = 65536 },
	} {
		t.Run(name, func(t *testing.T) {
			f := currentTrainingCity(t)
			mutate(&f.Carried[0])
			before, graph, gold := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.Town.Gold()
			if msg := train1099(t, f, "hero", 0); strings.HasPrefix(msg, "trained ") {
				t.Fatal("malformed current state trained", msg)
			}
			if !reflect.DeepEqual(before, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || gold != f.Town.Gold() {
				t.Fatal("rejected training mutated party, graph or purse")
			}
			if name == "body" {
				return
			}
			s, _, err := f.Snapshot(false)
			if err != nil {
				return
			}
			if raw, err := f.ExportOriginalSave(s, "malformed"); err == nil || len(raw) != 0 {
				t.Fatal("malformed current state emitted SAV", len(raw), err)
			}
			if !reflect.DeepEqual(before, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || gold != f.Town.Gold() {
				t.Fatal("SAVE changed malformed live input")
			}
		})
	}
}

func TestCurrentCityContradictoryHumanRefusesAtomically(t *testing.T) {
	f := currentTrainingCity(t)
	f.Carried[0].Hero.Body++
	before, graph, gold := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.Town.Gold()
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	const want = `city Human "hero" invariant: Hero disagrees with current actor load`
	if raw, err := f.ExportCurrentSave(s, "contradictory Human"); err == nil || err.Error() != want || len(raw) != 0 {
		t.Fatal("contradictory current mirrors were normalized", len(raw), err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	if name, err := save(false); name != "" || err == nil || err.Error() != want {
		t.Fatal("ordinary SAVE did not preserve the invariant failure", name, err)
	}
	if files, err := os.ReadDir(store.Dir); err != nil || len(files) != 0 {
		t.Fatal("invariant failure published a file", files, err)
	}
	if !reflect.DeepEqual(before, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || gold != f.Town.Gold() {
		t.Fatal("rejected SAVE changed live input")
	}
}

func TestCurrentCityInvalidLoadNeverFallsThroughHumanConstruction(t *testing.T) {
	f := currentTrainingCity(t)
	member := trainingPartyMember(t, f, "hero")
	member.Carry.LiveLoad.Load = 65536
	prior := mapload.CloneParty([]mapload.PartyMember{member})[0]
	unit := cityfixture.City(false).Objects[1].Unit
	unchanged := cityfixture.City(false).Objects[1].Unit
	const want = `city Human "hero" invariant: invalid current load: sim: original actor load/capacity exceeds signed word`
	if err := applyCurrentCityHuman(unit, member, f.Table); err == nil || err.Error() != want {
		t.Fatal("invalid current load reached Human construction", err)
	}
	if !reflect.DeepEqual(prior, member) || !reflect.DeepEqual(unit, unchanged) {
		t.Fatal("failed Human construction changed its inputs")
	}
}

func TestCurrentCityNativeHumanWithoutLoadStillSaves(t *testing.T) {
	f, _ := shopRoom(t, nil)
	f.Town.open = true
	member := &f.Carried[0]
	member.ID, member.Name = "hero", "Native Hero"
	member.StartingHero, member.PlayerCharacter, member.Profile.Fighter = true, true, true
	member.Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	if member.Carry == nil || member.Carry.LiveLoad != nil || member.OriginalHuman != nil {
		t.Fatal("native fixture acquired a retained Human or actor load")
	}
	before := mapload.CloneParty(f.Carried)
	want, hp, mp := mapload.PartyDisplayWithTable(*member, f.Table)
	raw := cityProjectionSave(t, f)
	fresh := &FrontEnd{InstallResources: f.InstallResources}
	if _, town, err := fresh.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("native current SAV without actor load", town, err)
	}
	after := trainingPartyMember(t, fresh, "hero")
	got, gotHP, gotMP := mapload.PartyDisplayWithTable(after, fresh.Table)
	if after.Hero != before[0].Hero || got != want || gotHP != hp || gotMP != mp ||
		after.Carry != nil && after.Carry.LiveLoad != nil || mapload.HasSourceActor(after) {
		t.Fatal("native absence of actor load changed current fields or acquired source state")
	}
	if !reflect.DeepEqual(before, f.Carried) {
		t.Fatal("native SAVE changed live input")
	}
}

func TestTrainedCityRetainedFieldsCannotReplaceCurrentHuman(t *testing.T) {
	for name, mutate := range map[string]func(*mapload.PartyMember){
		"modifier": func(m *mapload.PartyMember) { m.OriginalHuman.State.Modifier.Attack.Tail[0]++ },
		"base":     func(m *mapload.PartyMember) { m.OriginalHuman.State.Base.Skill[1]++ },
		"identity": func(m *mapload.PartyMember) { m.OriginalHuman.PartyID = "npc:22" },
	} {
		t.Run(name, func(t *testing.T) {
			f := currentTrainingCity(t)
			before := currentCitySaleHuman(t, f.Carried[0])
			mutate(&f.Carried[0])
			if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 12 for 471" || f.Town.Gold() != 2529 {
				t.Fatal("retained fields displaced current training", msg, f.Town.Gold())
			}
			got := currentCitySaleHuman(t, f.Carried[0])
			if got.Modifier != before.Modifier || got.Base.Skill[1] != 10 || got.Attack.Skill[1] != 12 ||
				got.SkillXP[1] != 1594 || got.Experience != 1594 || f.Carried[0].OriginalHuman.PartyID != "hero" {
				t.Fatal("training read retained fields", got)
			}
			reloadTrainingCity(t, f)
		})
	}
}

func TestTrainedCityLegacyNativeDefaultsAndMalformedHistory(t *testing.T) {
	f := currentTrainingCity(t)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	for i := range s.Party {
		s.Party[i].OriginalHuman = nil
	}
	for i := range s.OriginalCity.Bindings {
		s.OriginalCity.Bindings[i].Baseline.OriginalHuman = nil
	}
	if _, _, err := f.Restore(s); err != nil {
		t.Fatal("legacy baseline rejected", err)
	}
	if msg := train1099(t, f, "hero", 0); !strings.HasPrefix(msg, "trained ") {
		t.Fatal(msg)
	}
	for _, slots := range [][]uint8{{0}, {6}, make([]uint8, maxCityTraining+1)} {
		s.OriginalCity.Bindings[0].Training = slots
		before := f.Town
		if _, _, err := f.Restore(s); err == nil {
			t.Fatal("invalid history accepted")
		}
		if f.Town != before {
			t.Fatal("invalid history committed a candidate")
		}
	}
}

func TestSchoolTrainsNativeStateCharacterOfAnOriginalCity(t *testing.T) {
	f := currentTrainingCity(t)
	for i := range f.Carried {
		if f.Carried[i].ID == "hero" {
			carry := *f.Carried[i].Carry
			carry.LiveLoad = nil
			f.Carried[i].Carry = &carry
			f.Carried[i].OriginalHuman = nil
			f.Carried[i].Hero.Spirit--
		}
	}
	gold := f.Town.Gold()
	s := f.TownScreen().(*townScreen)
	s.room, s.schoolCell, s.shopMember = roomSchool, 0, 0
	msg := s.trainHeroSkill(1)
	hero := trainingPartyMember(t, f, "hero")
	if !strings.HasPrefix(msg, "trained ") || f.Town.Gold() >= gold || hero.Hero.Skill[1] != 12 {
		t.Fatalf("native-state character was not trained: %q gold %d->%d skill %d", msg, gold, f.Town.Gold(), hero.Hero.Skill[1])
	}
}

func TestTrainingPostedLineShowsFaultsAndHidesPlayerRefusals(t *testing.T) {
	for msg, want := range map[string]string{
		"trained Air to 3 for 200":                                  "trained Air to 3 for 200",
		"SAV training requires unchanged items and character state": "SAV training requires unchanged items and character state",
		"training costs 500":                                        "",
		"cannot train that skill":                                   "",
	} {
		if got := trainingPostedLine(msg); got != want {
			t.Errorf("trainingPostedLine(%q) = %q, want %q", msg, got, want)
		}
	}
}
