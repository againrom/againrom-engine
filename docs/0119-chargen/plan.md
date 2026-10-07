# Plan — 0119-chargen

## Shape

Six tasks, in dependency order: the definition-table half of `pkg/data`, the pure screen model in
`pkg/ui`, the screen's wiring into the application, the pools in `pkg/mapload`, the front end in
`pkg/game`, the flag in `cmd/againrom`. Each is one commit.

The story adds no package and no import edge. It adds one screen to an existing state machine, one
field group to an existing loader input, and one flag to an existing command.

## Design decisions

**DD-1 — the screen model is a `pkg/ui` value with no game in it.** `pkg/ui/chargen.go` gains
`ChargenSetup` (a title, a list of choice rows, a list of statistic rows, a cumulative-cost table
indexed by statistic value, the budget, a confirm label), `ChargenResult` (an index per choice row
and a value per statistic row) and `Chargen`, the model. It imports nothing this package does not
already import. This is FR-3, FR-7, P-2 and P-3 in one type.

**DD-2 — a dependent choice is declared, not special-cased.** A choice row carries either a flat
option list or a list of option lists plus the index of the row it depends on. The skill row depends
on the class row; nothing in `pkg/ui` knows that, and a later story that makes the sex row matter
declares it the same way (FR-5).

**DD-3 — the cost curve crosses the tier as a table.** `pkg/ui` may not call `data.PointCost`, so the
wiring tier hands over `Cost[v]` for every `v` up to the highest ceiling. A table is also what makes
the screen's arithmetic decidable in a test without the decoded curve.

**DD-4 — refusals are silent and total.** `Adjust` computes the candidate state, tests floor,
ceiling and Remaining, and returns having changed nothing if any fails. There is no error, no
message and no partial move, which is what makes AC-3 provable by exhaustion rather than by
sampling: every reachable state is legal because no unlawful one is representable.

**DD-5 — the screen is the fourth `Screen`, appended.** `ScreenChargen` is added after `ScreenMap`
so no existing constant changes value. `flow` gains the model and the callback; `App` gains
`OpenChargen`, which arms the screen and returns before any frame. Confirming calls the callback,
and a non-nil opener is entered through the **same** `flow.enter` the picker and the mission door
already use — a third entry path would drift from the cadence rung and the command mode.

**DD-6 — two new press edges.** `appInput` gains `Left` and `Right`, read on the chargen arm alone,
beside the existing convention for the cadence and blow keys: "on every other screen they do
nothing" is then a property of where the read stands. Up/Down move the focus, Left/Right adjust,
Enter confirms, Escape returns to the menu.

**DD-7 — the shipped type id column is NOT the archetype, and the first draft of this plan said it
was.** The four rows carry 3, 14, 24 and 24 in that column on a real install, which are the drawn
class ids of the appearance name chain — swordsman, archer, mage_st — not the `0x21 + 2*class +
gender` a constructor builds at run time. The decomposition was applied to the wrong column, every
row decomposed to nothing, and every archetype silently fell back to the first row. The archetype is
therefore taken from the SLOT: the four published names are in archetype order, so `2*class + female`
indexes them.

**DD-8 — the base row is found by name at that slot.** `FindHumanByName` joins the three searches
already in `pkg/data/defsearch.go`. The resolver takes the name at the archetype's slot; if it does
not resolve it falls back to the first of the four that does; if none does, nothing. The fallbacks
are FR-8 and are what keeps an install this tree misreads playable.

**DD-9 — the profile is a method on the row, and one bit of it is authored.** The two column flags
come off the row's own health and mana maxima. The flag the pool graph multiplies by is set for a row
with NO mana column — the reading under which a fighter is the tougher of the two. The other reading
is live and named in the doc: the flag's runtime writer sets it for an actor holding a spellbook and
a pool, which swaps the two. Research resolves neither. It is the ONE expression that turns a row
into a profile (P-1) and one function to invert.

**DD-10 — the party member carries three loader inputs, not one.** `mapload.PartyMember` gains a
profile, a figure directory and a figure face, beside the existing `Body`, which is already a loader
input that reaches no entity and no byte form. The two figure fields retire
`pkg/game/inventory.go`'s authored constants: the builder reads the member instead, and falls back to
the man-fighter directory and face 1 when the member states neither, so a party assembled without
them is drawn exactly as it is today.

**DD-11 — one recompute per member.** `pkg/mapload/start.go` replaces `p.Hero.Derive(p.Weapon)`,
`p.Hero.Speed()` and `p.Hero.Sight()` — three calls, three different argument sets — with a single
`Recompute` over the member's profile and weapon, taking the combat block, the step rate, the sight
radius and the two pools off it. Speed and sight read neither the profile nor the loadout, so the
zero profile reproduces today's numbers exactly; a test pins that rather than assuming it (FR-15,
AC-1).

