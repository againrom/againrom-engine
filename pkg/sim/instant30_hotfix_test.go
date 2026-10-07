package sim

import (
	"bytes"
	"strconv"
	"testing"
)

func instant30Effect(target EntityID, spell, remaining uint16) attachedEffect {
	return attachedEffect{
		Target: target, Caster: 2, HasCaster: true, Spell: spell,
		Kind: EffectBless, Mode: EffectDuration, Magnitude: 9, Remaining: remaining,
	}
}

func instant30World(t *testing.T) *World {
	t.Helper()
	w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5), engFighter(2, 1, 6, 5))
	w.attached = []attachedEffect{
		instant30Effect(1, 20, 111),
		instant30Effect(1, 21, 222),
		instant30Effect(1, 276, 333),
		instant30Effect(2, 20, 444),
	}
	return w
}

func TestInstantThirtyWritesEveryCanonicalMatchAndNothingElse(t *testing.T) {
	t.Parallel()

	w := instant30World(t)
	want := instant30World(t)
	want.attached[0].Remaining = 60000
	want.attached[2].Remaining = 60000
	beforeHash := w.Hash()

	laNode(w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{276, 60000}})

	if got, expected := laForm(t, w), laForm(t, want); !bytes.Equal(got, expected) {
		t.Fatal("instant 30 changed state outside the two target-and-byte-spell Remaining fields")
	}
	if w.Hash() != want.Hash() {
		t.Errorf("retimed world hashes %#016x, independently expected world %#016x", w.Hash(), want.Hash())
	}
	if w.Hash() == beforeHash {
		t.Error("two canonical Remaining writes did not move the world hash")
	}
	got := w.ActiveEffects()
	if len(got) != 4 || got[0].Remaining != 60000 || got[1].Remaining != 222 ||
		got[2].Remaining != 60000 || got[3].Remaining != 444 {
		t.Errorf("canonical effects after instant 30 are %+v", got)
	}
	if e := laEnt(t, w, 1); e.SpellFX != 0 || e.SpellFXSpell != 0 {
		t.Errorf("instant 30 created presentation state %d/%d", e.SpellFX, e.SpellFXSpell)
	}
}

func TestInstantThirtyRunsThroughNewScriptAndStepTraced(t *testing.T) {
	t.Parallel()

	for _, duration := range []int32{60000, 1} {
		duration := duration
		t.Run(strconv.FormatInt(int64(duration), 10), func(t *testing.T) {
			t.Parallel()
			base := instant30World(t)
			program, err := NewScript(nil, []ScriptInstant{{
				Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{276, duration},
			}}, []ScriptTrigger{{
				Instants: [4]int32{0, ScriptNone, ScriptNone, ScriptNone}, Once: true,
			}})
			if err != nil {
				t.Fatalf("NewScript: %v", err)
			}
			w, err := NewControlledScriptWorld(base, program)
			if err != nil {
				t.Fatalf("NewControlledScriptWorld: %v", err)
			}
			trace := sttPass(w)
			if !trace.Pass || len(trace.Firings) != 1 || len(trace.Firings[0].Instants) != 1 {
				t.Fatalf("StepTraced returned %+v, want one instant on one firing", trace)
			}
			run := trace.Firings[0].Instants[0]
			if run.Op != ScriptInstantUnitEffectAge || run.Outcome != ScriptInstantStateChanged {
				t.Errorf("trace run is %+v, want state-changing instant 30", run)
			}
			effects := w.ActiveEffects()
			if effects[0].Remaining != uint16(duration) || effects[2].Remaining != uint16(duration) {
				t.Errorf("duration %d left matching records at %d and %d", duration,
					effects[0].Remaining, effects[2].Remaining)
			}
		})
	}
}

