# 0155 — the headless scenario language gains a mission stage

## Problem

`againrom --headless <scenario.json>` drives the production controller with no
window. Its version-1 vocabulary is a front-end vocabulary: keys, controls,
menus, saves, party assertions. It enters a game only by loading a save, so a
scenario needs a save file on disk before it can observe anything, and once
inside it can order nothing and assert nothing about the world.

`cmd/missionrun` can do the other half. It starts a campaign mission by number,
orders units, waits for arrivals, and reports the outcome and the script census.
Its vocabulary is command-line flags, and its phase order is fixed: all
waypoints, then all attacks, then all takes. A run is not a written artifact and
a sequence of play is not expressible in it.

What is missing is a written file that can start a mission and play it.

## Scope

One contract: **the scenario file is the instrument**. A scenario declares which
driver it is written against, and the mission driver gains the vocabulary of
play.

Out of scope, stated so it is a decision rather than an omission. The owner asked
for four things and this story delivers the instrument for all four while driving
only two missions; what is deferred is named here with what it needs:

- **Seeding a party to stated values** — class, sex, name, statistics, skill
  levels, equipment — is not in this story. It has to go through the production
  generator that `0153` built, whose own vocabulary is a slice in its own right,
  and a seed written through any other door would produce state no ordinary game
  can reach and a digest that means something else. The scenario carries `mage`
  and `difficulty` only.
- **Chaining mission to town to next mission** in one scenario is not in this
  story. The mission stage builds one world; chaining needs the front-end stage's
  continuity path, which is the other stage.
- **A 10 to 150 campaign sweep** is not in this story. The language can express
  the per-mission half of it now; the sweep is a set of files and a gate.
- **Keying an expected digest to an install fingerprint** is not in this story. A
  digest is recorded; asserting one needs a stored baseline keyed to the
  measured install, which is its own artifact.
- **Authoring a script inside a synthetic world** is not in this story. A
  synthetic world carries units and relations, not triggers.

Also out of scope:

- `cmd/missionrun` is not changed, not deleted and not merged into this route.
  It is the instrument `pipeline/check-milestone.sh` drives, its argv is frozen
  by that gate, and its per-tick script trace has no equivalent here.
- Per-tick script tracing is not added to the scenario language. A mission
  scenario reports the script-gap count and latch state, not the arms that fired.
- A scenario cannot mix front-end and mission steps in one file.
- Sacks, equipment transfers and skill re-derivation stay in `cmd/missionrun`.

## Functional requirements

**FR-1 — A scenario declares a stage.** `stage` is `frontend` or `mission`.
Absent, it is `frontend`. A command belongs to exactly one stage, except
`wait_ticks` which both run, and a step naming the other stage's command is
refused at validation with the stage it belongs to and the stage the file is on.

**FR-2 — A mission-stage scenario starts a mission with no save file.** The
top-level `mission` number is required and positive; `difficulty` is `easy`,
`normal` or `hard`, defaulting to `normal`; `mage` starts the hero as a caster.
The mission is built through the same start path the front end and
`cmd/missionrun` use, with a party built by the same `MissionPartyAs`. The save
directory fields are refused on this stage.

**FR-3 — A scenario orders units.** An `order` step names a unit, a verb, and the
verb's own parameters: `move`, `patrol` and `march` take `x` and `y`; `attack`
takes `target`, a second unit reference; `guard` and `stand` take neither. The
step issues the production simulation command and advances one tick.

**FR-4 — A scenario waits for a condition, not a tick count.** A `wait_until`
step names exactly one of three conditions — a unit within a Chebyshev radius of
a cell, a unit fallen, or the mission decided — and a tick ceiling defaulting to
40000. It steps until the condition holds. A ceiling spent with the condition
unmet fails the run and names the condition and the ceiling. A mission decided
while a unit condition is outstanding fails the run and names the verdict and the
tick, because no further work is accepted after a verdict.

**FR-5 — A scenario asserts about a unit and about the world.** `assert_unit`
checks `alive`, `x`, `y`, `within`, `hp_at_least`, `hp_at_most`, `owner` and
`group` for one reference. `assert_world` checks `outcome`, `tick_at_least`,
`tick_at_most`, `alive_at_least`, `fallen_at_most`, `latched`, `not_latched`, and
`unsupported_at_most` — the count of arms of this mission's compiled script this
build cannot run, which is the number `pipeline/check-milestone.sh` tracks. Every
field is optional and an assertion stating nothing is refused.

**FR-6 — A run's output is evidence.** Every mission step emits one JSON event
carrying tick, outcome, living count, fallen count and the script-gap count. A
`report` step additionally carries every entity's reference, id, position,
health, owner slot and group. Standard error carries one compact row per step.
The two writers are the contract the front-end stage already publishes.

**FR-7 — Versioning is explicit and an older file does not change meaning.** The
current version is 2 and this build reads 1 and 2. A version-1 file is refused
the moment it names a stage, a mission number, a difficulty, the mage flag, or a
command introduced in version 2. A version above the current one is refused
naming the range this build reads.

