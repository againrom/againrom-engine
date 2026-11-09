package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// The one decision over a gesture's worn slots, for both mixed-gesture rules.
func TestDecideDollChange(t *testing.T) {
	a, b := sim.PlainItem(0x0e05), sim.PlainItem(0x0e03)
	var none, pairAB, onlyA, onlyB, aInSlot2 [sim.EquipSlots]sim.ItemInstance
	pairAB[0], pairAB[1] = a, b
	onlyA[0] = a
	onlyB[0] = b
	aInSlot2[1] = a
	cases := []struct {
		name          string
		before, after [sim.EquipSlots]sim.ItemInstance
		anyGain, net  dollChange
	}{
		{"no change", onlyA, onlyA, dollUnchanged, dollUnchanged},
		{"both empty", none, none, dollUnchanged, dollUnchanged},
		{"put on", none, onlyA, dollPutOn, dollPutOn},
		{"take off", onlyA, none, dollTakeOff, dollTakeOff},
		{"replace in a slot", onlyA, onlyB, dollPutOn, dollPutOn},
		{"move to another slot", onlyA, aInSlot2, dollPutOn, dollPutOn},
		{"add a second", onlyA, pairAB, dollPutOn, dollPutOn},
		{"take one of two off", pairAB, onlyA, dollTakeOff, dollTakeOff},
		{"take one off and replace the other", pairAB, [sim.EquipSlots]sim.ItemInstance{b}, dollPutOn, dollTakeOff},
	}
	for _, c := range cases {
		if got := decideDollChange(c.before, c.after, dollAnyGain); got != c.anyGain {
			t.Errorf("%s: any-gain rule %d, want %d", c.name, got, c.anyGain)
		}
		if got := decideDollChange(c.before, c.after, dollNetCount); got != c.net {
			t.Errorf("%s: net-count rule %d, want %d", c.name, got, c.net)
		}
	}
}

// Each surface reaches the source its profile states: the shop a room event,
// the second game's mission a named member, the first game's mission none.
func TestDollSoundProfileSources(t *testing.T) {
	var events, members []string
	ev := func(s string) { events = append(events, s) }
	mem := func(s string) { members = append(members, s) }
	shopDollSounds.play(dollPutOn, ev, mem)
	shopDollSounds.play(dollTakeOff, ev, mem)
	if !reflect.DeepEqual(events, []string{"doll-on", "doll-off"}) || len(members) != 0 {
		t.Errorf("shop: events %v members %v", events, members)
	}
	events, members = nil, nil
	p := secondGameFiles.dollSounds
	p.play(dollPutOn, ev, mem)
	p.play(dollTakeOff, ev, mem)
	p.play(dollUnchanged, ev, mem)
	if len(events) != 0 || !reflect.DeepEqual(members, []string{"put_on.wav", "put_off.wav"}) {
		t.Errorf("second mission: events %v members %v", events, members)
	}
	events, members = nil, nil
	firstGameFiles.dollSounds.play(dollPutOn, ev, mem)
	firstGameFiles.dollSounds.play(dollTakeOff, ev, mem)
	if len(events) != 0 || len(members) != 0 {
		t.Errorf("first mission: events %v members %v", events, members)
	}
}
