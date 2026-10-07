# Story `1033` — the three check opcodes and the paired instant: as-built spec

Canonical at landing. Supersedes `contract.md`'s forward-looking claim summary where the two
disagree; `contract.md` records what was known before the work, this file records what was built.

## Domains touched

Campaign & Scripts (`pkg/sim/script.go`, `pkg/mapload/script.go`), Sim Core (`pkg/sim/structure.go`,
`pkg/sim/world.go`), Persistence (`pkg/sim/structurebinary.go`, `pkg/sim/binary.go`). Three, matching
`contract.md`'s prediction.

## B1 — check opcode 4 (`ScriptCheckHealth`)

Gated on the node's own first plain parameter (`c.Args[0]`) equal to `scriptCheckHealthGate` (6,
`pkg/sim/script.go`). On the gate, the register receives the referenced unit's `Entity.HP`, already a
signed `int32` carrying the value the original's sign-extended `unit+0x94` word represents
(`HERO-HEALTH-032`). Off the gate, the arm returns without writing the register at all — not zero, a
state indistinguishable to a later reader from "not yet measured."

An unresolved unit reference reaches the shared `ScriptSilenceNoUnit` path before this case runs, the
same silence every other check applies to an unresolved unit.

## B2 — check opcode 16 (`ScriptCheckItemDistance`)

Reuses check 17's own container search, `containerHolds` (`TRIG-ITEMTEST-040`), against the referenced
unit's carried container (`w.carried[i]`, looked up by `indexOfEntity`). A node naming no item, or
whose item is not found, writes the not-found sentinel `0xff`. A found item's distance is measured
**from the unit's own position** — never from the item, which is the reading `EXP-0080`'s uncited
vocabulary note gives and `TRIG-CHECK-052` refutes.

The two operands are read at DIFFERENT WIDTHS, which is `TRIG-CHECK-052`'s own reading of the
arm: the node's X and Y are byte loads (`MOV AL,byte ptr [ESI+0x8]`, `MOV DL,byte ptr [ESI+0xc]`)
and the unit's position is taken whole from its own position object (`[unit+0x10]`). The call is
therefore `scriptDistance(e.X, e.Y, c.Args[0]&0xff, c.Args[1]&0xff)`, and the result is masked
again through `scriptByte` because the arm masks before its store. Masking the parameters is the
same thing check 14's own arm does with its X/Y pair.

No shipped node reaches the truncation: `TRIG-CHECK-054` gives the widest authored X/Y in the
whole corpus as 116 and 130, both inside 8 bits, on both roots. This does not settle
`scriptDistance`'s own open byte-width clause, which asks whether the helper masks the
coordinates it subtracts; that stays Medium and untouched.

## B3 — check opcode 21 (`ScriptCheckStructField`) and instant opcode 26 (`ScriptInstantStructField`)

### Per-structure state added

`pkg/sim/structure.go` adds the minimum state the two arms share, over and only over the contract's
own boundary — identity, one 16-bit field, its serialization:

```go
type StructureID uint32

type Structure struct {
    ID      StructureID
    Field42 uint16
}
```

`Field42` is named for `structure+0x42`, the original's own offset (`TRIG-CHECK-053`), not for a
meaning no claim gives. This is the same naming convention `alm.Object`'s own `Field0E`/`Field12`
already use, for the same reason.

**No health, no destruction, no occupancy, no binding to `pkg/game/structures.go`'s art-layer
`StructureSet`.** A world carries one `Structure` per placed type-4 record on the map it was built
from, whether or not any script node ever names it — the entity list's own precedent, where every
type-6 record becomes an `Entity` whether or not a script reads it.

### Check 21

