# Tasks — the campaign mission start

Legend: **Kind** is `impl` (one commit, trailered) or `test` where a task's whole product is a test
file. `Done when:` is the entry's own exit condition.

## T1 the arms become a ladder, and the npc registry answers the first rung

Kind: impl. Carries FR-3, FR-4, DD-1, DD-2, DD-3.
Files: `pkg/data/npc.go` (+ test), `pkg/mapload/spawn.go`, `pkg/mapload/spawn_test.go`.

The npc lookup is a new `pkg/data` type built from a parsed registry; `mapload.Table` gains a field
for it and `Resolve` gains the rung that reads it. Sections are named `npc<n>` and the key is
`DataBinID`.

Scope fence: no other file changes, and `Resolve`'s signature, arm constants and `Resolution` do not
move. `definitionFor` still yields a definition on the Units arm alone.

Done when: a placement at or above the band resolves against Units whatever its flag word and
definition id; below it, the flag word beats the definition id and the definition id beats the type
id; a nil table and a nil npc lookup answer no entry on every arm; the existing `pkg/mapload` and
`pkg/game` tests still pass unchanged except where they pinned the old band, and each such change is
in the commit.

## T2 the drop cells, read out of the map's own trigger payload

Kind: impl. Carries FR-5, DD-4, DD-5, DD-6.
Files: `pkg/mapload/drop.go` (+ test).

One exported reader over `alm.Map`, answering the cells in node order. The payload is
`[u32 nAct][nAct × 796][u32 nCond][nCond × 796][u32 nTrg][nTrg × 184]`, the count word at `+0x00`
being the first of the three and already decoded by the format tier; within a node the opcode is at
`+0x40` and the ten stored values begin at `+0x4c`. The drop opcode is `0x10002`.

Scope fence: `pkg/formats/alm` is not touched, and no opcode but the drop one is given a meaning.

Done when: a synthetic payload carrying one drop node yields its cell; one carrying two yields both,
in order; a payload with residue, a truncated one, an absent trigger record and a count that
overruns the payload each yield none; and the two coordinates are the low bytes of the two values.

## T3 the draw a loader can make before a world exists

Kind: impl. Carries FR-6, DD-7.
Files: `pkg/sim/draw.go` (+ test).

A standalone, seeded sequence over the package's existing generator, exposing a uniform draw
**inclusive of its bound**. Nothing else in `pkg/sim` changes.

Scope fence: the world's own generator, its ownership and its seeding are untouched; no float, no
clock, no process-global source enters the package.

Done when: a bound of 0 always answers 0; a bound of n answers every value in `[0, n]` over enough
draws and never one outside it; two sequences from one seed agree draw for draw; the package's
determinism scan and its import check stay green.

## T4 the start: a cell is picked, and the party stands on it

Kind: impl. Carries FR-6, FR-7, FR-8, DD-8, DD-9, DD-10.
Files: `pkg/mapload/start.go` (+ test).

One exported entry building on `FromALMWith`: it picks a drop cell from T2's list with T3's draw,
falls back per axis when it must, walks the party outward from the cell over the same derived plane
the world routes on, and appends the party's entities after the map's. It answers the world and a
report carrying the drop cell, whether the fallback ran, and each member's cell.

Scope fence: `FromALM` and `FromALMWith` keep their contracts and their behaviour byte for byte; the
report is returned, never stored on the world.

Done when: the hero is on the drop cell; no two members share a cell and none stands where the plane
closes the ground; the exhaustion case is counted rather than silently overlapping; the draw order is
pick, then column, then row; two starts from one input agree on every cell and on the world's digest;
and a start with an empty party builds the world `FromALMWith` builds.

## T5 the campaign entry, and the registry it needs

Kind: impl. Carries FR-1, FR-2, DD-11, DD-12.
Files: `pkg/game/mission.go` (+ test), `pkg/game/table.go`.

The address derivation from a mission number, and the entry that reads the map at it, loads what T1
and T4 need, and answers the decoded map, the world and the report. `LoadTable` additionally reads
the campaign container's npc registry and fails on its absence.

Scope fence: no front-end changes; `cmd/againrom` and the map picker are untouched.

Done when: mission 10 addresses the campaign identity's `10.alm`; a non-positive number is refused
with no read attempted; a read or decode failure names the address; the entry opens nothing outside
the filesystem it is given; and the existing front-end tests pass with the widened `LoadTable`.

## T6 the census over a lawful install

Kind: impl. Carries FR-3, FR-5, DD-13.
Files: `cmd/classdump/campaign.go` (+ test), `cmd/classdump/main.go`.

A verb taking an asset root, opening the campaign and world containers through `pkg/vfs`, and
printing per map: the placement count, the four arm counts, and the drop cells. Totals at the end.

Scope fence: counts and cells only — no entry name, no string, no parameter value; the existing verbs
and their output do not change.

Done when: the verb runs against a lawful install, covers the campaign container's maps and the loose
maps under the root, and exits non-zero on a failure to open, read or walk any of them.

## T7 the npc arm's two routes, side by side

Kind: impl. Carries FR-4, DD-13.
Files: `cmd/classdump/campaign.go` (+ test), `cmd/classdump/main.go`.

The census verb takes an optional map name. Named one, it additionally prints one line per placement
of that map that takes the NPC arm: the subscript, the record's own definition id, the server id the
registry names for that subscript, and the entry each of the two reaches.

Scope fence: the census's own rows, totals and summary do not change; the added report is counts and
indices only, no entry name and no parameter value.

Done when: the verb accepts three arguments and refuses four; a named map that the corpus does not
hold is a failure naming it; a map with no NPC-arm placement prints the report's heading and no row;
and each row's two entry indices are read off the same searches the loader uses rather than
recomputed.