func TestInstantThirtyZeroDurationRoundTripsBeforeNextEffectPass(t *testing.T) {
	t.Parallel()

	base := instant30World(t)
	program, err := NewScript(nil, []ScriptInstant{{
		Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{276, 0},
	}}, []ScriptTrigger{{
		Instants: [4]int32{0, ScriptNone, ScriptNone, ScriptNone}, Once: true,
	}})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := NewControlledScriptWorld(base, program)
	if err != nil {
		t.Fatalf("NewControlledScriptWorld: %v", err)
	}
	trace := sttPass(w)
	if !trace.Pass || len(trace.Firings) != 1 || len(trace.Firings[0].Instants) != 1 {
		t.Fatalf("StepTraced returned %+v, want one instant on one firing", trace)
	}
	run := trace.Firings[0].Instants[0]
	if run.Op != ScriptInstantUnitEffectAge || run.Outcome != ScriptInstantStateChanged {
		t.Fatalf("trace run is %+v, want state-changing instant 30", run)
	}
	effects := w.ActiveEffects()
	if len(effects) != 4 || effects[0].Remaining != 0 || effects[2].Remaining != 0 {
		t.Fatalf("zero duration did not reach every target-one low-byte alias: %+v", effects)
	}

	form, savedHash := laForm(t, w), w.Hash()
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary of the transient zero-duration state: %v", err)
	}
	if got := laForm(t, &loaded); !bytes.Equal(got, form) {
		t.Error("zero-duration state changed across MarshalBinary and UnmarshalBinary")
	}
	if loaded.Hash() != savedHash {
		t.Errorf("loaded hash %#016x, saved hash %#016x", loaded.Hash(), savedHash)
	}
	loadedEffects := loaded.ActiveEffects()
	if len(loadedEffects) != 4 || loadedEffects[0].Remaining != 0 || loadedEffects[2].Remaining != 0 {
		t.Fatalf("loaded effects lost a zero-duration low-byte alias: %+v", loadedEffects)
	}

	Step(w, nil)
	Step(&loaded, nil)
	if got := loaded.ActiveEffects(); len(got) != 2 || got[0].Target != 1 || got[0].Spell != 21 ||
		got[1].Target != 2 || got[1].Spell != 20 {
		t.Fatalf("next ordinary effect pass left %+v, want only the two nonmatches", got)
	}
	if got, want := laForm(t, &loaded), laForm(t, w); !bytes.Equal(got, want) {
		t.Error("loaded world and source world diverged when the next effect pass removed both aliases")
	}
	if loaded.Hash() != w.Hash() {
		t.Errorf("post-removal hashes differ: loaded=%#016x source=%#016x", loaded.Hash(), w.Hash())
	}
}

func TestAttachedEffectDecoderAllowsZeroDurationAndRejectsEmptyIdentity(t *testing.T) {
	t.Parallel()

	w := instant30World(t)
	w.attached = []attachedEffect{instant30Effect(1, 20, 0)}
	data := make([]byte, w.castingSectionLen())
	if used := w.encodeCasting(data, 0); used != len(data) {
		t.Fatalf("encodeCasting used %d bytes of %d", used, len(data))
	}
	_, _, _, attached, used, err := decodeCasting(data)
	if err != nil {
		t.Fatalf("decodeCasting rejected zero Remaining: %v", err)
	}
	if used != len(data) || len(attached) != 1 || attached[0].Remaining != 0 {
		t.Fatalf("decoded %d bytes and effects %+v", used, attached)
	}

	// The first attached payload follows the two counts and its tag. Spell and
	// kind remain required even though Remaining zero is now meaningful.
	record := 2*castingCountLen + 1
	for _, tc := range []struct {
		name string
		zero func([]byte)
	}{
		{"spell", func(b []byte) { b[record+9], b[record+10] = 0, 0 }},
		{"kind", func(b []byte) { b[record+11] = byte(EffectNone) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			malformed := append([]byte(nil), data...)
			tc.zero(malformed)
			if _, _, _, _, _, err := decodeCasting(malformed); err == nil {
				t.Fatalf("decodeCasting accepted an attached effect with empty %s", tc.name)
			}
		})
	}
}

func TestInstantThirtyNoOpPopulationIsByteIdentical(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		build func(*testing.T) (*World, ScriptInstant)
	}{
		{"empty attached list", func(t *testing.T) (*World, ScriptInstant) {
			return engWorld(t, engRel(t), engFighter(1, 1, 5, 5)), ScriptInstant{
				Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{7, 60000},
			}
		}},
		{"absent compiled reference", func(t *testing.T) (*World, ScriptInstant) {
			w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5), engFighter(2, 1, 6, 5))
			w.attached = []attachedEffect{instant30Effect(1, 7, 90)}
			return w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1,
				Args: [scriptParams]int32{7, 60000}}
		}},
		{"unit no longer exists", func(t *testing.T) (*World, ScriptInstant) {
			w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5), engFighter(2, 1, 6, 5))
			w.attached = []attachedEffect{instant30Effect(1, 7, 90)}
			return w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 99, HasUnit: true,
				Args: [scriptParams]int32{7, 60000}}
		}},
		{"nonmatching spell", func(t *testing.T) (*World, ScriptInstant) {
			w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5), engFighter(2, 1, 6, 5))
			w.attached = []attachedEffect{instant30Effect(1, 8, 90)}
			return w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{7, 60000}}
		}},
		{"matching spell on a second target", func(t *testing.T) (*World, ScriptInstant) {
			w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5), engFighter(2, 1, 6, 5))
			w.attached = []attachedEffect{instant30Effect(2, 7, 90)}
			return w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{7, 60000}}
		}},
		{"stray presentation mark", func(t *testing.T) (*World, ScriptInstant) {
			w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5))
			w.entities[0].SpellFX, w.entities[0].SpellFXSpell = 90, 7
			return w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{7, 60000}}
		}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w, in := tc.build(t)
			before, beforeHash := laForm(t, w), w.Hash()
			laNode(w, in)
			if after := laForm(t, w); !bytes.Equal(before, after) {
				t.Error("no-match instant 30 changed the byte form")
			}
			if w.Hash() != beforeHash {
				t.Errorf("no-match instant 30 changed hash %#016x to %#016x", beforeHash, w.Hash())
			}
		})
	}
}

