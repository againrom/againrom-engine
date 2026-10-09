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
)

// ownInOriginal names the streams that keep their own generator in original
// mode: the claims give the mission ambience no call form (MAGIC-285), and
// the bolt and heal figures are drawn at paint, not at the original's driver
// call. Every other stream joins the shared stream.
var ownInOriginal = map[Name]bool{AmbientBirds: true, BoltFigures: true}

// JoinsShared reports whether name draws from the shared stream in original
// mode.
func JoinsShared(name Name) bool { return !ownInOriginal[name] }

// Session is what a save records of the service: the session seed, the mode
// it ran in and, between missions, the shared stream's state.
type Session struct {
	Seed   uint64
	Mode   Mode
	Shared uint32
}

// Shared is the shared stream while a mission holds it: the World's own
// generator in original mode.
type Shared interface {
	OriginalRand() int32
}

// Service is the session's random service. It is not safe for concurrent use.
type Service struct {
	session   Session
	shared    MSVC
	bound     Shared
	streams   map[Name]*Stream
	overrides map[Name]int64
}

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
	s.bound = nil
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
	out.Shared = s.shared.State
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

// Bind hands the shared stream to a running mission; draws go to w until
// Unbind.
func (s *Service) Bind(w Shared) { s.bound = w }

// Unbind takes the shared stream back from a mission at state.
func (s *Service) Unbind(state uint32) {
	s.bound = nil
	s.shared.State = state
}

// Bound reports whether a mission holds the shared stream.
func (s *Service) Bound() bool { return s != nil && s.bound != nil }

// SharedState answers the shared stream's state between missions.
func (s *Service) SharedState() uint32 { return s.shared.State }

// rand is one raw draw of the shared stream.
func (s *Service) rand() int32 {
	if s.bound != nil {
		return s.bound.OriginalRand()
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

// Reseed moves the shared stream between missions to the value the session
// gives site. The original takes it from a clock (DIV-2730).
func (s *Service) Reseed(site Site) {
	s.shared.State = ReseedValue(s.session.Seed, site, s.shared.State)
}

// ReseedValue is the state a reseed at site installs: the mix of the session
// seed, the site and the state it replaces, so a session replays from its
// seed and two reseeds at one site differ.
func ReseedValue(seed uint64, site Site, state uint32) uint32 {
	return uint32(mix64(DeriveSeed(seed, string(site)) ^ uint64(state)))
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
