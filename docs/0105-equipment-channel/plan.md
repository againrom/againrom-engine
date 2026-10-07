# Plan — 0105

Three tiers, bottom up, and the story is small because `0085` already built everything downstream of
a body name. What is added is the step in front of it: a slot, a row, a list, and one lookup.

## The shape

`pkg/data` gains the law and no I/O — the slots, the list type, the derivation, and a weapon's own
row. `pkg/game` gains the one read from the install and the retirement of the authored name.
`pkg/mapload` gains one field written where none was written before. The drawing tier gains nothing
at all (P-2), and neither does `pkg/sim` (FR-8).

## FR-1, FR-2, DD-1, AC-1 — the slots

`pkg/data/equip.go`, new. `EquipSlots = 12` and an `Equipment` value holding twelve `int32`
definition rows, addressed by the original's numbering 1..12 rather than by a Go index. A getter, a
setter and an occupancy predicate; a slot number outside 1..12 is refused rather than clamped, which
is what keeps FR-1's "no thirteenth slot and no slot 0" checkable, and what AC-1 asks of it.

**Zero is empty, and that is the original's own encoding rather than a convenience** — the sender
writes 0 for an unoccupied slot instead of omitting it, and no real item resolves to row 0 because
row 0 of the shipped collection carries no name. Say so at the type, because a reader who does not
know it will read the zero as Go's zero value and think the distinction was skipped.

Nothing in this story sets a slot other than 1 (DD-1). The type does not enforce that: a rule that
only slot 1 may be filled would have to be undone by the story that fills the others, and the eleven
empty slots are the disclosure, not a restriction.

## FR-3, P-5 — the shipped list

Two halves, split on I/O, the way this tree already splits every payload.

**The parse is `pkg/data`.** `ParseBodyList([]byte) BodyList` over CRLF or LF, where `BodyList` is an
ordered slice of `HeroBody`. **Every line is an entry, including an empty one** (DD-4), and a
trailing newline does not add a final empty entry — that is the one line-splitting decision worth
stating, because it is the difference between the shipped payload having 26 entries and 27.

**The read is `pkg/game`.** `HeroPictureAddress` composed from the same `mainPrefix` the event-text
address is composed from, plus `ReadBodyList(src entrySource) (data.BodyList, bool)` — deliberately
`ReadEventText`'s exact signature and exact contract, one file over. It reports absence rather than
erroring, because a front end that cannot read the list still opens missions; what it loses is the
derivation, and FR-4's totality already says what an empty list answers.

P-5 is structural: the payload is read at run time and nothing is committed. `check-no-game-assets.sh`
is the standing check, and no test may assert the file's contents from a copy held here — AC-2 is a
**verification-stage measurement against both installs**, not a unit test.

## FR-4, DD-5, P-3, AC-3, AC-4, AC-5 — the derivation

`pkg/data/equip.go` again: `HeroBodyFor(l BodyList, e Equipment) (HeroBody, bool)`.

- slot 1 empty -> `l[0]`, the bare-handed name, `true` (AC-4);
- slot 1 holding row `r` -> `l[r-1]`, `true` (AC-3);
- `r-1` outside the list, or `l[r-1]` empty -> `"", false` (AC-5).

The two live arms meet at row 1, and they should: row 1 of the shipped collection is the bare-handed
weapon and its entry is the bare-handed name, so an empty slot and a bare hand answer the same name
by two routes. That coincidence is the reason FR-4 can be total without a special case.

The refusal is DD-5 and it is **authored** — what the original does past its list's end is not
established. Write it as authored at the function, in the spec and in the evidence, exactly as
`0085` wrote its own authored term, and give the same shape of answer `0085`'s unmatched-name arm
gives: report the refusal, invent nothing.

P-3 is what this function's *absence* of arguments buys: it takes a list and a slot value, and there
is nowhere in it for a skill, a statistic or a shape word to enter. Keep it that way — a second
argument is how an invented correspondence would arrive.

## FR-6 — a weapon's row

`pkg/data/weapon.go`. `Weapon` gains `Row int32`, filled inside `ResolveWeapon` from the value
`findByName` already computes and currently discards. One field, one assignment, no new search.

**Do not relax `ResolveWeapon`'s ranged refusal.** `pkg/mapload`'s `firstWeapon` uses that refusal as
its "is this cell a weapon" predicate, and `0104` added `WeaponRange` as a second entry point rather
than loosening it. This story needs no third entry point: the party's weapon is already resolved
through `ResolveWeapon`, so the row arrives with it.

`Row` is a **loader value and reaches no entity, no byte form and no digest** — the same terms the
weapon's other numbers are already on. Say so at the field.

## FR-5, DD-2, DD-3, P-1 — the party's name is derived

