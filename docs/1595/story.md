# Shared mission hover

## Intent

Hovering a unit or structure names one object for the cursor, unit highlight,
information card and map release. Transparent sprite pixels do not shrink its
kind rectangle. Changing cursor pictures keeps the registered hotspot on the
pointer in both games.

## Authority

Owner direction requires shared readers in one frame. The pinned k420 snapshot
supplies AI-768 through AI-773 and R2-ENGINE-849 through R2-ENGINE-854.
AI-CURSOR-231 retains the structure-class admission rules. AI-CLICK-050 supplies
cursor-to-order dispatch. AI-CURSOR-230 retains widget and held-item gates.
AI-771 and R2-ENGINE-852 supply the separate 28-slot hotspot profiles;
SPR16A-CURSOR-046 and SPR256-CURSOR-046 retain art and animation metadata.

## As-built behaviour

`hoverRegistrations` emits one kind rectangle for each admitted current object.
The existing unit and structure loaders now distinguish a decoded empty
rectangle from a synthetic kind without rectangle data. The shared
`KindRectangleHover` profile supplies the rectangle sources and registration
phases to both install profiles in `gameFiles`; `sim.gameProfile` is unchanged.
Flat and other structures precede ordinary units, with air units later. Rows
ascend and columns descend within a phase; same-cell inputs retain snapshot
order. `cellSpace` supplies cell centres, lift and cell-sized extents.

`registeredHover` walks the registrations backward and intersects them with
the pointer box `(x-1,y-1,x+1,y+1)`. Intersection requires positive area, as
IntersectRect does. A pointer at the right or bottom endpoint is one pixel
beyond the last rectangle pixel and still hits; one pixel farther does not.
`hoverAt` caches the object and tagged ID together. A new input frame,
`SetEntities`, `SetStructures`, `SetFog` or `SetPhase` invalidates it. Pointer,
camera, body-spell eligibility and `fogReveal` are part of its cache key.
`SetFogReveal`, `ToggleFogReveal` and `SetCheatMapReveal` therefore change the
next reader's hit in the same frame. Cheat visibility and exploration changes
use `SetFog`, including the visibility expiry in `RefreshCheatFog`.

Fog-plane contents, entity and structure snapshots, and interpolation phase
are not cache keys; their setters invalidate the hit. Synthetic checks cover
fog-plane replacement, cheat visibility expiry and entity snapshot movement
and restoration without another input frame. A group camera jump after
`Viewer.step` uses the camera key. QuickLoad replaces the viewer and its cache;
the loaded card and the next pointer input use the restored fog state.

The cursor capability mask, cast target, inspection, unit shading and map tap
read that hit. Attack, cast, defend, item cast and structure use target the
same object; a selection tap applies the existing selectable-population rule
to the same unit. Selection itself remains presentation state; emitted orders
can change deterministic World state. Marquee selection still uses footprint
rectangles. `topAt` remains for selection instruments and is no longer a live
map-tap oracle. Headless pointer searches use the production hover walk.

The sprite texture path selects shading row zero for the hovered unit. It
does not amplify already shaded pixels or MAX-compose an authored hover gain.
Stone and tinted textures accept the same row. Existing diagnostic and
held-input inspection gates remain. The draw-call recorder exposes the row
of the actual submitted texture.

`cursorregistry.go` loads art once and takes hotspots from the install's
`gameFiles` profile. Animated and still attack pictures read the resolved
attack slot. With no resolved slot the authored cross remains; there is no
independent attack-hotspot literal in UI code.

DIV-1349 now states the Medium mapping from the native zero argument to final
pixels. DIV-4178, DIV-4180 and DIV-4492 retain the signpost admission, card and
fog Unknowns while using the new rectangle rule and correcting the retracted
3x3 description to span 2. DIV-4506 records owner-directed same-frame coherence;
DIV-4507 records synthetic bounds and incomplete native admission/order;
DIV-4508 records the retained ROM1 fog gate. DIV-4509 is unused.

## Proof

The failing-test commit adds five ordinary tests against the pinned base:
`TestHoverKindRectangleBoxAndRegistration`,
`TestHoverReadersAndClickShareTheRectangle`,
`TestHoverFinishingBodyAndStructureReaders`,
`TestHoverUsesZeroShadingPixels`, and
`TestAttackFallbackReadsItsRegisteredHotspot`. All five compiled and failed
before production changes, and pass after them. They cover transparent pixels,
the positive-area box edge, overlapping rectangles, ordinary and air units,
finishing bodies, ordinary and flat structures, actual row-zero pixels,
unchanged other units and the attack order at the shared rectangle edge.
The cursor test also drives default, attack and move pictures with both
games' hotspot triples. The existing cursor registry test checks all 28
decoded synthetic slots for both game profiles against independent values.

Related hover, inspection, cursor, tap, cast, click and composition tests pass.
`go build ./...`, the comment ratchet and divergence-claim parsing pass.
The full architecture pass found a bypass of `cellSpace` and a relative-root
lookup in the new guard; both were corrected and the two focused guards pass.
`TestHoverHasOneRectangleWalkAndHotspotSource` guards the removed parallel
oracles and the shared reader routes. The final fast gate is returned with
the exact candidate and its external receipt.

`TestHoverRevealInputRefreshesReaders` and
`TestHoverRevealReleaseAttacksVisibleEnemy` fail at the correction base
`4288e92f5` through production `App.step` input for both profiles. They check
reveal in both directions, the card, shading, cursor and Attack instead of Move
on the reveal frame. `TestHoverRevealSettersRefreshReaders` covers direct
reveal setters. The fog/snapshot, group-camera and QuickLoad controls name the
other checked cache dependencies. Their receipts accompany the correction
return.

## Open debt

`TestReleaseMissionHoverRectangleReaders` is written and registered without a
root column. It must run at the next release on EN and RU for both games. It
opens the installed first mission, pauses simulation, uses an independently
placed unit and structure kind rectangle, and checks their exposed box edges,
card identity, attack hotspot, actual hovered texture row and unchanged World
hash and command queue. It has not run in this lane. Existing release pointer,
inspection, cursor, panel and target-order witnesses and mission scenarios
must run at that release too; first-game-only witnesses retain their existing
scope.

Original out-of-sight admission, native scheduling, complete alternate/corpse
registration, exact native hover pixels and the second game's exact Pointer
card values remain Unknown at the cited DIV rows. Fixture tests establish no
installed kind values or original-runtime observations. The sole review has
returned; the seat pushes and lands this one correction pass without another
review.
