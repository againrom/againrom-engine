package game

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseNativeTrainingSurvivesBoundedBonuses(t *testing.T) {
	for _, tc := range []struct {
		name  string
		base  int32
		bonus int16
	}{{"upper bound", 100, 200}, {"zero floor", 40, -100}} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := bonusSwordParty(f, tc.bonus)
			party[0].Hero.Skill[1] = tc.base
			app := f.App("native training")
			if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
				t.Fatal(err)
			}
			got := mapload.CarryParty(party, f.live.world, f.live.mission.ids)
			if got[0].Hero.Skill[1] != tc.base {
				t.Fatalf("carried trained Sword %d, want %d", got[0].Hero.Skill[1], tc.base)
			}
		})
	}
}

func trainingActor(t *testing.T, raw []byte, member int) (*sav.DocumentData, *sav.DocumentRecordData) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || member >= len(a.Party)+len(a.Roster) {
		t.Fatal("training party binding", err)
	}
	rows := append(append([]currentPartyMember(nil), a.Party...), a.Roster...)
	for _, b := range a.Bindings {
		if b.ID == rows[member].Entity && !b.Structure && !b.Missing {
			return &doc, &doc.Objects[b.Object-1]
		}
	}
	t.Fatal("training actor is absent")
	return nil, nil
}

func trainingSave(t *testing.T, f *FrontEnd, member int, base int32) string {
	t.Helper()
	path := saveCorpseMission(t, f, t.TempDir())
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, actor := trainingActor(t, raw, member)
	if b := modBlock(actor, "U114"); len(b) != 24 || binary.LittleEndian.Uint16(b[4:]) != uint16(base) {
		t.Fatalf("ordinary trained Sword base %x, want %d", b, base)
	}
	if l := modBlock(actor, "UA6"); len(l) != 24 || int16(binary.LittleEndian.Uint16(l[4:])) > 100 {
		t.Fatal("ordinary effective level exceeds 100")
	}
	return path
}

func trainingCheck(t *testing.T, f *FrontEnd, id sim.EntityID, base, effective int32) {
	t.Helper()
	e := releaseEntity(t, f.live, id)
	if !e.NativeTraining.Present || e.NativeTraining.Levels[1] != base || e.Skill[1] != effective {
		t.Fatalf("actor %d trained/effective = %+v/%d, want %d/%d", id, e.NativeTraining, e.Skill[1], base, effective)
	}
}

func trainingEquip(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID, equip bool) {
	t.Helper()
	e := releaseEntity(t, f.live, id)
	f.live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	if equip {
		stacks, _ := f.live.world.CarriedStacks(id)
		i := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == 0xec7e })
		if i < 0 {
			t.Fatal("training item is absent from pack")
		}
		f.live.enqueueEquip(i)
	} else {
		slot, ok := EquipTarget(data.ItemCode(0xec7e), f.Table)
		if !ok {
			t.Fatal("training armor has no installed slot")
		}
		f.live.enqueueUnequip(slot - 1)
	}
	for n := 0; n < 4; n++ {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
}

func trainingStrike(t *testing.T, f *FrontEnd, id sim.EntityID, raises bool) int32 {
	t.Helper()
	hero := releaseEntity(t, f.live, id)
	var victim sim.Entity
	distance := int32(1 << 30)
	for _, e := range f.live.world.Entities() {
		dx, dy := e.X-hero.X, e.Y-hero.Y
		if e.HP > 0 && e.Domain == hero.Domain && f.live.world.Relations().Hostile(hero.Owner, e.Owner) && e.XPValue > 0 && dx*dx+dy*dy < distance {
			distance, victim = dx*dx+dy*dy, e
		}
	}
	if victim.MaxHP == 0 || hero.XPSlot != 1 {
		t.Fatal("training strike fixture lacks Sword and hostile XP target", hero.XPSlot)
	}
	if err := f.live.world.HeadlessPlace(id, victim.X-1, victim.Y); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2400; n++ {
		f.live.strike(uint32(id), uint32(victim.ID))
		f.live.tick()
		e := releaseEntity(t, f.live, id)
		if e.HP <= 0 {
			t.Fatal("training witness actor fell")
		}
		if raises && e.NativeTraining.Levels[1] > hero.NativeTraining.Levels[1] || !raises && releaseEntity(t, f.live, victim.ID).HP < victim.HP {
			if !raises && e.SkillXP != hero.SkillXP {
				t.Fatal("capped trained actor gained XP")
			}
			if raises && (e.SkillXP[1] <= hero.SkillXP[1] || e.Skill[1] != hero.Skill[1]) {
				t.Fatal("concealed award did not change base and XP alone")
			}
			return e.NativeTraining.Levels[1]
		}
	}
	now := releaseEntity(t, f.live, id)
	t.Fatalf("strike: XP %v -> %v, base %v -> %v, victim HP %d -> %d", hero.SkillXP, now.SkillXP, hero.NativeTraining, now.NativeTraining, victim.HP, releaseEntity(t, f.live, victim.ID).HP)
	return 0
}

