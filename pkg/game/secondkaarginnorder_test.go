package game

import (
	"fmt"
	"image"
	"testing"
	"time"

	"againrom/pkg/town"
)

// A selected hire seat of the Kaarg inn draws its sprite and then advances
// its cursor; a selected talk seat advances and then draws (R2-ENGINE-686,
// DIV-4272). Per paint: the frame the seat draws and its cursor after it.
func TestKaargInnSeatDrawOrder(t *testing.T) {
	f, app, screen := secondCampaignFixtureFiles(t, false)
	c := innAt(2, 30, map[int]int32{532: 1, 552: 2})
	c.room = secondTownInn
	f.Town.second = c
	app.SetTown(screen)
	d, ok := screen.composedTown()
	if !ok || d.Town != "rom2-town2" {
		t.Fatal("no composed town 2")
	}
	list, hires := c.innSeats()
	if hires < 1 || len(list) <= hires {
		t.Fatalf("the fixture inn has %d hire seats of %d", hires, len(list))
	}
	h := &kaargInnHost{now: time.Unix(100, 0)}
	f.TownAnimationNow = func() time.Time { return h.now }
	seats := d.Room("inn").Seats
	pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
	frames := []image.Image{pic, pic, pic}
	run := func(sel int) (drawn, cursor, other []int) {
		s := kaargInn(t, h, &town.Process{})
		screen.enterSeats()
		screen.composed.seat = sel
		s.Advance()
		screen.advanceSeats(s, seats, frames)
		for i := 0; i < 12; i++ {
			h.now = h.now.Add(126 * time.Millisecond)
			s.Advance()
			screen.advanceSeats(s, seats, frames)
			drawn = append(drawn, screen.seatFrame(sel))
			cursor = append(cursor, screen.composed.seatFrames[sel])
			other = append(other, screen.seatFrame(1-min(sel, 1)))
		}
		return
	}
	hd, hc, ho := run(0)
	td, tc, to := run(hires)
	got := fmt.Sprint(hd, hc, ho, td, tc, to)
	want := fmt.Sprint(
		[]int{0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2},
		[]int{1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0},
		[]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		[]int{1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0},
		[]int{1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0},
		[]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	if got != want {
		t.Fatalf("hire seat drawn/cursor/unselected, talk seat drawn/cursor/unselected:\n got %s\nwant %s", got, want)
	}
	for name, desc := range townDescriptions {
		for _, r := range desc.Rooms {
			if r.Seats != nil && r.Seats.HireOrder != "" && name != "rom2-town2" {
				t.Errorf("%s room %s names a hire order %q", name, r.Name, r.Seats.HireOrder)
			}
		}
	}
}
