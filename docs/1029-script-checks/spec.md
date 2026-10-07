# Story `1029` — check opcodes 17 and 9

As-built behavioural specification. It describes what shipped, not what was intended. The contract
is `contract.md`; the evidence is `closure.md`.

## Summary

`pkg/sim` evaluates two more mission-script check opcodes: 17, which asks whether a named unit's
container holds a named item code, and 9, which returns the authored map id of the unit a named
subject is pursuing. Both write their own register in the 100-int register file and nothing else.
Every shipped campaign trigger that reads one of the two stops being permanently inert.

Supporting this, `ScriptCheck` gained an item reference, `Entity` gained the authored map id its
map record names, and the byte form widened once, from `formatVersion` 56 to 57.

## Functional requirements

### FR-1 The check record carries an item reference

`ScriptCheck` has two new fields:

- `Item uint16` — a packed item code, carried and not resolved.
- `HasItem bool` — the presence flag, because 0 is a legal packed code.

They are `ScriptInstant`'s existing item reference on the same terms and for the reasons that
field's own doc block gives: the code is packed by the loader, `pkg/sim` compares codes and never
inspects an item table, and the presence flag exists because the zero value is legal data.

### FR-2 The loader stops dropping the item parameter

`pkg/mapload/script.go` already decoded a condition's `Target_Item` parameter for the instant
binder. It now also writes `Item` and `HasItem` onto the check record it builds. No new decoding
was added; the binder stopped discarding what it had.

### FR-3 Check opcode 17 evaluates

`ScriptCheckItemTest` (17) is in `scriptCheckSupported`. Its arm runs in the second check switch,
after the unit reference is resolved, and writes its register:

- `1` when the named unit exists in the world and its **container** holds a stack whose code equals
  the node's `Item`.
- `0` in every other case: the node names no item (`HasItem` false, tested before any container is
  read), the unit is not in the world, or the container does not hold the code.

The presence flag decides the third case, not the code being zero. An arm that searched for code 0
instead would make the answer depend on a property of the carry store, and would leave `HasItem`
parsed and unread. `scriptItemHolder`, which the item instants share, tests its own flag the same
way. No shipped node reaches the case: `TRIG-ITEMTEST-040` reports all 23 authored check-17 nodes
binding `[0:Unit t=4][1:Item t=8]`.

It reads the container only. Worn and equipped places are not searched. It writes no simulation
state other than its own register.

### FR-4 Check opcode 9 evaluates

`ScriptCheckTargetID` (9) is in `scriptCheckSupported`, in the same switch, and writes its register:

- the pursued entity's `MapUnitID`, when the subject holds an attack victim and that victim is
  still in the world.
- `0` when the subject holds no victim, when its order is a movement order, and when the victim
  carries no authored map id.

A victim that has left the world is not a state this package holds: the constructor and the removal
sweep both clear a pursuit naming an entity the world does not hold, so the register takes 0 through
`HasAttackTarget` being false. The arm's own index guard covers the same case and is unreachable;
`DIV-240` and `closure.md` record it as such.

The subject is the node's own resolved unit. The pursuit is `Entity.HasAttackTarget` and
`Entity.AttackTarget`. It writes no simulation state other than its own register.

### FR-5 The entity carries its authored map id

`Entity` has a new last field `MapUnitID uint16`. It is the map record's own identifier word — the
type-6 record's `+0x40`, which `pkg/formats/alm` exposes as `alm.Unit.UnitID`. Zero means the
entity carries no authored map id.

It is a field on the entity rather than a table on the world because check 9 chooses its subject's
target at run time. A compile-time binding cannot supply it: the node names only the subject, and
the answer is a property of whoever that subject happens to be pursuing on the tick the pass runs.

### FR-6 Both loader paths write the authored map id

- `pkg/mapload/fromalm.go` writes `MapUnitID: u.UnitID` for every placement it turns into an entity.
- `pkg/mapload/start.go` writes `MapUnitID: savedMapUnitID(p)` at both party-member construction
  sites. `savedMapUnitID` returns the id the member's restored original-save record names, and 0
  when the member has no such record.

A generated party member therefore carries 0, and check 9 answers 0 for him.

The third producer is not a loader path and is named here so the enumeration is complete:
`raisedGhost` in `pkg/sim/spell.go` mints a Control Spirit actor mid-simulation and sets no
`MapUnitID`, so that actor carries 0 on the same terms a generated party member does. Nothing
places it, so it has no authored id to carry.

### FR-7 The byte form carries both new fields

`formatVersion` is 57. Two records widened in the same move:

- the entity record, `entityLen` 275 to 277, `MapUnitID` little-endian at `+275`.
- the check record, `scriptCheckLen` 73 to 76, `Item` little-endian at `+73` and the `HasItem` flag
  byte at `+75`.

Both fields round-trip byte-identically. A form declaring version 56 that carries either field is
refused.

### FR-8 The trace names both arms

`cmd/missionrun`'s `checkName` answers `targetid` for 9 and `itemtest` for 17, so the census and the
trace report the two by name rather than by number.

