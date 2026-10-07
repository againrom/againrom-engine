package game

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Explicitly synthetic: mutate only the source hero's current/modifier pair.
// This makes no claim that a lawful original process produced these values.
func secondPhysicalCity1104(t *testing.T, source []byte, current, modifier [2]byte) []byte {
	t.Helper()
	f, err := sav.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	hero := ""
	for _, c := range p.Roster() {
		if c.Hero {
			hero = c.Name
		}
	}
	d, changed := p.Data(), 0
	for _, o := range d.Objects {
		if u := o.Unit; u != nil && u.Name == hero {
			u.RawA6[17], u.RawA6[18] = current[0], current[1]
			u.RawD4[35], u.RawD4[36] = modifier[0], modifier[1]
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("synthetic mutation selected %d heroes", changed)
	}
	p, err = sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	u := sav.CityUpdate{Label: []byte("synthetic second physical")}
	for _, player := range f.Players {
		if player.Participant == 0 {
			u.Money = player.Money
			break
		}
	}
	for _, c := range p.Roster() {
		u.Characters = append(u.Characters, originalCityBaselineUpdate(c))
	}
	out, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSecondPhysicalCityTrainAndSAVWire(t *testing.T) {
	for _, pair := range [][2]byte{{20, 7}, {0, 255}, {255, 0}, {0, 0}} {
		f := city1099Front(t, secondPhysicalCity1104(t, city1099Fixture(t), [2]byte{91, 3}, pair))
		h, _ := trainingPartyMember(t, f, "hero").OriginalHumanState()
		if d := h.Derived(nil, 0); d.Combat.SecondBase != 91 || d.Combat.SecondSpread != 3 {
			t.Fatal("LOAD recomputed current pair")
		}
		if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 12 for 471" {
			t.Fatal(msg)
		}
		if f.Town.Gold() != 2529 {
			t.Fatal("purse")
		}
		h, _ = trainingPartyMember(t, f, "hero").OriginalHumanState()
		if h.Attack.SecondBase != pair[0] || h.Attack.SecondSpread != pair[1] {
			t.Fatal("Train accumulated current instead of clearing/folding")
		}
		s, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportOriginalSave(s, "second physical")
		if err != nil {
			t.Fatal(err)
		}
		file, _ := sav.Open(raw)
		p, _ := file.CityProvenance()
		for _, c := range p.Roster() {
			if !c.Hero {
				continue
			}
			human, err := p.Human(c.Identity)
			if err != nil {
				t.Fatal(err)
			}
			// Wire offsets are literal source block offsets, not Derived output.
			if !bytes.Equal(human.Fields.Attack[17:19], pair[:]) || !bytes.Equal(human.Fields.Modifier[35:37], pair[:]) {
				t.Fatal("SAV pair bytes")
			}
		}
		native, err := EncodeSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		back, _, err := DecodeSave(native)
		if err != nil {
			t.Fatal(err)
		}
		if got := back.Party[0].OriginalHuman.State.Attack; got.SecondBase != pair[0] || got.SecondSpread != pair[1] {
			t.Fatal("AGS pair")
		}
	}
}

func TestSecondPhysicalNativeRearmAndEarnedLevelRetainObservedPair(t *testing.T) {
	table, h := eqDefsTable(t), eqBladeHero()
	r := h.Reward()
	a := sim.Entity{ID: 7, X: 3, Y: 3, HP: 100, MaxHP: 100, Reach: 1, Owner: 2,
		GainsXP: true, TypeID: sim.HumanTypeID, Mind: r.Mind, SkillXP: r.SkillXP,
		XPSlot: uint8(data.SkillBlade), DamageBase: 5, AlwaysHits: true,
		SecondBase: 20, SecondSpread: 7, AttackCharge: 1, AttackRelax: 1, Facing: 64, DesiredFacing: 64}
	a.Skill[data.SkillBlade] = h.Skill[data.SkillBlade]
	v := sim.Entity{ID: 8, X: 4, Y: 3, HP: 1000000, MaxHP: 1000000, DyingTime: 200, Owner: 3, XPValue: 4}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{a, v}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 7, Items: []uint16{eqMaceCode}}})
	if err != nil {
		t.Fatal(err)
	}
	mw := equipMission(t, w, 7, h, eqSword(t, table), table)
	mw.enqueueEquip(0)
	mw.tick()
	if e := gaEntity(t, w, 7); e.SecondBase != 20 || e.SecondSpread != 7 {
		t.Fatal("equipment erased observed pair")
	}
	mw.pending = append(mw.pending, sim.Command{Kind: sim.KindAttack, Entity: 7, X: 8})
	mw.tick()
	if e := gaEntity(t, w, 7); e.Skill[data.SkillBludgen] != 1 || e.SecondBase != 20 || e.SecondSpread != 7 {
		t.Fatal("earned level erased pair or did not run")
	}
	// The unsupported campaign settlement is deliberately outside the route.
	p := mapload.CarryParty(mw.mission.party, w, mw.mission.ids)
	if len(p) != 1 || p[0].OriginalHuman != nil {
		t.Fatal("CarryParty retained stale source basis")
	}
}

func TestSecondPhysicalPurchaseTrainingAndMalformedCurrentPair(t *testing.T) {
	for _, pair := range [][2]byte{{1, 0}, {0, 1}} {
		for _, retired := range []bool{false, true} {
			f := city1099Front(t, secondPhysicalCity1104(t, city1099Fixture(t), pair, [2]byte{}))
			if retired {
				buyTrainingItem(t, f)
			} else {
				f.Carried[0].Hero.Body++
			}
			if retired {
				h := currentCitySaleHuman(t, trainingPartyMember(t, f, "hero"))
				if [2]byte{h.Attack.SecondBase, h.Attack.SecondSpread} != pair {
					t.Fatal("purchase lost the current pair", h.Attack)
				}
				if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 12 for 471" || f.Town.Gold() != 2519 {
					t.Fatal("current pair blocked training", msg, f.Town.Gold())
				}
				h = currentCitySaleHuman(t, trainingPartyMember(t, f, "hero"))
				if h.Attack.SecondBase != 0 || h.Attack.SecondSpread != 0 || h.Modifier.Attack.SecondBase != 0 || h.Modifier.Attack.SecondSpread != 0 {
					t.Fatal("training did not fold the current zero modifier", h.Attack)
				}
				reloadTrainingCity(t, f)
				if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 13 for 518" || f.Town.Gold() != 2001 {
					t.Fatal("cold current-pair continuation", msg, f.Town.Gold())
				}
				reloadTrainingCity(t, f)
				continue
			}
			before, gold := mapload.CloneParty(f.Carried), f.Town.Gold()
			if msg := train1099(t, f, "hero", 0); strings.HasPrefix(msg, "trained ") {
				t.Fatal("fallback trained", msg)
			}
			if !reflect.DeepEqual(f.Carried, before) || f.Town.Gold() != gold {
				t.Fatal("refusal mutated state")
			}
			if _, err := StartMissionFrom(&alm.Map{}, "fallback", 10, f.Table, 0, f.Carried); err == nil || !strings.Contains(err.Error(), "second physical") {
				t.Fatal("fallback lost observed current pair", err)
			}
		}
	}
}
