//go:build sessioncorpusaudit

package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestLowSAVResaveCorpusCensus(t *testing.T) {
	worlds, resaved, clockNear, offMap, offMapZero := 0, 0, 0, 0, 0
	var refused, projectionUnavailable []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		worlds++
		open, town, err := fe.RestoreOriginal(mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		if town {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, fmt.Errorf("world save opened as town")})
			return
		}
		if err := fe.App("low SAV corpus").OpenMission(open); err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		snapshot, label, err := fe.Snapshot(true)
		if err != nil {
			projectionUnavailable = append(projectionUnavailable, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		wire, err := fe.ExportCurrentSave(snapshot, label)
		if err != nil {
			projectionUnavailable = append(projectionUnavailable, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		written, err := sav.DecodeDocumentData(wire)
		if err != nil {
			projectionUnavailable = append(projectionUnavailable, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		resaved++
		delta := int64(written.Head.CounterB)*16 - int64(written.Head.CounterA)
		if delta >= -16 && delta <= 16 {
			clockNear++
		} else {
			t.Logf("clock outside A/16 by more than one %s: A=%d B=%d", mf.rel, written.Head.CounterA, written.Head.CounterB)
		}
		for i := range written.Objects {
			record := &written.Objects[i]
			if record.Class != "Human" && record.Class != "Unit" {
				continue
			}
			flags, err := savedStructureValue(record, "U4C")
			if err != nil || flags&sav.ActorOffMapFlag == 0 {
				continue
			}
			offMap++
			mask, err := savedStructureValue(record, "T18")
			if err == nil && mask == 0 {
				offMapZero++
			} else {
				t.Logf("off-map mask %s object %d: %#x (%v)", mf.rel, i+1, mask, err)
			}
		}
	})
	milestone2LogRefusals(t, "low SAV corpus", refused)
	milestone2LogRefusals(t, "low SAV projection", projectionUnavailable)
	t.Logf("low SAV corpus: %d world files, %d resaved, %d B near A/16, %d/%d off-map actor masks zero, %d resume refusals, %d projection refusals", worlds, resaved, clockNear, offMapZero, offMap, len(refused), len(projectionUnavailable))
	if worlds == 0 || resaved == 0 {
		t.Fatal("no world SAV resave population")
	}
}
