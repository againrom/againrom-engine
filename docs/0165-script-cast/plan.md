# 0165-script-cast — plan

Steps are numbered `Step n` so they cannot be read as spec ids.

## Shape

Two new pieces of world state, one new section of the byte form, one new fork in the spell table,
and three new arms of the instant dispatch. The state is small and flat on purpose: neither a
pending cast nor an area effect is an entity, and modelling either as one would put it into the
entity record, the occupancy plane and the routing scratch, none of which it belongs in.

| File | What changes |
|---|---|
| `pkg/sim/scriptcast.go` | new. The pending-cast record, the two instant arms, the per-tick resolve pass. |
| `pkg/sim/celleffect.go` | new. The area effect record, the placement, the per-tick decay, instant 29's arm. |
| `pkg/sim/castbinary.go` | new. The byte form's casting section: length, encode, decode. |
| `pkg/sim/script.go` | the three opcode constants, the two support tables, three dispatch cases. |
| `pkg/sim/step.go` | two calls at the head of the tick. |
| `pkg/sim/spell.go` | `SpellRule` gains the two columns. |
| `pkg/sim/binary.go` | version 49; the spell record gains two fields; the casting section is decoded between the spells and the script. |
| `pkg/sim/world.go` | the two new slices on `World`. |
| `pkg/data/spell.go` | two slots read, two fields on `data.Spell`. |
| `pkg/mapload/spell.go`, `pkg/game/scenario.go` | the two fields carried through to `sim.SpellRule`. |

## Design decisions

**DD-1 — a pending cast is a flat record on the world, not an entity.** The decoded actor is 0x198
bytes, carries no runtime id and is appended to a list that is not the actor list. It has no health,
no order, no group and no owner, and it never occupies a cell. Making it an `Entity` would give it
all of those, put it in the occupancy plane and the route scratch, and hand every existing arm of the
step a record it must learn to skip. A `[]scriptCast` on the world is what the decode describes.
(FR-1, FR-2, FR-3.)

**DD-2 — an area effect is keyed by the engine's own 16-bit key and stores nothing else about its
cell.** `x` is the low byte and `y` the high byte, so the key **is** the cell and carrying `x` and
`y` beside it would be two more bytes that can disagree with it. The key is computed once, where the
instant's parameters are truncated, and the decode's carry-into-y behaviour then follows for free
rather than being a special case. (FR-5, FR-6, AC-8.)

**DD-3 — the effect slice is kept sorted by key, and within a key in arrival order.** Canonical
order is what makes the byte form a function of the logical world, on `Hash`'s own terms: two worlds
that hold the same effects must encode the same bytes whatever order they were placed in. Sorted
insertion also makes "at most six on this cell" a bounded scan of one contiguous run rather than a
walk of the whole slice. (FR-5, FR-8, AC-9.)

**DD-4 — the fork at the landing is on the spell row, not on the instant.** `MAGIC-SHAPE-008` puts
the shape on the `Distribution system` column, which is a property of the spell and not of who cast
it. So one resolve routine serves both instants, and the two differ only in where the destination
comes from: the record's own bytes for 21, the target's current cell for 24. That is also what makes
FR-4's "an instant-21 cast of a point row lands nothing" fall out rather than be written: the point
arm asks for a target unit and the cell record has none. (FR-4.)

**DD-5 — `SpellRule` gains an `Area` flag and not the raw column.** The column's other values are an
enum `MAGIC-SHAPE-008` explicitly leaves unread, and storing a value nothing can interpret invites a
later reader to interpret it. What the build needs is the one bit the claim decodes, so that is what
is stored. The duration column is stored as its own int32 because FR-4's arithmetic uses the value.
(FR-7, DD-4.)

**DD-6 — the resolve pass runs at the head of the tick, beside the two decays.** The script pass
runs on phase 6, after commands, so a cast authored on tick T is walked at the head of T+1 whatever
else that tick does. Placing it beside `decaySpellEffects` — which documents "the marks age first,
before anything can set one" — gives the area effect the same rule: an effect placed this tick
stands its full life, and one placed last tick loses exactly one tick of it. The order at the head of
a tick is: spell marks age, area effects age, pending casts resolve, unbidden casts run. (FR-3,
FR-5, AC-3, AC-6.)

