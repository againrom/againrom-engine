# Plan — carrying one column across two tiers

## Approach

One column already sits on a loaded definition and one field on an entity; the work is a wire between
them and a path to the screen for it to sit on. The map-loading tier gains a total mapping from the
column to the simulation's domain and applies it at the single statement that already turns a
resolved placement into an entity, beside the health it takes from the same resolution (FR-1, FR-2).
The front-end gains the archive holding the table, loads it once at startup as it loads its two art
bundles, and builds every opened map through the resolving entry point at the identity difficulty
(FR-5, FR-6, FR-7). The verb that already resolves placements prints the domain each entity ended up
in (FR-8). Nothing in the simulation moves: FR-4 is demonstrated, not written. FR-3 is discharged by
leaving the table-free entry point alone and pinning it — it is defined in terms of the resolving
one with a nil table, so the pin is a claim about the nil path through the code this story edits.

## Facts verified during planning

Baseline, from reading the tree and running its own loader over both lawful installs.

1. **No file outside `pkg/sim` names a movement domain**, tests included. `data.UnitDef.MovementType`
   is written by the slot walk and read nowhere. `sim.Domain` carries three methods, all unexported,
   and **no `String`** — nothing outside that package can name its values.
2. **`FromALMWith` builds every entity in one composite literal**, taking `HP`/`MaxHP` from the
   resolution's definition when it resolved and from `SpawnHP` when it did not. `FromALM` is defined
   in terms of it with a nil table, so there is one world-building implementation and not two. An
   **unresolved** placement yields the **zero** definition, whose column is 0 — not the
   constructor's 1, which only `NewUnitDef` produces.
3. **`openMapWorld` calls `FromALM`** and returns no error; `loadMap` above it returns one and wraps
   two failures with the map's name. It is **not the only** caller of the table-free entry point:
   `UnitCensus` calls it too, and a developer tool reaches that census.
4. **`OpenArchives` opens a fixed list of three hosts** and returns the filesystem's own
   `open <path>: <err>`; which of two missing archives is reported is pinned by its own test.
   `NewFrontEnd` calls it first, then the menu assets and the two art bundles, **and then scans and
   builds the map list before returning**.
5. **The table is in none of those three.** Both roots carry a further archive holding it, whose
   identity segment resolves under either root's spelling because the filesystem folds a stem.
6. **The two roots' tables are not byte-identical, and their Units collection is.** All 119 entries
   agree name for name and parameter for parameter, and all 56 named entries yield definitions
   agreeing on every column a world carries; the byte difference lies elsewhere in the file.
7. **Every shipped map builds through the resolving entry point without a refusal**, on both roots
   at all three difficulty values, and one map's worlds hash equal across roots. **No named Units
   entry falls short of the streamed slots**, so that refusal is unreachable from either table.
8. **The derived plane sets bit 0 and bit 1 and no other**, and every domain's terrain term is one
   of those two.
9. **`pkg/game` may import every `pkg/` package**; `cmd/classdump` may not import the simulation,
   and no test pins its allow-map row.
10. **`pkg/mapload`'s suite already builds synthetic collections and rows by slot** through a
    three-argument row helper, and its routing tests build and step a world. **No package can write
    a parseable definition table.** The command's own startup test writes exactly the three
    archives, and its missing-archive table enumerates them.

## Files to touch

