# Agent Review Brief — integration fidelity harness

Status: review instructions for the proposal branch. **Do not implement the proposal while performing this review.**

## Material under review

Read:

- `docs/proposals/INTEGRATION-FIDELITY-HARNESS.md`
- `docs/proposals/PROJECT-COMPLETION-MODEL.md`

Then inspect the current repository mechanisms they overlap with, especially:

- `AGENTS.md`;
- `scenarios/README.md` and existing scenarios;
- `pkg/game/scenario.go` and the headless front-end/mission adapters;
- `cmd/missionrun`;
- `scripts/campaign-sweep.sh` and current milestone scripts;
- `docs/DIVERGENCES.md`;
- several recent `contract.md` / `spec.md` / `closure.md` documents that used adversarial review heavily.

Research is relevant only where the proposal makes an authority or fidelity claim. Do not perform new reverse engineering merely to review the harness.

## Review question

The question is not "can this be built?" It almost certainly can.

The question is:

> Does this proposal reduce the probability of shipping a cross-system player-visible defect enough to justify its permanent process and maintenance cost, given the mechanisms againrom already has?

## Required attacks

Try to falsify the proposal from each direction.

### Duplication

Find existing mechanisms that already provide the same evidence. Identify exactly what the proposed harness adds, if anything. A renamed campaign sweep or a second index over scenarios should be rejected.

### Correlated evidence

Find places where the proposed journey would assert a value derived from the same code or assumption as production. Such a witness creates confidence without independence.

### Maintenance cost

Estimate what happens after twenty more stories. Identify fields, manifests or journey expectations that would churn despite production remaining correct.

### False authority

Look for any wording that could let a future agent treat harness output as evidence of ROM1 behaviour. Research must remain the authority on ROM1 truth.

### Process multiplication

Look for any step that creates another review chain, mandatory prose artifact, ledger, checker or per-story ceremony. The proposal explicitly claims not to do this; report contradictions.

### Coverage illusion

Construct at least three plausible player-visible defects that all proposed journeys would miss. The existence of such defects does not by itself reject the harness; it tests whether the proposal describes its limits honestly.

### Failure triage

Take at least three existing historical defect shapes from story closures or review findings and classify how the proposed harness would have reported them: PRODUCT, WITNESS, EXPECTATION or INFRA. If classification is ambiguous, explain why.

### Completion model

Attempt to classify the current project under R0–R5 and the four axes using repository evidence only. Record every place where the model forces you to guess. Do not fill gaps with intuition.

## Pilot design task

Without writing code, choose the smallest three existing scenarios that could serve as J1/J2/J4 or explain why no existing scenario is sufficient.

For each candidate state:

- which production boundaries it really crosses;
- which observation is currently strong;
- which observation is currently self-derived, missing or too weak;
- one production mutation that should make it fail;
- whether that mutation is already killed by a cheaper existing test.

If existing tests already kill every meaningful mutation, say so: that is evidence against adding the journey to the curated portfolio.

## Cost estimate

Give two estimates:

1. one-time implementation cost to reach a useful three-journey pilot;
2. recurring cost per ten ordinary stories, including expectation maintenance and investigation of false failures.

Use repository evidence and concrete affected files. Do not estimate in developer-days unless there is enough evidence to justify it; counts of files, scenario changes, commands and expected review actions are preferable.

## Verdict format

Return exactly one recommendation:

- `ADOPT` — useful essentially as written;
- `ADOPT WITH CUTS` — useful after naming specific parts to remove or weaken;
- `PILOT ONLY` — evidence is insufficient for permanent adoption; run the three-journey experiment;
- `REJECT` — duplicates existing evidence or costs more than it buys.

The recommendation must include:

- strongest argument **for** adoption;
- strongest argument **against** adoption;
- exact proposed edits to the two proposal documents;
- the three-journey pilot mapping, if any;
- what observation after the pilot would cause the reviewer to reverse its verdict.

Do not reward the proposal for being elaborate. Prefer deletion over a new mechanism when existing evidence can be made stronger in place.
