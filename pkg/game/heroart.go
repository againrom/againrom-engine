package game

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
)

// LoadHeroBody resolves one hero body into the bundle: the sheet composed from
// dir and body, on the geometry of the class record body resolves to.
//
// EVERY FAILURE IS A SKIP AND NOTHING IS FATAL. A body naming no loaded
// class record, an entry that is not in the archive, a stream the decoder
// refuses, a palette-less sheet and a sheet of no frames all leave the
// bundle without an entry — and a missing entry draws the class record's
// own art, which is the picture the whole tree had before this loader
// existed. That is the font's rule and it is deliberate: a mission must not
// fail to open over a cosmetic asset.
//
// THE CLASS KEY IS THE LAW'S ANSWER WHATEVER IT IS. The match flag is not
// consulted, because the law is total on purpose: a name matching no arm is
// drawn as the class the original leaves standing, from a sheet composed with
// that same name. Consulting the flag here would invent a refusal the original
// does not have — and costs nothing to omit, since such a name ships no sheet
// and is refused one line later by the archive itself.
//
// IT WRITES UNDER data.HeroBodyKey(dir, body), NOT UNDER THE BARE NAME.
//
// A BODY ONCE LOADED IS NEVER DECODED TWICE: this function returns at once
// when the bundle already holds the composed key, before the class record is
// even looked up. That is what makes the frame-paced refresh cheap to call
// on demand rather than something a caller has to remember to guard — the
// second and every later request for a body already reached is a map read,
// not an archive one.
//
// It reads and decodes through a cache of its OWN, dropped with the call. Two of
// the shipped class records name a sheet under a hero directory, so one address
// can be reached by both this loader and the unit loader and be decoded twice
// per process; that is a load-time cost of a few sheets, and sharing a memo
// across the two would mean the unit loader knowing which bodies a party wears.
func LoadHeroBody(src terrain.EntrySource, set *terrain.UnitSet, dir string, body data.HeroBody) {
	if src == nil || set == nil {
		return
	}
	key := data.HeroBodyKey(dir, body)
	if _, ok := set.Bodies[key]; ok {
		return
	}
	id, _ := data.HeroBodyClass(body)
	rec := set.Classes[id]
	if rec == nil {
		return
	}
	sheets := sheetCache{
		src:       src,
		decoded:   make(map[string]*spr256.Sprite),
		converted: make(map[string][]*terrain.StaticFrame),
	}
	frames := sheets.frames(data.HeroSheetPath(dir, body))
	if len(frames) == 0 {
		return
	}

	// The corpse link is the hero's own dying body: the original recomposes
	// the whole appearance for a dying character and forces it into one of two
	// named bodies (HERO-APPEAR-042), composed from the same directory.
	boundary := sheets.boundaryFrames(frames, data.HeroSheetPath(dir, body), data.HeroOverlayPath(dir, body))
	drawn := composeHeroBody(rec, frames, boundary, heroDyingBody(&sheets, set, dir, body))

	if set.Bodies == nil {
		set.Bodies = make(map[string]*terrain.UnitClass)
	}
	set.Bodies[key] = drawn
}

// composeHeroBody is the one builder of a hero body's drawn class, for a
// shipped sheet and a mod's alike: rec supplies the geometry and every other
// field, frames the picture, boundary the second silhouette (nil for none) and
// corpse the fallen body (nil keeps rec's own link).
//
// A STRUCT COPY, not a wrapper. THE TIER SLICES ARE CLEARED: a tier is a
// colour table for the RECORD's own sheet, and carrying them over would
// recolour a hero's picture through a table built for a different one.
func composeHeroBody(rec *terrain.UnitClass, frames, boundary []*terrain.StaticFrame, corpse *terrain.UnitClass) *terrain.UnitClass {
	drawn := *rec
	drawn.Frames = frames
	drawn.Tiers = nil
	drawn.Boundary = boundary
	if corpse != nil {
		drawn.Corpse = corpse
	}
	return &drawn
}

// heroDyingBody is the class a fallen hero is drawn as: the forced dying body's
// composed sheet on that body's own class record (HERO-APPEAR-042,
// REG-UNITS-050: both forced bodies name themselves as their Dying class). It
// answers nil when the record or the sheet is missing, which leaves the class
// record's own corpse link in force.
func heroDyingBody(sheets *sheetCache, set *terrain.UnitSet, dir string, body data.HeroBody) *terrain.UnitClass {
	mage := body == data.BodyMage || body == data.BodyMageStaff
	dying := data.HeroBodyName(data.BodyUnarmed, false, mage, true)
	id, _ := data.HeroBodyClass(dying)
	rec := set.Classes[id]
	if rec == nil {
		return nil
	}
	frames := sheets.frames(data.HeroSheetPath(dir, dying))
	if len(frames) == 0 {
		return nil
	}
	corpse := composeHeroBody(rec, frames, sheets.boundaryFrames(frames, data.HeroSheetPath(dir, dying), data.HeroOverlayPath(dir, dying)), nil)
	corpse.Corpse = nil
	return corpse
}
