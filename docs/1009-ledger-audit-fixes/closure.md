# 1009 — ledger audit fixes: closure

## Result

`movies.res` is a required archive and carries the shop merchant's static picture, drawn at
(277,112). Each of the four shop shelf regions draws its own folder's first animation frame out of
`graphics.res`. A click on a world map region no longer selects or cycles a mission; only a scroll
card does. `DIV-106` is closed as leaning on a retracted claim, and re-reading the corrected claims
surfaced one new, distinct row, `DIV-128`.

A fifth item, a `WeaponMaterialized` migration for a pre-`WeaponHistoryKnown` save, was built and
reverted in full at the adversarial review (P finding, 2026-08-18): `ResolveEquipmentLoadout`
(`pkg/mapload/loadout.go:67-68`) reads an empty weapon slot with the latch unraised as the
ordinary state of a member who never triggered the starting-weapon fallback, and a blanket
per-member raise flips that correct `false` into a wrong `true` for such a member, reaching
`sim.Entity`'s combat fields through `StartMission`'s mint loop at the next mission's
construction. A save alone cannot separate that member from the one `DIV-112` is about, whose
fallback resolved and was then disposed of, since both leave the slot empty the same way.
`pkg/game/save.go`, `pkg/game/resume.go` and `pkg/game/save_test.go` are byte-identical to master.
`DIV-112` stays `OPEN`, restated with this finding.

## Twelve-aspect matrix

| Aspect | Scope |
|---|---|
| Data | `movies.res` opened as a fifth required archive; two new decoded picture addresses (`shopMoviesMerchant`, `shopShelfAnimFiles[4]`). PASS |
| Runtime state | `ShopScreenArt.Merchant` and `.ShelfAnim[4]`; `worldMapState` unchanged in shape, only `WorldMapClick`'s branch removed. PASS |
| Simulation | Not touched. `pkg/sim` unchanged. A `WeaponMaterialized` migration was considered for item 4 and cut because it reaches hashed simulation state through `StartMission`'s mint loop (Behaviour evidence, item 4). N/A |
| Player input | `WorldMapClick`'s region branch removed; region hover unchanged. PASS |
| AI | Not touched. N/A |
| UI/HUD | Shop screen composition draws two new pictures. PASS |
| Triggers/scripts | Not touched; the mission 10/mission 20 UNSUPPORTED trace count is unchanged (below). N/A |
| Inventory/equipment | Not touched. Item 4's migration is cut; `pkg/mapload` is unchanged from master. N/A |
| Persistence/save-load | Not touched. Item 4's migration is cut; `pkg/game/save.go` and `pkg/game/resume.go` are byte-identical to master, no new gob field, no envelope or simulation-form version change. N/A |
| Campaign/session | World map mission selection and availability unaffected; `Town.Available()` remains the sole producer of exposed missions. PASS |
| Shipped content | `shopdump` and `worldmapcheck` run against the real EN install (below). PASS |
| Interactions with existing mechanics | `materializeStartingWeapon` remains the latch's one writer, unchanged from master (`TestWeaponMaterializedHasOneWriter` passes unmodified). PASS |

No aspect is a known in-scope GAP. Item 4 is a cut behaviour, not a silent gap: `DIV-112` stays
`OPEN` and is restated with the finding that cut it.

## Behaviour evidence

Each behaviour names one reverted line and the test that reddens when it is reverted. Every kill
below was re-run in this pass: mutate, confirm red, restore, confirm green.

1. **Merchant and shelf art draw** (FR-1, FR-2). Reverted: the four-line block in
   `ComposeShopScreen` (`pkg/ui/shopscreen.go`) blitting `art.ShelfAnim[i]` and `art.Merchant`.
   Test: `TestComposeShopScreenDrawsTheMerchantAndTheShelfAnimations`
   (`pkg/ui/shopscreen_test.go:406`) — reddened with `merchant corner = ... want the art's own
   picture` and four `shelf N corner = ... want its own folder's first frame` failures.
