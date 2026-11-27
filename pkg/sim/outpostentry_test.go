package sim

import (
	"reflect"
	"testing"
)

func TestDeclareOutpostsFollowsTheEntryValues(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Entries: []OutpostEntry{{Group: 7, Percent: 5}}})
	if len(w.Outposts()) != 0 {
		t.Fatalf("map values with a shut gate declared %+v", w.Outposts())
	}
	w.DeclareOutposts(OutpostMap{IgnoresGate: true, AnyOwner: true, KeepsID: true, ProfileReturn: true,
		Entries: []OutpostEntry{{Group: 7, Percent: 5}}})
	o := outpostOf(t, w, 2, 7)
	if len(w.Outposts()) != 3 || o.Reload != sessionReturnPasses || !o.KeepsID {
		t.Fatalf("entry values declared %+v", w.Outposts())
	}
	w.DeclareOutposts(OutpostMap{Runs: true, Entries: []OutpostEntry{{Group: 7, Percent: 5}}})
	if o := outpostOf(t, w, 2, 7); o.Reload != 5 || o.KeepsID {
		t.Fatalf("campaign values declared %+v", o)
	}
}

func TestOutpostKeepsIDSurvivesTheByteForm(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{IgnoresGate: true, AnyOwner: true, KeepsID: true, ProfileReturn: true})
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if !outpostOf(t, &back, 2, 7).KeepsID || !reflect.DeepEqual(back.Outposts(), w.Outposts()) || back.Hash() != w.Hash() {
		t.Fatal("the id rule did not survive the byte form")
	}
}
