# tasks — 0136

Reading key: `FR`/`AC` → `spec.md`; `DD`/`R`/`SC` → `plan.md` (§Decisions, §Risks, §Success
criteria). All five tasks are **implementation**: one coherent product change each.

## T1 — what a piece is worth, and what a worn set adds up to

**Kind:** implementation.
**Boundary:** stops inside `pkg/data`. No caller learns anything new; the equip gate still refuses
an armour after this task and the world a map builds is byte-identical.

**Files:** `pkg/data/wear.go` MODIFY · `pkg/data/foldwear.go` ADD · `pkg/data/recompute.go` MODIFY
(doc only) · `pkg/data/wear_test.go` MODIFY · `pkg/data/foldwear_test.go` ADD ·
`pkg/mapload/wear_test.go` MODIFY (fixture width only)

**Covers:** FR-1, FR-2, FR-3, FR-4, FR-5, FR-5a, FR-6, FR-6a, FR-6b, FR-7a · AC-1, AC-2, AC-3, AC-4,
AC-5, AC-8 · DD-1, DD-2, DD-3, DD-3a, DD-4, DD-5, DD-5a, DD-11 · R-2 · SC-1, SC-2, SC-3, SC-7, SC-9.

**Scope fences.** Do not touch `pkg/game`, `pkg/sim`, `cmd/`, or a production file under
`pkg/mapload`. Add no field to `EquipMod`, `Loadout`, `Derived` or `Combat` and do not change
`Recompute`'s arithmetic — the additive seam exists and this task fills it. Do not read or name the
`sutableFor` column. Do not touch `Shield`, `ResolveShield`, weight or magic capacity.
`ResolveArmor`'s refusal set may widen for row LENGTH only: a Slot of 0 or 13 still comes back on
the returned value exactly as read.

**Done when:** `go build ./... && go vet ./... && go test -count=1 -trimpath ./...` is green, and
deleting the `+ 0.5` from the defence term alone makes a named test fail.

## T2 — the gate opens, and the block is re-derived from the whole set

**Kind:** implementation.
**Boundary:** the equip gate and the re-derivation. It reads what T1 built and edits none of it.

**Files:** `pkg/game/rearm.go` ADD · `pkg/game/world.go` MODIFY · `pkg/game/rearm_test.go` ADD ·
`pkg/game/equip_test.go` MODIFY

**Covers:** FR-7, FR-8, FR-9, FR-10 · AC-6, AC-7, AC-9, AC-10, AC-11 · DD-5a, DD-6, DD-7, DD-8 ·
R-3 · SC-4, SC-8.

**Scope fences.** In `world.go` touch ONLY `enqueueEquip`, `rearm` and `creditedSlot`; two other
lanes hold that file, so change no other function, no type and no import you do not need. Do not
touch `currentEquipment` or `refreshEquipment` — FR-10 asks whether they already follow a piece into
slot 12, not that they be changed. Do not touch `pkg/sim`, `pkg/mapload`, `pkg/data` or `cmd/`. Add
no field to `mapWorld` or `invPartyGear`. Do not make the gate refuse anything on account of the
wearer.

**Done when:** `go build ./... && go vet ./... && go test -count=1 -trimpath ./...` is green, and a
test drives an equip of an armour code through the ordinary command path and reads the moved
defence back off the entity.

## T3 — the tool can fell, take and wear

**Kind:** implementation.
**Boundary:** `cmd/missionrun` alone. It calls what T2 exported and changes no package.

**Files:** `cmd/missionrun/main.go` MODIFY · `cmd/missionrun/wear_test.go` ADD

**Covers:** FR-11 · AC-12 · DD-9, DD-10, DD-12 · R-1.

**Scope fences.** Change no file outside `cmd/missionrun`. Do not change the existing flags, the
waypoint or attack grammars, or any line either drive prints — the milestone gate compares that
output byte for byte. Every new test must pass with NO install present: the parse is a pure
function and is tested as one, and no fixture may contain a shipped name, count or number.

**Done when:** `go build ./... && go vet ./... && go test -count=1 -trimpath ./...` is green, and
`-mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3` prints exactly what it printed before.

## T4 — the tracker that follows a piece into the twelfth slot

**Kind:** implementation.
**Boundary:** one test file. No production line moves; if one has to, stop and say so instead.

**Files:** `pkg/game/equipment_test.go` ADD

**Covers:** FR-10 · AC-11 · SC-4.

**Scope fences.** Change nothing outside the added file — not `world.go`, not `inventory.go`, not
an existing test. Do not widen what the window DRAWS: that is a later entry's own change, and this
one measures the tracker alone. Build the subject through the same
in-package helpers the neighbouring equip tests already use rather than a new harness.

**Done when:** `go build ./... && go vet ./... && go test -count=1 -trimpath ./...` is green, and
changing `refreshEquipment`'s guard to compare only the first slot makes a named test in the added
file fail.

## T5 — the doll wears what he wears

**Kind:** implementation.
**Boundary:** the window's composition and the one builder that feeds it. No art path, no
appearance derivation, no sheet loader and no `ui` type changes.

**Files:** `pkg/game/inventory.go` MODIFY · `pkg/game/inventory_test.go` MODIFY ·
`pkg/game/equipment_test.go` MODIFY (re-pin only)

**Covers:** FR-10b, FR-10c, FR-10d · AC-11a, AC-11b · DD-13, DD-14 · R-4 · SC-10, SC-11.

**Scope fences.** Do not change how a layer is painted, how an icon is loaded, what an unreadable
address does, or the base's own read. Do not touch `pkg/data`, `pkg/sim`, `pkg/mapload`, `cmd/`,
`world.go`, `hero.go`, `heroart.go` or `appearance.go`. Add no field to any `ui` type — the twelve
icons already exist. An occupied slot 1 must still produce exactly today's addresses in today's
order, so the one-slot case is the loop's own first turn and not a second arm.
**One landed test now asserts a defect** —
`TestRefreshEquipmentDrawsOnlySlotOnesIconEvenWhenSlotTwelveIsWorn` in `equipment_test.go`. Re-pin
it at what the contract now says, with the reason it moved.

**Done when:** `go build ./... && go vet ./... && go test -count=1 -trimpath ./...` is green, and
narrowing the loop back to slot 1 alone makes a named test in the added file fail.

## Traceability

| Spec | Plan | Task |
|---|---|---|
| FR-1…FR-5a, FR-6, FR-6a, FR-6b, FR-7a; AC-1…AC-5, AC-8 | DD-1…DD-3a, DD-4, DD-5, DD-5a, DD-11; R-2; SC-1, SC-2, SC-3, SC-7, SC-9 | T1 |
| FR-7, FR-8, FR-9, FR-10; AC-6, AC-7, AC-9, AC-10, AC-11 | DD-5a, DD-6, DD-7, DD-8; R-3; SC-4, SC-8 | T2 |
| FR-11; AC-12 | DD-9, DD-10, DD-12; R-1 | T3 |
| FR-10; AC-11 | SC-4 | T4 |
| FR-10b, FR-10c, FR-10d; AC-11a, AC-11b | DD-13, DD-14; R-4; SC-10, SC-11 | T5 |
| — | SC-5, SC-6 | verification stage |
