package game

import (
	"testing"
	"time"

	"againrom/pkg/random"
	"againrom/pkg/town"
)

// sharedStateBefore answers a shared-stream state whose next rand() is want.
func sharedStateBefore(want int32) uint32 {
	for x := uint32(0); ; x++ {
		m := random.MSVC{State: x}
		if m.Rand() == want {
			return x
		}
	}
}

// Under the original generator a raw pick reads rand() itself: at rand() =
// 32767 the first game's tavern delay is 3000+32767/16 = 5047 ms (TOWN-409)
// and the Kaarg taverner's value ((32767*21)>>15)+1 = 21 starts a4
// (R2-ENGINE-277).
func TestTownRawPickReadsRandUnderTheOriginalGenerator(t *testing.T) {
	top := random.Session{Mode: random.Original, Shared: sharedStateBefore(random.RandMax)}

	s := bareTownScreen(t)
	rt := &RuntimeServices{Random: random.NewService(random.Session{})}
	s.bindServices(rt, s.in, &Presentation{})
	tavern := s.scene(s.townDescription(), "tavern")
	rt.Random.Begin(top)
	tavern.Enter()
	if got := tavern.Actor("tender").(*town.Alternating).Delay; got != 5047*time.Millisecond {
		t.Errorf("the tavern delay at rand() 32767 is %v, want 5047ms", got)
	}

	f, app, screen := secondCampaignFixtureFiles(t, false)
	c := innAt(2, 30, map[int]int32{532: 1, 552: 2})
	c.room = secondTownInn
	f.Town.second = c
	app.SetTown(screen)
	d, ok := screen.composedTown()
	if !ok || d.Town != "rom2-town2" {
		t.Fatal("no composed town 2")
	}
	screen.composed.art[d.Town+"/inn"] = &townRoomArt{scene: kaargInnArt()}
	inn := screen.scene(d, "inn")
	inn.SetActive(true, true)
	f.randomService().Begin(top)
	inn.Advance()
	if a := inn.Actor("taverner").(*town.Alternating); a.State != 4 {
		t.Errorf("the taverner at rand() 32767 runs state %d, want a4 (state 4)", a.State)
	}
}

// serviceTownDraws is a townDraws with no seam over a random service.
type serviceTownDraws struct {
	fakeTownDraws
	svc *random.Service
}

func (d serviceTownDraws) stream(name random.Name) *random.Stream { return d.svc.Stream(name) }

// A host source's raw value equals the bounded draw of 32768 a raw pick took
// before in a seeded session, draw for draw; under the original generator it
// differs only where rand() is 32767, which that draw read as 0.
func TestTownRawPickSeededDrawsAsBefore(t *testing.T) {
	const run = 400000
	for _, mode := range []random.Mode{random.Seeded, random.Original} {
		for source, bind := range sceneDraws {
			session := random.Session{Seed: 1, Mode: mode, Shared: 1}
			draws, twin := serviceTownDraws{svc: random.NewService(session)}, random.NewService(session)
			top := 0
			for i := 0; i < run; i++ {
				got, before := sceneRawFrom(draws, source), boundedPresentationRoll(nil, twin.Stream(bind.stream), random.RandMax+1)
				switch {
				case got == before:
				case mode == random.Original && got == random.RandMax && before == 0:
					top++
				default:
					t.Fatalf("mode %d source %s draw %d: raw %d, the bounded draw %d", mode, source, i, got, before)
				}
			}
			if (mode == random.Seeded) != (top == 0) {
				t.Errorf("mode %d source %s: %d draws of 32767 in %d", mode, source, top, run)
			}
		}
	}
}
