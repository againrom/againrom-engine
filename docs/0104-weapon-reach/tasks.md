# Tasks — 0104

Three tasks, bottom up. T1 is a leaf. T2 gives the simulation the field, the distance and the
version, and touches no placement. T3 joins them.

| Task | FRs | DDs | ACs | Ps |
|---|---|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-5 | — | AC-1 | P-4, P-7 |
| T2 | FR-7, FR-8, FR-9, FR-10, FR-11 | DD-1, DD-3, DD-4, DD-6, DD-7 | AC-3, AC-4, AC-5 | P-1, P-2, P-5, P-6, P-8 |
| T3 | FR-4, FR-6 | DD-2, DD-5 | AC-2, AC-6 | P-3 |

AC-1, AC-2, AC-5 and AC-6 are measurements the verification stage takes against an installed root; a
task neither asserts them nor reads an install.

## T1 — the range resolver and the combat field

**Files:** `pkg/data/{weapon,hero,unitdef,humandef}.go` and the `_test.go` file beside each.

`weapon.go`: an unexported helper that removes a trailing `{…}` and trims, called as the first
statement of `ResolveWeapon` and of the new exported `WeaponRange` (FR-2); `WeaponRange` is the same
prefix walk with a different tail and its own guard on `weaponRangeSlot` (FR-1).

**Do not relax `ResolveWeapon`'s ranged refusal** — `pkg/mapload` uses it as a predicate. Read the
paragraph under FR-1 in `plan.md` first.

`hero.go`: `Combat` gains `Reach int32`; `HumanDef.Combat(w)` fills it from `w.Range` or 1 when `w`
is nil (FR-3, FR-5). `UnitDef.Combat()` copies `d.Reach` (FR-3). Rewrite the `Reach` comment in
`unitdef.go`, which currently calls equipment a disclosed divergence.

Tests, from fixture collections: a name with a `{…}` suffix and one without; a range cell of `−1`
and one of 20; a row shorter than the range slot; a name matching no row; a ranged row `WeaponRange`
answers and `ResolveWeapon` still refuses (P-7); a bare definition of each kind yielding 1 (P-4).

## T2 — the entity field, the distance and version 23

**Files:** `pkg/sim/{world,combat,binary}.go`, their `_test.go` siblings, one new test file.

`world.go` (FR-7): `Reach uint8` beside `ScanRange`; `reachFault` beside `transitFault`. **The
constructor NORMALISES a zero to 1 and only the decoder REFUSES one** (DD-3) — backwards turns
every entity literal in the package red.

`combat.go` (FR-8, FR-9, FR-10): delete `const reach` and its comment; add `strikeDistance(a, t
Entity) int32` in the 1/256-cell form FR-8 gives, footprint term written out at its size-1 value.
`inReach` keeps its name and becomes `strikeDistance(a, t) <= int32(a.Reach)`. Rewrite the sentence
in `approach`'s comment calling the stop distance a Chebyshev distance of one cell.

`binary.go` (FR-11): `formatVersion` **23**, `entityLen` **119**, the byte at record offset 118, the
offset table row, a version paragraph in the voice of those above it, `reachFault` on decode.

**Landed tests go red and are meant to:** every digest and byte-form pin in `pkg/sim` and
`pkg/mapload`. **Re-pin from a run, old -> new in the commit body.** Red otherwise is a stop.

Tests: a round trip over several reaches; 0 refused, 255 accepted, version 22 refused; the
constructor folding 0 to 1; `strikeDistance` against `max(1, Chebyshev)` at every separation 0..24;
reach 4 striking at 4, refusing at 5, stopping its walk at 4.

## T3 — the placement reads the row's own weapon

**Files:** `pkg/mapload/{spawn,fromalm}.go`, their `_test.go` siblings, one new test file.

`spawn.go` (FR-4): `unitReach(names []string, t *Table) int32` beside `firstWeapon` — same shape,
same three-collection guard, returning the first `data.WeaponRange` that resolves and 1 when none
does. `definitionFor` calls it with `c.EntryStrings(r.Index)` and assigns `d.Reach` before it
returns. Assigning the range where the original adds `range − 1` to a reach of 1 is DD-2.

**Take the range and nothing else.** The same weapon row carries a charge and a relax column and the
original's equip does write them over the class template. That is a separate story: do not touch
`AttackChargeTime` or `AttackRelaxTime` (DD-5).

`fromalm.go`: `blockFor` needs no change on either resolved arm, and its unresolved arm already
substitutes the constructor defaults whole; `Adjust` is unchanged (FR-6). Carry `combat.Reach` onto
the entity in the one composite literal that spends a `spawnBlock`, narrowing to the byte the way
`sightOf` narrows the scan range.

Tests: a row naming a bow yields the bow's range; a melee weapon, no string, and an unresolvable
string each yield 1 with no load error; a placement through `FromALM` carries it onto the entity;
`Adjust` at each difficulty leaves it alone.
