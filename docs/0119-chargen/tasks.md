# Tasks — 0119-chargen

## T1 — the shipped row a generated character starts from *(implementation)*

New `pkg/data/chargenbase.go` + test, for DD-7, DD-8, DD-9 and FR-8 to FR-11.

`HeroArchetype(typeID int32) (class, female, ok bool)` decomposes `0x21 + 2*class + gender`; ids
outside `0x21..0x24` answer `ok == false`. One authored constant says which addend is female.
`HeroTypeID(class, female bool) int32` is its inverse.

`ChargenBaseNames() []string` returns, in this order: `PC_Danath`, `PC_Naira`, `PC_Fergard`,
`PC_Reniesta`. A function, not a variable. `FindHumanByName(c Collection, name string) int` joins
the searches in `defsearch.go`. `ChargenBase(c Collection, class, female bool) (HumanDef, bool)`
implements DD-8's three-step fallback exactly.

`func (d HumanDef) Profile() Profile` implements DD-9. Its doc must state that this tree reads the
bit and does not decide which archetype it names.

In `pkg/data/chargen.go`: `MageSkillName(slot int32) string` giving `Fire`, `Water`, `Air`, `Earth`,
`Astral` for slots 1..5 beside the warrior names, and `SkillNames(mage bool) []string` returning the
five in slot order.

Also this task: the pool-margin measurement, a new test file, for DD-16 and AC-13 — and rewrite
`recompute.go`'s `logBase11` doc and `Derived`'s "wired to nothing" paragraph into the finding.

Tests are synthetic: a `Collection` built in test code.

## T2 — the generation screen's model *(implementation)*

New `pkg/ui/chargen.go` + test, for FR-2 to FR-7 and AC-2 to AC-6 by DD-1, DD-2, DD-3, DD-4.

`ChargenSetup{Title string; Choices []ChargenChoice; Stats []ChargenStat; Cost []int; Budget int;
Confirm string}`. `ChargenChoice{Name string; Options []string; OptionsFor [][]string; Parent int}`
— `Parent` is an index into `Choices` and is `-1` for a flat row. `ChargenStat{Name string; Floor,
Ceiling, Start int}`. `ChargenResult{Choices []int; Stats []int}`.

`NewChargen(ChargenSetup) *Chargen`, then `Move(d int)` (wrapping focus over choice rows then
statistic rows), `Adjust(d int)`, `Focus() int`, `Rows() int`, `RowText(i int) string`,
`HeaderText()`, `FooterText()`, `Remaining() int`, `Legal() bool`, `Result() (ChargenResult, bool)`.

`Adjust` on a choice row cycles its option index within the row's CURRENT option set; on a statistic
row it steps by `d` and refuses per DD-4. `Result` answers false while illegal.

The package's import list does not change. No engine type, no clock, no randomness.

## T3 — the screen inside the application *(implementation)*

`pkg/ui/flow.go` and `pkg/ui/app.go` + a new test, for FR-1 and AC-12's window half, by DD-5 and
DD-6. **Every edit is additive** (R-1): append, do not reorder, rename or tidy.

`ScreenChargen` is appended after `ScreenMap`, with its `String()` arm. `flow` gains the model
pointer and the callback. `appInput` gains `Left` and `Right` press edges, read on the chargen arm
alone, populated in `readAppInput` beside the existing keys.

`func (a *App) OpenChargen(c *Chargen, begin func(ChargenResult) (MapOpener, error)) error` arms the
screen and returns; it opens no window and enters no map.

`stepChargen` handles Up/Down/Left/Right/Enter. Enter reads `Result()`; while illegal it sets the
flow's message and stays. Otherwise it calls `begin` and enters a non-nil opener through the SAME
`flow.enter` the picker and the mission door use, then syncs the viewer layout; an error from
either sets the message and stays. Escape returns to the menu, in the existing Escape arm.

`drawChargen` prints the header, one line per row with a marker on the focused row, the remaining
counter and the confirm line, through the picker's own debug-font call. Labels are English.

Tests drive `step` directly with synthetic input and assert the screen, the focus, the refusals and
that a confirm reaches the callback exactly once.

## T4 — a hero's statistics reach his health *(implementation)*

`pkg/mapload/start.go` + a test, for FR-14 to FR-17 and AC-8, AC-9, AC-10 by DD-10, DD-11, DD-12.

`PartyMember` gains `Profile data.Profile`, `FigureDir string` and `FigureFace int`. Each is a
LOADER INPUT beside `Body`: none reaches an entity, a byte form or a digest, and the doc block must
say so. No field is added to `sim.Entity` and `pkg/sim` is not touched at all, so the byte form's
version constant is not spent.

The member loop calls `Recompute` ONCE over the member's profile and a loadout carrying his weapon,
and takes the combat block, the step rate, the sight radius and the two pools off that one value.
The three existing calls go.

The health pair takes the recompute's health maximum on both fields when it is positive and
`SpawnHP` on both otherwise. The mana pair takes the mana maximum on both fields when positive and
stays zero otherwise. The two regeneration periods are unchanged.

Tests: a member with a health column has a maximum equal to the recompute's and it MOVES WITH BODY;
a member with the zero profile is byte-for-byte today's member; a member with a mana column has a
positive mana pair. Witness the defect by reverting the health line and showing the first test go
red — record it in the commit body, not in a file.

## T5 — the front end assembles a generated character *(implementation)*

`pkg/game/hero.go`, `frontend.go`, `inventory.go`, a new `pkg/game/chargen.go` + tests, for FR-8,
FR-10 to FR-13, FR-21 and AC-1, AC-7 by DD-13, DD-14.

