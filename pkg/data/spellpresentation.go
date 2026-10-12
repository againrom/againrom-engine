package data

import "againrom/pkg/rules"

// SpellPresentation is one game's spell presentation: per effect
// identifier, how long the object lives, what it draws, what it lights and
// when its cell sound plays. A spell row r casts identifier 2r+8 and leaves
// cells of identifier 2r+9 in both games (CastPicture, BurstPicture;
// R2-ENGINE-445). The update arm of each identifier is
// rules.EffectDriveOf, which the World reads too.
type SpellPresentation struct {
	Game rules.SpellGame
	// LastSoundRow is the highest spell row that voices a sound slot.
	LastSoundRow int
	// SoundBase is added to an identifier to give its sound slot.
	SoundBase int
	// BurstLight is the light of a LightBurst effect by frame; a frame past
	// it takes BurstLightTail when HasBurstLightTail, else nothing.
	BurstLight        []LightStep
	BurstLightTail    LightStep
	HasBurstLightTail bool
	// PathLightStep is a path's light level per frame.
	PathLightStep uint8
	// NamedRows is how many spell rows `spell.txt` names, row r by line r-1.
	NamedRows int
	// BookRows is the spell row of each spellbook slot; `spells.txt` line s
	// names the row of slot s.
	BookRows []uint16
	// DrainLifeAliasRow is the row a language's Drain Life alias names
	// (DIV-2177); zero names none, and every row keeps its `spell.txt` line.
	DrainLifeAliasRow int
	effects           []EffectPresentation
}

// LightStep is one light stamp: a radius in cells and a level.
type LightStep struct {
	Radius int
	Level  uint8
}

// LightKind is how an effect lights the cells around it.
type LightKind uint8

const (
	LightNone LightKind = iota
	// LightPoint stamps Light at the object's cell.
	LightPoint
	// LightBurst stamps the set's BurstLight step of the frame.
	LightBurst
	// LightPath stamps each path point's cell at PathLightStep*frame.
	LightPath
	// LightCell stamps Light at each retained overlay cell.
	LightCell
)

// EffectPresentation is one identifier's row. The zero row, with no trail,
// is what an identifier no row names takes.
type EffectPresentation struct {
	ID int32
	// The cast object lives LifeTicks updates; with LifeDivisor the
	// caster-to-target distance over it, at least 1; with LifePhases the
	// sheet's phase count. A zero life builds no object.
	LifeTicks   int
	LifeDivisor int
	LifePhases  bool
	// Trail is which of the two trail sheets the object leaves, with
	// HasTrail.
	HasTrail bool
	Trail    int
	// SecondRecord builds a second record at the destination.
	SecondRecord bool
	// Hidden draws nothing for the object itself.
	Hidden bool
	// Chain is the chain-cast identifier whose loaded record draws no path.
	Chain bool
	// Overlay draws the identifier at each retained area cell.
	Overlay bool
	// BurstLife is a cell object's life; 0 takes burstLifeDefault.
	BurstLife int
	// SoundDelay is the cue count a cell object's sound waits. The cue counts
	// down on the update that creates the object, so the sound plays with
	// frame SoundDelay-1.
	SoundDelay int
	// A falling piece draws frame FallFrames raised FallLift-FallStep*f
	// pixels while its frame f is below FallFrames, then frame f-FallFrames;
	// zero FallFrames draws frame f where it stands.
	FallFrames int
	FallLift   int
	FallStep   int
	LightKind  LightKind
	Light      LightStep
}

// Effect is identifier id's row.
func (s *SpellPresentation) Effect(id int32) EffectPresentation {
	if s != nil {
		for _, e := range s.effects {
			if e.ID == id {
				return e
			}
		}
	}
	return EffectPresentation{ID: id}
}

// Drive is identifier id's update arm in this game.
func (s *SpellPresentation) Drive(id int32) rules.EffectDrive {
	return rules.EffectDriveOf(s.Game, id)
}

// Flight is how many updates the cast object of id lives at a distance of
// dist PictureCellUnits for a sheet of phases frames. A negative dist counts
// as zero.
func (s *SpellPresentation) Flight(id int32, dist, phases int) int {
	e := s.Effect(id)
	switch {
	case e.LifeDivisor > 0:
		return atLeastOne(max(dist, 0) / e.LifeDivisor)
	case e.LifePhases:
		return max(phases, 0)
	}
	return e.LifeTicks
}

