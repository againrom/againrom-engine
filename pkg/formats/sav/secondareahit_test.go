package sav

import "testing"

func TestSecondGameUnitCommandPreservesAreaHitCounter(t *testing.T) {
	doc := attackDocument(t, 11, 0x30)
	keeps := ProfileKeeps{
		Commands: map[uint32]CommandStrike{
			0x20: {Stamp: commandWord, AreaHits: 7},
		},
	}
	out, changed, err := DocumentInProfile(doc, SecondGame, keeps)
	if err != nil || !changed {
		t.Fatalf("DocumentInProfile: changed=%v err=%v", changed, err)
	}
	command := documentRaw(t, out.Objects[1], "URaw184")
	got, ok := readCommandStrike(command)
	if !ok {
		t.Fatal("readCommandStrike returned false on URaw184")
	}
	if got.AreaHits != 7 || got.Stamp != commandWord {
		t.Fatalf("readCommandStrike = %+v, want Stamp=%d AreaHits=7", got, commandWord)
	}
	if int32(u32(command, 0xac)) != 7 {
		t.Fatalf("URaw184 +0xac = %d, want 7", int32(u32(command, 0xac)))
	}
}
