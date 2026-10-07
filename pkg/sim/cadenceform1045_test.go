package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func cadenceFormWorld(t *testing.T, cast bookCast) *World {
	t.Helper()
	caster := spMage(1, 2, 2, 20, 30, 30, 1)
	caster.Humanoid, caster.Reaction, caster.AttackRelax = true, 20, 4
	victim := spEnt(2, 4, 2)
	rule := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 8,
		DamageMin: 3, DamageMax: 7, TargetsUnit: true, Damaging: true,
		Complication: 9}
	w := spWorld(t, 0x1045, []SpellRule{rule}, caster, victim)
	w.bookCasts = []bookCast{cast}
	return w
}

func cadenceBookBytes(c bookCast) []byte {
	b := make([]byte, bookRecordLen)
	binary.LittleEndian.PutUint32(b[0:4], uint32(c.Caster))
	binary.LittleEndian.PutUint32(b[4:8], uint32(c.Target))
	binary.LittleEndian.PutUint16(b[8:10], c.Spell)
	binary.LittleEndian.PutUint32(b[10:14], uint32(c.X))
	binary.LittleEndian.PutUint32(b[14:18], uint32(c.Y))
	b[18] = c.Remaining
	if c.AtCell {
		b[19] = 1
	}
	b[20], b[21] = byte(c.Phase), c.Progress
	if c.Complete {
		b[22] = 1
	}
	if c.Retained {
		b[23] = 1
	}
	return b
}

func cadenceBookAt(t *testing.T, form []byte, c bookCast) int {
	t.Helper()
	at := bytes.Index(form, cadenceBookBytes(c))
	if at < 0 {
		t.Fatalf("book record %+v is absent from the form", c)
	}
	if bytes.Index(form[at+1:], cadenceBookBytes(c)) >= 0 {
		t.Fatalf("book record %+v is not unique in the form", c)
	}
	return at
}

func TestCadenceFormRoundTripsEveryLifecycleBoundary(t *testing.T) {
	states := []bookCast{
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookCharging, Remaining: 2, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookRelaxing, Remaining: 2, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookBoundaryOne, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookBoundaryTwo, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookPending, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookPending, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookPending, Progress: 1, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookPending, Progress: 2, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookPending, Progress: 3, Complete: true, Retained: true},
		{Caster: 1, Spell: 1, X: 7, Y: 9, Phase: bookCharging, Remaining: 2, AtCell: true, Retained: true},
	}
	for _, state := range states {
		w := cadenceFormWorld(t, state)
		form := mustMarshal(t, w)
		if form[0] != formatVersion {
			t.Fatalf("form version = %d, want current %d", form[0], formatVersion)
		}
		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatalf("state %+v: UnmarshalBinary: %v", state, err)
		}
		if len(back.bookCasts) != 1 || back.bookCasts[0] != state {
			t.Errorf("state %+v came back as %+v", state, back.bookCasts)
		}
		if !back.entities[0].Humanoid || back.spells[0].Complication != 9 {
			t.Errorf("state %+v lost Humanoid/Complication: entity=%+v spell=%+v",
				state, back.entities[0], back.spells[0])
		}
		if again := mustMarshal(t, &back); !bytes.Equal(again, form) || back.Hash() != w.Hash() {
			t.Errorf("state %+v changed bytes or hash across round trip", state)
		}
	}
}

func TestCadenceFormResumeHasTheSameNextTicks(t *testing.T) {
	states := []bookCast{
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookCharging, Remaining: 1, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookRelaxing, Remaining: 1, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookBoundaryOne, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookBoundaryTwo, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookPending, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2, Phase: bookPending, Progress: 3, Complete: true, Retained: true},
	}
	for _, state := range states {
		live := cadenceFormWorld(t, state)
		if state.Phase == bookPending {
			live.entities[0].Mana = 0
		}
		var resumed World
		if err := resumed.UnmarshalBinary(mustMarshal(t, live)); err != nil {
			t.Fatalf("state %+v: UnmarshalBinary: %v", state, err)
		}
		for tick := 0; tick < 24; tick++ {
			if tick == 6 && state.Phase == bookPending {
				live.entities[0].Mana, resumed.entities[0].Mana = 30, 30
			}
			wantEvents := StepObserved(live, nil)
			gotEvents := StepObserved(&resumed, nil)
			if !reflect.DeepEqual(gotEvents, wantEvents) {
				t.Fatalf("state %+v tick %d events = %+v, want %+v", state, tick, gotEvents, wantEvents)
			}
			if got, want := mustMarshal(t, &resumed), mustMarshal(t, live); !bytes.Equal(got, want) {
				t.Fatalf("state %+v tick %d diverged after resume", state, tick)
			}
		}
	}
}

