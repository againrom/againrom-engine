# 0163 — the campaign drive

## Terms

**Scenario** — the versioned JSON file `againrom --headless` reads
(`scenarios/README.md`). **Stage** — which driver a scenario's steps reach:
`frontend` or `mission`. **Generation screen** — the production two-stage
character generator: a pre-create page holding four combined class-and-sex
pictures and a name field, and a detailed page holding one skill row, four
point-buy statistics, and Back / Reset / Play. **Crossing** — the transition a
decided mission makes into the town or into its declared successor. **Sweep** —
one command that drives every campaign mission in order and prints one row each.

## Problem

The headless instrument can drive one mission and it can drive the menus, but it
cannot drive a campaign. Three things are missing.

A scenario cannot create a character. Every front-end scenario starts from a save
or from the front end's own default hero, so nothing headless reaches the
production generator at all, and the generator is the first thing a player uses.

A scenario cannot follow a mission out of itself. The crossing into the town is
built and the town screen is built, but a scenario has no way to wait for a
mission to decide: `wait_ticks` needs a tick count the author must know in
advance, and there is no condition form on the front-end stage.

There is no committed command that drives the whole campaign. The census in
`pipeline/check-milestone.sh` loads each of the 28 campaign maps for one tick; it
does not drive one. No instrument in this repository answers whether a campaign
map runs for two thousand ticks without failing.

## Functional requirements

**FR-1 — a scenario creates its character through the production generator.** A
front-end scenario may carry a `create_character` step. It runs only on the
generation screen. It drives that screen through the same `ui.App` input dispatch
a windowed session drives it through: focus movement, Enter, and typed
characters. It calls no method of `ui.Chargen` directly and installs no
scenario-only path into the generator.

**FR-1a — a scenario reaches the generator the way a player reaches it.** The
generation screen is armed by choosing a mission row from the map list that NEW
GAME opens, which is the campaign's own starting door. A scenario must therefore
be able to press NEW GAME on the main menu and then choose a row, and the press
must go through the production hit test rather than a coordinate the scenario
supplies. Until this story no front-end scenario could start a new game at all;
every one began by loading a save.

**FR-2 — the step names the player's choices, not keystrokes.** The step's
`character` block carries `name`, `sex`, `class`, `skill` and `stats`. `sex`,
`class` and `skill` are the option labels the screen itself shows. `stats` maps a
statistic's own name to the value it is to be bought up or down to. Every field
is optional; a field not named keeps what the generator opened with.

**FR-3 — the created character is what the interactive generator would produce.**
After the step the front end holds exactly the party the generation screen's own
Play control produces for that spread. The step fails, and changes nothing, when
the requested spread is one the generator would refuse — an unaffordable
statistic, a value outside its floor or ceiling, an option label no row offers, a
name the screen reserves.

**FR-4 — the generator is observable.** Every front-end event carries a `chargen`
object while the generation screen is showing: the stage, the focus, the name,
the remaining budget, whether the spread is legal, and one row per control the
screen offers with its label and its current value. It is absent on every other
screen.

**FR-5 — a front-end scenario can wait for a condition.** `wait_until` runs on
the front-end stage. It steps the application through the production dispatch
until the condition holds or the tick ceiling is reached, and a ceiling reached
without the condition fails the run. Two forms: `{"screen": "<name>"}` holds when
that screen is showing, and `{"notice": true}` holds when a notice is open over
the map.

**FR-6 — a scenario observes the crossing.** A scenario that drives a mission to
a verdict, dismisses its notice, and arrives in the town can assert the roster,
the purse and the document count on both sides of the crossing, and can then
proceed to the next mission from the town's own gates.

**FR-7 — one command sweeps the campaign.** `scripts/campaign-sweep.sh` drives
every campaign mission in ascending order, headless, against a lawful root named
by `-assets` or `AGAINROM_ASSETS`. It prints one row per mission: the mission
number, the tick it stopped at, the outcome, the compiled script-gap count, the
reached script-gap count, the live and fallen entity counts, and the simulation
digest. It prints a totals row.

