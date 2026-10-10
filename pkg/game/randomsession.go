package game

import (
	"time"

	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// clockSessionSeed is the one clock read that chooses a session seed. Every
// stream then derives from the seed, so a session replays from it.
func clockSessionSeed() uint64 { return uint64(time.Now().UnixNano()) }

// SetRandomLaunch applies the launch settings: a fixed session seed when
// fixed, and the original generator when original. It begins the process's
// session; a new game then begins the next one.
func (f *FrontEnd) SetRandomLaunch(seed uint64, fixed, original bool) {
	mode := random.Seeded
	if original {
		mode = random.Original
	}
	ui.SetOriginalItemStars(original)
	f.randomService().SetLaunch(random.Launch{Seed: seed, Fixed: fixed, Mode: mode}, clockSessionSeed())
}

// randomService answers the runtime's random service, made over seed zero
// for a runtime built without one.
func (rt *RuntimeServices) randomService() *random.Service {
	if rt.Random == nil {
		rt.Random = random.NewService(random.Session{})
	}
	return rt.Random
}

// randomService answers the front end's random service, tracking the live
// mission as the holder of the shared stream.
func (f *FrontEnd) randomService() *random.Service {
	svc := f.RuntimeServices.randomService()
	svc.Track(f.liveRandom)
	return svc
}

// prepareLoadRandom prepares the session a LOAD of saved begins and answers
// it with the call that drops it. A runtime built without a service reads an
// unconfigured one and keeps none, so a LOAD that fails changes nothing.
func (f *FrontEnd) prepareLoadRandom(saved random.Session) (random.Session, func()) {
	if f.Random == nil {
		return random.NewService(random.Session{}).Loaded(saved), func() {}
	}
	loaded := f.Random.Loaded(saved)
	f.Random.Prepare(loaded)
	return loaded, f.Random.Cancel
}

// randomSessionNow is the session a save records: the service's session, with
// the running mission's state as the shared state while one holds it.
func (f *FrontEnd) randomSessionNow() random.Session {
	return f.randomService().Session()
}

// liveRandom answers the live mission's World while it runs on the
// original's generator: the holder of the shared stream.
func (f *FrontEnd) liveRandom() random.Shared {
	if f.live == nil || f.live.world == nil || f.live.world.RandomMode() != random.Original {
		return nil
	}
	return f.live.world
}

// missionRandom prepares a mission World's stream in original mode: the
// mission loader and the AI manager reseed the shared stream (SESS-083) and
// placement draws continue it. Seeded mode answers nil and keeps the
// placement seed and the World seed.
func missionRandom(svc *random.Service) *sim.Draws {
	if svc.Mode() != random.Original {
		return nil
	}
	s := svc.Prepared()
	return sim.NewOriginalDraws(random.MissionLoadState(s.Seed, s.Shared))
}

// loadedWorldRandom carries a loaded World's saved stream into the current
// mode. Seeded mode keeps a seeded state as saved; original mode runs the
// load path's reseeds over it (SESS-083).
func loadedWorldRandom(w *sim.World, current random.Session) {
	state := w.RandomState()
	saved := w.RandomMode()
	if current.Mode == random.Seeded {
		if saved == random.Original {
			w.SetRandom(random.Seeded, random.UnfoldOriginal(uint32(state)))
		}
		return
	}
	shared := uint32(state)
	if saved == random.Seeded {
		shared = random.FoldSeeded(state)
	}
	w.SetRandom(random.Original, uint64(random.MissionLoadState(current.Seed, shared)))
}
