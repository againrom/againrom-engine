package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func memberFigure(m mapload.PartyMember) (data.FigureDir, int) {
	dir := data.FigureDir(m.FigureDir)
	if dir == "" {
		dir = data.FigureDirManFighter
	}
	face := m.FigureFace
	if face == 0 {
		face = 1
	}
	return dir, face
}

// buildInventorySubject is T4's builder (plan D-6, D-11), widened by
// docs/0136-armour-counts T5 (spec FR-10d) to compose from the member's
// WHOLE WORN SET rather than his weapon alone: it takes an entry source,
// reads the archive, PAINTS THE FIGURE, and returns the finished subject
// alongside every address it could not read, IN ADDRESS ORDER — figure
// base, then, for each occupied slot in data.FigureDrawOrder, that slot's
// figure layer, its second layer if data.HasItemFigureSecondaryLayer names
// one, and then its icon. That order is composeInventorySubject's own build
// order (FR-10b, widened by 0151-layers-and-names T1, T2 and T7): the base
// is read first because it is the canvas every layer paints onto, and each
// slot's layer and icon are independent of the base and of every other slot.
//
// IT WRITES TO NO STREAM AND HOLDS NO VIEWER (T4 fence). A caller decides
// what, if anything, happens with either return — see openMission, which
// keeps the subject and discards the list, and FrontEnd.MissionLine, which
// recomputes the same list to report it (plan D-11's own comment there says
// why the two differ).
//
// THE EQUIPMENT TO COMPOSE FROM IS missionDollEquipment's OWN ANSWER
// (FR-10d, corrected round-2 adversarial review, twelfth pass, C1) — see
// that function's own doc for the Carry-over-Worn rule and the weapon
// fallback it applies; this function no longer states either inline.
//
// A MEMBER WITH NO WORN SET AND NO WEAPON IS A BARE HERO, exactly as before
// this story: every slot's own Occupied answers false, and no layer address
// and no icon address is even composed for any of the twelve — there is no
// code to build any of them from.
//
// THE EQUIPMENT AND THE FALLBACK ARE missionDollEquipment's OWN RETURN
// (C1, twelfth pass), split out below rather than composed inline: a
// caller that needs to know what THIS COMPOSITION actually drew from —
// world.go's openMission and switchInventorySubject, seeding
// mw.invComposedEquipment for C1b's own reason — calls the same function
// rather than restating its two-line body a second time next to a comment
// claiming the two agree.
func buildInventorySubject(src entrySource, ms *Mission) (ui.InventorySubject, []string) {
	if len(ms.Party) == 0 || len(ms.Start.IDs) == 0 {
		return ui.InventorySubject{}, nil
	}
	member := ms.Party[0]

	eq, fallback := missionDollEquipment(ms.World, ms.Start.IDs[0], member)

	subject, extras := composeInventoryPortrait(src, uint32(ms.Start.IDs[0]), eq, memberFigureID(member))
	// WeaponFallback IS SET HERE, NOT LEFT TO A LATER CALLER (counterexample 4,
	// round-2 adversarial review, third pass): openMission recomputes the same
	// condition through mw.currentWeaponFallbackActive() once mw.invParty
	// exists (world.go), which supersedes this value for the mission's own live
	// subject — but this function is buildInventorySubject's whole contract,
	// self-contained, and a caller that never reaches that second computation
	// must still get a subject that correctly refuses to arm a drag it cannot
	// act on.
	subject.WeaponFallback = fallback
	return subject, extras
}

