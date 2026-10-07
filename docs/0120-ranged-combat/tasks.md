# Tasks — 0120-ranged-combat

**Reading key.** `FR-n`, `D-n` → `spec.md` §The contract, §Divergences. `AC-n`, `P-n` →
`spec.md` §Acceptance, §Derived properties. `DD-n` → `plan.md` §Design decisions. `R-n` →
`plan.md` §Risks. `SC-n` → `plan.md` §Success criteria.

Tasks are ordered, and each leaves `go build ./...`, `go vet ./...`, `gofmt` and
`go test -count=1 ./...` green on its own.

---

## T1 — the equip fold grows its ranged arm *(implementation)*

**Boundary.** `pkg/data` only. A ranged weapon resolves and folds; nothing that reads a weapon
outside this package changes yet.

**Files**

- `pkg/data/weapon.go` — `MODIFY` (delete the ranged refusal; add `Ranged()` on `Weapon`)
- `pkg/data/foldweapon.go` — `ADD` (`FoldWeapon(Combat, *Weapon) Combat`, DD-3, DD-6)
- `pkg/data/recompute.go` — `MODIFY` (the equipment step calls `FoldWeapon`)
- `pkg/data/foldweapon_test.go` — `ADD`; `pkg/data/weapon_test.go` — `MODIFY`

**Covers** FR-1, FR-2 · DD-1, DD-2, DD-3, DD-6 · D-1, D-2, D-3 (AC-1, AC-2, P-1, P-2)

**Fences.** Do not add a damage component or a protection selector — D-1, and it is a whole
story. Do not change how any number is *scaled*; only where it lands. Do not touch `pkg/sim`,
`pkg/mapload` or `pkg/game`. Do not change `WeaponRange`. Keep the melee arm's arithmetic
byte-identical to what `Recompute` does today.

**Done when:** a ranged row resolves and reports its attack type; folding a ranged weapon moves
reach and cadence and leaves damage, to-hit and defence exactly as found; folding a melee weapon
gives the numbers `Recompute` gives today; and reverting the branch turns the ranged test red.

---

## T2 — a creature folds its own equipment *(implementation)*

**Boundary.** `pkg/mapload`. What a placed creature arrives with, not how it fights.

**Files**

- `pkg/mapload/spawn.go` — `MODIFY` (`definitionFor` folds; a resolution beside `unitReach`)
- `pkg/mapload/reach_test.go` — `MODIFY`; `pkg/mapload/unitequip_test.go` — `ADD`

**Covers** FR-3, FR-4 · DD-4, DD-5, DD-6 · R-2, R-3 · D-5 (AC-3, AC-4)

**Fences.** Fold **after** the difficulty adjustment, where the reach assignment is today
(DD-5). The fold **adds** damage, to-hit and defence and **assigns** cadence and reach — an
assignment there would silently wreck 26 shipped classes (R-3). Keep `unitReach` and its
fallback: a name that still does not resolve must yield exactly today's reach (FR-4, DD-4). Do
not read the brace clause (D-5). Do not touch the humans arm, `firstWeapon`, or any package
outside `pkg/mapload`.

**Done when:** a class row naming a melee weapon arrives with that weapon's numbers added to its
row's own; one naming a ranged weapon arrives with its row's numbers and the weapon's reach;
every reach a synthetic shipped-shaped table produces is unchanged; and reverting the fold turns
the addition test red.

---

## T3 — the hero's trained skill stops being a constant *(implementation)*

**Boundary.** `pkg/game` and `cmd/againrom`. One setting, today's default.

**Files**

- `pkg/game/hero.go` — `MODIFY` (`PartySkillSlot` becomes a read accessor over package state;
  add a setter that refuses a slot trained for no weapon)
- `pkg/game/frontend.go`, `pkg/game/table.go` — `MODIFY` (the two call sites)
- `cmd/againrom/main.go` — `MODIFY` (a `-skill` flag applied before the front end loads)
- `pkg/game/hero_test.go`, `cmd/againrom/main_test.go` — `MODIFY`

**Covers** FR-5 · DD-7 · R-1 (AC-5)

**Fences.** Keep the default exactly what it is today, so an unset flag changes nothing. The
setter must be callable only before the front end loads and must refuse an out-of-range slot.
Do not restructure `NewFrontEnd`, `LoadDefinitions`, `MissionParty` or the `Definitions` struct
— another lane is inside character generation (R-1). Do not add a second skill constant
anywhere. Nothing in `pkg/sim`, `pkg/data` or `pkg/mapload`.

**Done when:** the flag set to the shoot skill yields a hero holding that skill's bow with a
reach above 1; unset yields today's hero and today's weapon exactly; an out-of-range value is
an error the command reports rather than a panic.

---

## T4 — a shot in flight *(implementation, and the declared cut)*

**Boundary.** Drawing only. A mark between a ranged attacker and its victim.

**Files**

- `pkg/ui/overlay.go` — `MODIFY` (one optional point on `MapEntity`, DD-8)
- `pkg/game/world.go` — `MODIFY` (fill it beside the swing path, DD-9)
- `pkg/ui/viewer.go` — `MODIFY` (draw it); a named colour beside the marker colours (DD-10)
- the matching `*_test.go` — `MODIFY`/`ADD`

**Covers** FR-6, FR-7 · DD-8, DD-9, DD-10 · D-4 · R-4 (AC-6, P-4)

**Fences.** Integer arithmetic only, and no attack, reach or cadence rule restated in `pkg/ui`
(FR-7, DD-8). Do not resolve the registry's projectile class, shoot delay or shoot offset (D-4).
Do not add a field to any type in `pkg/sim`, do not touch the byte form, and do not repin a
digest (AC-7). If this task cannot land inside one commit without disturbing T1 to T3, drop it
and say so — it is the declared cut (R-4).

**Done when:** a reach-above-1 attacker mid-cycle against a target two or more cells away puts
exactly one mark on the seam, on the segment between them and advancing with the cycle; an
adjacent attacker and a reach-1 attacker put none; and no digest or version moves.

---

## Traceability

FR-1, FR-2 → T1. FR-3, FR-4 → T2. FR-5 → T3. FR-6, FR-7 → T4. D-1, D-2, D-3 → T1. D-4 → T4.
D-5 → T2. AC-1, AC-2 → T1. AC-3, AC-4 → T2. AC-5 → T3. AC-6 → T4. AC-7 → every task, by not
touching `pkg/sim`. P-1, P-2 → T1. P-4 → T4.
