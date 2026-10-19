package game

import (
	"bytes"
	"crypto/sha256"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/random"
)

func TestReleaseSchoolYardSounds(t *testing.T) {
	f, a, tr, _ := townFront(t, 7, false)
	rec := &roomSoundRecorder{}
	f.SoundPlayer = rec
	f.TownAmbientRandom = func(n int) int { return n - 1 }
	for _, key := range yardKeys {
		sample, ok := f.SoundBank.namedSample(key)
		if !ok || len(sample.PCM) == 0 {
			t.Fatalf("installed sample %s missing", key)
		}
	}
	s := f.townUI
	enterTownRoom(t, a, s, "SCHOOL", roomSchool)
	for _, mage := range []bool{false, true} {
		f.Carried[0].Mage = mage
		s.enterSchoolPage()
		rec.requests, rec.voices = nil, nil
		start := *tr.now
		paint := func(ms int) {
			*tr.now = start.Add(time.Duration(ms) * time.Millisecond)
			s.AdvanceTownSurfaceAnimation()
		}
		paint(1500)
		if len(yardSelectors(rec)) != 0 {
			t.Fatal("early command")
		}
		paint(1501)
		paint(4499)
		if got := yardSelectors(rec); !reflect.DeepEqual(got, []string{yardKeys[0]}) {
			t.Fatal(got)
		}
		paint(4500)
		fight := yardKeys[2]
		if mage {
			fight = yardKeys[3]
		}
		if got := yardSelectors(rec); !reflect.DeepEqual(got, []string{yardKeys[0], yardKeys[1], fight}) {
			t.Fatal(got)
		}
		var v *exteriorVoice
		for i, r := range rec.requests {
			if r.Selector == fight {
				if !r.Repeat || r.Priority != 128 {
					t.Fatalf("fight %+v", r)
				}
				v = rec.voices[i]
			}
		}
		paint(84499)
		if v == nil || !v.Playing() {
			t.Fatal("fight ended at equality")
		}
		paint(84500)
		if v.Playing() || v.stops != 1 {
			t.Fatal("fight did not stop")
		}
		if got := yardSelectors(rec); !reflect.DeepEqual(got, []string{yardKeys[0], yardKeys[1], fight, yardKeys[4]}) {
			t.Fatal(got)
		}
		t.Logf("mage=%t: Command1 1501ms; Command2 + %s loop 4500ms; stop + Command3 84500ms", mage, fight)
	}
}

func schoolYardStateWitness(t *testing.T, original, bypassDraw bool) (uint32, []byte, []uint64) {
	t.Helper()
	f, a, tr, _ := townFront(t, 7, original)
	if bypassDraw {
		f.TownAmbientRandom = func(int) int { return 0 }
	}
	s := f.townUI
	enterTownRoom(t, a, s, "SCHOOL", roomSchool)
	start := *tr.now
	for _, ms := range []int{1501, 5000, 90000} {
		*tr.now = start.Add(time.Duration(ms) * time.Millisecond)
		s.AdvanceTownSurfaceAnimation()
	}
	shared := f.randomService().SharedState()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(f.MissionOpener(41)); err != nil {
		t.Fatal(err)
	}
	for n := 0; a.HeadlessNoticeOpen() && n < 16; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	hashes := []uint64{f.live.world.Hash()}
	for i := 0; i < 300; i++ {
		f.live.tick()
	}
	return shared, saved, append(hashes, f.live.world.Hash())
}

func TestReleaseSchoolYardDrawsKeepMissionHashes(t *testing.T) {
	for _, original := range []bool{false, true} {
		t.Run(map[bool]string{false: "seeded", true: "original"}[original], func(t *testing.T) {
			controlShared, controlSave, control := schoolYardStateWitness(t, original, true)
			shared, saved, hashes := schoolYardStateWitness(t, original, false)
			if !reflect.DeepEqual(control, hashes) {
				t.Fatalf("school draws moved mission hashes: %x != %x", control, hashes)
			}
			if original && shared == controlShared {
				t.Fatal("shared-stream loss control did not consume draws")
			}
			if !original && !bytes.Equal(controlSave, saved) {
				t.Fatal("seeded school draws moved SAV bytes")
			}
			t.Logf("Original=%t TownAmbient joins shared=%t: shared %#x/%#x, town SAV %x/%x, equal=%t; mission41 hashes %x", original, random.JoinsShared(random.TownAmbient), controlShared, shared, sha256.Sum256(controlSave), sha256.Sum256(saved), bytes.Equal(controlSave, saved), hashes)
		})
	}
}