**FR-8 — the sweep is deterministic and reads no clock.** Two runs over the same
root print byte-identical tables. The sweep passes no seed, no time and no
randomness to anything it drives.

**FR-9 — the sweep never holds.** A mission that does not decide inside the tick
budget is recorded `undecided` and the sweep continues. A mission that fails to
load or whose drive returns an error is recorded as that failure and the sweep
continues. The sweep's exit code is non-zero only when a mission failed to load
or to drive; an outcome word never decides it.

**FR-10 — older scenarios are unchanged.** A version-1 or version-2 file keeps
exactly the vocabulary it shipped with. `create_character` and the front-end
`wait_until` belong to version 3 and are refused in an earlier file. Every
scenario in `scenarios/` written before this story validates and runs unchanged.

## Acceptance criteria

**AC-1** A version-3 front-end scenario reaches the generation screen, runs one
`create_character` step naming a sex, a class, a skill, a name and a statistic
spread, and enters a mission whose first party member carries that name, that
class, that skill and that spread.

**AC-2** Driving the generation screen by hand — the same up/down/enter/typed
dispatch, written out step by step — reaches a party byte-identical to the one
`create_character` reaches for the same choices.

**AC-3** `create_character` naming a statistic value the budget cannot pay for
fails, names the statistic, and leaves the screen showing with the front end's
party unchanged.

**AC-4** `create_character` naming an option label no row offers fails and lists
the labels that row does offer.

**AC-5** A front-end event carries a `chargen` object on the generation screen
and carries none on the menu, the picker, the map or the town.

**AC-6** `wait_until` with `{"notice": true}` over a decided mission stops on the
tick the notice opens; the same form with a ceiling below that tick fails and
names the ceiling.

**AC-7** A scenario captures the roster, the purse and the document count on the
map, crosses into the town, and asserts the same three there.

**AC-8** From the town, a scenario reaches the gates and opens a further mission.

**AC-9** `bash scripts/campaign-sweep.sh` prints one row for each of the 28
campaign missions and a totals row; every row carries an outcome word and a
digest.

**AC-10** Two consecutive sweeps over the same root produce identical output.

**AC-11** A sweep whose mission list names a map the root does not hold records
that mission as a load failure, keeps going, and exits non-zero.

**AC-12** `scenarios/0155-mission10-escort.json`, `0155-mission20-sweep.json`,
`0155-synthetic-melee.json`, `0152-save666.json`, `0154-synthetic-spells.json`,
`0156-mission30-cure.json` and `0159-mission40-join.json` validate unchanged, and
a version-2 file naming `create_character` is refused.

## Properties

**P-1** No new nondeterminism. `pkg/sim` is untouched. Nothing added reads a
clock, a random source or the environment beyond the asset root.

**P-2** The serialized byte-form version stays 46. This story changes no save
format and no hashed simulation state.

**P-3** `cmd/missionrun`'s argv is unchanged, so
`pipeline/check-milestone.sh` keeps measuring what it measured.

**P-4** The generator is reached through one path. `create_character` adds no
second door into `ui.Chargen`; the screen's own model methods stay unexported to
`pkg/game`.

**P-5** The import graph is unchanged in direction. `pkg/ui` gains no import;
`pkg/game` gains none.

## Scope cuts

**SC-1** The sweep does not carry a roster between missions. Each mission is
driven in its own process with the front end's own default party. Campaign
continuity across a win is witnessed by the chaining scenario, on the one
crossing that is drivable, and not by the sweep.

**SC-2** The sweep does not attempt to win. It issues no orders. Its value is the
census and a crash-free drive.

**SC-3** No scenario in this story drives the legacy one-stage generation model.
`create_character` refuses it and says so.

**SC-4** Mouse input into the generation screen is not added to the headless
surface. Every control the step reaches is reachable by focus and Enter, which is
what a keyboard player uses.