// missionDollEquipment is the equipment the mission doll's figure and slot
// mask are composed from.
//
// IT PREFERS THE LIVE WORLD, w.Equipped(id), OVER THE PARTY RECORD (round-2
// adversarial review, twelfth pass, C1 — corrected a second time within
// the same pass after a scenario-level mutation-kill exposed the first
// correction's own gap). The first correction routed this function through
// mapload.EquipmentFromParty(member) — Carry.Equipped over Worn, the rule
// every other compositor already applied — which fixed the cross-mission
// case (C1's own original counterexample: a member who finished a PREVIOUS
// mission wearing something his frozen member.Worn still disagreed with) but
// left a NARROWER, same-mission case open: member.Carry is nil for the whole
// of the CURRENT mission (CarryParty only sets it at that mission's own
// finish), so EquipmentFromParty falls back to member.Worn — the array as
// read at the party's LAST assembly, never rewritten by a live
// sim.KindEquip/KindUnequip during play. A save taken mid-mission, after the
// player changed slot 7 through the doll, and then RELOADED, opened showing
// the SLOT 7 THE MISSION STARTED WITH: the loaded world (resumeWorld,
// resume.go, replaces ms.World's own entity data BEFORE openMission runs)
// already held the correct, current equipment, but this function was still
// reading the stale party record beside it.
// `scenarios/1005-doll-and-shop.json` step 87 (existing, pre-1005-round-2)
// witnesses exactly this: it saves and reloads mid-mission and asserts slot
// 1 stays retired, and it failed under the first correction with `figure
// slot 1 = 265, want empty` — the doll showing the mission-start weapon
// back on after a reload that had genuinely taken it off.
//
// w.Equipped(id) is what closes both cases at once, because it is also what
// mapload/start.go's own mint already writes at TRUE mission open — that
// mint's own p.Carry-then-p.Worn arm (the same preference
// EquipmentFromParty restates) is what SEEDS the entity's live equipped
// array in the first place, so reading it back off w agrees with
// EquipmentFromParty at open and additionally reflects whatever a resumed
// save's own binary world snapshot, or a live in-mission change, holds by
// the time this runs. mapload.EquipmentFromParty(member) remains the
// fallback for a caller with no world or no minted entity for id yet — the
// zero-value id case buildInventorySubject already guards, and
// switchInventorySubject's own throwaway one-member Mission if it were ever
// built without w set.
//
// The weapon is folded in only as slot 1's fallback, and only when the
// resolved set leaves slot 1 empty and member.WeaponMaterialized has not
// already been set (round-2 adversarial review, fifth pass). This used to be
// a live scan of the member's own pack for the weapon's own code — the
// same present-state proxy world.go's weaponFallbackSpent documents as the
// root cause behind counterexamples A and B: it stopped holding once the
// item left the container the scan inspected (a sale, for instance).
// WeaponMaterialized is the persisted history bit that replaces it; nothing
// here writes it, since this function only composes a display and has no
// mission to write back into. member.Weapon itself is read from the party
// record regardless of w, because it is not equipment at all — it is the
// STARTING WEAPON REFERENCE, a fact about the class/roster row a member was
// minted from, and the live world carries no equivalent field to prefer over
// it.
func missionDollEquipment(w *sim.World, id sim.EntityID, member mapload.PartyMember) (eq data.Equipment, fallback bool) {
	if slots, ok := equippedIfAny(w, id); ok {
		eq = equipmentFromSlots(slots)
	} else {
		eq = mapload.EquipmentFromParty(member)
	}
	if occupied, _ := eq.Occupied(1); !occupied && member.Weapon != nil && !member.WeaponMaterialized {
		fallback = true
		eq.SetCode(1, member.Weapon.Code)
	}
	return eq, fallback
}

// equippedIfAny is w.Equipped(id) widened to tolerate w == nil, which
// buildInventorySubject's own callers can legitimately pass: openMission
// always has a world by the time it calls missionDollEquipment (the world
// is built before invSubject is), but a caller composing a subject with no
// mission context at all — none exists in the shipped tree today, and none
// is added by this pass — would otherwise panic on a nil receiver rather
// than falling back to the party record like every pre-C1 caller did.
func equippedIfAny(w *sim.World, id sim.EntityID) ([sim.EquipSlots]uint16, bool) {
	if w == nil {
		return [sim.EquipSlots]uint16{}, false
	}
	return w.Equipped(id)
}

func composeInventorySubject(src entrySource, id uint32, eq data.Equipment,
	figureDir data.FigureDir, figureFace int) (ui.InventorySubject, []string) {
	return composeInventorySubjectLayered(src, id, eq, nil, figureDir, figureFace)
}

