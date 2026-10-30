package game

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/sav"
)

// deadLoadedValue names a scalar of a dead record and the value the test
// writes into the loaded file.
type deadLoadedValue struct {
	name  string
	value uint32
}

var deadLoadedValues = []deadLoadedValue{{"U12C", 7}, {"U8E", 41}, {"UA3", 70}, {"U90", 22}}

// deadLoadedBlocks are the raw blocks the test fills in the loaded file; byte
// 8 of the mover block is its search radius. The five skill level words of a
// Human's attack block stay within the original cap, which SAVE applies.
var deadLoadedBlocks = []string{"U154", "U158", "UA6", "UBE", "UD4"}

// deadLoadedBlock is the pattern the test writes into block name.
func deadLoadedBlock(name string, width int) []byte {
	out := make([]byte, width)
	for i := range out {
		out[i] = byte(len(name)*17 + i*3 + 1)
	}
	switch name {
	case "U154":
		out[8] = 9
	case "UA6":
		for level := range 5 {
			out[4+2*level], out[5+2*level] = byte(11*(level+1)), 0
		}
	}
	return out
}

// deadRecordFixture writes an original-shaped SAV with one living Group
// member and one Human on the dead list with no map unit, so the World holds
// it as no entity. edit fills the dead record before the file is read.
func deadRecordFixture(t *testing.T, edit func(*sav.DocumentRecordData)) []byte {
	t.Helper()
	runtime := uint32(71)
	body := &poolFixtureActor{cell: 0x0807, hp: uint16(65536 - 14), maxHP: 30, stage: 2, human: true, runtime: &runtime}
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 30, name: "Living binding"}
	f := currentRetainedRuntimeFront(t)
	raw := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, []*poolFixtureActor{body}))
	raw = completeCurrentDeadDocument(t, f, raw)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	edit(&doc.Objects[doc.DeadActors[0]-1])
	if raw, err = sav.EncodeDocumentData(doc); err != nil {
		t.Fatal(err)
	}
	return raw
}

// writtenDeadRecord is the one dead record of a SAV the engine wrote.
func writtenDeadRecord(t *testing.T, raw []byte) sav.DocumentRecordData {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.DeadActors) != 1 {
		t.Fatalf("%d dead records, want 1", len(doc.DeadActors))
	}
	return doc.Objects[doc.DeadActors[0]-1]
}

// A dead record whose body the World holds as no entity is written with the
// actor values of its loaded record: the World carries none of them, and the
// loaded bytes rank above a constructed value. The World's dead tuple still
// wins after the body decays. A SAVE, a cold LOAD and a second SAVE keep them.
func TestDeadRecordKeepsItsLoadedActorValues(t *testing.T) {
	raw := deadRecordFixture(t, func(r *sav.DocumentRecordData) {
		for _, v := range deadLoadedValues {
			mustSetValue(r, v.name, v.value)
		}
		for _, name := range deadLoadedBlocks {
			mustSetRaw(r, name, deadLoadedBlock(name, len(mustRawField(t, r, name))))
		}
	})
	f := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
	for cycle := 0; cycle < 2; cycle++ {
		w := f.live.world
		body := w.OriginalDeadActors()[0]
		if _, entity := w.Entity(body.ID); entity {
			t.Fatal("the fixture body is a World entity")
		}
		for range 40 {
			f.live.tick()
		}
		body = w.OriginalDeadActors()[0]
		raw = currentRuntimeSave(t, f)
		r := writtenDeadRecord(t, raw)
		for _, v := range deadLoadedValues {
			if got, err := savedStructureValue(&r, v.name); err != nil || got != v.value {
				t.Fatalf("cycle %d: %s %d (%v), want the loaded %d", cycle, v.name, got, err, v.value)
			}
		}
		for _, name := range deadLoadedBlocks {
			got := mustRawField(t, &r, name)
			if want := deadLoadedBlock(name, len(got)); !bytes.Equal(got, want) {
				t.Fatalf("cycle %d: %s %x, want the loaded %x", cycle, name, got, want)
			}
		}
		stage, _ := savedStructureValue(&r, "Stage")
		health, _ := savedStructureValue(&r, "Health")
		if stage != uint32(body.Current.Stage) || health != uint32(uint16(body.Current.HP)) {
			t.Fatalf("cycle %d: dead tuple %d/%d, want the World's %d/%d", cycle, stage, health, body.Current.Stage, body.Current.HP)
		}
		f = openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
	}
}

// mustRawField is a copy of raw block name of r.
func mustRawField(t *testing.T, r *sav.DocumentRecordData, name string) []byte {
	t.Helper()
	for _, field := range r.Raw {
		if field.Name == name {
			return bytes.Clone(field.Bytes)
		}
	}
	t.Fatalf("%s record lacks %s", r.Class, name)
	return nil
}
