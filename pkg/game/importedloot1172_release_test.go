package game

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type lootOracle1172 struct {
	Human       returnOracle1169
	OwnerKey    uint32
	Present     bool
	InsertIndex uint32
	Accumulator int32
	Items       []sim.SavedItemObject
	Effects     []sim.SavedEffectObject
	Spells      []sim.SavedSpellObject
}

func lootOracleFromWorld1172(t *testing.T, f *FrontEnd) []lootOracle1172 {
	t.Helper()
	r := f.live.world.SavedObjects()
	if r == nil {
		t.Fatal("no current registry")
	}
	var out []lootOracle1172
	for i, p := range f.live.mission.party {
		e := releaseEntity(t, f.live, f.live.mission.ids[i])
		a := e.CurrentActorLoad()
		o := lootOracle1172{Human: returnOracleFromWorld1169(p, e), Present: a.Inventory.ContainerPresent, InsertIndex: a.Inventory.InsertIndex, Accumulator: a.Inventory.Accumulator}
		for _, b := range originalCityBindings(f) {
			if b.partyID == p.ID && b.character.Identity != 0 {
				o.OwnerKey = b.character.Identity
			}
		}
		pack, _ := f.live.world.CarriedStacks(e.ID)
		for index, item := range pack {
			row, ok := r.Item(item.ObjectID)
			owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: e.ID}
			if !ok || !sim.StackStateEqual(item, row.Value) || !r.HasLocation(item.ObjectID, sim.SavedItemLocation{Owner: owner, Index: uint32(index)}) {
				t.Fatalf("current owner/stack not represented: actor=%d party=%s index=%d item=%+v ok=%t row=%+v locations=%+v", e.ID, p.ID, index, item, ok, row, r.Locations(item.ObjectID))
			}
			o.Items = append(o.Items, row)
			for _, id := range row.Effects {
				for _, child := range r.Effects {
					if child.ID == id {
						o.Effects = append(o.Effects, child)
					}
				}
			}
			for _, child := range r.Spells {
				if child.ID == row.Spell {
					o.Spells = append(o.Spells, child)
				}
			}
		}
		out = append(out, o)
	}
	return out
}

func lootToken1172(t *testing.T, r sav.DocumentRecordData, token sim.SavedObjectToken) {
	t.Helper()
	for name, value := range map[string]uint32{"T1C": token.T1C, "RuntimeID": token.RuntimeID, "T0C": uint32(token.T0C), "T0E": uint32(token.T0E), "T08": token.T08, "T18": uint32(token.T18)} {
		if got := itemObjectValue1115(t, r, name); got != value {
			t.Fatalf("current Token field lost %s: got=%#x want=%#x", name, got, value)
		}
	}
	b, err := savedObjectRaw(&r, "Block12", 12)
	if err != nil || !reflect.DeepEqual(b, token.Position[:]) {
		t.Fatalf("current Token position bytes lost: got=%x want=%x err=%v", b, token.Position, err)
	}
}