### FR-9 The shipped population is measured

`cmd/mapunitcensus` walks the 28 shipped campaign maps of one install and reports the authored map
id population: total placements, placements authored at id 0, per-map duplicate ids, and the highest
id. It takes `-assets` or `AGAINROM_ASSETS`.

It exists because FR-5's sentinel is safe only while no shipped placement is authored at 0, and that
is a measurement rather than an assumption.

## Design decisions

### DD-1 The item reference is carried, not resolved

`pkg/sim` receives a packed code and compares it against the codes its own stacks carry. It does not
consult an item table, which is `pkg/data`'s and above the determinism wall. This is
`ScriptInstant`'s existing decision, restated because a second field now depends on it.

### DD-2 The authored map id is a field on `Entity`, not a table on the world

The contract named two candidates. The field was chosen for three reasons:

1. **The lookup direction is entity to id.** Check 9 has an entity and needs its id. A table keyed
   by id would have to be walked or inverted on every evaluation, and the world would carry a second
   structure that must stay in step with the entity list through spawn, death and removal.
2. **The entity list is already the serialized form.** A field rides the entity record and the one
   `formatVersion` move FR-1 needed anyway. A table is a new byte-form section with its own length
   prefix and its own version story.
3. **`Entity.Group` and `Entity.Owner` are the same shape.** Both are authored identifiers the map
   record supplies, carried on the entity with no presence flag beside them. A third identifier
   behaving differently would be the odd one out.

The cost is `DIV-242`: with no presence flag, an authored id of 0 is indistinguishable from no
authored id. `cmd/mapunitcensus` measures that no shipped placement reaches the case.

### DD-3 The pursuit test is `HasAttackTarget`

The original tests an order object (`ord+0x08 == 5`). This build has no order object; its one
pursuit representation is the attack victim an attack order writes. `DIV-244` records the
difference, including that the original's `ord+0x08` also takes 6 for a second pursuit arm this
build does not distinguish.

### DD-4 A stale target answers 0 rather than faulting

`DIV-240`. The original's arm has no null branch and faults. This build cannot reach the state: a
dangling pursuit is normalised at construction and at removal. The arm keeps an index guard anyway
and writes 0 rather than indexing with -1, because a panic in `pkg/sim` on shipped content is the
one outcome that must not ship. The guard is unreachable and therefore unwitnessed, which
`closure.md` records rather than leaves to be discovered.

### DD-5 Check opcode 12 is not implemented

`DIV-243`. It is decoded as byte-identical to 17 and the campaign authors it zero times, so the
table entry would witness nothing. The contract required that this be recorded rather than done
silently.

### DD-6 The sentinel opcode the tests use moved

Six tests in `pkg/sim` and one in `pkg/mapload` used opcode 9 as a stand-in for "an opcode this
build does not support". Opcode 9 is now supported, so `pkg/sim` gained
`scriptCheckSentinelOp int32 = 20` with a guard test asserting it stays outside
`scriptCheckSupported`, and `pkg/mapload`'s own sentinel node moved to 20 in place.

## Out of scope

- Check opcodes 4, 16 and 21, which hold the remaining 33 census nodes. `DIV-239`.
- Check opcode 12. `DIV-243`.
- The order model. Check 9 reads the pursuit this build already has and changes nothing that
  decides one.
- What a trigger does once its register is live. The trigger machine already existed; this story
  writes registers into it.

## Divergences

| Id | Type | What |
|---|---|---|
| `DIV-239` | UNKNOWN | Check opcodes 4, 16 and 21 stay unsupported, holding 33 census nodes per root |
| `DIV-240` | DEVIATION | Check 9 on a stale or absent target answers 0 where the original faults |
| `DIV-241` | UNKNOWN | Check 9 answers 0 for a pursued unit the map file never placed |
| `DIV-242` | UNKNOWN | Authored map id 0 is not distinguishable from no authored map id |
| `DIV-243` | FIDELITY-DEBT | Check opcode 12 not implemented, although decoded and identical to 17 |
| `DIV-244` | UNKNOWN | The pursuit test is `HasAttackTarget`, not the original's order-object branch |

## Claims

Read whole through the pin (`790cce1`) with `go run ./tools/claim <ID>` from `research/`.

| Row | What it answers here |
|---|---|
| `TRIG-ITEMTEST-040` | what check 17 tests, what it reads, that it writes no simulation state, and the authored counts |
| `TRIG-TARGETID-032` | what check 9 returns, from which fields, and what it does when the order is not a pursuit |
| `TRIG-STORE-002` | the 100-int register file both arms write into |
| `TRIG-COND-003` | the condition dispatch and the register poisoning of an unsupported arm |
| `ALM-TRIG-046` | the type-6 `+0x40` word is the identifier a script's `Target_Unit` names |
| `ALM-UNIT-040` | the type-6 record field by field, including which word is the unit id |
| `AI-PURSUE-040` | a pursuit in the original is an order object with two arms, 5 and 6 |