// composeInventorySubjectLayered is composeInventorySubject with the clothing
// layers a mod adds. A layer is painted at its anchor slot's turn in
// data.FigureDrawOrder: an under layer just before the item worn there, an over
// layer just after it, whether or not the slot is occupied. A layer owns no
// slot in the mask, so a press on it starts no drag of the anchor's item.
func composeInventorySubjectLayered(src entrySource, id uint32, eq data.Equipment, layers []figureLayer,
	figureDir data.FigureDir, figureFace int) (ui.InventorySubject, []string) {
	var subject ui.InventorySubject
	subject.ID = id
	var hoverCoverage figureSlotCoverage

	basePath := data.ItemFigureBasePath(figureDir, figureFace)
	baseAddr := graphicsPrefix + basePath

	// A NIL SOURCE READS NOTHING (LoadAttackPointer's own rule, cursor.go),
	// and every address this call would otherwise attempt is unread rather
	// than silently skipped: a caller that reaches here with no archive at
	// all still gets a total, honest answer about what it could not show,
	// and never a nil-pointer panic out of sheetCache. eq.Occupied and
	// eq.Code read no src at all, so this arm still walks every slot and
	// lists the SAME addresses per occupied slot, in the SAME order, the
	// archive arm below would have attempted (FR-10b). data.FigureDrawOrder
	// is asked with a nil body list (T7): there is no archive here to read
	// one from, and data.FigureHeldLast(nil, eq) is total over that — it
	// answers slot 2 last, the same as an archive that cannot supply the
	// list would (FigureHeldLast's own doc).
	if src == nil {
		unread := []string{baseAddr}
		steps := data.FigureDrawSteps(figureDir, nil, eq)
		first, _ := figureStepSpans(steps)
		for i, st := range steps {
			n := st.Slot
			occupied, _ := eq.Occupied(n)
			if !occupied {
				continue
			}
			code, _ := eq.Code(n)
			switch {
			case st.Kind == data.FigureStepNone:
			case st.Secondary:
				if data.HasItemFigureSecondaryLayer(n) {
					unread = append(unread, graphicsPrefix+data.ItemFigureSecondaryLayerPath(figureDir, code))
				}
			default:
				unread = append(unread, graphicsPrefix+data.ItemFigureLayerPath(figureDir, code))
			}
			if first[n] == i {
				unread = append(unread, graphicsPrefix+data.ItemIconPath(code))
			}
		}
		return subject, unread
	}

	sheets := sheetCache{src: src, decoded: map[string]*spr256.Sprite{}, converted: map[string][]*terrain.StaticFrame{}}
	var unread []string

	if base := figureRGBA(&sheets, basePath); base != nil {
		subject.Figure = base
		subject.SlotMask = newFigureSlotMask(base.Bounds())
		hoverCoverage = newFigureSlotCoverage(base.Bounds())
	} else {
		unread = append(unread, baseAddr)
	}

	// HERO-FIGURE-059's mage programme paints slot 8's primary sheet before
	// the head/body and its secondary sheet at the very end. The shipped mage
	// row uses this slot for the cloak. Building a fresh canvas is necessary:
	// painting the body onto the cloak is the opposite composition from
	// painting the cloak over the body, even when both images use the same
	// bounds.
	const mageCloakSlot = 8
	if figureDir.Mage() {
		if occupied, _ := eq.Occupied(mageCloakSlot); occupied {
			code, _ := eq.Code(mageCloakSlot)
			layerPath := data.ItemFigureLayerPath(figureDir, code)
			if layer := figureRGBA(&sheets, layerPath); layer != nil {
				if subject.Figure != nil {
					body := subject.Figure
					composed := image.NewRGBA(body.Bounds())
					// THE MASK IS REBUILT FOR THE NEW CANVAS, same bounds and
					// all zero again: nothing has painted composed yet, so the
					// mask the base alone earned above would misname every
					// pixel the cloak or the body is about to claim.
					subject.SlotMask = newFigureSlotMask(composed.Bounds())
					paintFigureLayerMaskedCovered(composed, layer, subject.SlotMask, mageCloakSlot, &hoverCoverage)
					// THE BODY REPAINT CLEARS THE CLOAK'S OWN MARK WHEREVER IT
					// COVERS IT (slot 0 — paintFigureLayerMasked's own doc):
					// "the topmost layer that painted a non-transparent pixel"
					// is the base body there, not the cloak beneath it, and the
					// mask says so the same way the picture does.
					paintFigureLayerMasked(composed, body, subject.SlotMask, 0)
					subject.Figure = composed
				}
			} else {
				unread = append(unread, graphicsPrefix+layerPath)
			}
		}
	}

	paintLayers := func(n int, depth int8) {
		for _, l := range layers {
			if l.Anchor != n || l.Depth != depth {
				continue
			}
			for _, path := range l.paths(figureDir) {
				if layer := figureRGBA(&sheets, path); layer != nil {
					if subject.Figure != nil {
						paintFigureLayerMasked(subject.Figure, layer, subject.SlotMask, 0)
					}
				} else {
					unread = append(unread, graphicsPrefix+path)
				}
			}
		}
	}

	list, _ := ReadBodyList(src)
	steps := data.FigureDrawSteps(figureDir, list, eq)
	first, last := figureStepSpans(steps)
	for i, st := range steps {
		n := st.Slot
		if first[n] == i {
			paintLayers(n, mapload.LayerUnder)
		}
		if occupied, _ := eq.Occupied(n); occupied {
			code, _ := eq.Code(n)
			paintStep := st.Kind != data.FigureStepNone &&
				!(figureDir.Mage() && n == mageCloakSlot && !st.Secondary) &&
				!(st.Secondary && !data.HasItemFigureSecondaryLayer(n))
			if paintStep {
				path := data.ItemFigureLayerPath(figureDir, code)
				if st.Secondary {
					path = data.ItemFigureSecondaryLayerPath(figureDir, code)
				}
				if layer := figureRGBA(&sheets, path); layer != nil {
					// A layer that decoded while the base did NOT is left
					// UNPAINTED rather than promoted to the figure on its own.
					if subject.Figure != nil {
						if st.Kind == data.FigureStepTag {
							tagFigureLayer(subject.SlotMask, layer, n, &hoverCoverage)
						} else {
							paintFigureLayerMaskedCovered(subject.Figure, layer, subject.SlotMask, n, &hoverCoverage)
						}
					}
				} else {
					unread = append(unread, graphicsPrefix+path)
				}
			}
			if first[n] == i {
				iconAddr := graphicsPrefix + data.ItemIconPath(code)
				if icon, err := loadItemIcon(src, iconAddr); err == nil {
					// Each of the twelve slot icons comes from its own slot's
					// code: slot n's icon lands in Slots[n-1].
					subject.Slots[n-1] = icon
				} else {
					unread = append(unread, iconAddr)
				}
			}
		}
		if last[n] == i {
			paintLayers(n, mapload.LayerOver)
		}
	}

	subject.HoverSlotMask = hoverCoverage.hoverMask(subject.SlotMask)
	return subject, unread
}

