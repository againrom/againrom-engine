package sav

// Profile is the game a SAV belongs to: the envelope magic, the serializer
// programmes the one document walk runs and the world-half widths. The two
// games share the envelope, the codec, the head, the Player-list position,
// the world selector and the trailer (R2-SESSION-017, R2-SESSION-018); a
// profile states only where the second game's serializers differ.
type Profile struct {
	// Game is the game the envelope alone names, the way base.Game
	// normalizes it. The Asg& envelope names none: the first game's
	// originals and both games' engine documents use it, and an engine
	// document names its game in its session record.
	Game string
	// Magic is the four envelope bytes at offset 0.
	Magic string

	programmes map[string][]step
	group      []step
	// cellLen is the stride of the terrain's second list and sessionLen the
	// extent of the session block (R2-SESSION-019, R2-ENGINE-603).
	cellLen, sessionLen int
	// cellTail is that a cell entry carries the first game's trigger tail
	// and residue at payload +0x2c..+0x33. The second game's 62-byte entry
	// (a u16 key and 60 raw bytes, R2-ENGINE-603) is read through the first
	// game's fields up to +0x2b only (DIV-4171).
	cellTail bool
	// leafState is that StateData checks a mission file's state store by
	// its grammar and root alone, leaving each leaf to the reader that takes
	// it: the second game's store holds an empty shortcut array (DIV-4154)
	// where the first game's exact shape wants four cells.
	leafState bool
	// session is where the session block's fields lie.
	session sessionLayout
	// extraList is the counted Token list after the Sacks that only the
	// second game writes (R2-ENGINE-601).
	extraList bool
	// maxSpellID is the highest spell ID a book slot holds: 28 in the first
	// game, 29 in the second (R2-ENGINE-019, R2-SESSION-237).
	maxSpellID uint32
	// mainSkill is that Humanoid's 24-byte experience block opens with the
	// main skill, then the experiences of skills 1..5 (R2-ENGINE-320); the
	// first game's block is six experiences.
	mainSkill bool
}

// FirstGame is the first game's profile, the one every engine document uses.
var FirstGame = &Profile{Magic: Magic, programmes: programmes, group: groupProgramme,
	cellLen: cellRecLen, cellTail: true, sessionLen: sessionLen, session: firstSessionLayout, maxSpellID: 28}

// SecondGame is the second game's `Bsg&` profile. Its Token head stores the
// +0x0c word as u16 (38 bytes, R2-ENGINE-209); Unit carries a 184-byte sixth
// raw block; Humanoid has no Diary reference (R2-ENGINE-208); Player writes
// 2,560 raw bytes before its groups and 36 after them, with no Diary
// (R2-SESSION-020); Pointer is a Building with one more DWORD
// (R2-ENGINE-603). Humanoid's raw 24 holds the main skill and the five school
// experiences (R2-ENGINE-320).
var SecondGame = &Profile{Game: "rom2", Magic: "Bsg&", programmes: secondProgrammes(), group: groupProgramme,
	cellLen: secondCellLen, sessionLen: secondSessionLen, session: secondSessionLayout, leafState: true,
	extraList: true, maxSpellID: 29, mainSkill: true}

const (
	secondCellLen    = 62
	secondSessionLen = 10374
)

// sessionLayout places the session block's fields: offsets from the block's
// first byte, the result slot count and the diplomacy matrix side.
type sessionLayout struct {
	results, latches, rawHead, rawMid, diplomacy, won, lost int
	resultCount, diplomacySide                              int
}

var firstSessionLayout = sessionLayout{
	results: sessTriggerResults, latches: sessTriggerLatches, rawHead: sessRawHead, rawMid: sessRawMid,
	diplomacy: sessDiplomacy, won: sessWon, lost: sessLose,
	resultCount: triggerResultCount, diplomacySide: diplomacySide,
}