**DD-7 — a script cast records no `CastEvent`.** `CastEvent` names a caster entity and a target
entity and is documented as "one APPLIED cast, past every refusal". A script cast has no caster
entity at all, and an instant-21 cast has no target entity either. Widening the record to carry a
sentinel would make every existing consumer branch on it. None of the eleven spells the shipped
campaign casts through these two instants has a `Delivery System` of 2, so no shipped script cast
would draw a projectile even if it were reported. (P-1, SC-2.)

**DD-8 — the casting section sits between the spells and the script in the byte form.** `decodeScript`
consumes the rest of the buffer and returns no used count, so it must stay last. Appending after it
would mean giving it a length, which is a change to a section this story has no business in. (FR-8.)

**DD-9 — no test name may spell the live version, and none does.** A test whose name spells the
current version goes stale at every bump; that has been repaired after the fact three times, at 0099,
0104 and 0106, each time by re-spelling the new number in. This story checked instead of assuming:
`grep -rhoE "func Test[A-Za-z0-9_]+" --include=*_test.go .` over the whole tree returns four names
carrying a version — `TestAVersion20FormIsRefused`, `22`, `26` and `37` — and every one of them
names an **old** version it refuses, where the number is permanent and load-bearing. No name spells
the live one. Nothing is renamed; the version is spelled only in literals inside test bodies, which
a bump makes fail loudly. (FR-8.)

**DD-10 — six refusals, each with its own message.** A decoder that accepts a state no tick can
leave hands back a world whose digest means something else. The six FR-8 names are each a single
comparison, and each is witnessed by reverting it.

## Steps

**Step 1 — the spell columns.** `pkg/data/spell.go` reads slots 8 and 11 into `Area` and
`AreaDuration`, with the same negative-cell clamp `MaxRange` gets. `pkg/mapload/spell.go` and
`pkg/game/scenario.go` carry them into `sim.SpellRule`. (FR-7, DD-5.)

**Step 2 — the world state.** `World` gains `casts []scriptCast` and `effects []cellEffect`.
`pkg/sim/celleffect.go` holds the record, `cellKey`, the sorted insert with its six-slot cap, the
per-tick decay, and instant 29's arm. `pkg/sim/scriptcast.go` holds the pending record, the two
creation arms and the resolve pass. (FR-1 … FR-6, DD-1, DD-2, DD-3, DD-4.)

**Step 3 — the dispatch.** Three constants, three cases in `runInstant`, three entries in
`scriptInstantSupported`, and the two calls in `stepWorld`. (FR-1, FR-2, FR-3, FR-6, DD-6.)

**Step 4 — the byte form.** Version 49. The spell record gains the flag bit and the duration word.
`pkg/sim/castbinary.go` holds the new section's length, encode and decode with its six refusals.
`binary.go` threads it between the spells and the script, and DD-9 is checked over the whole tree. (FR-8, DD-8,
DD-9, DD-10.)

**Step 5 — the tests, then the census.** Unit tests for every AC and P; then the milestone census
over both roots and the two mission-10 and mission-20 `UNSUPPORTED` counts.

## Traceability

| FR | Built by | Decided by | Witnessed by |
|---|---|---|---|
| FR-1 | Step 2, Step 3 | DD-1 | AC-1, SC-3 |
| FR-2 | Step 2, Step 3 | DD-1, DD-4 | AC-2, SC-1 |
| FR-3 | Step 2, Step 3 | DD-1, DD-6 | AC-3, P-2, SC-4 |
| FR-4 | Step 2 | DD-4, DD-7 | AC-3, AC-4, AC-5, SC-1, SC-2 |
| FR-5 | Step 2, Step 3 | DD-2, DD-3, DD-6 | AC-6, SC-5 |
| FR-6 | Step 2, Step 3 | DD-2 | AC-7, AC-8 |
| FR-7 | Step 1 | DD-5 | AC-4, AC-5 |
| FR-8 | Step 4 | DD-3, DD-8, DD-9, DD-10 | AC-9, AC-10, P-1 |
| — | Step 5 | — | AC-11, P-3 |

## Success criteria

**SC-6** `go build ./...`, `go vet ./...`, `gofmt -l` and `go test -trimpath -count=1 ./...` are
green over both repos, and every `scripts/check-*.sh` found by glob passes.

**SC-7** The milestone census over both preserved roots reports no `instant op 21`, `instant op 24`
or `instant op 29` line on any of the 28 campaign maps, and no other census line moves.

**SC-8** Each of the six byte-form refusals and the new version gate is witnessed by reverting it and
seeing a named test fail.

**SC-9** Mission 10 and mission 20's `UNSUPPORTED` counts are recorded against the values master
carried before this story.