// figureRGBA resolves path's frame 0 through the .256 pipeline pkg/game's
// other consumers already use — heroart.go's LoadHeroBody and statics.go's
// loader, both built on sheetCache: decode, palette-check, convert to the
// render tier's own frame type, and — the one step neither of those needs —
// blit it onto a fresh, transparent canvas the size of the frame
// (StaticFrame.RGBA, render/terrain/blit.go). That blit is what makes the
// result premultiplied the same way spr256 always is: every opaque pixel is
// written at alpha 0xff, so multiplying by a coverage of exactly 0 or
// exactly 1 leaves every channel exactly where it started.
//
// It answers nil for every one of the format's own exclusions — no entry, an
// undecodable stream, no palette, no frame at index 0 — with no error value
// of its own: the caller needs only to know whether the address read, and
// which of the four exclusions it met is not part of what this file reports.
func figureRGBA(sheets *sheetCache, path string) *image.RGBA {
	f := sheets.frame(path, 0)
	if f == nil {
		return nil
	}
	return f.RGBA()
}

// figureStepSpans maps each slot to the index of its first and last step.
func figureStepSpans(steps []data.FigureStep) (first, last map[int]int) {
	first, last = map[int]int{}, map[int]int{}
	for i, st := range steps {
		if _, ok := first[st.Slot]; !ok {
			first[st.Slot] = i
		}
		last[st.Slot] = i
	}
	return first, last
}

