package game

import (
	"crypto/sha256"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Which figure each PLACED HUMAN composes its doll from: the directory and the
// face (`HERO-APPEAR-041`, `HERO-DOLL-078`), resolved once when a map opens.
//
// IT IS entityTiers' OWN SHAPE FOR THE OTHER BAND, and the two are deliberately
// separate walks over the same placements. A placement resolves down one of two
// bands — the units collection or the humans one — and each band's picture is
// decided by different columns of a different record. tiers.go answers the
// creature band's tier; this answers the human band's figure. One walk carrying
// both would have to branch on the arm twice and would hold two kinds of nothing
// in one map.
//
// LIKE THE TIER, IT IS PRESENTATION AND IT LIVES BESIDE THE WORLD. No field of
// a world or of an entity carries a figure directory, so none of this enters a
// byte form or a digest, and pkg/sim is untouched.

// figureID is one composing actor's own figure: which of the four shipped
// directories its sheets live under, and which face inside it.
//
// IT IS COMPARABLE AND THAT IS LOAD-BEARING: it is half of the key the composed
// picture is cached under, the other half being the equipment.
type figureID struct {
	Dir   data.FigureDir
	Face  int
	Horse bool
	Hero  bool
}

// entityFigures is the figure every HUMAN placement of m resolves to, keyed by
// the entity id the world builder mints for it.
//
// THE KEY IS THE MINTED ID, on entityTiers' own ground: the i-th unit record
// takes id i, counting from zero, and both this lookup and the world beside it
// are built from the same decoded map in the same call.
//
// ONLY THE HUMAN BAND CONTRIBUTES. A placement that resolved down the units
// collection is a creature: it has no figure at all, it has a flat portrait, and
// it is ABSENT here rather than present with a zero — absent being the answer,
// exactly as tiers.go says of a creature with no tier.
//
// It is TOTAL and reports nothing, and it is a pure function of the map and the
// table — tiers.go's own two properties, for its own two reasons.
func entityFigures(m *alm.Map, t *mapload.Table) map[sim.EntityID]figureID {
	if m == nil || t == nil || t.Humans == nil {
		return nil
	}
	out := make(map[sim.EntityID]figureID, len(m.Units))
	for i, u := range m.Units {
		r := mapload.Resolve(u, t)
		if r.Arm == mapload.ArmUnits || !r.Found() {
			continue
		}
		d, err := data.NewHumanDef(t.Humans.EntryName(r.Index), t.Humans.EntryParams(r.Index))
		if err != nil {
			continue
		}
		dir, face, _ := mapload.PlacedFigure(u, t)
		out[sim.EntityID(i)] = figureID{Dir: dir, Face: face, Horse: data.FigureHasHorse(d.TypeID),
			Hero: r.Arm == mapload.ArmNPC && t.NPC.Hero(int32(u.ClassSubID))}
	}
	return out
}

func savedActorFigures(m *alm.Map, t *mapload.Table, entities []sim.Entity, state *SnapshotSAVDocument) map[sim.EntityID]figureID {
	placed := entityFigures(m, t)
	if state == nil || len(placed) == 0 {
		return placed
	}
	byMapID := make(map[uint16]int, len(m.Units))
	for i, unit := range m.Units {
		if unit.UnitID == 0 {
			continue
		}
		if _, found := byMapID[unit.UnitID]; found {
			byMapID[unit.UnitID] = -1
		} else {
			byMapID[unit.UnitID] = i
		}
	}
	figures := make(map[sim.EntityID]figureID, len(placed))
	for _, entity := range entities {
		if !entity.Humanoid || entity.MapUnitID == 0 {
			continue
		}
		i, found := byMapID[entity.MapUnitID]
		if !found || i < 0 {
			continue
		}
		if figure, found := placed[sim.EntityID(i)]; found {
			figures[entity.ID] = figure
		}
	}
	return figures
}

// partyFigures adds party portrait identities to the map-placement identities.
// Human hires have read-only composed portraits; their world sprites and
// inventory eligibility are controlled separately.
func partyFigures(out map[sim.EntityID]figureID, party []mapload.PartyMember, ids []sim.EntityID) map[sim.EntityID]figureID {
	n := len(party)
	if len(ids) < n {
		n = len(ids)
	}
	if n == 0 {
		return out
	}
	if out == nil {
		out = make(map[sim.EntityID]figureID, n)
	}
	for i := 0; i < n; i++ {
		if !data.ComposesFigure(party[i].Class) {
			continue
		}
		out[ids[i]] = memberFigureID(party[i])
	}
	return out
}

// rosterFigureID is a placed person's figure. Its roster template supplies the
// directory and face. The hero background and the horse follow the live actor's
// own type id (UNIT-APPEAR-030, HERO-DOLL-078): the loader marks every template
// a PlayerCharacter, and the class a template carries is not the actor's drawn
// class, because the equipment law rewrites it to a body class when the mission
// opens and a SAV load replaces it with the class the entity was placed with.
func rosterFigureID(m mapload.PartyMember, w *sim.World, id sim.EntityID) figureID {
	dir, face := memberFigure(m)
	fig := figureID{Dir: dir, Face: face}
	if w != nil {
		if e, ok := w.Entity(id); ok {
			fig.Hero = data.FigureIsHero(e.TypeID)
			fig.Horse = data.FigureHasHorse(e.TypeID)
		}
	}
	return fig
}

// figureCacheKey is what a composed figure is cached under: whose figure it is
// and what that actor is wearing.
//
// THE EQUIPMENT IS IN THE KEY BECAUSE THE PICTURE IS A FUNCTION OF IT. A doll is
// a base sheet with one layer per occupied slot painted over it, so two actors
// of one figure wearing different things are two pictures — and one actor whose
// worn set changes mid-mission is a third. data.Equipment is a plain comparable
// array, which is what lets it be half a map key rather than a revision counter
// somebody has to remember to bump.
type figureCacheKey struct {
	fig figureID
	eq  data.Equipment
}

// LiveDollEvidence is the no-window measurement of the actual unit-figure
// compositor. Full is the live worn set; Bare, Weapon and Shield are controlled
// recompositions of the same actor figure and installed archive. A digest
// difference proves that the corresponding live layer changed compositor
// output without committing or exporting game pixels.
type LiveDollEvidence struct {
	Equipment                  [sim.EquipSlots]uint16
	Full, Bare, Weapon, Shield [sha256.Size]byte
	FullDrawn, BareDrawn       bool
	WeaponDrawn, ShieldDrawn   bool
}

func figureDigest(pic *image.RGBA) ([sha256.Size]byte, bool) {
	if pic == nil {
		return [sha256.Size]byte{}, false
	}
	return sha256.Sum256(pic.Pix), true
}

func composeDollEvidence(src entrySource, slots [sim.EquipSlots]uint16, fig figureID) LiveDollEvidence {
	full := equipmentFromSlots(slots)
	var weapon, shield data.Equipment
	if slots[0] != 0 {
		weapon.SetCode(1, data.ItemCode(slots[0]))
	}
	if slots[1] != 0 {
		shield.SetCode(2, data.ItemCode(slots[1]))
	}
	out := LiveDollEvidence{Equipment: slots}
	fullPic, _ := composeUnitFigure(src, full, fig)
	barePic, _ := composeUnitFigure(src, data.Equipment{}, fig)
	weaponPic, _ := composeUnitFigure(src, weapon, fig)
	shieldPic, _ := composeUnitFigure(src, shield, fig)
	out.Full, out.FullDrawn = figureDigest(fullPic)
	out.Bare, out.BareDrawn = figureDigest(barePic)
	out.Weapon, out.WeaponDrawn = figureDigest(weaponPic)
	out.Shield, out.ShieldDrawn = figureDigest(shieldPic)
	return out
}

// composeUnitFigure is the doll for one composing actor: its base sheet with
// each occupied slot's layer painted over it, in data.FigureDrawOrder — for
// anyone but a mage the weapon or the shield, whichever data.FigureHeldLast
// names, last over every other occupied slot (0151-layers-and-names T1, T7);
// a mage follows the programme of its own — and, for a slot
// data.HasItemFigureSecondaryLayer names, its second sheet too (T2).
//
// IT COMPOSES THE FIGURE AND NOT THE WORN ICONS. composeInventorySubject
// (inventory.go) builds both, because the inventory window draws both; this box
// draws one picture, and the twelve icon reads that function performs would be
// twelve archive reads per newly selected actor for pictures nothing here shows.
// The figure half is the same steps in the same order and is written here
// rather than factored out of that function, because factoring it would leave
// that one reading its own loop's guard from somewhere else. THE ORDER ITSELF
// IS data.FigureDrawOrder, shared with that function so the two composers
// cannot silently disagree on it (0151 T1's own brief).
//
// THE BODY LIST IS READ HERE, off src, THE SAME PRECEDENT world.go's own
// refreshAppearance already set for invPartyGear.list (T7): this composer
// carries no Table and no FrontEnd field to reach an already-read copy
// through, so ReadBodyList runs again over the same shipped bytes — bounded
// by mw.unitFigure's own cache, one read per (figure, equipment) pair this
// build has not composed before, never per frame.
func composeUnitFigure(src entrySource, eq data.Equipment, fig figureID) (*image.RGBA, *ui.SlotMask) {
	if src == nil {
		return nil, nil
	}
	sheets := sheetCache{src: src, decoded: map[string]*spr256.Sprite{}, converted: map[string][]*terrain.StaticFrame{}}
	base := figureRGBA(&sheets, data.ItemFigureBasePath(fig.Dir, fig.Face))
	if base == nil {
		return nil, nil
	}
	// THE MASK IS BUILT BESIDE base, the moment there is a canvas of the right
	// bounds to match — 1005 "the interactive doll", the same treatment
	// composeInventorySubject (inventory.go) gives its own base.
	mask := newFigureSlotMask(base.Bounds())
	const mageCloakSlot = 8
	if fig.Dir.Mage() {
		if occupied, _ := eq.Occupied(mageCloakSlot); occupied {
			code, _ := eq.Code(mageCloakSlot)
			if layer := figureRGBA(&sheets, data.ItemFigureLayerPath(fig.Dir, code)); layer != nil {
				composed := image.NewRGBA(base.Bounds())
				// A FRESH CANVAS EARNS A FRESH MASK, same bounds and all zero
				// again: nothing has painted composed yet.
				mask = newFigureSlotMask(composed.Bounds())
				paintFigureLayerMasked(composed, layer, mask, mageCloakSlot)
				paintFigureLayerMasked(composed, base, mask, 0)
				base = composed
			}
		}
	}
	list, _ := ReadBodyList(src)
	for _, st := range data.FigureDrawSteps(fig.Dir, list, eq) {
		n := st.Slot
		occupied, _ := eq.Occupied(n)
		if !occupied || st.Kind == data.FigureStepNone {
			continue
		}
		if fig.Dir.Mage() && n == mageCloakSlot && !st.Secondary {
			// Primary already stands behind the body.
			continue
		}
		if st.Secondary && !data.HasItemFigureSecondaryLayer(n) {
			continue
		}
		code, _ := eq.Code(n)
		path := data.ItemFigureLayerPath(fig.Dir, code)
		if st.Secondary {
			path = data.ItemFigureSecondaryLayerPath(fig.Dir, code)
		}
		if layer := figureRGBA(&sheets, path); layer != nil {
			if st.Kind == data.FigureStepTag {
				tagFigureLayer(mask, layer, n, nil)
			} else {
				paintFigureLayerMasked(base, layer, mask, n)
			}
		}
	}
	return addFigureBackground(src, base, fig), mask
}

// unitFigure is the composed doll for one entity, or nil for one this build
// cannot compose a figure for — resolved through the cache the caller holds.
//
// A CACHED MISS IS A CACHED ANSWER, classPortrait's own convention: an actor
// whose base sheet the install does not carry costs one read per (figure,
// equipment) pair rather than one per frame.
func (mw *mapWorld) unitFigure(id sim.EntityID) *image.RGBA {
	fig, ok := mw.figures[id]
	if !ok {
		return nil
	}
	key := figureCacheKey{fig: fig, eq: mw.equipmentOf(id)}
	if pic, tried := mw.figurePics[key]; tried {
		return pic
	}
	pic, mask := composeUnitFigure(mw.archive(), key.eq, fig)
	if mw.figurePics == nil {
		mw.figurePics = make(map[figureCacheKey]*image.RGBA)
	}
	mw.figurePics[key] = pic
	// THE MASK IS CACHED UNDER THE SAME KEY THE PICTURE IS (1005 contract,
	// item 1) — a sibling map rather than a struct value, so unitFigure's own
	// callers (portrait.go's classPortrait, among others) keep reading a bare
	// *image.RGBA and never learn this story added a second cache. Round 1
	// wires no consumer to unitFigureMask: only the party's own doll subject
	// (composeInventorySubject) is interactive this round, and this map exists
	// so a later round asking "which slot did THIS unit's doll pixel belong
	// to" is a cache lookup, not a second compositor pass.
	if mw.figureMasks == nil {
		mw.figureMasks = make(map[figureCacheKey]*ui.SlotMask)
	}
	mw.figureMasks[key] = mask
	return pic
}

// equipmentOf is one entity's worn set as this package's own record, or the
// zero value for an entity the world no longer holds — currentEquipment's own
// reading (world.go), for an id that is not the inventory subject's.
func (mw *mapWorld) equipmentOf(id sim.EntityID) data.Equipment {
	slots, ok := mw.world.Equipped(id)
	if !ok {
		return data.Equipment{}
	}
	return equipmentFromSlots(slots)
}
