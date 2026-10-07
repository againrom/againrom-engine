package sav

import (
	"reflect"
	"testing"
)

func TestPieceImportsOrderedStateZeroEffectsAndReportsOtherStates(t *testing.T) {
	state0a := newRecord("Effect", 0, 2)
	state0a.Value["E0C"] = 0
	state0a.Value["E3C"] = 8
	state0a.Value["E3D"] = 1
	state0a.Value["E40"] = 0x03c00064
	unsupported := newRecord("Effect", 0, 3)
	unsupported.Value["E0C"] = 12
	unsupported.Value["E3C"] = 15
	unsupported.Value["E40"] = 999
	state0b := newRecord("Effect", 0, 4)
	state0b.Value["E3C"] = 21
	state0b.Value["E40"] = 7

	record := newRecord("Item", 0, 1)
	record.Value["F40"] = 0x0e06
	record.Value["T0C"] = 6
	record.Value["F42"] = 3
	record.Value["F44"] = 3
	record.Value["T1C"] = ^uint32(0)
	record.Value["F4A"] = 0xfff7
	record.Refs["Effects"] = []*Record{state0a, unsupported, state0b}

	got := piece(record)
	wantEffects := []ItemEffect{
		{Kind: 8, Mode: 1, Operand: 0x03c00064},
		{Kind: 21, Operand: 7},
	}
	if got.Class != "Item" || got.Code != 0x0e06 || got.Row != 6 || got.Stack != 3 ||
		got.Kind != 3 || got.Price != -1 || got.Weight != -9 || !reflect.DeepEqual(got.Effects, wantEffects) ||
		!reflect.DeepEqual(got.UnsupportedEffectStates, []uint8{12}) {
		t.Fatalf("piece = %+v, want ordered state-0 effects %+v and unsupported [12]", got, wantEffects)
	}
}

func TestPieceDivertsANonEffectClassEvenAtStateZero(t *testing.T) {
	direct := newRecord("Effect_DirectDamage", 0, 2)
	direct.Value["E0C"] = 0
	direct.Value["E3C"] = 8
	direct.Value["E3D"] = 1
	direct.Value["E40"] = 0x03c00064

	record := newRecord("Item", 0, 1)
	record.Value["F40"] = 0x0e06
	record.Value["F42"] = 1
	record.Refs["Effects"] = []*Record{direct}

	got := piece(record)
	if len(got.Effects) != 0 {
		t.Fatalf("piece = %+v, an Effect_DirectDamage record must not be read as a supported Effect", got)
	}
	if !reflect.DeepEqual(got.UnsupportedEffectStates, []uint8{0}) {
		t.Fatalf("UnsupportedEffectStates = %v, want [0] (diverted by class, not by a nonzero state)", got.UnsupportedEffectStates)
	}
}
