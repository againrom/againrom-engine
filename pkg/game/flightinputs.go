package game

import (
	"errors"
	"io/fs"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

func loadFlightInputs(src terrain.EntrySource, spells *data.SpellPresentation, bodies data.BodyList) (*sim.FlightInputs, error) {
	if spells.Game == rules.SpellGameSecond {
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
	in := flightInputsFromData(pictures, classes, spells, bodies)
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

func flightInputsFromData(pictures *data.Projectiles, classes *data.UnitClasses, spells *data.SpellPresentation, bodies data.BodyList) sim.FlightInputs {
	in := sim.FlightInputs{Pictures: flightPictures(spells), Classes: make(map[int32]sim.FlightClass), ClassRule: flightClassRule(bodies)}
	for _, id := range pictures.IDs() {
		raw, _ := pictures.ByID(id)
		p := in.Pictures[id]
		p.Present, p.Homes = true, raw.Homing != 0
		p.Phases = flightPhases(int(raw.Phases))
		p.CellLife = int32(spells.CellLife(id))
		in.Pictures[id] = p
	}
	if classes != nil {
		for _, c := range classes.All() {
			in.Classes[c.ID] = sim.FlightClass{Projectile: c.Projectile, ShootDelay: c.ShootDelay, TileSize: c.TileSize,
				CenterX: c.CenterX, CenterY: c.CenterY, SelectionWidth: c.SelectionX2 - c.SelectionX1,
				SelectionHeight: c.SelectionY2 - c.SelectionY1, ShootOffset: slices.Clone(c.ShootOffset)}
		}
	}
	return in
}

func flightPhases(phases int) uint16 {
	if phases > 0 && phases <= 65535 {
		return uint16(phases)
	}
	return 0
}

func flightClassRule(bodies data.BodyList) sim.FlightClassRule {
	return func(e sim.Entity, slots [sim.EquipSlots]uint16) int32 {
		if e.ClientClassFixed {
			return e.Class
		}
		source := e.SourceNow()
		if source.Class == 1 || !e.Humanoid {
			return e.Class
		}
		if !data.FigureIsHero(e.TypeID) {
			if e.TypeID > 0 {
				return e.TypeID
			}
			return e.Class
		}
		fighter := source.Fighter
		if source.Class == 0 {
			if !e.NativeClass.Present {
				return e.Class
			}
			fighter = e.NativeClass.Fighter
		}
		_, _, class, matched := data.HeroAppearance(bodies, equipmentFromSlots(slots), !fighter, false)
		if matched || bodies.Len() != 0 {
			return class
		}
		return e.Class
	}
}

// bindFlightInputs admits late synthetic bundles; installed Worlds bind at load.
func (mw *mapWorld) bindFlightInputs() {
	if mw == nil || mw.world == nil || mw.world.SecondGame() || mw.world.HasFlightInputs() {
		return
	}
	spells := mw.spells()
	in := sim.FlightInputs{Pictures: flightPictures(spells), Classes: make(map[int32]sim.FlightClass)}
	var bodies data.BodyList
	if mw.mission != nil {
		bodies = mw.mission.list
	}
	in.ClassRule = flightClassRule(bodies)
	if mw.projectiles != nil {
		for id, present := range mw.projectiles.Pictures {
			p := in.Pictures[int32(id)]
			p.Present, p.Homes, p.CellLife = present, mw.projectiles.Homes(id), int32(spells.CellLife(int32(id)))
			in.Pictures[int32(id)] = p
		}
		for id, sheet := range mw.projectiles.Sheets {
			if sheet == nil {
				continue
			}
			p := in.Pictures[int32(id)]
			p.Present, p.Homes, p.Phases = mw.projectiles.HasPicture(id), mw.projectiles.Homes(id), flightPhases(sheet.Phases)
			p.CellLife = int32(spells.CellLife(int32(id)))
			in.Pictures[int32(id)] = p
		}
	}
	if mw.units != nil {
		for id, c := range mw.units.Classes {
			if c == nil {
				continue
			}
			in.Classes[id] = sim.FlightClass{Projectile: int32(c.Projectile), ShootDelay: int32(c.ShootDelay), TileSize: int32(c.TileSize),
				CenterX: int32(c.CenterX), CenterY: int32(c.CenterY), SelectionWidth: int32(c.Selection.Dx()),
				SelectionHeight: int32(c.Selection.Dy()), ShootOffset: flightOffsets(c.ShootOffset)}
		}
	}
	mw.world.SetFlightInputs(in)
}

func flightOffsets(offsets []int) []int32 {
	out := make([]int32, len(offsets))
	for i, v := range offsets {
		out[i] = int32(v)
	}
	return out
}
