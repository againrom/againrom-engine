package random

// Mode is how a session draws.
type Mode uint8

const (
	// Seeded is the default: every stream runs on its own generator, each
	// presentation stream seeded from the session seed and its name.
	Seeded Mode = 0
	// Original puts every consumer the original places on its main thread
	// (MAGIC-283) on one MSVC stream (SESS-082), with the original's reseed
	// points (SESS-083) taking their values from the session seed.
	Original Mode = 1
)

// Name names one stream. Every consumer draws from a named stream.
type Name string

// The named streams.
const (
	World         Name = "world"
	Placement     Name = "placement"
	CommandVoice  Name = "command-voice"
	Music         Name = "music"
	AmbientBirds  Name = "ambient-birds"
	TownAnimation Name = "town-animation"
	TownAmbient   Name = "town-ambient"
	TownWildlife  Name = "town-wildlife"
	Tavern        Name = "tavern"
	ShopInterior  Name = "shop-interior"
	School        Name = "school"
	BoltFigures   Name = "bolt-figures"
	ItemStars     Name = "item-stars"
	ShopStock     Name = "shop-stock"
	Generator     Name = "generator"
)

// ownInOriginal names the streams that keep their own generator in original
// mode: the claims give the mission ambience no call form (MAGIC-285), and
// the bolt and heal figures and the character generator's sparkle are drawn
// at paint, not at the original's driver call. Every other stream joins the
// shared stream.
var ownInOriginal = map[Name]bool{AmbientBirds: true, BoltFigures: true, Generator: true}

// JoinsShared reports whether name draws from the shared stream in original
// mode.
func JoinsShared(name Name) bool { return !ownInOriginal[name] }

// Session is what a save records of the service: the session seed, the mode
// it ran in, between missions the shared stream's state, and how many reseeds
// the session has taken (original mode only).
type Session struct {
	Seed    uint64
	Mode    Mode
	Shared  uint32
	Reseeds uint32
}

// Shared is the shared stream while a mission holds it: the World's own
// generator in original mode.
type Shared interface {
	OriginalRand() int32
	RandomState() uint64
}

// Service is the session's random service. It is not safe for concurrent use.
type Service struct {
	session   Session
	shared    MSVC
	track     func() Shared
	last      Shared
	stale     Shared
	streams   map[Name]*Stream
	overrides map[Name]int64
	launch    Launch
	pending   *Session
}

// Launch is what the launch settings choose: a fixed session seed, or none so
// that each new game takes the caller's clock value, and the mode. Generator
// names the original generator the game's evidence establishes; original
// mode runs only over one this package implements.
type Launch struct {
	Seed      uint64
	Fixed     bool
	Mode      Mode
	Generator string
	// configured is set by SetLaunch. A service whose launch was never set
	// begins every new game at seed zero, so a session built without launch
	// settings replays without a clock.
	configured bool
}

// MSVCGenerator names the original's CRT generator (MAGIC-279), the one
// original generator this package implements.
const MSVCGenerator = "msvc"

// SetLaunch records the launch settings and begins the process's session.
// In original mode the shared stream starts at seed 1, the item-star grids
// take its first draws and sound initialisation reseeds it (SESS-083); the
// reseed discards the state the grids left. Original mode over a generator
// this package does not implement runs seeded instead, and SetLaunch reports
// that it fell back.
func (s *Service) SetLaunch(l Launch, clock uint64) (fellBack bool) {
	if l.Mode == Original && l.Generator != MSVCGenerator {
		l.Mode, fellBack = Seeded, true
	}
	l.configured = true
	s.launch = l
	session := Session{Seed: clock, Mode: l.Mode}
	if l.Fixed {
		session.Seed = l.Seed
	}
	if l.Mode == Original {
		session.reseed(SoundInit)
	}
	s.Begin(session)
	return fellBack
}

// Prepare holds session for a game being prepared: a new game or a LOAD
// builds its mission over it while the running game keeps its own. Commit
// begins it; Cancel drops it.
func (s *Service) Prepare(session Session) { s.pending = &session }

// Prepared answers the session being prepared, else the running one.
func (s *Service) Prepared() Session {
	if s.pending != nil {
		return *s.pending
	}
	return s.Session()
}

// Commit begins the prepared session, if any.
func (s *Service) Commit() {
	if s.pending != nil {
		p := *s.pending
		s.pending = nil
		s.Begin(p)
	}
}

