//go:build sessioncorpusaudit

package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestMilestone2DyingOwnerActors reads the file side directly. ActorGraph
// supplies only object enumeration and the starts of two serialized runs.
// SAV-UNITPROG-156 gives both layouts: Control is 4 bytes + three raw dwords
// + bytes 60/61/6c; State is CString + fourteen stat words + the fixed suffix.
// Stat words 5/6/8/9/11/12 are own weight/load/HP/max HP/mana/max mana;
// the stage byte follows 28+2+2+2+1+4+3+4 = 46 bytes after the CString.
// SAV-DEADLOAD-126 names signed HP/timer and unsigned stage independently.
// No decoded Actor.Stage/HP/DyingTimer, ActorHoldings or stat array is the
// expected value. Token head identity and position come from SAV-TOKEN-034.
func TestMilestone2DyingOwnerActors(t *testing.T) {
	files, subjects, checked, mismatches := 0, 0, 0, 0
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		graph, err := mf.f.ActorGraph()
		if err != nil {
			t.Fatal(err)
		}
		type expected struct {
			identity                               uint32
			cell                                   uint16
			hp, maxHP, mana, maxMana, weight, load int16
			timer                                  int8
			stage                                  uint8
		}
		var wants []expected
		for _, a := range graph.Actors {
			b, p := mf.body, a.StateOff
			if a.ControlOff <= 0 || p <= 0 || p >= len(b) {
				t.Fatal("missing serialized Unit run locator")
			}
			n := int(b[p])
			if n == 255 || p+1+n+55 > len(b) {
				t.Fatal("Unit state CString exceeds this byte witness's bounded layout")
			}
			p += 1 + n
			hp, stage := int16(rawU16(b, p+16)), b[p+46]
			if stage != 1 || hp > 0 || rawU32(b, a.Off+12) == 0 {
				continue
			}
			wants = append(wants, expected{rawU32(b, a.Off+29), rawU16(b, a.Off), hp,
				int16(rawU16(b, p+18)), int16(rawU16(b, p+22)), int16(rawU16(b, p+24)),
				int16(rawU16(b, p+10)), int16(rawU16(b, p+12)), int8(b[a.ControlOff+18]), stage})
		}
		subjects += len(wants)
		if len(wants) == 0 {
			return
		}
		ms, _, err := ResumeOriginalSave(fe.Archives.Containers, mf.raw, fe.Table, mapload.DifficultyNormal, nil, fe.Bodies)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		files++
		byIdentity := map[uint32]sim.Entity{}
		for _, e := range ms.World.Entities() {
			if e.SourceBinding.Class == 0 {
				continue
			}
			if _, exists := byIdentity[e.SourceBinding.Identity]; exists {
				t.Fatal("duplicate live source identity")
			}
			byIdentity[e.SourceBinding.Identity] = e
		}
		for _, want := range wants {
			e, ok := byIdentity[want.identity]
			if !ok {
				mismatches++
				t.Errorf("dying source %#x has no live source binding", want.identity)
				continue
			}
			checked++
			if e.Alive() || e.HP != int32(want.hp) || e.MaxHP != int32(want.maxHP) ||
				e.Mana != int32(want.mana) || e.MaxMana != int32(want.maxMana) ||
				uint8(e.Decay) != want.stage || e.Dwell != uint16(want.timer) ||
				e.X != int32(want.cell&255) || e.Y != int32(want.cell>>8) ||
				!e.ActorLoad.Present || e.ActorLoad.OwnWeight != want.weight || e.Load != int32(want.load) {
				mismatches++
				t.Errorf("dying source %#x: file=%+v; live HP=%d/%d mana=%d/%d stage=%d dwell=%d xy=%d,%d weight/load=%d/%d present=%v alive=%v", want.identity, want, e.HP, e.MaxHP, e.Mana, e.MaxMana, e.Decay, e.Dwell, e.X, e.Y, e.ActorLoad.OwnWeight, e.Load, e.ActorLoad.Present, e.Alive())
			}
		}
	})
	milestone2LogRefusals(t, "dying owner actors", refused)
	t.Logf("dying owner actors: %d subject files resumed; %d raw subjects, %d live compared, %d mismatches, %d refused files", files, subjects, checked, mismatches, len(refused))
	if subjects == 0 || checked == 0 {
		t.Fatal("no dying owner-graph subject compared")
	}
}
