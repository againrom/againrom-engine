package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestPartyDisplayUsesCurrentSavedPools(t *testing.T) {
	p := PartyMember{Hero: data.Hero{Body: 20, Reaction: 25, Mind: 30, Spirit: 35}, Profile: data.Profile{ManaColumn: true, HealthColumn: true}}
	base, hp, mp := PartyDisplayWithTable(p, nil)
	p.Saved = &Saved{HP: 17, MaxHP: 257, Mana: 19, MaxMana: 113}
	d, gotHP, gotMP := PartyDisplayWithTable(p, nil)
	if gotHP != 17 || d.HealthMax != 257 || gotMP != 19 || d.ManaMax != 113 {
		t.Fatalf("current pools got %d/%d %d/%d want 17/257 19/113", gotHP, d.HealthMax, gotMP, d.ManaMax)
	}
	p.PotionEffect = &sim.ActiveEffect{Kind: sim.EffectHealthRegeneration, Mode: sim.EffectDuration, Magnitude: 3, Remaining: 20}
	d, gotHP, gotMP = PartyDisplayWithTable(p, nil)
	if d.HealthRegeneration != base.HealthRegeneration+3 || gotHP != 17 || gotMP != 19 || d.HealthMax != 257 || d.ManaMax != 113 {
		t.Fatal("timer changed current pools or applied twice")
	}
	p.PotionEffect, p.Saved = nil, nil
	d, gotHP, gotMP = PartyDisplayWithTable(p, nil)
	if d != base || gotHP != hp || gotMP != mp {
		t.Fatal("absent Saved changed constructor projection")
	}
}

func TestNativeTownRegenerationPotionKeepsCurrentPools(t *testing.T) {
	p := PartyMember{Hero: data.Hero{Body: 20, Reaction: 25, Mind: 30, Spirit: 35}, Profile: data.Profile{ManaColumn: true, HealthColumn: true}, Saved: &Saved{HP: 17, MaxHP: 257, Mana: 19, MaxMana: 113}}
	result, ok := ApplyTownPotion(p, sim.ItemInstance{Code: 0xe08, Kind: 3, Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 3 | 20<<16}}}, nil)
	if !ok || result.PotionEffect == nil {
		t.Fatal("actual native timer attachment")
	}
	if *result.Saved != *p.Saved {
		t.Fatalf("regeneration attachment changed current pools: got %+v want %+v", result.Saved, p.Saved)
	}
}