// Cancel drops the prepared session.
func (s *Service) Cancel() { s.pending = nil }

// LaunchSettings answers the launch settings.
func (s *Service) LaunchSettings() Launch { return s.launch }

// Fresh is the session a new game begins: the launch seed when one is fixed,
// otherwise clock, in the launch mode. In original mode its count starts at
// zero and the scenario constructor's reseed sets the shared stream, so a new
// game does not depend on what the process drew before it.
func (s *Service) Fresh(clock uint64) Session {
	out := Session{Seed: clock, Mode: s.launch.Mode, Shared: s.SharedState()}
	if s.launch.Fixed || !s.launch.configured {
		out.Seed = s.launch.Seed
	}
	if out.Mode == Original {
		out.Shared = 0
		out.reseed(ScenarioConstructor)
	}
	return out
}

// Loaded is the session a LOAD begins from a saved session: its seed, its
// reseed count and the launch mode. In seeded mode it carries the saved
// shared state unchanged. In original mode the load path's two reseeds
// (SESS-083) set the shared state from the seed and the count. A LOAD is never
// refused for the mode it was saved in.
func (s *Service) Loaded(saved Session) Session {
	out := Session{Seed: saved.Seed, Mode: s.launch.Mode, Shared: saved.Shared, Reseeds: saved.Reseeds}
	if out.Mode == Original {
		out.missionLoad()
	}
	return out
}

// MissionStart takes a mission start's two reseeds in the session being
// prepared, else the running one, and answers the shared state they leave.
func (s *Service) MissionStart() uint32 {
	session := &s.session
	if s.pending != nil {
		session = s.pending
	}
	session.missionLoad()
	if session == &s.session {
		s.shared.State = session.Shared
	}
	return session.Shared
}

// MissionLoadState is the shared state a mission or save load's two reseeds
// leave at count n: the mission loader's at n, then the AI manager's at n+1
// (SESS-083).
func MissionLoadState(seed uint64, n uint32) uint32 {
	return ReseedValue(seed, AIManager, n+1)
}

// missionLoad takes the mission loader's and the AI manager's reseeds.
func (s *Session) missionLoad() {
	s.reseed(MissionLoader)
	s.reseed(AIManager)
}

// reseed takes one reseed at site: the shared state becomes the value the
// seed, the site and the count give, and the count advances.
func (s *Session) reseed(site Site) {
	s.Shared = ReseedValue(s.Seed, site, s.Reseeds)
	s.Reseeds++
}

// UnfoldOriginal carries an original 32-bit state into a seeded 64-bit one.
func UnfoldOriginal(state uint32) uint64 { return mix64(uint64(state) | 1<<32) }

// NewService is a service over session.
func NewService(session Session) *Service {
	s := &Service{}
	s.Begin(session)
	return s
}

// Begin adopts session: the shared state is the session's and every stream
// already handed out restarts from its seed, in place.
func (s *Service) Begin(session Session) {
	s.session = session
	s.shared = MSVC{State: session.Shared}
	s.last, s.stale = nil, s.holder()
	for name, st := range s.streams {
		st.g = NewGo(s.StreamSeed(name))
	}
}

// Session answers the session with the shared stream's current state.
func (s *Service) Session() Session {
	if s == nil {
		return Session{}
	}
	out := s.session
	out.Shared = s.SharedState()
	return out
}

// Mode answers the session's mode; a nil service is seeded.
func (s *Service) Mode() Mode {
	if s == nil {
		return Seeded
	}
	return s.session.Mode
}

// StreamSeed is the seed a seeded stream starts from: an override when a test
// set one, otherwise the mix of the session seed and the stream's name.
func (s *Service) StreamSeed(name Name) int64 {
	if v, ok := s.overrides[name]; ok {
		return v
	}
	return int64(DeriveSeed(s.session.Seed, string(name)))
}

// SetStreamSeed fixes one seeded stream's seed and restarts it. It is a test
// injection point.
func (s *Service) SetStreamSeed(name Name, seed int64) {
	if s.overrides == nil {
		s.overrides = map[Name]int64{}
	}
	s.overrides[name] = seed
	if st := s.streams[name]; st != nil {
		st.g = NewGo(seed)
	}
}

