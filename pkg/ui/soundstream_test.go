package ui

import (
	"crypto/sha256"
	"fmt"
	"image"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
	"time"
)

// soundStreamDigest drives one viewer through a seeded stream of snapshots,
// catch-up runs, damage messages, fog changes, entries and frames, and
// hashes every play each device received with its placement, then the final
// voice memories. Fields the sound step never reads vary too.
func soundStreamDigest(t *testing.T, seed uint64) string {
	t.Helper()
	v, effects, speech := hurtDevices(t)
	r := rand.New(rand.NewPCG(seed, seed^0x5eed))
	voices := []string{"", "mf_hero", "f_mage"}
	units := make([]MapEntity, 9)
	for i := range units {
		units[i] = hurtUnit(uint32(i+1), uint32(r.IntN(3)), 100, voices[r.IntN(len(voices))])
	}
	now := at0
	for step := 0; step < 600; step++ {
		for i := range units {
			u := &units[i]
			if r.IntN(3) == 0 {
				u.HP += r.IntN(41) - 30
				u.Life = lifeAt(u.HP)
			}
			u.Cell = image.Pt(r.IntN(numeralGrid), r.IntN(numeralGrid))
			u.FinePosition, u.FineX, u.FineY = r.IntN(2) == 0, uint8(r.IntN(256)), uint8(r.IntN(256))
			u.CorpseStage = uint8(r.IntN(3))
			u.Name, u.Speed, u.Load = fmt.Sprint("unit", r.IntN(9)), r.IntN(9), r.IntN(9)
			u.Step = image.Pt(r.IntN(3)-1, r.IntN(3)-1)
		}
		switch r.IntN(12) {
		case 0:
			v.SetFog(make([]uint8, numeralGrid*numeralGrid), numeralGrid, numeralGrid)
		case 1:
			v.SetFog(nil, 0, 0)
		case 2:
			v.beginSoundEntry()
		}
		for tick := r.IntN(4); tick > 0; tick-- {
			var messages []DamageMessage
			for n := r.IntN(4); n > 0; n-- {
				u := units[r.IntN(len(units))]
				messages = append(messages, DamageMessage{Entity: u, AfterHP: u.HP - r.IntN(30)})
			}
			v.AppendDamageMessages(messages)
			shown := slices.Clone(units[:3+r.IntN(len(units)-2)])
			v.SetEntities(shown)
		}
		now = now.Add(time.Duration(r.IntN(1500)) * time.Millisecond)
		if r.IntN(2) == 0 {
			v.step(Input{}, now)
		} else {
			v.stepSound(now)
		}
	}
	h := sha256.New()
	for i, rec := range []*recordingPlayer{effects, speech} {
		fmt.Fprintln(h, i, len(rec.plays))
		for _, p := range rec.plays {
			fmt.Fprintln(h, p.Sample.PCM[0], p.Placement.Left, p.Placement.Right)
		}
	}
	ids := make([]uint32, 0, len(v.voices))
	for id := range v.voices {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		m := v.voices[id]
		fmt.Fprintln(h, id, m.at.UnixNano(), m.framed, m.hp, m.damaged, m.alive)
	}
	return fmt.Sprintf("%x %d %d", h.Sum(nil), len(effects.plays), len(speech.plays))
}

// The sound step queues only the drawable fields it reads. These digests are
// the plays and voice memories the queue of whole drawables produced for the
// same streams.
func TestSoundStreamMatchesTheWholeDrawableQueue(t *testing.T) {
	for seed, want := range map[uint64]string{
		1: "ae24c905868fbb9c009e92444f224c864484f5f64250e02190ecc576f628c9eb 13 111",
		2: "747aba26014da9acd79e9afe2f3b4428695829543e1b8f269a5b5e0214787590 21 96",
		3: "4b604d3b9bd87e8e736971d7b824949b702c6aa6c3da0e3fed62fba50a64efd2 11 114",
	} {
		if got := soundStreamDigest(t, seed); got != want {
			t.Errorf("seed %d: digest %s, want %s", seed, got, want)
		}
	}
}

// Every queued field is the drawable's field of the same name and type.
func TestSoundEntityCopiesEachFieldItNames(t *testing.T) {
	e := hurtUnit(7, 3, -12, "f_mage")
	e.MaxHP, e.Life, e.Cell = 91, LifeDead, image.Pt(5, 9)
	e.FinePosition, e.FineX, e.FineY = true, 17, 230
	got, want := reflect.ValueOf(soundEntityOf(e)), reflect.ValueOf(e)
	for i := 0; i < got.NumField(); i++ {
		name := got.Type().Field(i).Name
		field := want.FieldByName(name)
		if !field.IsValid() || field.Type() != got.Field(i).Type() {
			t.Fatalf("soundEntity.%s has no drawable field of its name and type", name)
		}
		if !reflect.DeepEqual(got.Field(i).Interface(), field.Interface()) {
			t.Errorf("soundEntity.%s = %v, drawable holds %v", name, got.Field(i), field)
		}
	}
}