// Travels reports whether id's object changes position over a life measured
// by distance.
func (s *SpellPresentation) Travels(id int32) bool {
	return s.Effect(id).LifeDivisor > 0 && s.Drive(id).Motion == rules.EffectTravel
}

// DrawsPath reports whether id draws a point list instead of one sprite.
func (s *SpellPresentation) DrawsPath(id int32) bool {
	return s.Drive(id).Motion == rules.EffectPath
}

// TrailSlot is id's trail sheet, or -1 for none.
func (s *SpellPresentation) TrailSlot(id int32) int {
	if e := s.Effect(id); e.HasTrail {
		return e.Trail
	}
	return -1
}

// CellLife is how many updates the cell object of id stands for.
func (s *SpellPresentation) CellLife(id int32) int {
	if life := s.Effect(id).BurstLife; life > 0 {
		return life
	}
	return burstLifeDefault
}

// OverlayPicture is the identifier a retained area cell of spell row draws,
// and whether it draws one.
func (s *SpellPresentation) OverlayPicture(row int) (int, bool) {
	id := BurstPicture(row)
	return id, s.Effect(int32(id)).Overlay
}

// CastSound is the sound slot a cast of row voices, or 0 outside the rows.
func (s *SpellPresentation) CastSound(row int) int {
	if s == nil || row < 1 || row > s.LastSoundRow {
		return 0
	}
	return s.SoundBase + CastPicture(row)
}

// CellSound is the sound slot a cell object of row voices and the updates it
// waits, or slot 0 outside the rows.
func (s *SpellPresentation) CellSound(row int) (slot, delay int) {
	if s == nil || row < 1 || row > s.LastSoundRow {
		return 0, 0
	}
	id := BurstPicture(row)
	return s.SoundBase + id, s.Effect(int32(id)).SoundDelay
}

// DrawnFrame is the sheet frame id's object draws at frame and how many
// pixels above its point it draws it.
func (s *SpellPresentation) DrawnFrame(id int32, frame int) (drawn, raise int) {
	e := s.Effect(id)
	switch {
	case e.FallFrames <= 0:
		return frame, 0
	case frame < e.FallFrames:
		return e.FallFrames, e.FallLift - e.FallStep*frame
	}
	return frame - e.FallFrames, 0
}

// BurstLightAt is a LightBurst effect's stamp at frame, and false for none.
func (s *SpellPresentation) BurstLightAt(frame int) (LightStep, bool) {
	switch {
	case frame >= 0 && frame < len(s.BurstLight):
		return s.BurstLight[frame], true
	case frame >= len(s.BurstLight) && s.HasBurstLightTail:
		return s.BurstLightTail, true
	}
	return LightStep{}, false
}

// SpellPresentationOf is game's set; an unknown game answers the first's.
func SpellPresentationOf(game rules.SpellGame) *SpellPresentation {
	if game == rules.SpellGameSecond {
		return &secondSpellPresentation
	}
	return &firstSpellPresentation
}

var spellPresentations = map[string]*SpellPresentation{
	"rom1": &firstSpellPresentation,
	"rom2": &secondSpellPresentation,
}

// SpellPresentationNamed is the set an edition names; an unknown name
// answers the first game's.
func SpellPresentationNamed(name string) *SpellPresentation {
	if s, ok := spellPresentations[name]; ok {
		return s
	}
	return &firstSpellPresentation
}

// FirstSpellPresentation is the first game's set.
func FirstSpellPresentation() *SpellPresentation { return &firstSpellPresentation }

var burstLightSteps = []LightStep{{1, 16}, {2, 8}, {3, 0}, {3, 0}, {3, 8}, {3, 16}, {3, 24}, {3, 32}, {3, 40}, {3, 46}, {3, 46}}

// firstBookRows is the first game's spellbook slot table, AI-SPELLIDENT-286.
var firstBookRows = []uint16{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12, 6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}

// secondBookRows is the second game's, R2-ENGINE-444; that each value is the
// row its slot's line names is Medium (DIV-3107).
var secondBookRows = []uint16{1, 2, 3, 4, 24, 20, 21, 26, 13, 12, 11, 10, 5, 6, 7, 8, 25, 22, 23, 27, 19, 18, 17, 16}