`pkg/game/hero.go`. **`PartyBody` is deleted, not rewritten** (P-1, AC-10): a function returning a
constant is exactly the literal FR-5 forbids, and leaving it as a fallback would give the tree two
answers to one question. Its long AUTHORED comment goes with it, and one sentence of the reason it
existed moves to the derivation's own site — the list it said was undecoded is now read.

`MissionParty` takes the body list beside the weapon it already takes. It builds an `Equipment` with
slot 1 set from the weapon's `Row` (empty when the weapon is nil), calls `HeroBodyFor`, and passes
the answer into the existing `HeroBodyClass`. **A refused derivation carries the empty body name
forward**, which is already a real state in this tree — `PartyMember.Body`'s own doc calls it "a
member drawn through its class record like any placed unit" — so FR-4's refusal needs no new
handling here, and `MissionParty` gains no second answer to disagree with the law.

`PartyBodyDir` stays and stays derived; `frontend.go`'s `LoadHeroBody` call takes the derived name
in place of `PartyBody()`. `cmd/missionrun` is the second caller of `MissionParty` and takes the
list from the same `Defs` value it already takes `StartWeapon` from — so the list is loaded where
`StartWeapon` is loaded, and a caller cannot hold one without the other.

DD-2 is the site itself: the derivation runs once, where the party is assembled, because nothing in
this tree can change a slot after a mission opens. DD-3 needs no code — the dying substitution is
already in `HeroBodyName` and already unreached.

## FR-7, P-4 — the reach

`pkg/data/hero.go`: `Hero.Derive` gains the two lines `HumanDef.Combat` currently carries after
calling it — `c.Reach = 1`, and `c.Reach = w.Range` when the weapon is not nil — and
`HumanDef.Combat` **drops** them. That is a move, not a copy, and it is what makes P-4 provable by
inspection: the placed person's number comes out of the same two lines it came out of before,
because `HumanDef.Combat`'s first statement is `d.Hero().Derive(w)`.

`pkg/mapload/start.go`: the entity mint gains `Reach: reachOf(c.Reach)`, using the narrowing
`fromalm.go` already applies to a placed person's. Today the field is not written at all, so a
started party member's reach is the world constructor's repair of a zero (FR-7's second sentence) —
that repair stays where it is, for the hand-assembled worlds its own comment names, and stops being
what a party member's reach comes from.

The change is invisible on the default party and that is expected, not a reason to skip it: the
blade arm's start weapon has range 1. The bow arm is where AC-7 bites.

## FR-8 — nothing moves in the simulation

No field is added to `sim.Entity`, the form version literal stays at **23**, and the encoded layout
is untouched. Byte-form version **24 was allocated to this story and is deliberately unused**: the
reach byte has existed since version 23 and this story only writes a value into it. AC-9's digest
check is the standing pin test already in `pkg/sim`, re-run unchanged rather than re-pinned.

## Success criteria

**SC-1** `go build ./... && go vet ./... && gofmt -l . && go test -count=1 ./...` clean, plus
`scripts/check-no-game-assets.sh`, `check-doc-budget.sh` and `check-sdd-audit.sh`.

**SC-2** No unit test reads a game install. Every fixture is bytes or values built in test code;
the two-install measurements are AC-2 and AC-6, taken at the verification stage.

**SC-3** No test asserts a body name against a transcription of the shipped list held in this
repository (P-5). A test may assert the *shape* of the derivation over a fixture list.

**SC-4** The existing appearance law in `pkg/data/appearance.go` is not edited. This story feeds it;
if a clause of it has to change, the cut was wrong and that is worth stopping for.

**SC-5** `pkg/render` and `pkg/sim` gain no import and no field. The diff in each is empty.

**SC-6** After the change, `grep` finds no body-name literal on the party's path outside the shipped
list's own parse and the appearance law's own name constants.

## Task split

Four tasks, bottom up, each a separate commit.

**T1** is the slots, the shipped list's parse and the derivation — one new file in `pkg/data`, no
caller changed, every fixture built in test code.

**T2** is the rest of `pkg/data`: a weapon's row, and the reach moving out of `HumanDef.Combat` into
`Derive`. It is separate from T1 because it touches three existing files and nothing T1 wrote, so
the two cannot mask each other's regressions.

**T3** is `pkg/game`: the address, the read, the deletion of `PartyBody`, and `MissionParty`'s new
argument. It depends on both leaves.

**T4** is the two call sites T3 breaks — `frontend.go` and `cmd/missionrun` — plus the one line in
`pkg/mapload/start.go`. It is last because it is where the story becomes visible.

AC-2, AC-6, AC-7 and AC-8 are measurements the verification stage takes against installed roots; a
task neither asserts them nor claims them.