2. **Merchant/shelf loading, folder mapping** (FR-1, FR-2). Covered separately at the loader:
   `TestLoadShopArtReadsTheMerchantOutOfMoviesArchive` (`pkg/game/shopart_test.go:65`) and
   `TestLoadShopArtReadsTheShelfAnimationsFolderFourMinusI` (`pkg/game/shopart_test.go:77`), the
   second mutation-killed by reversing `shopShelfAnimFiles`'s folder order.
   `movies.res` required: `TestOpenArchives` (`pkg/game/archives_test.go:63`), subtest "movies.res
   is required even though the menu never reads it" (`archives_test.go:123`).
3. **World map region click** (FR-3). Reverted: `WorldMapClick`
   (`pkg/game/worldmap.go:411`) restored to its former shape, adding back an
   `if i, ok := ui.WorldMapRegionAt(v, p); ok { t.selectWorldMission(i) }` branch after the card
   branch. Test: `TestWorldMapRegionClickSelectsAndOpensNothing`
   (`pkg/game/worldmap_test.go:208`) — reddened with `a click inside the mission's own map region
   selected 0, want none`.
4. **`WeaponMaterialized` migration** (formerly FR-4). Cut in full at the adversarial review
   (P finding, 2026-08-18), not reverted for a defect found by mutation testing in this lane:
   `Snapshot.WeaponHistoryKnown` and the top-of-function migration block in `prepareRestore`
   (`pkg/game/resume.go`) are removed. `git diff master -- pkg/game/save.go pkg/game/resume.go
   pkg/game/save_test.go` is empty. `TestOldSaveWeaponHistoryMigrationSuppressesTheStartingWeaponFallback`
   and its `weaponHistoryParty` fixture are removed with it; `currentReleasedSaveFixtureSHA256`
   reverts to master's own value, `4e34bef57fab8c86f5222e34449989f06e2290cbeae5b207b7a31bf346413341`.
   `TestWeaponMaterializedHasOneWriter` (`pkg/game/weaponlatch_scan_test.go`) passes unmodified,
   confirming the revert left no direct assignment or second writer behind. The reviewer's own
   counterexample is the mutation-kill evidence for why this behaviour is cut: the same member,
   constructed with a real starting weapon and every equipment slot empty, resolves
   `DamageBase=9 DamageSpread=3 ToHit=46 Reach=0` through `ResolveEquipmentLoadout` with the latch
   read `false` (his ordinary, correct state) and `DamageBase=0 DamageSpread=0 ToHit=6 Reach=1`
   with the latch forced `true` by the reverted migration — hashed simulation state, reached
   because a save alone cannot tell that member apart from one whose fallback genuinely resolved
   and was disposed of.
5. **`DIV-106`/`DIV-128`**. No code changed; this item is a ledger reconciliation. No mutation
   applies.

## Integration witness

Run against the real EN install (`gameversions/en`), outside any test:

```
shopdump -assets gameversions/en -png <tmp>
  black pixels: screen 24738/307200 (8%), buttons 6675/41888 (15%), pack 11212/43200 (25%), corner 97/38720 (0%)
```

The composed screen (`shop.png`, not committed — golden rule 1) shows the merchant standing in the
shop room at (277,112), where `DIV-020`'s row states the panel previously drew nothing. Overall
screen black coverage is 8%, down from `DIV-003`'s own recorded 9% after that earlier story's fix —
consistent with two more pictures now painting over what was black canvas, though this story draws
no separate before/after comparison of its own.

```
worldmapcheck -assets gameversions/en
  world map: chapter 30 accepted mission 30; 1 scroll(s); title 10 bytes; briefing 75 bytes;
  payment 1000; route 24 stamps; frame composed; mission opened
  world map sweep: 28 mappings, 28 title/briefing pairs, 15 main, 9 side and 4 mapping-only rows;
  26 building-offered; 5 picture and 23 fallback markers; frame composed; all consumed
```

