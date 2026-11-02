package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The fixture's Unit store arm starts 509 bytes after an actor's class tag
// when the actor has no profile; U4C, whose bit 4 clears the fighter flag, is
// its fourth byte.
const equipmentRecordMageFlagAt = 509 + 3

// equipmentRecordFixtures are synthetic holdings: a fighter Human with a held
// weapon, a held shield and every worn armour slot; a mage Human; a Unit with
// both held slots; an empty Human; and a Humanoid in a second Player. overlap
// adds a Human whose Worn position 2 holds the shield slot's item.
func equipmentRecordFixtures(overlap bool) []byte {
	item := func(class string, code uint16) *holdingFixtureItem {
		kind := uint8(1)
		if class == "Weapon" {
			kind = 2
		}
		return &holdingFixtureItem{class: class, code: code, count: 1, kind: kind, price: 10}
	}
	fighter := &holdingFixture{weapon: item("Weapon", 0x0101), shield: item("Shield", 0x0201)}
	for at := 2; at < 12; at++ {
		fighter.worn[at] = item("Armor", uint16(at+1)<<8|1)
	}
	mage := &holdingFixture{weapon: item("Weapon", 0x0102), items: []*holdingFixtureItem{item("Armor", 0x0402)}}
	mage.worn[2], mage.worn[11] = item("Armor", 0x0302), item("Armor", 0x0c02)
	humanoid := &holdingFixture{humanoid: true}
	humanoid.worn[5] = item("Armor", 0x0603)
	actors := []*poolFixtureActor{
		{mapID: 91, cell: 0x0605, hp: 20, maxHP: 20, human: true, holdings: fighter},
		{mapID: 92, cell: 0x0606, hp: 20, maxHP: 20, human: true, holdings: mage},
		{mapID: 93, cell: 0x0607, hp: 20, maxHP: 20, holdings: &holdingFixture{weapon: item("Weapon", 0x0103), shield: item("Shield", 0x0203)}},
		{mapID: 94, cell: 0x0608, hp: 20, maxHP: 20, human: true},
	}
	if overlap {
		held := &holdingFixture{}
		held.worn[1] = item("Armor", 0x0204)
		actors = append(actors, &poolFixtureActor{mapID: 96, cell: 0x060a, hp: 20, maxHP: 20, human: true, holdings: held})
	}
	second := &poolFixtureActor{mapID: 95, cell: 0x0609, hp: 20, maxHP: 20, holdings: humanoid}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{actors}}, {groups: [][]*poolFixtureActor{{second}}}}, nil)
	body[actors[1].off+equipmentRecordMageFlagAt] |= 4
	return savedContainer(body)
}

func equipmentRecordDead() []byte {
	weapon := &holdingFixtureItem{class: "Weapon", code: 0x0105, count: 1, kind: 2, price: 10}
	armour := &holdingFixtureItem{class: "Armor", code: 0x0405, count: 1, kind: 1, price: 10}
	shield := &holdingFixtureItem{class: "Shield", code: 0x0205, count: 1, kind: 1, price: 10}
	worn := &holdingFixture{weapon: &holdingFixtureItem{class: "Weapon", code: 0x0106, count: 1, kind: 2, price: 10}, shield: shield}
	worn.worn[3] = armour
	dead := []*poolFixtureActor{
		{cell: 0x0605, hp: 0xd8ef, stage: 5, holdings: &holdingFixture{weapon: weapon}},
		{cell: 0x0606, hp: 0xd8ef, stage: 5, human: true, holdings: worn},
		{cell: 0x0607, hp: 0xd8ef, stage: 5},
	}
	return savedContainer(poolFixtureBody([]*poolFixturePlayer{{}}, dead))
}

func equipmentRecordMission(t *testing.T) *Mission {
	t.Helper()
	var entities []sim.Entity
	var stocks []sim.Stock
	for i, id := range []uint16{91, 92, 93, 94, 95, 96} {
		entities = append(entities, sim.Entity{ID: sim.EntityID(i + 1), MapUnitID: id, X: 5 + int32(i), Y: 6, HP: 20, MaxHP: 20})
		stocks = append(stocks, sim.Stock{ID: sim.EntityID(i + 1)})
	}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, sim.Terrain{}, entities, nil, sim.Relations{}, nil, stocks)
	if err != nil {
		t.Fatal(err)
	}
	return &Mission{Map: &alm.Map{Width: 40, Height: 40}, World: w}
}

