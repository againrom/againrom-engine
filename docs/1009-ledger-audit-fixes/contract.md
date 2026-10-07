# 1009 — ledger audit fixes

## Result

Five items of cheap fidelity debt, found by a divergence-ledger audit on 2026-08-17, are closed or
correctly restated.

1. **Shop merchant.** The shop screen draws the merchant's own static picture,
   `movies\shopanim\Pose2-3\1.bmp` (`SHOP-MERCHANT-046`), at (277,112). `movies.res` becomes a
   required archive, for the same reason `world.res` already is: an install missing it should say
   so before the player opens a shop, not after.
2. **Shop shelf art.** Each of the four shelf regions draws folder `4 - i`'s own first frame out of
   `graphics.res` (`SHOP-SHELF-047`). No claim states a frame rate, so the frame is drawn static
   rather than animated at an authored cadence; the missing rate is a new `UNKNOWN` row
   (`DIV-127`), not an invented number.
3. **World map region click.** A mission is selected only through its scroll card
   (`TOWN-118`, High). `WorldMapClick` (`pkg/game/worldmap.go`) no longer selects or cycles a
   mission when a click lands on a map region; a region click reaches neither selection nor route
   construction. Region hover highlighting, which `TOWN-118` does not condemn, is unchanged.
4. **`WeaponMaterialized` migration — cut at adversarial review.** A pre-`WeaponHistoryKnown`
   migration was built and then reverted (P finding, 2026-08-18): a blanket raise of every
   member's latch also disarms a member who never triggered the fallback. `ResolveEquipmentLoadout`
   (`pkg/mapload/loadout.go:67-68`) reads an empty slot 1 with the latch unraised as that member's
   ORDINARY state; raising it flips a correct `false` into a wrong `true`, and `StartMission`'s
   mint loop bakes the resulting bare loadout's combat fields into the `sim.Entity` at the next
   mission's construction — hashed simulation state. A save alone cannot tell that member apart
   from the one `DIV-112` is about, whose fallback DID resolve and was then disposed of, since both
   leave slot 1 empty the same way. `DIV-112` stays `OPEN`, restated with this finding.
5. **`DIV-106` settled.** The row's own claim, `TOWN-040`'s original two-availability-arm reading,
   is retracted and superseded by `TOWN-123`. The corrected mechanism is a map-rectangle
   hit-test's returned table value, gated on a `Picture == "nothing"` bit and a persisted
   marker-selection cache — not an availability test, and not a producer of which missions the
   world map exposes (that is `Town.Available()`, already correct). `DIV-106` closes. Re-reading
   the corrected claims surfaces a distinct, second mismatch — this build's own marker painting has
   no equivalent to the marker-selection cache — recorded as `DIV-128`.

## Observable

A shop screen shows the merchant standing in his own picture and each shelf showing its first
animation frame, where both previously drew nothing. Clicking a map region on the world map screen
no longer selects or cycles a mission; only clicking a scroll card does. Loading an old save is
unchanged from master: item 4's migration is cut (see Result, item 4).

## Claims and what is `UNKNOWN`

- `SHOP-MERCHANT-046` — the merchant's static picture, path and anchor.
- `SHOP-SHELF-047` — the four shelf regions, the `4 - i` folder mapping, eleven shipped frames a
  folder. States no frame rate or advance rule (`DIV-127`, new `UNKNOWN` row).
- `TOWN-118` (High) — mission selection is scroll-card-only; a map-rectangle click hit-tests a
  region and returns a text-table value, calling neither selection nor route construction.
- `TOWN-040` (active clause, amended) and `TOWN-123` (High) — the map-rectangle hit-test's
  returned value is gated on a `Picture == "nothing"` bit and a persisted marker-selection cache;
  the cache is also read by world-map paint. What the returned value itself drives beyond that is
  `TOWN-039`'s own open Medium-confidence question, unaffected by this story.
- `WeaponMaterialized`'s proposed migration is authored state; no claim covers it. It was built and
  cut at the adversarial review (item 4); `DIV-112` stays `OPEN`, restated with that finding.

## Domains

- **Assets** — `movies.res` becomes a required archive; `pkg/game/archives.go`.
- **Town & Economy** — shop art loading (`pkg/game/shopart.go`); world map click and
  region/selection behaviour (`pkg/game/worldmap.go`).
- **Client** — shop screen composition draws the two new pictures
  (`pkg/ui/shopscreen.go`).

Three domains, three landed behaviours. A fourth behaviour (item 4, `WeaponMaterialized`
migration) touched Party, Items & Heroes and Persistence, and was cut in full at the adversarial
review: it reaches hashed simulation state through `StartMission`'s mint loop, and does so
incorrectly for a member who never triggered the fallback. See item 4 above and `closure.md`.

## Scope limits

Out of scope: the merchant's animated Yes/No poses (`DIV-020` stays `OPEN`); the shelf animation's
cadence and the shop tip widget (`DIV-018` stays `OPEN`); the world map's exact scroll-card
geometry, card count and keyboard route (`DIV-105` stays `OPEN`, `UNKNOWN`); a session-state seam
recording per-mission selection history for world-map marker paint (`DIV-128`, new row, stays
`OPEN`); a `WeaponMaterialized` migration for a pre-existing save (item 4, cut at review — see
`DIV-112`). No `saveVersion` or simulation `formatVersion` bump for items 1-3: neither touches
`pkg/game/save.go` or the envelope.

`DIV-127` and `DIV-128` are allocated. Both are spent: `DIV-127` for the shelf animation cadence,
`DIV-128` for the marker-paint/selection-history gap. `DIV-106` closes; `DIV-112` stays `OPEN`,
restated with the review's finding (item 4 cut).