Unconditional: no `p0`/`p1` test. Reads `Structure.Field42`, sign-extends it (`int32(int16(st.Field42))`,
mirroring the arm's own `MOVSX`), and writes the register. A structure reference that did not resolve
at compile time reaches `ScriptSilenceNoStructure` rather than a panic — `TRIG-BIND-010` states no
shipped node reaches this path (every authored `Target_Structure` reference resolves against the
map's own structure table), so this is a safety net on the check-9 index guard's own precedent
(1029 DD-4), not a reachable path from shipped content.

### Instant 26

Unconditional: the low 16 bits of the node's own first plain parameter (`uint16(in.Args[0])`) replace
`Field42` on the referenced structure, found by `indexOfStructure`. No selector (unlike instant 34's
three-way `p0` dispatch) and no notification call after the write, matching the arm's own instruction
sequence. An unresolved structure reference does nothing, the same treatment every other instant arm
gives an unresolved reference.

### Wiring from map to simulation

- `pkg/mapload/fromalm.go`'s `Structures(m *alm.Map, t *Table) []sim.Structure` builds one
  `Structure` per `alm.Object` (type-4 record) in file order, `ID = StructureID(i)`, and SEEDS
  `Field42` from the definition table. `ALM-CLS-053` gives the type-4 spawn's own writes —
  `sizeX`/`sizeY` to the footprint, `scanRange` to `obj+0x48`, `healthMax` to `obj+0x44`/`+0x42` —
  and `+0x42` is the word both arms read. `SAV-BLDG-037` measures that pair in the owner's own
  original saves as equal non-zero values (1000/1000, 30000/30000, 100/100, 2000/2000), which are
  the `healthMax` values `DAT-BLD-005` gives for those kinds. `DAT-SCHEMA-004` puts `healthMax` at
  Buildings column 4, which is parameter position 3.

  The entry is selected by `buildingParams` (`pkg/mapload/structures.go`), factored out of the
  footprint pass's own `resolve` so the selection rule — low byte of the class key, one-based,
  bounded by the collection's count — is read once rather than written twice. The store
  truncates to 16 bits, the width of the destination the claim names; `healthMax` runs 0..30000
  over all 66 entries on both roots, so no shipped entry reaches it. A placement resolving no
  entry, and an entry too narrow to carry position 3, each leave the field at zero and still
  yield a `Structure`; neither is reached on shipped content.

  Seeding does not name the field. The semantics stay Unknown and the field stays named for its
  offset: what is established is which table position the original copies into that word.
- `pkg/mapload/script.go`'s `ScriptStructures(m *alm.Map) map[uint16]sim.StructureID` builds the
  lookup table `bindParams`'s `typeStructure` case resolves a node's `Field12` word against, the same
  pattern `ScriptUnits`/`alm.Unit.UnitID` already uses for unit references.
- `sim.NewStructuredWorld` (world.go, the tenth positional constructor reaching `newWorld`) takes
  `structs []Structure` as its final argument. All three world-construction call sites —
  `fromalm.go`'s `FromALMRoster`, and `start.go`'s two rebuild sites in `StartMission` and
  `StartMissionScripted` — pass the map's own structure list (`Structures(m, t)` or
  `base.Structures()`).
- `pkg/game/mission.go`'s `StartMissionFrom` and `cmd/almtool/script.go` both wire
  `Structures: mapload.ScriptStructures(m)` into their `mapload.ScriptRefs{}` literal.
- `cmd/missionrun/trace.go`'s `checkName`/`instantName` name the four opcodes (`"health"`,
  `"itemdist"`, `"structfield"`, `"structfield"`) so `-trace` output is self-describing.
  `checkDesc` renders a check-21 node's own STRUCTURE reference through `structText`
  (`structure N`, or `NO STRUCTURE`). Its fallback arm renders a UNIT reference, and a check-21
  node names no unit, so through that arm the node read `structfield(NO UNIT)` -- a true statement
  about units in the place every other arm puts its own subject.
- `silenceText` names `ScriptSilenceNoStructure` and `ScriptSilenceNoPlayer`, and an unnamed reason
  prints its own number rather than the word `unknown`. Both reasons had rendered as `unknown`:
  the structure one is this story's, the player one predates it. A single shared word is why two
  went unnoticed, so the fallback carries the number the next reader needs.
- `pkg/game/scenario.go`'s `playCensus.observe` counts every silence reason that is not
  `ScriptSilenceUnsupported` as an unresolved reference, through a `default` arm rather than a list
  of the reasons that existed when it was written. A reason missing from such a list is counted
  NOWHERE, so the silence becomes invisible to the milestone census instead of being reported under
  the wrong heading; `ScriptSilenceNoStructure` was missed exactly that way.

### The instrument that reads the seeded field

`mapload.StructureFieldRefs(m, t)` lists every compiled node that reads or writes `+0x42` -- check
opcode 21 and instant opcode 26 -- with the structure each names and that structure's seeded value.
`cmd/classdump`'s `-databin` verb prints it beneath the per-map summary of the field.

The summary alone cannot witness the seed. A build seeding the wrong value and one seeding the right
value print the same population on every shipped map, because no shipped trigger holds either way.
Measured over all 28 campaign maps on both roots: 16 check-21 nodes on maps 91, 101, 131 and 150,
every one resolving to a structure its map placed, every one reading a seeded value of 1; and no
instant-26 node anywhere. `closure.md` carries the per-node table.

### Byte-form: the structure section, `formatVersion` 57 to 58

**`formatVersion` moved from 57 to 58.** One version, one section, appended — reported to the seat
per the brief's instruction, since `wt-1032` (open concurrently) rewrites `binary.go`'s version
dispatch and `wt-1031` touches `pkg/ui`/`pkg/game/cursor.go` in the same file family.