// secondSessionLayout reads the second game's widths (R2-SESSION-019) with
// the first game's fields by position: 1,000 result slots, the 1,000 latches,
// the two raw regions, an 8-byte head and a 70x70 matrix, then the same
// 18-byte tail. The slot and matrix readings are inference (DIV-4170).
var secondSessionLayout = sessionLayout{
	results: 0, latches: 4000, rawHead: 5000, rawMid: 5048,
	diplomacy: 5456, won: 10362, lost: 10370,
	resultCount: 1000, diplomacySide: 70,
}

// profiles are the envelopes Open accepts, by magic.
var profiles = []*Profile{FirstGame, SecondGame}

// ProfileOf answers the profile whose magic opens b.
func ProfileOf(b []byte) (*Profile, bool) {
	if len(b) < 4 {
		return nil, false
	}
	for _, p := range profiles {
		if string(b[:4]) == p.Magic {
			return p, true
		}
	}
	return nil, false
}

// profile is the file's profile; a File built by hand is the first game's.
func (f *File) profile() *Profile {
	if f == nil || f.Profile == nil {
		return FirstGame
	}
	return f.Profile
}

// SecondGame reports that the file is a second-game `Bsg&` save.
func (f *File) SecondGame() bool { return f.profile() == SecondGame }

// EnvelopeGame is the game the file's envelope names; empty for Asg&.
func (f *File) EnvelopeGame() string { return f.profile().Game }

func secondProgrammes() map[string][]step {
	out := make(map[string][]step, len(programmes)+1)
	for class, prog := range programmes {
		out[class] = prog
	}
	token := make([]Member, len(tokenMembers))
	copy(token, tokenMembers)
	for i := range token {
		if token[i].Name == "T0C" {
			token[i].Kind = KindU16
		}
	}
	out["Token"] = []step{{op: stpRun, members: token}}
	effect := make([]Member, len(effectMembers))
	copy(effect, effectMembers)
	for i := range effect {
		if effect[i].Name == "E0C" {
			effect[i].Kind = KindU16
		}
	}
	out["Effect"] = []step{{op: stpClass, class: "Token"}, {op: stpRun, members: effect}}
	out["Unit"] = unitProgramme(Member{Name: "URaw184", Kind: KindRaw, Len: 184})
	out["Humanoid"] = humanoidProgramme(false)
	out["Player"] = []step{
		{op: stpRun, members: playerMembers},
		{op: stpRun, members: []Member{{Name: "PRaw2560", Kind: KindRaw, Len: 2560}}},
		{op: stpGroups, name: "Groups"},
		{op: stpRun, members: []Member{{Name: "PRaw36", Kind: KindRaw, Len: 36}}},
	}
	out["Pointer"] = []step{
		{op: stpClass, class: "Building"},
		{op: stpRun, members: []Member{{Name: "P6C", Kind: KindU32}}},
	}
	delete(out, "Diary")
	return out
}

// profile answers the walk's profile; nil is the first game.
func (w *walker) profile() *Profile {
	if w.prof == nil {
		return FirstGame
	}
	return w.prof
}

// profile answers the profile of the walk that read the record.
func (r *Record) profile() *Profile {
	if r == nil || r.prof == nil {
		return FirstGame
	}
	return r.prof
}

// humanoidXP splits a Humanoid's experience block into the main skill and
// the six per-skill experiences; the first game's block has no main skill.
func humanoidXP(r *Record, block []byte) (main uint32, xp [CharacterSkillSlots]uint32) {
	for i := range xp {
		xp[i] = u32(block, 4*i)
	}
	if r.profile().mainSkill {
		main, xp[0] = xp[0], 0
	}
	return main, xp
}

// programmes answers the walk's programme table.
func (w *walker) programmes() map[string][]step {
	if w.prof == nil {
		return programmes
	}
	return w.prof.programmes
}

// groupProgramme answers the walk's inline Group programme.
func (w *walker) groupSteps() []step {
	if w.prof == nil {
		return groupProgramme
	}
	return w.prof.group
}
