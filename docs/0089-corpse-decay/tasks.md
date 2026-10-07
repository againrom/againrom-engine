# Tasks — a body decays

**Legend.** Kind `impl` = one commit trailered `SDD-Task: 0089-corpse-decay/T<n>`. The executor
holds `spec.md`, `plan.md` and its own entry, and nothing else. Tests are synthetic and read no game
install. Do not touch `pkg/sim/engage.go` or the acquisition path.

## T1 — the dying-time column and the bone slot — kind impl

Files: `pkg/data/unitdef.go`, `pkg/data/humandef.go`, `pkg/data/anim.go`, and the tests beside them.

- Carry the dwell column on `UnitDef` and `HumanDef` as `DyingTime int32`: unit slot 33, which is
  read and dropped today, and human slot 23, which is one past the streamed bound.
- Carry the bone block's slot length on `UnitAnim` as `BoneSlot`, clamped like every other slot. No
  base, no total and no gate changes value.

Covers FR-2 (the column half), FR-7, C-4, DD-9, DD-10's first clause. Fences: no consumer of either value in this task; no other
slot moves; `unitCtorDefaults` gains no second copy of the default.

**Done when:** a row naming the column yields it and a row leaving it empty yields the constructor's
default; a class declaring no bone phase yields a zero slot and every base already derived is
unchanged; the reflect-driven key pins and the whole package are green.

## T2 — the decay ladder and the byte form at version 17 — kind impl

Files: `pkg/sim/world.go`, `pkg/sim/step.go`, `pkg/sim/route.go`, `pkg/sim/binary.go`, and the tests
beside them.

The whole of FR-1 to FR-6, FR-10 and FR-11 in one commit, because a canonical field and its
encoding cannot land apart: a tree holding one without the other fails its own field-set pin.

- The three fields, their constructor rules and their decoder refusals.
- The death transition at the site that fells a unit; the pass at the end of an advance; the walk's
  period and phase; the ladder; the pin for a non-ground mover; removal and its two sweeps.
- The occupancy clause, in the predicate that already answers it.
- Version 17, the record's new width and the three offsets at its tail.

Covers FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-10, FR-11, P-1, P-2, P-5, C-1, C-3, DD-1, DD-2, DD-3,
DD-4, DD-5, DD-6, DD-7, DD-8.
Fences: `pkg/sim` stays stdlib-only with no float, no clock and no map on any path a world touches;
no existing offset moves; nothing outside `pkg/sim` changes in this task.

**Done when:** AC-1 to AC-10 and AC-16, AC-17 have tests; the field-set pin lists the three new
fields; a version-16 buffer is refused; and `go test ./...` is green.

## T3 — the spawn carries the dying time — kind impl

Files: `pkg/mapload/spawn.go`, `pkg/mapload/fromalm.go`, and the tests beside them.

Both spawn paths put the resolved definition's dwell column on the entity they build, beside the
health pair and the combat numbers they already carry. A placement that resolves to no definition
takes the constructor's default by the same route it takes every other default.

Covers FR-2 (the spawn half), C-4, DD-9's spawn side. Fences: no other spawned field changes; no new resolution rule.

**Done when:** AC-18 has a test over hand-built definition tables for both the units arm and the
humans arm, and the package is green.

## T4 — the bone selection — kind impl

Files: `pkg/render/terrain/unitanim.go`, the one line of `pkg/game/units.go` that mirrors a
descriptor field, and the tests beside them.

The render tier's descriptor gains the bone slot and the loader copies it, value for value, as it
does the rest — the two go together because the reflect-driven pin between the two descriptors is
what would otherwise be red between this task and the next. One new selection then answers the bone
frame from the corpse class's descriptor, its own frame count, the octant and the stage. It refuses rather than
answering an index the caller cannot tell from a real one, and it shares the direction rule with the
selections already there rather than restating it.

Covers FR-7 (the mirror half), FR-8, P-3, C-5, DD-10.
Fences: the three existing selections answer exactly what they answered before; nothing in this file
opens an archive, reads a sheet or consults a clock.

**Done when:** AC-12 and AC-13 have tests, including both layouts and every octant, and no answer
lies outside its own direction's slot at any stage.

## T5 — the seam draws the stage — kind impl

Files: `pkg/game/world.go` and the tests beside it.

The snapshot's not-alive arm tries the bone selection first when the stage calls for it, then the
fall, then what it drew before.

Covers FR-9, P-4, DD-11. Fences: no change to the fall's own clock or its stamp; no entity stops being
drawn; the swing and live selections are untouched; the descriptor mirror is already done.

**Done when:** AC-14 has tests over a hand-built world and a two-class bundle, and the package is
green.

## Traceability

| Requirement | Task |
|---|---|
| FR-1, FR-3, FR-4, FR-5, FR-6, FR-10, FR-11 | T2 |
| FR-2 | T1, T2, T3 |
| FR-7 | T1, T4 |
| FR-8 | T4 |
| FR-9 | T5 |
| P-1, P-2, P-5 | T2 |
| P-3 | T4 |
| P-4 | T5 |
| C-1, C-3 | T2 |
| C-4 | T1, T3 |
| C-5 | T4 |
| DD-1, DD-2, DD-3, DD-4, DD-5, DD-6, DD-7, DD-8 | T2 |
| DD-9 | T1, T3 |
| DD-10 | T1, T4 |
| DD-11 | T5 |