func equipmentRecordPiece(p *sav.Piece) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%s:%#04x", p.Class, p.Code)
}

// TestSavEquipmentReadersRecordHeldAndWornRoutes records what each reader of
// a saved actor's held and worn items yields for synthetic holdings: the
// holdings projection, the original-save import's restored stock and its
// refusals, the party reader's merged equipment and the dead reader's held
// slots.
func TestSavEquipmentReadersRecordHeldAndWornRoutes(t *testing.T) {
	var got strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&got, format+"\n", args...) }
	f, err := sav.Open(equipmentRecordFixtures(false))
	if err != nil {
		t.Fatal(err)
	}
	holdings, err := f.ActorHoldings()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range holdings {
		var worn []string
		for at, p := range a.Worn {
			if p != nil {
				worn = append(worn, fmt.Sprintf("%d=%s", at, equipmentRecordPiece(p)))
			}
		}
		line("holdings %d weapon=%s shield=%s worn=%v items=%d", a.MapUnitID, equipmentRecordPiece(a.HeldWeapon), equipmentRecordPiece(a.HeldShield), worn, len(a.Items))
	}
	ms := equipmentRecordMission(t)
	var report OriginalSaveResume
	line("import err=%v stocked=%d", restoreOriginalActorStock(ms, holdings, nil, &report), report.Stocked)
	for _, e := range ms.World.Entities() {
		equipped, _ := ms.World.Equipped(e.ID)
		stacks, _ := ms.World.CarriedStacks(e.ID)
		var carried []string
		for _, s := range stacks {
			carried = append(carried, fmt.Sprintf("%#04x*%d", s.Code, s.Count))
		}
		line("stock %d equipped=%#04x carried=%v", e.MapUnitID, equipped, carried)
	}
	overlapped, err := sav.Open(equipmentRecordFixtures(true))
	if err != nil {
		t.Fatal(err)
	}
	overlapHoldings, err := overlapped.ActorHoldings()
	if err != nil {
		t.Fatal(err)
	}
	ms = equipmentRecordMission(t)
	before := ms.World.Hash()
	report = OriginalSaveResume{}
	err = restoreOriginalActorStock(ms, overlapHoldings, nil, &report)
	line("overlap err=%v unchanged=%t stocked=%d", err, ms.World.Hash() == before, report.Stocked)
	refusals := []struct {
		name string
		edit func(*sav.ActorHoldings)
	}{
		{"worn position 1", func(a *sav.ActorHoldings) { a.Worn[0] = &sav.Piece{Class: "Armor", Code: 0x0101, Stack: 1} }},
		{"worn position 2", func(a *sav.ActorHoldings) { a.Worn[1] = &sav.Piece{Class: "Armor", Code: 0x0201, Stack: 1} }},
		{"held weapon class", func(a *sav.ActorHoldings) { a.HeldWeapon = &sav.Piece{Class: "Shield", Code: 0x0101, Stack: 1} }},
		{"held shield class", func(a *sav.ActorHoldings) { a.HeldShield = &sav.Piece{Class: "Weapon", Code: 0x0201, Stack: 1} }},
		{"worn class", func(a *sav.ActorHoldings) { a.Worn[4] = &sav.Piece{Class: "Weapon", Code: 0x0501, Stack: 1} }},
		{"worn code slot", func(a *sav.ActorHoldings) { a.Worn[3] = &sav.Piece{Class: "Armor", Code: 0x0501, Stack: 1} }},
		{"held code slot", func(a *sav.ActorHoldings) { a.HeldShield = &sav.Piece{Class: "Shield", Code: 0x0101, Stack: 1} }},
	}
	for _, refusal := range refusals {
		a := sav.ActorHoldings{Off: 7, MapUnitID: 91, Cell: 0x0605, HP: 20}
		refusal.edit(&a)
		ms = equipmentRecordMission(t)
		before = ms.World.Hash()
		report = OriginalSaveResume{}
		err := restoreOriginalActorStock(ms, []sav.ActorHoldings{a}, nil, &report)
		line("refusal %s err=%v unchanged=%t", refusal.name, err, ms.World.Hash() == before)
	}
	party, err := overlapped.Party()
	line("party err=%v", err)
	for _, c := range party {
		var worn []string
		for i := range c.Worn {
			worn = append(worn, equipmentRecordPiece(&c.Worn[i]))
		}
		fighter := c.Basis != nil && c.Basis.Human.Fighter
		line("party %d fighter=%t worn=%v items=%d", c.MapUnitID, fighter, worn, len(c.Items))
	}
	dead, err := sav.Open(equipmentRecordDead())
	if err != nil {
		t.Fatal(err)
	}
	actors, err := dead.DeadActors()
	line("dead err=%v", err)
	for _, d := range actors {
		line("dead %s worn=%d weapon=%t weapon-index=%d carried=%d", d.Class, d.Worn, d.HeldWeapon.Present, d.HeldWeapon.ArchiveIndex, d.Carried)
	}
	if got.String() != equipmentReadersRecord {
		t.Fatalf("equipment readers record changed:\n%s", got.String())
	}
}

