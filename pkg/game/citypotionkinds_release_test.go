package game

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type cityPotionKind struct {
	name   string
	member int
	timed  bool
	mana   bool
}

var cityPotionKinds = []cityPotionKind{
	{name: "Potion Health Regeneration", member: 0, timed: true},
	{name: "Potion Medium Healing", member: 0},
	{name: "Potion Big Healing", member: 0},
	{name: "Potion Mana Regeneration", member: 1, timed: true, mana: true},
	{name: "Potion Medium Mana", member: 1, mana: true},
	{name: "Potion Big Mana", member: 1, mana: true},
}

const (
	cityPotionWoundHealth = 20
	cityPotionWoundMana   = 10
)

func cityPotionWound(t *testing.T, m *mapload.PartyMember, mana bool) {
	t.Helper()
	if m.Carry == nil || m.Carry.LiveLoad == nil || m.Carry.LiveLoad.Inventory.Source.Class != 2 || m.Saved == nil {
		t.Fatal("wounding needs an original city source Human")
	}
	s := &m.Carry.LiveLoad.Inventory.Source
	if mana {
		s.Stats[sav.StatMana], m.Saved.Mana = cityPotionWoundMana, cityPotionWoundMana
		return
	}
	s.Stats[sav.StatHealth], m.Saved.HP = cityPotionWoundHealth, cityPotionWoundHealth
}

type cityPotionObservation struct {
	HP, Mana, HealthRegeneration, ManaRegeneration int32
	Effects                                        []sim.ActiveEffect
}

func cityPotionObserve(f *FrontEnd, id sim.EntityID) cityPotionObservation {
	e, _ := f.live.entity(id)
	o := cityPotionObservation{HP: e.HP, Mana: e.Mana, HealthRegeneration: e.HealthRegeneration, ManaRegeneration: e.ManaRegeneration}
	for _, effect := range f.live.world.ActiveEffects() {
		if effect.Target == id {
			o.Effects = append(o.Effects, effect)
		}
	}
	return o
}

func cityPotionOwner(t *testing.T, f *FrontEnd, memberID string) sim.EntityID {
	t.Helper()
	for i, p := range f.live.mission.party {
		if p.ID == memberID {
			return f.live.mission.ids[i]
		}
	}
	t.Fatal("next mission omitted the potion owner")
	return 0
}

type cityPotionExpect struct {
	effect                      *sim.ActiveEffect
	health, mana                int32
	healthRegen, manaRegen      int32
	hp0, mp0                    int32
	observable                  bool
	memberID                    string
	preUse                      mapload.PartyMember
	party                       []mapload.PartyMember
	live, f                     *FrontEnd
	liveApp, app                *ui.App
	store                       SaveStore
	shop                        *townScreen
	kind                        cityPotionKind
	origin                      string
	derivedBefore, derivedAfter data.Derived
	horizonTicks                int
	wounded                     bool
}

