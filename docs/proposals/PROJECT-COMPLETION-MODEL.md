# Project Completion Model — proposal

Status: **PROPOSAL, NON-NORMATIVE**. This is a candidate way to answer "how close is againrom to done?" without using story count as a proxy. It does not change any current milestone or gate.

## Why story count is the wrong denominator

A story is a unit of change, not a unit of game completeness. Story size changes with process decisions; defects and amendments create new stories; one story can close a whole subsystem while another changes one pixel rule. Therefore `story N / expected final story` is not a stable progress measure.

Completion should be measured against **player-reachable capabilities and known fidelity debt**.

## Four independent completion axes

Do not collapse these into one percentage unless a release note explicitly defines the weighting.

### 1. Campaign reachability

Question: can a player naturally traverse the shipped campaign?

A campaign node is `PASS` when it is reachable through production game flow, can reach its intended terminal outcome, and does not require a developer-only bypass.

Track separately:

- missions reachable;
- missions completable;
- required town/world-map transitions working;
- campaign terminal state reachable.

A map merely loading is not campaign reachability.

### 2. System completeness

Question: are the game's player-facing systems present end to end?

Use the project's domain model rather than story directories. For each domain, classify:

- `ABSENT` — player path does not exist;
- `PARTIAL` — useful path exists but a known required behaviour is missing;
- `FUNCTIONAL` — required player path works end to end, fidelity gaps may remain;
- `FIDELITY` — required behaviour is reconciled with current research except accepted deviations/unknowns.

A domain can be FUNCTIONAL while research is incomplete. That distinction is useful and should remain visible.

### 3. Fidelity reconciliation

Question: what is known about disagreement with ROM1?

Report the divergence ledger by meaning, not just row count:

- open known fidelity debt;
- accepted deliberate deviations;
- authored behaviour where research is silent;
- closed divergences;
- research claims that are provisional or retracted and still affect live expectations.

An UNKNOWN is not automatically a defect. A FIDELITY-DEBT row is.

### 4. Journey integrity

Question: do the major systems continue to compose?

Use the curated integration-fidelity journeys if that proposal is adopted. Report `passed / applicable` journeys and name failures. Do not turn this into a percentage of scenario files; the portfolio is intentionally curated.

## Release levels

These levels are descriptive. They are intended to make disagreements precise.

### R0 — engine substrate

Formats and engine layers can load meaningful original data and exercise core simulation/rendering paths. Not claimed playable.

### R1 — playable slices

At least one production player journey crosses menu/gameplay and reaches a real mission outcome. Major systems may be absent.

### R2 — campaign playable

Every required campaign mission is naturally reachable and completable without developer-only bypasses. Required transitions between mission, town, world map and subsequent missions work. Save/load is usable for campaign play.

R2 permits known fidelity debt. It means "the game can be played through," not "the remake matches ROM1."

### R3 — feature complete

All required player-facing systems have reached at least FUNCTIONAL on the system-completeness axis. There is no known missing feature required to experience the shipped campaign as designed by againrom's current owner intent.

### R4 — fidelity candidate

All known FIDELITY-DEBT rows that affect the target fidelity scope are closed or explicitly accepted by the owner. Current promoted research has been reconciled against implementation expectations. Curated journeys pass on both supported roots where applicable.

### R5 — fidelity release

A frozen research pin and implementation release have been audited together. Remaining differences are explicit ACCEPTED deviations or clearly identified research unknowns; no known unrecorded player-visible mismatch is being carried as "probably fine."

R5 does not mean mathematical proof of 100% identity with ROM1. It means the project's evidence model has no known unaccounted mismatch within its declared scope.

## Candidate definition of "campaign playable"

The following is intentionally stronger than "all maps launch":

1. A fresh session can start through the production menu and character-generation flow.
2. Every mandatory campaign mission can be entered through the campaign's production routing.
3. Each mandatory mission can reach its success/failure verdict through ordinary simulation and player commands.
4. Mission verdicts cross correctly into notices, town/world-map state and subsequent mission selection.
5. Party membership, required inventory/equipment, purse/documents, skills/XP and other campaign-carried state survive the transitions that consume them.
6. Save/load can interrupt and resume a representative campaign run without requiring a developer repair step.
7. No mandatory transition requires a debug command, direct map selector, hand-edited save or special-case harness-only state.

This definition says nothing about exact ROM1 fidelity; that is a separate axis.

## Candidate definition of "fidelity candidate"

A build is a fidelity candidate when:

- campaign playable is satisfied;
- all player-facing domains are FUNCTIONAL or FIDELITY;
- no OPEN FIDELITY-DEBT row remains inside the declared target scope unless the owner explicitly waives it;
- every current promoted claim that materially constrains player-visible behaviour has either an implementation mapping, an accepted divergence, or a documented reason it is outside scope;
- curated cross-system journeys pass;
- known research silence is reported as uncertainty rather than silently counted as fidelity.

## Progress report format

A useful status report fits on one screen and avoids a synthetic master percentage:

```text
Release level: R2 candidate
Campaign: 24/28 mandatory missions naturally completable
Systems: 5 FIDELITY / 3 FUNCTIONAL / 1 PARTIAL / 0 ABSENT
Fidelity debt: 12 OPEN FIDELITY-DEBT / 7 ACCEPTED deviations
Research silence: 31 authored UNKNOWN rows
Journeys: 5/6 PASS (J5 dialogue/trigger progression failing)
Top blockers: mission 27 trigger chain; original-save transition; ranged AI order
```

The numbers above are an example of the format only, not current project measurements.

## Rules for agents producing completion reports

1. Never infer completion from the latest story number.
2. Never count a loaded map as a completed mission.
3. Never count an UNKNOWN research row as known ROM1 fidelity.
4. Never call an accepted deviation a defect unless the owner reopens it.
5. Never claim a system is FIDELITY from unit tests alone; reconciliation with research is required.
6. Name the research pin and implementation SHA for any fidelity report intended to survive more than the current working session.
7. Prefer named blockers over a decimal percentage.

## Adoption test

Before making this model normative, agents should map the current repository onto it and report ambiguities. If two independent agents cannot classify the same current state within one adjacent category without inventing facts, refine the definitions before adoption.