func TestInstantThirtyUsesByteSpellAndWordDuration(t *testing.T) {
	t.Parallel()

	for _, duration := range []int32{0, 1, 255, 256, 60000} {
		duration := duration
		t.Run(strconv.FormatInt(int64(duration), 10), func(t *testing.T) {
			t.Parallel()
			w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5), engFighter(2, 1, 6, 5))
			w.attached = []attachedEffect{instant30Effect(1, 263, 90)}
			laNode(w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{7, duration}})
			if got := w.attached[0].Remaining; got != uint16(duration) {
				t.Errorf("duration store is %d, want %d", got, duration)
			}
			if e := laEnt(t, w, 1); e.SpellFX != 0 || e.SpellFXSpell != 0 {
				t.Errorf("canonical match changed absent presentation state to %d/%d", e.SpellFX, e.SpellFXSpell)
			}
			if duration == 0 {
				Step(w, nil)
				if len(w.attached) != 0 {
					t.Errorf("zero-duration record survived the next ordinary effect step: %+v", w.attached)
				}
			}
		})
	}

	for _, tc := range []struct {
		name   string
		stored uint16
		author int32
		match  bool
	}{
		{"zero matches 256", 256, 0, true},
		{"one matches 257", 257, 1, true},
		{"255 matches 255", 255, 255, true},
		{"256 matches 256", 256, 256, true},
		{"one does not match 256", 1, 256, false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5), engFighter(2, 1, 6, 5))
			w.attached = []attachedEffect{instant30Effect(1, tc.stored, 90)}
			laNode(w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{tc.author, 60000}})
			want := uint16(90)
			if tc.match {
				want = 60000
			}
			if got := w.attached[0].Remaining; got != want {
				t.Errorf("stored spell %d and authored spell %d left %d, want %d",
					tc.stored, tc.author, got, want)
			}
		})
	}
}

func TestInstantThirtyDurationPersistsHashesAndTicksFromAuthoredValue(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 1, 5, 5), engFighter(2, 1, 6, 5))
	w.attached = []attachedEffect{instant30Effect(1, 20, 400)}
	beforeHash := w.Hash()
	laNode(w, ScriptInstant{Op: ScriptInstantUnitEffectAge, Unit: 1, HasUnit: true,
		Args: [scriptParams]int32{20, 256}})
	if w.Hash() == beforeHash {
		t.Error("canonical duration 400 to 256 did not move the hash")
	}
	if e := laEnt(t, w, 1); e.SpellFX != 0 || e.SpellFXSpell != 0 {
		t.Fatalf("instant 30 used presentation state: %d/%d", e.SpellFX, e.SpellFXSpell)
	}

	form := laForm(t, w)
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got := laForm(t, &loaded); !bytes.Equal(got, form) {
		t.Error("loaded instant-30 duration does not reproduce its save form")
	}
	if loaded.Hash() != w.Hash() {
		t.Errorf("loaded hash %#016x, saved world %#016x", loaded.Hash(), w.Hash())
	}
	if got := loaded.ActiveEffects(); len(got) != 1 || got[0].Remaining != 256 {
		t.Fatalf("loaded attached effects are %+v, want one at 256", got)
	}

	Step(w, nil)
	Step(&loaded, nil)
	if got := w.ActiveEffects(); len(got) != 1 || got[0].Remaining != 255 {
		t.Fatalf("next ordinary effect step left %+v, want one at 255", got)
	}
	if e := laEnt(t, w, 1); e.SpellFX != 255 || e.SpellFXSpell != 20 {
		t.Errorf("derived presentation mark is %d/%d, want 255/20", e.SpellFX, e.SpellFXSpell)
	}
	if got, want := laForm(t, &loaded), laForm(t, w); !bytes.Equal(got, want) {
		t.Error("loaded world and source world diverged on the next effect step")
	}
	if loaded.Hash() != w.Hash() {
		t.Errorf("next-step hashes differ: loaded=%#016x source=%#016x", loaded.Hash(), w.Hash())
	}
}
