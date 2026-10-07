package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/mapload"
)

// Repeating message triggers of the shipped maps: RU mission 140 latch 5, EN none.
func TestReleaseAnnouncerRepeatingTriggersOfShippedMissions(t *testing.T) {
	f := releaseFront(t)
	defs, err := LoadDefinitions(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	var got []string
	maps := 0
	for number := 10; number <= 290; number++ {
		ms, err := StartMission(f.Archives.Containers, number, defs.Table, mapload.DifficultyNormal, party)
		if err != nil || ms.World.Script() == nil {
			continue
		}
		maps++
		a := NewAnnouncer(ms.World, ms.Raises)
		for _, latch := range a.order {
			if a.repeating[latch] {
				got = append(got, fmt.Sprintf("mission %d latch %d", number, latch))
			}
		}
		for _, tr := range ms.World.Script().Triggers() {
			if want := !tr.Once && !tr.Inert; a.repeating[tr.Latch] != want {
				t.Fatalf("mission %d latch %d: repeating %v, want %v", number, tr.Latch, a.repeating[tr.Latch], want)
			}
		}
	}
	var want []string
	if LanguageSelector(f.Archives.Containers) == 1 {
		want = []string{"mission 140 latch 5"}
	}
	if maps < 28 || !slices.Equal(got, want) {
		t.Fatalf("%d scripted maps, repeating message triggers %v, want %v", maps, got, want)
	}
}
