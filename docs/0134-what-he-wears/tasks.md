# 0134 — tasks

**Reading key:** `FR`/`AC`/`P` → `spec.md`; `D-x` and `R-x` → `plan.md` (Design decisions, Risks);
criterion numbers → `plan.md` (Success criteria).

## T1 — the composed derivation

**Kind:** implementation. **Boundary:** `pkg/data` only; no caller of it changes.

**Files:** `pkg/data/appearance.go` MODIFY · `pkg/data/appearance_test.go` MODIFY

**Covers:** FR-4, FR-5, FR-7 (the key alone), AC-6, AC-7, AC-8, AC-9, AC-16, P-1, P-2 · D-1, D-2,
D-3 · criterion 4.

**Scope fences:** do not change `HeroBodyDir`'s signature or `HeroBodyName`'s. Do not add, rename
or remove a body name or a material block. Do not read a body list from anywhere — it arrives as
an argument. Touch no file outside `pkg/data`.

**Done when:** `go test -count=1 ./pkg/data/...` is green and the new derivation is exported and
total over an empty equipment set and an empty body list.

## T2 — a party member carries what he wears

**Kind:** implementation. **Boundary:** `pkg/mapload` only; nothing derives a body name here and
no caller in `pkg/game` is updated.

**Files:** `pkg/mapload/spawn.go` MODIFY · `pkg/mapload/start.go` MODIFY ·
`pkg/mapload/startequip_test.go` MODIFY · `pkg/mapload/party_test.go` MODIFY ·
`pkg/mapload/start_test.go` MODIFY

**Covers:** FR-1, AC-1, AC-2, AC-15 · D-4, D-5 · R-1 · criterion 1's `pkg/mapload` half.

**Scope fences:** add no field to any type in `pkg/sim` and do not touch that package at all; the
byte-form version constant must not move. Do not change what a member with a carry arrives
holding. Do not resolve a body name or a directory here.

**Done when:** `go test -count=1 ./pkg/mapload/... ./pkg/sim/...` is green, a member's starting
`sim.Stock` is composed from his worn array and his carried slice, and a member carrying neither
still produces exactly the entry he produced before.

## T3 — the class axis and the assembled character

**Kind:** implementation. **Boundary:** the two party builders and their callers; the art bundle
and the frame driver are untouched.

**Files:** `pkg/data/chargenbase.go` MODIFY · `pkg/data/chargenbase_test.go` MODIFY ·
`pkg/game/hero.go` MODIFY · `pkg/game/chargen.go` MODIFY · `pkg/game/frontend.go` MODIFY (the
party call sites alone) · `cmd/missionrun/main.go` MODIFY · `cmd/paneldump/main.go` MODIFY ·
`pkg/game/hero_test.go` MODIFY · `pkg/game/chargen_test.go` MODIFY ·
`pkg/game/heroappear_test.go` MODIFY

**Covers:** FR-1, FR-2, FR-3, FR-10, AC-3, AC-4, AC-5, AC-14, AC-15, P-6 · D-8, D-9, D-11, D-13 ·
R-4, R-5 · criteria 1, 2, 3, 7.

**Scope fences:** do not touch `pkg/game/heroart.go`, `frontend.go` or `world.go`. Add no refusal
of any kind keyed on a character's class. Do not invent a second mage weapon literal or a per-slot
mage table.

**Done when:** `go test -count=1 ./...` is green and an assembled member carries the body name and
the body directory the derivation answers for the worn set he was assembled with.

## T4 — the bundle holds more than one body, and the picture follows

**Kind:** implementation. **Boundary:** the art loader, the one preload call, and the appearance
half of the frame-paced refresh.

**Files:** `pkg/game/heroart.go` MODIFY · `pkg/game/frontend.go` MODIFY (the preload alone) ·
`pkg/game/hero.go` MODIFY (the fixed directory's removal alone) ·
`pkg/game/world.go` MODIFY · `pkg/game/heroart_test.go` MODIFY ·
`pkg/game/frontend_test.go` MODIFY · `pkg/game/heropicture_test.go` MODIFY ·
`cmd/againrom/main_test.go` MODIFY

**Covers:** FR-6, FR-7, FR-8, FR-9, AC-10, AC-11, AC-12, AC-17, AC-18, P-3, P-4, P-5 · D-3, D-6,
D-7, D-10, D-12 · R-2, R-3 · criteria 5, 6.

**Scope fences:** in `frontend.go` change only the preload call and its comment, and in `hero.go`
only the now-unreached fixed directory — `FinishMission` and everything below it belong to another
lane. In `world.go` do not touch `rearm`, `enqueueEquip`
or `refreshEquipment`; call `sim.World.SetCombat` nowhere. Change no signature in
`pkg/render/terrain`.

**Done when:** `go test -count=1 ./...` is green, the bundle's writer and its reader compose their
key through the same function, and a test drives an equipment change through the paced path and
observes the drawn class move.

## T5 — the instrument

**Kind:** developer-run verification. **Boundary:** a new command; no package under `pkg/` changes.

**Files:** `cmd/appearcheck/main.go` ADD · `cmd/appearcheck/main_test.go` ADD ·
`internal/archtest/dag.go` MODIFY (the new command's allow-map row alone)

**Covers:** FR-11, AC-13 · criterion 8.

**Scope fences:** write no file anywhere and take no output path. Contain no shipped name, count or
byte in the source. Change nothing under `pkg/`.

**Done when:** it builds, its own test covers the no-root refusal with no install present, and it
runs against a lawful install printing one block per archetype.

## Traceability

| Requirement | Plan criterion | Task |
|---|---|---|
| FR-1, AC-1, AC-2, AC-15 | 1 | T2, T3 |
| FR-2, AC-4, AC-5 | 2 | T3 |
| FR-3, AC-3, P-6 | 3 | T3 |
| FR-4, FR-5, AC-6..AC-9, AC-16, P-1, P-2 | 4 | T1 |
| FR-7, AC-17 | 5 | T1, T4 |
| FR-6, FR-8, FR-9, AC-10..AC-12, AC-18, P-3..P-5 | 6 | T4 |
| FR-10, AC-14 | 7 | T3 |
| FR-11, AC-13 | 8 | T5 |