// tagFigureLayer claims the layer's opaque pixels for slot in the mask and
// the hover coverage without painting any colour.
func tagFigureLayer(mask *ui.SlotMask, layer *image.RGBA, slot int, coverage *figureSlotCoverage) {
	lb := layer.Bounds()
	for y := 0; y < lb.Dy(); y++ {
		for x := 0; x < lb.Dx(); x++ {
			if layer.Pix[layer.PixOffset(lb.Min.X+x, lb.Min.Y+y)+3] == 0 {
				continue
			}
			if mask != nil && x < mask.W && y < mask.H {
				mask.Slot[y*mask.W+x] = uint8(slot)
			}
			if coverage != nil {
				coverage.mark(x, y, slot)
			}
		}
	}
}

func paintFigureLayer(dst, layer *image.RGBA) {
	paintFigureLayerMasked(dst, layer, nil, 0)
}

// paintFigureLayerMasked is paintFigureLayer's own body, widened to record
// WHICH SLOT painted each pixel it touches (1005, "the interactive doll";
// contract DIV-085) — built BESIDE the composition, from the identical
// opacity test that already decides whether to paint, rather than by a
// second pass over the picture paintFigureLayer produces: mask is nil for
// every call this story does not add (paintFigureLayer's own wrapper,
// above), and every other detail — the smaller-of-two-bounds walk, the
// transparent-source skip, the raw byte copy — is unchanged from what this
// function replaced.
//
// slot 0 marks a pixel as no equipment slot's own: the mage-cloak
// recombination's own body repaint (figures.go, composeInventorySubject)
// passes it to CLEAR whatever slot number an earlier, now-covered paint left
// there, which is what makes "the topmost layer that painted a
// non-transparent pixel" true of the mask even where the topmost layer is
// the base body repainted over an equipped layer rather than an equipped
// layer itself.
func paintFigureLayerMasked(dst, layer *image.RGBA, mask *ui.SlotMask, slot int) {
	paintFigureLayerMaskedCovered(dst, layer, mask, slot, nil)
}

func paintFigureLayerMaskedCovered(dst, layer *image.RGBA, mask *ui.SlotMask, slot int, coverage *figureSlotCoverage) {
	db, lb := dst.Bounds(), layer.Bounds()
	w, h := min(db.Dx(), lb.Dx()), min(db.Dy(), lb.Dy())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			so := layer.PixOffset(lb.Min.X+x, lb.Min.Y+y)
			if layer.Pix[so+3] == 0 {
				// A transparent source pixel leaves dst exactly as it was —
				// the base's own colour there, never a hole punched in it.
				continue
			}
			do := dst.PixOffset(db.Min.X+x, db.Min.Y+y)
			copy(dst.Pix[do:do+4], layer.Pix[so:so+4])
			if mask != nil && x < mask.W && y < mask.H {
				mask.Slot[y*mask.W+x] = uint8(slot)
			}
			if coverage != nil {
				coverage.mark(x, y, slot)
			}
		}
	}
}

// figureSlotCoverage is the sparse-lived compositor record behind
// InventorySubject.HoverSlotMask. Each occupied slot gets one bitset while
// its real shipped layer is painted; the bitsets are discarded before this
// function returns, leaving only one mask-sized result.
type figureSlotCoverage struct {
	w, h  int
	slots [data.EquipSlots][]uint64
}

func newFigureSlotCoverage(bounds image.Rectangle) figureSlotCoverage {
	return figureSlotCoverage{w: bounds.Dx(), h: bounds.Dy()}
}

func (c *figureSlotCoverage) mark(x, y, slot int) {
	if c == nil || slot < 1 || slot > len(c.slots) || x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	i := y*c.w + x
	bits := c.slots[slot-1]
	if bits == nil {
		bits = make([]uint64, (c.w*c.h+63)/64)
		c.slots[slot-1] = bits
	}
	bits[i/64] |= uint64(1) << uint(i%64)
}