| Path | Intent | Why |
|---|---|---|
| `pkg/mapload/spawn.go` | MODIFY | the mapping (FR-1) |
| `pkg/mapload/fromalm.go` | MODIFY | the assignment (FR-2) |
| `pkg/mapload/spawn_test.go` | MODIFY | the mapping's cases; the row helper gains the column, so its callers move (AC-1) |
| `pkg/mapload/fromalm_test.go` | MODIFY | AC-2, AC-3, AC-7, AC-10, P-1, P-2, P-4 |
| `pkg/mapload/domain_test.go` | ADD | the crossing and the contention, world-built and stepped (AC-4, AC-5) |
| `internal/synth/databin.go` + `_test.go` | ADD | a synthetic table on the wire (R-5) |
| `pkg/game/archives.go` | MODIFY | the fourth archive and its address prefix (FR-6) |
| `pkg/game/archives_test.go`, `address_census_test.go` | MODIFY | that archive in their fixtures and in the missing set (AC-6, P-3) |
| `pkg/game/table.go` + `table_test.go` | ADD | the load and its three failure shapes (FR-5, FR-6, AC-6, P-3) |
| `pkg/game/frontend.go` | MODIFY | the table on the front-end, before the map list (FR-5, FR-6) |
| `pkg/game/frontend_test.go` | MODIFY | construction with no readable table (AC-6) |
| `pkg/game/world.go`, `world_test.go` | MODIFY | the resolving entry point behind the map open, its difficulty and its failure (FR-5, FR-7) |
| `cmd/classdump/databin.go` + `main.go` | MODIFY | the per-placement domain and the per-map census (FR-8) |
| `cmd/againrom/main_test.go` | MODIFY | the install fixture gains the archive and a table; the missing set gains a row |
| `cmd/againrom/main.go` | MODIFY | what startup requires |

**Not touched, though a reader would expect them:** the allow-map and the architecture table (DD-2),
`pkg/sim` (DD-1, DD-3), `pkg/game/census.go` (DD-10).

## Design decisions

- **DD-1 — the mapping is a total, unexported function in the map-loading tier, taking a resolved
  definition, with an explicit default arm.** It is the tier holding both types: the data tier may
  not import the simulation, and the simulation reads no table. *Rejected:* a lookup keyed by the
  column value — it would answer for a key it does not hold with the same silence as for one it
  does, making the contract's unknown-value arm indistinguishable from a hit.

- **DD-2 — the report reads each placement's domain off the world it already builds**, not off a
  second call to the mapping. The verb's own rule is that the health it prints comes from the world
  rather than from a second application of the arithmetic, because a report that recomputes a number
  agrees with itself instead of with what a player's world holds; the domain takes the same route.
  Nothing therefore needs exporting and the allow-map does not move. *Rejected:* exporting the
  mapping and calling it from the tool — it reverses that rule in the one file that states it, and
  makes the report a witness of the mapping rather than of the world.

- **DD-3 — the tool names the three domains from a fixed table indexed by the domain's numeric
  value**, which the world constructor makes total over what can arrive. *Rejected:* a naming method
  on the simulation's type — a presentation concern is a poor reason to open a package this story
  otherwise leaves alone.

- **DD-4 — the domain is written in the entity-building literal, and an unresolved placement is
  carried by the mapping's default rather than by a branch.** Its zero definition carries column 0,
  which is not one of the three, so the arm the contract already requires for an unknown value is
  what makes it ground, and no second condition exists to disagree with the health's. *Rejected:* a
  branch beside the health's — two conditions on one fact.

- **DD-5 — the table's archive joins the fixed host list at its end**, its name beside the other
  three and its address prefix beside the graphics container's. The order decides which failure a
  doubly-broken install reports, and every pinned pair today is a pair of the existing three:
  appending changes none, inserting rewrites pinned behaviour for no gain. *Rejected:* a handle of
  its own — a second handle onto a container the set covers.

- **DD-6 — the table is loaded once, in construction, immediately after the archives and before the
  map list, and every way that load can fail fails construction.** Before the list, so a root that
  cannot yield a table lists no map; once, as the art bundles are. *Rejected:* loading it in the
  map-open path — the failure would move from startup to the first map. *Also rejected:* treating an
  unreadable table as "no table" while keeping the archive's absence fatal — the archive check would
  then guard nothing.

