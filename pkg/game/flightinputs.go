package game

import (
	"cmp"
	"errors"
	"io/fs"
	"maps"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// flightProfile is a game's record producer profile. registryRows reads the
// installed registries at load, else the loaded art's rows; drawnClass takes
// geometry from the class the client draws (spellClientClass).
type flightProfile struct {
	sim.FlightProfile
	registryRows bool
	drawnClass   bool
}

func loadFlightInputs(src terrain.EntrySource, spells *data.SpellPresentation, profile flightProfile) (*sim.FlightInputs, error) {
	if !profile.registryRows {
		return nil, nil
	}
	pictures, err := loadProjectileRecords(src)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	classes, err := loadUnitClasses(src)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	in := buildFlightInputs(spells, profile.FlightProfile, flightClassRule, installedPictureRows(pictures), installedClassRows(classes))
	return &in, nil
}

func flightPictures(spells *data.SpellPresentation) map[int32]sim.FlightPicture {
	out := make(map[int32]sim.FlightPicture)
	for _, e := range spells.Effects() {
		out[e.ID] = sim.FlightPicture{LifeTicks: int32(e.LifeTicks), LifeDivisor: int32(e.LifeDivisor),
			LifePhases: e.LifePhases, SecondRecord: e.SecondRecord, CellLife: int32(spells.CellLife(e.ID))}
	}
	return out
}

type flightPictureRow struct {
	ID                    int32
	Present, Homes, Drawn bool
	Phases                uint16
}

type flightClassRow struct {
	ID int32
	sim.FlightClass
}

// buildFlightInputs is the one producer of sim.FlightInputs; a source only
// exposes its picture and class rows.
func buildFlightInputs(spells *data.SpellPresentation, profile sim.FlightProfile, rule sim.FlightClassRule, pictures []flightPictureRow, classes []flightClassRow) sim.FlightInputs {
	in := sim.FlightInputs{Pictures: flightPictures(spells), Classes: make(map[int32]sim.FlightClass), ClassRule: rule, Profile: profile}
	for _, r := range pictures {
		p := in.Pictures[r.ID]
		p.Present, p.Homes, p.Drawn, p.Phases = r.Present, r.Homes, r.Drawn, r.Phases
		p.CellLife = int32(spells.CellLife(r.ID))
		in.Pictures[r.ID] = p
	}
	for _, c := range classes {
		in.Classes[c.ID] = c.FlightClass
	}
	return in
}

func installedPictureRows(pictures *data.Projectiles) []flightPictureRow {
	var rows []flightPictureRow
	for _, id := range pictures.IDs() {
		raw, _ := pictures.ByID(id)
		rows = append(rows, flightPictureRow{ID: id, Present: true, Homes: raw.Homing != 0, Phases: flightPhases(int(raw.Phases))})
	}
	return rows
}

func installedClassRows(classes *data.UnitClasses) []flightClassRow {
	if classes == nil {
		return nil
	}
	var rows []flightClassRow
	for _, c := range classes.All() {
		rows = append(rows, flightClassRow{c.ID, sim.FlightClass{Projectile: c.Projectile, ShootDelay: c.ShootDelay, TileSize: c.TileSize,
			CenterX: c.CenterX, CenterY: c.CenterY, SelectionWidth: c.SelectionX2 - c.SelectionX1,
			SelectionHeight: c.SelectionY2 - c.SelectionY1, ShootOffset: slices.Clone(c.ShootOffset)}})
	}
	return rows
}

func flightPhases(phases int) uint16 {
	if phases > 0 && phases <= 65535 {
		return uint16(phases)
	}
	return 0
}

// flightClassRule is the class whose flight geometry an actor's shots and
// casts use. A hero's is the class the World stores for him, which the party
// redress writes from heroAppearance; a fixed class, a Unit and a non-humanoid
// keep theirs; another Human uses his Humans-row type (ANIM-106).
func flightClassRule(e sim.Entity) int32 {
	if e.ClientClassFixed || e.SourceNow().Class == 1 || !e.Humanoid || data.FigureIsHero(e.TypeID) {
		return e.Class
	}
	if e.TypeID > 0 {
		return e.TypeID
	}
	return e.Class
}

// bindFlightInputs binds the loaded art's rows to a World the load did not.
func (mw *mapWorld) bindFlightInputs() {
	if mw == nil || mw.world == nil || mw.world.HasFlightInputs() {
		return
	}
	profile := mw.files().flight
	rule := flightClassRule
	if profile.drawnClass {
		rule = mw.drawnFlightClass
	}
	mw.world.SetFlightInputs(buildFlightInputs(mw.spells(), profile.FlightProfile, rule, syntheticPictureRows(mw.spells().Game, mw.projectiles), syntheticClassRows(mw.units)))
}

func (mw *mapWorld) drawnFlightClass(e sim.Entity) int32 { return mw.spellClientClass(e.ID, e.Class) }

// syntheticPictureRows: a sheet's row replaces its picture's row; a homing
// row alone still homes.
func syntheticPictureRows(game rules.SpellGame, set *terrain.EffectSet) []flightPictureRow {
	if set == nil {
		return nil
	}
	rows := make(map[int]flightPictureRow, len(set.Pictures)+len(set.Sheets))
	for id := range set.Homing {
		rows[id] = flightPictureRow{ID: int32(id), Present: set.HasPicture(id), Homes: set.Homes(id), Drawn: shotDrawn(game, set, id)}
	}
	for id, present := range set.Pictures {
		rows[id] = flightPictureRow{ID: int32(id), Present: present, Homes: set.Homes(id), Drawn: shotDrawn(game, set, id)}
	}
	for id, sheet := range set.Sheets {
		if sheet != nil {
			rows[id] = flightPictureRow{ID: int32(id), Present: set.HasPicture(id), Homes: set.Homes(id), Drawn: true, Phases: flightPhases(sheet.Phases)}
		}
	}
	return slices.SortedFunc(maps.Values(rows), func(a, b flightPictureRow) int { return cmp.Compare(a.ID, b.ID) })
}

func syntheticClassRows(units *terrain.UnitSet) []flightClassRow {
	if units == nil {
		return nil
	}
	var rows []flightClassRow
	for id, c := range units.Classes {
		if c == nil {
			continue
		}
		rows = append(rows, flightClassRow{id, sim.FlightClass{Projectile: int32(c.Projectile), ShootDelay: int32(c.ShootDelay), TileSize: int32(c.TileSize),
			CenterX: int32(c.CenterX), CenterY: int32(c.CenterY), SelectionWidth: int32(c.Selection.Dx()),
			SelectionHeight: int32(c.Selection.Dy()), ShootOffset: flightOffsets(c.ShootOffset)}})
	}
	slices.SortFunc(rows, func(a, b flightClassRow) int { return cmp.Compare(a.ID, b.ID) })
	return rows
}

func flightOffsets(offsets []int) []int32 {
	out := make([]int32, len(offsets))
	for i, v := range offsets {
		out[i] = int32(v)
	}
	return out
}
