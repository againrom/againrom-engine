package town

import (
	"image"
	"testing"
	"time"
)

// A room program runs on the square and a square program in a room page.
func TestEveryProgramRunsOnTheSquareAndInARoomPage(t *testing.T) {
	d := baseTown()
	d["art"] = append(d["art"].(list), obj{"name": "lp", "format": "series", "key": "l%d.bmp", "count": 2})
	d["actors"] = append(d["actors"].(list), obj{"name": "lp", "program": "loop", "art": "lp", "frames": 2, "loop": 2})
	d["step"] = obj{"admitted": list{"advance spin", "publish lp", "advance lp"}}
	d["layers"] = append(d["layers"].(list), obj{"actor": "lp", "mode": "copy", "at": list{30, 20}})
	d["rooms"].(list)[0].(obj)["scene"] = obj{
		"random": list{obj{"name": "dice", "source": "host"}},
		"art":    list{series("e", 3)},
		"sounds": obj{"slots": list{obj{"name": "x", "source": "inn"}}},
		"clocks": list{obj{"name": "c", "period-ms": 10, "compare": "greater"}},
		"actors": list{obj{"name": "e", "program": "episode", "art": "e",
			"trigger":     obj{"draw": "dice", "n": 100, "above": 50},
			"start-sound": obj{"slot": "x", "key": "e.wav"}}},
		"steps":  list{obj{"clock": "c", "run": list{"trigger e", "advance e"}}},
		"layers": list{layer("g", "e")},
	}
	desc := mustDecode(t, d)

	l := testLoader()
	l.sizes["l0.bmp"], l.sizes["l1.bmp"] = image.Pt(2, 2), image.Pt(2, 2)
	art, err := LoadArt(desc.SquareScene(), l)
	if err != nil {
		t.Fatal(err)
	}
	sh := &fakeHost{art: art, now: time.Unix(100, 0), conditions: map[string]bool{}}
	square := NewScene(desc, desc.Square.Name, sh, nil)
	square.SetActive(true, true)
	lp := square.Actor("lp").(*Loop)
	square.Advance() // stamps the square's clock
	for i := 0; i < 2; i++ {
		sh.tick(51 * time.Millisecond)
		square.Advance()
	}
	if lp.Shown != 1 || lp.Index != 0 {
		t.Fatalf("a loop on the square = %+v, want frame 1 shown and the index wrapped", *lp)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 40, 30))
	square.Paint(dst, "")
	if got := dst.RGBAAt(30, 20); got != keyColour("l1.bmp") {
		t.Fatalf("the square painted %v at the loop's point, want frame 1", got)
	}

	rh := &fakeHost{art: &Art{Frames: map[string][]image.Image{"e": pageFrames(8, 3)}},
		now: time.Unix(100, 0), draws: []int{99}}
	room := NewScene(desc, "inn", rh, nil)
	room.Enter()
	ep := room.Actor("e").(*Episode)
	rh.tick(11 * time.Millisecond)
	room.Advance()
	if !ep.Enabled || ep.Frame != 1 || rh.take() != "play inn:e.wav" {
		t.Fatalf("an episode in a room page = %+v, want armed by its trigger and on frame 1 with its cue", *ep)
	}
	page := image.NewRGBA(image.Rect(0, 0, 4, 4))
	room.Paint(page, "g")
	if got := page.RGBAAt(0, 0); got.R != 8 || got.G != 2 {
		t.Fatalf("the room page painted %v, want the episode's frame 1", got)
	}
}