Card-based selection and mission-open still work end to end against the real campaign after
`WorldMapClick`'s region branch was removed. `worldmapcheck` does not itself drive a region click;
that path is covered by `TestWorldMapRegionClickSelectsAndOpensNothing`'s direct construction (the
shared archive fixture's region rectangle sits inside the town card's own rectangle, so a test built
over it cannot reach the region-click code path at all — the reason this test constructs its
`townScreen`/`worldMapState` directly rather than through the shared fixture).

## Milestone census

`pipeline/milestone-baseline.txt` (master) carries no `cannot run` line for `en m10` or `en m20`;
both already run their full script node set:

```
en	m10	script 16 checks, 27 instants, 12 triggers
en	m20	script 14 checks, 15 instants, 11 triggers
```

Measured this story, `missionrun -mission <n> -trace -ticks 1 | grep -c UNSUPPORTED` against
`gameversions/en`:

```
mission 10: 0
mission 20: 0
```

Unchanged from master. This story does not touch `pkg/sim`'s script engine or trigger evaluation;
the census was expected to hold and does.

## Research reconciliation

- `SHOP-MERCHANT-046` — consumed. The merchant's static picture is drawn at its decoded path,
  size and anchor. The animated Yes/No poses it also names stay undrawn (`DIV-020` stays `OPEN`).
- `SHOP-SHELF-047` — partially consumed. The `4 - i` folder mapping and first-frame path are
  drawn; the eleven-frame cadence is not stated by any claim and is not invented (`DIV-127`, new
  `UNKNOWN` row). The tip widget (`DIV-018`) stays `OPEN`.
- `TOWN-118` (High) — consumed. `WorldMapClick`'s region branch is removed; selection is
  scroll-card-only.
- `TOWN-040` (amended, active clause) and `TOWN-123` (High) — read in full to settle `DIV-106`.
  Both confirm the map-rectangle hit-test's returned value is gated on a `Picture == "nothing"`
  bit and a persisted marker-selection cache, and that the cache is also read by world-map paint
  (`R1530`). `DIV-106` closed (`docs/DIVERGENCES-CLOSED.md`): its own premise, `TOWN-040`'s
  original two-availability-arm reading, is the retracted clause. `DIV-128` opened: this build's
  `worldMapAssets.marker` paints every available mission's picture-bearing marker unconditionally,
  with no equivalent to the corrected mechanism's selection-history gate. No code changes for
  `DIV-128` in this story; a session-state seam is out of scope.
- `WeaponMaterialized`/`WeaponHistoryKnown` — no claim covers either; both are authored state. The
  migration `DIV-112`'s own former revisit condition proposed was built and reverted at the
  adversarial review: it is unsound because a save alone cannot distinguish a member who never
  triggered the fallback from one who did and disposed of the result, and a blanket raise
  misreads the first case, reaching hashed simulation state through `StartMission`'s mint loop.
  `DIV-112` stays `OPEN` (`docs/DIVERGENCES.md`), restated with this finding and scoped to saves
  older than story `1005`, reachable in practice only through the town-only load path — a
  mission-mode save of that age is already refused on simulation-form mismatch (`DIV-095`).

`DIV-127` and `DIV-128` are spent, both allocated to this story. `DIV-106` closes. `DIV-112` stays
`OPEN`, restated. `DIV-018`, `DIV-020` and `DIV-105` stay `OPEN`, each with its `Implemented
behaviour` cell updated to state exactly what this story built and what remains out of scope.

## Gate

`go build ./...`, `go vet ./...`, `gofmt -l` and `go test -trimpath -count=1 ./...` all pass across
every package, including `pkg/game` and `pkg/ui`. `scripts/check-no-game-assets.sh` reports clean.
Re-run after item 4's revert, on the tree that landed items 1-3 and 5 and restated `DIV-112`.
`TestWeaponMaterializedHasOneWriter` (`pkg/game/weaponlatch_scan_test.go`) passes unmodified,
unchanged from master: `materializeStartingWeapon` remains the latch's one writer.
