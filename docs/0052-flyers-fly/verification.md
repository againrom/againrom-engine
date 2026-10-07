# Verification — 0052-flyers-fly

Environment: Go toolchain pinned in `go.mod`, Windows 11. Automated evidence is
`go test` with no game present. Install evidence comes from the two preserved
lawful roots, English and Russian, through developer runs; no test reads either.

## The gate

Run as one `&&`-chain, redirected outside the repository:

```
(go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
 sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
 sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))")
EXIT=0
```

Every package ok, `check-no-game-assets: clean (tree scan)`, the document budget
clean, and the audit's FAIL set **empty**. `-trimpath` is on every test run: without
it Windows Defender quarantines one test binary.

## AC by AC

| AC | Criterion | Result | Evidence |
|---|---|---|---|
| **AC-1** | SC-1 | pass | `TestTheDomainColumnDecidesTheMoverSDomain` — five placements over columns 1, 2, 3, an empty cell and 4; domains ground, ghost, air, ground, ground. The wants are the simulation's constants, not the column's numbers. |
| **AC-2** | SC-2 | pass | `TestOnlyAMatchedUnitsEntryCanYieldANonGroundMover` — npc, server-id, humans and an unmatched units key are all ground and all keep the provisional pair, while the table's one units row carries the air column. |
| **AC-3** | SC-3 | pass | `TestTheSingleArgumentEntryPointIsUnmoved`, unchanged from before this story: digest `0x6d7c57873172163c`, byte form 5125 bytes, entities equal field for field, and the two entry points agreeing over a nil table. Both literals were recorded before the definition table reached the package and neither moved. |
| **AC-4** | SC-4 | pass | `TestTheBandStopsAWalkerAndNeitherOtherDomain` — a three-cell band of water across the whole map; ghost and air arrive, the ground mover is asserted off the band **at every one of 300 ticks** and never reaches the far side. |
| **AC-5** | SC-5 | pass | `TestAGhostContendsWithAWalkerAndAFlyerDoesNot` — a ghost and a walker sent to one cell end on distinct cells; two flyers on crossing paths share a cell at some tick and both still arrive. |
| **AC-6** | SC-6 | pass | `TestEveryWayTheTableCanFailIsAFailure` (no table in the archive, bytes that will not walk, an empty entry) plus the archive's own missing case in `TestOpenArchives` and the startup case in `cmd/againrom`'s `TestStartupFailures`. Each names what failed; none returns a table beside its error. |
| **AC-7** | SC-7 | pass | `TestTwoWorldsFromOneMapAndOneTableStayIdentical` — two worlds from one map and table, stepped in lockstep for 40 ticks, hash-compared **at every tick**, then byte-compared. |
| **AC-8** | SC-9 | pass | The tool run below, on both roots. |
| **AC-9** | SC-10 | **partial — see the limitation below** | Driven headless through the shipped load path on both roots; the windowed run is the owner's. |
| **AC-10** | SC-8 | pass | `TestTwoClassesTakeTheirOwnUnscaledMaxima` — 37/37 and 211/211, distinct from each other and from the provisional 100. Also end to end in `TestTheTableReachesTheWorldTheApplicationOpens`. |

**SC-11** is the gate above: it exits 0 with an empty FAIL set, and
`internal/archtest` is among the packages that pass, so the import graph refuses
exactly the edges it refused before — this story widened no allow-map row.

Properties: **P-1** and **P-2** are witnessed where AC-1, AC-2 and AC-7 read;
**P-3** by AC-6, which asserts no map is listed and no world built; **P-4** by
AC-3's two frozen literals.

## The developer run — AC-8, both roots

`classdump -databin <world.res> Waters.alm`, English root:

```
world/data/data.bin: 88327 of 88327 byte(s) consumed, 0 left over
Waters.alm: 273 placement(s) at difficulty 2
  arm npc       taken     0  reached an entry     0
  arm server-id taken    48  reached an entry    48
  arm humans    taken     0  reached an entry     0
  arm units     taken   225  reached an entry   225
  domain ground     225
  domain ghost       41
  domain air          7
  domain total      273 of 273 placement(s)
```