- **DD-7 — a nil table on a hand-assembled front-end means "no table", as a nil art bundle means "no
  art".** Construction either fills it or fails, so no route an *install* can take reaches a map open
  without one; what remains is test code. *Rejected:* requiring it non-nil — every such test would
  then build a table it never reads.

- **DD-8 — the map-open path gains the table and a named difficulty constant and returns the
  resolving entry point's error, wrapped with the map's name as the two loads before it are.** The
  constant sits there rather than on the front-end: a field is settable, and a settable value is a
  setting. *Rejected:* falling back to the table-free build on that error — a refused table would
  silently restore the ground world this story exists to remove.

- **DD-10 — the unit census stays on the table-free entry point.** It counts how many placements
  have art, keyed by class id alone, and reads no stat, domain or health, so a table would change
  none of its numbers while giving a developer census a new way to fail.

- **DD-11 — the synthetic table writer lives in the shared fixture package**, and lands before the
  archive becomes required. Three test binaries need a table that parses; an encoder hand-rolled in
  each is three copies of one wire format.

- **DD-12 — the table's entry path is spelled once per tier that opens an archive for itself.** An
  address prefix is spelt beside the archive name it belongs to, and the tool and the front-end open
  different archives. *Rejected:* one shared constant — it would live in a tier that opens nothing.

## Risks

- **R-1 — a shipped map that opens today stops opening.** The resolving entry point refuses an entry
  whose damage selector takes an unmodelled arm, and that refusal now reaches the map-open path
  rather than a developer tool. *Mitigation:* measured unreachable from both shipped tables — no
  named entry is even wide enough to be built — and every shipped map was built at all three
  difficulty values on both roots without one. The failure names the entry.

- **R-2 — an install missing the table's archive stops working.** *Mitigation:* both lawful roots
  carry it; the refusal names the path at startup, before a window opens, as the other three do; and
  the alternative is the defect returning silently.

- **R-3 — a world under the game screen is no longer a function of the map alone** but of the map
  and an installed, per-locale file, and nothing records which file built it; two roots could
  disagree on a column that now reaches the digest. *Mitigation:* facts 6 and 7 — a corpus fact
  rather than a guarantee, and this risk rests on it.

- **R-4 — the deliverable shows a ghost passing a tree, which the game does not do.** *Mitigation:*
  disclosed in the contract; the demonstration is staged on water, where this build and the game
  agree, and on the classes the owner reports crossing.

- **R-5 — the fourth archive invalidates every existing install fixture at once**, in three test
  binaries, and a fixture that merely exists is not enough where the table must parse.
  *Mitigation:* DD-11's writer lands first, so no commit has a red suite for want of a fixture.

## Success criteria

| # | Condition | How it is judged |
|---|---|---|
| **SC-1** | The column's five cases each produce the domain FR-1 names. | automated |
| **SC-2** | No arm but a matched units entry produces a non-ground mover, and every arm produces one domain. | automated |
| **SC-3** | A world built with no table is field-, byte- and digest-identical to the one that map built before. | automated |
| **SC-4** | A non-ground mover crosses a band of ground-blocking cells and a ground mover does not. | automated |
| **SC-5** | A ghost and a ground mover do not share a resting cell; two air movers share one while moving. | automated |
| **SC-6** | Each of a missing archive, one with no table, and a table that will not parse fails construction, naming what failed, with no map listed or opened. | automated |
| **SC-7** | Two worlds from one map and one table hash equal at every tick and marshal equal. | automated |
| **SC-8** | Two classes with different health maxima take their own, unscaled, and neither the provisional constant. | automated |
| **SC-9** | On a lawful install every placement of a shipped map reports a domain, the counts sum to the placement count, and the non-ground counts fall on the table's own non-ground rows. | developer-run, **both roots** |
| **SC-10** | In the built application, a non-ground unit ordered across water crosses it and a ground unit ordered at the same water does not. | manual, from the owner's seat |
| **SC-11** | The gate is clean, and the import graph refuses exactly the edges it refused before. | automated |
