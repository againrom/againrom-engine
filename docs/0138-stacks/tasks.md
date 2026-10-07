# 0138-stacks — tasks

## Legend

Every task here is kind `impl`. **Files** is exhaustive: a task touches nothing else.

## T1 — the container becomes a list of stacks (impl)

**Files:** `pkg/sim/carry.go`, `pkg/sim/world.go`, `pkg/sim/equip.go`, `pkg/sim/step.go`,
`pkg/sim/script.go`, `pkg/sim/binary.go`, and tests beside them.

**Covers:** FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10; AC-1…AC-10; P-1…P-4;
D-1…D-8.

**Boundary:** everything inside `pkg/sim`. `Sack`, `pourSack`, `Stock`'s own fields, `Equipped` and
`formatVersion` are not edited. No package outside `pkg/sim` is touched — D-3 exists so that none
has to be.

**Done when:** `pkg/sim` compiles and its tests are green; a test names each of AC-1…AC-10, plus one
for D-7's displaced-code case; the byte-form offset table in `binary.go` is unchanged; `step.go`'s
0132 DD-3 note is corrected where D-8 makes it untrue; and any pinned digest constant that moved is
re-taken with a comment beside it saying which container folded and why the form did not change.

## T2 — the pack shows one cell per stack, with its count (impl)

**Files:** `pkg/ui/inventory.go`, `pkg/game/inventory.go` (`buildInventoryPack` only),
`pkg/game/world.go` (`openMission`'s pack arm, `refreshPack`, `enqueueEquip`, and the `invCodes`
field), and tests beside them.

**Covers:** FR-11; AC-11, AC-12; P-5; D-9, D-10.

**Boundary:** `pkg/game/inventory.go`'s figure composition, its icon loader and its subject builder
are another story's ground — only `buildInventoryPack` changes there. The pack area's cell count,
its geometry and the window's frame do not change.

**Note:** the pack cell index the equip command carries now names an **element**, so the guard in
`enqueueEquip` must resolve it against the stacks and not against the flat codes, or a click on the
second cell equips the wrong item.

**Done when:** a test shows the composed window differs between a subject whose first pack element
holds 3 and the same subject at 1 (AC-11), and another shows it composes with a nil font and never
panics (AC-12); `pkg/ui` and `pkg/game` are green.

## T3 — the drive takes a sack and reports stacks (impl)

**Files:** `cmd/missionrun/main.go` and its tests.

**Covers:** FR-12; AC-13; D-11.

**Boundary:** the waypoint arm, the attack arm, the census arm and the trace arm keep their present
behaviour and output; the take arm runs after the attacks. Nothing outside `cmd/missionrun`.

**Done when:** `-take TAKER:X:Y` parses like the two flags beside it, refuses the same malformed
shapes they do with a message naming the flag, prints what the transfer answered and then the
taker's elements with a count beside any above 1; the before-the-blow carried line prints elements
the same way; `cmd/missionrun`'s tests are green and cover the parse and the report.

## Traceability

| Task | FR | AC | P | Design |
|---|---|---|---|---|
| T1 | FR-1…FR-10 | AC-1…AC-10 | P-1…P-4 | D-1…D-8 |
| T2 | FR-11 | AC-11, AC-12 | P-5 | D-9, D-10 |
| T3 | FR-12 | AC-13 | — | D-11 |
</content>
</invoke>
