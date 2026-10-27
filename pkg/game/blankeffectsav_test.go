package game

import (
	"testing"

	"againrom/pkg/sim"
)

// A second-game Stone Curse record is blank: kind none, magnitude 0, a timed
// mode, the row's own id (R2-ENGINE-453). The SAV Effect writer gives it tag
// 0 and the reader brings back the same record; tag 0 under spell 0 and an
// unknown tag stay refused.
func TestBlankEffectRecordRoundTripsThroughTheSAVEffectFields(t *testing.T) {
	e := sim.ActiveEffect{Target: 7, Spell: 18, Kind: sim.EffectNone, Mode: sim.EffectDuration, Remaining: 140}
	if _, err := newEffectAttachment(e); err != nil {
		t.Fatalf("blank record refused by the writer: %v", err)
	}
	row := sim.SavedEffectObject{E0C: 18, Value: sim.ItemEffect{Kind: 0, Mode: uint8(sim.EffectDuration), Operand: 140 << 16}}
	back, err := originalActorEffect(row, 7)
	if err != nil || back != e {
		t.Fatalf("tag 0 read back %+v, %v; want %+v", back, err, e)
	}
	row.E0C = 0
	if _, err := originalActorEffect(row, 7); err == nil {
		t.Fatal("tag 0 under spell 0 admitted")
	}
	row.E0C, row.Value.Kind = 18, 99
	if _, err := originalActorEffect(row, 7); err == nil {
		t.Fatal("an unknown tag admitted")
	}
}
