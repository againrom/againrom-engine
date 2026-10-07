# 0163 — plan

## Shape

Three tiers move. `pkg/ui` gains a read surface over the generation screen and
two input entry points. `pkg/game` gains the version-3 vocabulary and the step
that drives the generator. `scripts/` gains the sweep. Nothing in `pkg/sim`
changes (P-1, P-2).

## Design decisions

**DD-1 — the generator is driven by focus and Enter, never by a model call.**
`ui.App.stepPreCreate` and `ui.App.stepChargenDetailed` already map a focus index
to a control and activate it on Enter. `create_character` therefore navigates
with `HeadlessKey("up"/"down")` and activates with `HeadlessKey("enter")`, which
is `a.step` — the same statement a windowed frame runs (FR-1, P-4). The
alternative, calling `Chargen.SelectPreChoice`/`AdjustStat` from `pkg/game`,
would be a second door into the generator and would pass while the screen itself
refused the same spread.

**DD-2 — layout knowledge stays in `pkg/ui`.** The focus order of the two pages
is `preFocusControl` and `detailedFocus`, both unexported. Rather than copy that
order into `pkg/game`, the new read surface reports, for every control, **the
focus index that reaches it**. `pkg/game` then only ever moves the focus to a
number it was told. A page whose focus order changes stays correct with no change
in `pkg/game` (FR-4).

**DD-3 — the four pictures report the sex and class they stand for.**
`Chargen.Forward` already maps a pre-choice index to a sex index and a class
index. That mapping is lifted into one unexported helper used by `Forward` and by
the read surface, so the picture labels a scenario matches against and the
identity `Forward` actually commits are the same statement read twice, not two
rules that can drift (FR-2, FR-3).

**DD-4 — the labels are the screen's own.** A picture's sex and class labels are
`setup.Choices[0].Options[...]` and `setup.Choices[1].Options[...]`; a skill's
label is `choiceOptions(2)`; a statistic's label is `setup.Stats[i].Name`. So a
scenario naming `"Mage"` matches because the screen says `Mage`, and a scenario
naming a label no row offers is refused with the offered labels listed (FR-2,
AC-4). `pkg/ui` authors none of these strings and keeps authoring none.

**DD-5 — the name is typed.** `App.HeadlessType` dispatches `appInput{Typed:...}`
and `appInput{Backspace:true}` through `a.step`. The generator's own
`EditName` clears the setup default on the first accepted character, so a step
naming a name types it and gets it; a step naming none keeps `Danath`. The
encoder is the install's (`setup.EncodeName`), so a character the install cannot
store is dropped exactly as it is for a player (FR-2).

**DD-6 — `create_character` is one step and not eight.** The alternative was a
step per control. Rejected: a scenario would then encode the page order, which is
DD-2's whole objection, and every file would restate the same eight steps. The
step is the player's intent; the navigation is the runner's.

**DD-7 — order inside the step is fixed and stated.** Pre-create: type the name,
select the picture, Forward. Detailed: select the skill, then set the statistics
in the order the screen lists them, then Play. The skill precedes the statistics
because the skill row is class-dependent and `Forward` reseeds it; setting it
first means nothing after it moves it. Statistics are set by pressing the
statistic's own minus or plus control until the value is reached, so an
unaffordable step is refused by `Chargen.AdjustStat` itself rather than by a
budget check written here (FR-3, AC-3).

**DD-8 — a refusal is read off the message line, and the step stops.** Every
production refusal — an illegal spread, an empty or reserved name, a wiring-tier
decline — lands in `flow.msg` and leaves the screen showing. The step compares
the state it asked for against the state it got after each control, and reports
the message line when they differ. Nothing is rolled back, because nothing that
was refused was applied (FR-3).

**DD-9 — the command table becomes per-stage.** `headlessCommandSpec.stage` and
`.version` are replaced by `stages map[string]int`: which stages run the command,
and the scenario version each stage got it in. `wait_until` is
`{mission: 2, frontend: 3}` and every other command keeps exactly the pair 0155
gave it. This is what lets one command reach a second stage in a later version
without the first stage's files acquiring vocabulary their author never wrote
(FR-5, FR-10). The alternative, a separate front-end command name, would be a
synonym for the same idea.

**DD-10 — `HeadlessScenarioVersion` becomes 3.** A version-2 file naming
`create_character` or a front-end `wait_until` is refused by the same table
lookup that already refuses a version-1 file naming a mission (FR-10, AC-12).

**DD-11 — the front-end `until` forms share `HeadlessUntil`.** The struct gains
`screen` and `notice`; validation takes the stage and admits only that stage's
forms, so a mission file naming `screen` is refused and a front-end file naming
`dead` is refused. One struct, two admitted vocabularies (FR-5).

**DD-12 — front-end `wait_until` steps and does not activate.** It calls
`App.HeadlessStep` up to `ticks` times. It never presses a key, so a notice it is
waiting for stays open and the scenario's next step is the one that dismisses it.
A ceiling reached without the condition fails and names the ceiling and the
condition (AC-6).