`FrontEnd` gains the `Humans` collection off the already-parsed table, where `StartWeapon` and
`Bodies` are taken.

New `pkg/game/chargen.go`: `ChargenSetup()` builds the setup — the statistic names in display order
Body, Reaction, Mind, Spirit, the bounds, start, cost table and budget from `pkg/data`, a sex row, a
class row, and a skill row whose option lists are `data.SkillNames(false)` and `data.SkillNames(true)`
with the class row as parent. `ChargenParty(res)` turns a result into a one-member party: base row,
profile, spread, slot, weapon, body, class key, figure directory and face.

`MissionParty` gains the collection argument and resolves a base row for THIS TREE'S DEFAULT axes,
so the no-flag path gets a real profile; its spread, slot, weapon, body and class key do not move.
`MissionOpenerWith(n, party)` is split out and `MissionOpener(n)` calls it with the default party.
`MissionLine` reports the hero's health and mana pair. `inventory.go`'s two constants become the
member's fields, old values as fallback (DD-10).

AC-1 is a test naming today's values outright: 43/26/15/15, Blade, the same weapon, body and class
key, `mfighter`, face 1.

## T6 — the flag the owner types *(implementation)*

`cmd/againrom/main.go` + its test, for FR-19, FR-20, FR-21 and AC-12 by DD-15.

`-chargen` is a bool flag with its own named help string, beside the existing named ones. Given
without `-mission` it is a usage error naming both flags, reported on the error stream with exit
status 2, before any archive is opened.

In the check mode it prints one line describing what the screen would offer — the row names and the
budget — and exits without a window.

In the windowed mode, given with `-mission`, it builds the setup from the front end, opens the
chargen screen through `OpenChargen`, and the confirm callback returns the mission opener built over
the generated party. Absent the flag, not one of those statements is reached and the existing
mission door is called exactly as it is today.

The package doc comment gains the flag, with the same care the existing ones get: what it does, what
it requires and that without it nothing changes.

Tests: the flag's default is false; `-chargen` without `-mission` exits 2 and names both; `-check
-mission N -chargen` against a synthetic install prints the extra line and opens no window.

## T7 — the archetype is the slot, not a column that does not carry it *(implementation)*

`pkg/data/chargenbase.go` and its test, for the rewritten FR-8, FR-9, FR-10 by DD-7, DD-8, DD-9.
Nothing outside `pkg/data` changes.

`HeroArchetype` and `HeroTypeID` go: they decompose a runtime type id no shipped column carries, and
applying them to `HumanDef.TypeID` made every archetype resolve to one row. The archetype is the
SLOT — `ChargenBaseNames` is in archetype order, so `2*class + female` indexes it. Keep the finding
in the file header with the observed column values 3, 14, 24, 24 and what they are.

`ChargenBase(c, class, female)` takes the name at that slot; failing that the first of the four that
resolves; failing that the zero row and false.

`HumanDef.Profile()` keeps its two column reads and sets the pool graph's class flag for a row with
**no** mana column. Its doc names the other reading — the flag's runtime writer sets it for an actor
holding a spellbook and a pool — says research resolves neither, and says this is the one function
to invert.

Tests, synthetic: each archetype reaches its own slot; an unresolvable slot falls back; none
resolving answers false; the flag follows the mana column. Pin that a row whose type id column holds
a drawn class id still resolves — the regression this task exists for.

## Traceability

| Task | Contract | Decisions |
|---|---|---|
| T1 | FR-18 | DD-16 |
| T2 | FR-2, FR-3, FR-4, FR-5, FR-6, FR-7 | DD-1, DD-2, DD-3, DD-4 |
| T3 | FR-1 | DD-5, DD-6 |
| T4 | FR-14, FR-15, FR-16, FR-17 | DD-10, DD-11, DD-12 |
| T5 | FR-11, FR-12, FR-13 | DD-13, DD-14 |
| T6 | FR-19, FR-20, FR-21 | DD-15 |
| T7 | FR-8, FR-9, FR-10 | DD-7, DD-8, DD-9 |
| T8 | FR-11, FR-19 | DD-17, DD-18 |

**FR-22** landed as hotfix `7e3176c`, not as a task of this story; folded into the contract by 0130.

## T8 — the tree after the correction and after master *(implementation)*

`pkg/game/chargen_test.go`, `pkg/game/chargen.go`, `pkg/ui/chargen.go`, `cmd/againrom/main.go` and
their tests. It closes T7's downstream and folds in what landed under this lane.

**The fixtures.** T7 made the profile's class flag follow the mana column; four tests in
`pkg/game/chargen_test.go` still assert the type-id decomposition it replaced, and their prose still
cites it. Fix both, and check every fixture reaches the row it names now that the archetype is the
SLOT — a fixture keyed on a type id proves nothing any more.

**`-skill` is superseded, not duplicated (DD-17).** It already sets the slot the hero trains; the
screen must not be a second writer of one field. It **seeds** the skill row instead:
`ChargenChoice` gains a `Start int`, default 0, honoured and clamped by `NewChargen`, filled by the
setup builder from the front end's current slot. `-skill` alone behaves exactly as it does.

**Decide whether `-chargen` reaches the picker's missions (DD-18).** `0121` made the map list open
campaign missions through the door `MissionParty` is spelled at. Implement one answer and say which
in the doc comment.

Tests: the four repaired; a seeded row opens on that skill and still moves; the screen's result wins
over the flag. Full gate green in every package.
