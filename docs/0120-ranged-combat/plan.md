# Plan — 0120-ranged-combat

## Shape

Four tasks, each one commit, in this order: the resolver's ranged arm, the person-side fold's
two arms, the creature-side fold, the hero's trained skill, and the drawn shot. Nothing in
`pkg/sim` changes at all, which is the property that keeps the whole story off the byte form.

| task | package | what |
|---|---|---|
| T1 | `pkg/data` | FR-1, FR-2 — the resolver stops refusing, and the fold grows its second arm |
| T2 | `pkg/mapload` | FR-3, FR-4 — a creature's class equipment folds whole |
| T3 | `pkg/game` | FR-5 — the trained skill becomes a setting with today's default |
| T4 | `pkg/game`, `pkg/ui` | FR-6, FR-7 — a mark in flight |

## Design decisions

**DD-1 — the attack type stays on the weapon and the branch lives in the fold, not in the
resolver.** `ResolveWeapon` already carries `AttackType`; the honest reading is that a Weapons
row states what it states whatever kind it is, and the *equip* is where the two kinds part.
So T1 deletes a refusal and adds no field, and the branch appears exactly once, in
`Hero.Recompute`'s equipment step, expressed over a predicate on the weapon.

**DD-2 — the predicate is a method on `Weapon`, not a comparison at each site.** `Ranged()`
against the existing threshold constant. Three call sites will ask (the person fold, the
creature fold, and a test), and three spellings of `>= 0xa` is how the threshold drifts.

**DD-3 — the ranged arm is expressed as *what it does*, not as *what it skips*.** Write the
cadence and reach assignments unconditionally above the branch and put only the four additions
inside it. A reader then sees FR-2's two bullets in the order the spec states them, and adding
a fifth contribution later cannot accidentally land on the wrong arm.

**DD-4 — the creature fold reuses the person fold's own resolution, and keeps the old range
read as a fallback.** `firstWeapon` and `unitReach` are the same search over the same strings
with different payloads. T2 makes one search that returns the resolved weapon, folds it, and —
only when nothing resolved — falls back to the existing range-only read. That fallback is what
makes FR-4 provable rather than hoped for: a row this story's resolver still cannot take gets
exactly the reach it gets today.

**DD-5 — the creature fold is applied where the reach assignment is applied today**, after the
difficulty adjustment, in `definitionFor`. Not before: the adjustment scales the row's own
columns and the original's equip runs after the row is streamed, so folding first would scale
the weapon too.

**DD-6 — the fold itself lives in `pkg/data`, beside the one the person side already has.**
Both sides fold the same weapon onto the same `Combat` shape under the same rule, and two
spellings of one rule in two packages is the failure mode this repo has a section about. T2's
work in `pkg/mapload` is then the *search*, and the arithmetic is one call.

**DD-7 — the trained skill becomes package state in `pkg/game` with a setter, not a parameter
threaded through six signatures.** It is a front-end setting, set once at start-up, read by
character generation and by the sheet. Threading it would touch `LoadDefinitions`, the
`Definitions` struct, `PartyHero`, `MissionParty` and both mission entry points, and another
lane is inside two of those files. One variable, one setter that refuses a slot the generator
trains no weapon for, and the existing name kept as the default. The simulation is not
involved, so no determinism rule is in play.

**DD-8 — the shot crosses the readout seam as a position and nothing else.** The drawing tier
receives, per entity, an optional point in the same cell coordinates it already receives the
entity's own cell in. Not a target id, not a reach, not a phase — those are the rule, and the
rule stays on the simulation side of the seam (FR-7). This is the same discipline the seam's
speed fields already state about the movement law.

**DD-9 — the interpolation is integer and lives where the swing clock already is.** The world
readout already computes, per entity, whether it is swinging and how far through the track it
is; that is the only place both the attacker's cell and the target's cell are in hand at once.
The fraction is the track index over the track length, applied to the cell delta with integer
arithmetic in a sub-cell unit the seam already uses for movement.

**DD-10 — the mark's appearance is a named colour beside the existing placement markers**, and
its doc block says the same thing they say: it is this project's diagnostic design and asserts
nothing about the original. D-4 is the reason it can be.

## What is deliberately not done

**No change in `pkg/sim`.** The reach arithmetic, the approach's stop arm, both target scorers
and `groupScorerReach` are all correct as they stand and were built by `0104`. Reversing the
literal ceiling was considered and rejected: a member of reach above 1 already passes that
refusal, because the distance rewrite immediately above it has turned its distance term into 1.
The reason `0104` gave — that the ceiling is the *law's* reach and widening it is a
group-behaviour story with its own measurement — is unchanged by anything found here.

**Byte-form version 30 is not used.** No simulated entity gains a field, so the form does not
move and no digest is repinned. The version stays available.

**No `builds/` backfill** and no owner-review artifact.

## Risks

**R-1 — another lane is in the same files.** `pkg/sim/engage.go` is held by a parallel lane;
this story does not touch it, which removes the conflict the brief expected. Character
generation is held by another; T3 must therefore be one variable and one setter in
`pkg/game/hero.go` and must not restructure the front end.

**R-2 — the creature fold changes 26 shipped classes' numbers.** It is the point of FR-3, but
it is also the largest behavioural change in the story and it is invisible in a unit test that
builds its own table. Mitigation: FR-4's reach invariance is the property that is actually
pinned, and it is pinned over a synthetic table shaped like the shipped one.

**R-3 — the fold direction could be an assignment rather than an addition.** If it were, 26
classes would land with their row's numbers replaced instead of raised, and it would look
plausible. It is settled in `provenance.md`: the row is streamed first and the equip runs after
it, through a routine read as an addition at every site. The tests assert the *addition*
explicitly — a fixture whose row and whose weapon both state a damage base, with the expected
value being the sum.

**R-4 — T4 touches three packages and is the one task with no decoded law behind it.** It is
the declared cut. If it does not land inside its own commit cleanly it is dropped, and the
story still satisfies the owner's two named facts.

## Success criteria

**SC-1** — AC-1 through AC-5 hold, each witnessed by a test that goes red when the line it
witnesses is reverted.

**SC-2** — `go build ./...`, `go vet ./...`, `gofmt` and `go test -count=1 ./...` are green on
the committed tree, and the three repo scripts pass with no new FAIL.

**SC-3** — the simulation package's version literal and every pinned digest are byte-identical
to master, and `git diff` shows no file under `pkg/sim` changed.

**SC-4** — the deletion set between the base commit and the branch head is empty.