// The first game's set: MAGIC-CASTSPAWN-033, MAGIC-BOLTSTILL-072,
// MAGIC-TRAIL-073, MAGIC-BURSTLIFE-034, MAGIC-OVERLAYART-051,
// ANIM-CAST-027, ANIM-147, MAGIC-265, MAGIC-270, MAGIC-271,
// MAGIC-UNITLIGHT-057, DIV-501.
var firstSpellPresentation = SpellPresentation{
	Game: rules.SpellGameFirst, LastSoundRow: 28, SoundBase: 500,
	BurstLight: burstLightSteps, PathLightStep: 10,
	NamedRows: 28, BookRows: firstBookRows, DrainLifeAliasRow: 11,
	effects: []EffectPresentation{
		{ID: 10, LifeDivisor: 200, HasTrail: true, Trail: 0, LightKind: LightPoint, Light: LightStep{0, 16}},
		{ID: 12, LifeDivisor: 384, HasTrail: true, Trail: 1, LightKind: LightPoint, Light: LightStep{1, 16}},
		{ID: 13, BurstLife: burstLifeFire, LightKind: LightBurst},
		{ID: 15, Overlay: true, LightKind: LightCell, Light: LightStep{Radius: 1}},
		{ID: 20, LifeTicks: 1, Hidden: true},
		{ID: 23, Overlay: true},
		{ID: 25, Overlay: true},
		{ID: 27, BurstLife: burstLifeAcid},
		{ID: 30, LifeTicks: 1},
		{ID: 34, LifeTicks: 13, LightKind: LightPath},
		{ID: 36, LifeTicks: 13, Chain: true, LightKind: LightPath},
		{ID: 47, Overlay: true},
		{ID: 51, SoundDelay: 8},
		{ID: 60, LifeTicks: 21, SecondRecord: true},
	},
}

// The second game's set: R2-ENGINE-445..448, R2-ENGINE-394. Lives other
// than the chain's 13 and Blizzard's 16, the trail-free travel of 18 and
// 40, the overlay join, the Teleport second record and LastSoundRow's
// silent slots are chosen: DIV-3102..3106.
var secondSpellPresentation = SpellPresentation{
	Game: rules.SpellGameSecond, LastSoundRow: 29, SoundBase: 500,
	BurstLight: burstLightSteps, BurstLightTail: LightStep{3, 0}, HasBurstLightTail: true,
	PathLightStep: 10, NamedRows: 29, BookRows: secondBookRows,
	effects: []EffectPresentation{
		{ID: 10, LifeDivisor: 200, HasTrail: true, Trail: 0, LightKind: LightPoint, Light: LightStep{0, 16}},
		{ID: 12, LifeDivisor: 384, HasTrail: true, Trail: 1, LightKind: LightPoint, Light: LightStep{1, 16}},
		{ID: 13, LightKind: LightBurst},
		{ID: 15, Overlay: true},
		{ID: 16, LifePhases: true},
		{ID: 18, LifeDivisor: 200, LightKind: LightPoint, Light: LightStep{0, 16}},
		{ID: 20, LifePhases: true},
		{ID: 21, Overlay: true},
		// The original's counter starts at -1 and voices at 8, with frame 8
		// (R2-ENGINE-446), so the cue waits 9.
		{ID: 23, SoundDelay: 9, FallFrames: 8, FallLift: 28, FallStep: 4},
		{ID: 24, LifePhases: true},
		{ID: 28, LifeTicks: 13, LightKind: LightPath},
		{ID: 30, LifeTicks: 13, Chain: true, LightKind: LightPath},
		{ID: 34, LifePhases: true},
		{ID: 40, LifeDivisor: 200, LightKind: LightPoint, Light: LightStep{0, 16}},
		{ID: 43, Overlay: true},
		{ID: 46, LifePhases: true},
		{ID: 48, LifePhases: true},
		{ID: 54, LifePhases: true, SecondRecord: true},
		{ID: 56, LifeTicks: 1, Hidden: true},
		{ID: 60, LifePhases: true},
		{ID: 62, LifePhases: true},
		{ID: 64, LifePhases: true},
	},
}