func TestReleaseNativeTrainingAwardEquipSaveAndTown(t *testing.T) {
	for _, tc := range []struct {
		name  string
		base  int32
		bonus int16
	}{{"cap", 100, 200}, {"hidden upper award", 90, 200}, {"hidden floor award", 40, -100}} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := bonusSwordParty(f, tc.bonus)
			blade := f.ChargenParty(ui.ChargenResult{Name: "training", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
			slot, ok := EquipTarget(data.ItemCode(0xec7e), f.Table)
			if !ok {
				t.Fatal("training armor has no installed slot")
			}
			blade[0].WornItems[slot-1], blade[0].Worn[slot-1] = party[0].WornItems[11], party[0].Worn[11]
			party = blade
			party[0].Carry = &mapload.Carry{Equipped: party[0].Worn, EquippedItems: party[0].WornItems}
			party[0].Hero.Skill[1], party[0].Hero.Mind, party[0].Hero.Body = tc.base, 400, 300
			party[0].Carry.SkillXP[1] = (sim.Rules{}).SkillXP(tc.base)
			app := f.App("trained skill lifecycle")
			app.Layout(1024, 768)
			if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
				t.Fatal(err)
			}
			id := shieldWitnessSelect(t, f, app)
			effective := f.live.world.Rules().EffectiveSkill(tc.base, int32(tc.bonus))
			trainingCheck(t, f, id, tc.base, effective)
			base := trainingStrike(t, f, id, tc.base < 100)
			trainingCheck(t, f, id, base, effective)
			if f.live.derivedSkills[id].training.Levels[1] != base {
				t.Fatal("hidden award did not invalidate the derived cache")
			}
			trainingEquip(t, f, app, id, false)
			trainingCheck(t, f, id, base, base)
			trainingEquip(t, f, app, id, true)
			trainingCheck(t, f, id, base, effective)
			path := trainingSave(t, f, 0, base)
			out := filepath.Join(t.TempDir(), "second.sav")
			cmd := exec.Command(os.Args[0], "-test.run=^TestNativeTrainingColdProcess$", "-test.v")
			cmd.Env = append(os.Environ(), "AGAINROM_TRAINING_INPUT="+path, "AGAINROM_TRAINING_OUTPUT="+out, "AGAINROM_TRAINING_BASE="+strconv.Itoa(int(base)))
			if log, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("cold process: %v\n%s", err, log)
			} else {
				t.Log(string(log))
			}
			cold := loadAreaContinuation(t, out)
			trainingCheck(t, cold, cold.live.mission.ids[0], base, base)
			if _, _, err := cold.LiveCompleteCampaign(); err != nil {
				t.Fatal("mission return", err)
			}
			if cold.Carried[0].Hero.Skill[1] != base {
				t.Fatal("town carry changed trained base")
			}
			s := currentTownShop(cold, cold.Carried[0].ID)
			action := s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, 0xec7e)}, ui.ShopControl{Kind: ui.ShopControlDoll})
			worn := mapload.MemberItemEquipment(cold.Carried[0], cold.Table)
			if worn[slot-1].Code != 0xec7e || cold.Carried[0].Hero.Skill[1] != base {
				t.Fatal("town equip changed trained base or failed", action, worn[slot-1].Code, cold.Carried[0].Hero.Skill[1])
			}
			g := currentTownReload(t, currentTownSave(t, cold))
			if g.Carried[0].Hero.Skill[1] != base {
				t.Fatal("town cold SAVE/LOAD changed trained base")
			}
			if err := g.App("next mission training").OpenMission(g.MissionOpenerWith(40, g.Carried)); err != nil {
				t.Fatal(err)
			}
			trainingCheck(t, g, g.live.mission.ids[0], base, effective)
			trainingSave(t, g, 0, base)
			t.Logf("base %d -> %d, effective %d; strike, unequip/equip, cold process, second SAVE, town equip/SAVE/LOAD, next mission", tc.base, base, effective)
		})
	}
}