// hoverMask retains the final topmost mask wholesale, then uses a small
// bipartite match to reserve one distinct painted pixel for every worn layer.
// Thus an entirely covered ring still has a hover target at a pixel its own
// shipped layer actually painted, while the rendered RGBA remains untouched.
func (c *figureSlotCoverage) hoverMask(top *ui.SlotMask) *ui.SlotMask {
	if c == nil || top == nil || c.w != top.W || c.h != top.H {
		return top
	}
	out := &ui.SlotMask{W: top.W, H: top.H, Slot: append([]uint8(nil), top.Slot...)}
	owner := make([]int, c.w*c.h)
	for i := range owner {
		owner[i] = -1
	}
	var assign func(int, []bool) bool
	assign = func(slot int, seen []bool) bool {
		bits := c.slots[slot]
		for word, set := range bits {
			for set != 0 {
				bit := bitsTrailingZeros64(set)
				p := word*64 + bit
				set &^= uint64(1) << uint(bit)
				if p >= len(owner) || seen[p] {
					continue
				}
				seen[p] = true
				if owner[p] < 0 || assign(owner[p], seen) {
					owner[p] = slot
					return true
				}
			}
		}
		return false
	}
	for slot, bits := range c.slots {
		if bits != nil {
			assign(slot, make([]bool, len(owner)))
		}
	}
	for p, slot := range owner {
		if slot >= 0 {
			out.Slot[p] = uint8(slot + 1)
		}
	}
	return out
}

func bitsTrailingZeros64(v uint64) int {
	if v == 0 {
		return 64
	}
	n := 0
	for v&1 == 0 {
		n++
		v >>= 1
	}
	return n
}

// newFigureSlotMask builds a mask sized to a just-established base canvas'
// own bounds, all zero — "no slot painted here yet", figureSlotMask's own
// starting state before the first call to paintFigureLayerMasked.
func newFigureSlotMask(b image.Rectangle) *ui.SlotMask {
	w, h := b.Dx(), b.Dy()
	return &ui.SlotMask{W: w, H: h, Slot: make([]uint8, w*h)}
}

// loadItemIcon resolves the .16a icon at addr — the full, prefixed address,
// unlike figureRGBA's path, which takes sheetCache's own unprefixed spelling
// — through LoadAttackPointer's own path (cursor.go): ReadFile, DecodeA with
// the palette declared present, frame 0, and the one .16a frame converter.
//
// Every failure is an error and none is a partial picture — LoadAttackPointer's
// own contract, restated here because this is a second, independent
// implementation of the same read-decode-resolve sequence over a different
// address, and the two must refuse alike or a caller could not treat their
// errors the same way.
func loadItemIcon(src entrySource, addr string) (*image.RGBA, error) {
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil, err
	}
	sprite, err := spr16.DecodeA(b, true)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	if len(sprite.Frames) == 0 {
		return nil, fmt.Errorf("%s holds no frames", addr)
	}
	f := sprite.Frames[0]
	if f.Width <= 0 || f.Height <= 0 {
		return nil, fmt.Errorf("%s frame 0 is %dx%d and would draw nothing", addr, f.Width, f.Height)
	}

	return f.RGBA(sprite.Palette), nil
}

// buildInventoryPack is T6's own builder, taking an ELEMENT per cell rather
// than a code since 0138: one icon per carried ELEMENT, IN ELEMENT ORDER,
// through loadItemIcon and data.ItemIconPath — the SAME .16a reader
// buildInventorySubject's own icon arm already uses, never a second one (T6
// fence). It returns two values shaped exactly like ui.InventorySubject.Pack
// and .PackCount — one entry per carried element — plus the address of
// every icon it could not read, in element order.
//
// COUNTS[i] IS stacks[i].Count WHENEVER A CELL IS BUILT AT ALL, set BEFORE
// the icon is even attempted and independent of whether it reads: the count
// is a fact about the CONTAINER, the icon a fact about the ARCHIVE, and an
// unread icon does not make the element's own count any less true — a pack
// cell that could not show its picture still carries whatever number the
// window would have drawn over it.
//
// IT NO LONGER STOPS AT EIGHT (0140). 0112 spec D-3 disclosed a cap here —
// pkg/sim's container is UNBOUNDED and the window's pack was not, so a
// carrier holding more elements than there were cells had the remainder
// UNDRAWN, a limit of the WINDOW and not of the container. The owner
// replaced that window with a bar that scrolls "potentially endlessly", so
// this builder composes every element the container holds and how many of
// them are on screen at once is decided by the bar, per window size
// (pkg/ui/hud.go). The cap is gone rather than raised: there is no number
// here to get wrong.
//
// THE CACHE IS KEYED BY CODE and is the CALLER'S, held across calls: a code
// this cache has already resolved — successfully or not, sheetCache's own
// "tried" convention (statics.go) — is never read again, so a recompose
// after a grab that only ADDED a code rereads that one code alone.
//
// A NIL src READS NOTHING (LoadAttackPointer's own rule, buildInventorySubject's
// own restatement of it) and reports every address it would have attempted,
// in element order, so a caller with no archive at all still gets a total,
// honest answer about what it could not show.
const inventoryGoldIconAddr = graphicsPrefix + "interface/money/money.16a"