The Russian root prints the same eight lines, figure for figure.

**The cross-check that would have failed if the join were wrong.** Every one of
the 48 non-ground placements reaches an entry in `{88, 89, 90, 92…99, 112, 113}`,
and no placement outside that set is non-ground:

```
  1 entry  88 air     3 entry  92 ghost    7 entry  96 ghost    1 entry 112 air
  3 entry  89 air     2 entry  93 ghost   13 entry  97 ghost    1 entry 113 air
  1 entry  90 air     2 entry  94 ghost    6 entry  98 ghost
                      5 entry  95 ghost    3 entry  99 ghost
```

Those indices are exactly the table's own non-ground rows — the four tiers each
of the two classes on the middle code and the two on the air code — with the
tiers this map does not place simply absent. A join that read the wrong column,
or read it off the wrong entry, could not produce that partition.

## The deliverable — AC-9

`againrom.exe -assets <root> -check` exits **0** on both roots (38 map rows
English, 34 Russian) and exits **1** on a root carrying the other three archives
but not `world.res`, naming that path.

Driven headless through the shipped path — `game.LoadTable` over the four-archive
set, then `mapload.FromALMWith` — on `Waters.alm`, identically on both roots:

```
order  id=260  domain=1 hp=10/10  from ( 68, 99) -> ( 54, 94)   the Bee
order  id=177  domain=0 hp=60/60  from ( 69, 94) -> ( 54, 94)   the Turtle
after 400 ticks:
  id=260  domain=1  now ( 54, 94)  arrived=true
  id=177  domain=0  now ( 69, 94)  arrived=false
```

Run against the same map with the domain assignment removed, the Bee does not
move either — which is the state the owner has been reporting.

The two health figures are the second half of what the table now delivers: both
units were born at 100/100 before, because the running game built its worlds
without ever reading the table.

**Limitation, stated rather than papered over.** The windowed run was not made:
it needs a display this environment does not have. What is verified is the load
path, the domains, the health and the movement the application drives — not the
pixels. The build note carries the map, the two cells and the two orders.

## Green-but-hollow audit

The domain mapping was replaced with one that answers **ground for every
column**, and the suite re-run. All four of this story's map-loading tests fail
under that mutant:

```
--- FAIL: TestTheBandStopsAWalkerAndNeitherOtherDomain
--- FAIL: TestAGhostContendsWithAWalkerAndAFlyerDoesNot
--- FAIL: TestTheDomainColumnDecidesTheMoverSDomain
--- FAIL: TestOnlyAMatchedUnitsEntryCanYieldANonGroundMover
```

So none of them passes on its fixture alone. The tool's own fixture was likewise
given a non-ground row: with every placement on the ground it could not have told
a column that is read from one that is not.

## Two findings, recorded rather than fixed

**The definition-table parser refuses an empty one-based collection at the end of
a stream.** Its guard compares the count word against the bytes remaining, where
for a one-based collection the entries written are the count minus one — so a
count of 1, meaning no entries, is refused when no bytes are left to satisfy it.
Every shipped collection has entries, so only a synthetic stream reaches it. The
fixture writer works around it by emitting 0 for an empty one-based collection,
documented at the field. Not fixed here: it is another story's contract.

**The two roots' tables differ in bytes and not in anything consumed.** Their
`data.bin` nodes are both 88327 bytes with different digests, and their Units
collections agree across all 119 entries, name for name and parameter for
parameter; all 56 named entries yield definitions agreeing on every column a
world carries. One map builds worlds that hash equal from either root. The
difference lies in collections nothing here reads.

## What was not verified

- The windowed run, above.
- **Speed** is untouched by design: the movement-rate law is not published, so
  nothing was wired to it and there is nothing to verify.
- Every definition column but the health pair and the domain remains loaded and
  read by nothing, so no evidence is offered for any of them.
- A ghost's behaviour against static objects. This build's plane carries no term
  for one, which the contract discloses; nothing here measures a gap it does not
  implement.