**DD-12a — the crossing needs no new step, but it did need the main menu.** FR-6
is met entirely by commands that already existed — `activate "notice"` crosses,
`assert_state` and `assert_member` read the roster, the purse and the documents
on both sides, and the town's own rows reach the gates — once `wait_until` can
say *when* to press. The one thing missing was the way in: `App.stepMenu` reads
no key but `L`, so the main menu was pointer-only and NEW GAME was unreachable to
a scenario. `activate` therefore gains a menu arm that presses one brooch button,
finding its point by asking the production hit test `App.buttonAt` rather than by
computing one from the placement table — the same oracle-driven shape
`HeadlessSelectEntity` already uses for an entity (FR-1a). The GENERATION screen
needs no such arm and gets none (SC-4): every one of its controls has a focus
path, so a keyboard drive reaches all of them.

**DD-13 — the sweep drives this story's own instrument.** It generates one
mission-stage scenario per mission into a temporary directory and runs
`cmd/againrom --headless` on each. The alternative, driving `cmd/missionrun`,
would measure a tool this story does not touch and would print no digest. Driving
the scenario runner means the sweep is a regression test of the scenario language
over all 28 campaign maps rather than over the two that have scenario files
(FR-7).

ONE PROCESS PER MISSION, so no roster crosses inside the sweep (SC-1). Most
campaign missions are not winnable unattended, so there would be nothing to
carry; the crossing is witnessed by the chaining scenario, on the one boundary
that can be driven.

**DD-14 — the sweep's tick budget is a flag with a stated default.**
`--ticks N`, default 2000. A budget is needed because an unattended campaign
mission does not decide: mission 10 was driven at 2000, 5000 and 20000 ticks and
returned `undecided` with nothing fallen at each. Raising the default buys no
verdict, so it is set where the whole sweep costs 20 seconds and is a flag for
anyone who wants more (FR-9).

**THE SWEEP ISSUES NO ORDERS** (SC-2). It is a census and a crash-free drive.
Making it play would need per-map knowledge — where to walk, what to attack —
that this story does not have and that would have to be written 28 times.

**DD-15 — the mission list is the same 28 `check-milestone.sh` walks**, in
ascending order, with `--missions` to override. Sharing the list rather than
importing it keeps the sweep runnable from a worktree, which has no
`pipeline/` beside it.

**DD-16 — the sweep's exit code separates the instrument from the game.** A load
failure or a drive error is the instrument failing and exits non-zero; every
outcome word, including `lost` and `undecided`, exits zero (FR-9, AC-11).

**DD-17 — determinism is asserted by the sweep's own `--check` mode**, which runs
the whole table twice and diffs it. It is not the default, because a determinism
check doubles the wall clock of a command whose purpose is a census (FR-8,
AC-10).

## Work

1. **`pkg/ui` read surface and input.** `HeadlessChargen` and
   `HeadlessChargenControl` structs; `App.HeadlessChargenState() (HeadlessChargen,
   bool)`; `App.HeadlessType(text string, backspace bool) error`;
   `App.HeadlessNoticeOpen() bool`. `preChoiceParts` lifted out of `Forward`
   (DD-1, DD-2, DD-3, DD-4, DD-5). Unit tests over a synthetic setup.

2. **`pkg/game` vocabulary.** `stages` table (DD-9); `HeadlessScenarioVersion` 3
   (DD-10); `HeadlessUntil.Screen`/`.Notice` with stage-aware validation (DD-11);
   `HeadlessCharacter` step block; `create_character` runner (DD-6, DD-7, DD-8);
   front-end `wait_until` runner (DD-12); `chargen` on `HeadlessState` (FR-4).
   Tests: the parser's refusals, and the generator drive over a synthetic front
   end.

3. **The witness scenarios**, which are where FR-6 is met (DD-12a).
   `scenarios/0163-chargen-mission10.json` — reach the generator the player's own
   way and enter mission 10 with what it produced (AC-1).
   `scenarios/0163-mission-to-town.json` — drive a mission to its verdict, cross
   into the town, assert roster, purse and documents on both sides of the
   crossing, take work at the tavern, and walk out of the gates into the next
   mission (FR-6, AC-7, AC-8). Both are `install` tier and are run by hand, not
   by `go test`.

4. **`scripts/campaign-sweep.sh`** (DD-13 … DD-17), plus the `scenarios/README.md`
   entries for the new vocabulary and the sweep.

5. **The hand-driven equivalence test** (AC-2): one test writes the same choices
   twice, once as `create_character` and once as a list of `key`/`type` steps,
   and compares the two parties. This is what keeps DD-1 honest — if the step
   ever stopped going through `a.step`, the two would diverge.

6. **`builds/0163-campaign-drive/`** in the main checkout, with a README whose
   commands were run before being written down.

## Risks

- The front end needs `ChargenAssets` to resolve for the two-stage generator. A
  test front end that resolves none holds the legacy model, and the step refuses
  it (SC-3). The equivalence test therefore builds a setup with a `PreCreate`
  block rather than relying on an install.
- A campaign map whose script does not decode raises `RaiseErr`. The sweep
  records the mission and continues; that is a load that succeeded with a
  degraded script, not a failure (FR-9).