func assertLootRaw1172(t *testing.T, raw []byte, want []lootOracle1172) {
	t.Helper()
	d, err := sav.DecodeDocumentData(raw)
	if err != nil || d.World != nil {
		t.Fatal("current city decode", err)
	}
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := sf.Party()
	if err != nil {
		t.Fatal(err)
	}
	var humans []returnOracle1169
	used := map[uint16]sim.SavedObjectID{}
	keys := map[uint32]uint32{0: 0}
	refsToCheck := map[uint16]uint32{}
	for _, o := range want {
		humans = append(humans, o.Human)
		var actor *sav.DocumentRecordData
		p := rawCharacterNamed(t, party, o.Human.Name)
		for i := range d.Objects {
			if d.Objects[i].Class == "Human" && itemObjectValue1115(t, d.Objects[i], "Identity") == p.Key {
				actor = &d.Objects[i]
			}
		}
		if actor == nil {
			t.Fatal("current Human owner omitted", o.Human.PartyID)
		}
		if o.OwnerKey != 0 {
			keys[o.OwnerKey] = itemObjectValue1115(t, *actor, "Identity")
		}
		present := itemObjectValue1115(t, *actor, "HasInventory") != 0
		refs, _ := savedObjectRefs(actor, "Inventory")
		if present != o.Present || len(refs) != len(o.Items) || present && (itemObjectValue1115(t, *actor, "Inventory1C") != o.InsertIndex || int32(itemObjectValue1115(t, *actor, "Inventory20")) != o.Accumulator) {
			t.Fatal("current pack ownership/count/index/load lost")
		}
		for i, row := range o.Items {
			ref := refs[i]
			if ref == 0 || used[ref] != 0 {
				t.Fatal("two current Items collapsed or missing")
			}
			used[ref], refsToCheck[ref] = row.ID, row.Token.Reference
			r := d.Objects[ref-1]
			class := map[uint8]string{0: "Item", sim.SourceWeapon: "Weapon", sim.SourceArmor: "Armor", sim.SourceShield: "Shield"}[row.Value.SourceEquipment.Class]
			if r.Class != class {
				t.Fatal("current concrete Item class lost", r.Class, class)
			}
			keys[row.Token.Identity] = itemObjectValue1115(t, r, "Identity")
			lootToken1172(t, r, row.Token)
			for name, value := range map[string]uint32{"F40": uint32(row.Value.Code), "F42": row.Value.Count, "F44": uint32(row.Value.Kind), "F45": uint32(row.F45), "F46": uint32(row.F46), "F47": uint32(row.F47), "F48": uint32(row.F48), "F4A": uint32(uint16(row.Value.Weight))} {
				if itemObjectValue1115(t, r, name) != value {
					t.Fatal("current Item field lost", name)
				}
			}
			children, _ := savedObjectRefs(&r, "Effects")
			if len(children) != len(row.Effects) {
				t.Fatal("current Effect edges omitted")
			}
			for j, childRef := range children {
				if childRef == 0 || used[childRef] != 0 {
					t.Fatal("distinct Effect child collapsed")
				}
				used[childRef] = row.Effects[j]
				child := d.Objects[childRef-1]
				var expected *sim.SavedEffectObject
				for k := range o.Effects {
					if o.Effects[k].ID == row.Effects[j] {
						expected = &o.Effects[k]
					}
				}
				if expected == nil {
					t.Fatal("oracle lacks Effect")
				}
				keys[expected.Token.Identity], refsToCheck[childRef] = itemObjectValue1115(t, child, "Identity"), expected.Token.Reference
				lootToken1172(t, child, expected.Token)
				for name, value := range map[string]uint32{"E3C": uint32(expected.Value.Kind), "E3D": uint32(expected.Value.Mode), "E40": expected.Value.Operand, "E0C": uint32(expected.E0C)} {
					if itemObjectValue1115(t, child, name) != value {
						t.Fatal("current Effect operand lost", name)
					}
				}
			}
			s := row.Value.SourceEquipment
			var blocks map[string][]byte
			switch s.Class {
			case sim.SourceWeapon:
				blocks = map[string][]byte{"W52": s.Attack[:], "W6A": s.Defence[:]}
				if itemObjectValue1115(t, r, "W50") != uint32(s.OwnKind) {
					t.Fatal("weapon OwnKind lost")
				}
			case sim.SourceArmor:
				blocks = map[string][]byte{"A52": s.Defence[:]}
				if itemObjectValue1115(t, r, "A50") != uint32(s.OwnKind) {
					t.Fatal("armor OwnKind lost")
				}
			case sim.SourceShield:
				blocks = map[string][]byte{"S50": s.Defence[:]}
			}
			for name, value := range blocks {
				got, err := savedObjectRaw(&r, name, len(value))
				if err != nil || !reflect.DeepEqual(got, value) {
					t.Fatal("equipment pack operands lost", name)
				}
			}
			spellRefs, _ := savedObjectRefs(&r, "WeaponSpell")
			if row.Spell != 0 {
				if len(spellRefs) != 1 || spellRefs[0] == 0 || used[spellRefs[0]] != 0 {
					t.Fatal("owned Spell edge lost")
				}
				used[spellRefs[0]] = row.Spell
				child := d.Objects[spellRefs[0]-1]
				var expected *sim.SavedSpellObject
				for k := range o.Spells {
					if o.Spells[k].ID == row.Spell {
						expected = &o.Spells[k]
					}
				}
				if expected == nil {
					t.Fatal("oracle lacks Spell")
				}
				keys[expected.This] = itemObjectValue1115(t, child, "This")
				for name, value := range map[string]uint32{"S08": uint32(expected.Value.ID), "S09": uint32(expected.Value.Range), "S0A": uint32(expected.Value.Defensive), "S0C": uint32(expected.Value.ManaCost)} {
					if itemObjectValue1115(t, child, name) != value {
						t.Fatal("owned Spell fields lost")
					}
				}
			} else if len(spellRefs) == 1 && spellRefs[0] != 0 {
				t.Fatal("unexpected Spell edge")
			}
		}
	}
	// Numeric keys have an explicit relation to native identity; no gameplay
	// field is zeroed or normalized for comparison.
	seenKeys := map[uint32]bool{}
	for old, key := range keys {
		if old != 0 && (key == 0 || seenKeys[key]) {
			t.Fatal("identity relation is not bijective")
		}
		seenKeys[key] = true
	}
	for ref, old := range refsToCheck {
		got := itemObjectValue1115(t, d.Objects[ref-1], "Reference")
		key, ok := keys[old]
		if !ok || got != key {
			t.Fatal("item/child reference relation lost", old, got)
		}
	}
	assertReturnRaw1169(t, raw, humans)
}