The STRUCTURE SECTION (`pkg/sim/structurebinary.go`) sits between the script-state section (formation
modes and cell-tail records, story 0166) and the script section (registers, latches, counters —
`pkg/sim/scriptbinary.go`), because `decodeScript` consumes the rest of the buffer and returns no
used count, so any section behind it would need one. One record is 6 bytes: a 4-byte `uint32` id, a
2-byte `uint16` `Field42`, preceded by a 4-byte count. No presence byte and no cross-section
reference — a structure is not resolved against an entity, a group or a player. Structures are
written in ascending id order (the constructor's own sort order, nothing sorted at encode time) and
decode refuses a non-ascending or duplicate id, the entity section's own rule.

## B4 — the census and the witness

`pipeline/check-milestone.sh` over the 28 shipped campaign maps on both preserved roots, before this
story (from `pipeline/milestone-baseline.txt`, matching `DIV-239`'s own figures): 10 nodes of check 4,
7 of check 16, 16 of check 21, 33 in all per root, 66 total. Eleven maps print a `cannot run` line: 40,
60, 71, 81, 90, 91, 100, 101, 131, 140, 150.

Measured after this story, with a `missionrun` built from this worktree
(`AGAINROM_MILESTONE_DRIVE`): **0 nodes of all three check opcodes and of instant 26, on both roots.**
None of the eleven maps prints a `cannot run` line for these opcodes any longer. Every other census
line — script node counts per mission, the drive's own outcome (`lost at tick 240`) and unit census
(`4 of 36 unit(s) moved, 1 fell`) — is unchanged from the `1029` baseline. `closure.md` carries the
full before/after and the witness trace.

**The census measures whether a node RUNS, not whether it answers correctly.** The same 33-to-0 is
measured before and after the `Field42` seed, because seeding a field does not change which arms
this build evaluates. That is the census's own scope and not a defect in it, and it is why the
census could not have caught the wrong initial value: the number was already 0 while every
check-21 node on four maps answered wrongly. The instrument that caught it is the fired-trigger
sweep in `closure.md`, which compares this build's trigger sets against the pre-story ones.

## G2 — the limit each arm implies, and its class

`TRIG-CHECK-054` gives a limit for each of the three arms and classes each one. The classes are not
the same, and the difference is what G2 needs: two are engine constants and one is a stored field.

**Check 4's gate is a call-site literal.** The comparison is `CMP ...,0x6` at `L12483`, an engine
constant rather than anything a shipped file carries. The corpus's only authored `Par1` value is 6,
so raising or removing the gate changes no shipped file. This build spells it as
`scriptCheckHealthGate` in `pkg/sim/script.go`, one named constant at one site.

**Check 16's X and Y are an 8-bit engine read width.** The parameters are byte loads (`MOV AL/DL,byte
ptr`); the limit is 0..255 per axis. The widest authored values in the whole corpus are 116 and 130,
so the limit is unreached in shipped data and lifting it changes no shipped file. This build
reproduces the width at the call site rather than lifting it (B2 above).

**Check 21's and instant 26's field is a stored 16-bit record member, not an engine width.** It is
`structure+0x42`, confirmed as a `u16` by the save serializer (`SAV-BLDG-037`). Lifting this width
changes the save record's own bytes, unlike the other two. It is the one limit of the three that G2
cannot lift without a format change, and this build carries the same 16 bits in its own byte form
(`structurebinary.go`), so the two agree on the constraint.

A fourth limit belongs to the seed rather than to an arm: `healthMax` is read from the Buildings
table and stored into a 16-bit field, so a table value above 65535 would truncate. The shipped range
is 0..30000 over all 66 entries on both roots, so this is unreached, and it is a property of the
installed table rather than of the executable.

## What is deliberately not in this story

Unchanged from `contract.md`: check opcode 12 (`DIV-243`, story `1029`'s exclusion), the meaning of
`structure+0x42`, structures as a subsystem (health, destruction, occupancy, art binding), and instant
opcodes other than 26.

## Divergence rows

`DIV-239` is closed (moved to `DIVERGENCES-CLOSED.md`): the three arms and the paired instant are
implemented as `TRIG-CHECK-051`..`053` describe. Its closure was re-verified at adversarial pass 1
and stands; two of its cells repeated the withdrawn absence claim below and are corrected.

`DIV-267` is opened and CLOSED. It was written in "Authored where research is silent", stating that
no map field authors `Field42`'s initial value, so this build authored zero. The premise was wrong
when it was written: `ALM-CLS-053` and `SAV-BLDG-037`, both High and both in this story's own pin,
give the value, and it does not come from the map record at all. The row's absence claim came from
searching this tree and the ALM ledger the subject belongs to, rather than the subject across every
ledger. The field is now seeded and the row is closed with the fix.

`DIV-264` is opened for what remains: nothing but instant 26 moves the field, this build having no
structure damage model. On mission 101, whose twelve records carry `healthMax = 1` against twelve
nodes testing the field `< 1`, that means the answer is fixed at false for the whole mission.

`DIV-265`, `DIV-266`, `DIV-268` and `DIV-269` of the `DIV-264`..`DIV-269` reservation are returned
unused. Check 16's 8-bit parameter width needed no row, because this build reproduces it rather
than diverging from it.