// Stream answers the named stream, made on first use. A stream lasts the
// service's life; a new session restarts it in place.
func (s *Service) Stream(name Name) *Stream {
	if s.streams == nil {
		s.streams = map[Name]*Stream{}
	}
	st := s.streams[name]
	if st == nil {
		st = &Stream{svc: s, name: name, g: NewGo(s.StreamSeed(name))}
		s.streams[name] = st
	}
	return st
}

// Track names the running mission's stream: holder answers the World that
// holds the shared stream now, or nil between missions. When a holder goes
// away the service takes its state back. A holder present when a session
// begins belongs to the session before and is never adopted.
func (s *Service) Track(holder func() Shared) { s.track = holder }

func (s *Service) holder() Shared {
	if s.track == nil {
		return nil
	}
	h := s.track()
	if h == nil || h == s.stale {
		return nil
	}
	return h
}

// sync answers the current holder, taking the state back from one that went
// away.
func (s *Service) sync() Shared {
	h := s.holder()
	if h != s.last && s.last != nil {
		s.shared.State = uint32(s.last.RandomState())
	}
	s.last = h
	return h
}

// Bound reports whether a mission holds the shared stream.
func (s *Service) Bound() bool { return s != nil && s.sync() != nil }

// SharedState answers the shared stream's state, the holder's while a
// mission holds it.
func (s *Service) SharedState() uint32 {
	if h := s.sync(); h != nil {
		return uint32(h.RandomState())
	}
	return s.shared.State
}

// rand is one raw draw of the shared stream.
func (s *Service) rand() int32 {
	if h := s.sync(); h != nil {
		return h.OriginalRand()
	}
	return s.shared.Rand()
}

// Site names one of the original's reseed points (SESS-083).
type Site string

// The reseed points.
const (
	SoundInit           Site = "sound-init"
	ScenarioConstructor Site = "scenario-constructor"
	MissionLoader       Site = "mission-loader"
	AIManager           Site = "ai-manager"
)

// ReseedValue is the state a reseed at site installs: the mix of the session
// seed, the site and the session's reseed count n. The original takes it from
// a clock (DIV-2730). It never reads the state it replaces, so what the
// session drew between two reseeds, the town's time-paced draws among them,
// does not reach the next value; two reseeds at one site differ by count.
func ReseedValue(seed uint64, site Site, n uint32) uint32 {
	return uint32(mix64(DeriveSeed(seed, string(site)) ^ uint64(n)))
}

// DeriveSeed mixes a seed with a name: FNV-1a over the name, then the
// SplitMix64 finalizer over the sum with the seed.
func DeriveSeed(seed uint64, name string) uint64 {
	h := uint64(14695981039346656037)
	for i := 0; i < len(name); i++ {
		h ^= uint64(name[i])
		h *= 1099511628211
	}
	return mix64(seed + h)
}

// SeedOf derives a session seed from bytes: an original save that carries no
// session seed takes the seed of its own document.
func SeedOf(b []byte) uint64 {
	h := uint64(14695981039346656037)
	for _, c := range b {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return mix64(h)
}

// NewStream is a standalone seeded stream that starts at seed, for a
// consumer a test drives without a session.
func NewStream(seed int64) *Stream {
	s := NewService(Session{})
	s.SetStreamSeed("standalone", seed)
	return s.Stream("standalone")
}

// Stream is one named stream.
type Stream struct {
	svc  *Service
	name Name
	g    *Go
}

// Shared reports whether the stream draws from the shared stream now.
func (st *Stream) Shared() bool {
	return st.svc.session.Mode == Original && JoinsShared(st.name)
}

// Raw answers one value 0..RandMax: a raw rand() on the shared stream, or a
// 15-bit draw of the stream's own generator.
func (st *Stream) Raw() int {
	if st.Shared() {
		return int(st.svc.rand())
	}
	return st.g.Raw()
}

// Intn answers a value in [0, n): the range wrapper over n-1 on the shared
// stream (no draw for n = 1), or the own generator's Intn.
func (st *Stream) Intn(n int) int {
	if st.Shared() {
		if n <= 1 {
			return 0
		}
		return int(rangeOf(st.svc.rand(), int32(n-1)))
	}
	return st.g.Intn(n)
}

// Scaled answers a value in [0, n): n*rand()/32767 reduced modulo n on the
// shared stream, or the own generator's Intn.
func (st *Stream) Scaled(n int) int {
	if n <= 0 {
		return 0
	}
	if st.Shared() {
		return int(st.svc.rand()) * n / RandMax % n
	}
	return st.g.Intn(n)
}