const equipmentReadersRecord = `holdings 91 weapon=Weapon:0x0101 shield=Shield:0x0201 worn=[2=Armor:0x0301 3=Armor:0x0401 4=Armor:0x0501 5=Armor:0x0601 6=Armor:0x0701 7=Armor:0x0801 8=Armor:0x0901 9=Armor:0x0a01 10=Armor:0x0b01 11=Armor:0x0c01] items=0
holdings 92 weapon=Weapon:0x0102 shield=- worn=[2=Armor:0x0302 11=Armor:0x0c02] items=1
holdings 93 weapon=Weapon:0x0103 shield=Shield:0x0203 worn=[] items=0
holdings 94 weapon=- shield=- worn=[] items=0
holdings 95 weapon=- shield=- worn=[5=Armor:0x0603] items=0
import err=<nil> stocked=5
stock 91 equipped=[0x0101 0x0201 0x0301 0x0401 0x0501 0x0601 0x0701 0x0801 0x0901 0x0a01 0x0b01 0x0c01] carried=[]
stock 92 equipped=[0x0102 0x0000 0x0302 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0c02] carried=[0x0402*1]
stock 93 equipped=[0x0103 0x0203 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000] carried=[]
stock 94 equipped=[0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000] carried=[]
stock 95 equipped=[0x0000 0x0000 0x0000 0x0000 0x0000 0x0603 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000] carried=[]
stock 96 equipped=[0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000] carried=[]
overlap err=original holdings actor 4368: armor slot 2 overlaps native held role unchanged=true stocked=0
refusal worn position 1 err=original holdings actor 7: armor slot 1 overlaps native held role unchanged=true
refusal worn position 2 err=original holdings actor 7: armor slot 2 overlaps native held role unchanged=true
refusal held weapon class err=original holdings actor 7: wrong held item class Shield unchanged=true
refusal held shield class err=original holdings actor 7: wrong held item class Weapon unchanged=true
refusal worn class err=original holdings actor 7: wrong armor class Weapon unchanged=true
refusal worn code slot err=original holdings actor 7: unrepresentable equipment slot 4 code 0x501 count 1 unchanged=true
refusal held code slot err=original holdings actor 7: unrepresentable equipment slot 2 code 0x101 count 1 unchanged=true
party err=<nil>
party 91 fighter=true worn=[Weapon:0x0101 Shield:0x0201 Armor:0x0301 Armor:0x0401 Armor:0x0501 Armor:0x0601 Armor:0x0701 Armor:0x0801 Armor:0x0901 Armor:0x0a01 Armor:0x0b01 Armor:0x0c01] items=0
party 92 fighter=false worn=[Weapon:0x0102 Armor:0x0302 Armor:0x0c02] items=1
party 93 fighter=true worn=[Weapon:0x0103 Shield:0x0203] items=0
party 94 fighter=true worn=[] items=0
party 96 fighter=true worn=[Armor:0x0204] items=0
dead err=<nil>
dead Unit worn=1 weapon=true weapon-index=6 carried=0
dead Human worn=3 weapon=true weapon-index=9 carried=0
dead Unit worn=0 weapon=false weapon-index=0 carried=0
`