func cityPotionKindWire(raw []byte, x *cityPotionExpect) error {
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World != nil {
		return fmt.Errorf("ordinary city SAV: %v", err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		return fmt.Errorf("current city policy: %v", err)
	}
	var object uint16
	for _, p := range a.Party {
		if string(p.ID) != x.memberID {
			continue
		}
		var recorded *sim.ActiveEffect
		if p.City != nil {
			recorded = p.City.Potion
		}
		if !reflect.DeepEqual(recorded, x.effect) {
			return fmt.Errorf("ordinary SAVE potion record %+v want %+v", recorded, x.effect)
		}
		for _, binding := range a.Bindings {
			if !binding.Structure && !binding.Missing && binding.ID == p.Entity {
				object = binding.Object
			}
		}
	}
	if object == 0 {
		return fmt.Errorf("potion member has no ordinary actor binding")
	}
	characters, err := sav.ReadDocumentCharacters(doc, []uint16{object})
	if err != nil || len(characters) != 1 || characters[0].Character.Basis == nil {
		return fmt.Errorf("ordinary Human: %v", err)
	}
	basis := characters[0].Character.Basis
	if refs, _ := savedObjectRefs(&doc.Objects[object-1], "Effects"); len(refs) != 0 {
		return fmt.Errorf("city Human Effects list holds %d objects", len(refs))
	}
	modifier := basis.Human.Fields.Modifier
	got := [4]int32{int32(int16(basis.Stats[sav.StatHealth])), int32(int16(basis.Stats[sav.StatMana])),
		int32(int16(binary.LittleEndian.Uint16(modifier[10:]))), int32(int16(binary.LittleEndian.Uint16(modifier[14:])))}
	if want := [4]int32{x.health, x.mana, x.healthRegen, x.manaRegen}; got != want {
		return fmt.Errorf("ordinary Human health, mana, health regeneration, mana regeneration %v want %v", got, want)
	}
	return nil
}

func cityPotionKindUse(t *testing.T, kind cityPotionKind, origin string) *cityPotionExpect {
	t.Helper()
	x := &cityPotionExpect{kind: kind, origin: origin}
	var f *FrontEnd
	var app *ui.App
	store := SaveStore{Dir: t.TempDir()}
	if origin == "current arrival" {
		f, app = saveDialogArrivedTown(t, store.Dir)
		t.Cleanup(app.StopAudio)
	} else {
		_, raw := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
		f, app, store = cityPotionLoadApp(t, raw)
	}
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	shop := f.TownScreen().(*townScreen)
	for n := 0; shop.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if shop.room != roomShop {
		t.Fatal("shop did not open", shop.room)
	}
	shop.shopMember = kind.member
	member := shop.shopPartyMember(shop.shopMemberIndex())
	if member == nil || member.PotionEffect != nil {
		t.Fatal("fixture requires a member without an existing potion")
	}
	x.wounded = origin == "original city" && !kind.timed
	if x.wounded {
		cityPotionWound(t, member, kind.mana)
	}
	x.memberID = member.ID
	var hp0, mp0 int32
	x.derivedBefore, hp0, mp0 = mapload.PartyDisplayWithTable(*member, f.Table)
	x.hp0, x.mp0 = hp0, mp0
	if kind.mana && x.derivedBefore.ManaMax == 0 || x.wounded && (kind.mana && mp0 != cityPotionWoundMana || !kind.mana && hp0 != cityPotionWoundHealth) {
		t.Fatalf("fixture member %s pools hp=%d mana=%d max mana %d", x.memberID, hp0, mp0, x.derivedBefore.ManaMax)
	}
	x.preUse = mapload.CloneParty([]mapload.PartyMember{*member})[0]

	var item ShopItem
	var found bool
	for _, s := range shopPotionStock(f.Table, rand.New(rand.NewSource(1))) {
		if f.Table.MagicItems.EntryName(int(s.Code&0xff)) == kind.name {
			item, found = s, true
		}
	}
	if !found || len(item.Effects) != 1 {
		t.Fatal("installed shop does not sell a one-effect", kind.name)
	}
	f.Shop.shelves[ShelfBooks] = []ShopItem{item}
	f.Town.gold = 100000
	cityPotionPointer(t, app, "shelf_pick", 3, "press", "release")
	gold, quantity := f.Town.Gold(), f.Shop.Shelf(ShelfBooks)[0].Count
	packAt := releaseShopBuyToPack(t, app, f, uint16(item.Code))
	if err := app.HeadlessPointer("press", packAt.X, packAt.Y); err != nil {
		t.Fatal(err)
	}
	cityPotionPointer(t, app, "doll_box", 0, "move", "release")
	member = shop.shopPartyMember(shop.shopMemberIndex())
	if f.Town.Gold() != gold-int(item.Price) || f.Shop.Shelf(ShelfBooks)[0].Count != quantity-1 {
		t.Fatal("potion use did not pay for exactly one merchant item")
	}

	var hp1, mp1 int32
	x.derivedAfter, hp1, mp1 = mapload.PartyDisplayWithTable(*member, f.Table)
	x.health, x.mana = hp1, mp1
	x.healthRegen, x.manaRegen = x.derivedAfter.HealthRegeneration, x.derivedAfter.ManaRegeneration
	if kind.timed {
		x.effect = member.PotionEffect
		want := sim.EffectHealthRegeneration
		if kind.mana {
			want = sim.EffectManaRegeneration
		}
		if x.effect == nil || x.effect.Kind != want || x.effect.Spell != 0 || x.effect.Magnitude != 100 || x.effect.Remaining != 960 {
			t.Fatalf("production potion use: %+v", x.effect)
		}
		wantHealthReg, wantManaReg := x.derivedBefore.HealthRegeneration, x.derivedBefore.ManaRegeneration
		if kind.mana {
			wantManaReg += 100
		} else {
			wantHealthReg += 100
		}
		if x.healthRegen != wantHealthReg || x.manaRegen != wantManaReg || hp1 != hp0 || mp1 != mp0 {
			t.Fatalf("city modifier %d/%d want %d/%d", x.healthRegen, x.manaRegen, wantHealthReg, wantManaReg)
		}
	} else {
		if member.PotionEffect != nil {
			t.Fatalf("one-shot potion stored a timer %+v", member.PotionEffect)
		}
		amount := int32(item.Effects[0].Operand)
		wantHP, wantMana := hp0, mp0
		if kind.mana {
			wantMana = min(mp0+amount, x.derivedBefore.ManaMax)
		} else {
			wantHP = min(hp0+amount, x.derivedBefore.HealthMax)
		}
		if hp1 != wantHP || mp1 != wantMana {
			t.Fatalf("production potion use: hp %d->%d mana %d->%d want %d/%d", hp0, hp1, mp0, mp1, wantHP, wantMana)
		}
		if x.wounded && hp1 == hp0 && mp1 == mp0 {
			t.Fatal("potion did not change the wounded pool")
		}
	}
	x.observable = kind.timed || x.wounded
	x.horizonTicks = 64
	if kind.timed {
		x.horizonTicks = 961
	}
	x.live, x.liveApp, x.f, x.app, x.store, x.shop = f, app, f, app, store, shop
	x.party = mapload.CloneParty(f.Carried)
	return x
}

func cityPotionOpenMissions(t *testing.T, x *cityPotionExpect) {
	t.Helper()
	for i, target := range []*FrontEnd{x.live, x.f} {
		missionApp := []*ui.App{x.liveApp, x.app}[i]
		if err := missionApp.OpenMission(target.MissionOpener(target.Town.Chapter())); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseCityPotionKindsF2SAVAndMissionExpiry(t *testing.T) {
	for _, kind := range cityPotionKinds {
		for _, origin := range []string{"current arrival", "original city"} {
			t.Run(kind.name+"/"+origin, func(t *testing.T) { cityPotionKindRun(t, kind, origin) })
			if kind.timed || origin == "original city" {
				t.Run(kind.name+"/"+origin+"/record dropped", func(t *testing.T) { cityPotionKindDropControl(t, kind, origin) })
			}
		}
	}
}

func cityPotionKindDropControl(t *testing.T, kind cityPotionKind, origin string) {
	x := cityPotionKindUse(t, kind, origin)
	if !x.observable {
		t.Fatal("the control needs an observable effect")
	}
	at := x.shop.shopMemberIndex()
	current := mapload.CloneParty([]mapload.PartyMember{*x.shop.shopPartyMember(at)})[0]
	*x.shop.shopPartyMember(at) = x.preUse
	raw := cityRosterF2Save(t, x.app, x.store, "dropped")
	*x.shop.shopPartyMember(at) = current
	if cityPotionKindWire(raw, x) == nil {
		t.Fatal("a SAVE written without the potion was read back as carrying it")
	}
	cold, coldApp, _ := cityPotionLoadApp(t, raw)
	x.f, x.app = cold, coldApp
	cityPotionOpenMissions(t, x)
	if reflect.DeepEqual(cityPotionObserve(x.live, cityPotionOwner(t, x.live, x.memberID)), cityPotionObserve(x.f, cityPotionOwner(t, x.f, x.memberID))) {
		t.Fatal("next mission is identical without the potion record, so the witness cannot detect its loss")
	}
}

func cityPotionKindRun(t *testing.T, kind cityPotionKind, origin string) {
	x := cityPotionKindUse(t, kind, origin)
	f, app, store := x.live, x.app, x.store
	for cycle := 0; cycle < 2; cycle++ {
		raw := cityRosterF2Save(t, app, store, fmt.Sprintf("kind-%d", cycle))
		if err := cityPotionKindWire(raw, x); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(x.live.Carried, x.party) {
			t.Fatal("city SAVE mutated the uninterrupted party")
		}
		cold, coldApp, coldStore := cityPotionLoadApp(t, raw)
		cityRosterSame(t, x.live, cold)
		for i, p := range cold.Carried {
			if !reflect.DeepEqual(p.PotionEffect, x.live.Carried[i].PotionEffect) {
				t.Fatal("cold city restarted or dropped a potion", p.ID)
			}
			if p.ID == x.memberID {
				d, hp, mp := mapload.PartyDisplayWithTable(p, cold.Table)
				if d.HealthRegeneration != x.healthRegen || d.ManaRegeneration != x.manaRegen || hp != x.health || mp != x.mana {
					t.Fatalf("cold city member hp=%d mana=%d regeneration=%d/%d want %d/%d %d/%d", hp, mp,
						d.HealthRegeneration, d.ManaRegeneration, x.health, x.mana, x.healthRegen, x.manaRegen)
				}
			}
		}
		f, app, store = cold, coldApp, coldStore
	}
	x.f, x.app = f, app
	cityPotionOpenMissions(t, x)
	live := x.live
	id := cityPotionOwner(t, live, x.memberID)
	coldID := cityPotionOwner(t, f, x.memberID)
	for tick := 0; tick <= x.horizonTicks; tick++ {
		cityPotionMissionEqual(t, live.live.world, f.live.world, tick)
		if tick == 0 {
			if a, b := cityPotionObserve(live, id), cityPotionObserve(f, coldID); !reflect.DeepEqual(a, b) {
				t.Fatalf("tick 0 observation differs: live %+v cold %+v", a, b)
			}
			if o := cityPotionObserve(f, coldID); !kind.timed && (o.HP != x.health || o.Mana != x.mana) {
				t.Fatalf("next mission pools hp=%d mana=%d want %d/%d", o.HP, o.Mana, x.health, x.mana)
			}
		}
		for _, target := range []*FrontEnd{live, f} {
			e, _ := target.live.entity(id)
			count := 0
			for _, current := range target.live.world.ActiveEffects() {
				if current.Target == id && current.Spell == 0 {
					count++
					if !kind.timed || tick >= 960 || current.Kind != x.effect.Kind || current.Magnitude != 100 || current.Remaining != uint16(960-tick) {
						t.Fatalf("tick %d potion attachment %+v", tick, current)
					}
				}
			}
			if !kind.timed {
				if count != 0 {
					t.Fatal("one-shot potion left an attachment", tick)
				}
				continue
			}
			expectedHealth, expectedMana := x.healthRegen, x.manaRegen
			if tick >= 960 {
				expectedHealth, expectedMana = x.derivedBefore.HealthRegeneration, x.derivedBefore.ManaRegeneration
			}
			if e.HealthRegeneration != expectedHealth || e.ManaRegeneration != expectedMana {
				t.Fatalf("tick %d regeneration=%d/%d want %d/%d", tick, e.HealthRegeneration, e.ManaRegeneration, expectedHealth, expectedMana)
			}
			if tick < 960 && count != 1 || tick >= 960 && count != 0 {
				t.Fatal("potion duplicated or failed to expire", tick, count)
			}
		}
		if tick < x.horizonTicks {
			sim.Step(live.live.world, nil)
			sim.Step(f.live.world, nil)
		}
	}
	t.Logf("%s %s: hp %d->%d mana %d->%d regeneration %d/%d->%d/%d, two F2 city SAV cycles, live and cold hashes equal through tick %d",
		kind.name, origin, x.hp0, x.health, x.mp0, x.mana, x.derivedBefore.HealthRegeneration, x.derivedBefore.ManaRegeneration,
		x.healthRegen, x.manaRegen, x.horizonTicks)
}
