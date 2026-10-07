package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestTrainedCityProcess(t *testing.T) {
	// Keep the registered environment gate visible to the release census even
	// when this invocation is not one of the parent witness's fresh children.
	f := releaseFront(t)
	mode := os.Getenv("AGAINROM_1099_PROCESS")
	if mode == "" {
		return
	}
	input, err := ReadSaveFile(os.Getenv("AGAINROM_1099_INPUT"))
	if err != nil {
		t.Fatal(err)
	}
	if _, town, err := f.RestoreOriginal(input); err != nil || !town {
		t.Fatalf("fresh SAV %t %v", town, err)
	}
	if mode == "mission" {
		trainedMission1099(t, f)
		return
	}
	slot, _ := strconv.Atoi(os.Getenv("AGAINROM_1099_SLOT"))
	price, _ := strconv.Atoi(os.Getenv("AGAINROM_1099_PRICE"))
	gold, _ := strconv.Atoi(os.Getenv("AGAINROM_1099_GOLD"))
	hero := trainingPartyMember(t, f, "hero")
	if got := memberSchoolPrice(hero, slot); got != price || f.Town.Gold() != gold {
		t.Fatalf("fresh repeated quote price=%d gold=%d", got, f.Town.Gold())
	}
	if msg := train1099(t, f, "hero", slot-1); !strings.HasPrefix(msg, "trained ") {
		t.Fatal(msg)
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.ExportOriginalSave(s, "second training")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteConvertedSave(os.Getenv("AGAINROM_1099_OUTPUT"), out, os.Getenv("AGAINROM_ASSETS")); err != nil {
		t.Fatal(err)
	}
	if f.Town.Gold() != gold-price {
		t.Fatal("fresh training debit")
	}
}

func TestReleaseImportedFighterTrainingUsesSAVAndFreshProcesses(t *testing.T) {
	f := releaseFront(t)
	sourcePath, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("1099-city-training")
	save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: filepath.Dir(sourcePath)}, nil)
	app.SetSaveSeams(save, list, load)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	var label string
	for _, row := range list() {
		if row.Name == filepath.Base(sourcePath) {
			label = row.Label
		}
	}
	if label == "" {
		t.Fatal("source absent from original picker")
	}
	if err := app.HeadlessActivate(label); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatalf("App LOAD screen=%s %v", app.Screen(), err)
	}
	hero := trainingPartyMember(t, f, "hero")
	h, ok := hero.OriginalHumanState()
	if !ok {
		t.Fatalf("source fighter not eligible: mage=%t retained=%+v", hero.Mage, hero.OriginalHuman)
	}
	// Choose an actually untrained school slot in this lawful character;
	// quote its stored maintained base, not an authored test purse.
	slot := 0
	for i := 1; i <= 5; i++ {
		if h.Base.Skill[i] == 0 {
			slot = i
			break
		}
	}
	if slot == 0 {
		t.Fatal("source has no zero-base school slot")
	}
	gold := f.Town.Gold()
	if memberSchoolPrice(hero, slot) != 200 || gold < 420 {
		t.Fatalf("source cannot afford two zero-base trainings: gold=%d", gold)
	}
	s := f.TownScreen().(*townScreen)
	s.room, s.schoolCell = roomSchool, slot-1
	for i, m := range f.Carried {
		if m.ID == "hero" {
			s.shopMember = i
		}
	}
	if err := app.HeadlessActivate(f.Words.SchoolTrain); err != nil {
		t.Fatal("App Train", err)
	}
	if f.Town.Gold() != gold-200 {
		t.Fatal("App Train did not debit200", app.HeadlessMessage())
	}
	trained := trainingPartyMember(t, f, "hero")
	if trained.OriginalHuman.State.Base.Skill[slot] != 1 || trained.Carry.SkillXP[slot] != 101 {
		t.Fatal("full Human training did not run")
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatalf("App SAVE entries=%+v error=%v message=%s", entries, err, app.HeadlessMessage())
	}
	trainedPath := filepath.Join(store.Dir, entries[0].Name)
	raw, err := ReadSaveFile(trainedPath)
	if err != nil || !bytes.HasPrefix(raw, []byte(sav.Magic)) {
		t.Fatal("App SAVE not Asg&", err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	// Check the wire independently of cityHumanUpdate's copied representation.
	for _, c := range p.Roster() {
		if c.Hero {
			human, err := p.Human(c.Identity)
			if err != nil {
				t.Fatal(err)
			}
			if c.SkillLevels[slot] != 1 || c.SkillXP[slot] != 101 || binary.LittleEndian.Uint16(human.Fields.Base[2+2*slot:]) != 1 {
				t.Fatal("wire lost trained skill/base/XP")
			}
		}
	}
	child := func(mode, in, out string) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestTrainedCityProcess$", "-test.v")
		cmd.Env = append(os.Environ(), "AGAINROM_1099_PROCESS="+mode, "AGAINROM_1099_INPUT="+in, "AGAINROM_1099_OUTPUT="+out,
			fmt.Sprintf("AGAINROM_1099_SLOT=%d", slot), "AGAINROM_1099_PRICE=220", fmt.Sprintf("AGAINROM_1099_GOLD=%d", gold-200))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fresh %s %v\n%s", mode, err, out)
		}
	}
	second := filepath.Join(store.Dir, "second.sav")
	child("train", trainedPath, second)
	child("mission", trainedPath, "")
	if bytes.Equal(source, raw) {
		t.Fatal("source replayed")
	}
	if got, err := os.ReadFile(sourcePath); err != nil || !bytes.Equal(got, source) {
		t.Fatal("lawful source changed")
	}
	if reflect.DeepEqual(hero, trained) {
		t.Fatal("no player-state result")
	}
	t.Logf("lawful source=%x; school slot=%d; gold=%d->%d; first/second prices=200/220; trained SAV=%d bytes SHA=%x; App LOAD/Train/SAVE; fresh-process training and mission; mission30/current pools/combat/movement and native continuation", sha256.Sum256(source), slot, gold, gold-200, len(raw), sha256.Sum256(raw))
}

