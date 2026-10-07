# Tasks — 0087-enemy-combat-block

**Kinds:** `impl` — one commit, trailer `SDD-Task: 0087-enemy-combat-block/T<n>`.
`verification.md` and the build folder are stages, not tasks: they commit untrailered.

**Order is strict** for T1 to T5: T2 needs T1's method, T3 needs T2's type, T4 needs T3's, T5 needs
T4's fields. T6 depends on none of them.

---

## T1 — a collection entry admits its trailing strings  *(impl)*

**Files:** `pkg/formats/databin/databin.go`, `pkg/data/defsearch.go`, and the three test fakes that
implement `data.Collection` (`pkg/data/defsearch_test.go`, `pkg/mapload/spawn_test.go`,
`pkg/game/world_test.go`).

**Covers:** plan DD-1.

**Fence:** nothing reads the new method yet, and no behaviour anywhere changes. The fakes gain the
method and return whatever they already hold for a string-less entry.

**Done when:** the whole tree builds and the suite is green with the interface widened, and the
format tier's accessor is exercised by the parser's own test against its existing fixture.

---

## T2 — the twenty-three slots  *(impl)*

**Files:** `pkg/data/humandef.go`, `pkg/data/humandef_test.go`.

**Covers:** FR-1, AC-1, AC-2, AC-3, P-1, P-4; plan DD-2, DD-3.

**Fence:** the type and its loader only. No derivation, no weapon, no caller — nothing outside
`pkg/data` names it after this task.

**Done when:** every slot is witnessed landing in its own field and in no other by a row empty
everywhere else; an all-empty row equals the defaults; a short row is refused by name with the zero
value; and no field of a loaded definition holds the empty value for any row the tests build.

---

## T3 — what a person's blow is made of  *(impl)*

**Files:** `pkg/data/humandef.go`, `pkg/data/unitdef.go`, `pkg/data/humandef_test.go`.

**Covers:** FR-2, FR-4, AC-5, AC-6, AC-9, P-2, P-3; plan DD-4.

**Fence:** `pkg/data` only. The other collection's type gains its accessor and NOTHING else — its
loader, its fields and its defaults are untouched, and the accessor must be witnessed returning
exactly what that type already carries.

**Done when:** a bare definition's eight are the derivation's, with the template's own cadence
standing in place of the hero's bare pair; an armed one differs from it by exactly that weapon's own
four numbers with the cadence assigned per half on the empty cell; and the same inputs yield the same
eight twice over.

---

## T4 — the loader builds a person  *(impl)*

**Files:** `pkg/mapload/spawn.go`, `pkg/mapload/fromalm.go`, `pkg/mapload/combat_test.go`,
`pkg/mapload/spawn_test.go`, `pkg/mapload/fromalm_test.go`, `pkg/mapload/human_test.go`,
`cmd/classdump/databin.go` and its test.

**Covers:** FR-3, FR-5, FR-6, FR-7, AC-4, AC-7, AC-8, AC-10, AC-11, AC-12, AC-12a; plan DD-5, DD-6,
DD-7, R-1, R-5, R-6.

**Fence:** no file under `pkg/sim` and none under `pkg/game`. The existing pinned digest and form
length in `pkg/mapload/fromalm_test.go` are not edited, re-recorded or re-derived — they must stay
green as they stand, and a change to either is a defect in this task rather than a number to update.
Every OTHER assertion this task repairs keeps its own question and is re-aimed at the population that
still resolves to nothing; none is deleted, and none may neutralise a number before comparing it.

**Done when:** each of the three humans rungs reaching one entry yields one set of numbers; the
weapon is chosen by resolving rather than by position, with the ranged and the unresolvable cases
both leaving the placement bare; a table missing the item collections raises nothing; the health,
the rate and the cadence are the row's; all three difficulty values agree on this arm; and the other
two arms are byte-identical at every difficulty.

---

## T5 — the front end's table carries them  *(impl)*

**Files:** `pkg/game/table.go`, `pkg/game/table_test.go`.

**Covers:** plan DD-8; the precondition of AC-14.

**Fence:** the one walk that already happens. No second read of the file, no new failure mode, and
`Definitions`' other field is untouched.

**Done when:** the table a front end and the mission tool receive carries the three item collections
off the same parse that answers the party's own weapon, and the test witnesses that a table built
from a file yields a placement an armed one.

---

## T6 — one answer to an empty cadence cell  *(impl)*

**Files:** `pkg/data/hero.go`, `pkg/data/hero_test.go`.

**Covers:** FR-4's second paragraph; plan DD-10.

**Fence:** the cadence assignment alone. No other term of the derive moves, and the bare pair's own
two constants are untouched.

**Done when:** a generated character holding a weapon whose charge cell is empty keeps his bare
charge and takes the weapon's relax, and the mirror case holds — witnessed as a difference from
today's behaviour rather than as agreement with it.

---

## T7 — the report may not call an armed row unequipped  *(impl)*

**Files:** `cmd/classdump/databin.go`, `cmd/classdump/main.go`, `cmd/classdump/databin_test.go`.

**Covers:** FR-3 and FR-7's consequence for the tool that witnesses them.

**Fence:** the report's own wording and one added column. No number in any row moves, and no
resolution changes; the creature band's caveat must survive intact rather than be softened.

**Done when:** the report says which band each row is, states the equipment caveat for the creature
band ALONE, and the check that used to pin the blanket sentence asserts the split instead — a check
whose expectation is the defect cannot fail on it.

---

## Traceability

| Upstream | Task |
|---|---|
| FR-1 | T2 |
| FR-2 | T3 |
| FR-3 | T4 |
| FR-4 | T3, T6 |
| FR-5 | T4 |
| FR-6 | T4 |
| FR-7 | T4 |
| FR-8 | T1–T5 by fence: no task names a file under `pkg/sim` |
| AC-1, AC-2, AC-3 | T2 |
| AC-4 | T4 |
| AC-5, AC-6 | T3 |
| AC-7, AC-8 | T4 |
| AC-9 | T3, T6 |
| AC-10, AC-11, AC-12, AC-12a | T4 |
| AC-13 | T1–T5 by fence; witnessed in verification |
| AC-14 | T5, witnessed in verification |
| P-1, P-4 | T2 |
| P-2, P-3 | T3 |
| P-5 | verification |
| DD-1 | T1 |
| DD-2, DD-3 | T2 |
| DD-4 | T3 |
| DD-5, DD-6, DD-7 | T4 |
| DD-8 | T5 |
| FR-3, FR-7 in the tool | T7 |
| DD-10 | T6 |
| DD-9 | every task's fence |
| R-1 | T4 |
| R-2 | T1 |
| R-5, R-6 | T4 |
| SC-1 | T2, T3, T4 |
| SC-2 | T4, T5 |
| SC-3 | T4 |
| SC-4 | every task's fence |
| SC-5 | verification |