func loadInventoryGoldIcon(src entrySource) *image.RGBA {
	if src == nil {
		return nil
	}
	icon, _ := loadItemIcon(src, inventoryGoldIconAddr)
	return icon
}

// appendInventoryGold adds the participant purse after the real container
// elements. The picture is graphics.res's decoded money sprite, not an authored
// substitute. Appending preserves every real item's pack index, so a
// double-click on a gold cell cannot be mistaken for another item by the equip
// path.
func appendInventoryGold(subject *ui.InventorySubject, gold uint32, icon *image.RGBA) {
	if subject == nil {
		return
	}
	// The purse mark belongs to the pack just rebuilt; a mark kept from an
	// earlier pack would tag an item cell as the purse.
	subject.PackPurse = nil
	if gold == 0 {
		return
	}
	subject.Pack = append(subject.Pack, icon)
	subject.PackCount = append(subject.PackCount, gold)
	subject.PackStars = append(subject.PackStars, false)
	subject.PackPurse = append(make([]bool, len(subject.Pack)-1), true)
	subject.PackInfo = append(subject.PackInfo, []string{"Gold", fmt.Sprintf("Quantity: %d", gold)})
}

// itemShowsStarTrail is ITEM-STARFLAG-096's normal compact-display selector.
// A non-empty Effect list is necessary, while Potion (kind 3) explicitly
// clears the display bit. Nonzero picture is tested later by the cell painter
// against the resolved icon, so a missing picture cannot produce bare stars.
func itemShowsStarTrail(item sim.ItemInstance) bool {
	return !item.Empty() && item.Kind != 3 && item.HasEnchantment()
}

func itemStackStarFlags(stacks []sim.ItemStack) []bool {
	if len(stacks) == 0 {
		return nil
	}
	flags := make([]bool, len(stacks))
	for i := range stacks {
		flags[i] = itemShowsStarTrail(stacks[i].Instance())
	}
	return flags
}

func inventoryGold(subject ui.InventorySubject) uint32 {
	if len(subject.PackInfo) == 0 || len(subject.PackCount) < len(subject.PackInfo) {
		return 0
	}
	last := len(subject.PackInfo) - 1
	if len(subject.PackInfo[last]) == 0 || subject.PackInfo[last][0] != "Gold" {
		return 0
	}
	return subject.PackCount[last]
}

func buildInventoryPack(src entrySource, stacks []sim.ItemStack, cache map[uint16]*image.RGBA) ([]*image.RGBA, []uint32, []string) {
	if len(stacks) == 0 {
		// A CARRIER OF NOTHING ANSWERS NIL AND NOT AN EMPTY SLICE, which is
		// the same value ui.InventorySubject's own zero holds — so a subject
		// built for an empty container and one never built at all present
		// identically, and neither is a special case anywhere downstream.
		return nil, nil, nil
	}
	pack := make([]*image.RGBA, len(stacks))
	counts := make([]uint32, len(stacks))
	var unread []string
	for i, st := range stacks {
		counts[i] = st.Count
		addr := graphicsPrefix + data.ItemIconPath(data.ItemCode(st.Code))
		if src == nil {
			unread = append(unread, addr)
			continue
		}
		icon, tried := cache[st.Code]
		if !tried {
			icon, _ = loadItemIcon(src, addr) // a failure resolves to a nil icon, cached alike
			cache[st.Code] = icon
		}
		if icon == nil {
			unread = append(unread, addr)
			continue
		}
		pack[i] = icon
	}
	return pack, counts, unread
}