func trainedMission1099(t *testing.T, f *FrontEnd) {
	t.Helper()
	h, ok := trainingPartyMember(t, f, "hero").OriginalHumanState()
	if !ok {
		t.Fatal("missing source fighter")
	}
	f.SetDeterministicFrames(true)
	app := f.App("1099-trained-mission")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(30)); err != nil {
		t.Fatal(err)
	}
	var id sim.EntityID
	for i, p := range f.live.mission.party {
		if p.ID == "hero" {
			id = f.live.mission.ids[i]
		}
	}
	if id == 0 {
		t.Fatal("missing mission party identity")
	}
	e, _ := f.live.entity(id)
	t.Logf("mission entry retained=%t load=%d", e.HumanMovement.Present, e.Load)
	assertTrainedEntity1099(t, e, h)
	if e.HP != int32(int16(h.Health)) || e.Mana != int32(int16(h.Mana)) {
		t.Fatal("mission refilled stored pools")
	}
	for n := 0; n < 4; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
	}
	e, _ = f.live.entity(id)
	assertTrainedEntity1099(t, e, h)
	snap, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	back := releaseFront(t)
	back.SetDeterministicFrames(true)
	opener, town, err := back.Restore(decoded)
	if err != nil || town {
		t.Fatalf("native mission restore %v %v", town, err)
	}
	app2 := back.App("1099-trained-native")
	app2.Layout(1024, 768)
	if err := app2.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	form, err := back.live.world.MarshalBinary()
	if err != nil || !bytes.Equal(form, snap.World) {
		t.Fatal("native opening changed saved world")
	}
	for n := 0; n < 8; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
		if err := consumableStep1090(app2); err != nil {
			t.Fatal(err)
		}
		if f.live.world.Hash() != back.live.world.Hash() {
			t.Fatalf("native continuation diverged at %d", n)
		}
	}
}

func assertTrainedEntity1099(t *testing.T, e sim.Entity, h data.HumanState) {
	t.Helper()
	if e.RotationSpeed != int32(h.MoverSpeed) {
		t.Fatalf("retained Human mover byte=%d, live turn rate=%d", h.MoverSpeed, e.RotationSpeed)
	}
	if e.MaxHP != int32(int16(h.HealthMax)) || e.MaxMana != int32(int16(h.ManaMax)) ||
		e.ToHit != int32(int16(h.Attack.ToHit)) || e.DamageBase != int32(h.Attack.DamageBase) || e.DamageSpread != int32(h.Attack.DamageSpread) ||
		e.SecondBase != h.Attack.SecondBase || e.SecondSpread != h.Attack.SecondSpread ||
		e.Defence != int32(int16(h.Defence.Defence)) || e.Absorption != int32(int16(h.Defence.Absorption)) || e.XPSlot != h.Attack.Active ||
		e.Capacity != int32(int16(h.Capacity)) || e.Load != int32(int16(h.Load)) || !e.HumanMovement.Present || e.HumanMovement.RawSpeed != int16(h.Speed) {
		t.Fatalf("mission did not retain coupled source values: entity=%+v source=%+v", e, h)
	}
	for i := range e.Skill {
		if e.Skill[i] != int32(int16(h.Attack.Skill[i])) || e.SkillXP[i] != int32(h.SkillXP[i]) {
			t.Fatal("mission skills/XP changed")
		}
	}
	for i := range e.Protection {
		if e.Protection[i] != int32(int16(h.Defence.Protection[i+1])) || e.Resistance[i] != h.Defence.Resistance[i+1] {
			t.Fatal("mission defence families changed")
		}
	}
}