func TestCadenceForm62RefusesEveryMalformedLifecycleClass(t *testing.T) {
	validCast := bookCast{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2,
		Phase: bookCharging, Remaining: 2, Retained: true}
	w := cadenceFormWorld(t, validCast)
	valid := mustMarshal(t, w)
	bookAt := cadenceBookAt(t, valid, validCast)
	entityAt := headerLen + 3*int(gridCells(w.bounds))
	script := len(valid) - 65 - entityIDFloorLen - spellDeliverySpanLen - relationLen - w.originalDeadSectionLen() - w.instanceWeightSectionLen() - w.actorLoadSectionLen() - w.scriptSectionLen()
	items := script - w.scrollSectionLen() - w.itemStateSectionLen()
	structures := items - w.structureSectionLen()
	stateAt := structures - w.scriptStateSectionLen()
	casting := stateAt - w.castingSectionLen()
	weights := casting - (itemWeightCountLen + itemWeightRecordLen*len(w.itemWeights))
	spellAt := weights - spellRecordLen*len(w.spells)

	mutateBook := func(off int, value byte) []byte { return withByte(valid, bookAt+off, value) }
	cases := []struct {
		name string
		form []byte
	}{
		{"Humanoid is not boolean", withByte(valid, entityAt+290, 2)},
		{"phase is outside the lifecycle", mutateBook(20, 5)},
		{"retry progress exceeds three", mutateBook(21, 4)},
		{"completion is not boolean", mutateBook(22, 2)},
		{"retention is not boolean", mutateBook(23, 2)},
		{"charging has no countdown", mutateBook(18, 0)},
		{"charging carries completion", mutateBook(22, 1)},
		{"charging carries retry progress", mutateBook(21, 1)},
		{"cell target carries a unit", mutateBook(19, 1)},
	}
	invalid := []struct {
		name string
		cast bookCast
	}{
		{"pending carries a countdown", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookPending, Remaining: 1, Retained: true}},
		{"incomplete pending carries progress", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookPending, Progress: 1, Retained: true}},
		{"incomplete pending is not retained", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookPending}},
		{"completed pending is not retained", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookPending, Complete: true}},
		{"relaxing has no countdown", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookRelaxing, Complete: true, Retained: true}},
		{"relaxing is incomplete", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookRelaxing, Remaining: 1, Retained: true}},
		{"relaxing is not retained", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookRelaxing, Remaining: 1, Complete: true}},
		{"relaxing carries progress", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookRelaxing, Remaining: 1, Progress: 1, Complete: true, Retained: true}},
		{"boundary carries a countdown", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookBoundaryOne, Remaining: 1, Complete: true, Retained: true}},
		{"boundary is incomplete", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookBoundaryTwo, Retained: true}},
		{"boundary is not retained", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookBoundaryOne, Complete: true}},
		{"boundary carries progress", bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookBoundaryTwo, Progress: 1, Complete: true, Retained: true}},
		{"cell target carries a unit id", bookCast{Caster: 1, Target: 2, Spell: 1, X: 7, Y: 9, Phase: bookCharging, Remaining: 1, AtCell: true, Retained: true}},
	}
	for _, tc := range invalid {
		cases = append(cases, struct {
			name string
			form []byte
		}{tc.name, mustMarshal(t, cadenceFormWorld(t, tc.cast))})
	}
	for _, tc := range cases {
		var back World
		if err := back.UnmarshalBinary(tc.form); err == nil {
			t.Errorf("%s was accepted", tc.name)
		}
	}
	// Complication is an unsigned byte with all 256 values canonical. Its top
	// value must read rather than be accidentally treated as a boolean.
	top := append([]byte(nil), valid...)
	top[spellAt+35] = 0xff
	var back World
	if err := back.UnmarshalBinary(top); err != nil || back.spells[0].Complication != 0xff {
		t.Errorf("Complication 255 did not round trip: value=%d err=%v", back.spells[0].Complication, err)
	}
}

func TestCadenceForm62NormalizesAndRefusesDeadCastRecoveryResidue(t *testing.T) {
	build := func(wait uint8) *World {
		dead := spEnt(1, 2, 2)
		dead.HP, dead.CastWait = -1, wait
		return spWorld(t, 1045, nil, dead)
	}
	clean, residue := build(0), build(9)
	if got := spAt(t, residue, 1).CastWait; got != 0 {
		t.Fatalf("constructor retained dead CastWait %d", got)
	}
	cleanForm, residueForm := mustMarshal(t, clean), mustMarshal(t, residue)
	if !bytes.Equal(cleanForm, residueForm) || clean.Hash() != residue.Hash() {
		t.Fatal("normalized dead CastWait changed form or hash")
	}

	entityAt := headerLen + 3*int(gridCells(clean.bounds))
	bad := withByte(cleanForm, entityAt+219, 9)
	var back World
	err := back.UnmarshalBinary(bad)
	if err == nil || !strings.Contains(err.Error(), "cast recovery") {
		t.Fatalf("dead CastWait form error = %v, want cast recovery refusal", err)
	}
}

func TestCadenceInputsAndLifecycleEachChangeTheCanonicalHash(t *testing.T) {
	baseCast := bookCast{Caster: 1, Target: 2, Spell: 1, X: 4, Y: 2,
		Phase: bookPending, Complete: true, Retained: true}
	base := cadenceFormWorld(t, baseCast)
	seen := map[uint64]string{base.Hash(): "base"}
	check := func(name string, change func(*World)) {
		w := cadenceFormWorld(t, baseCast)
		change(w)
		if previous, ok := seen[w.Hash()]; ok {
			t.Errorf("%s hashes like %s at %#x", name, previous, w.Hash())
		}
		seen[w.Hash()] = name
	}
	check("Humanoid", func(w *World) { w.entities[0].Humanoid = false })
	check("Complication", func(w *World) { w.spells[0].Complication++ })
	check("phase", func(w *World) { w.bookCasts[0].Phase = bookBoundaryOne })
	check("progress", func(w *World) { w.bookCasts[0].Progress = 1 })
	check("target form", func(w *World) { w.bookCasts[0].Target, w.bookCasts[0].AtCell = 0, true })
}
