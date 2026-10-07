# Tasks — 0105

Four tasks, bottom up. T1 and T2 are leaves in `pkg/data`. T3 reads the install and retires the
authored name. T4 joins them at the call sites. Read `plan.md`'s matching section first: it carries
the reason for each decision below, and a task entry states only what to do.

| Task | FRs | DDs | ACs | Ps | SCs |
|---|---|---|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4 | DD-1, DD-4, DD-5 | AC-1, AC-3, AC-4, AC-5 | P-3 | SC-2, SC-3 |
| T2 | FR-6, FR-7 | — | — | P-4 | SC-4 |
| T3 | FR-3, FR-5 | DD-2, DD-3 | AC-10 | P-1, P-5 | SC-6 |
| T4 | FR-7, FR-8 | — | AC-2, AC-6, AC-7, AC-8, AC-9 | P-2, P-4 | SC-1, SC-5 |

AC-2, AC-6, AC-7 and AC-8 are measurements the verification stage takes against an installed root; a
task neither asserts them from an install nor claims them.

## T1 — the slots, the list and the derivation

**Files:** `pkg/data/equip.go`, new, and `equip_test.go`.

`EquipSlots = 12`. An `Equipment` value of twelve definition rows addressed by the original's
numbering 1..12, with a getter, a setter and an occupancy predicate that **refuse** a slot number
outside that range (FR-1). A slot carries a definition row and nothing else (FR-2). Zero is empty,
and it is the original's own encoding rather than Go's zero value — say so at the type. Nothing
enforces that only slot 1 is filled (DD-1).

`ParseBodyList([]byte) BodyList` over CRLF or LF, `BodyList` an ordered slice of `HeroBody`
(FR-3, parse half). Every line is an entry, **including an empty one** (DD-4); a trailing newline
adds no final entry.

`HeroBodyFor(l BodyList, e Equipment) (HeroBody, bool)` (FR-4): an empty slot 1 answers `l[0]`; a
slot 1 holding row `r` answers `l[r-1]`; a row before the start, past the end, or naming an empty
entry answers `"", false`. That refusal is **authored — write it as authored at the function**
(DD-5). The function takes a list and a slot value and nothing else (P-3).

Tests from values built in test code (SC-2): AC-1, AC-3, AC-4, AC-5. **No test transcribes the
shipped list** (SC-3).

## T2 — a weapon's row, and the reach law

**Files:** `pkg/data/{weapon,hero,humandef}.go` and the `_test.go` beside each.

`weapon.go`: `Weapon` gains `Row int32`, filled inside `ResolveWeapon` from the index `findByName`
already computes and discards (FR-6). No new search. **Do not relax `ResolveWeapon`'s ranged
refusal** — `pkg/mapload`'s `firstWeapon` uses it as its "is this a weapon" predicate. Note at the
field that `Row` reaches no entity, no byte form and no digest.

`hero.go` / `humandef.go`: **move** the two reach lines out of `HumanDef.Combat` and into
`Hero.Derive` — 1 for a nil weapon, `w.Range` otherwise (FR-7, law half). A move, not a copy:
`HumanDef.Combat`'s first statement already calls `Derive`, so a placed person's reach is
byte-identical afterwards and the existing tests over it must pass unchanged (P-4).

Do not edit `pkg/data/appearance.go` (SC-4).

## T3 — the install read, and the retirement of the authored name

**Files:** `pkg/game/{bodylist,hero}.go` and the `_test.go` beside each.

`bodylist.go`, new: `HeroPictureAddress`, composed from the same `mainPrefix` `EventTextPath` uses
and naming the shipped payload under `text/`; and `ReadBodyList(src entrySource)
(data.BodyList, bool)` — **`ReadEventText`'s exact signature and contract**, reporting absence
rather than erroring (FR-3, read half). Read `ReadEventText`'s doc comment first.

`hero.go`: **delete `PartyBody` and its comment.** Do not rewrite it and do not keep it as a
fallback (FR-5, P-1). Move one sentence of why it existed to the derivation's site.

`MissionParty` takes the body list beside the weapon it already takes: it builds an `Equipment` with
slot 1 from the weapon's `Row`, empty when the weapon is nil, calls `HeroBodyFor`, and hands the
answer to the existing `HeroBodyClass`. A refused derivation **carries the empty body name
forward** — already a real state — so add no second answer here. `PartyBodyDir` is unchanged.

DD-2 is the site: the derivation runs once, here. DD-3 needs no code. Nothing of the shipped payload
is committed (P-5). Tests: `MissionParty` over a fixture list, and that no body-name literal
survives on the party path (AC-10, SC-6).

## T4 — the call sites and the reach

**Files:** `pkg/game/{frontend,table}.go`, `cmd/missionrun/main.go`, `pkg/mapload/start.go`.

Load the body list where `StartWeapon` is loaded and hold it beside it, so no caller can hold one
without the other. `frontend.go`'s `LoadHeroBody` call takes the derived name where it took
`PartyBody()`; `cmd/missionrun` takes the list from the same `Defs` value it already takes
`StartWeapon` from.

`start.go`: the party entity mint gains `Reach: reachOf(c.Reach)`, the narrowing `fromalm.go`
already applies to a placed person (FR-7). **Leave the world constructor's zero-reach repair where
it is** — it is there for hand-assembled worlds.

Add no field to `sim.Entity`; the form version literal stays at **23** (FR-8). Version 24 was
allocated to this story and is **not used**. `pkg/render` and `pkg/sim` gain no import and no field
— the diff in each is empty (SC-5, P-2). Placed people are unchanged (P-4); their tests and
`pkg/sim`'s digest pin are re-run, not re-pinned (AC-9).

Run the full local gate (SC-1). The change is invisible on the default party, whose start weapon has
range 1; AC-2, AC-6, AC-7 and AC-8 are the verification stage's.