**FR-8 — The mission stage runs with no game install present.** The play half is
defined over `*sim.World` and two reference tables, so a `PlayWorld` is
constructible in test code. A scenario file read by the production reader,
validated by the production validator and executed by the production runner opens
no archive, no map and no save (golden rule 2).

**FR-9 — Unit references are `cmd/missionrun`'s grammar.** `uNN` is the
identifier the map's script uses, `pN` is the party's Nth member in start order.
`eNN`, a raw entity id, is the one addition, for a world with neither table. An
unresolvable reference fails the step naming what it could not find.

**FR-10 — The asset root comes from the flag or the environment.** A mission-stage
run resolves the root through the same precedence every other mode uses, so a
scenario run with the root in `AGAINROM_ASSETS` reads the same install as one run
with `-assets` (golden rule 3).

**FR-11 — A scenario declares where its bytes come from.** `assets` is `install`
or `synthetic`, defaulting to `install`. A synthetic mission scenario states its
own world — size, units and hostile owner slots — under `world`, and runs with no
asset root configured at all. It is composed from the scenario's own declaration
and never sampled out of an archive (golden rule 1). A synthetic scenario naming
a mission number, and an install scenario naming a world, are each refused. The
front-end stage runs on `install` only.

What each provider proves bounds what any scenario may claim. Synthetic proves
**mechanism**: the rule fired, the state changed, the order is right. It cannot
prove this build reads the original's bytes correctly, because the bytes were
authored here from the same belief the decode holds. Install proves **fidelity of
ingestion**: the real archive opens, the real map loads, the counts come out. It
cannot run where no install is present, so it never runs under `go test`. Neither
states what the original game does; only a research claim does.

**FR-12 — A run reports the script arms it reached, not only the arms the file
contains.** `pipeline/check-milestone.sh` compiles each map and counts the
operations this build cannot run, which includes operations on branches nobody
walks. A mission-stage run additionally counts what a played mission arrived at:
an unsupported instant in a trigger that fired, and an unsupported check the pass
evaluated. Both counts appear on every event; a `report` step carries the
per-opcode breakdown. A check whose unit, group or player reference did not
resolve is counted beside them under `unresolved`, because that silence leaves a
live trigger comparing a stale register. `assert_world` can bound the reached
count with `reached_unsupported_at_most`.

**FR-13 — A run records a digest.** Every observation carries `sim.World.Hash()`.
`pkg/sim` steps on a fixed integer tick with no clock, no `math/rand` and no
floats, so the same scenario over the same assets yields the same digest. It is
recorded and not asserted: a digest is only meaningful against byte-identical
assets, and keying an expected value to an install fingerprint is separate work.

**FR-14 — An outcome is recorded, not asserted, unless the scenario's own orders
make it deterministic.** An unattended drive that loses is not a defect: mission
10 is an escort and an unattended drive loses by design (owner, 2026-08-11). A
scenario that drives a party to an objective may assert a verdict; a scenario
that steps a mission without playing it may not, and the shipped install
scenarios record the verdict instead.

## Acceptance criteria

**AC-1** A mission scenario written as a file, read through
`ReadHeadlessScenario`, runs to completion over a world built in test code, with
no install present, and its event stream shows the ordered unit moved.

**AC-2** `p0`, `u57` and `e1` resolve to the entities the tables name; a
reference with no prefix, an unknown prefix, an out-of-range party slot, an
unplaced script id and a non-numeric suffix are each refused.

**AC-3** A `wait_until` whose condition never holds fails with the condition and
the ceiling in the message.

**AC-4** `unsupported_at_most` fails over a world whose compiled script carries
one arm this build does not run.

**AC-5** A version-1 file naming a version-2 command or a stage is refused; a
mission command in a front-end file and a menu command in a mission file are each
refused naming both stages; the front-end runner refuses a mission scenario and
the mission runner refuses a front-end one.

**AC-6** A scenario orders an attack, waits for the victim to fall, and asserts
the victim dead and the survivor standing.

**AC-7** `scenarios/0155-mission10-escort.json` runs against both preserved
installs and records the same verdict, at the same tick, with the same fallen
count as the milestone drive over the same mission.

**AC-8** Over a world whose script holds one unrunnable instant behind a trigger
that fires, the compiled count is one before any tick has run and stays one,
while the reached count is zero before the first tick and rises afterwards, and
the per-opcode breakdown accounts for every arrival.

**AC-9** A synthetic scenario builds its world from its own declaration and runs
with no asset root configured. A world with no units, a unit outside its bounds,
a repeated reference, a party slot with a hole, and a unit with no health are
each refused.

**AC-10** The suite runs every shipped synthetic scenario end to end and reports
how many install-tier scenarios it skipped and why.

## Properties

**P-1** A mission-stage run reads no wall clock and no random seed of its own.
Every wait is a fixed tick count and every observation is taken between steps.

**P-2** A mission-stage run writes no file. Its whole result is the two streams.

**P-3** The scenario language is additive across versions: no version-1 file's
meaning is changed by this story, and no version-1 file gains a step it did not
have.