**DD-12 — the COLUMN gates the pool, not the positive maximum.** A profile stating a health column
and a positive maximum puts that maximum on both health fields; anything else leaves the spawn
constant on both. The same two conditions on the mana column, and zero otherwise. The first draft of
this decision gated on positivity alone and was **wrong**, caught by T4 against a landed test: the
health arm's column gate covers only its first term, so a trained character with the zero profile
derives a small positive maximum — two, for this tree's own hero — and gating on positivity would
have given him two health rather than the spawn constant. There is still no clamp, no default and no
invented column; the zero profile now genuinely reproduces today's behaviour (FR-16, AC-9).

**DD-13 — the front end resolves the row once and holds it.** `FrontEnd` already parses the
definition table. The `Humans` collection is taken off it where the starting weapon and the body list
already are, so nothing re-parses. A setup builder turns that table plus `pkg/data/chargen.go`'s
decoded arithmetic into the screen's setup; a party builder turns a result into a one-member party.
`MissionParty` keeps its name and gains the collection argument, so the default path resolves a base
row too — that is what fixes the owner's defect without the flag.

**DD-14 — the mission door takes a party.** The opener is split: the work moves into a form taking a
party, and the existing name calls it with the default. Nothing else about that closure moves; the
sack sheet, the attack pointer, the font, the start view and the readout all stay where they are.
`cmd/missionrun` keeps working through the same default.

**DD-15 — the flag requires the mission flag.** The chargen flag alone is a usage error naming both,
returned before a window exists. The alternative — a screen whose result reaches nothing — is worse
than a refusal a user can read (FR-19).

**DD-16 — the margin is measured in two domains and one of them is not clean.** The measurement is a
test in `pkg/data`. Over the whole integer experience range the worst nonzero distance to a
truncation boundary is at the values where `experience/5000 + 1` is exactly a power of 1.1 — and
there are exactly three, because `(1.1^k - 1) * 5000` is an integer only for k in 1..3. At two of
them our value sits a few units in the last place BELOW the true integer. Over the range a character
this tree can construct the margin is orders of magnitude wider than any error a logarithm can make.
The test asserts both, and `recompute.go`'s doc block is rewritten from a debt into a finding.

## Files

| Task | Files |
|---|---|
| T1 | `pkg/data/chargenbase.go` and its test (new), `pkg/data/defsearch.go`, `pkg/data/chargen.go`, `pkg/data/recompute.go` doc, a new pool-margin test |
| T2 | `pkg/ui/chargen.go` and its test (new) |
| T3 | `pkg/ui/flow.go`, `pkg/ui/app.go`, a new test beside them |
| T4 | `pkg/mapload/start.go` and a test beside it |
| T5 | `pkg/game/hero.go`, `frontend.go`, `inventory.go`, a new `chargen.go`, tests beside them |
| T6 | `cmd/againrom/main.go` and its test |

## Risks

**R-1 — three other lanes are live and two of them touch `pkg/ui`.** Every edit in T3 is additive:
one appended constant, one appended field group, one new method, one new arm in the two switches,
two new input fields. Nothing existing is reordered or renamed, and no unrelated code is tidied.

**R-2 — the default hero's health changes.** It must: it is the defect. What must NOT change is who
he is, and AC-1 pins his spread, slot, weapon, body, class key, figure directory and face against
values written out in the test.

**R-3 — the shipped rows might not carry the columns we expect.** Every failure mode lands on the
zero profile and therefore on today's behaviour. No refusal, no panic, no invented column.

**R-4 — the margin sweep is 8.3e7 iterations.** One logarithm per value, about a second and a half.
It stays a single pass and logs its own elapsed time; it does not fan out over statistics, because
the statistic term of that step is an integer added to the logarithm and cannot move its fraction.

## Out of plan

No save format, no second member, no casting, no artifact. The Russian-chrome question is recorded
elsewhere and is not touched here.

## Traceability

| Contract | Decisions |
|---|---|
| FR-1 | DD-5 |
| FR-2 | DD-1, DD-6 |
| FR-3 | DD-1, DD-3 |
| FR-4, FR-6 | DD-4 |
| FR-5 | DD-2 |
| FR-7 | DD-1 |
| FR-8 | DD-8 |
| FR-9 | DD-7 |
| FR-10 | DD-9 |
| FR-11, FR-13 | DD-13 |
| FR-12, FR-14 | DD-10 |
| FR-15, FR-17 | DD-11 |
| FR-16 | DD-12 |
| FR-18 | DD-16 |
| FR-19 | DD-15 |
| FR-20, FR-21 | DD-14 |

**DD-17 — `-skill` seeds the screen, it is not a second path to the same field.** `0120` landed a
`-skill` flag that sets the slot the party's hero trains. A screen that also set it would give one
field two writers. So the flag keeps its meaning when it is alone and, under `-chargen`, becomes the
skill row's **opening index**: `ChargenChoice` gains a `Start`, the setup builder fills it from the
front end's current slot, and the player's own move still wins.

**DD-18 — the picker's missions.** `0121` made the map list open campaign missions through the same
door `MissionParty` is spelled at. Whether a generated character reaches those entries is decided in
T8 and stated in the code, not left to be discovered.

**FR-22** landed as hotfix `7e3176c`, not as a task of this story; folded into the contract by 0130.
