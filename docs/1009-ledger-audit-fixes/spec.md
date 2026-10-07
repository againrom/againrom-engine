# 1009 — ledger audit fixes: spec

Canonicalized to the shipped behaviour.

## FR-1 — `movies.res` is required, and carries the merchant's picture

`game.MoviesArchive = "movies.res"` is added to `Archives.Containers`'s open set alongside
`GraphicsArchive`, `MainArchive`, `WorldArchive` and `ScenarioArchive`, joined at the end of the
`hosts` slice. It is required: `OpenArchives` refuses an install missing it, with the same
rationale already stated for `WorldArchive` — the failure surfaces before the player reaches the
shop, not during it.

`shopMoviesMerchant = moviesPrefix + "shopanim/pose2-3/1.bmp"` is the one path this build reads out
of `MoviesArchive`. `loadShopArt` (`pkg/game/shopart.go`) reads it through `loadShopBMPAddr`, which
`loadShopBMP` (every other shop bitmap, all still under `graphicsPrefix`) now delegates to.
`ui.ShopScreenArt.Merchant` carries the decoded `*image.RGBA`, alpha-keyed on pure black exactly as
every other shop bitmap is. `ComposeShopScreen` (`pkg/ui/shopscreen.go`) blits it at
`shopMerchantRect.Min`, after the arrow blits, before nothing else changes about the merchant panel:
`shopMerchantRect` still doubles as the control rectangle and the Offer outline.

A `nil` `Merchant` (no archive, or the file absent or undecodable) draws nothing, matching every
other optional shop picture's contract.

## FR-2 — the four shelf regions draw their first animation frame

`shopShelfAnimFiles` names folder `4 - i`'s own `1.bmp` for shelf index `i` (0..3), reversed
against `ui.shopShelfDrawRects`'s own left-to-right order: `shopanim/04/1.bmp`, `shopanim/03/1.bmp`,
`shopanim/02/1.bmp`, `shopanim/01/1.bmp`. `loadShopArt` reads all four through `loadShopBMP`
(`graphics.res`, same archive and prefix as every other interface bitmap) into
`ui.ShopScreenArt.ShelfAnim[4]`. `ComposeShopScreen` blits each into its own
`shopShelfDrawRects[i]`, before the arrow and merchant blits.

Frames 2 through 11 of each folder are never read. No claim states a frame rate or an advance
rule for this animation (`DIV-127`); inventing one would author a cadence research has not
measured, so the shelf is drawn as a static first frame rather than animated.

## FR-3 — a map region click selects and activates nothing

`(*townScreen).WorldMapClick` (`pkg/game/worldmap.go`) tests only `ui.WorldMapCardAt` against the
click point. A hit on the town card calls `t.atSquare()`; a hit on the currently selected mission
card opens it; a hit on a different mission card selects it. A click that lands on neither a card
nor the town scroll — including a click inside a map region's own rectangle — returns
`ui.TownAction{}` and calls neither `t.selectWorldMission` nor `t.openWorldMission`.

`(*townScreen).WorldMapHover` is unchanged: it still calls `ui.WorldMapRegionAt` for hover
highlighting. `TOWN-118` condemns the region click's own former selection/cycling effect, not
hit-testing a region at all — hover is a presentation-only read with no selection or activation
consequence, and stays.

## FR-4 — cut at adversarial review

A `WeaponMaterialized` migration on decode from a pre-`WeaponHistoryKnown` save was built and
reverted in full (P finding, 2026-08-18): a blanket per-member raise of the latch also disarms a
member who never triggered the starting-weapon fallback. `ResolveEquipmentLoadout`
(`pkg/mapload/loadout.go:67-68`) reads an empty slot 1 with the latch unraised as that member's
ordinary state; raising it flips a correct `false` into a wrong `true`, and `StartMission`'s mint
loop bakes the resulting bare loadout's combat fields (`DamageBase`, `DamageSpread`, `ToHit`,
`Defence`, `Absorption`, `AlwaysHits`, `WeaponSpell`, `Reach`) into the `sim.Entity` at the next
mission's construction — hashed simulation state. A save alone cannot tell that member apart from
the one `DIV-112` is about, whose fallback DID resolve and was then disposed of, since both leave
slot 1 empty the same way.

`pkg/game/save.go` and `pkg/game/resume.go` are unchanged from master. `docs/DIVERGENCES.md`'s
`DIV-112` row is restated with this finding and stays `OPEN`. See `closure.md`'s Behaviour evidence
section for what was tried and the mutation-kill evidence that exposed it.

## FR-5 — `DIV-106` and `DIV-128`

`DIV-106`'s own claim (`TOWN-040`'s original two-availability-arm reading, with both writers
unlocated) is retracted, superseded by `TOWN-123` and `TOWN-040`'s own amended clause. The
corrected mechanism is the map-rectangle hit-test's own returned table value (`TOWN-039`), gated on
a `Picture == "nothing"` per-object bit and a persisted marker-selection cache — not an
availability test, and not a producer of which missions this build's world map exposes. That is
`Town.Available()`'s own responsibility, already correctly implemented and unaffected by this
story. `DIV-106` closes to `docs/DIVERGENCES-CLOSED.md`.

Re-reading the corrected claims against this build's own `worldMapAssets.marker`
(`pkg/game/worldmap.go`) surfaces a second, distinct mismatch: `marker` draws every
`Town.Available()` mission's own picture-bearing marker unconditionally, once its `Picture` decodes
and is not `"nothing"`. `TOWN-123`'s corrected mechanism gates ROM1's own marker paint on the same
persisted cache — a picture-bearing mission's marker paints only after the player has selected it
at least once. This build has no equivalent persisted "selected once" state. Recorded as `DIV-128`;
no code changes for it in this story (a session-state seam is out of scope).

## Out of scope

The merchant's animated Yes/No poses; the shelf animation's cadence and advance rule; the shop tip
widget; the world map's exact scroll-card geometry, card count and keyboard route; a session-state
seam gating world-map marker paint on per-mission selection history; a `WeaponMaterialized`
migration for a pre-`WeaponHistoryKnown` save (FR-4, cut — the disposed-versus-never-materialized
case is not resolvable from a save's own present state).