func TestNativeTrainingColdProcess(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("set AGAINROM_ASSETS to a lawful install")
	}
	path := os.Getenv("AGAINROM_TRAINING_INPUT")
	if path == "" {
		t.Skip("cold-process child")
	}
	base, err := strconv.Atoi(os.Getenv("AGAINROM_TRAINING_BASE"))
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("cold mission preparation", err)
	}
	app := f.App("cold training next action")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	id := shieldWitnessSelect(t, f, app)
	trainingEquip(t, f, app, id, false)
	trainingCheck(t, f, id, int32(base), int32(base))
	second := trainingSave(t, f, 0, int32(base))
	raw, err = os.ReadFile(second)
	if err == nil {
		err = os.WriteFile(os.Getenv("AGAINROM_TRAINING_OUTPUT"), raw, 0644)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestReleaseNativeTrainingOrdinaryBaseOverridesLift(t *testing.T) {
	f := releaseFront(t)
	party := bonusSwordParty(f, 200)
	party[0].Hero.Skill[1] = 70000
	if err := f.App("wide training").OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	path := trainingSave(t, f, 0, 70000)
	trainingCheck(t, loadAreaContinuation(t, path), f.live.mission.ids[0], 70000, 255)
	raw, _ := os.ReadFile(path)
	doc, actor := trainingActor(t, raw, 0)
	a, _ := readCurrentActions(doc)
	if len(a.Party[0].Base.Lifts) == 0 {
		t.Fatal("wide training fixture has no anchored lift")
	}
	binary.LittleEndian.PutUint16(modBlock(actor, "U114")[4:], 70)
	leaf, _ := json.Marshal(a)
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(*doc)
	if err != nil {
		t.Fatal(err)
	}
	edited := filepath.Join(t.TempDir(), "edited.sav")
	if err := os.WriteFile(edited, raw, 0644); err != nil {
		t.Fatal(err)
	}
	g := loadAreaContinuation(t, edited)
	trainingCheck(t, g, g.live.mission.ids[0], 70, 255)
	trainingSave(t, g, 0, 70)
}

func TestReleaseNativeTrainingJoinedRoster(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("joined training").OpenMission(f.MissionOpener(40)); err != nil {
		t.Fatal(err)
	}
	id := releaseRosterNPC(t, f.live.mission.state, 25)
	e := releaseEntity(t, f.live, id)
	base := e.NativeTraining.Levels
	base[1] = 90
	if !f.live.world.SetNativeTraining(id, sim.NativeTraining{Present: true, Levels: base}) {
		t.Fatal("native roster base")
	}
	stocks := f.live.world.Stock()
	stock := stocks[slices.IndexFunc(stocks, func(s sim.Stock) bool { return s.ID == id })]
	item := mapload.SourceConstructedItem(sim.PlainItem(0xec7e), f.Table)
	item.Effects = []sim.ItemEffect{{Kind: 27, Operand: 200}}
	stock.EquippedItems[11], stock.Equipped[11] = item, item.Code
	if !f.live.world.ReplaceStock(stock) {
		t.Fatal("roster fixture equipment")
	}
	f.live.recomputeRaisedSkills()
	trainingCheck(t, f, id, 90, 255)
	controlled, err := sim.NewControlledScriptWorld(f.live.world, joinedLiveScript(t, id))
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = controlled, controlled
	for n := 0; n < 32 && len(f.live.mission.ids) < 2; n++ {
		f.live.tick()
	}
	if len(f.live.mission.ids) != 2 || f.live.mission.party[1].Hero.Skill[1] != 90 {
		t.Fatal("production join carry lost native base")
	}
	carried, _ := mapload.CarryRosterIDs(nil, f.live.world, nil, f.live.mission.state.Start.Roster)
	if len(carried) != 1 || carried[0].Hero.Skill[1] != 90 {
		t.Fatal("roster ID carry lost native base")
	}
	path := trainingSave(t, f, 1, 90)
	g := loadAreaContinuation(t, path)
	trainingCheck(t, g, id, 90, 255)
	if _, _, err := g.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(g.Carried, func(p mapload.PartyMember) bool { return p.CompanionNPC == 25 })
	if i < 0 || g.Carried[i].Hero.Skill[1] != 90 {
		t.Fatal("joined town carry lost trained base")
	}
	if err := g.App("joined next mission").OpenMission(g.MissionOpenerWith(70, g.Carried)); err != nil {
		t.Fatal(err)
	}
	trainingCheck(t, g, g.live.mission.ids[i], 90, 255)
}