func reloadMissionSAV1172(t *testing.T, f *FrontEnd, app *ui.App) *FrontEnd {
	t.Helper()
	before := f.live.world.Hash()
	store, name, written := menuSAVE(t, f, app, OriginalStore{})
	g := loadLocalLegacySave(t, store, name)
	if before != g.live.world.Hash() || g.live.world.SavedObjects() == nil {
		currentMenuWorldDiagnostics(t, f.live.world, g.live.world)
		t.Fatalf("mission SAV lost current World/graph: World=%016x -> %016x registry=%t", before, g.live.world.Hash(), g.live.world.SavedObjects() != nil)
	}
	if lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
		for i := range doc.Objects {
			if hp, err := savedStructureValue(&doc.Objects[i], "Health"); err == nil && doc.Objects[i].Class == "Unit" && int16(hp) > 1 {
				return savedStructureSetValue(&doc.Objects[i], "Health", uint32(uint16(int16(hp)-1))) == nil
			}
		}
		return false
	}); lost.live.world.Hash() == before {
		t.Fatal("loss control: an altered mission SAV loads the same World")
	}
	return g
}

func TestReleaseImportedLoot1172VictoryColdNextTransfer(t *testing.T) {
	f, private := townReturnImported1168(t)
	app := f.App("1172 real retained loot")
	if err := app.OpenMission(f.MissionOpenerWith(30, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	hero, _ := returnIDs1169(t, f)
	w := f.live.world
	if w.SavedObjects() == nil {
		t.Fatal("city source registry missing")
	}
	sack := groundAt(w.Sacks(), 32, 31)
	if sack == nil || len(sack.ItemInstances) != 1 || sack.ItemInstances[0].Code != 63109 || sack.ItemInstances[0].Price != 500 {
		t.Fatal("installed nonquest M30 loot fixture changed")
	}
	if err := w.HeadlessPlace(hero, 32, 31); err != nil {
		t.Fatal(err)
	}
	liveTakeAt(t, f.live, hero, 32, 31)
	pack, _ := w.CarriedStacks(hero)
	acquired := false
	for _, item := range pack {
		if item.Code == 63109 && item.ObjectID != 0 && item.Count == 1 && item.SourceEquipment.Class == sim.SourceArmor {
			row, _ := w.SavedObjects().Item(item.ObjectID)
			if row.Token.T08 != 0 {
				t.Fatal("actual pickup did not publish the stamped Item flag")
			}
			acquired = true
		}
	}
	if !acquired {
		t.Fatal("pickup did not construct retained armor in PACK", pack)
	}
	for tick := 0; tick < 32; tick++ {
		sim.Step(w, nil)
	}
	f = reloadMissionSAV1172(t, f, app)
	w = f.live.world
	if err := w.HeadlessPlace(hero, 64, 15); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 96 && w.Outcome() == sim.OutcomeUndecided; tick++ {
		sim.Step(w, nil)
	}
	if w.Outcome() != sim.OutcomeWon {
		t.Fatal("real installed mission30 victory did not fire")
	}
	if next, line := f.FinishMissionWithRoster(30, f.live.mission.party, w, f.live.mission.ids, f.live.mission.state.Start.Roster); next < 0 {
		t.Fatal(line)
	}
	want := lootOracleFromWorld1172(t, f)
	for _, o := range want {
		for _, row := range o.Items {
			if row.Value.Code == 0x0e1e {
				t.Fatal("victory did not consume quest object")
			}
		}
	}
	store := SaveStore{Dir: t.TempDir()}
	name, raw := townReturnSave1168(t, f, store.Dir)
	assertLootRaw1172(t, raw, want)
	if err := os.Remove(private); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, "expected.json")
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestImportedLoot1172ColdProcess$", "-test.v")
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(entry), "AGAINROM_SAVE_CORPUS=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "AGAINROM_1172_SAVE="+filepath.Join(store.Dir, name), "AGAINROM_1172_EXPECT="+path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cold only-new-SAV process: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
	t.Log("actual installed M30 armor63109 pickup -> mission SAV full hash -> real quest-consuming victory -> town SAV -> cold LOAD -> mission40 transfer -> second SAV")
}

func TestImportedLoot1172ColdProcess(t *testing.T) {
	f := releaseFront(t)
	path := os.Getenv("AGAINROM_1172_SAVE")
	if path == "" {
		return
	}
	if os.Getenv("AGAINROM_SAVE_CORPUS") != "" {
		t.Fatal("cold child must not receive source corpus input")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(os.Getenv("AGAINROM_1172_EXPECT"))
	if err != nil {
		t.Fatal(err)
	}
	var want []lootOracle1172
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatal(err)
	}
	app := openLocalTownSAV(t, f, filepath.Dir(path), filepath.Base(path))
	assertLootRaw1172(t, raw, want)
	if err := app.OpenMission(f.MissionOpenerWith(40, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	hero, mage := returnIDs1169(t, f)
	w := f.live.world
	if w.SavedObjects() == nil {
		t.Fatal("cold SAV did not seed its exact graph")
	}
	assertLootRaw1172(t, raw, lootOracleFromWorld1172(t, f))
	if err := w.MoveCarried(hero, mage, 63109, 1); err != nil {
		t.Fatal("production next-mission transfer", err)
	}
	for _, id := range []sim.EntityID{hero, mage} {
		pack, _ := w.CarriedStacks(id)
		count := 0
		for _, item := range pack {
			if item.Code == 63109 {
				count += int(item.Count)
				if item.ObjectID == 0 {
					t.Fatal("transfer lost object identity")
				}
			}
		}
		if id == hero && count != 0 || id == mage && count != 1 {
			t.Fatal("next item action used wrong owner", id, count)
		}
	}
	f = reloadMissionSAV1172(t, f, app)
	if next, line := f.FinishMissionWithRoster(40, f.live.mission.party, f.live.world, f.live.mission.ids, f.live.mission.state.Start.Roster); next < 0 {
		t.Fatal(line)
	}
	_, second := townReturnSave1168(t, f, t.TempDir())
	assertLootRaw1172(t, second, lootOracleFromWorld1172(t, f))
	t.Log("fresh process used only new SAV/assets, transferred acquired armor, preserved intervening SAV, and read second SAV ownership/load")
}

func originalCityBindings(f *FrontEnd) []originalCityBinding {
	if f.originalCity == nil {
		return nil
	}
	return f.originalCity.bindings
}
