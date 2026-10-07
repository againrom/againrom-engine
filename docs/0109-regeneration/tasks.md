# Tasks — 0109

Four tasks, bottom up. T1 moves every pinned digest in two packages and adds no behaviour. T2 adds
the behaviour and, by P-4, moves no digest. T3 moves the loader's. T4 is a leaf on the display
side.

| Task | FRs | DDs | ACs | Ps |
|---|---|---|---|---|
| T1 | FR-1, FR-8, FR-9 | DD-3, DD-9, DD-11 | AC-6 | — |
| T2 | FR-2, FR-3, FR-4, FR-5, FR-6 | DD-1, DD-2, DD-4, DD-5, DD-6, DD-7, DD-10 | AC-1, AC-2, AC-3, AC-4, AC-5 | P-1, P-2, P-3, P-4, P-5 |
| T3 | FR-7 | DD-8, DD-12 | AC-7 | — |
| T4 | FR-10 | — | AC-8 | — |

AC-9 and AC-10 are measurements the verification stage takes against an installed root; a task
neither asserts them nor reads an install.

## T1 — the six fields, the fault and version 25

**Files:** `pkg/sim/{world,binary}.go` and the `_test.go` file beside each.

`world.go` (FR-1, FR-8): the six fields at `Entity`'s tail; `regenFault` beside `decayFault`, and
**the decoder its only caller**; the constructor folding each remainder above 99 to 0 inline.
**No clause in the not-alive block** (DD-9).

`binary.go` (FR-9): `formatVersion` **25**; `entityLen` grown by 18; the six writes and reads at
the record's tail; the offset-table rows; `regenFault` on decode; one version paragraph. **Take
both constants off the branch point**; 24 is live there.

**Rename both version-named tests to names carrying no number** (DD-11), taking the refusal's two
numbers from `formatVersion` and a literal `preRegenFormVersion`.

**Landed tests go red and are meant to:** every digest and byte-form pin in `pkg/sim` and
`pkg/mapload`, and `nostate_test.go`'s ordered field list. Re-pin from a run, old -> new in the
commit body. Red otherwise is a stop.

Tests: a round trip over each field at its extremes; a remainder of 100 and of 255 refused on
decode and folded by the constructor; the previous version refused with both numbers named; the
stripped form, **version byte restored**, hashing to the previous story's digest.

**Done when:** the form carries the six, the previous version is refused, every pin is re-taken.

## T2 — the regeneration pass

**Files:** new `pkg/sim/regen.go`, `pkg/sim/step.go`, new `pkg/sim/regen_test.go`.

`regen.go`: the six constants, `regenerate` and `regenPass`, exactly as `plan.md` shapes them.
`step.go`: one call before `w.decayPass()` with its own comment, in the voice of the phase
comments around it.

**Both hundreds stay in the expression** (DD-4) and **the remainder is stored whether or not the
pool was capped**. Cancelling them, or dropping the remainder, gives a pass that is green
everywhere except AC-2.

**The non-negative remainder is three lines and is required** (DD-7): the quotient must be taken
from the same adjusted accumulator the remainder is, or the two disagree by a point. **The
`max <= 0` gate is required too** (DD-10) — without it the mana arm runs backwards forever.

Tests: the exact hundredths a qualifying tick adds, per arm; a unit of maximum 45 at period 100
driven from 40 to full; the two cadences read off the tick index; both caps; a zero and a negative
period on each arm with the other still working; a dead, a downed and a 0-health unit unchanged; a
negative maximum on each arm leaving the pool alone; a world with no periods whose every pinned
digest is what T1 left it (P-4); the RNG state unmoved (P-1).

**Done when:** the two arms behave as FR-3 and FR-4 state, the source scan is green, no digest
moved.

## T3 — a placement is born with its periods and its pool

**Files:** `pkg/mapload/{fromalm,start}.go` and the `_test.go` file beside each.

`fromalm.go` (FR-7): four fields on `spawnBlock`; the creature arm off `UnitDef`, the humans arm
taking the mana pair off `HumanDef` and **both periods from `data.UnitDefaults()`** — the Humans
table has no such column and a literal 100 or 50 written here would be a second source for a
number that already has one. **The unresolved arm names all four too**, off the definition it
already holds. Carry all four onto the entity in the one composite literal that spends a
`spawnBlock`. `Adjust` is unchanged, and DD-12 says what that means.

`start.go` (FR-7, DD-8): the two periods from the same `data.UnitDefaults()` on the party mint,
and **no mana pair** — do not invent one, and do not derive one from Spirit.

**Landed tests go red and are meant to:** the loader's digests. Re-pin as T1 does.

Tests: a placement whose row names periods and a mana pair carries all four; one resolving to no
row carries the constructor's; the humans arm carries 100/50 with the row's pool; a started party
member carries 100/50 at 0/0 mana; `Adjust` at each difficulty leaves all four alone.

**Done when:** every path that builds a world from a map names all four numbers.

## T4 — the panel says how much mana

**Files:** `pkg/ui/{panel,overlay}.go`, `pkg/game/world.go`, and the `_test.go` file beside each.

`panel.go` (FR-10): `PanelFieldMana`; `Mana, MaxMana int` on `PanelSubject`; one `panelText` arm
returning the health arm's format and reporting `s.MaxMana > 0`; one layout row labelled `MANA`
directly under the health row; the two fields read off the entity in the one place `PanelSubject`
is built.

`overlay.go`: the same pair on `MapEntity`, beside its health pair. `world.go`: fill it at the one
site that builds a `MapEntity`.

**Nothing here reads a period or a remainder.** The panel shows a pool, not a rate.

Tests: a subject with a pool draws the row and one without omits it; the text matches the health
row's form; the pair reaches the panel from a stepped world.

**Done when:** a unit with mana shows it and a unit without shows no empty row.
